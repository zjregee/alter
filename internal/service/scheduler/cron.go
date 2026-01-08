package scheduler

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/zjregee/alter/internal/models"
)

var (
	cronParser = cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)
)

func calculateNextRun(schedule *models.Schedule, from time.Time) (time.Time, error) {
	cronSchedule, err := cronParser.Parse(schedule.CronExpr)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse cron expression: %w", err)
	}

	loc, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to load timezone: %w", err)
	}

	fromInTz := from.In(loc)
	nextRun := cronSchedule.Next(fromInTz)

	return nextRun, nil
}

func calculateNextRunFromNow(schedule *models.Schedule) (time.Time, error) {
	return calculateNextRun(schedule, time.Now())
}
