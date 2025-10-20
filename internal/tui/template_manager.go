package tui

import (
	"fmt"
	"strings"

	"github.com/YangQing-Lin/cc-switch-cli/internal/template"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) handleTemplateListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.mode = "list"
		m.templateMode = ""
		m.message = ""
		m.err = nil
	case "up", "k":
		if m.templateCursor > 0 {
			m.templateCursor--
		}
	case "down", "j":
		if m.templateCursor < len(m.templates)-1 {
			m.templateCursor++
		}
	case "enter":
		if len(m.templates) > 0 {
			selectedTemplate := m.templates[m.templateCursor]
			m.selectedTemplate = &selectedTemplate
			m.templateMode = "apply_select_target"
			m.targetSelectCursor = 0
			m.message = ""
			m.err = nil
		}
	case "p":
		if len(m.templates) > 0 {
			selectedTemplate := m.templates[m.templateCursor]
			m.selectedTemplate = &selectedTemplate
			m.templateMode = "preview"
			m.previewScrollOffset = 0
			m.message = ""
			m.err = nil
		}
	case "s":
		m.templateMode = "save_select_source"
		m.sourceSelectCursor = 0
		m.message = ""
		m.err = nil
	case "d":
		if len(m.templates) > 0 {
			selectedTemplate := m.templates[m.templateCursor]
			if selectedTemplate.IsBuiltin {
				m.err = fmt.Errorf("✗ 预定义模板无法删除")
				m.message = ""
			} else {
				m.selectedTemplate = &selectedTemplate
				m.templateMode = "delete_confirm"
				m.message = ""
				m.err = nil
			}
		}
	case "r":
		m.refreshTemplates()
		m.message = "模板列表已刷新"
		m.err = nil
	}
	return m, nil
}

func (m Model) viewTemplateList() string {
	var s strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#007AFF")).
		Padding(0, 1).
		Render(fmt.Sprintf("模板管理 (v%s)", m.getVersion()))
	s.WriteString(title + "\n\n")

	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3B30")).Bold(true)
		s.WriteString(errStyle.Render(m.err.Error()) + "\n\n")
	} else if m.message != "" {
		msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#34C759")).Bold(true)
		s.WriteString(msgStyle.Render("✓ "+m.message) + "\n\n")
	}

	if len(m.templates) == 0 {
		s.WriteString("暂无模板\n\n")
	} else {
		builtinTemplates := []template.Template{}
		userTemplates := []template.Template{}

		for _, t := range m.templates {
			if t.IsBuiltin {
				builtinTemplates = append(builtinTemplates, t)
			} else {
				userTemplates = append(userTemplates, t)
			}
		}

		currentIndex := 0

		if len(builtinTemplates) > 0 {
			sectionTitle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#8E8E93")).
				Render("预定义模板:")
			s.WriteString(sectionTitle + "\n")

			for _, t := range builtinTemplates {
				isCursor := currentIndex == m.templateCursor

				marker := " "
				if isCursor {
					marker = "→"
				}

				idStr := t.ID
				if len(idStr) > 15 {
					idStr = idStr[:12] + "..."
				}

				nameText := fmt.Sprintf("%-20s (%s)", t.Name, idStr)

				if isCursor {
					nameText = lipgloss.NewStyle().
						Background(lipgloss.Color("#007AFF")).
						Foreground(lipgloss.Color("#FFFFFF")).
						Bold(true).
						Padding(0, 1).
						Render(nameText)
				} else {
					nameText = lipgloss.NewStyle().
						Padding(0, 1).
						Render(nameText)
				}

				line := fmt.Sprintf("%s %s", marker, nameText)
				s.WriteString(line + "\n")
				currentIndex++
			}

			s.WriteString("\n")
		}

		separatorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8E8E93"))
		s.WriteString(separatorStyle.Render(strings.Repeat("─", 40)) + "\n\n")

		sectionTitle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8E8E93")).
			Render("用户模板:")
		s.WriteString(sectionTitle + "\n")

		if len(userTemplates) == 0 {
			s.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color("#8E8E93")).
				Render("  (暂无用户模板)") + "\n")
		} else {
			for _, t := range userTemplates {
				isCursor := currentIndex == m.templateCursor

				marker := " "
				if isCursor {
					marker = "→"
				}

				idStr := t.ID
				if len(idStr) > 15 {
					idStr = idStr[:12] + "..."
				}

				nameText := fmt.Sprintf("%-20s (%s)", t.Name, idStr)

				if isCursor {
					nameText = lipgloss.NewStyle().
						Background(lipgloss.Color("#007AFF")).
						Foreground(lipgloss.Color("#FFFFFF")).
						Bold(true).
						Padding(0, 1).
						Render(nameText)
				} else {
					nameText = lipgloss.NewStyle().
						Padding(0, 1).
						Render(nameText)
				}

				line := fmt.Sprintf("%s %s", marker, nameText)
				s.WriteString(line + "\n")
				currentIndex++
			}
		}

		s.WriteString("\n")
		countText := fmt.Sprintf("(共 %d 个模板)", len(m.templates))
		s.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8E8E93")).
			Render(countText) + "\n")
	}

	s.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8E8E93"))
	helps := []string{
		"↑/↓: 选择",
		"Enter: 应用",
		"p: 预览",
		"s: 保存用户配置",
		"d: 删除",
		"r: 刷新",
		"ESC: 返回",
	}
	s.WriteString(helpStyle.Render(strings.Join(helps, " • ")))

	return s.String()
}
