package models

type Schedule struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	WorkflowConfig WorkflowConfig `json:"workflow_config"`

	Enabled   bool   `json:"enabled"`
	CronExpr  string `json:"cron_expr"`
	Timezone  string `json:"timezone"`
	LastRunAt string `json:"last_run_at,omitempty"`
	NextRunAt string `json:"next_run_at,omitempty"`

	MaxRetries     int     `json:"max_retries"`
	RetryInterval  int64   `json:"retry_interval"`
	RetryBackoff   float64 `json:"retry_backoff"`
	TimeoutSeconds int     `json:"timeout_seconds"`
}

type ScheduleRun struct {
	ID         string `json:"id"`
	ScheduleID string `json:"schedule_id"`

	Status     WorkflowState             `json:"status"`
	Summary    string                    `json:"summary,omitempty"`
	Error      string                    `json:"error,omitempty"`
	RetryCount int                       `json:"retry_count"`
	StartedAt  string                    `json:"started_at"`
	EndedAt    string                    `json:"ended_at,omitempty"`
	ToolTrace  []*MarshaledThreadMessage `json:"tool_trace,omitempty"`
}
