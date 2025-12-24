package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zjregee/alter/internal/models"
	"github.com/zjregee/alter/internal/service"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// AgentChat sends a user message to the agent and streams the response.
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

	a.ctxMu.RLock()
	appCtx := a.ctx
	a.ctxMu.RUnlock()

	if appCtx == nil {
		appCtx = context.Background()
	}

	// Check if this is the first message in the thread (before the new message is added)
	isFirstMessage, err := a.agentService.IsFirstMessageToThread(threadID)
	if err != nil {
		return err
	}

	go func() {
		msgChan, err := a.agentService.StreamRequestToThread(appCtx, threadID, userInput)
		if err != nil {
			runtime.EventsEmit(appCtx, "agent:message", map[string]string{
				"type":    "error",
				"content": fmt.Sprintf("Failed to start agent: %v", err),
			})
			return
		}

		var conversationSuccess bool
		for msg := range msgChan {
			var msgType string
			var content string

			switch m := msg.(type) {
			case models.AgentStartThinking:
				msgType = "start_thinking"
				content = ""
			case models.AgentThought:
				msgType = "thought"
				content = formatAgentMessageContent(m.Content)
			case models.AgentExecutingToolStart:
				msgType = string(m.GetType())
				payload, _ := json.Marshal(m)
				content = string(payload)
			case models.AgentExecutingToolFinish:
				msgType = string(m.GetType())
				payload, _ := json.Marshal(m)
				content = string(payload)
			case models.AgentFinalResponse:
				msgType = "final_response"
				content = formatAgentMessageContent(m.Content)
				conversationSuccess = true
			case models.AgentError:
				msgType = "error"
				content = formatAgentMessageContent(m.Error)
			default:
				continue
			}

			runtime.EventsEmit(appCtx, "agent:message", map[string]string{
				"type":    msgType,
				"content": content,
			})
		}

		// Generate title for the first message only if conversation succeeded
		if isFirstMessage && conversationSuccess {
			if err := a.generateAndUpdateThreadTitle(appCtx, threadID); err != nil {
				// Log error but don't interrupt the conversation
				fmt.Printf("Failed to generate thread title: %v\n", err)
			}
		}
	}()

	return nil
}

// EditAndResendMessage edits a message and resends from that point.
func (a *App) EditAndResendMessage(threadID string, messageIndex int, newContent string) error {
	if a.agentService == nil {
		return fmt.Errorf("agent service not initialized")
	}
	if threadID == "" {
		return fmt.Errorf("thread ID is required")
	}
	if newContent == "" {
		return fmt.Errorf("message content is required")
	}
	if messageIndex < 0 {
		return fmt.Errorf("invalid message index")
	}

	a.ctxMu.RLock()
	appCtx := a.ctx
	a.ctxMu.RUnlock()

	if appCtx == nil {
		appCtx = context.Background()
	}

	// Check if this is editing the first message (might need to regenerate title)
	messages, err := a.agentService.GetThreadMessages(threadID)
	if err != nil {
		return err
	}
	isEditingFirstMessage := messageIndex == 0 && len(messages) > 0

	go func() {
		msgChan, err := a.agentService.EditAndResendRequestToThread(appCtx, threadID, messageIndex, newContent)
		if err != nil {
			runtime.EventsEmit(appCtx, "agent:message", map[string]string{
				"type":    "error",
				"content": fmt.Sprintf("Failed to edit and resend message: %v", err),
			})
			return
		}

		// Emit event to notify frontend that messages are being truncated
		// Only emit after successful truncation
		// from_index: messages from this index (inclusive) onwards have been deleted
		runtime.EventsEmit(appCtx, "agent:messages_truncated", map[string]interface{}{
			"thread_id":  threadID,
			"from_index": messageIndex,
		})

		var conversationSuccess bool
		for msg := range msgChan {
			var msgType string
			var content string

			switch m := msg.(type) {
			case models.AgentStartThinking:
				msgType = "start_thinking"
				content = ""
			case models.AgentThought:
				msgType = "thought"
				content = formatAgentMessageContent(m.Content)
			case models.AgentExecutingToolStart:
				msgType = string(m.GetType())
				payload, _ := json.Marshal(m)
				content = string(payload)
			case models.AgentExecutingToolFinish:
				msgType = string(m.GetType())
				payload, _ := json.Marshal(m)
				content = string(payload)
			case models.AgentFinalResponse:
				msgType = "final_response"
				content = formatAgentMessageContent(m.Content)
				conversationSuccess = true
			case models.AgentError:
				msgType = "error"
				content = formatAgentMessageContent(m.Error)
			default:
				continue
			}

			runtime.EventsEmit(appCtx, "agent:message", map[string]string{
				"type":    msgType,
				"content": content,
			})
		}

		// Regenerate title if editing the first message
		if isEditingFirstMessage && conversationSuccess {
			if err := a.generateAndUpdateThreadTitle(appCtx, threadID); err != nil {
				fmt.Printf("Failed to regenerate thread title: %v\n", err)
			}
		}
	}()

	return nil
}

// RegenerateLastResponse regenerates the last agent response.
func (a *App) RegenerateLastResponse(threadID string) error {
	if a.agentService == nil {
		return fmt.Errorf("agent service not initialized")
	}
	if threadID == "" {
		return fmt.Errorf("thread ID is required")
	}

	a.ctxMu.RLock()
	appCtx := a.ctx
	a.ctxMu.RUnlock()

	if appCtx == nil {
		appCtx = context.Background()
	}

	// Get the last user message index before truncation
	messages, err := a.agentService.GetThreadMessages(threadID)
	if err != nil {
		return err
	}

	lastUserIndex := -1
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			lastUserIndex = i
			break
		}
	}

	if lastUserIndex == -1 {
		return fmt.Errorf("no user message found")
	}

	go func() {
		msgChan, err := a.agentService.RegenerateLastResponseToThread(appCtx, threadID)
		if err != nil {
			runtime.EventsEmit(appCtx, "agent:message", map[string]string{
				"type":    "error",
				"content": fmt.Sprintf("Failed to regenerate response: %v", err),
			})
			return
		}

		// Emit event to notify frontend that messages are being truncated
		// Only emit after successful truncation
		// from_index: messages from this index (inclusive) onwards have been deleted
		runtime.EventsEmit(appCtx, "agent:messages_truncated", map[string]interface{}{
			"thread_id":  threadID,
			"from_index": lastUserIndex,
		})

		for msg := range msgChan {
			var msgType string
			var content string

			switch m := msg.(type) {
			case models.AgentStartThinking:
				msgType = "start_thinking"
				content = ""
			case models.AgentThought:
				msgType = "thought"
				content = formatAgentMessageContent(m.Content)
			case models.AgentExecutingToolStart:
				msgType = string(m.GetType())
				payload, _ := json.Marshal(m)
				content = string(payload)
			case models.AgentExecutingToolFinish:
				msgType = string(m.GetType())
				payload, _ := json.Marshal(m)
				content = string(payload)
			case models.AgentFinalResponse:
				msgType = "final_response"
				content = formatAgentMessageContent(m.Content)
			case models.AgentError:
				msgType = "error"
				content = formatAgentMessageContent(m.Error)
			default:
				continue
			}

			runtime.EventsEmit(appCtx, "agent:message", map[string]string{
				"type":    msgType,
				"content": content,
			})
		}
	}()

	return nil
}

// generateAndUpdateThreadTitle generates and updates a thread title based on conversation.
func (a *App) generateAndUpdateThreadTitle(ctx context.Context, threadID string) error {
	// Get the conversation messages
	messages, err := a.agentService.GetThreadMessages(threadID)
	if err != nil {
		return fmt.Errorf("failed to get thread messages: %w", err)
	}

	if len(messages) == 0 {
		return fmt.Errorf("no messages in thread")
	}

	// Generate title using the agent service
	title, err := service.GenerateThreadTitle(ctx, messages)
	if err != nil {
		return fmt.Errorf("failed to generate title: %w", err)
	}

	// Update the thread title
	formattedTitle, err := normalizeThreadTitle(title)
	if err != nil {
		return err
	}
	if err := a.agentService.UpdateThreadTitle(threadID, formattedTitle); err != nil {
		return fmt.Errorf("failed to update thread title: %w", err)
	}

	// Emit event to update UI
	a.ctxMu.RLock()
	appCtx := a.ctx
	a.ctxMu.RUnlock()

	if appCtx != nil {
		runtime.EventsEmit(appCtx, "thread:title_updated", map[string]string{
			"thread_id": threadID,
			"title":     formattedTitle,
		})
	}

	return nil
}

// ListModels returns a list of available models.
func (a *App) ListModels() []*models.ModelInfo {
	if a.agentService == nil {
		return []*models.ModelInfo{}
	}
	return a.agentService.ListModels()
}
