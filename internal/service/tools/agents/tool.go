package agents

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"

	"github.com/zjregee/alter/internal/service/tools"
)

const (
	AgentsToolName        = "agents"
	AgentsToolDescription = "Manages agent lifecycle: create, start, stop, and delete."
)

func GetAgentsTool(ctx context.Context) (*schema.ToolInfo, tool.InvokableTool, error) {
	t, err := utils.InferTool(AgentsToolName, AgentsToolDescription, AgentsTool)
	if err != nil {
		return nil, nil, err
	}
	info, err := t.Info(ctx)
	if err != nil {
		return nil, nil, err
	}
	return info, t, nil
}

func init() {
	tools.RegisterTool(AgentsToolName, GetAgentsTool)
}
