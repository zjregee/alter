package scheduler

import (
	"container/heap"
	"context"
	"fmt"
	"time"

	"github.com/zjregee/alter/internal/models"
	"github.com/zjregee/alter/internal/service/storage"
	"github.com/zjregee/alter/internal/service/workflow"
	"github.com/zjregee/alter/internal/utils"
)

const (
	schedulerTickInterval = 60 * time.Second
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
		item = heap.Pop(s.queue).(*scheduleQueueItem)
		s.mu.Unlock()

		s.wg.Add(1)
		go s.executeSchedule(item.schedule)

		nextRun, err := calculateNextRunFromNow(item.schedule)
		if err != nil {
			utils.GetLogger().Printf("Failed to calculate next run for schedule %s: %v", item.schedule.ID, err)
			continue
		}

		item.schedule.NextRunAt = timeToString(nextRun)
		item.nextRun = nextRun
		s.mu.Lock()
		heap.Push(s.queue, item)
		s.mu.Unlock()

		if err := storage.SaveSchedule(item.schedule); err != nil {
			utils.GetLogger().Printf("Failed to save schedule %s: %v", item.schedule.ID, err)
			continue
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

	startedAt := time.Now()
	run.StartedAt = timeToString(startedAt)

	err := s.executeScheduleWithRetry(schedule, run)
	if err != nil {
		run.Status = models.WorkflowStateFailed
		run.Error = err.Error()
	} else {
		run.Status = models.WorkflowStateFinished
	}

	now := time.Now()
	run.EndedAt = timeToString(now)
	schedule.LastRunAt = run.StartedAt

	if err := storage.SaveSchedule(schedule); err != nil {
		utils.GetLogger().Printf("Failed to save schedule %s: %v", schedule.ID, err)
		return
	}

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

	for attempt := 0; attempt <= maxRetries; attempt += 1 {
		if attempt > 0 {
			waitDuration := time.Duration(float64(retryInterval) * pow(backoff, float64(attempt-1)))
			utils.GetLogger().Printf("Retrying schedule %s (attempt %d/%d) after %v", schedule.ID, attempt, maxRetries, waitDuration)

			select {
			case <-s.ctx.Done():
				return s.ctx.Err()
			case <-time.After(waitDuration):
			}
		}

		run.RetryCount = attempt

		wf := workflow.NewWorkflow(schedule.WorkflowConfig)

		executeCtx := s.ctx
		var cancel context.CancelFunc
		if schedule.TimeoutSeconds > 0 {
			executeCtx, cancel = context.WithTimeout(s.ctx, time.Duration(schedule.TimeoutSeconds)*time.Second)
		}

		err := s.executor.Execute(executeCtx, wf)
		if cancel != nil {
			cancel()
		}
		if err == nil {
			return nil
		}

		lastErr = err

		if err := storage.SaveScheduleRun(run); err != nil {
			return err
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", maxRetries+1, lastErr)
}

func (s *Scheduler) loadSchedules() error {
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

		heap.Push(s.queue, &scheduleQueueItem{
			schedule: schedule,
			nextRun:  nextRun,
		})
	}

	return nil
}

func pow(base float64, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i += 1 {
		result *= base
	}

	return result
}
