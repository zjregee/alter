package scheduler

import (
	"fmt"
	"time"

	"github.com/zjregee/alter/internal/utils"
)

func GenerateSchedulerID(prefix string) string {
	return fmt.Sprintf("scheduler-%s-%s", prefix, utils.GenerateUUID())
}

func timeToString(t time.Time) string {
	return t.Format(time.RFC3339)
}

func stringToTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, s)
}
