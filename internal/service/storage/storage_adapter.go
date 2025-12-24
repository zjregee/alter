package storage

import (
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/schema"

	"github.com/zjregee/alter/internal/models"
)

const (
	threadKeyPrefix   = "thread:"
	workspaceInfosKey = "workspace:infos"
)

const defaultWorkspacePath = "/Users/zjregee/Code/alter"

type ThreadRecord struct {
	Info              *models.ThreadInfo `json:"info"`
	Messages          []*schema.Message  `json:"messages"`
	MessageTimestamps []int64            `json:"message_timestamps"`
	Stats             *models.AgentStats `json:"stats"`
}

type WorkspaceInfosRecord struct {
	Infos []*models.WorkspaceInfo `json:"infos"`
}

func SaveThread(info *models.ThreadInfo, messages []*schema.Message, messageTimestamps []int64, stats *models.AgentStats) error {
	if info == nil {
		return fmt.Errorf("thread info is required")
	}

	payload := ThreadRecord{
		Info:              info,
		Messages:          messages,
		MessageTimestamps: messageTimestamps,
		Stats:             stats,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal thread %s: %w", info.ID, err)
	}

	return Put([]byte(threadKeyPrefix+info.ID), data)
}

func LoadThreads() ([]*ThreadRecord, error) {
	entries, err := List([]byte(threadKeyPrefix))
	if err != nil {
		return nil, err
	}

	threads := make([]*ThreadRecord, 0, len(entries))
	for key, value := range entries {
		if len(value) == 0 {
			continue
		}

		var stored ThreadRecord
		if err := json.Unmarshal(value, &stored); err != nil {
			return nil, fmt.Errorf("failed to unmarshal thread %s: %w", key, err)
		}

		if stored.Info == nil {
			continue
		}

		threads = append(threads, &stored)
	}

	return threads, nil
}

func DeleteThread(id string) error {
	if id == "" {
		return fmt.Errorf("thread id is required")
	}

	return Delete([]byte(threadKeyPrefix + id))
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
