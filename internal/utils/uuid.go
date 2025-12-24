package utils

import (
	"strings"

	"github.com/google/uuid"
)

func GenerateUUID() string {
	uuidStr := uuid.New().String()
	return strings.ReplaceAll(uuidStr, "-", "")[:8]
}
