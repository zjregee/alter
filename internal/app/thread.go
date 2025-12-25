package app

import (
	"context"
	"fmt"

	"github.com/zjregee/alter/internal/models"
)

func (a *App) CreateThread() (string, error) {
	if a.agentService == nil {
		return "", fmt.Errorf("agent service not initialized")
	}

	threadID, err := a.agentService.CreateThread(context.Background())
	if err != nil {
		return "", err
	}

	a.threadOrderMu.Lock()
	a.threadOrder = append([]string{threadID}, a.threadOrder...)
	a.threadOrderMu.Unlock()

	return threadID, nil
}

func (a *App) ListThreads() []*models.ThreadInfo {
	if a.agentService == nil {
		return []*models.ThreadInfo{}
	}

	threads := a.agentService.ListThreads()

	a.threadOrderMu.Lock()
	a.syncThreadOrder(threads)
	ordered := a.orderedThreads(threads)
	a.threadOrderMu.Unlock()

	return ordered
}

func (a *App) DeleteThread(threadID string) error {
	if a.agentService == nil {
		return fmt.Errorf("agent service not initialized")
	}
	if threadID == "" {
		return fmt.Errorf("thread ID is required")
	}
	if err := a.agentService.DeleteThread(threadID); err != nil {
		return err
	}

	a.threadOrderMu.Lock()
	for i, id := range a.threadOrder {
		if id == threadID {
			a.threadOrder = append(a.threadOrder[:i], a.threadOrder[i+1:]...)
			break
		}
	}
	a.threadOrderMu.Unlock()

	return nil
}

func (a *App) GetThreadMessages(threadID string) ([]*models.ThreadMessage, error) {
	if a.agentService == nil {
		return nil, fmt.Errorf("agent service not initialized")
	}
	if threadID == "" {
		return nil, fmt.Errorf("thread ID is required")
	}

	return a.agentService.GetThreadMessages(threadID)
}

func (a *App) UpdateThreadModel(threadID, modelID string) error {
	if a.agentService == nil {
		return fmt.Errorf("agent service not initialized")
	}
	if threadID == "" {
		return fmt.Errorf("thread ID is required")
	}
	if modelID == "" {
		return fmt.Errorf("model ID is required")
	}

	return a.agentService.UpdateThreadModel(threadID, modelID)
}

func (a *App) ReorderThreads(order []string) error {
	if a.agentService == nil {
		return fmt.Errorf("agent service not initialized")
	}

	threads := a.agentService.ListThreads()
	if len(order) != len(threads) {
		return fmt.Errorf("invalid order length: expected %d, got %d", len(threads), len(order))
	}

	exists := make(map[string]struct{}, len(threads))
	for _, thread := range threads {
		exists[thread.ID] = struct{}{}
	}

	seen := make(map[string]bool, len(order))
	for _, id := range order {
		if _, ok := exists[id]; !ok {
			return fmt.Errorf("thread not found: %s", id)
		}
		if seen[id] {
			return fmt.Errorf("duplicate thread ID in order: %s", id)
		}

		seen[id] = true
	}

	a.threadOrderMu.Lock()
	a.threadOrder = order
	a.threadOrderMu.Unlock()

	return nil
}

func (a *App) syncThreadOrder(threads []*models.ThreadInfo) {
	if len(threads) == 0 {
		a.threadOrder = []string{}
		return
	}

	exists := make(map[string]struct{}, len(threads))
	for _, thread := range threads {
		exists[thread.ID] = struct{}{}
	}

	filtered := make([]string, 0, len(a.threadOrder))
	seen := make(map[string]struct{}, len(threads))
	for _, id := range a.threadOrder {
		if _, ok := exists[id]; ok {
			filtered = append(filtered, id)
			seen[id] = struct{}{}
		}
	}

	for _, thread := range threads {
		if _, ok := seen[thread.ID]; !ok {
			filtered = append(filtered, thread.ID)
		}
	}

	a.threadOrder = filtered
}

func (a *App) orderedThreads(threads []*models.ThreadInfo) []*models.ThreadInfo {
	if len(threads) == 0 {
		return []*models.ThreadInfo{}
	}

	byID := make(map[string]*models.ThreadInfo, len(threads))
	for _, thread := range threads {
		byID[thread.ID] = thread
	}

	ordered := make([]*models.ThreadInfo, 0, len(threads))
	seen := make(map[string]struct{}, len(threads))
	for _, id := range a.threadOrder {
		if thread, ok := byID[id]; ok {
			ordered = append(ordered, thread)
			seen[id] = struct{}{}
		}
	}

	for _, thread := range threads {
		if _, ok := seen[thread.ID]; !ok {
			ordered = append(ordered, thread)
		}
	}

	return ordered
}
