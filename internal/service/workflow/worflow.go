package workflow

import (
	"github.com/zjregee/alter/internal/models"
)

type WorkflowAction struct {
	Prompt      string
	AgentConfig *models.AgentConfig
}

type Workflow struct {
	Name        string
	Description string
	State       models.WorkflowState
	Action      *WorkflowAction
}

func NewWorkflow(config models.WorkflowConfig) *Workflow {
	return &Workflow{
		Name:        config.Name,
		Description: config.Description,
		State:       models.WorkflowStatePending,
		Action: &WorkflowAction{
			Prompt: config.Prompt,
			AgentConfig: &models.AgentConfig{
				ModelID: config.ModelID,
				WorkDir: config.WorkDir,
			},
		},
	}
}
