package template

// Template 模板定义
type Template struct {
	ID        string `json:"id"`         // 唯一标识（UUID）
	Name      string `json:"name"`       // 显示名称（如 "Linus人格"）
	Category  string `json:"category"`   // 模板类型（claude_md, settings_json 等）
	Content   string `json:"content"`    // 模板内容（完整文件内容）
	IsBuiltin bool   `json:"is_builtin"` // 是否预定义（true=项目自带，false=用户自定义）
	CreatedAt int64  `json:"created_at"` // 创建时间（毫秒时间戳）
}

// TemplateManager 模板管理器
type TemplateManager struct {
	Templates  map[string]Template // ID -> Template
	Categories map[string][]string // Category -> Template IDs
	configPath string              // 用户配置文件路径
}

// TemplateConfig 持久化配置文件格式（~/.cc-switch/claude_templates.json）
type TemplateConfig struct {
	Version   int                 `json:"version"`   // 配置版本（1）
	Templates map[string]Template `json:"templates"` // 用户自定义模板
}

// TemplateTarget 目标路径定义
type TemplateTarget struct {
	ID          string // 标识（project_root, global, local）
	Name        string // 显示名称
	Path        string // 绝对路径（运行时计算）
	Description string // 说明文字
}
