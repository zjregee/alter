package models

import (
	"time"
)

type AgentConfig struct {
	ModelID         string
	MaxIterations   int
	RequestInterval time.Duration
	WorkDir         string
}

type AgentUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type AgentStats struct {
	Usage               *AgentUsage
	NextExecutingToolID int64
	LastRequestTime     time.Time
}
