package yacd

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type Runner struct {
	mu     sync.RWMutex
	runs   map[string]*PipelineRun
	logger *slog.Logger
}

func NewRunner(logger *slog.Logger) *Runner {
	return &Runner{
		runs:   make(map[string]*PipelineRun),
		logger: logger,
	}
}

func (r *Runner) GetRun(id string) *PipelineRun {
	r.mu.RLock()
	defer r.mu.RUnlock()
	run := r.runs[id]
	if run == nil {
		return nil
	}
	// Return a copy
	cp := *run
	cp.Stages = make([]StageResult, len(run.Stages))
	copy(cp.Stages, run.Stages)
	return &cp
}

func (r *Runner) ListRuns() []PipelineRun {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []PipelineRun
	for _, run := range r.runs {
		cp := *run
		cp.Stages = make([]StageResult, len(run.Stages))
		copy(cp.Stages, run.Stages)
		list = append(list, cp)
	}
	return list
}

// Run executes a pipeline asynchronously, returning the run ID.
func (r *Runner) Run(ctx context.Context, pipeline *Pipeline, owner, repo, path string, inputs map[string]string) string {
	id := fmt.Sprintf("%d", time.Now().UnixNano())

	// Merge variables: pipeline vars + user inputs (inputs override)
	vars := make(map[string]string)
	for k, v := range pipeline.Variables {
		vars[k] = v
	}
	// Apply defaults for inputs not provided
	for name, inp := range pipeline.Inputs {
		if _, ok := inputs[name]; !ok && inp.Default != nil {
			inputs[name] = fmt.Sprintf("%v", inp.Default)
		}
	}
	for k, v := range inputs {
		vars[k] = v
	}

	stages := make([]StageResult, len(pipeline.Stages))
	for i, s := range pipeline.Stages {
		stages[i] = StageResult{Name: s.Name, Status: RunPending}
	}

	run := &PipelineRun{
		ID:        id,
		Pipeline:  pipeline.Name,
		Repo:      repo,
		Owner:     owner,
		Path:      path,
		Status:    RunRunning,
		Inputs:    inputs,
		Stages:    stages,
		StartedAt: time.Now(),
	}

	r.mu.Lock()
	r.runs[id] = run
	r.mu.Unlock()

	go r.execute(ctx, run, pipeline, vars)

	return id
}

func (r *Runner) execute(ctx context.Context, run *PipelineRun, pipeline *Pipeline, vars map[string]string) {
	r.logger.Info("pipeline started", "id", run.ID, "pipeline", run.Pipeline)

	var allOutput strings.Builder
	failed := false

	for i, stage := range pipeline.Stages {
		if failed {
			r.updateStage(run, i, RunFailure, "skipped (previous stage failed)", time.Time{}, time.Time{})
			continue
		}

		start := time.Now()
		r.updateStage(run, i, RunRunning, "", start, time.Time{})

		var stageOutput strings.Builder
		stageFailed := false

		for _, step := range stage.Steps {
			script := substituteVars(step.Run, vars)

			r.logger.Info("running step", "stage", stage.Name, "step", step.Name)

			cmd := exec.CommandContext(ctx, "bash", "-c", script)

			// Set env from step-level env
			for k, v := range step.Env {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, substituteVars(v, vars)))
			}
			// Also set all vars as env
			for k, v := range vars {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
			}

			var out bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &out

			err := cmd.Run()
			output := out.String()
			stageOutput.WriteString(output)

			if err != nil {
				stageOutput.WriteString(fmt.Sprintf("\nError: %s\n", err))
				stageFailed = true
				break
			}
		}

		end := time.Now()
		status := RunSuccess
		if stageFailed {
			status = RunFailure
			failed = true
		}

		r.updateStage(run, i, status, stageOutput.String(), start, end)
		allOutput.WriteString(stageOutput.String())
	}

	r.mu.Lock()
	if failed {
		run.Status = RunFailure
	} else {
		run.Status = RunSuccess
	}
	run.EndedAt = time.Now()
	run.Output = allOutput.String()
	r.mu.Unlock()

	r.logger.Info("pipeline finished", "id", run.ID, "status", run.Status)
}

func (r *Runner) updateStage(run *PipelineRun, idx int, status RunStatus, output string, start, end time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	run.Stages[idx].Status = status
	if output != "" {
		run.Stages[idx].Output = output
	}
	if !start.IsZero() {
		run.Stages[idx].StartedAt = start
	}
	if !end.IsZero() {
		run.Stages[idx].EndedAt = end
		run.Stages[idx].Duration = end.Sub(start).Round(time.Millisecond).String()
	}
}

func substituteVars(script string, vars map[string]string) string {
	result := script
	for k, v := range vars {
		result = strings.ReplaceAll(result, "${"+k+"}", v)
	}
	return result
}
