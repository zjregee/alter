package agents

import (
	"context"
	"time"
)

type AgentState string

type AgentType string

const (
	TypeCodexAgent  AgentType = "codex"
	TypeGeminiAgent AgentType = "gemini_cli"
	TypeClaudeAgent AgentType = "claude_code"
)

const (
	StateIdle     AgentState = "idle"
	StateRunning  AgentState = "running"
	StateFinished AgentState = "finished"
	StateFailed   AgentState = "failed"
)

type Config struct {
	Name        string
	Description string
	Prompt      string
	WorkDir     string
	Env         []string
	Timeout     time.Duration
}

type Status struct {
	State       AgentState
	Output      string
	ErrorOutput string
	Error       error
	ExitCode    int
	StartedAt   *time.Time
	EndedAt     *time.Time
	Duration    time.Duration
}

type Agent interface {
	ID() string
	Config() Config
	Execute(ctx context.Context) error
	Cancel(ctx context.Context) error
	Done() <-chan struct{}
	Status() Status
}
