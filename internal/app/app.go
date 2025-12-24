package app

import (
	"context"
	"fmt"
	"sync"

	"github.com/zjregee/alter/internal/models"
	"github.com/zjregee/alter/internal/service"
)

type App struct {
	ctx           context.Context
	ctxMu         sync.RWMutex
	agentService  *service.AgentService
	threadOrder   []string
	threadOrderMu sync.RWMutex
	workspaceMu   sync.RWMutex
	workspaces    []*models.WorkspaceInfo
	currentWSID   string
	defaultWSID   string
}

func NewApp() *App {
	agentService, err := service.NewAgentService()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize Agent service: %v", err))
	}

	app := &App{
		agentService: agentService,
		threadOrder:  []string{},
	}
	app.initWorkspaces()
	return app
}

func (a *App) Startup(ctx context.Context) {
	a.ctxMu.Lock()
	a.ctx = ctx
	a.ctxMu.Unlock()
}
