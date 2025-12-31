package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/cloudwego/eino/schema"

	"github.com/zjregee/alter/internal/models"
	"github.com/zjregee/alter/internal/service/storage"
)

const defaultThreadTitle = "New chat"

type AgentService struct {
	agents map[string]*Thread
	mu     sync.RWMutex
}

type Thread struct {
	Info  *models.ThreadInfo
	Agent *Agent
}

func newDefaultAgentConfig() models.AgentConfig {
	return models.AgentConfig{
		ModelID: getDefaultModelInfo().ID,
		WorkDir: getDefaultWorkspace().Path,
	}
}

func NewAgentService(ctx context.Context) (*AgentService, error) {
	service := &AgentService{
		agents: make(map[string]*Thread),
	}

	if err := service.loadThreadsFromStorage(ctx); err != nil {
		return nil, err
	}

	return service, nil
}

func (s *AgentService) ListModels() []*models.ModelInfo {
	return getAvailableModelInfos()
}

func (s *AgentService) ListWorkspaces() []*models.WorkspaceInfo {
	return getAvailableWorkspaces()
}

func (s *AgentService) AddWorkspace(workspacePath string) error {
	return addWorkspace(workspacePath)
}

func (s *AgentService) DeleteWorkspace(workspacePath string) error {
	return deleteWorkspace(workspacePath)
}

func (s *AgentService) CreateThread(ctx context.Context) (string, error) {
	config := newDefaultAgentConfig()
	agent, err := NewAgent(ctx, config)
	if err != nil {
		return "", err
	}

	thread := &Thread{
		Info: &models.ThreadInfo{
			ID:        agent.ID(),
			Title:     defaultThreadTitle,
			Model:     agent.Config().ModelID,
			WorkDir:   agent.Config().WorkDir,
			CreatedAt: time.Now().UnixMilli(),
			UpdatedAt: time.Now().UnixMilli(),
		},
		Agent: agent,
	}

	if err := s.persistThread(thread); err != nil {
		return "", err
	}

	s.mu.Lock()
	s.agents[thread.Info.ID] = thread
	s.mu.Unlock()

	return thread.Info.ID, nil
}

func (s *AgentService) ListThreads() []*models.ThreadInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	infos := make([]*models.ThreadInfo, 0, len(s.agents))
	for _, thread := range s.agents {
		infos = append(infos, thread.Info)
	}

	return infos
}

func (s *AgentService) DeleteThread(id string) error {
	s.mu.RLock()
	_, exists := s.agents[id]
	s.mu.RUnlock()

	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("thread not found: %s", id)
	}

	if err := storage.DeleteThread(id); err != nil {
		return err
	}

	s.mu.Lock()
	delete(s.agents, id)
	s.mu.Unlock()

	return nil
}

func (s *AgentService) StreamRequestToThread(ctx context.Context, id string, userInput string) (<-chan models.AgentMessage, error) {
	s.mu.RLock()
	_, exists := s.agents[id]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("thread not found: %s", id)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current, found := s.agents[id]
	if !found {
		return nil, fmt.Errorf("thread not found: %s", id)
	}

	current.Info.UpdatedAt = time.Now().UnixMilli()
	originChan := current.Agent.StreamRequest(ctx, userInput)

	outChan := make(chan models.AgentMessage)
	go func(thread *Thread) {
		for msg := range originChan {
			outChan <- msg
		}

		close(outChan)

		if err := s.persistThread(thread); err != nil {
			fmt.Printf("Failed to persist thread %s: %v\n", thread.Info.ID, err)
		}
	}(current)

	return outChan, nil
}

func (s *AgentService) EditAndResendRequestToThread(ctx context.Context, id string, userMessageIndex int, userInput string) (<-chan models.AgentMessage, error) {
	s.mu.RLock()
	_, exists := s.agents[id]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("thread not found: %s", id)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current, found := s.agents[id]
	if !found {
		return nil, fmt.Errorf("thread not found: %s", id)
	}

	turns := current.Agent.GetTurns()
	userIndex := -1
	turnIndex := -1
	for i, turn := range turns {
		if turn.Role != schema.User {
			continue
		}
		userIndex += 1
		if userIndex == userMessageIndex {
			turnIndex = i
			break
		}
	}

	if turnIndex == -1 {
		return nil, fmt.Errorf("invalid user message index: %d", userMessageIndex)
	}

	if err := current.Agent.TruncateMessagesSince(turnIndex); err != nil {
		return nil, fmt.Errorf("failed to truncate thread messages since index %d: %w", turnIndex, err)
	}

	current.Info.UpdatedAt = time.Now().UnixMilli()
	originChan := current.Agent.StreamRequest(ctx, userInput)

	outChan := make(chan models.AgentMessage)
	go func(thread *Thread) {
		for msg := range originChan {
			outChan <- msg
		}

		close(outChan)

		if err := s.persistThread(thread); err != nil {
			fmt.Printf("Failed to persist thread %s: %v\n", thread.Info.ID, err)
		}
	}(current)

	return outChan, nil
}

func (s *AgentService) RegenerateLastResponseToThread(ctx context.Context, id string) (<-chan models.AgentMessage, error) {
	s.mu.RLock()
	_, exists := s.agents[id]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("thread not found: %s", id)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current, found := s.agents[id]
	if !found {
		return nil, fmt.Errorf("thread not found: %s", id)
	}

	turns := current.Agent.GetTurns()
	lastUserTurnIndex := -1
	lastUserContent := ""
	for i := len(turns) - 1; i >= 0; i -= 1 {
		if turns[i].Role != schema.User {
			continue
		}

		var userMsg models.UserMessage
		if err := json.Unmarshal(turns[i].Events[0].Content, &userMsg); err != nil {
			continue
		}

		lastUserTurnIndex = i
		lastUserContent = userMsg.Content
		break
	}

	if lastUserTurnIndex == -1 {
		return nil, fmt.Errorf("no user message found to regenerate from")
	}

	if err := current.Agent.TruncateMessagesSince(lastUserTurnIndex); err != nil {
		return nil, fmt.Errorf("failed to truncate thread messages since index %d: %w", lastUserTurnIndex, err)
	}

	current.Info.UpdatedAt = time.Now().UnixMilli()
	originChan := current.Agent.StreamRequest(ctx, lastUserContent)

	outChan := make(chan models.AgentMessage)
	go func(thread *Thread) {
		for msg := range originChan {
			outChan <- msg
		}

		close(outChan)

		if err := s.persistThread(thread); err != nil {
			fmt.Printf("Failed to persist thread %s: %v\n", thread.Info.ID, err)
		}
	}(current)

	return outChan, nil
}

func (s *AgentService) CancelStreamRequestToThread(id string) error {
	s.mu.RLock()
	_, exists := s.agents[id]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("thread not found: %s", id)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current, found := s.agents[id]
	if !found {
		return fmt.Errorf("thread not found: %s", id)
	}

	current.Agent.CancelStreamRequest()
	return nil
}

func (s *AgentService) GetThreadMessages(id string) ([]*models.ThreadMessageTurn, error) {
	s.mu.RLock()
	thread, exists := s.agents[id]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("thread not found: %s", id)
	}

	return thread.Agent.GetTurns(), nil
}

func (s *AgentService) IsFirstMessageToThread(id string) (bool, error) {
	s.mu.RLock()
	thread, exists := s.agents[id]
	s.mu.RUnlock()

	if !exists {
		return false, fmt.Errorf("thread not found: %s", id)
	}

	return len(thread.Agent.GetTurns()) == 0, nil
}

func (s *AgentService) UpdateThreadModel(id string, modelID string) error {
	s.mu.RLock()
	thread, exists := s.agents[id]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("thread not found: %s", id)
	}

	if err := thread.Agent.UpdateModelID(modelID); err != nil {
		return fmt.Errorf("failed to update thread model: %w", err)
	}

	s.mu.Lock()
	current, found := s.agents[id]
	if found {
		current.Info.Model = modelID
		current.Info.UpdatedAt = time.Now().UnixMilli()
	}
	s.mu.Unlock()

	if !found {
		return fmt.Errorf("thread not found: %s", id)
	}

	return s.persistThread(current)
}

func (s *AgentService) UpdateThreadWorkDir(id string, workDir string) error {
	s.mu.RLock()
	thread, exists := s.agents[id]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("thread not found: %s", id)
	}

	if err := thread.Agent.UpdateWorkDir(workDir); err != nil {
		return fmt.Errorf("failed to update thread work dir: %w", err)
	}

	s.mu.Lock()
	current, found := s.agents[id]
	if found {
		current.Info.WorkDir = workDir
		current.Info.UpdatedAt = time.Now().UnixMilli()
	}
	s.mu.Unlock()

	if !found {
		return fmt.Errorf("thread not found: %s", id)
	}

	return s.persistThread(current)
}

func (s *AgentService) UpdateThreadTitle(id string, title string) error {
	s.mu.RLock()
	_, exists := s.agents[id]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("thread not found: %s", id)
	}

	s.mu.Lock()
	current, found := s.agents[id]
	if found {
		current.Info.Title = title
		current.Info.UpdatedAt = time.Now().UnixMilli()
	}
	s.mu.Unlock()

	if !found {
		return fmt.Errorf("thread not found: %s", id)
	}

	return s.persistThread(current)
}

func (s *AgentService) loadThreadsFromStorage(ctx context.Context) error {
	storedThreads, err := storage.LoadThreads()
	if err != nil {
		return err
	}

	for _, stored := range storedThreads {
		if stored == nil || stored.Info == nil {
			continue
		}

		info := stored.Info
		config := models.AgentConfig{
			ModelID: info.Model,
			WorkDir: info.WorkDir,
		}

		agent, err := NewAgentWithMessages(ctx, info.ID, config, stored.History, stored.Turns, stored.Stats)
		if err != nil {
			return err
		}

		s.agents[info.ID] = &Thread{
			Info:  info,
			Agent: agent,
		}
	}

	return nil
}

func (s *AgentService) persistThread(thread *Thread) error {
	if thread == nil || thread.Info == nil {
		return fmt.Errorf("thread is nil")
	}

	turns := thread.Agent.GetTurns()
	history := thread.Agent.GetHistory()
	stats := thread.Agent.Stats()
	if stats == nil {
		return fmt.Errorf("thread stats is nil")
	}

	return storage.SaveThread(thread.Info, history, turns, stats)
}

func GenerateThreadTitle(ctx context.Context, messages []*schema.Message) (string, error) {
	if len(messages) == 0 {
		return defaultThreadTitle, nil
	}

	var conversationSummary strings.Builder
	for _, msg := range messages {
		switch msg.Role {
		case schema.User:
			conversationSummary.WriteString("User: ")
			conversationSummary.WriteString(msg.Content)
			conversationSummary.WriteString("\n")
		case schema.Assistant:
			conversationSummary.WriteString("Assistant: ")
			conversationSummary.WriteString(msg.Content)
			conversationSummary.WriteString("\n")
		case schema.Tool:
			conversationSummary.WriteString("Tool: ")
			conversationSummary.WriteString(msg.Content)
			conversationSummary.WriteString("\n")
		default:
			continue
		}
	}

	if conversationSummary.Len() == 0 {
		return defaultThreadTitle, nil
	}

	systemPrompt := "You are a helpful assistant that generates concise titles for conversations."
	userPrompt := fmt.Sprintf("Based on the following conversation, generate a concise and descriptive title (maximum 10 characters). The title should capture the main topic or question. Only return the title text, nothing else.\nConversation:\n%s", conversationSummary.String())
	titleMessages := []*schema.Message{
		{
			Role:    schema.System,
			Content: systemPrompt,
		},
		{
			Role:    schema.User,
			Content: userPrompt,
		},
	}

	model, err := getModel(ctx, getDefaultModelInfo().ID)
	if err != nil {
		return "", fmt.Errorf("failed to generate thread title: %w", err)
	}

	response, err := model.Generate(ctx, titleMessages)
	if err != nil {
		return "", fmt.Errorf("failed to generate thread title: %w", err)
	}

	title := cleanThreadTitle(response.Content)
	if title == "" {
		return defaultThreadTitle, nil
	}

	return title, nil
}

func cleanThreadTitle(title string) string {
	title = strings.TrimSpace(title)

	if len(title) >= 2 && title[0] == '"' && title[len(title)-1] == '"' {
		title = title[1 : len(title)-1]
		title = strings.TrimSpace(title)
	}

	runeCount := utf8.RuneCountInString(title)
	if runeCount > 10 {
		runes := []rune(title)
		title = string(runes[:9]) + "..."
	}

	if title == "" {
		return defaultThreadTitle
	}

	return title
}
