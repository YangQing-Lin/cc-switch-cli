package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) handlePreviewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.templateMode = "list"
		m.selectedTemplate = nil
		m.previewScrollOffset = 0
		m.message = ""
		m.err = nil
	case "up", "k":
		if m.previewScrollOffset > 0 {
			m.previewScrollOffset--
		}
	case "down", "j":
		lines := strings.Split(m.selectedTemplate.Content, "\n")
		maxOffset := len(lines) - 20
		if maxOffset < 0 {
			maxOffset = 0
		}
		if m.previewScrollOffset < maxOffset {
			m.previewScrollOffset++
		}
	case "pgup":
		m.previewScrollOffset -= 10
		if m.previewScrollOffset < 0 {
			m.previewScrollOffset = 0
		}
	case "pgdown":
		lines := strings.Split(m.selectedTemplate.Content, "\n")
		maxOffset := len(lines) - 20
		if maxOffset < 0 {
			maxOffset = 0
		}
		m.previewScrollOffset += 10
		if m.previewScrollOffset > maxOffset {
			m.previewScrollOffset = maxOffset
		}
	}
	return m, nil
}

func (m Model) viewTemplatePreview() string {
	var s strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#007AFF")).
		Padding(0, 1).
		Render(fmt.Sprintf("预览: %s (v%s)", m.selectedTemplate.Name, m.getVersion()))
	s.WriteString(title + "\n")

	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8E8E93"))
	s.WriteString(infoStyle.Render(fmt.Sprintf("  ID: %s\n", m.selectedTemplate.ID)))
	s.WriteString(infoStyle.Render(fmt.Sprintf("  类型: %s\n", m.templateCategoryDisplay(m.selectedTemplate.Category))))

	typeLabel := "用户自定义"
	if m.selectedTemplate.IsBuiltin {
		typeLabel = "预定义模板"
	}
	s.WriteString(infoStyle.Render(fmt.Sprintf("  %s\n\n", typeLabel)))

	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8E8E93"))
	s.WriteString(separatorStyle.Render(strings.Repeat("─", 60)) + "\n\n")

	lines := strings.Split(m.selectedTemplate.Content, "\n")
	visibleLines := lines
	if m.previewScrollOffset < len(lines) {
		endLine := m.previewScrollOffset + 25
		if endLine > len(lines) {
			endLine = len(lines)
		}
		visibleLines = lines[m.previewScrollOffset:endLine]
	}

	for _, line := range visibleLines {
		s.WriteString(line + "\n")
	}

	if len(lines) > 25 && m.previewScrollOffset+25 < len(lines) {
		s.WriteString("\n" + infoStyle.Render("(向下滚动查看更多...)") + "\n")
	}

	s.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8E8E93"))
	s.WriteString(helpStyle.Render("↑/↓: 滚动 • PgUp/PgDn: 翻页 • ESC: 返回"))

	return s.String()
}
