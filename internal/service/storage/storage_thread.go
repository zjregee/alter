package storage

import (
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/schema"

	"github.com/zjregee/alter/internal/models"
)

const (
	threadKeyPrefix = "thread:"
)

type ThreadRecord struct {
	Info    *models.ThreadInfo          `json:"info"`
	History []*schema.Message           `json:"history"`
	Turns   []*models.ThreadMessageTurn `json:"turns"`
	Stats   *models.AgentStats          `json:"stats"`
}

func SaveThread(info *models.ThreadInfo, history []*schema.Message, turns []*models.ThreadMessageTurn, stats *models.AgentStats) error {
	if info == nil {
		return fmt.Errorf("thread info is required")
	}

	payload := ThreadRecord{
		Info:    info,
		History: history,
		Turns:   turns,
		Stats:   stats,
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
