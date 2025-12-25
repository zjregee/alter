package app

import (
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/zjregee/alter/internal/models"
)

func (a *App) ListWorkspaces() []*models.WorkspaceInfo {
	if a.agentService == nil {
		return []*models.WorkspaceInfo{}
	}

	return a.agentService.ListWorkspaces()
}

func (a *App) UpdateWorkspace(threadID string, workspacePath string) error {
	if a.agentService == nil {
		return fmt.Errorf("agent service not initialized")
	}
	if threadID == "" {
		return fmt.Errorf("thread ID is required")
	}

	return a.agentService.UpdateThreadWorkDir(threadID, workspacePath)
}

func (a *App) AddWorkspace(workspacePath string) error {
	if a.agentService == nil {
		return fmt.Errorf("agent service not initialized")
	}

	return a.agentService.AddWorkspace(workspacePath)
}

func (a *App) DeleteWorkspace(workspacePath string) error {
	if a.agentService == nil {
		return fmt.Errorf("agent service not initialized")
	}

	return a.agentService.DeleteWorkspace(workspacePath)
}

func (a *App) SelectWorkspace(threadID string) (string, error) {
	if a.agentService == nil {
		return "", fmt.Errorf("agent service not initialized")
	}
	if threadID == "" {
		return "", fmt.Errorf("thread ID is required")
	}

	defaultDirectory, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	workspacePath, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:                "Select Workspace Directory",
		DefaultDirectory:     defaultDirectory,
		CanCreateDirectories: true,
	})
	if err != nil {
		return "", err
	}
	if workspacePath == "" {
		return "", nil
	}

	err = a.AddWorkspace(workspacePath)
	if err != nil {
		return "", err
	}

	err = a.UpdateWorkspace(threadID, workspacePath)
	if err != nil {
		return "", err
	}

	return workspacePath, nil
}
