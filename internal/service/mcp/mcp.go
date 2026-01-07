package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	mcpp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/zjregee/alter/internal/utils"
)

const (
	MCPClientName    = "alter"
	MCPClientVersion = "1.0.0"
	ConfigDirName    = ".alter/mcp"
)

type MCPConfig struct {
	Command   string            `json:"command"`
	Args      []string          `json:"args"`
	Env       map[string]string `json:"env"`
	ToolNames []string          `json:"toolNames"`
	Disabled  bool              `json:"disabled"`
}

var (
	mcpOnce     sync.Once
	mcpTools    []tool.BaseTool
	mcpCleanups []func()
)

func GetMCPTools(ctx context.Context) ([]tool.BaseTool, error) {
	mcpOnce.Do(func() {
		tools, cleanups, err := initializeMCPTools(ctx)
		if err != nil {
			utils.GetLogger().Printf("Error initializing MCP tools: %v", err)
		}

		mcpTools = tools
		mcpCleanups = cleanups
	})

	return mcpTools, nil
}

func CloseMCPTools() {
	for _, cleanup := range mcpCleanups {
		if cleanup != nil {
			cleanup()
		}
	}

	mcpCleanups = nil
}

func initializeMCPTools(ctx context.Context) ([]tool.BaseTool, []func(), error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ConfigDirName)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return nil, nil, nil
	}

	entries, err := os.ReadDir(configDir)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read mcp config directory: %w", err)
	}

	var allTools []tool.BaseTool
	var allCleanups []func()

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(configDir, entry.Name())
		cfg, err := loadConfig(path)
		if err != nil {
			utils.GetLogger().Printf("Failed to load mcp config %s: %v", entry.Name(), err)
			continue
		}

		if cfg.Disabled {
			continue
		}

		tools, cleanup, err := startMCPClient(ctx, cfg)
		if err != nil {
			utils.GetLogger().Printf("Failed to start mcp client for %s: %v", entry.Name(), err)
			if cleanup != nil {
				cleanup()
			}
			continue
		}

		allTools = append(allTools, tools...)
		allCleanups = append(allCleanups, cleanup)
	}

	return allTools, allCleanups, nil
}

func loadConfig(path string) (*MCPConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg MCPConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func startMCPClient(ctx context.Context, cfg *MCPConfig) ([]tool.BaseTool, func(), error) {
	if strings.TrimSpace(cfg.Command) == "" {
		return nil, nil, fmt.Errorf("command is required")
	}

	var env []string
	for k, v := range cfg.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	fullEnv := append(os.Environ(), env...)

	cli, err := client.NewStdioMCPClient(cfg.Command, fullEnv, cfg.Args...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create mcp client: %w", err)
	}

	cleanup := func() {
		if err := cli.Close(); err != nil {
			utils.GetLogger().Printf("failed to close mcp client: %v", err)
		}
	}

	if err := cli.Start(ctx); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("failed to start mcp client: %w", err)
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    MCPClientName,
		Version: MCPClientVersion,
	}

	if _, err := cli.Initialize(ctx, initRequest); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("failed to initialize mcp client: %w", err)
	}

	tools, err := mcpp.GetTools(ctx, &mcpp.Config{
		Cli:          cli,
		ToolNameList: cfg.ToolNames,
	})
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("failed to list mcp tools: %w", err)
	}

	return tools, cleanup, nil
}
