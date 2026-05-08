package yacd

import "time"

type InputType string

const (
	InputString  InputType = "string"
	InputChoice  InputType = "choice"
	InputBoolean InputType = "boolean"
	InputNumber  InputType = "number"
	InputText    InputType = "text"
)

type Input struct {
	Type        InputType `yaml:"type" json:"type"`
	Description string    `yaml:"description" json:"description"`
	Default     any       `yaml:"default" json:"default"`
	Options     []string  `yaml:"options,omitempty" json:"options,omitempty"`
}

type Step struct {
	Name string            `yaml:"name" json:"name"`
	Run  string            `yaml:"run" json:"run"`
	Env  map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
}

type Stage struct {
	Name  string   `yaml:"name" json:"name"`
	Needs []string `yaml:"needs,omitempty" json:"needs,omitempty"`
	Steps []Step   `yaml:"steps" json:"steps"`
}

type Pipeline struct {
	Name        string           `yaml:"name" json:"name"`
	Description string           `yaml:"description" json:"description"`
	Inputs      map[string]Input `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Variables   map[string]string `yaml:"variables,omitempty" json:"variables,omitempty"`
	Stages      []Stage          `yaml:"stages" json:"stages"`
}

type PipelineRef struct {
	Repo     string `json:"repo"`
	Owner    string `json:"owner"`
	Path     string `json:"path"`
	Name     string `json:"name"`
	Desc     string `json:"description"`
	InputCnt int    `json:"input_count"`
	StageCnt int    `json:"stage_count"`
}

type RunStatus string

const (
	RunPending  RunStatus = "pending"
	RunRunning  RunStatus = "running"
	RunSuccess  RunStatus = "success"
	RunFailure  RunStatus = "failure"
)

type StageResult struct {
	Name      string    `json:"name"`
	Status    RunStatus `json:"status"`
	Output    string    `json:"output"`
	StartedAt time.Time `json:"started_at,omitempty"`
	EndedAt   time.Time `json:"ended_at,omitempty"`
	Duration  string    `json:"duration,omitempty"`
}

type PipelineRun struct {
	ID        string            `json:"id"`
	Pipeline  string            `json:"pipeline"`
	Repo      string            `json:"repo"`
	Owner     string            `json:"owner"`
	Path      string            `json:"path"`
	Status    RunStatus         `json:"status"`
	Inputs    map[string]string `json:"inputs"`
	Stages    []StageResult     `json:"stages"`
	StartedAt time.Time         `json:"started_at"`
	EndedAt   time.Time         `json:"ended_at,omitempty"`
	Output    string            `json:"output"`
}
