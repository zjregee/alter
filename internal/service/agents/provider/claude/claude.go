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

var _ agents.Agent = (*ClaudeAgent)(nil)

type ClaudeAgent struct {
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

func NewClaudeAgent(cfg agents.Config) (*ClaudeAgent, error) {
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

	return &ClaudeAgent{
		id:     agents.GenerateAgentID(string(agents.TypeClaudeAgent)),
		config: cfg,
		status: agents.Status{
			State: agents.StateIdle,
		},
		doneCh: make(chan struct{}),
	}, nil
}

func (c *ClaudeAgent) ID() string {
	return c.id
}

func (c *ClaudeAgent) Config() agents.Config {
	return c.config
}

func (c *ClaudeAgent) Execute(ctx context.Context) error {
	c.mu.Lock()

	if c.status.State == agents.StateRunning {
		c.mu.Unlock()
		return fmt.Errorf("agent is already running")
	}

	var taskCtx context.Context
	var cancel context.CancelFunc
	if c.config.Timeout > 0 {
		taskCtx, cancel = context.WithTimeout(ctx, c.config.Timeout)
	} else {
		taskCtx, cancel = context.WithCancel(ctx)
	}
	c.cancel = cancel

	c.clearStatus()
	c.output.Reset()
	c.errorOut.Reset()
	close(c.doneCh)
	c.doneCh = make(chan struct{})

	args := c.buildClaudeArgs()
	c.cmd = exec.CommandContext(taskCtx, "claude", args...)
	c.cmd.Dir = c.config.WorkDir
	c.cmd.Env = c.config.Env
	c.cmd.Stdin = nil

	c.cmd.Stdout = &c.output
	c.cmd.Stderr = &c.errorOut

	if err := c.cmd.Start(); err != nil {
		c.status.State = agents.StateFailed
		c.status.Error = err
		close(c.doneCh)
		c.mu.Unlock()
		return fmt.Errorf("failed to start agent: %w", err)
	}

	now := time.Now()
	c.status.State = agents.StateRunning
	c.status.StartedAt = &now

	c.mu.Unlock()
	go c.monitor()

	return nil
}

func (c *ClaudeAgent) Cancel(ctx context.Context) error {
	c.mu.Lock()

	if c.status.State != agents.StateRunning {
		c.mu.Unlock()
		return fmt.Errorf("agent is not running")
	}

	if c.cancel != nil {
		c.cancel()
	}

	process := c.cmd.Process
	c.mu.Unlock()

	select {
	case <-ctx.Done():
		select {
		case <-c.doneCh:
			return nil
		default:
		}
		if process == nil {
			return fmt.Errorf("failed to kill agent")
		}
		if err := process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			return fmt.Errorf("failed to kill agent: %w", err)
		}
		<-c.doneCh
	case <-c.doneCh:
	}

	return nil
}

func (c *ClaudeAgent) Done() <-chan struct{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.doneCh
}

func (c *ClaudeAgent) Status() agents.Status {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.status
}

func (c *ClaudeAgent) buildClaudeArgs() []string {
	args := []string{
		"-p", c.config.Prompt,
		"--dangerously-skip-permissions",
	}
	return args
}

func (c *ClaudeAgent) monitor() {
	defer close(c.doneCh)
	defer func() {
		if c.cancel != nil {
			c.cancel()
		}
	}()

	if c.cmd == nil {
		return
	}
	err := c.cmd.Wait()

	c.mu.Lock()
	defer c.mu.Unlock()

	c.status.Error = err

	now := time.Now()
	c.status.EndedAt = &now
	c.status.Duration = now.Sub(*c.status.StartedAt)

	if c.cmd.ProcessState != nil {
		c.status.ExitCode = c.cmd.ProcessState.ExitCode()
	}

	if err != nil || c.status.ExitCode != 0 {
		c.status.State = agents.StateFailed
	} else {
		c.status.State = agents.StateFinished
	}

	c.status.Output = c.output.String()
	c.status.ErrorOutput = c.errorOut.String()
}

func (c *ClaudeAgent) clearStatus() {
	c.status.State = agents.StateIdle
	c.status.Output = ""
	c.status.ErrorOutput = ""
	c.status.Error = nil
	c.status.ExitCode = 0
	c.status.StartedAt = nil
	c.status.EndedAt = nil
	c.status.Duration = 0
}
