package app

import (
	"github.com/zjregee/alter/internal/models"
	"github.com/zjregee/alter/internal/service/scheduler"
)

func (a *App) ListScheduleActiveRuns() []*models.ScheduleRun {
	return scheduler.GetScheduler().GetActiveRuns()
}

func (a *App) ListSchedules() ([]*models.Schedule, error) {
	return scheduler.GetScheduler().GetSchedules()
}

func (a *App) EnableSchedule(id string) error {
	return scheduler.GetScheduler().EnableSchedule(id)
}

func (a *App) DisableSchedule(id string) error {
	return scheduler.GetScheduler().DisableSchedule(id)
}

func (a *App) TriggerSchedule(id string) error {
	return scheduler.GetScheduler().TriggerSchedule(id)
}

func (a *App) ListScheduleRuns(scheduleID string) ([]*models.ScheduleRun, error) {
	return scheduler.GetScheduler().GetScheduleRuns(scheduleID)
}

func (a *App) UpdateSchedule(id string, schedule *models.Schedule) error {
	return scheduler.GetScheduler().UpdateSchedule(id, schedule)
}
