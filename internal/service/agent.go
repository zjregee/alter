package service

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"github.com/zjregee/alter/internal/models"
	"github.com/zjregee/alter/internal/service/tools"
)

//go:embed assets/prompts/agent.txt
var promptContent []byte

const (
	defaultMaxIterations   = 40
	defaultRequestInterval = 3 * time.Second
)

type Agent struct {
	id       string
	config   models.AgentConfig
	tools    []*schema.ToolInfo
	toolsMap map[string]tool.InvokableTool

	messages          []*schema.Message
	messageTimestamps []int64
	stats             *models.AgentStats

	cancelFunc context.CancelFunc
}

func applyDefaults(c *models.AgentConfig) error {
	c.ModelID = strings.TrimSpace(c.ModelID)
	c.WorkDir = strings.TrimSpace(c.WorkDir)

	if c.ModelID == "" {
		return fmt.Errorf("agent model is required")
	}
	if isModelAvailable(c.ModelID) {
		return fmt.Errorf("agent model is not available: %s", c.ModelID)
	}
	if c.MaxIterations <= 0 {
		c.MaxIterations = defaultMaxIterations
	}
	if c.RequestInterval <= 0 {
		c.RequestInterval = defaultRequestInterval
	}
	if c.WorkDir == "" {
		return fmt.Errorf("agent work dir is required")
	}
	if isWorkspacePathAvailable(c.WorkDir) {
		return fmt.Errorf("agent work dir is not available: %s", c.WorkDir)
	}

	return nil
}

func buildSystemPrompt(workDir string) string {
	prompt := string(promptContent)
	prompt = strings.ReplaceAll(prompt, "[ROOT_DIRECTORY]", workDir)
	prompt = strings.ReplaceAll(prompt, "[SYSTEM_TIME]", time.Now().Format(time.RFC3339))
	return prompt
}

func NewAgent(ctx context.Context, cfg models.AgentConfig) (*Agent, error) {
	if err := applyDefaults(&cfg); err != nil {
		return nil, err
	}

	toolInfos, toolsMap, err := tools.GetAllRegisteredTools(ctx)
	if err != nil {
		return nil, err
	}

	return &Agent{
		id:       GenerateAgentID(),
		config:   cfg,
		tools:    toolInfos,
		toolsMap: toolsMap,
		messages: []*schema.Message{
			{
				Role:    schema.System,
				Content: buildSystemPrompt(cfg.WorkDir),
			},
		},
		messageTimestamps: []int64{time.Now().UnixMilli()},
		stats: &models.AgentStats{
			Usage:               &models.AgentUsage{},
			NextExecutingToolID: 0,
			LastRequestTime:     time.Now(),
		},
	}, nil
}

func NewAgentWithMessages(ctx context.Context, id string, cfg models.AgentConfig, messages []*schema.Message, messageTimestamps []int64, stats *models.AgentStats) (*Agent, error) {
	if err := applyDefaults(&cfg); err != nil {
		return nil, err
	}

	toolInfos, toolsMap, err := tools.GetAllRegisteredTools(ctx)
	if err != nil {
		return nil, err
	}

	return &Agent{
		id:                id,
		config:            cfg,
		tools:             toolInfos,
		toolsMap:          toolsMap,
		messages:          messages,
		messageTimestamps: messageTimestamps,
		stats:             stats,
	}, nil
}

func (a *Agent) ID() string {
	return a.id
}

func (a *Agent) Config() models.AgentConfig {
	return a.config
}

func (a *Agent) Stats() *models.AgentStats {
	return a.stats
}

func (a *Agent) UpdateModelID(modelID string) error {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return fmt.Errorf("agent model is required")
	}

	if isModelAvailable(modelID) {
		return fmt.Errorf("agent model is not available: %s", modelID)
	}

	a.config.ModelID = modelID
	return nil
}

func (a *Agent) UpdateWorkDir(workDir string) error {
	workDir = strings.TrimSpace(workDir)
	if workDir == "" {
		return fmt.Errorf("agent work dir is required")
	}

	if workDir == a.config.WorkDir {
		return nil
	}

	if len(a.messages) > 1 {
		return fmt.Errorf("agent messages are not empty")
	}

	a.config.WorkDir = workDir
	a.messages[0].Content = buildSystemPrompt(workDir)
	a.messageTimestamps[0] = time.Now().UnixMilli()
	return nil
}

func (a *Agent) StreamRequest(ctx context.Context, userInput string) <-chan models.AgentMessage {
	msgChan := make(chan models.AgentMessage)

	streamCtx, cancel := context.WithCancel(ctx)
	a.cancelFunc = cancel

	go a.reActLoop(streamCtx, userInput, msgChan)

	return msgChan
}

func (a *Agent) CancelStreamRequest() {
	if a.cancelFunc != nil {
		a.cancelFunc()
	}
}

func (a *Agent) GetMessagesWithTimestamps() ([]*schema.Message, []int64) {
	return a.messages, a.messageTimestamps
}

func (a *Agent) TruncateMessagesSince(index int) error {
	nonSystemIndex := -1
	actualIndex := -1

	for i, msg := range a.messages {
		if msg.Role != schema.System {
			nonSystemIndex += 1
			if nonSystemIndex == index {
				actualIndex = i
				break
			}
		}
	}

	if actualIndex == -1 {
		return fmt.Errorf("invalid message index: %d", index)
	}

	a.messages = a.messages[:actualIndex+1]
	a.messageTimestamps = a.messageTimestamps[:actualIndex+1]

	return nil
}

func (a *Agent) reActLoop(ctx context.Context, userInput string, msgChan chan models.AgentMessage) {
	defer close(msgChan)
	defer func() {
		if a.cancelFunc != nil {
			a.cancelFunc = nil
		}
	}()

	if strings.TrimSpace(userInput) == "" {
		msgChan <- models.AgentError{Error: "user input is empty"}
		return
	}

	userMessage := &schema.Message{
		Role:    schema.User,
		Content: userInput,
	}
	a.messages = append(a.messages, userMessage)
	a.messageTimestamps = append(a.messageTimestamps, time.Now().UnixMilli())

	iterations := 0
	for iterations < a.config.MaxIterations {
		select {
		case <-ctx.Done():
			msgChan <- models.AgentError{Error: "agent generation cancelled"}
			return
		default:
		}

		a.waitForNextTurn()
		msgChan <- models.AgentStartThinking{}

		response, err := a.generate(ctx)
		if err != nil {
			if strings.Contains(err.Error(), "429") {
				time.Sleep(3 * time.Second)
				iterations += 1
				continue
			}
			msgChan <- models.AgentError{Error: fmt.Sprintf("agent generation failed: %v", err)}
			return
		}

		if response.ResponseMeta != nil && response.ResponseMeta.Usage != nil {
			usage := response.ResponseMeta.Usage
			a.stats.Usage.PromptTokens += usage.PromptTokens
			a.stats.Usage.CompletionTokens += usage.CompletionTokens
			a.stats.Usage.TotalTokens = a.stats.Usage.PromptTokens + a.stats.Usage.CompletionTokens
		}

		if response.Content != "" {
			msgChan <- models.AgentThought{Content: response.Content}
		}

		a.messages = append(a.messages, response)
		a.messageTimestamps = append(a.messageTimestamps, time.Now().UnixMilli())

		if len(response.ToolCalls) == 0 {
			content := response.Content
			if content == "" {
				content = "Sorry, I couldn't generate a meaningful response."
			}
			msgChan <- models.AgentFinalResponse{Content: content}
			return
		}

		type toolResult struct {
			call   schema.ToolCall
			result string
			err    error
		}

		toolResultChan := make(chan toolResult, len(response.ToolCalls))
		for _, toolCall := range response.ToolCalls {
			go func(tc schema.ToolCall) {
				toolID := int(atomic.AddInt64(&a.stats.NextExecutingToolID, 1))
				msgChan <- models.AgentExecutingToolStart{
					ID:   toolID,
					Name: tc.Function.Name,
					Args: tc.Function.Arguments,
				}
				result, err := a.invokeTool(ctx, tc)
				msgChan <- models.AgentExecutingToolFinish{
					ID:      toolID,
					Name:    tc.Function.Name,
					Args:    tc.Function.Arguments,
					Content: result,
				}
				toolResultChan <- toolResult{
					call:   tc,
					result: result,
					err:    err,
				}
			}(toolCall)
		}

		toolResultsByID := make(map[string]toolResult, len(response.ToolCalls))
		for range response.ToolCalls {
			r := <-toolResultChan
			toolResultsByID[r.call.ID] = r
		}

		for _, tc := range response.ToolCalls {
			if res, ok := toolResultsByID[tc.ID]; ok {
				content := res.result
				if res.err != nil {
					content = fmt.Sprintf("Tool %s call failed: %v", tc.Function.Name, res.err)
				}
				a.messages = append(a.messages, &schema.Message{
					Role:       schema.Tool,
					ToolCallID: tc.ID,
					Content:    content,
				})
				a.messageTimestamps = append(a.messageTimestamps, time.Now().UnixMilli())
			}
		}

		iterations += 1
	}

	msgChan <- models.AgentError{
		Error: fmt.Sprintf("Sorry, I've reached the maximum iterations %d and still couldn't produce a final answer", a.config.MaxIterations),
	}
}

func (a *Agent) waitForNextTurn() {
	if a.config.RequestInterval <= 0 {
		return
	}

	if a.stats.LastRequestTime.IsZero() {
		a.stats.LastRequestTime = time.Now()
		return
	}

	elapsed := time.Since(a.stats.LastRequestTime)
	if sleep := a.config.RequestInterval - elapsed; sleep > 0 {
		time.Sleep(sleep)
	}
	a.stats.LastRequestTime = time.Now()
}

func (a *Agent) generate(ctx context.Context) (*schema.Message, error) {
	model, err := getModel(ctx, a.config.ModelID)
	if err != nil {
		return nil, err
	}

	modelWithTools, err := model.WithTools(a.tools)
	if err != nil {
		return nil, err
	}

	response, err := modelWithTools.Generate(ctx, a.messages)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (a *Agent) invokeTool(ctx context.Context, toolCall schema.ToolCall) (string, error) {
	targetTool, exists := a.toolsMap[toolCall.Function.Name]
	if !exists {
		return "", fmt.Errorf("agent tool not found: %s", toolCall.Function.Name)
	}

	result, err := targetTool.InvokableRun(ctx, toolCall.Function.Arguments)
	if err != nil {
		return "", err
	}
	return result, nil
}
