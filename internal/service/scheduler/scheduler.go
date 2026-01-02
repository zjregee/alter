package scheduler

import (
	"container/heap"
	"context"
	"fmt"
	"sync"

	"github.com/zjregee/alter/internal/models"
	"github.com/zjregee/alter/internal/service/storage"
	"github.com/zjregee/alter/internal/service/workflow"
)

var (
	instance *Scheduler
)

func init() {
	instance = newScheduler()

	ctx := context.Background()
	if err := instance.start(ctx); err != nil {
		panic(err)
	}
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

func (s *Scheduler) start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("scheduler is already running")
	}
	s.running = true
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.mu.Unlock()

	if err := s.loadSchedules(); err != nil {
		s.cancel()
		s.cancel = nil
		s.mu.Lock()
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
	s.mu.Unlock()

	s.cancel()
	s.cancel = nil

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
		runs = append(runs, run)
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

	s.mu.Lock()
	s.queue.Remove(scheduleID)
	s.mu.Unlock()

	return storage.SaveSchedule(schedule)
}

func (s *Scheduler) TriggerSchedule(scheduleID string) error {
	schedule, err := storage.GetSchedule(scheduleID)
	if err != nil {
		return err
	}

	s.wg.Add(1)
	go s.executeSchedule(schedule)

	return nil
}

func (s *Scheduler) UpdateSchedule(scheduleID string, updates *models.Schedule) error {
	s.mu.Lock()
	s.queue.Remove(scheduleID)
	s.mu.Unlock()

	if err := storage.SaveSchedule(updates); err != nil {
		return err
	}

	if updates.Enabled {
		nextRun, err := calculateNextRunFromNow(updates)
		if err != nil {
			return err
		}

		updates.NextRunAt = &nextRun
		if err := storage.SaveSchedule(updates); err != nil {
			return err
		}

		s.mu.Lock()
		if s.running {
			heap.Push(s.queue, &scheduleQueueItem{
				schedule: updates,
				nextRun:  nextRun,
			})
		}
		s.mu.Unlock()
	}

	return nil
}
