package skills

import (
	"context"
	"fmt"
	"strings"

	skillsService "github.com/zjregee/alter/internal/service/skills"
)

type serviceSkillsContent struct {
	Name        string
	Description string
	Content     string
}

func LoadSkill(ctx context.Context, params *LoadSkillParams) (string, error) {
	if params == nil {
		return "", fmt.Errorf("params must be provided")
	}

	name := strings.TrimSpace(params.Name)
	if name == "" {
		return "", fmt.Errorf("name must be provided")
	}

	contents, err := skillsService.LoadAllSkillContents()
	if err != nil {
		return "", err
	}

	var matched *serviceSkillsContent
	for _, content := range contents {
		if content == nil || content.Summary == nil {
			continue
		}
		if strings.EqualFold(content.Summary.Name, name) {
			matched = &serviceSkillsContent{
				Name:        content.Summary.Name,
				Description: content.Summary.Description,
				Content:     content.Content,
			}
			break
		}
	}

	if matched == nil {
		return "", fmt.Errorf("skill not found: %s", name)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Name: %s", matched.Name)

	if matched.Description != "" {
		fmt.Fprintf(&b, "\nDescription: %s", matched.Description)
	}

	if matched.Content == "" {
		fmt.Fprint(&b, "\nContent: (empty)")
		return b.String(), nil
	}

	fmt.Fprint(&b, "\nContent:\n```markdown\n")
	fmt.Fprint(&b, strings.TrimRight(matched.Content, "\n"))
	fmt.Fprint(&b, "\n```")

	return b.String(), nil
}
