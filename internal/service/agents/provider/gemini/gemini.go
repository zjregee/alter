package agents

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/zjregee/alter/internal/service/agents"
)

var _ agents.Agent = (*GeminiAgent)(nil)

type GeminiAgent struct {
	mu     sync.RWMutex
	id     string
	config agents.Config
	status agents.Status

	cmd      *exec.Cmd
	cancel   context.CancelFunc
	output   bytes.Buffer
	errorOut bytes.Buffer
	doneCh   chan struct{}
}

func NewGeminiAgent(cfg agents.Config) (*GeminiAgent, error) {
	if cfg.WorkDir == "" {
		return nil, fmt.Errorf("work dir is required")
	}

	envMap := make(map[string]string)
	for _, env := range os.Environ() {
		key, value, ok := agents.ParseEnvVar(env)
		if !ok {
			continue
		}
		envMap[key] = value
	}
	for _, env := range cfg.Env {
		key, value, ok := agents.ParseEnvVar(env)
		if !ok {
			continue
		}
		envMap[key] = value
	}

	mergedEnv := make([]string, 0, len(envMap))
	for key, value := range envMap {
		mergedEnv = append(mergedEnv, key+"="+value)
	}
	cfg.Env = mergedEnv

	return &GeminiAgent{
		id:     agents.GenerateAgentID(string(agents.TypeGeminiAgent)),
		config: cfg,
		status: agents.Status{
			State: agents.StateIdle,
		},
		doneCh: make(chan struct{}),
	}, nil
}

func (g *GeminiAgent) ID() string {
	return g.id
}

func (g *GeminiAgent) Config() agents.Config {
	return g.config
}

func (g *GeminiAgent) Execute(ctx context.Context) error {
	g.mu.Lock()

	if g.status.State == agents.StateRunning {
		g.mu.Unlock()
		return fmt.Errorf("agent is already running")
	}

	var taskCtx context.Context
	var cancel context.CancelFunc
	if g.config.Timeout > 0 {
		taskCtx, cancel = context.WithTimeout(ctx, g.config.Timeout)
	} else {
		taskCtx, cancel = context.WithCancel(ctx)
	}
	g.cancel = cancel

	g.clearStatus()
	g.output.Reset()
	g.errorOut.Reset()
	close(g.doneCh)
	g.doneCh = make(chan struct{})

	args := g.buildGeminiArgs()
	g.cmd = exec.CommandContext(taskCtx, "gemini", args...)
	g.cmd.Dir = g.config.WorkDir
	g.cmd.Env = g.config.Env
	g.cmd.Stdin = nil

	g.cmd.Stdout = &g.output
	g.cmd.Stderr = &g.errorOut

	if err := g.cmd.Start(); err != nil {
		g.status.State = agents.StateFailed
		g.status.Error = err
		close(g.doneCh)
		g.mu.Unlock()
		return fmt.Errorf("failed to start agent: %w", err)
	}

	now := time.Now()
	g.status.State = agents.StateRunning
	g.status.StartedAt = &now

	g.mu.Unlock()
	go g.monitor()

	return nil
}

func (g *GeminiAgent) Cancel(ctx context.Context) error {
	g.mu.Lock()

	if g.status.State != agents.StateRunning {
		g.mu.Unlock()
		return fmt.Errorf("agent is not running")
	}

	if g.cancel != nil {
		g.cancel()
	}

	process := g.cmd.Process
	g.mu.Unlock()

	select {
	case <-ctx.Done():
		select {
		case <-g.doneCh:
			return nil
		default:
		}
		if process == nil {
			return fmt.Errorf("failed to kill agent")
		}
		if err := process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			return fmt.Errorf("failed to kill agent: %w", err)
		}
		<-g.doneCh
	case <-g.doneCh:
	}

	return nil
}

func (g *GeminiAgent) Done() <-chan struct{} {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.doneCh
}

func (g *GeminiAgent) Status() agents.Status {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.status
}

func (g *GeminiAgent) buildGeminiArgs() []string {
	args := []string{
		"-p", g.config.Prompt,
		"--dangerously-skip-permissions",
	}
	return args
}

func (g *GeminiAgent) monitor() {
	defer close(g.doneCh)
	defer func() {
		if g.cancel != nil {
			g.cancel()
		}
	}()

	if g.cmd == nil {
		return
	}
	err := g.cmd.Wait()

	g.mu.Lock()
	defer g.mu.Unlock()

	g.status.Error = err

	now := time.Now()
	g.status.EndedAt = &now
	g.status.Duration = now.Sub(*g.status.StartedAt)

	if g.cmd.ProcessState != nil {
		g.status.ExitCode = g.cmd.ProcessState.ExitCode()
	}

	if err != nil || g.status.ExitCode != 0 {
		g.status.State = agents.StateFailed
	} else {
		g.status.State = agents.StateFinished
	}

	g.status.Output = g.output.String()
	g.status.ErrorOutput = g.errorOut.String()
}

func (g *GeminiAgent) clearStatus() {
	g.status.State = agents.StateIdle
	g.status.Output = ""
	g.status.ErrorOutput = ""
	g.status.Error = nil
	g.status.ExitCode = 0
	g.status.StartedAt = nil
	g.status.EndedAt = nil
	g.status.Duration = 0
}
