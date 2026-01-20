package bash

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

const (
	BashToolName        = "bash"
	BashToolDescription = "Executes a single-line bash command and returns the combined output with the exit code. Supported commands: ls, tree, rg, grep, find, cat, head, tail, sed, awk, git. Shell operators (|, &, ;, >, <, `, $()) are not supported."
)

type BashParams struct {
	Command        string `json:"command" jsonschema:"description=The bash command to execute."`
	WorkDir        string `json:"work_dir" jsonschema:"description=The absolute path of the directory to run the command in."`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty" jsonschema:"description=Maximum execution time in seconds. If the value is less than or equal to 0, it defaults to 10 seconds."`
}

func GetBashTool(ctx context.Context) (*schema.ToolInfo, tool.InvokableTool, error) {
	t, err := utils.InferTool(BashToolName, BashToolDescription, BashTool)
	if err != nil {
		return nil, nil, err
	}

	info, err := t.Info(ctx)
	if err != nil {
		return nil, nil, err
	}

	return info, t, nil
}

const (
	BashRunToolName        = "bash_run"
	BashRunToolDescription = "Runs a specified bash/shell script file with optional arguments and returns the output with the exit code."
)

type BashRunParams struct {
	ScriptPath     string   `json:"script_path" jsonschema:"description=The absolute path to the bash/shell script file to execute."`
	WorkDir        string   `json:"work_dir" jsonschema:"description=The absolute path of the directory to run the script in."`
	Args           []string `json:"args,omitempty" jsonschema:"description=Optional arguments to pass to the script."`
	Env            []string `json:"env,omitempty" jsonschema:"description=Environment variables in KEY=VALUE format to set for the script execution."`
	TimeoutSeconds int      `json:"timeout_seconds,omitempty" jsonschema:"description=Maximum execution time in seconds. If the value is less than or equal to 0, it defaults to 120 seconds."`
}

func GetBashRunTool(ctx context.Context) (*schema.ToolInfo, tool.InvokableTool, error) {
	t, err := utils.InferTool(BashRunToolName, BashRunToolDescription, BashRunTool)
	if err != nil {
		return nil, nil, err
	}

	info, err := t.Info(ctx)
	if err != nil {
		return nil, nil, err
	}

	return info, t, nil
}
