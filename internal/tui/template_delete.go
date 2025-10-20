package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) handleDeleteConfirmKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "n", "N", "esc":
		m.templateMode = "list"
		m.selectedTemplate = nil
		m.message = ""
		m.err = nil
	case "y", "Y":
		err := m.templateManager.DeleteTemplate(m.selectedTemplate.ID)
		if err != nil {
			m.err = fmt.Errorf("删除失败: %w", err)
			m.message = ""
		} else {
			m.message = fmt.Sprintf("✓ 模板已删除: %s", m.selectedTemplate.Name)
			m.err = nil
			m.refreshTemplates()

			if m.templateCursor >= len(m.templates) && m.templateCursor > 0 {
				m.templateCursor--
			}

			m.templateMode = "list"
			m.selectedTemplate = nil
		}
	}
	return m, nil
}

func (m Model) viewTemplateDelete() string {
	var s strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#007AFF")).
		Padding(0, 1).
		Render(fmt.Sprintf("确认删除 (v%s)", m.getVersion()))
	s.WriteString(title + "\n\n")

	s.WriteString("确定要删除模板吗？\n\n")

	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8E8E93"))
	s.WriteString(infoStyle.Render(fmt.Sprintf("  名称: %s\n", m.selectedTemplate.Name)))
	s.WriteString(infoStyle.Render(fmt.Sprintf("  ID: %s\n", m.selectedTemplate.ID)))
	s.WriteString(infoStyle.Render(fmt.Sprintf("  类型: %s\n\n", m.selectedTemplate.Category)))

	warningStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF3B30")).
		Bold(true)
	s.WriteString(warningStyle.Render("⚠ 此操作无法撤销！") + "\n\n")

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
