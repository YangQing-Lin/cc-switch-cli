package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/YangQing-Lin/cc-switch-cli/internal/template"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) handleSourceSelectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.templateMode = "list"
		m.message = ""
		m.err = nil
	case "up", "k":
		targets, _ := template.GetClaudeMdTargets()
		for {
			if m.sourceSelectCursor > 0 {
				m.sourceSelectCursor--
			} else {
				break
			}
			if m.sourceSelectCursor < len(targets) {
				_, err := os.Stat(targets[m.sourceSelectCursor].Path)
				if err == nil {
					break
				}
			}
		}
	case "down", "j":
		targets, _ := template.GetClaudeMdTargets()
		for {
			if m.sourceSelectCursor < 2 {
				m.sourceSelectCursor++
			} else {
				break
			}
			if m.sourceSelectCursor < len(targets) {
				_, err := os.Stat(targets[m.sourceSelectCursor].Path)
				if err == nil {
					break
				}
			}
		}
	case "enter":
		targets, err := template.GetClaudeMdTargets()
		if err != nil {
			m.err = fmt.Errorf("获取源路径失败: %w", err)
			m.message = ""
			return m, nil
		}

		selectedTarget := targets[m.sourceSelectCursor]
		_, err = os.Stat(selectedTarget.Path)
		if err != nil {
			m.err = fmt.Errorf("✗ 该文件不存在，无法保存")
			m.message = ""
			return m, nil
		}

		m.selectedSourcePath = selectedTarget.Path
		m.templateMode = "save_input_name"

		m.saveNameInput = textinput.New()
		m.saveNameInput.Placeholder = m.templateManager.GenerateDefaultTemplateName()
		m.saveNameInput.Focus()
		m.saveNameInput.CharLimit = 50
		m.saveNameInput.Width = 50

		m.message = ""
		m.err = nil
		return m, textinput.Blink
	}
	return m, nil
}

func (m Model) handleSaveNameKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.templateMode = "save_select_source"
		m.saveNameInput = textinput.Model{}
		m.message = ""
		m.err = nil
		return m, nil
	case "enter":
		name := m.saveNameInput.Value()
		if name == "" {
			name = m.templateManager.GenerateDefaultTemplateName()
		}

		templateID, err := m.templateManager.SaveAsTemplate(
			m.selectedSourcePath,
			name,
			template.CategoryClaudeMd,
		)
		if err != nil {
			m.err = fmt.Errorf("保存失败: %w", err)
			m.message = ""
		} else {
			m.message = fmt.Sprintf("✓ 模板已保存\n  ID: %s\n  名称: %s\n  类型: %s",
				templateID, name, template.CategoryClaudeMd)
			m.err = nil
			m.refreshTemplates()

			for i, t := range m.templates {
				if t.ID == templateID {
					m.templateCursor = i
					break
				}
			}

			m.templateMode = "list"
			m.selectedSourcePath = ""
			m.saveNameInput = textinput.Model{}
		}
		return m, nil
	default:
		var cmd tea.Cmd
		m.saveNameInput, cmd = m.saveNameInput.Update(msg)
		return m, cmd
	}
}

func (m Model) viewSourceSelect() string {
	var s strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#007AFF")).
		Padding(0, 1).
		Render("保存为模板")
	s.WriteString(title + "\n\n")

	s.WriteString("选择源文件:\n\n")

	targets, err := template.GetClaudeMdTargets()
	if err != nil {
		s.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF3B30")).
			Render("✗ 获取源路径失败") + "\n")
		return s.String()
	}

	for i, target := range targets {
		isCursor := i == m.sourceSelectCursor

		_, err := os.Stat(target.Path)
		exists := err == nil

		marker := "[✗]"
		markerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8E8E93"))
		if exists {
			marker = "[✓]"
			markerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#34C759"))
		}
		styledMarker := markerStyle.Render(marker)

		arrow := "   "
		if isCursor {
			arrow = " → "
		}

		nameText := target.Name
		descText := target.Description

		if !exists {
			nameText = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#8E8E93")).
				Padding(0, 1).
				Render(nameText)
		} else if isCursor {
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

		line := fmt.Sprintf("%s %s %s\n", arrow, styledMarker, nameText)
		s.WriteString(line)
		s.WriteString(fmt.Sprintf("         %s\n\n", descText))
	}

	s.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8E8E93")).
		Render("只能选择已存在的文件") + "\n\n")

	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3B30")).Bold(true)
		s.WriteString(errStyle.Render(m.err.Error()) + "\n\n")
	}

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8E8E93"))
	s.WriteString(helpStyle.Render("↑/↓: 选择 • Enter: 下一步 • ESC: 取消"))

	return s.String()
}

func (m Model) viewSaveNameInput() string {
	var s strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#007AFF")).
		Padding(0, 1).
		Render("保存为模板")
	s.WriteString(title + "\n\n")

	s.WriteString(fmt.Sprintf("源文件: %s\n\n", m.selectedSourcePath))

	s.WriteString(lipgloss.NewStyle().Bold(true).Render("模板名称:") + "\n")

	inputBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#007AFF")).
		Render(m.saveNameInput.View())
	s.WriteString(inputBox + "\n\n")

	s.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8E8E93")).
		Render(fmt.Sprintf("留空将使用默认名称: \"%s\"", m.templateManager.GenerateDefaultTemplateName())) + "\n\n")

	if m.message != "" {
		msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#34C759")).Bold(true)
		s.WriteString(msgStyle.Render(m.message) + "\n\n")
	}

	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3B30")).Bold(true)
		s.WriteString(errStyle.Render("✗ "+m.err.Error()) + "\n\n")
	}

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8E8E93"))
	s.WriteString(helpStyle.Render("Enter: 保存 • ESC: 取消"))

	return s.String()
}
