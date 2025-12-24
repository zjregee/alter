package service

import (
	"fmt"

	"github.com/zjregee/alter/internal/utils"
)

func GenerateAgentID() string {
	return fmt.Sprintf("agent-%s", utils.GenerateUUID())
}
