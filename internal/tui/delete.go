package tui

import (
	"fmt"
	"strings"

	"github.com/YangQing-Lin/cc-switch-cli/internal/i18n"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// handleDeleteKeys 处理删除确认模式的键盘事件
func (m Model) handleDeleteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		err := m.manager.DeleteProviderForApp(m.currentApp, m.deleteName)
		if err != nil {
			m.err = err
			m.message = ""
		} else {
			m.message = i18n.T("success.provider_deleted")
			m.err = nil
			m.refreshProviders()
			if m.cursor >= len(m.providers) && m.cursor > 0 {
				m.cursor--
			}
			// 三列模式下也刷新所有列并检查光标越界
			if m.viewMode == "multi" {
				m.refreshAllColumns()
			}
			m.syncModTime()
		}
		m.mode = "list"
		m.deleteName = ""
	case "n", "N", "esc":
		m.mode = "list"
		m.deleteName = ""
	}
	return m, nil
}

// viewDelete 渲染删除确认视图
func (m Model) viewDelete() string {
	var s strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#007AFF")).
		Padding(0, 1).
		Render(fmt.Sprintf("确认删除 (v%s)", m.getVersion()))
	s.WriteString(title + "\n\n")

	msg := fmt.Sprintf("确定要删除配置 '%s' 吗？", m.deleteName)
	s.WriteString(msg + "\n\n")

	warning := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF3B30")).
		Bold(true).
		Render("⚠ 此操作无法撤销！")
	s.WriteString(warning + "\n\n")

	// Buttons
	deleteStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#FF3B30")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 2).
		Bold(true)
	cancelStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#8E8E93")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 2)

	s.WriteString(deleteStyle.Render("删除 (Y)") + " ")
	s.WriteString(cancelStyle.Render("取消 (N)"))

	return s.String()
}
