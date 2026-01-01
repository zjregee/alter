package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/cloudwego/eino/schema"

	"github.com/zjregee/alter/internal/models"
	"github.com/zjregee/alter/internal/notify"
	"github.com/zjregee/alter/internal/service"
)

func (a *App) AgentChat(threadID string, userInput string) error {
	if a.agentService == nil {
		return fmt.Errorf("agent service not initialized")
	}
	if threadID == "" {
		return fmt.Errorf("thread ID is required")
	}
	if userInput == "" {
		return fmt.Errorf("user input is required")
	}

	isFirstMessage, err := a.agentService.IsFirstMessageToThread(threadID)
	if err != nil {
		return err
	}

	go func() {
		msgChan, err := a.agentService.StreamRequestToThread(a.ctx, threadID, userInput)
		if err != nil {
			notify.EmitAgentMessage(a.ctx, "error", fmt.Sprintf("Failed to start agent: %v", err))
			return
		}

		var conversationSuccess bool
		for msg := range msgChan {
			var content string
			msgType := string(msg.GetType())

			switch m := msg.(type) {
			case models.AgentStartThinking:
				content = ""
			case models.AgentThought:
				content = formatThreadMessage(m.Content)
			case models.AgentExecutingToolStart:
				payload, _ := json.Marshal(m)
				content = string(payload)
			case models.AgentExecutingToolFinish:
				payload, _ := json.Marshal(m)
				content = string(payload)
			case models.AgentFinalResponse:
				content = ""
				conversationSuccess = true
			case models.AgentError:
				content = formatThreadMessage(m.Error)
			default:
				continue
			}

			notify.EmitAgentMessage(a.ctx, msgType, content)
		}

		if isFirstMessage && conversationSuccess {
			if err := a.generateAndUpdateThreadTitle(a.ctx, threadID); err != nil {
				fmt.Printf("Failed to generate thread title: %v\n", err)
			}
		}
	}()

	return nil
}

func (a *App) EditAndResendMessage(threadID string, userInput string, messageIndex int) error {
	if a.agentService == nil {
		return fmt.Errorf("agent service not initialized")
	}
	if threadID == "" {
		return fmt.Errorf("thread ID is required")
	}
	if userInput == "" {
		return fmt.Errorf("message content is required")
	}
	if messageIndex < 0 {
		return fmt.Errorf("invalid message index")
	}

	messages, err := a.agentService.GetThreadMessages(threadID)
	if err != nil {
		return err
	}

	isEditingFirstMessage := messageIndex == 0 && len(messages) > 0

	go func() {
		msgChan, err := a.agentService.EditAndResendRequestToThread(a.ctx, threadID, messageIndex, userInput)
		if err != nil {
			notify.EmitAgentMessage(a.ctx, "error", fmt.Sprintf("Failed to edit and resend message: %v", err))
			return
		}

		notify.EmitAgentMessagesTruncated(a.ctx, threadID, messageIndex)

		var conversationSuccess bool
		for msg := range msgChan {
			var content string
			msgType := string(msg.GetType())

			switch m := msg.(type) {
			case models.AgentStartThinking:
				content = ""
			case models.AgentThought:
				content = formatThreadMessage(m.Content)
			case models.AgentExecutingToolStart:
				payload, _ := json.Marshal(m)
				content = string(payload)
			case models.AgentExecutingToolFinish:
				payload, _ := json.Marshal(m)
				content = string(payload)
			case models.AgentFinalResponse:
				content = ""
				conversationSuccess = true
			case models.AgentError:
				content = formatThreadMessage(m.Error)
			default:
				continue
			}

			notify.EmitAgentMessage(a.ctx, msgType, content)
		}

		if isEditingFirstMessage && conversationSuccess {
			if err := a.generateAndUpdateThreadTitle(a.ctx, threadID); err != nil {
				fmt.Printf("Failed to regenerate thread title: %v\n", err)
			}
		}
	}()

	return nil
}

func (a *App) RegenerateLastResponse(threadID string) error {
	if a.agentService == nil {
		return fmt.Errorf("agent service not initialized")
	}
	if threadID == "" {
		return fmt.Errorf("thread ID is required")
	}

	messages, err := a.agentService.GetThreadMessages(threadID)
	if err != nil {
		return err
	}

	lastUserIndex := -1
	for i := len(messages) - 1; i >= 0; i -= 1 {
		if messages[i].Role == schema.User {
			lastUserIndex = i
			break
		}
	}

	if lastUserIndex == -1 {
		return fmt.Errorf("no user message found")
	}

	go func() {
		msgChan, err := a.agentService.RegenerateLastResponseToThread(a.ctx, threadID)
		if err != nil {
			notify.EmitAgentMessage(a.ctx, "error", fmt.Sprintf("Failed to regenerate response: %v", err))
			return
		}

		notify.EmitAgentMessagesTruncated(a.ctx, threadID, lastUserIndex)

		for msg := range msgChan {
			var content string
			msgType := string(msg.GetType())

			switch m := msg.(type) {
			case models.AgentStartThinking:
				content = ""
			case models.AgentThought:
				content = formatThreadMessage(m.Content)
			case models.AgentExecutingToolStart:
				payload, _ := json.Marshal(m)
				content = string(payload)
			case models.AgentExecutingToolFinish:
				payload, _ := json.Marshal(m)
				content = string(payload)
			case models.AgentFinalResponse:
				content = ""
			case models.AgentError:
				content = formatThreadMessage(m.Error)
			default:
				continue
			}

			notify.EmitAgentMessage(a.ctx, msgType, content)
		}
	}()

	return nil
}

func (a *App) generateAndUpdateThreadTitle(ctx context.Context, threadID string) error {
	messages, err := a.agentService.GetThreadMessages(threadID)
	if err != nil {
		return fmt.Errorf("failed to get thread messages: %w", err)
	}

	if len(messages) == 0 {
		return fmt.Errorf("no messages in thread")
	}

	schemaMessages := make([]*schema.Message, 0)
	for _, turn := range messages {
		if turn.Role == "user" && len(turn.Events) > 0 {
			var userMsg models.UserMessage
			if err := json.Unmarshal(turn.Events[0].Content, &userMsg); err == nil {
				schemaMessages = append(schemaMessages, &schema.Message{Role: schema.User, Content: userMsg.Content})
			}
			break
		}
	}

	title, err := service.GenerateThreadTitle(ctx, schemaMessages)
	if err != nil {
		return fmt.Errorf("failed to generate title: %w", err)
	}

	formattedTitle := formatThreadTitle(title)
	if err := a.agentService.UpdateThreadTitle(threadID, formattedTitle); err != nil {
		return fmt.Errorf("failed to update thread title: %w", err)
	}

	notify.EmitThreadTitleUpdated(a.ctx, threadID, formattedTitle)

	return nil
}

func (a *App) ListModels() []*models.ModelInfo {
	if a.agentService == nil {
		return []*models.ModelInfo{}
	}

	models := a.agentService.ListModels()
	sort.SliceStable(models, func(i, j int) bool {
		if models[i].Provider != models[j].Provider {
			return models[i].Provider < models[j].Provider
		}
		return models[i].Name < models[j].Name
	})

	return models
}
