package scheduler

import (
	"container/heap"
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"

	"github.com/zjregee/alter/internal/models"
	"github.com/zjregee/alter/internal/service"
	"github.com/zjregee/alter/internal/service/storage"
	"github.com/zjregee/alter/internal/service/workflow"
	"github.com/zjregee/alter/internal/utils"
)

const (
	schedulerTickInterval = 1 * time.Second
)

func (s *Scheduler) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(schedulerTickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case now := <-ticker.C:
			s.processSchedules(now)
		}
	}
}

func (s *Scheduler) processSchedules(now time.Time) {
	for {
		s.mu.Lock()
		if s.queue.Len() == 0 {
			s.mu.Unlock()
			return
		}
		item := s.queue.Peek()
		if item == nil || now.Before(item.nextRun) {
			s.mu.Unlock()
			return
		}

		nextRun, err := calculateNextRun(item.schedule, now)
		if err != nil {
			utils.GetLogger().Printf("Critical: Failed to calculate next run for schedule %s, disabling: %v", item.schedule.ID, err)

			heap.Remove(s.queue, item.index)
			s.mu.Unlock()

			item.schedule.Enabled = false
			if err := storage.SaveSchedule(item.schedule); err != nil {
				utils.GetLogger().Printf("Failed to save disabled schedule %s: %v", item.schedule.ID, err)
			}
			continue
		}

		item.schedule.LastRunAt = timeToString(time.Now())
		item.schedule.NextRunAt = timeToString(nextRun)
		item.nextRun = nextRun
		heap.Fix(s.queue, item.index)

		scheduleCopy := *item.schedule

		currentSchedulePtr := item.schedule

		s.mu.Unlock()

		s.wg.Add(1)
		go s.executeSchedule(&scheduleCopy)

		s.mu.Lock()
		storedItem, exists := s.queue.index[item.schedule.ID]
		shouldSave := exists && storedItem == item && item.schedule == currentSchedulePtr
		s.mu.Unlock()

		if shouldSave {
			if err := storage.SaveSchedule(&scheduleCopy); err != nil {
				utils.GetLogger().Printf("Failed to save schedule %s: %v", item.schedule.ID, err)
				continue
			}
		}
	}
}

func (s *Scheduler) executeSchedule(schedule *models.Schedule) {
	defer s.wg.Done()

	run := &models.ScheduleRun{
		ID:         GenerateSchedulerID("schedule-run"),
		ScheduleID: schedule.ID,
		Status:     models.WorkflowStatePending,
		RetryCount: 0,
	}

	if err := storage.SaveScheduleRun(run); err != nil {
		utils.GetLogger().Printf("Failed to save schedule run: %v", err)
		return
	}

	s.activeMu.Lock()
	s.activeRuns[run.ID] = run
	s.activeMu.Unlock()

	defer func() {
		s.activeMu.Lock()
		delete(s.activeRuns, run.ID)
		s.activeMu.Unlock()
	}()

	s.activeMu.Lock()
	startedAt := time.Now()
	run.StartedAt = timeToString(startedAt)
	run.Status = models.WorkflowStateRunning
	s.activeMu.Unlock()

	if err := storage.SaveScheduleRun(run); err != nil {
		utils.GetLogger().Printf("Failed to save schedule run status Running: %v", err)
	}

	err := s.executeScheduleWithRetry(schedule, run)

	s.activeMu.Lock()
	if err != nil {
		run.Status = models.WorkflowStateFailed
		run.Error = err.Error()
	} else {
		run.Status = models.WorkflowStateFinished
	}

	now := time.Now()
	run.EndedAt = timeToString(now)
	s.activeMu.Unlock()

	if err := storage.SaveScheduleRun(run); err != nil {
		utils.GetLogger().Printf("Failed to save schedule run %s: %v", run.ID, err)
		return
	}
}

func (s *Scheduler) executeScheduleWithRetry(schedule *models.Schedule, run *models.ScheduleRun) error {
	maxRetries := schedule.MaxRetries
	retryInterval := time.Duration(schedule.RetryInterval)
	backoff := schedule.RetryBackoff

	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			waitDuration := time.Duration(float64(retryInterval) * math.Pow(backoff, float64(attempt-1)))
			utils.GetLogger().Printf("Retrying schedule %s (attempt %d/%d) after %v", schedule.ID, attempt, maxRetries, waitDuration)

			select {
			case <-s.ctx.Done():
				return s.ctx.Err()
			case <-time.After(waitDuration):
			}
		}

		s.activeMu.Lock()
		run.RetryCount = attempt
		s.activeMu.Unlock()

		wf := workflow.NewWorkflow(schedule.WorkflowConfig)

		executeCtx := s.ctx
		var cancel context.CancelFunc
		if schedule.TimeoutSeconds > 0 {
			executeCtx, cancel = context.WithTimeout(s.ctx, time.Duration(schedule.TimeoutSeconds)*time.Second)
		}

		execResult := &workflow.ExecutionResult{}
		executeCtx = workflow.WithExecutionResult(executeCtx, execResult)

		err := s.executor.Execute(executeCtx, wf)
		if cancel != nil {
			cancel()
		}
		if err == nil {
			summaryCtx, summaryCancel := context.WithTimeout(s.ctx, 30*time.Second)
			defer summaryCancel()

			summary, sumErr := s.generateSummary(summaryCtx, schedule, execResult)
			if sumErr != nil {
				utils.GetLogger().Printf("Failed to generate summary for schedule %s: %v", schedule.ID, sumErr)
			} else {
				s.activeMu.Lock()
				run.Summary = summary
				s.activeMu.Unlock()
			}
			return nil
		}
		lastErr = err

		if err := storage.SaveScheduleRun(run); err != nil {
			return err
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", maxRetries+1, lastErr)
}

func (s *Scheduler) generateSummary(ctx context.Context, schedule *models.Schedule, result *workflow.ExecutionResult) (string, error) {
	if result == nil || len(result.Record) == 0 {
		return "", nil
	}

	var sb strings.Builder
	for _, msg := range result.Record {
		switch m := msg.(type) {
		case models.AgentThought:
			if m.Content != "" {
				fmt.Fprintf(&sb, "Thought: %s\n", m.Content)
			}
		case models.AgentExecutingToolFinish:
			fmt.Fprintf(&sb, "Tool %s executed\n", m.Name)
		case models.AgentError:
			fmt.Fprintf(&sb, "Error: %s\n", m.Error)
		}
	}

	logContent := sb.String()
	if logContent == "" {
		return "", nil
	}

	prompt := fmt.Sprintf("Based on the execution log below, provide a concise summary of the workflow's actions and results. Focus on the key steps taken and the final outcome. The summary should be informative but brief. Do not include any reasoning or prefixes.\n\nLog:\n%s", logContent)

	model, err := service.GetModel(ctx, schedule.WorkflowConfig.ModelID)
	if err != nil {
		return "", err
	}

	messages := []*schema.Message{
		{
			Role:    schema.User,
			Content: prompt,
		},
	}

	response, err := model.Generate(ctx, messages)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(response.Content), nil
}

func (s *Scheduler) loadSchedules() error {
	if err := s.syncSchedules(); err != nil {
		utils.GetLogger().Printf("Error syncing schedules: %v", err)
	}

	schedules, err := storage.LoadSchedules()
	if err != nil {
		return err
	}

	for _, schedule := range schedules {
		if !schedule.Enabled {
			continue
		}

		var nextRun time.Time
		if schedule.NextRunAt == "" {
			var err error
			nextRun, err = calculateNextRunFromNow(schedule)
			if err != nil {
				utils.GetLogger().Printf("Failed to calculate next run for schedule %s: %v", schedule.ID, err)
				continue
			}

			schedule.NextRunAt = timeToString(nextRun)
			if err := storage.SaveSchedule(schedule); err != nil {
				utils.GetLogger().Printf("Failed to save schedule %s: %v", schedule.ID, err)
				continue
			}
		} else {
			var err error
			nextRun, err = stringToTime(schedule.NextRunAt)
			if err != nil {
				utils.GetLogger().Printf("Failed to parse next run time for schedule %s: %v", schedule.ID, err)
				continue
			}
		}

		s.mu.Lock()
		if _, exists := s.queue.index[schedule.ID]; !exists {
			heap.Push(s.queue, &scheduleQueueItem{
				schedule: schedule,
				nextRun:  nextRun,
			})
		}
		s.mu.Unlock()
	}

	return nil
}

func (s *Scheduler) syncSchedules() error {
	fileSchedules, err := loadSchedulesFromFiles()
	if err != nil {
		return err
	}

	dbSchedules, err := storage.LoadSchedules()
	if err != nil {
		return err
	}
	dbSchedulesMap := make(map[string]*models.Schedule)
	for _, s := range dbSchedules {
		dbSchedulesMap[s.ID] = s
	}

	for id, fileSch := range fileSchedules {
		dbSch, exists := dbSchedulesMap[id]
		if !exists {
			utils.GetLogger().Printf("Creating new schedule from file: %s", id)
			if err := storage.SaveSchedule(fileSch); err != nil {
				utils.GetLogger().Printf("Failed to save new schedule %s: %v", id, err)
			}
		} else {
			configChanged := dbSch.CronExpr != fileSch.CronExpr || dbSch.Timezone != fileSch.Timezone

			dbSch.Name = fileSch.Name
			dbSch.WorkflowConfig = fileSch.WorkflowConfig
			dbSch.CronExpr = fileSch.CronExpr
			dbSch.Timezone = fileSch.Timezone
			dbSch.MaxRetries = fileSch.MaxRetries
			dbSch.RetryInterval = fileSch.RetryInterval
			dbSch.RetryBackoff = fileSch.RetryBackoff
			dbSch.TimeoutSeconds = fileSch.TimeoutSeconds

			if configChanged {
				nextRun, err := calculateNextRunFromNow(dbSch)
				if err == nil {
					dbSch.NextRunAt = timeToString(nextRun)
				}
			}

			if err := storage.SaveSchedule(dbSch); err != nil {
				utils.GetLogger().Printf("Failed to update schedule %s: %v", id, err)
			}
		}

		delete(dbSchedulesMap, id)
	}

	for id, dbSch := range dbSchedulesMap {
		if dbSch.Enabled {
			utils.GetLogger().Printf("Disabling schedule not found in files: %s", id)
			dbSch.Enabled = false
			if err := storage.SaveSchedule(dbSch); err != nil {
				utils.GetLogger().Printf("Failed to disable schedule %s: %v", id, err)
			}
		}
	}

	return nil
}
