package agents

import (
	"context"
	"fmt"
	"sync"
)

type manager struct {
	mu     sync.RWMutex
	agents map[string]Agent
	slots  chan struct{}
}

func (m *manager) register(agent Agent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if agent == nil {
		return fmt.Errorf("agent cannot be nil")
	}

	id := agent.ID()
	if id == "" {
		return fmt.Errorf("agent ID cannot be empty")
	}

	if _, exists := m.agents[id]; exists {
		return fmt.Errorf("agent with ID %s already registered", id)
	}

	m.agents[id] = agent
	return nil
}

func (m *manager) unregister(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, exists := m.agents[id]
	if !exists {
		return fmt.Errorf("agent with ID %s not found", id)
	}

	status := agent.Status()
	if status.State == StateRunning {
		return fmt.Errorf("cannot unregister running agent %s", id)
	}

	delete(m.agents, id)
	return nil
}

func (m *manager) get(id string) (Agent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agent, exists := m.agents[id]
	if !exists {
		return nil, fmt.Errorf("agent with ID %s not found", id)
	}

	return agent, nil
}

func (m *manager) create(agentType AgentType, cfg Config) (string, error) {
	factoriesMu.RLock()
	factory, exists := factories[agentType]
	factoriesMu.RUnlock()

	if !exists {
		return "", fmt.Errorf("no factory registered for agent type: %s", agentType)
	}

	agent := factory(cfg)
	if agent == nil {
		return "", fmt.Errorf("factory returned nil agent for type: %s", agentType)
	}

	if err := m.register(agent); err != nil {
		return "", fmt.Errorf("failed to register agent: %w", err)
	}

	return agent.ID(), nil
}

func (m *manager) delete(id string) error {
	return m.unregister(id)
}

func (m *manager) start(ctx context.Context, id string) error {
	agent, err := m.get(id)
	if err != nil {
		return err
	}

	status := agent.Status()
	if status.State != StateIdle {
		return fmt.Errorf("agent %s is not idle (current state: %s)", id, status.State)
	}

	select {
	case m.slots <- struct{}{}:
	case <-ctx.Done():
		return fmt.Errorf("context cancelled while waiting for slot")
	}

	err = agent.Execute(ctx)
	if err != nil {
		<-m.slots
		return fmt.Errorf("failed to start agent %s: %w", id, err)
	}

	go func() {
		defer func() {
			<-m.slots
		}()
		<-agent.Done()
	}()

	return nil
}

func (m *manager) stop(ctx context.Context, id string) error {
	agent, err := m.get(id)
	if err != nil {
		return err
	}

	status := agent.Status()
	if status.State != StateRunning {
		return fmt.Errorf("agent %s is not running (current state: %s)", id, status.State)
	}

	return agent.Cancel(ctx)
}
