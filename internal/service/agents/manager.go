package agents

import (
	"context"
	"sync"
)

const (
	maxConcurrentAgents = 3
)

type agentFactory func(cfg Config) Agent

var (
	factoriesMu sync.RWMutex
	factories   = make(map[AgentType]agentFactory)

	instance *manager
)

func RegisterFactory(agentType AgentType, factory agentFactory) {
	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	factories[agentType] = factory
}

func init() {
	instance = &manager{
		agents: make(map[string]Agent),
		slots:  make(chan struct{}, maxConcurrentAgents),
	}
}

func CreateAgent(agentType AgentType, cfg Config) (string, error) {
	agentID, err := instance.create(agentType, cfg)
	if err != nil {
		return "", err
	}
	return agentID, nil
}

func StartAgentExecution(ctx context.Context, id string) error {
	return instance.start(ctx, id)
}

func StopAgent(ctx context.Context, id string) error {
	return instance.stop(ctx, id)
}

func DeleteAgent(id string) error {
	return instance.delete(id)
}
