package models

import (
	"time"
)

type Schedule struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	WorkflowConfig WorkflowConfig `json:"workflow_config"`

	Enabled   bool       `json:"enabled"`
	CronExpr  string     `json:"cron_expr"`
	Timezone  string     `json:"timezone"`
	LastRunAt *time.Time `json:"last_run_at,omitempty"`
	NextRunAt *time.Time `json:"next_run_at,omitempty"`

	MaxRetries     int           `json:"max_retries"`
	RetryInterval  time.Duration `json:"retry_interval"`
	RetryBackoff   float64       `json:"retry_backoff"`
	TimeoutSeconds int           `json:"timeout_seconds"`
}

type ScheduleRun struct {
	ID         string `json:"id"`
	ScheduleID string `json:"schedule_id"`

	Status     WorkflowState `json:"status"`
	Error      string        `json:"error,omitempty"`
	RetryCount int           `json:"retry_count"`
	StartedAt  time.Time     `json:"started_at"`
	EndedAt    *time.Time    `json:"ended_at,omitempty"`
}
