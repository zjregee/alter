package models

import (
	"time"
)

type WorkflowState string

const (
	WorkflowStatePending  WorkflowState = "pending"
	WorkflowStateRunning  WorkflowState = "running"
	WorkflowStateFinished WorkflowState = "finished"
	WorkflowStateFailed   WorkflowState = "failed"
)

type WorkflowConfig struct {
	Name            string        `json:"name"`
	Description     string        `json:"description"`
	Prompt          string        `json:"prompt"`
	ModelID         string        `json:"model_id"`
	MaxIterations   int           `json:"max_iterations"`
	RequestInterval time.Duration `json:"request_interval"`
	WorkDir         string        `json:"work_dir"`
}
