package tui

import (
	"os"
	"unicode"

	"github.com/YangQing-Lin/cc-switch-cli/internal/template"
	"github.com/YangQing-Lin/cc-switch-cli/internal/version"
)

// displayWidth 计算字符串的显示宽度（中文等宽字符占2格）
func displayWidth(s string) int {
	width := 0
	for _, r := range s {
		if unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) ||
			unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r) ||
			(r >= 0xFF00 && r <= 0xFFEF) { // 全角字符
			width += 2
		} else {
			width += 1
		}
	}
	return width
}

// getVersion 获取版本号
func (m Model) getVersion() string {
	return version.GetVersion()
}

// columnToAppName 列索引转应用名
func (m Model) columnToAppName(col int) string {
	switch col {
	case 0:
		return "claude"
	case 1:
		return "codex"
	case 2:
		return "gemini"
	default:
		return "claude"
	}
}

// refreshProviders 刷新当前应用的配置列表
func (m *Model) refreshProviders() {
	m.providers = m.manager.ListProvidersForApp(m.currentApp)
}

// refreshAllColumns 刷新三列配置缓存
func (m *Model) refreshAllColumns() {
	m.columnProviders[0] = m.manager.ListProvidersForApp("claude")
	m.columnProviders[1] = m.manager.ListProvidersForApp("codex")
	m.columnProviders[2] = m.manager.ListProvidersForApp("gemini")

	// 确保光标不越界
	for i := 0; i < 3; i++ {
		if len(m.columnProviders[i]) > 0 && m.columnCursors[i] >= len(m.columnProviders[i]) {
			m.columnCursors[i] = len(m.columnProviders[i]) - 1
		}
	}
}

// syncColumnCursors 从单列模式切换到三列模式时同步光标
func (m *Model) syncColumnCursors() {
	m.refreshAllColumns()

	// 设置 columnCursor 为当前应用对应的列
	switch m.currentApp {
	case "claude":
		m.columnCursor = 0
	case "codex":
		m.columnCursor = 1
	case "gemini":
		m.columnCursor = 2
	}

	// 同步当前应用的光标位置到对应列
	m.columnCursors[m.columnCursor] = m.cursor
}

// saveViewModePreference 静默保存视图模式偏好
func (m *Model) saveViewModePreference() {
	_ = m.manager.SetViewMode(m.viewMode)
}

// refreshTemplates 刷新模板列表
func (m *Model) refreshTemplates() {
	if m.templateManager == nil {
		return
	}
	allTemplates := m.templateManager.ListTemplates(m.currentTemplateCategory())
	m.templates = allTemplates
}

// refreshMcpServers 刷新 MCP 服务器列表
func (m *Model) refreshMcpServers() {
	m.mcpServers = m.manager.ListMcpServers()
}

// currentTemplateCategory 获取当前应用对应的模板分类
func (m Model) currentTemplateCategory() string {
	switch m.currentApp {
	case "codex":
		return template.CategoryCodexMd
	case "gemini":
		return "" // Gemini 暂不支持模板功能
	default:
		return template.CategoryClaudeMd
	}
}

// templateCategoryDisplay 获取模板分类的显示名称
func (m Model) templateCategoryDisplay(category string) string {
	switch category {
	case template.CategoryCodexMd:
		return "Codex 指南 (CODEX.md)"
	case template.CategoryClaudeMd:
		return "Claude 指南 (CLAUDE.md)"
	default:
		return category
	}
}

// syncModTime 更新配置文件修改时间缓存
func (m *Model) syncModTime() {
	if info, err := os.Stat(m.configPath); err == nil {
		m.lastModTime = info.ModTime()
	}
}
