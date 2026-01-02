package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	mcpp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	MCPClientName    = "alter"
	MCPClientVersion = "1.0.0"
)

type stdioMCPConfig struct {
	Command   string
	Args      []string
	Env       []string
	ToolNames []string
}

var (
	stdioOnce    sync.Once
	stdioTools   []tool.BaseTool
	stdioCleanup func()
)

func getDefaultStdioMCPConfig() (*stdioMCPConfig, error) {
	return &stdioMCPConfig{
		Command: "npx",
		Args: []string{
			"@playwright/mcp@latest",
		},
	}, nil
}

func GetMCPTools(ctx context.Context) ([]tool.BaseTool, error) {
	stdioOnce.Do(func() {
		cfg, err := getDefaultStdioMCPConfig()
		if err != nil {
			panic(err)
		}
		stdioTools, stdioCleanup, err = getStdioMCPTools(ctx, cfg)
		if err != nil {
			panic(err)
		}
	})

	return stdioTools, nil
}

func CloseMCPTools() {
	if stdioCleanup != nil {
		stdioCleanup()
	}
}

func getStdioMCPTools(ctx context.Context, cfg *stdioMCPConfig) ([]tool.BaseTool, func(), error) {
	if cfg == nil {
		return nil, nil, fmt.Errorf("mcp config must be provided")
	}
	if strings.TrimSpace(cfg.Command) == "" {
		return nil, nil, fmt.Errorf("mcp command must be provided")
	}

	cli, err := client.NewStdioMCPClient(cfg.Command, cfg.Env, cfg.Args...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create mcp client: %w", err)
	}

	cleanup := func() {
		if err := cli.Close(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "failed to close mcp client: %v\n", err)
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
