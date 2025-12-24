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

const defaultTimeoutSeconds = 10

var allowedCommands = map[string]struct{}{
	"ls":   {},
	"tree": {},
	"rg":   {},
	"grep": {},
	"cat":  {},
	"head": {},
	"tail": {},
	"sed":  {},
	"awk":  {},
}

func validateAllowedCommand(command string) error {
	if containsShellOperator(command) {
		return fmt.Errorf("command contains unsupported shell operators")
	}

	fields := strings.Fields(command)
	if len(fields) == 0 {
		return fmt.Errorf("command must be provided")
	}

	base := filepath.Base(fields[0])
	if _, ok := allowedCommands[base]; !ok {
		return fmt.Errorf("command is not allowed: %s", base)
	}

	return nil
}

func containsShellOperator(command string) bool {
	if strings.ContainsAny(command, "|;&><`\n") {
		return true
	}
	if strings.Contains(command, "$(") {
		return true
	}

	return false
}

func BashTool(ctx context.Context, params *BashParams) (string, error) {
	if params == nil {
		return "", fmt.Errorf("params must be provided")
	}

	command := strings.TrimSpace(params.Command)
	if command == "" {
		return "", fmt.Errorf("command must be provided")
	}
	if err := validateAllowedCommand(command); err != nil {
		return "", err
	}

	workDir := strings.TrimSpace(params.WorkDir)
	if workDir != "" && !filepath.IsAbs(workDir) {
		return "", fmt.Errorf("path must be absolute: %s", workDir)
	}

	info, err := os.Stat(workDir)
	if err != nil {
		return "", fmt.Errorf("path is not exists: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", workDir)
	}

	timeoutSeconds := params.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = defaultTimeoutSeconds
	}

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "bash", "-lc", command)
	cmd.Dir = workDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return "", fmt.Errorf("command timed out: %w", err)
		}

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return formatResult(command, workDir, exitErr.ExitCode(), output), nil
		}

		return "", fmt.Errorf("command failed to run: %w", err)
	}

	return formatResult(command, workDir, 0, output), nil
}

func formatResult(command string, workDir string, exitCode int, output []byte) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Command: %s\n", command)
	fmt.Fprintf(&b, "Work directory: %s\n", workDir)
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
