package tools

import (
	"context"
	"fmt"
	"sync"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"github.com/zjregee/alter/internal/service/mcp"
	"github.com/zjregee/alter/internal/service/tools/agents"
	"github.com/zjregee/alter/internal/service/tools/bash"
	"github.com/zjregee/alter/internal/service/tools/duckduckgo"
	"github.com/zjregee/alter/internal/service/tools/feed"
	"github.com/zjregee/alter/internal/service/tools/skills"
	"github.com/zjregee/alter/internal/utils"
)

var registeredTools = make(map[string]func(context.Context) (*schema.ToolInfo, tool.InvokableTool, error))

func registerTool(name string, getToolFunc func(context.Context) (*schema.ToolInfo, tool.InvokableTool, error)) {
	registeredTools[name] = getToolFunc
}

type mcpToolCache struct {
	mu        sync.RWMutex
	ready     bool
	loading   bool
	err       error
	toolInfos []*schema.ToolInfo
	toolsMap  map[string]tool.InvokableTool
}

var cachedMCPTools mcpToolCache

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

	StartMCPToolWarmup(ctx)
	if err := addCachedMCPTools(&allToolInfos, allToolsMap); err != nil {
		return nil, nil, err
	}

	return allToolInfos, allToolsMap, nil
}

func StartMCPToolWarmup(ctx context.Context) {
	cachedMCPTools.mu.Lock()
	if cachedMCPTools.ready || cachedMCPTools.loading {
		cachedMCPTools.mu.Unlock()
		return
	}
	cachedMCPTools.loading = true
	cachedMCPTools.mu.Unlock()

	go func() {
		toolInfos, toolsMap, err := loadMCPTools(ctx)
		if err != nil {
			utils.GetLogger().Printf("Warning: Failed to load MCP tools: %v\n", err)
		}

		cachedMCPTools.mu.Lock()
		cachedMCPTools.ready = err == nil
		cachedMCPTools.loading = false
		cachedMCPTools.err = err
		if err == nil {
			cachedMCPTools.toolInfos = toolInfos
			cachedMCPTools.toolsMap = toolsMap
		}
		cachedMCPTools.mu.Unlock()
	}()
}

func addCachedMCPTools(allToolInfos *[]*schema.ToolInfo, allToolsMap map[string]tool.InvokableTool) error {
	cachedMCPTools.mu.RLock()
	ready := cachedMCPTools.ready
	toolInfos := cachedMCPTools.toolInfos
	toolsMap := cachedMCPTools.toolsMap
	cachedMCPTools.mu.RUnlock()

	if !ready || len(toolInfos) == 0 {
		return nil
	}

	for _, info := range toolInfos {
		if _, exists := allToolsMap[info.Name]; exists {
			return fmt.Errorf("duplicate tool name: %s", info.Name)
		}

		*allToolInfos = append(*allToolInfos, info)
		allToolsMap[info.Name] = toolsMap[info.Name]
	}

	return nil
}

func loadMCPTools(ctx context.Context) ([]*schema.ToolInfo, map[string]tool.InvokableTool, error) {
	tools, err := mcp.GetMCPTools(ctx)
	if err != nil {
		return nil, nil, err
	}
	if len(tools) == 0 {
		return nil, nil, nil
	}

	toolInfos := make([]*schema.ToolInfo, 0, len(tools))
	toolsMap := make(map[string]tool.InvokableTool, len(tools))
	for _, t := range tools {
		invokableTool, ok := t.(tool.InvokableTool)
		if !ok {
			continue
		}

		info, err := t.Info(ctx)
		if err != nil {
			return nil, nil, err
		}

		if _, exists := toolsMap[info.Name]; exists {
			return nil, nil, fmt.Errorf("duplicate tool name: %s", info.Name)
		}

		toolInfos = append(toolInfos, info)
		toolsMap[info.Name] = invokableTool
	}

	return toolInfos, toolsMap, nil
}

func init() {
	registerTool(bash.BashToolName, bash.GetBashTool)
	registerTool(duckduckgo.DuckDuckGoToolName, duckduckgo.GetDuckDuckGoTool)
	registerTool(feed.PushFeedToolName, feed.GetPushFeedTool)
	registerTool(skills.LoadSkillToolName, skills.LoadSkillTool)
	registerTool(agents.RunAgentToolName, agents.GetAgentsTool)
}
