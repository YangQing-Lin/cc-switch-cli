package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// handleAppSelectKeys 处理应用选择模式的键盘事件
func (m Model) handleAppSelectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.mode = "list"
	case "up", "k":
		if m.appCursor > 0 {
			m.appCursor--
		} else {
			m.appCursor = 2 // 循环到 Gemini
		}
	case "down", "j":
		if m.appCursor < 2 { // 现在有 3 个选项：Claude(0), Codex(1), Gemini(2)
			m.appCursor++
		} else {
			m.appCursor = 0 // 循环到 Claude
		}
	case "enter":
		switch m.appCursor {
		case 0:
			m.currentApp = "claude"
		case 1:
			m.currentApp = "codex"
		case 2:
			m.currentApp = "gemini"
		}
		m.cursor = 0
		m.refreshProviders()
		m.mode = "list"
		m.message = fmt.Sprintf("切换到 %s", m.currentApp)
	}
	return m, nil
}

// viewAppSelect 渲染应用选择视图
func (m Model) viewAppSelect() string {
	var s strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#007AFF")).
		Padding(0, 1).
		Render(fmt.Sprintf("选择应用 (v%s)", m.getVersion()))
	s.WriteString(title + "\n\n")

	apps := []string{"Claude Code", "Codex CLI", "Gemini CLI"}
	for i, app := range apps {
		marker := "○"
		style := lipgloss.NewStyle().Padding(0, 1)

		if i == m.appCursor {
			marker = "●"
			style = style.
				Background(lipgloss.Color("#007AFF")).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)
		}

		line := fmt.Sprintf("%s %s", marker, style.Render(app))
		s.WriteString(line + "\n")
	}

	s.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8E8E93"))
	s.WriteString(helpStyle.Render("↑/↓: 选择 • Enter: 确认 • ESC: 取消"))

	return s.String()
}
