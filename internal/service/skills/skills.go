package skills

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/zjregee/alter/internal/models"
)

const (
	skillsDirName = ".alter/skills"
	skillFileName = "SKILL.md"
)

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
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	skillsPath := filepath.Join(home, skillsDirName)
	if _, err := os.Stat(skillsPath); os.IsNotExist(err) {
		return nil, nil, nil
	}

	skillFiles, err := listSkillFiles(skillsPath)
	if err != nil {
		return nil, nil, err
	}

	summaries := make([]*models.SkillSummary, 0, len(skillFiles))
	contents := make([]*models.SkillContent, 0, len(skillFiles))

	for _, path := range skillFiles {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, fmt.Errorf("read skill file %s: %w", path, err)
		}

		summary, body, err := parseSkillFile(path, string(content))
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
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return nil
		}

		if strings.EqualFold(entry.Name(), skillFileName) {
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
		frontMatter.Name = filepath.Base(filepath.Dir(filePath))
	}

	return &models.SkillSummary{
		Name:        frontMatter.Name,
		Description: frontMatter.Description,
	}, strings.Join(bodyLines, "\n"), nil
}
