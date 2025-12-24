package agents

import (
	"fmt"
	"strings"

	"github.com/zjregee/alter/internal/utils"
)

func GenerateAgentID(prefix string) string {
	return fmt.Sprintf("agent-%s-%s", prefix, utils.GenerateUUID())
}

func ParseEnvVar(env string) (key, value string, ok bool) {
	key, value, ok = strings.Cut(env, "=")
	return key, value, ok
}
