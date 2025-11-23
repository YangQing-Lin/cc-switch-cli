package tui

import "github.com/charmbracelet/lipgloss"

var (
	// 颜色定义
	primaryColor   = lipgloss.Color("#007AFF")
	successColor   = lipgloss.Color("#34C759")
	dangerColor    = lipgloss.Color("#FF3B30")
	warningColor   = lipgloss.Color("#FF9500")
	subtleColor    = lipgloss.Color("#8E8E93")
	borderColor    = lipgloss.Color("#E5E5EA")
	bgColor        = lipgloss.Color("#FFFFFF")
	selectedBg     = lipgloss.Color("#F2F2F7")
	textColor      = lipgloss.Color("#000000")
	mutedTextColor = lipgloss.Color("#6C6C70")

	// 标题样式
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Padding(0, 1)

	// 状态栏样式
	statusBarStyle = lipgloss.NewStyle().
			Foreground(mutedTextColor).
			Padding(0, 1)

	// 帮助文本样式
	helpStyle = lipgloss.NewStyle().
			Foreground(subtleColor).
			Padding(0, 1)

	// 选中项样式
	selectedItemStyle = lipgloss.NewStyle().
				Background(selectedBg).
				Foreground(primaryColor).
				Bold(true).
				Padding(0, 1)

	// 普通项样式
	normalItemStyle = lipgloss.NewStyle().
			Padding(0, 1)

	// 当前激活标记样式
	activeMarkerStyle = lipgloss.NewStyle().
				Foreground(successColor).
				Bold(true)

	// 普通标记样式
	inactiveMarkerStyle = lipgloss.NewStyle().
				Foreground(subtleColor)

	// 面板边框样式
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(1, 2)

	// 详情标题样式
	detailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(textColor)

	// 详情内容样式
	detailContentStyle = lipgloss.NewStyle().
				Foreground(mutedTextColor)

	// 成功消息样式
	successMessageStyle = lipgloss.NewStyle().
				Foreground(successColor).
				Bold(true)

	// 错误消息样式
	errorMessageStyle = lipgloss.NewStyle().
				Foreground(dangerColor).
				Bold(true)

	// 输入框样式
	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 1)

	// 输入标签样式
	inputLabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(textColor)

	// 按钮样式
	buttonStyle = lipgloss.NewStyle().
			Foreground(bgColor).
			Background(primaryColor).
			Padding(0, 2).
			Bold(true)

	// 取消按钮样式
	cancelButtonStyle = lipgloss.NewStyle().
				Foreground(bgColor).
				Background(subtleColor).
				Padding(0, 2)

	// 禁用按钮样式（浅灰色背景，深灰色文字）
	disabledButtonStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#636366")).
				Background(lipgloss.Color("#E5E5EA")).
				Padding(0, 2)

	// 对话框样式
	dialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2).
			Width(60)

	// 分类徽章样式
	badgeStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true)

	// 分类徽章颜色映射
	categoryColors = map[string]lipgloss.Color{
		"official":    successColor,
		"cn_official": lipgloss.Color("#5AC8FA"),
		"aggregator":  warningColor,
		"third_party": lipgloss.Color("#AF52DE"),
		"custom":      subtleColor,
	}

	// 输入框焦点样式
	focusedStyle = lipgloss.NewStyle().Foreground(primaryColor)
	noStyle      = lipgloss.NewStyle()
)

// GetCategoryBadge 根据分类返回带样式的徽章
func GetCategoryBadge(category string) string {
	var label string
	switch category {
	case "official":
		label = "官方"
	case "cn_official":
		label = "国产官方"
	case "aggregator":
		label = "聚合"
	case "third_party":
		label = "第三方"
	case "custom":
		label = "自定义"
	default:
		return ""
	}

	color, ok := categoryColors[category]
	if !ok {
		color = subtleColor
	}

	return badgeStyle.
		Foreground(color).
		Render(label)
}
