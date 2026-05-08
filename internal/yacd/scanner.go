package yacd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type Scanner struct {
	httpClient *http.Client
	graphqlURL string
	logger     *slog.Logger

	mu    sync.RWMutex
	cache map[string]cacheEntry // key: "owner/repo"
	ttl   time.Duration
}

type cacheEntry struct {
	pipelines []PipelineRef
	fetchedAt time.Time
}

func NewScanner(httpClient *http.Client, graphqlURL string, logger *slog.Logger) *Scanner {
	return &Scanner{
		httpClient: httpClient,
		graphqlURL: graphqlURL,
		logger:     logger,
		cache:      make(map[string]cacheEntry),
		ttl:        5 * time.Minute,
	}
}

// ScanOrg scans all repos in an org for .yacd.yaml files.
func (s *Scanner) ScanOrg(ctx context.Context, org string) ([]PipelineRef, error) {
	repos, err := s.listRepos(ctx, org)
	if err != nil {
		return nil, err
	}

	var all []PipelineRef
	for _, repo := range repos {
		refs, err := s.ScanRepo(ctx, org, repo)
		if err != nil {
			s.logger.Warn("failed to scan repo for yacd files", "repo", repo, "error", err)
			continue
		}
		all = append(all, refs...)
	}
	return all, nil
}

// ScanRepo finds all .yacd.yaml files in a repo, using cache if fresh.
func (s *Scanner) ScanRepo(ctx context.Context, owner, repo string) ([]PipelineRef, error) {
	key := owner + "/" + repo

	s.mu.RLock()
	entry, ok := s.cache[key]
	s.mu.RUnlock()

	if ok && time.Since(entry.fetchedAt) < s.ttl {
		return entry.pipelines, nil
	}

	refs, err := s.fetchYacdFiles(ctx, owner, repo)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.cache[key] = cacheEntry{pipelines: refs, fetchedAt: time.Now()}
	s.mu.Unlock()

	return refs, nil
}

// FetchPipeline fetches and parses a specific .yacd.yaml file from a repo.
func (s *Scanner) FetchPipeline(ctx context.Context, owner, repo, path string) (*Pipeline, error) {
	content, err := s.fetchFileContent(ctx, owner, repo, path)
	if err != nil {
		return nil, err
	}
	var p Pipeline
	if err := yaml.Unmarshal(content, &p); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &p, nil
}

func (s *Scanner) fetchYacdFiles(ctx context.Context, owner, repo string) ([]PipelineRef, error) {
	// Use Git tree API to list all files
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/HEAD?recursive=1", owner, repo)
	if !strings.Contains(s.graphqlURL, "api.github.com") {
		// Enterprise: derive REST API base from graphql URL
		base := strings.TrimSuffix(s.graphqlURL, "/graphql")
		url = fmt.Sprintf("%s/repos/%s/%s/git/trees/HEAD?recursive=1", base, owner, repo)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tree API returned %d for %s/%s", resp.StatusCode, owner, repo)
	}

	var tree struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"tree"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tree); err != nil {
		return nil, err
	}

	var refs []PipelineRef
	for _, t := range tree.Tree {
		if t.Type != "blob" || !strings.HasSuffix(t.Path, ".yacd.yaml") {
			continue
		}
		// Fetch and parse to get name/description
		content, err := s.fetchFileContent(ctx, owner, repo, t.Path)
		if err != nil {
			s.logger.Warn("failed to fetch yacd file", "path", t.Path, "error", err)
			continue
		}
		var p Pipeline
		if err := yaml.Unmarshal(content, &p); err != nil {
			s.logger.Warn("failed to parse yacd file", "path", t.Path, "error", err)
			continue
		}
		refs = append(refs, PipelineRef{
			Repo:     repo,
			Owner:    owner,
			Path:     t.Path,
			Name:     p.Name,
			Desc:     p.Description,
			InputCnt: len(p.Inputs),
			StageCnt: len(p.Stages),
		})
	}
	return refs, nil
}

func (s *Scanner) fetchFileContent(ctx context.Context, owner, repo, path string) ([]byte, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, path)
	if !strings.Contains(s.graphqlURL, "api.github.com") {
		base := strings.TrimSuffix(s.graphqlURL, "/graphql")
		url = fmt.Sprintf("%s/repos/%s/%s/contents/%s", base, owner, repo, path)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.raw+json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("contents API returned %d for %s", resp.StatusCode, path)
	}

	return io.ReadAll(resp.Body)
}

const repoListQuery = `query($org: String!, $cursor: String) {
  organization(login: $org) {
    repositories(first: 100, after: $cursor, orderBy: {field: PUSHED_AT, direction: DESC}) {
      pageInfo { hasNextPage endCursor }
      nodes { name }
    }
  }
}`

func (s *Scanner) listRepos(ctx context.Context, org string) ([]string, error) {
	var repos []string
	var cursor *string

	for {
		variables := map[string]any{"org": org, "cursor": cursor}
		body, _ := json.Marshal(map[string]any{"query": repoListQuery, "variables": variables})

		req, err := http.NewRequestWithContext(ctx, "POST", s.graphqlURL, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var result struct {
			Data struct {
				Organization struct {
					Repositories struct {
						PageInfo struct {
							HasNextPage bool   `json:"hasNextPage"`
							EndCursor   string `json:"endCursor"`
						} `json:"pageInfo"`
						Nodes []struct {
							Name string `json:"name"`
						} `json:"nodes"`
					} `json:"repositories"`
				} `json:"organization"`
			} `json:"data"`
		}
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, err
		}

		r := result.Data.Organization.Repositories
		for _, n := range r.Nodes {
			repos = append(repos, n.Name)
		}
		if !r.PageInfo.HasNextPage {
			break
		}
		ec := r.PageInfo.EndCursor
		cursor = &ec
	}

	return repos, nil
}
