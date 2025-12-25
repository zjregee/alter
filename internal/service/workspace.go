package service

import (
	"os"

	"github.com/zjregee/alter/internal/models"
	"github.com/zjregee/alter/internal/service/storage"
)

func getDefaultWorkspace() *models.WorkspaceInfo {
	infos, err := storage.LoadWorkspaceInfos()
	if err != nil {
		return nil
	}

	for _, info := range infos.Infos {
		if info.IsDefault {
			return info
		}
	}

	return nil
}

func getAvailableWorkspaces() []*models.WorkspaceInfo {
	infos, err := storage.LoadWorkspaceInfos()
	if err != nil {
		return []*models.WorkspaceInfo{}
	}

	return infos.Infos
}

func isWorkspacePathAvailable(workspacePath string) bool {
	infos := getAvailableWorkspaces()
	for _, info := range infos {
		if info.Path == workspacePath {
			info, err := os.Stat(workspacePath)
			if err != nil || !info.IsDir() {
				return false
			}

			return true
		}
	}

	return false
}

func addWorkspace(workspacePath string) error {
	infos, err := storage.LoadWorkspaceInfos()
	if err != nil {
		return err
	}

	infos.Infos = append(infos.Infos, &models.WorkspaceInfo{
		Path:      workspacePath,
		IsDefault: false,
	})

	return storage.SaveWorkspaceInfos(infos.Infos)
}

func deleteWorkspace(workspacePath string) error {
	infos, err := storage.LoadWorkspaceInfos()
	if err != nil {
		return err
	}

	for i, info := range infos.Infos {
		if info.Path == workspacePath {
			infos.Infos = append(infos.Infos[:i], infos.Infos[i+1:]...)
			break
		}
	}

	return storage.SaveWorkspaceInfos(infos.Infos)
}
