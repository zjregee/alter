package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zjregee/alter/internal/models"
	"github.com/zjregee/alter/internal/service/storage"
)

// defaultWorkspacePath returns the default workspace path.
func defaultWorkspacePath() string {
	if configDir, err := os.UserConfigDir(); err == nil && configDir != "" {
		candidate := filepath.Join(configDir, "Alter")
		if err := os.MkdirAll(candidate, 0o755); err == nil {
			return candidate
		}
	}

	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return home
	}

	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		return cwd
	}

	if abs, err := filepath.Abs("."); err == nil {
		return abs
	}

	return ""
}

// initWorkspaces initializes the workspace list.
func (a *App) initWorkspaces() {
	workspacePaths := []string{}
	storedCurrent := ""
	storedDefault := ""
	if stored, err := storage.LoadWorkspaceState(); err == nil && stored != nil {
		workspacePaths = append(workspacePaths, stored.Paths...)
		storedCurrent = strings.TrimSpace(stored.CurrentID)
		storedDefault = strings.TrimSpace(stored.DefaultID)
	}
	if a.agentService != nil {
		if len(workspacePaths) == 0 {
			workspacePaths = a.agentService.ListWorkspaces()
		}
		for _, thread := range a.agentService.ListThreads() {
			if thread == nil || thread.WorkDir == "" {
				continue
			}
			if err := ensureWorkspacePath(thread.WorkDir); err != nil {
				continue
			}
			workspacePaths = append(workspacePaths, thread.WorkDir)
		}
	}
	if len(workspacePaths) > 0 {
		allowed := map[string]struct{}{}
		if a.agentService != nil {
			for _, candidate := range a.agentService.ListWorkspaces() {
				allowed[candidate] = struct{}{}
			}
		}
		seen := make(map[string]struct{}, len(workspacePaths))
		deduped := make([]string, 0, len(workspacePaths))
		for _, path := range workspacePaths {
			path = filepath.Clean(strings.TrimSpace(path))
			if path == "" {
				continue
			}
			if !filepath.IsAbs(path) {
				continue
			}
			if err := ensureWorkspacePath(path); err != nil {
				continue
			}
			if len(allowed) > 0 {
				if _, ok := allowed[path]; !ok {
					continue
				}
			}
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			deduped = append(deduped, path)
		}
		workspacePaths = deduped
	}
	if len(workspacePaths) == 0 {
		defaultPath := defaultWorkspacePath()
		if defaultPath == "" {
			defaultPath = "."
		}
		workspacePaths = []string{defaultPath}
	}

	defaultID := ""
	if len(workspacePaths) > 0 {
		defaultID = workspacePaths[0]
	}
	for _, path := range workspacePaths {
		if path == storedDefault {
			defaultID = storedDefault
			break
		}
	}
	currentID := defaultID
	for _, path := range workspacePaths {
		if path == storedCurrent {
			currentID = storedCurrent
			break
		}
	}

	workspaces := make([]*models.WorkspaceInfo, 0, len(workspacePaths))
	for _, path := range workspacePaths {
		name := filepath.Base(path)
		if name == "" || name == "." || name == string(filepath.Separator) {
			name = path
		}
		workspaces = append(workspaces, &models.WorkspaceInfo{
			ID:        path,
			Name:      name,
			Path:      path,
			IsDefault: path == defaultID,
		})
	}

	a.workspaceMu.Lock()
	a.workspaces = workspaces
	a.currentWSID = currentID
	a.defaultWSID = defaultID
	a.workspaceMu.Unlock()

	_ = a.persistWorkspaces()
}

// ensureWorkspacePath validates that the path exists and is a directory.
func ensureWorkspacePath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("workspace path is not a directory: %s", path)
	}
	return nil
}

// addWorkspace adds a new workspace or switches to an existing one.
func (a *App) addWorkspace(path string) (*models.WorkspaceInfo, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" {
		return nil, fmt.Errorf("workspace path is required")
	}
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("workspace path must be absolute: %s", path)
	}
	if err := ensureWorkspacePath(path); err != nil {
		return nil, err
	}
	if a.agentService != nil {
		available := a.agentService.ListWorkspaces()
		allowed := false
		for _, candidate := range available {
			if candidate == path {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("workspace path is not available: %s", path)
		}
	}

	a.workspaceMu.Lock()

	for _, ws := range a.workspaces {
		if ws != nil && ws.ID == path {
			a.currentWSID = ws.ID
			a.workspaceMu.Unlock()
			if err := a.persistWorkspaces(); err != nil {
				return ws, err
			}
			return ws, nil
		}
	}

	name := filepath.Base(path)
	if name == "" || name == "." || name == string(filepath.Separator) {
		name = path
	}

	newWorkspace := &models.WorkspaceInfo{
		ID:   path,
		Name: name,
		Path: path,
	}

	a.workspaces = append(a.workspaces, newWorkspace)
	a.currentWSID = newWorkspace.ID
	a.workspaceMu.Unlock()

	if err := a.persistWorkspaces(); err != nil {
		return newWorkspace, err
	}
	return newWorkspace, nil
}

// currentWorkspacePath returns the current workspace path.
func (a *App) currentWorkspacePath() string {
	a.workspaceMu.RLock()
	defer a.workspaceMu.RUnlock()

	for _, ws := range a.workspaces {
		if ws != nil && ws.ID == a.currentWSID {
			return ws.Path
		}
	}
	return ""
}

// workspacePathByID returns the path for a given workspace ID.
func (a *App) workspacePathByID(id string) (string, error) {
	a.workspaceMu.RLock()
	defer a.workspaceMu.RUnlock()

	for _, ws := range a.workspaces {
		if ws != nil && ws.ID == id {
			return ws.Path, nil
		}
	}

	return "", fmt.Errorf("workspace not found: %s", id)
}

// ListWorkspaces returns a list of all workspaces.
func (a *App) ListWorkspaces() []*models.WorkspaceInfo {
	a.workspaceMu.RLock()
	defer a.workspaceMu.RUnlock()

	workspaces := make([]*models.WorkspaceInfo, 0, len(a.workspaces))
	for _, ws := range a.workspaces {
		if ws == nil {
			continue
		}
		copyWS := *ws
		copyWS.IsActive = ws.ID == a.currentWSID
		workspaces = append(workspaces, &copyWS)
	}
	return workspaces
}

// SetWorkspace sets the workspace for a thread.
func (a *App) SetWorkspace(workspaceID string, threadID string) error {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return fmt.Errorf("workspace ID is required")
	}
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return fmt.Errorf("thread ID is required")
	}

	path, err := a.workspacePathByID(workspaceID)
	if err != nil {
		return err
	}
	if a.agentService == nil {
		return fmt.Errorf("agent service not initialized")
	}
	if err := a.agentService.UpdateThreadWorkDir(threadID, path); err != nil {
		return err
	}

	a.workspaceMu.Lock()
	a.currentWSID = workspaceID
	a.workspaceMu.Unlock()

	if err := a.persistWorkspaces(); err != nil {
		return err
	}
	return nil
}

// ResetWorkspace resets the workspace to the default one.
func (a *App) ResetWorkspace(threadID string) error {
	a.workspaceMu.RLock()
	defaultID := a.defaultWSID
	a.workspaceMu.RUnlock()

	if defaultID == "" {
		return fmt.Errorf("default workspace is not configured")
	}
	return a.SetWorkspace(defaultID, threadID)
}

func (a *App) persistWorkspaces() error {
	a.workspaceMu.RLock()
	defer a.workspaceMu.RUnlock()

	paths := make([]string, 0, len(a.workspaces))
	for _, ws := range a.workspaces {
		if ws == nil || ws.ID == "" {
			continue
		}
		paths = append(paths, ws.ID)
	}

	return storage.SaveWorkspaceState(paths, a.currentWSID, a.defaultWSID)
}
