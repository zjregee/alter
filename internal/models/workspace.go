package models

type WorkspaceInfo struct {
	Path      string `json:"path"`
	IsDefault bool   `json:"is_default"`
}
