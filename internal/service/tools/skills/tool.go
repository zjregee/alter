package skills

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"

	"github.com/zjregee/alter/internal/service/tools"
)

const (
	ListSkillsToolName        = "list_skills"
	ListSkillsToolDescription = "Lists available skill summaries."
	LoadSkillToolName         = "load_skill"
	LoadSkillToolDescription  = "Loads the full content for a specific skill."
)

type ListSkillsParams struct{}

type LoadSkillParams struct {
	Name string `json:"name" jsonschema:"description=The name of the skill to load."`
}

func ListSkillsTool(ctx context.Context) (*schema.ToolInfo, tool.InvokableTool, error) {
	t, err := utils.InferTool(ListSkillsToolName, ListSkillsToolDescription, ListSkills)
	if err != nil {
		return nil, nil, err
	}

	info, err := t.Info(ctx)
	if err != nil {
		return nil, nil, err
	}

	return info, t, nil
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

func init() {
	tools.RegisterTool(ListSkillsToolName, ListSkillsTool)
	tools.RegisterTool(LoadSkillToolName, LoadSkillTool)
}
