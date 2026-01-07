package models

type AgentMessageType string

const (
	AgentMessageTypeUserMessage         AgentMessageType = "user_message"
	AgentMessageTypeStartThinking       AgentMessageType = "start_thinking"
	AgentMessageTypeThinking            AgentMessageType = "thinking"
	AgentMessageTypeStreamChunk         AgentMessageType = "stream_chunk"
	AgentMessageTypeThought             AgentMessageType = "thought"
	AgentMessageTypeExecutingToolStart  AgentMessageType = "executing_tool_start"
	AgentMessageTypeExecutingToolFinish AgentMessageType = "executing_tool_finish"
	AgentMessageTypeFinalResponse       AgentMessageType = "final_response"
	AgentMessageTypeError               AgentMessageType = "error"
)

type AgentMessage interface {
	GetType() AgentMessageType
}

type UserMessage struct {
	Content string `json:"content"`
}

func (m UserMessage) GetType() AgentMessageType {
	return AgentMessageTypeUserMessage
}

type AgentStartThinking struct{}

func (m AgentStartThinking) GetType() AgentMessageType {
	return AgentMessageTypeStartThinking
}

type AgentThinking struct {
	Content string `json:"content"`
}

func (m AgentThinking) GetType() AgentMessageType {
	return AgentMessageTypeThinking
}

type AgentStreamChunk struct {
	Content string `json:"content"`
}

func (m AgentStreamChunk) GetType() AgentMessageType {
	return AgentMessageTypeStreamChunk
}

type AgentThought struct {
	Reasoning       string  `json:"reasoning,omitempty"`
	Content         string  `json:"content"`
	DurationSeconds float64 `json:"duration_seconds,omitempty"`
}

func (m AgentThought) GetType() AgentMessageType {
	return AgentMessageTypeThought
}

type AgentExecutingToolStart struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Args string `json:"args"`
}

func (m AgentExecutingToolStart) GetType() AgentMessageType {
	return AgentMessageTypeExecutingToolStart
}

type AgentExecutingToolFinish struct {
	ID              int     `json:"id"`
	Name            string  `json:"name"`
	Args            string  `json:"args"`
	Content         string  `json:"content"`
	DurationSeconds float64 `json:"duration_seconds,omitempty"`
}

func (m AgentExecutingToolFinish) GetType() AgentMessageType {
	return AgentMessageTypeExecutingToolFinish
}

type AgentFinalResponse struct{}

func (m AgentFinalResponse) GetType() AgentMessageType {
	return AgentMessageTypeFinalResponse
}

type AgentError struct {
	Error string `json:"error"`
}

func (m AgentError) GetType() AgentMessageType {
	return AgentMessageTypeError
}
