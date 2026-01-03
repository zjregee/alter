package storage

import (
	"encoding/json"
	"fmt"

	"github.com/zjregee/alter/internal/models"
)

const (
	workspaceInfosKey = "workspace:infos"
)

const defaultWorkspacePath = "/Users/zjregee/Code/alter"

type WorkspaceInfosRecord struct {
	Infos []*models.WorkspaceInfo `json:"infos"`
}

func SaveWorkspaceInfos(infos []*models.WorkspaceInfo) error {
	record := WorkspaceInfosRecord{
		Infos: infos,
	}

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal workspace infos: %w", err)
	}

	return Put([]byte(workspaceInfosKey), data)
}

func LoadWorkspaceInfos() (*WorkspaceInfosRecord, error) {
	value, err := Get([]byte(workspaceInfosKey))
	if err != nil {
		return nil, err
	}

	if len(value) == 0 {
		return &WorkspaceInfosRecord{
			Infos: []*models.WorkspaceInfo{},
		}, nil
	}

	var record WorkspaceInfosRecord
	if err := json.Unmarshal(value, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal workspace infos: %w", err)
	}

	return &record, nil
}

func initWorkspaceInfos() {
	infos := []*models.WorkspaceInfo{
		{
			Path:      defaultWorkspacePath,
			IsDefault: true,
		},
	}

	if err := SaveWorkspaceInfos(infos); err != nil {
		panic(err)
	}
}
