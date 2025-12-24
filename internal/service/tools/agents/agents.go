package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	serviceagents "github.com/zjregee/alter/internal/service/agents"
)

type AgentsParams struct {
	Action    string                  `json:"action" jsonschema:"description=Action to perform: create, start, stop, delete."`
	AgentID   string                  `json:"agent_id,omitempty" jsonschema:"description=Agent ID for start, stop, or delete."`
	AgentType serviceagents.AgentType `json:"agent_type,omitempty" jsonschema:"description=Agent type for create."`
	Config    *AgentConfig            `json:"config,omitempty" jsonschema:"description=Agent config for create."`
}

type AgentConfig struct {
	Name           string   `json:"name,omitempty" jsonschema:"description=Agent display name."`
	Description    string   `json:"description,omitempty" jsonschema:"description=Agent description."`
	Prompt         string   `json:"prompt,omitempty" jsonschema:"description=Agent system prompt."`
	WorkDir        string   `json:"work_dir,omitempty" jsonschema:"description=Agent working directory."`
	Env            []string `json:"env,omitempty" jsonschema:"description=Environment variables in KEY=VALUE format."`
	TimeoutSeconds int      `json:"timeout_seconds,omitempty" jsonschema:"description=Execution timeout in seconds."`
}

func AgentsTool(ctx context.Context, params *AgentsParams) (string, error) {
	if params == nil {
		return "", fmt.Errorf("params must be provided")
	}

	action := strings.ToLower(strings.TrimSpace(params.Action))
	switch action {
	case "create":
		return handleCreate(params)
	case "start":
		return handleStart(ctx, params)
	case "stop":
		return handleStop(ctx, params)
	case "delete":
		return handleDelete(params)
	default:
		if action == "" {
			return "", fmt.Errorf("action must be provided")
		}
		return "", fmt.Errorf("unsupported action: %s", action)
	}
}

func handleCreate(params *AgentsParams) (string, error) {
	if params.AgentType == "" {
		return "", fmt.Errorf("agent_type must be provided for create")
	}
	if params.Config == nil {
		return "", fmt.Errorf("config must be provided for create")
	}

	cfg := params.Config.toAgentConfig()
	agentID, err := serviceagents.CreateAgent(params.AgentType, cfg)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Action: create\nAgent ID: %s", agentID), nil
}

func handleStart(ctx context.Context, params *AgentsParams) (string, error) {
	agentID := strings.TrimSpace(params.AgentID)
	if agentID == "" {
		return "", fmt.Errorf("agent_id must be provided for start")
	}

	if err := serviceagents.StartAgentExecution(ctx, agentID); err != nil {
		return "", err
	}

	return fmt.Sprintf("Action: start\nAgent ID: %s", agentID), nil
}

func handleStop(ctx context.Context, params *AgentsParams) (string, error) {
	agentID := strings.TrimSpace(params.AgentID)
	if agentID == "" {
		return "", fmt.Errorf("agent_id must be provided for stop")
	}

	if err := serviceagents.StopAgent(ctx, agentID); err != nil {
		return "", err
	}

	return fmt.Sprintf("Action: stop\nAgent ID: %s", agentID), nil
}

func handleDelete(params *AgentsParams) (string, error) {
	agentID := strings.TrimSpace(params.AgentID)
	if agentID == "" {
		return "", fmt.Errorf("agent_id must be provided for delete")
	}

	if err := serviceagents.DeleteAgent(agentID); err != nil {
		return "", err
	}

	return fmt.Sprintf("Action: delete\nAgent ID: %s", agentID), nil
}

func (c *AgentConfig) toAgentConfig() serviceagents.Config {
	var timeout time.Duration
	if c.TimeoutSeconds > 0 {
		timeout = time.Duration(c.TimeoutSeconds) * time.Second
	}

	return serviceagents.Config{
		Name:        strings.TrimSpace(c.Name),
		Description: strings.TrimSpace(c.Description),
		Prompt:      c.Prompt,
		WorkDir:     strings.TrimSpace(c.WorkDir),
		Env:         c.Env,
		Timeout:     timeout,
	}
}
