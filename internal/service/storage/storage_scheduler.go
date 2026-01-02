package storage

import (
	"encoding/json"
	"fmt"

	"github.com/zjregee/alter/internal/models"
)

const (
	schedulesKeyPrefix    = "schedules:"
	scheduleRunsKeyPrefix = "schedule_runs:"
)

func SaveSchedule(schedule *models.Schedule) error {
	if schedule == nil {
		return fmt.Errorf("schedule is required")
	}

	data, err := json.Marshal(schedule)
	if err != nil {
		return fmt.Errorf("failed to marshal schedule %s: %w", schedule.ID, err)
	}

	return Put([]byte(schedulesKeyPrefix+schedule.ID), data)
}

func GetSchedule(scheduleID string) (*models.Schedule, error) {
	key := []byte(schedulesKeyPrefix + scheduleID)
	data, err := Get(key)
	if err != nil {
		return nil, err
	}

	if data == nil {
		return nil, fmt.Errorf("schedule %s not found", scheduleID)
	}

	var schedule models.Schedule
	if err := json.Unmarshal(data, &schedule); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schedule %s: %w", scheduleID, err)
	}

	return &schedule, nil
}

func LoadSchedules() ([]*models.Schedule, error) {
	entries, err := List([]byte(schedulesKeyPrefix))
	if err != nil {
		return nil, err
	}

	schedules := make([]*models.Schedule, 0, len(entries))
	for key, value := range entries {
		if len(value) == 0 {
			continue
		}

		var stored models.Schedule
		if err := json.Unmarshal(value, &stored); err != nil {
			return nil, fmt.Errorf("failed to unmarshal schedule %s: %w", key, err)
		}

		schedules = append(schedules, &stored)
	}

	return schedules, nil
}

func SaveScheduleRun(run *models.ScheduleRun) error {
	if run == nil {
		return fmt.Errorf("schedule run is required")
	}

	data, err := json.Marshal(run)
	if err != nil {
		return fmt.Errorf("failed to marshal schedule run %s: %w", run.ID, err)
	}

	return Put([]byte(scheduleRunsKeyPrefix+run.ScheduleID+":"+run.ID), data)
}

func LoadScheduleRuns(scheduleID string) ([]*models.ScheduleRun, error) {
	if scheduleID == "" {
		return nil, fmt.Errorf("schedule id is required")
	}

	entries, err := List([]byte(scheduleRunsKeyPrefix + scheduleID + ":"))
	if err != nil {
		return nil, err
	}

	runs := make([]*models.ScheduleRun, 0, len(entries))
	for _, value := range entries {
		if len(value) == 0 {
			continue
		}

		var stored models.ScheduleRun
		if err := json.Unmarshal(value, &stored); err != nil {
			return nil, fmt.Errorf("failed to unmarshal schedule run %s: %w", stored.ID, err)
		}
		runs = append(runs, &stored)
	}

	return runs, nil
}
