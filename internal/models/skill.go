package models

type SkillSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type SkillContent struct {
	Summary *SkillSummary `json:"summary"`
	Content string        `json:"content"`
}
