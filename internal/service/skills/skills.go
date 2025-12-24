package skills

import (
	"bufio"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/zjregee/alter/internal/models"
	assets "github.com/zjregee/alter/internal/service/assets/skills"
)

const skillsRoot = "."

type skillFrontMatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

func LoadAllSkillContents() ([]*models.SkillContent, error) {
	_, contents, err := loadSkills()
	if err != nil {
		return nil, err
	}

	return contents, nil
}

func LoadAllSkillSummaries() ([]*models.SkillSummary, error) {
	summaries, _, err := loadSkills()
	if err != nil {
		return nil, err
	}

	return summaries, nil
}

func loadSkills() ([]*models.SkillSummary, []*models.SkillContent, error) {
	skillFiles, err := listSkillFiles(skillsRoot)
	if err != nil {
		return nil, nil, err
	}

	summaries := make([]*models.SkillSummary, 0, len(skillFiles))
	contents := make([]*models.SkillContent, 0, len(skillFiles))

	for _, filePath := range skillFiles {
		content, err := fs.ReadFile(assets.SkillsFS, filePath)
		if err != nil {
			return nil, nil, fmt.Errorf("read skill file %s: %w", filePath, err)
		}

		summary, body, err := parseSkillFile(filePath, string(content))
		if err != nil {
			return nil, nil, err
		}

		summaryCopy := *summary
		summaries = append(summaries, &summaryCopy)
		contents = append(contents, &models.SkillContent{Summary: summary, Content: body})
	}

	return summaries, contents, nil
}

func listSkillFiles(root string) ([]string, error) {
	var files []string
	err := fs.WalkDir(assets.SkillsFS, root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return nil
		}

		if strings.EqualFold(entry.Name(), "SKILL.md") {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

func parseSkillFile(filePath string, content string) (*models.SkillSummary, string, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "---" {
		return nil, "", fmt.Errorf("invalid skill file front matter: %s", filePath)
	}

	var yamlLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			break
		}
		yamlLines = append(yamlLines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, "", fmt.Errorf("read skill file front matter %s: %w", filePath, err)
	}

	var frontMatter skillFrontMatter
	if err := yaml.Unmarshal([]byte(strings.Join(yamlLines, "\n")), &frontMatter); err != nil {
		return nil, "", fmt.Errorf("parse skill file front matter %s: %w", filePath, err)
	}

	var bodyLines []string
	for scanner.Scan() {
		bodyLines = append(bodyLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, "", fmt.Errorf("read skill file body %s: %w", filePath, err)
	}

	if frontMatter.Name == "" {
		frontMatter.Name = path.Base(path.Dir(filePath))
	}

	return &models.SkillSummary{
		Name:        frontMatter.Name,
		Description: frontMatter.Description,
	}, strings.Join(bodyLines, "\n"), nil
}
