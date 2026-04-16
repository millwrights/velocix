package github

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/go-github/v60/github"
	"github.com/skalluru/velocix/internal/model"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
)

const (
	maxConcurrency = 5
	perPage        = 100
)

type Client struct {
	gh     *github.Client
	logger *slog.Logger
}

func NewClient(token string, logger *slog.Logger) *Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	return &Client{
		gh:     github.NewClient(tc),
		logger: logger,
	}
}

func (c *Client) FetchAllWorkflowRuns(ctx context.Context, org string) ([]model.WorkflowRun, error) {
	repos, err := c.listOrgRepos(ctx, org)
	if err != nil {
		return nil, fmt.Errorf("listing repos: %w", err)
	}

	c.logger.Info("fetching workflow runs", "org", org, "repos", len(repos))

	var (
		mu      sync.Mutex
		allRuns []model.WorkflowRun
	)

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrency)

	for _, repo := range repos {
		repo := repo
		g.Go(func() error {
			runs, err := c.fetchRepoRuns(gctx, org, repo)
			if err != nil {
				c.logger.Warn("failed to fetch runs", "repo", repo, "error", err)
				return nil // don't fail the whole batch
			}
			mu.Lock()
			allRuns = append(allRuns, runs...)
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	c.logger.Info("fetched workflow runs", "total", len(allRuns))
	return allRuns, nil
}

func (c *Client) listOrgRepos(ctx context.Context, org string) ([]string, error) {
	var allRepos []string
	opts := &github.RepositoryListByOrgOptions{
		Sort: "updated",
		ListOptions: github.ListOptions{
			PerPage: perPage,
		},
	}

	for {
		repos, resp, err := c.gh.Repositories.ListByOrg(ctx, org, opts)
		if err != nil {
			return nil, err
		}
		for _, r := range repos {
			allRepos = append(allRepos, r.GetName())
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allRepos, nil
}

func (c *Client) fetchRepoRuns(ctx context.Context, owner, repo string) ([]model.WorkflowRun, error) {
	since := time.Now().Add(-24 * time.Hour)
	opts := &github.ListWorkflowRunsOptions{
		Created: ">=" + since.Format("2006-01-02"),
		ListOptions: github.ListOptions{
			PerPage: perPage,
		},
	}

	var runs []model.WorkflowRun

	for {
		result, resp, err := c.gh.Actions.ListRepositoryWorkflowRuns(ctx, owner, repo, opts)
		if err != nil {
			return nil, err
		}

		for _, r := range result.WorkflowRuns {
			runs = append(runs, toWorkflowRun(owner, repo, r))
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return runs, nil
}

func toWorkflowRun(owner, repo string, r *github.WorkflowRun) model.WorkflowRun {
	return model.WorkflowRun{
		ID:           r.GetID(),
		RepoName:     repo,
		RepoOwner:    owner,
		WorkflowName: r.GetName(),
		WorkflowPath: r.GetURL(),
		HeadBranch:   r.GetHeadBranch(),
		HeadSHA:      r.GetHeadSHA(),
		Status:       r.GetStatus(),
		Conclusion:   r.GetConclusion(),
		Event:        r.GetEvent(),
		HTMLURL:      r.GetHTMLURL(),
		CreatedAt:    r.GetCreatedAt().Time,
		UpdatedAt:    r.GetUpdatedAt().Time,
		RunNumber:    r.GetRunNumber(),
		RunAttempt:   r.GetRunAttempt(),
		Actor:        r.GetActor().GetLogin(),
	}
}
