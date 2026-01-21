package sequential_thinking

import (
	"context"

	sequentialthinking "github.com/cloudwego/eino-ext/components/tool/sequentialthinking"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const (
	SequentialThinkingToolName        = "sequential_thinking"
	SequentialThinkingToolDescription = "A detailed tool for dynamic and reflective problem-solving through thoughts. This tool helps analyze problems through a flexible thinking process that can adapt and evolve."
)

func GetSequentialThinkingTool(ctx context.Context) (*schema.ToolInfo, tool.InvokableTool, error) {
	t, err := sequentialthinking.NewTool()
	if err != nil {
		return nil, nil, err
	}

	info, err := t.Info(ctx)
	if err != nil {
		return nil, nil, err
	}

	return info, t, nil
}
