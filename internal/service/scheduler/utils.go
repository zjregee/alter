package scheduler

import (
	"fmt"

	"github.com/zjregee/alter/internal/utils"
)

func GenerateSchedulerID(prefix string) string {
	return fmt.Sprintf("scheduler-%s-%s", prefix, utils.GenerateUUID())
}
