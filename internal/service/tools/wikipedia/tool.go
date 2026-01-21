package wikipedia

import (
	"context"

	wiki "github.com/cloudwego/eino-ext/components/tool/wikipedia"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const (
	WikipediaToolName        = "wikipedia_search"
	WikipediaToolDescription = "Searches Wikipedia for information about a given query."
)

func GetWikipediaTool(ctx context.Context) (*schema.ToolInfo, tool.InvokableTool, error) {
	cfg := &wiki.Config{
		ToolName: WikipediaToolName,
		ToolDesc: WikipediaToolDescription,
	}

	t, err := wiki.NewTool(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}

	info, err := t.Info(ctx)
	if err != nil {
		return nil, nil, err
	}

	return info, t, nil
}
