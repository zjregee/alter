package feed

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

const (
	PushFeedToolName        = "push_feed"
	PushFeedToolDescription = "Pushes a feed item to a topic."
)

type PushFeedParams struct {
	Topic   string `json:"topic" jsonschema:"description=Feed topic."`
	Title   string `json:"title" jsonschema:"description=Feed item title."`
	Content string `json:"content" jsonschema:"description=Feed item content."`
}

func GetPushFeedTool(ctx context.Context) (*schema.ToolInfo, tool.InvokableTool, error) {
	t, err := utils.InferTool(PushFeedToolName, PushFeedToolDescription, PushFeed)
	if err != nil {
		return nil, nil, err
	}

	info, err := t.Info(ctx)
	if err != nil {
		return nil, nil, err
	}

	return info, t, nil
}
