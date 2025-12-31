package models

import (
	"encoding/json"

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

type MarshaledThreadMessage struct {
	Type    AgentMessageType `json:"type"`
	Content json.RawMessage  `json:"content"`
}

type ThreadMessageTurn struct {
	Role   schema.RoleType           `json:"role"`
	Events []*MarshaledThreadMessage `json:"events"`
}
