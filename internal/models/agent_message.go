package models

type AgentMessageType string

const (
	AgentMessageTypeStartThinking       AgentMessageType = "start_thinking"
	AgentMessageTypeThought             AgentMessageType = "thought"
	AgentMessageTypeExecutingToolStart  AgentMessageType = "executing_tool_start"
	AgentMessageTypeExecutingToolFinish AgentMessageType = "executing_tool_finish"
	AgentMessageTypeFinalResponse       AgentMessageType = "final_response"
	AgentMessageTypeError               AgentMessageType = "error"
)

type AgentMessage interface {
	GetType() AgentMessageType
}

type AgentStartThinking struct{}

func (m AgentStartThinking) GetType() AgentMessageType {
	return AgentMessageTypeStartThinking
}

type AgentThought struct {
	Content string `json:"content"`
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
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Args    string `json:"args"`
	Content string `json:"content"`
}

func (m AgentExecutingToolFinish) GetType() AgentMessageType {
	return AgentMessageTypeExecutingToolFinish
}

type AgentFinalResponse struct {
	Content string `json:"content"`
}

func (m AgentFinalResponse) GetType() AgentMessageType {
	return AgentMessageTypeFinalResponse
}

type AgentError struct {
	Error string `json:"error"`
}

func (m AgentError) GetType() AgentMessageType {
	return AgentMessageTypeError
}
