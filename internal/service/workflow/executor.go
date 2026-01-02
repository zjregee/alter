package workflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/zjregee/alter/internal/models"
	"github.com/zjregee/alter/internal/service"
)

type Executor struct{}

func NewExecutor() *Executor {
	return &Executor{}
}

type ExecutionResult struct {
	Record []models.AgentMessage
	Result string
}

type executionResultKey struct{}

func WithExecutionResult(ctx context.Context, result *ExecutionResult) context.Context {
	if result == nil {
		return ctx
	}
	return context.WithValue(ctx, executionResultKey{}, result)
}

func GetExecutionResult(ctx context.Context) (*ExecutionResult, bool) {
	if ctx == nil {
		return nil, false
	}
	result, ok := ctx.Value(executionResultKey{}).(*ExecutionResult)
	return result, ok
}

func (e *Executor) Execute(ctx context.Context, workflow *Workflow) error {
	if workflow == nil {
		return fmt.Errorf("workflow is required")
	}
	if workflow.Action == nil {
		return fmt.Errorf("workflow action is required")
	}
	if workflow.Action.AgentConfig == nil {
		return fmt.Errorf("workflow agent config is required")
	}
	prompt := strings.TrimSpace(workflow.Action.Prompt)
	if prompt == "" {
		return fmt.Errorf("workflow prompt is required")
	}

	workflow.State = models.WorkflowStateRunning

	result, _ := GetExecutionResult(ctx)
	if result != nil && result.Record == nil {
		result.Record = make([]models.AgentMessage, 0, 16)
	}

	agent, err := service.NewAgent(ctx, *workflow.Action.AgentConfig)
	if err != nil {
		workflow.State = models.WorkflowStateFailed
		return fmt.Errorf("failed to create agent: %w", err)
	}

	msgChan := agent.StreamRequest(ctx, prompt)

	var lastThought string
	var execErr error

	for msg := range msgChan {
		if result != nil {
			result.Record = append(result.Record, msg)
		}

		switch event := msg.(type) {
		case models.AgentThought:
			if strings.TrimSpace(event.Content) != "" {
				lastThought = event.Content
			}
		case models.AgentError:
			execErr = fmt.Errorf("agent error: %s", event.Error)
		}
	}

	if result != nil {
		result.Result = lastThought
	}

	if execErr != nil {
		workflow.State = models.WorkflowStateFailed
		return execErr
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		workflow.State = models.WorkflowStateFailed
		return ctxErr
	}

	workflow.State = models.WorkflowStateFinished
	return nil
}
