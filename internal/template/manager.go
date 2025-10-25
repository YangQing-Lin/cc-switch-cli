package template

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed claude_templates/*
//go:embed codex_templates/*
var builtinTemplatesFS embed.FS

const (
	CategoryClaudeMd = "claude_md"
	CategoryCodexMd  = "codex_md"
	ConfigVersion    = 1
)

// NewTemplateManager 创建新的模板管理器
func NewTemplateManager(configPath string) (*TemplateManager, error) {
	tm := &TemplateManager{
		Templates:  make(map[string]Template),
		Categories: make(map[string][]string),
		configPath: configPath,
	}

	// 加载预定义模板
	if err := tm.loadBuiltinTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load builtin templates: %w", err)
	}

	// 加载用户自定义模板
	if err := tm.loadUserTemplates(); err != nil {
		// 如果文件不存在，创建默认配置
		if os.IsNotExist(err) {
			if err := tm.saveUserTemplates(); err != nil {
				return nil, fmt.Errorf("failed to create default config: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to load user templates: %w", err)
		}
	}

	return tm, nil
}

// loadBuiltinTemplates 从嵌入文件系统加载预定义模板
func (tm *TemplateManager) loadBuiltinTemplates() error {
	type builtinSet struct {
		dir      string
		category string
	}

	builtinSets := []builtinSet{
		{dir: "claude_templates", category: CategoryClaudeMd},
		{dir: "codex_templates", category: CategoryCodexMd},
	}

	for _, set := range builtinSets {
		entries, err := builtinTemplatesFS.ReadDir(set.dir)
		if err != nil {
			// 如果目录不存在则跳过，允许部分模式缺失
			if os.IsNotExist(err) {
				continue
			}
			return err
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			filename := entry.Name()
			if !strings.HasSuffix(filename, ".md") {
				continue
			}

			// 读取模板内容
			// 注意：embed.FS 要求使用正斜杠 / 作为路径分隔符，不能使用 filepath.Join
			content, err := builtinTemplatesFS.ReadFile(set.dir + "/" + filename)
			if err != nil {
				return fmt.Errorf("failed to read %s: %w", filename, err)
			}

			// 生成模板 ID（去掉 .md 后缀）
			id := strings.TrimSuffix(filename, ".md")

			// 生成友好的显示名称
			name := tm.generateDisplayName(id)

			template := Template{
				ID:        id,
				Name:      name,
				Category:  set.category,
				Content:   string(content),
				IsBuiltin: true,
				CreatedAt: time.Now().UnixMilli(),
			}

			tm.Templates[id] = template
			tm.addToCategory(set.category, id)
		}
	}

	return nil
}

// loadUserTemplates 从配置文件加载用户自定义模板
func (tm *TemplateManager) loadUserTemplates() error {
	data, err := os.ReadFile(tm.configPath)
	if err != nil {
		return err
	}

	var config TemplateConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// 合并用户模板到管理器
	for id, template := range config.Templates {
		tm.Templates[id] = template
		tm.addToCategory(template.Category, id)
	}

	return nil
}

// saveUserTemplates 保存用户自定义模板到配置文件
func (tm *TemplateManager) saveUserTemplates() error {
	// 只保存用户自定义模板
	userTemplates := make(map[string]Template)
	for id, template := range tm.Templates {
		if !template.IsBuiltin {
			userTemplates[id] = template
		}
	}

	config := TemplateConfig{
		Version:   ConfigVersion,
		Templates: userTemplates,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// 确保目录存在
	dir := filepath.Dir(tm.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// 原子写入
	tempFile := tm.configPath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tempFile, tm.configPath); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// GetTemplate 根据 ID 获取模板
func (tm *TemplateManager) GetTemplate(id string) (*Template, error) {
	template, exists := tm.Templates[id]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", id)
	}
	return &template, nil
}

// ListTemplates 列出指定分类的所有模板
func (tm *TemplateManager) ListTemplates(category string) []Template {
	ids, exists := tm.Categories[category]
	if !exists {
		return []Template{}
	}

	templates := make([]Template, 0, len(ids))
	for _, id := range ids {
		if template, exists := tm.Templates[id]; exists {
			templates = append(templates, template)
		}
	}

	return templates
}

// AddTemplate 添加用户自定义模板
func (tm *TemplateManager) AddTemplate(name, category, content string) (string, error) {
	// 生成唯一 ID
	id := fmt.Sprintf("user_%d", time.Now().UnixMilli())

	template := Template{
		ID:        id,
		Name:      name,
		Category:  category,
		Content:   content,
		IsBuiltin: false,
		CreatedAt: time.Now().UnixMilli(),
	}

	tm.Templates[id] = template
	tm.addToCategory(category, id)

	// 持久化
	if err := tm.saveUserTemplates(); err != nil {
		return "", err
	}

	return id, nil
}

// DeleteTemplate 删除用户自定义模板
func (tm *TemplateManager) DeleteTemplate(id string) error {
	template, exists := tm.Templates[id]
	if !exists {
		return fmt.Errorf("template not found: %s", id)
	}

	if template.IsBuiltin {
		return fmt.Errorf("cannot delete builtin template: %s", id)
	}

	// 从分类中移除
	tm.removeFromCategory(template.Category, id)

	// 从模板列表中移除
	delete(tm.Templates, id)

	// 持久化
	return tm.saveUserTemplates()
}

// ApplyTemplate 应用模板到目标路径
func (tm *TemplateManager) ApplyTemplate(templateID string, targetPath string) error {
	template, err := tm.GetTemplate(templateID)
	if err != nil {
		return err
	}

	// 确保目录存在
	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 原子写入
	tempFile := targetPath + ".tmp"
	if err := os.WriteFile(tempFile, []byte(template.Content), 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tempFile, targetPath); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// SaveAsTemplate 从目标路径保存为新模板
func (tm *TemplateManager) SaveAsTemplate(sourcePath, name, category string) (string, error) {
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to read source file: %w", err)
	}

	return tm.AddTemplate(name, category, string(content))
}

// GetDiff 获取模板与目标文件的差异
func (tm *TemplateManager) GetDiff(templateID string, targetPath string) (string, error) {
	template, err := tm.GetTemplate(templateID)
	if err != nil {
		return "", err
	}

	// 读取当前文件内容（如果不存在，视为空文件）
	currentContent := ""
	if data, err := os.ReadFile(targetPath); err == nil {
		currentContent = string(data)
	}

	// 生成 diff
	diff := GenerateDiff(currentContent, template.Content, "Current", "Template: "+template.Name)
	return diff, nil
}

// generateDisplayName 根据 ID 生成友好的显示名称
func (tm *TemplateManager) generateDisplayName(id string) string {
	// 将 kebab-case 转换为标题格式
	parts := strings.Split(id, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, " ")
}

// addToCategory 将模板添加到分类
func (tm *TemplateManager) addToCategory(category, id string) {
	if _, exists := tm.Categories[category]; !exists {
		tm.Categories[category] = []string{}
	}
	tm.Categories[category] = append(tm.Categories[category], id)
}

// removeFromCategory 从分类中移除模板
func (tm *TemplateManager) removeFromCategory(category, id string) {
	ids, exists := tm.Categories[category]
	if !exists {
		return
	}

	newIds := []string{}
	for _, existingID := range ids {
		if existingID != id {
			newIds = append(newIds, existingID)
		}
	}

	tm.Categories[category] = newIds
}

// GenerateDefaultTemplateName 生成默认模板名称（避免冲突）
func (tm *TemplateManager) GenerateDefaultTemplateName() string {
	prefix := "用户配置"
	counter := 1

	for {
		name := fmt.Sprintf("%s%d", prefix, counter)
		exists := false

		for _, template := range tm.Templates {
			if template.Name == name {
				exists = true
				break
			}
		}

		if !exists {
			return name
		}

		counter++
	}
}
