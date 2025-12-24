package tools

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

var registeredTools = make(map[string]func(context.Context) (*schema.ToolInfo, tool.InvokableTool, error))

func RegisterTool(name string, getToolFunc func(context.Context) (*schema.ToolInfo, tool.InvokableTool, error)) {
	registeredTools[name] = getToolFunc
}

func GetAllRegisteredTools(ctx context.Context) ([]*schema.ToolInfo, map[string]tool.InvokableTool, error) {
	var allToolInfos []*schema.ToolInfo
	allToolsMap := make(map[string]tool.InvokableTool)

	addTool := func(getToolFunc func(context.Context) (*schema.ToolInfo, tool.InvokableTool, error)) error {
		info, t, err := getToolFunc(ctx)
		if err != nil {
			return err
		}
		allToolInfos = append(allToolInfos, info)
		allToolsMap[info.Name] = t
		return nil
	}

	for _, getToolFunc := range registeredTools {
		if err := addTool(getToolFunc); err != nil {
			return nil, nil, err
		}
	}

	return allToolInfos, allToolsMap, nil
}
