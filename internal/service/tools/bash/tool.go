package bash

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"

	"github.com/zjregee/alter/internal/service/tools"
)

const (
	BashToolName        = "bash"
	BashToolDescription = "Executes a single-line bash command and returns the combined output with the exit code. Supported commands: ls, tree, rg, grep, cat, head, tail, sed, awk. Shell operators (|, &, ;, >, <, `, $()) are not supported."
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

func init() {
	tools.RegisterTool(BashToolName, GetBashTool)
}
