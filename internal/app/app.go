package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zjregee/alter/internal/service"
	"github.com/zjregee/alter/internal/utils"
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

func (a *App) Shutdown(ctx context.Context) {
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := service.ShutdownTelemetry(shutdownCtx); err != nil {
		utils.GetLogger().Printf("Warning: Failed to shutdown telemetry: %v\n", err)
	}
}
