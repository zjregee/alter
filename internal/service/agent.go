package service

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/zjregee/alter/internal/models"
	"github.com/zjregee/alter/internal/service/skills"
	"github.com/zjregee/alter/internal/service/tools"
	"github.com/zjregee/alter/internal/utils"
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

	history []*schema.Message
	turns   []*models.ThreadMessageTurn

	stats *models.AgentStats

	cancelFunc context.CancelFunc
}

func applyDefaults(c *models.AgentConfig) error {
	c.ModelID = strings.TrimSpace(c.ModelID)
	c.WorkDir = strings.TrimSpace(c.WorkDir)

	if c.ModelID == "" {
		return fmt.Errorf("agent model is required")
	}
	if !isModelAvailable(c.ModelID) {
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
	if !isWorkspacePathAvailable(c.WorkDir) {
		return fmt.Errorf("agent work dir is not available: %s", c.WorkDir)
	}

	return nil
}

func buildSystemPrompt(workDir string) string {
	prompt := string(promptContent)
	prompt = strings.ReplaceAll(prompt, "[ROOT_DIRECTORY]", workDir)
	prompt = strings.ReplaceAll(prompt, "[SYSTEM_TIME]", time.Now().Format(time.RFC3339))

	summaries, err := skills.LoadAllSkillSummaries()
	if err != nil {
		utils.GetLogger().Printf("failed to load skill summaries: %v", err)
	}

	var skillsSection strings.Builder
	if len(summaries) > 0 {
		skillsSection.WriteString("可用技能：")
		for _, s := range summaries {
			fmt.Fprintf(&skillsSection, "\n- %s: %s", s.Name, s.Description)
		}
	}
	prompt = strings.ReplaceAll(prompt, "[SKILLS]", skillsSection.String())

	return prompt
}

func sanitizeHistory(workDir string, history []*schema.Message) []*schema.Message {
	if len(history) == 0 {
		return []*schema.Message{
			{
				Role:    schema.System,
				Content: buildSystemPrompt(workDir),
			},
		}
	}

	cleaned := make([]*schema.Message, 0, len(history))
	for _, msg := range history {
		if msg == nil {
			continue
		}
		cleaned = append(cleaned, msg)
	}

	if len(cleaned) == 0 {
		cleaned = append(cleaned, &schema.Message{
			Role:    schema.System,
			Content: buildSystemPrompt(workDir),
		})
	}

	return cleaned
}

func NewAgent(ctx context.Context, cfg models.AgentConfig) (*Agent, error) {
	if err := applyDefaults(&cfg); err != nil {
		return nil, err
	}

	ensureTelemetry(ctx)

	toolInfos, toolsMap, err := tools.GetAllRegisteredTools(ctx)
	if err != nil {
		return nil, err
	}

	return &Agent{
		id:       GenerateAgentID(),
		config:   cfg,
		tools:    toolInfos,
		toolsMap: toolsMap,
		history: []*schema.Message{
			{
				Role:    schema.System,
				Content: buildSystemPrompt(cfg.WorkDir),
			},
		},
		turns: make([]*models.ThreadMessageTurn, 0),
		stats: &models.AgentStats{
			Usage:               &models.AgentUsage{},
			NextExecutingToolID: 0,
			LastRequestTime:     time.Now(),
		},
	}, nil
}

func NewAgentWithMessages(ctx context.Context, id string, cfg models.AgentConfig, history []*schema.Message, turns []*models.ThreadMessageTurn, stats *models.AgentStats) (*Agent, error) {
	if err := applyDefaults(&cfg); err != nil {
		return nil, err
	}

	ensureTelemetry(ctx)

	toolInfos, toolsMap, err := tools.GetAllRegisteredTools(ctx)
	if err != nil {
		return nil, err
	}

	return &Agent{
		id:       id,
		config:   cfg,
		tools:    toolInfos,
		toolsMap: toolsMap,
		history:  sanitizeHistory(cfg.WorkDir, history),
		turns:    turns,
		stats:    stats,
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

	if !isModelAvailable(modelID) {
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

	if len(a.turns) > 0 {
		return fmt.Errorf("cannot change work dir on a non-empty thread")
	}

	a.config.WorkDir = workDir
	a.history[0].Content = buildSystemPrompt(workDir)
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

func (a *Agent) GetTurns() []*models.ThreadMessageTurn {
	return a.turns
}

func (a *Agent) GetHistory() []*schema.Message {
	return a.history
}

func (a *Agent) TruncateMessagesSince(turnIndex int) error {
	if turnIndex < 0 || turnIndex >= len(a.turns) {
		return fmt.Errorf("invalid turn index: %d", turnIndex)
	}

	a.turns = a.turns[:turnIndex]

	historyCount := 1
	for _, turn := range a.turns {
		historyCount += len(turn.Events)
	}

	if historyCount <= len(a.history)+1 {
		a.history = a.history[:historyCount+1]
	}

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

	var runSpan trace.Span
	var runStartTime time.Time
	if isTelemetryEnabled() {
		runStartTime = time.Now()
		tracer := otel.Tracer("alter/agent")
		var newCtx context.Context
		newCtx, runSpan = tracer.Start(ctx, "agent.run",
			trace.WithAttributes(
				attribute.String("agent.id", a.id),
				attribute.String("agent.model_id", a.config.ModelID),
				attribute.String("agent.work_dir", a.config.WorkDir),
				attribute.Int("agent.max_iterations", a.config.MaxIterations),
				attribute.Int("agent.available_tools", len(a.tools)),
				attribute.Int("agent.history_size", len(a.history)),
				attribute.Int("agent.turns_count", len(a.turns)),
			),
		)
		ctx = newCtx
		runSpan.AddEvent("agent.user_input", trace.WithAttributes(
			attribute.String("content", userInput),
			attribute.Int("input_length", len(userInput)),
		))
		defer runSpan.End()
	}

	userMsg := models.UserMessage{
		Content: userInput,
	}
	userEventContent, _ := json.Marshal(userMsg)
	userTurn := &models.ThreadMessageTurn{
		Role: schema.User,
		Events: []*models.MarshaledThreadMessage{
			{
				Type:    models.AgentMessageTypeUserMessage,
				Content: userEventContent,
			},
		},
	}
	a.turns = append(a.turns, userTurn)
	a.history = append(a.history, &schema.Message{
		Role:    schema.User,
		Content: userInput,
	})

	var assistantEventsMu sync.Mutex
	var assistantEvents []*models.MarshaledThreadMessage

	sendAndCollect := func(event models.AgentMessage) {
		eventContent, _ := json.Marshal(event)
		assistantEventsMu.Lock()
		assistantEvents = append(assistantEvents, &models.MarshaledThreadMessage{
			Type:    event.GetType(),
			Content: eventContent,
		})
		assistantEventsMu.Unlock()

		msgChan <- event
	}

	sendOnly := func(event models.AgentMessage) {
		msgChan <- event
	}

	assistantTurn := &models.ThreadMessageTurn{
		Role:   schema.Assistant,
		Events: assistantEvents,
	}
	defer func() {
		assistantEventsMu.Lock()
		assistantTurn.Events = assistantEvents
		hasEvents := len(assistantEvents) > 0
		assistantEventsMu.Unlock()
		if hasEvents {
			a.turns = append(a.turns, assistantTurn)
		}
	}()

	iterations := 0
	totalThinkingTime := 0.0
	totalToolExecutionTime := 0.0
	totalToolCalls := 0
	successfulToolCalls := 0
	failedToolCalls := 0

	for iterations < a.config.MaxIterations {
		select {
		case <-ctx.Done():
			sendAndCollect(models.AgentError{Error: "agent generation cancelled"})
			if runSpan != nil {
				runSpan.SetAttributes(
					attribute.Int("agent.iterations", iterations),
					attribute.Float64("agent.total_thinking_time_seconds", totalThinkingTime),
					attribute.Float64("agent.total_tool_execution_time_seconds", totalToolExecutionTime),
					attribute.Int("agent.total_tool_calls", totalToolCalls),
				)
				runSpan.RecordError(ctx.Err())
				runSpan.SetStatus(codes.Error, ctx.Err().Error())
			}
			return
		default:
		}

		a.waitForNextTurn()

		thinkingStartedAt := time.Now()
		sendAndCollect(models.AgentStartThinking{})

		var contentBuilder strings.Builder
		response, err := a.generate(ctx,
			func(chunk string) {
				sendOnly(models.AgentThinking{Content: chunk})
			},
			func(chunk string) {
				contentBuilder.WriteString(chunk)
				sendOnly(models.AgentStreamChunk{Content: chunk})
			},
		)
		thinkingDuration := time.Since(thinkingStartedAt).Seconds()
		totalThinkingTime += thinkingDuration

		if err != nil {
			if strings.Contains(err.Error(), "429") {
				if runSpan != nil {
					runSpan.AddEvent("agent.rate_limited", trace.WithAttributes(
						attribute.Int("iteration", iterations),
					))
				}
				time.Sleep(3 * time.Second)
				iterations += 1
				continue
			}
			sendAndCollect(models.AgentError{Error: fmt.Sprintf("agent generation failed: %v", err)})
			if runSpan != nil {
				runSpan.SetAttributes(
					attribute.Int("agent.iterations", iterations),
					attribute.Float64("agent.total_thinking_time_seconds", totalThinkingTime),
				)
			}
			recordSpanError(runSpan, err)
			return
		}
		if response != nil {
			if response.Content == "" {
				response.Content = contentBuilder.String()
			}
		}
		if response.ResponseMeta != nil && response.ResponseMeta.Usage != nil {
			usage := response.ResponseMeta.Usage
			a.stats.Usage.PromptTokens += usage.PromptTokens
			a.stats.Usage.CompletionTokens += usage.CompletionTokens
			a.stats.Usage.TotalTokens = a.stats.Usage.PromptTokens + a.stats.Usage.CompletionTokens
			if runSpan != nil {
				runSpan.AddEvent("agent.usage", trace.WithAttributes(
					attribute.Int("prompt_tokens", usage.PromptTokens),
					attribute.Int("completion_tokens", usage.CompletionTokens),
					attribute.Int("total_tokens", a.stats.Usage.TotalTokens),
				))
			}
		}

		if response.Content == "" && len(response.ToolCalls) == 0 {
			sendAndCollect(models.AgentError{Error: "agent returned an empty response"})
			if runSpan != nil {
				runSpan.SetStatus(codes.Error, "agent returned an empty response")
			}
			return
		}

		durationSeconds := time.Since(thinkingStartedAt).Seconds()
		if response.Content != "" || response.ReasoningContent != "" {
			sendAndCollect(models.AgentThought{
				Reasoning:       response.ReasoningContent,
				Content:         response.Content,
				DurationSeconds: thinkingDuration,
			})
			if runSpan != nil {
				runSpan.AddEvent("agent.thought", trace.WithAttributes(
					attribute.Int("iteration", iterations),
					attribute.Float64("thinking_duration_seconds", thinkingDuration),
					attribute.Int("thought_length", len(response.Content)),
					attribute.Int("reasoning_length", len(response.ReasoningContent)),
				))
			}
		} else if len(response.ToolCalls) > 0 {
			sendAndCollect(models.AgentThought{
				Content:         "",
				DurationSeconds: durationSeconds,
			})
		}
		a.history = append(a.history, response)

		if len(response.ToolCalls) == 0 {
			sendAndCollect(models.AgentFinalResponse{})
			if runSpan != nil {
				totalDuration := time.Since(runStartTime).Seconds()
				runSpan.SetAttributes(
					attribute.Int("agent.iterations", iterations+1),
					attribute.Float64("agent.total_duration_seconds", totalDuration),
					attribute.Float64("agent.total_thinking_time_seconds", totalThinkingTime),
					attribute.Float64("agent.total_tool_execution_time_seconds", totalToolExecutionTime),
					attribute.Int("agent.total_tool_calls", totalToolCalls),
					attribute.Int("agent.successful_tool_calls", successfulToolCalls),
					attribute.Int("agent.failed_tool_calls", failedToolCalls),
					attribute.Int("agent.final_history_size", len(a.history)),
				)
				runSpan.SetStatus(codes.Ok, "agent finished")
			}
			return
		}

		type toolResult struct {
			call   schema.ToolCall
			result string
			err    error
		}

		toolResultChan := make(chan toolResult, len(response.ToolCalls))
		totalToolCalls += len(response.ToolCalls)

		if runSpan != nil {
			runSpan.AddEvent("agent.tool_calls_batch", trace.WithAttributes(
				attribute.Int("iteration", iterations),
				attribute.Int("tool_count", len(response.ToolCalls)),
			))
		}

		for _, toolCall := range response.ToolCalls {
			go func(tc schema.ToolCall) {
				toolID := int(atomic.AddInt64(&a.stats.NextExecutingToolID, 1))
				startedAt := time.Now()
				sendAndCollect(models.AgentExecutingToolStart{
					ID:   toolID,
					Name: tc.Function.Name,
					Args: tc.Function.Arguments,
				})

				result, err := a.invokeTool(ctx, tc)

				durationSeconds := time.Since(startedAt).Seconds()
				totalToolExecutionTime += durationSeconds

				if err != nil {
					failedToolCalls += 1
				} else {
					successfulToolCalls += 1
				}

				content := result
				if err != nil {
					content = fmt.Sprintf("Error: %v", err)
				}

				sendAndCollect(models.AgentExecutingToolFinish{
					ID:              toolID,
					Name:            tc.Function.Name,
					Args:            tc.Function.Arguments,
					Content:         content,
					DurationSeconds: durationSeconds,
				})

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
				a.history = append(a.history, &schema.Message{
					Role:       schema.Tool,
					ToolCallID: tc.ID,
					Content:    content,
				})
			}
		}

		iterations += 1
	}

	sendAndCollect(models.AgentError{
		Error: fmt.Sprintf("Sorry, I've reached the maximum iterations %d and still couldn't produce a final answer", a.config.MaxIterations),
	})
	if runSpan != nil {
		totalDuration := time.Since(runStartTime).Seconds()
		runSpan.SetAttributes(
			attribute.Int("agent.iterations", iterations),
			attribute.Float64("agent.total_duration_seconds", totalDuration),
			attribute.Float64("agent.total_thinking_time_seconds", totalThinkingTime),
			attribute.Float64("agent.total_tool_execution_time_seconds", totalToolExecutionTime),
			attribute.Int("agent.total_tool_calls", totalToolCalls),
			attribute.Int("agent.successful_tool_calls", successfulToolCalls),
			attribute.Int("agent.failed_tool_calls", failedToolCalls),
			attribute.Int("agent.final_history_size", len(a.history)),
		)
		runSpan.SetStatus(codes.Error, "agent reached max iterations")
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

func (a *Agent) generate(ctx context.Context, onThinking func(string), onContent func(string)) (*schema.Message, error) {
	var span trace.Span
	if isTelemetryEnabled() {
		tracer := otel.Tracer("alter/agent")
		ctx, span = tracer.Start(ctx, "agent.generate",
			trace.WithAttributes(
				attribute.String("agent.model_id", a.config.ModelID),
			),
		)
		defer span.End()
	}

	a.history = sanitizeHistory(a.config.WorkDir, a.history)

	cm, err := GetModel(ctx, a.config.ModelID)
	if err != nil {
		recordSpanError(span, err)
		return nil, err
	}

	modelWithTools, err := cm.WithTools(a.tools)
	if err != nil {
		recordSpanError(span, err)
		return nil, err
	}

	stream, err := modelWithTools.Stream(ctx, a.history)
	if err != nil {
		recordSpanError(span, err)
		return nil, err
	}
	defer stream.Close()

	var finalResponse *schema.Message
	var contentBuilder strings.Builder
	var reasoningBuilder strings.Builder
	var allToolCalls []schema.ToolCall
	toolCallsMap := make(map[int]*schema.ToolCall)

	finalResponse = &schema.Message{
		Role: schema.Assistant,
	}

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			recordSpanError(span, err)
			return nil, err
		}

		if chunk.ReasoningContent != "" && chunk.ReasoningContent == chunk.Content {
			chunk.ReasoningContent = ""
		}

		if chunk.ReasoningContent != "" {
			reasoningBuilder.WriteString(chunk.ReasoningContent)
			if onThinking != nil {
				onThinking(chunk.ReasoningContent)
			}
		}

		if chunk.Content != "" {
			contentBuilder.WriteString(chunk.Content)
			if onContent != nil {
				onContent(chunk.Content)
			}
		}

		if chunk.ResponseMeta != nil {
			finalResponse.ResponseMeta = chunk.ResponseMeta
		}

		for _, tc := range chunk.ToolCalls {
			idx := 0
			if tc.Index != nil {
				idx = *tc.Index
			}

			if existing, ok := toolCallsMap[idx]; ok {
				existing.Function.Arguments += tc.Function.Arguments
			} else {
				newTC := tc
				toolCallsMap[idx] = &newTC
			}
		}
	}

	var indices []int
	for idx := range toolCallsMap {
		indices = append(indices, idx)
	}
	sort.Ints(indices)
	for _, idx := range indices {
		allToolCalls = append(allToolCalls, *toolCallsMap[idx])
	}

	finalResponse.Content = contentBuilder.String()
	finalResponse.ReasoningContent = reasoningBuilder.String()
	finalResponse.ToolCalls = allToolCalls

	if finalResponse.ReasoningContent != "" && finalResponse.ReasoningContent == finalResponse.Content {
		finalResponse.ReasoningContent = ""
	}

	if span != nil {
		span.SetAttributes(
			attribute.Int("agent.history_length", len(a.history)),
			attribute.Int("agent.tool_calls", len(finalResponse.ToolCalls)),
			attribute.Int("agent.response_length", len(finalResponse.Content)),
		)
		if finalResponse.ResponseMeta != nil && finalResponse.ResponseMeta.Usage != nil {
			span.SetAttributes(
				attribute.Int("llm.prompt_tokens", finalResponse.ResponseMeta.Usage.PromptTokens),
				attribute.Int("llm.completion_tokens", finalResponse.ResponseMeta.Usage.CompletionTokens),
				attribute.Int("llm.total_tokens", finalResponse.ResponseMeta.Usage.PromptTokens+finalResponse.ResponseMeta.Usage.CompletionTokens),
			)
		}
		span.SetStatus(codes.Ok, "generation successful")
	}

	return finalResponse, nil
}

func (a *Agent) invokeTool(ctx context.Context, toolCall schema.ToolCall) (string, error) {
	var span trace.Span
	if isTelemetryEnabled() {
		tracer := otel.Tracer("alter/agent")
		ctx, span = tracer.Start(ctx, "agent.tool_call",
			trace.WithAttributes(
				attribute.String("tool.name", toolCall.Function.Name),
				attribute.String("tool.call_id", toolCall.ID),
				attribute.String("tool.args", toolCall.Function.Arguments),
			),
		)
		defer span.End()
	}

	targetTool, exists := a.toolsMap[toolCall.Function.Name]
	if !exists {
		err := fmt.Errorf("agent tool not found: %s", toolCall.Function.Name)
		recordSpanError(span, err)
		return "", err
	}

	result, err := targetTool.InvokableRun(ctx, toolCall.Function.Arguments)
	if err != nil {
		recordSpanError(span, err)
		return "", err
	}

	if span != nil {
		span.SetAttributes(
			attribute.Int("tool.result_length", len(result)),
			attribute.Bool("tool.success", true),
		)
		span.SetStatus(codes.Ok, "tool execution successful")
	}
	return result, nil
}

func recordSpanError(span trace.Span, err error) {
	if err == nil || span == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
