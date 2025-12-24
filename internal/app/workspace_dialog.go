package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/zjregee/alter/internal/models"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// SelectWorkspaceDirectory opens a directory picker dialog and adds the selected workspace.
func (a *App) SelectWorkspaceDirectory(threadID string) (*models.WorkspaceInfo, error) {
	a.ctxMu.RLock()
	appCtx := a.ctx
	a.ctxMu.RUnlock()

	if appCtx == nil {
		appCtx = context.Background()
	}

	path, err := runtime.OpenDirectoryDialog(appCtx, runtime.OpenDialogOptions{
		Title:                "Select Workspace Directory",
		DefaultDirectory:     a.currentWorkspacePath(),
		CanCreateDirectories: true,
	})
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil
	}

	workspace, err := a.addWorkspace(path)
	if err != nil {
		return nil, err
	}
	if a.agentService == nil {
		return workspace, fmt.Errorf("agent service not initialized")
	}
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return workspace, fmt.Errorf("thread ID is required")
	}
	if err := a.agentService.UpdateThreadWorkDir(threadID, workspace.Path); err != nil {
		return workspace, err
	}

	return workspace, nil
}
