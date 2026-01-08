package scheduler

import (
	"container/heap"
	"context"
	"fmt"
	"sync"

	"github.com/zjregee/alter/internal/models"
	"github.com/zjregee/alter/internal/service/storage"
	"github.com/zjregee/alter/internal/service/workflow"
	"github.com/zjregee/alter/internal/utils"
)

var (
	instance *Scheduler
)

func init() {
	instance = newScheduler()
}

type Scheduler struct {
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	running  bool
	executor *workflow.Executor

	mu    sync.RWMutex
	queue *scheduleQueue

	activeMu   sync.RWMutex
	activeRuns map[string]*models.ScheduleRun
}

func newScheduler() *Scheduler {
	return &Scheduler{
		running:    false,
		executor:   workflow.NewExecutor(),
		queue:      newScheduleQueue(),
		activeRuns: make(map[string]*models.ScheduleRun),
	}
}

func GetScheduler() *Scheduler {
	return instance
}

func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("scheduler is already running")
	}
	s.running = true
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.mu.Unlock()

	if err := s.loadSchedules(); err != nil {
		s.mu.Lock()
		if s.cancel != nil {
			s.cancel()
			s.cancel = nil
		}
		s.running = false
		s.mu.Unlock()
		return err
	}

	s.wg.Add(1)
	go s.run()

	return nil
}

func (s *Scheduler) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return fmt.Errorf("scheduler is not running")
	}
	if s.cancel == nil {
		s.mu.Unlock()
		return fmt.Errorf("scheduler is already stopping")
	}

	cancel := s.cancel
	s.cancel = nil
	s.mu.Unlock()

	cancel()

	s.wg.Wait()

	s.mu.Lock()
	s.running = false
	s.mu.Unlock()

	return nil
}

func (s *Scheduler) GetActiveRuns() []*models.ScheduleRun {
	s.activeMu.RLock()
	defer s.activeMu.RUnlock()

	runs := make([]*models.ScheduleRun, 0, len(s.activeRuns))
	for _, run := range s.activeRuns {
		val := *run
		runs = append(runs, &val)
	}
	return runs
}

func (s *Scheduler) GetSchedules() ([]*models.Schedule, error) {
	return storage.LoadSchedules()
}

func (s *Scheduler) GetScheduleRuns(scheduleID string) ([]*models.ScheduleRun, error) {
	return storage.LoadScheduleRuns(scheduleID)
}

func (s *Scheduler) EnableSchedule(scheduleID string) error {
	schedule, err := storage.GetSchedule(scheduleID)
	if err != nil {
		return err
	}

	if schedule.Enabled {
		return nil
	}

	schedule.Enabled = true
	return s.UpdateSchedule(scheduleID, schedule)
}

func (s *Scheduler) DisableSchedule(scheduleID string) error {
	schedule, err := storage.GetSchedule(scheduleID)
	if err != nil {
		return err
	}

	if !schedule.Enabled {
		return nil
	}

	schedule.Enabled = false

	if err := storage.SaveSchedule(schedule); err != nil {
		return err
	}

	s.mu.Lock()
	s.queue.Remove(scheduleID)
	s.mu.Unlock()

	return nil
}

func (s *Scheduler) TriggerSchedule(scheduleID string) error {
	s.mu.RLock()
	if !s.running {
		s.mu.RUnlock()
		return fmt.Errorf("scheduler is not running")
	}
	s.mu.RUnlock()

	schedule, err := storage.GetSchedule(scheduleID)
	if err != nil {
		return err
	}

	s.wg.Add(1)
	go s.executeSchedule(schedule)

	return nil
}

func (s *Scheduler) UpdateSchedule(scheduleID string, updates *models.Schedule) error {
	old, err := storage.GetSchedule(scheduleID)
	if err != nil {
		old = &models.Schedule{}
	}

	s.mu.Lock()
	if existingItem, ok := s.queue.index[scheduleID]; ok {
		updates.LastRunAt = existingItem.schedule.LastRunAt

		if old.CronExpr == updates.CronExpr && old.Timezone == updates.Timezone {
			updates.NextRunAt = existingItem.schedule.NextRunAt
		}
	}
	s.mu.Unlock()

	if updates.Enabled {
		configChanged := old.CronExpr != updates.CronExpr || old.Timezone != updates.Timezone
		if configChanged || updates.NextRunAt == "" {
			nextRun, err := calculateNextRunFromNow(updates)
			if err != nil {
				return err
			}
			updates.NextRunAt = timeToString(nextRun)
		}

		if err := storage.SaveSchedule(updates); err != nil {
			return err
		}

		nextRun, err := stringToTime(updates.NextRunAt)
		if err != nil {
			return fmt.Errorf("failed to parse next run time: %w", err)
		}

		s.mu.Lock()
		if s.running {
			if existingItem, ok := s.queue.index[scheduleID]; ok {
				if existingItem.schedule.LastRunAt > updates.LastRunAt {
					updates.LastRunAt = existingItem.schedule.LastRunAt
					if err := storage.SaveSchedule(updates); err != nil {
						utils.GetLogger().Printf("Failed to save merged schedule %s: %v", updates.ID, err)
					}
				}

				existingItem.schedule = updates
				existingItem.nextRun = nextRun
				heap.Fix(s.queue, existingItem.index)
			} else {
				heap.Push(s.queue, &scheduleQueueItem{
					schedule: updates,
					nextRun:  nextRun,
				})
			}
		}
		s.mu.Unlock()
	} else {
		if err := storage.SaveSchedule(updates); err != nil {
			return err
		}

		s.mu.Lock()
		s.queue.Remove(scheduleID)
		s.mu.Unlock()
	}

	return nil
}
