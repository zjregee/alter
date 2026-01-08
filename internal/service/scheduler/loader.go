package scheduler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/zjregee/alter/internal/models"
	"github.com/zjregee/alter/internal/utils"
)

const (
	workflowsDirName = ".alter/workflows"
)

type workflowYAML struct {
	Name            string        `yaml:"name"`
	Description     string        `yaml:"description"`
	Prompt          string        `yaml:"prompt"`
	ModelID         string        `yaml:"model_id"`
	MaxIterations   int           `yaml:"max_iterations"`
	RequestInterval time.Duration `yaml:"request_interval"`
	WorkDir         string        `yaml:"work_dir"`
	Schedule        *scheduleYAML `yaml:"schedule"`
}

type scheduleYAML struct {
	CronExpr      string        `yaml:"cron"`
	Timezone      string        `yaml:"timezone"`
	MaxRetries    int           `yaml:"max_retries"`
	RetryInterval time.Duration `yaml:"retry_interval"`
	RetryBackoff  float64       `yaml:"retry_backoff"`
	Timeout       time.Duration `yaml:"timeout"`
}

func loadSchedulesFromFiles() (map[string]*models.Schedule, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	workflowsDir := filepath.Join(home, workflowsDirName)
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflows directory: %w", err)
	}

	schedules := make(map[string]*models.Schedule)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(workflowsDir, entry.Name())
		schedule, err := parseWorkflowFile(path)
		if err != nil {
			utils.GetLogger().Printf("Warning: failed to parse workflow file %s: %v", entry.Name(), err)
			continue
		}

		if schedule != nil {
			id := strings.TrimSuffix(entry.Name(), ext)
			schedule.ID = id
			schedules[id] = schedule
		}
	}

	return schedules, nil
}

func parseWorkflowFile(path string) (*models.Schedule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var wf workflowYAML
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return nil, err
	}

	if wf.Schedule == nil {
		return nil, nil
	}

	timezone := wf.Schedule.Timezone
	if timezone == "" {
		timezone = "Local"
	}

	return &models.Schedule{
		Name: wf.Name,
		WorkflowConfig: models.WorkflowConfig{
			Name:            wf.Name,
			Description:     wf.Description,
			Prompt:          wf.Prompt,
			ModelID:         wf.ModelID,
			MaxIterations:   wf.MaxIterations,
			RequestInterval: wf.RequestInterval,
			WorkDir:         wf.WorkDir,
		},
		Enabled:        false,
		CronExpr:       wf.Schedule.CronExpr,
		Timezone:       timezone,
		MaxRetries:     wf.Schedule.MaxRetries,
		RetryInterval:  int64(wf.Schedule.RetryInterval),
		RetryBackoff:   wf.Schedule.RetryBackoff,
		TimeoutSeconds: int(wf.Schedule.Timeout.Seconds()),
	}, nil
}
