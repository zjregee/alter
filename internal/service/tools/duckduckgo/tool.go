package duckduckgo

import (
	"context"

	ddg "github.com/cloudwego/eino-ext/components/tool/duckduckgo/v2"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const (
	DuckDuckGoToolName        = "duckduckgo_text_search"
	DuckDuckGoToolDescription = "Searches the web for information using DuckDuckGo text search."
)

func GetDuckDuckGoTool(ctx context.Context) (*schema.ToolInfo, tool.InvokableTool, error) {
	cfg := &ddg.Config{
		ToolName: DuckDuckGoToolName,
		ToolDesc: DuckDuckGoToolDescription,
	}

	t, err := ddg.NewTextSearchTool(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}

	info, err := t.Info(ctx)
	if err != nil {
		return nil, nil, err
	}

	return info, t, nil
}
