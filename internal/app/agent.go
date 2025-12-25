package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/schema"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/zjregee/alter/internal/models"
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
			runtime.EventsEmit(a.ctx, "agent:message", map[string]string{
				"type":    "error",
				"content": fmt.Sprintf("Failed to start agent: %v", err),
			})
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
				content = formatThreadMessage(m.Content)
				conversationSuccess = true
			case models.AgentError:
				content = formatThreadMessage(m.Error)
			default:
				continue
			}

			runtime.EventsEmit(a.ctx, "agent:message", map[string]string{
				"type":    msgType,
				"content": content,
			})
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
			runtime.EventsEmit(a.ctx, "agent:message", map[string]string{
				"type":    "error",
				"content": fmt.Sprintf("Failed to edit and resend message: %v", err),
			})
			return
		}

		runtime.EventsEmit(a.ctx, "agent:messages_truncated", map[string]any{
			"thread_id":  threadID,
			"from_index": messageIndex,
		})

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
				content = formatThreadMessage(m.Content)
				conversationSuccess = true
			case models.AgentError:
				content = formatThreadMessage(m.Error)
			default:
				continue
			}

			runtime.EventsEmit(a.ctx, "agent:message", map[string]string{
				"type":    msgType,
				"content": content,
			})
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
			runtime.EventsEmit(a.ctx, "agent:message", map[string]string{
				"type":    "error",
				"content": fmt.Sprintf("Failed to regenerate response: %v", err),
			})
			return
		}

		runtime.EventsEmit(a.ctx, "agent:messages_truncated", map[string]any{
			"thread_id":  threadID,
			"from_index": lastUserIndex,
		})

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
				content = formatThreadMessage(m.Content)
			case models.AgentError:
				content = formatThreadMessage(m.Error)
			default:
				continue
			}

			runtime.EventsEmit(a.ctx, "agent:message", map[string]string{
				"type":    msgType,
				"content": content,
			})
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

	title, err := service.GenerateThreadTitle(ctx, messages)
	if err != nil {
		return fmt.Errorf("failed to generate title: %w", err)
	}

	formattedTitle := formatThreadTitle(title)
	if err := a.agentService.UpdateThreadTitle(threadID, formattedTitle); err != nil {
		return fmt.Errorf("failed to update thread title: %w", err)
	}

	runtime.EventsEmit(a.ctx, "thread:title_updated", map[string]string{
		"thread_id": threadID,
		"title":     formattedTitle,
	})

	return nil
}

func (a *App) ListModels() []*models.ModelInfo {
	if a.agentService == nil {
		return []*models.ModelInfo{}
	}

	return a.agentService.ListModels()
}
