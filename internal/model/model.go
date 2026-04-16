package model

import "time"

// WorkflowRun represents a single GitHub Actions workflow run.
type WorkflowRun struct {
	ID           int64     `json:"id"`
	RepoName     string    `json:"repo_name"`
	RepoOwner    string    `json:"repo_owner"`
	WorkflowName string    `json:"workflow_name"`
	WorkflowPath string    `json:"workflow_path"`
	HeadBranch   string    `json:"head_branch"`
	HeadSHA      string    `json:"head_sha"`
	Status       string    `json:"status"`       // queued, in_progress, completed
	Conclusion   string    `json:"conclusion"`   // success, failure, cancelled, skipped, etc.
	Event        string    `json:"event"`        // push, pull_request, schedule, etc.
	HTMLURL      string    `json:"html_url"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	RunNumber    int       `json:"run_number"`
	RunAttempt   int       `json:"run_attempt"`
	Actor        string    `json:"actor"`
}

// Stats holds aggregate statistics about workflow runs.
type Stats struct {
	TotalRuns   int `json:"total_runs"`
	Success     int `json:"success"`
	Failure     int `json:"failure"`
	InProgress  int `json:"in_progress"`
	Queued      int `json:"queued"`
	Cancelled   int `json:"cancelled"`
	RepoCount   int `json:"repo_count"`
}
