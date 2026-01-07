package skills

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

const (
	LoadSkillToolName        = "load_skill"
	LoadSkillToolDescription = "Loads the full content for a specific skill."
)

type LoadSkillParams struct {
	Name string `json:"name" jsonschema:"description=The name of the skill to load."`
}

func LoadSkillTool(ctx context.Context) (*schema.ToolInfo, tool.InvokableTool, error) {
	t, err := utils.InferTool(LoadSkillToolName, LoadSkillToolDescription, LoadSkill)
	if err != nil {
		return nil, nil, err
	}

	info, err := t.Info(ctx)
	if err != nil {
		return nil, nil, err
	}

	return info, t, nil
}
