package store

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/skalluru/velocix/internal/model"
)

type Store struct {
	mu        sync.RWMutex
	runs      []model.WorkflowRun
	listeners []chan struct{}
	dataDir   string
	org       string
	logger    *slog.Logger
}

func NewStore(dataDir string, org string, logger *slog.Logger) *Store {
	s := &Store{
		dataDir: dataDir,
		org:     org,
		logger:  logger,
	}
	s.loadFromDisk()
	return s
}

func (s *Store) Update(runs []model.WorkflowRun) {
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].UpdatedAt.After(runs[j].UpdatedAt)
	})

	s.mu.Lock()
	s.runs = runs
	listeners := make([]chan struct{}, len(s.listeners))
	copy(listeners, s.listeners)
	s.mu.Unlock()

	s.persistToDisk()

	for _, ch := range listeners {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func (s *Store) GetAll() []model.WorkflowRun {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.WorkflowRun, len(s.runs))
	copy(out, s.runs)
	return out
}

func (s *Store) Filter(repo, status, workflow string) []model.WorkflowRun {
	s.mu.RLock()
	defer s.mu.RUnlock()

	repos := splitFilter(repo)
	statuses := splitFilter(status)
	workflows := splitFilter(workflow)

	var filtered []model.WorkflowRun
	for _, r := range s.runs {
		if len(repos) > 0 && !matchesAnyFold(r.RepoName, repos) {
			continue
		}
		if len(statuses) > 0 {
			if !matchesStatus(r, statuses) {
				continue
			}
		}
		if len(workflows) > 0 && !matchesAnyFold(r.WorkflowName, workflows) {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

func (s *Store) GetStats() model.Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	repos := make(map[string]bool)
	stats := model.Stats{TotalRuns: len(s.runs)}

	for _, r := range s.runs {
		repos[r.RepoName] = true
		switch {
		case r.Status == "in_progress":
			stats.InProgress++
		case r.Status == "queued":
			stats.Queued++
		case r.Conclusion == "success":
			stats.Success++
		case r.Conclusion == "failure":
			stats.Failure++
		case r.Conclusion == "cancelled":
			stats.Cancelled++
		}
	}
	stats.RepoCount = len(repos)
	return stats
}

func (s *Store) GetRepoNames() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	seen := make(map[string]bool)
	var names []string
	for _, r := range s.runs {
		if !seen[r.RepoName] {
			seen[r.RepoName] = true
			names = append(names, r.RepoName)
		}
	}
	sort.Strings(names)
	return names
}

func (s *Store) GetWorkflowNames() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	seen := make(map[string]bool)
	var names []string
	for _, r := range s.runs {
		if !seen[r.WorkflowName] {
			seen[r.WorkflowName] = true
			names = append(names, r.WorkflowName)
		}
	}
	sort.Strings(names)
	return names
}

func (s *Store) Subscribe() chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch := make(chan struct{}, 1)
	s.listeners = append(s.listeners, ch)
	return ch
}

func (s *Store) Unsubscribe(ch chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, l := range s.listeners {
		if l == ch {
			s.listeners = append(s.listeners[:i], s.listeners[i+1:]...)
			break
		}
	}
}

func (s *Store) cachePath() string {
	if s.org != "" {
		return filepath.Join(s.dataDir, "cache-"+s.org+".json")
	}
	return filepath.Join(s.dataDir, "cache.json")
}

func (s *Store) persistToDisk() {
	s.mu.RLock()
	data, err := json.Marshal(s.runs)
	s.mu.RUnlock()
	if err != nil {
		s.logger.Warn("failed to marshal cache", "error", err)
		return
	}
	if err := os.MkdirAll(s.dataDir, 0o755); err != nil {
		s.logger.Warn("failed to create data dir", "error", err)
		return
	}
	if err := os.WriteFile(s.cachePath(), data, 0o644); err != nil {
		s.logger.Warn("failed to write cache", "error", err)
	}
}

func (s *Store) loadFromDisk() {
	data, err := os.ReadFile(s.cachePath())
	if err != nil {
		return
	}
	var runs []model.WorkflowRun
	if err := json.Unmarshal(data, &runs); err != nil {
		s.logger.Warn("failed to parse cache", "error", err)
		return
	}
	s.runs = runs
	s.logger.Info("loaded cached runs", "count", len(runs))
}

func splitFilter(val string) []string {
	if val == "" {
		return nil
	}
	parts := strings.Split(val, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func matchesAnyFold(val string, targets []string) bool {
	for _, t := range targets {
		if strings.EqualFold(val, t) {
			return true
		}
	}
	return false
}

func matchesStatus(r model.WorkflowRun, statuses []string) bool {
	for _, s := range statuses {
		switch strings.ToLower(s) {
		case "success":
			if r.Conclusion == "success" { return true }
		case "failure":
			if r.Conclusion == "failure" { return true }
		case "in_progress":
			if r.Status == "in_progress" { return true }
		case "queued":
			if r.Status == "queued" { return true }
		case "cancelled":
			if r.Conclusion == "cancelled" { return true }
		default:
			if r.Status == s || r.Conclusion == s { return true }
		}
	}
	return false
}
