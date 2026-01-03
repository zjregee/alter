package tools

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"github.com/zjregee/alter/internal/service/mcp"
	"github.com/zjregee/alter/internal/service/tools/agents"
	"github.com/zjregee/alter/internal/service/tools/bash"
	"github.com/zjregee/alter/internal/service/tools/duckduckgo"
	"github.com/zjregee/alter/internal/service/tools/feed"
	"github.com/zjregee/alter/internal/service/tools/skills"
)

var registeredTools = make(map[string]func(context.Context) (*schema.ToolInfo, tool.InvokableTool, error))

func registerTool(name string, getToolFunc func(context.Context) (*schema.ToolInfo, tool.InvokableTool, error)) {
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

	if err := addMCPTools(ctx, &allToolInfos, allToolsMap); err != nil {
		return nil, nil, err
	}

	return allToolInfos, allToolsMap, nil
}

func addMCPTools(ctx context.Context, allToolInfos *[]*schema.ToolInfo, allToolsMap map[string]tool.InvokableTool) error {
	tools, err := mcp.GetMCPTools(ctx)
	if err != nil {
		return err
	}
	if len(tools) == 0 {
		return nil
	}

	for _, t := range tools {
		invokableTool, ok := t.(tool.InvokableTool)
		if !ok {
			continue
		}

		info, err := t.Info(ctx)
		if err != nil {
			return err
		}

		if _, exists := allToolsMap[info.Name]; exists {
			return fmt.Errorf("duplicate tool name: %s", info.Name)
		}

		*allToolInfos = append(*allToolInfos, info)
		allToolsMap[info.Name] = invokableTool
	}

	return nil
}

func init() {
	registerTool(bash.BashToolName, bash.GetBashTool)
	registerTool(duckduckgo.DuckDuckGoToolName, duckduckgo.GetDuckDuckGoTool)
	registerTool(feed.PushFeedToolName, feed.GetPushFeedTool)
	registerTool(skills.ListSkillsToolName, skills.ListSkillsTool)
	registerTool(skills.LoadSkillToolName, skills.LoadSkillTool)
	registerTool(agents.RunAgentToolName, agents.GetAgentsTool)
}
