package bash

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const defaultBashRunTimeoutSeconds = 120

func BashRunTool(ctx context.Context, params *BashRunParams) (string, error) {
	if params == nil {
		return "", fmt.Errorf("params must be provided")
	}

	scriptPath := strings.TrimSpace(params.ScriptPath)
	if scriptPath == "" {
		return "", fmt.Errorf("script_path must be provided")
	}

	if !filepath.IsAbs(scriptPath) {
		return "", fmt.Errorf("script_path must be absolute: %s", scriptPath)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	allowedDir := filepath.Join(homeDir, ".alter", "skills")

	rel, err := filepath.Rel(allowedDir, scriptPath)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("script execution is restricted to %s", allowedDir)
	}

	info, err := os.Stat(scriptPath)
	if err != nil {
		return "", fmt.Errorf("script file does not exist: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("script_path must be a file, not a directory: %s", scriptPath)
	}

	if info.Mode().Perm()&0444 == 0 {
		return "", fmt.Errorf("script file is not readable: %s", scriptPath)
	}
	if info.Mode().Perm()&0111 == 0 {
		return "", fmt.Errorf("script file is not executable: %s", scriptPath)
	}

	workDir := strings.TrimSpace(params.WorkDir)
	if workDir == "" {
		return "", fmt.Errorf("work_dir must be provided")
	} else if !filepath.IsAbs(workDir) {
		return "", fmt.Errorf("work_dir must be absolute: %s", workDir)
	}

	dirInfo, err := os.Stat(workDir)
	if err != nil {
		return "", fmt.Errorf("work directory does not exist: %w", err)
	}
	if !dirInfo.IsDir() {
		return "", fmt.Errorf("work_dir must be a directory: %s", workDir)
	}

	timeoutSeconds := params.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = defaultBashRunTimeoutSeconds
	}

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
		defer cancel()
	}

	args := []string{scriptPath}
	if len(params.Args) > 0 {
		args = append(args, params.Args...)
	}

	cmd := exec.CommandContext(ctx, "bash", args...)
	cmd.Dir = workDir

	if len(params.Env) > 0 {
		for _, env := range params.Env {
			if !strings.Contains(env, "=") {
				return "", fmt.Errorf("invalid environment variable format (expected KEY=VALUE): %s", env)
			}
		}
		cmd.Env = append(os.Environ(), params.Env...)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return "", fmt.Errorf("script execution timed out: %w", err)
		}

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return formatBashRunResult(scriptPath, workDir, params.Args, exitErr.ExitCode(), output), nil
		}

		return "", fmt.Errorf("script failed to run: %w", err)
	}

	return formatBashRunResult(scriptPath, workDir, params.Args, 0, output), nil
}

func formatBashRunResult(scriptPath string, workDir string, args []string, exitCode int, output []byte) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Script: %s\n", scriptPath)
	fmt.Fprintf(&b, "Work directory: %s\n", workDir)
	if len(args) > 0 {
		fmt.Fprintf(&b, "Arguments: %s\n", strings.Join(args, " "))
	} else {
		fmt.Fprintf(&b, "Arguments: (none)\n")
	}
	fmt.Fprintf(&b, "Exit code: %d\n", exitCode)
	if len(output) == 0 {
		fmt.Fprint(&b, "Output: (empty)")
	} else {
		fmt.Fprint(&b, "Output:\n```text\n")
		fmt.Fprint(&b, strings.TrimRight(string(output), "\n"))
		fmt.Fprint(&b, "\n```")
	}

	return b.String()
}
