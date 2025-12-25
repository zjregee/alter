package app

import (
	"context"
	"fmt"
	"sync"

	"github.com/zjregee/alter/internal/service"
)

type App struct {
	ctx          context.Context
	agentService *service.AgentService

	threadOrder   []string
	threadOrderMu sync.RWMutex
}

func NewApp() *App {
	return &App{}
}

func (a *App) Startup(ctx context.Context) {
	agentService, err := service.NewAgentService(ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize agent service: %v", err))
	}

	a.ctx = ctx
	a.agentService = agentService
}
