package agents

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

const (
	RunAgentToolName        = "run_agent"
	RunAgentToolDescription = "Runs an agent task with a single request. Supported agent types: claude_code, codex, gemini_cli."
)

type RunAgentParams struct {
	AgentType string       `json:"agent_type" jsonschema:"description=Agent type to run. Supported: claude_code, codex, gemini_cli."`
	Config    *AgentConfig `json:"config" jsonschema:"description=Agent config for the run."`
}

type AgentConfig struct {
	Name           string   `json:"name" jsonschema:"description=Agent display name."`
	Description    string   `json:"description" jsonschema:"description=Agent description."`
	Prompt         string   `json:"prompt" jsonschema:"description=Agent system prompt."`
	WorkDir        string   `json:"work_dir" jsonschema:"description=Agent working directory."`
	Env            []string `json:"env,omitempty" jsonschema:"description=Environment variables in KEY=VALUE format."`
	TimeoutSeconds int      `json:"timeout_seconds,omitempty" jsonschema:"description=Execution timeout in seconds."`
}

func GetAgentsTool(ctx context.Context) (*schema.ToolInfo, tool.InvokableTool, error) {
	t, err := utils.InferTool(RunAgentToolName, RunAgentToolDescription, AgentsTool)
	if err != nil {
		return nil, nil, err
	}

	info, err := t.Info(ctx)
	if err != nil {
		return nil, nil, err
	}

	return info, t, nil
}
