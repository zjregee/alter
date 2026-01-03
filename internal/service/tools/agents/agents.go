package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	agentsService "github.com/zjregee/alter/internal/service/agents"
	_ "github.com/zjregee/alter/internal/service/agents/provider/claude"
	_ "github.com/zjregee/alter/internal/service/agents/provider/codex"
	_ "github.com/zjregee/alter/internal/service/agents/provider/gemini"
)

func AgentsTool(ctx context.Context, params *RunAgentParams) (string, error) {
	if params == nil {
		return "", fmt.Errorf("params must be provided")
	}

	if params.AgentType == "" {
		return "", fmt.Errorf("agent_type must be provided")
	}
	if params.Config == nil {
		return "", fmt.Errorf("config must be provided")
	}

	cfg := params.Config.toAgentConfig()
	if agentsService.AgentType(params.AgentType) == agentsService.TypeClaudeAgent {
		cfg.Env = mergeClaudeEnvDefaults(cfg.Env)
	}
	status, err := agentsService.RunAgent(ctx, agentsService.AgentType(params.AgentType), cfg)
	if err != nil && status.State == "" {
		return "", err
	}

	result := formatRunResult(params, status, err)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (c *AgentConfig) toAgentConfig() agentsService.Config {
	var timeout time.Duration
	if c.TimeoutSeconds > 0 {
		timeout = time.Duration(c.TimeoutSeconds) * time.Second
	}

	return agentsService.Config{
		Name:        strings.TrimSpace(c.Name),
		Description: strings.TrimSpace(c.Description),
		Prompt:      c.Prompt,
		WorkDir:     strings.TrimSpace(c.WorkDir),
		Env:         c.Env,
		Timeout:     timeout,
	}
}

func mergeClaudeEnvDefaults(env []string) []string {
	defaults := []string{}

	envMap := make(map[string]string, len(defaults)+len(env))
	for _, item := range defaults {
		if key, value, ok := agentsService.ParseEnvVar(item); ok {
			envMap[key] = value
		}
	}
	for _, item := range env {
		if key, value, ok := agentsService.ParseEnvVar(item); ok {
			envMap[key] = value
		}
	}

	merged := make([]string, 0, len(envMap))
	for key, value := range envMap {
		merged = append(merged, key+"="+value)
	}
	return merged
}

func formatRunResult(params *RunAgentParams, status agentsService.Status, runErr error) string {
	var builder strings.Builder

	if name := strings.TrimSpace(params.Config.Name); name != "" {
		fmt.Fprintf(&builder, "Agent name: %s\n", name)
	}
	fmt.Fprintf(&builder, "Agent type: %s\n", strings.TrimSpace(params.AgentType))
	if workDir := strings.TrimSpace(params.Config.WorkDir); workDir != "" {
		fmt.Fprintf(&builder, "Work directory: %s\n", workDir)
	}

	fmt.Fprintf(&builder, "State: %s\n", status.State)
	fmt.Fprintf(&builder, "Exit code: %d\n", status.ExitCode)
	if runErr != nil {
		fmt.Fprintf(&builder, "Error: %s\n", runErr.Error())
	}

	if status.Output == "" {
		fmt.Fprint(&builder, "Output: (empty)\n")
	} else {
		fmt.Fprint(&builder, "Output:\n```text\n")
		builder.WriteString(strings.TrimRight(status.Output, "\n"))
		fmt.Fprint(&builder, "\n```\n")
	}
	if status.ErrorOutput == "" {
		fmt.Fprint(&builder, "Error output: (empty)")
	} else {
		fmt.Fprint(&builder, "Error output:\n```text\n")
		builder.WriteString(strings.TrimRight(status.ErrorOutput, "\n"))
		fmt.Fprint(&builder, "\n```")
	}

	return builder.String()
}
