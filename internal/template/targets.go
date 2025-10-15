package template

import (
	"os"
	"path/filepath"
)

// GetClaudeMdTargets 返回 CLAUDE.md 的三个目标路径
func GetClaudeMdTargets() ([]TemplateTarget, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	return []TemplateTarget{
		{
			ID:          "global",
			Name:        "~/.claude/CLAUDE.md",
			Path:        filepath.Join(homeDir, ".claude", "CLAUDE.md"),
			Description: "全局 Claude 配置",
		},
		{
			ID:          "project_root",
			Name:        "./CLAUDE.md",
			Path:        filepath.Join(cwd, "CLAUDE.md"),
			Description: "项目根目录配置",
		},
		{
			ID:          "local",
			Name:        "./CLAUDE.local.md",
			Path:        filepath.Join(cwd, "CLAUDE.local.md"),
			Description: "项目本地配置（Git 忽略）",
		},
	}, nil
}

// GetTargetByID 根据 ID 获取目标路径
func GetTargetByID(id string) (*TemplateTarget, error) {
	targets, err := GetClaudeMdTargets()
	if err != nil {
		return nil, err
	}

	for _, target := range targets {
		if target.ID == id {
			return &target, nil
		}
	}

	return nil, nil
}
