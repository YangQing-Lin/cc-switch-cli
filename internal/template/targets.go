package template

import (
	"fmt"
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

// GetCodexMdTargets 返回 CODEX.md 的三个目标路径
func GetCodexMdTargets() ([]TemplateTarget, error) {
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
			Name:        "~/.codex/CODEX.md",
			Path:        filepath.Join(homeDir, ".codex", "CODEX.md"),
			Description: "全局 Codex/OpenAI 指南",
		},
		{
			ID:          "project_root",
			Name:        "./CODEX.md",
			Path:        filepath.Join(cwd, "CODEX.md"),
			Description: "项目根目录的 Codex 模式说明",
		},
		{
			ID:          "local",
			Name:        "./CODEX.local.md",
			Path:        filepath.Join(cwd, "CODEX.local.md"),
			Description: "项目本地 Codex 配置（Git 忽略）",
		},
	}, nil
}

// GetTargetsForCategory 根据模板分类获取目标路径列表
func GetTargetsForCategory(category string) ([]TemplateTarget, error) {
	switch category {
	case CategoryClaudeMd:
		return GetClaudeMdTargets()
	case CategoryCodexMd:
		return GetCodexMdTargets()
	default:
		return nil, fmt.Errorf("unsupported template category: %s", category)
	}
}

// GetTargetByCategory 根据分类和 ID 获取目标路径
func GetTargetByCategory(category, id string) (*TemplateTarget, error) {
	targets, err := GetTargetsForCategory(category)
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

// GetTargetByID 根据 ID 获取目标路径
func GetTargetByID(id string) (*TemplateTarget, error) {
	return GetTargetByCategory(CategoryClaudeMd, id)
}
