package models

import (
	"github.com/cloudwego/eino/schema"
)

type ThreadInfo struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Model     string `json:"model"`
	WorkDir   string `json:"work_dir"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type ThreadMessage struct {
	Role      schema.RoleType `json:"role"`
	Content   string          `json:"content"`
	Timestamp int64           `json:"timestamp"`
}
