package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/YangQing-Lin/cc-switch-cli/internal/template"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) handleTargetSelectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "esc" {
		m.templateMode = "list"
		m.selectedTemplate = nil
		m.message = ""
		m.err = nil
		return m, nil
	}

	if m.selectedTemplate == nil {
		m.err = fmt.Errorf("未选择模板")
		m.message = ""
		return m, nil
	}

	targets, err := template.GetTargetsForCategory(m.selectedTemplate.Category)
	if err != nil {
		m.err = fmt.Errorf("获取目标路径失败: %w", err)
		m.message = ""
		return m, nil
	}
	if len(targets) == 0 {
		m.err = fmt.Errorf("当前模板无可用目标路径")
		m.message = ""
		return m, nil
	}

	if m.targetSelectCursor >= len(targets) {
		m.targetSelectCursor = len(targets) - 1
	}

	switch key {
	case "up", "k":
		if m.targetSelectCursor > 0 {
			m.targetSelectCursor--
		}
	case "down", "j":
		if m.targetSelectCursor < len(targets)-1 {
			m.targetSelectCursor++
		}
	case "enter":
		selectedTarget := targets[m.targetSelectCursor]
		m.selectedTargetPath = selectedTarget.Path

		diff, err := m.templateManager.GetDiff(m.selectedTemplate.ID, selectedTarget.Path)
		if err != nil {
			m.err = fmt.Errorf("生成 diff 失败: %w", err)
			m.message = ""
			return m, nil
		}

		m.diffContent = diff
		m.diffScrollOffset = 0
		m.templateMode = "apply_preview_diff"
		m.message = ""
		m.err = nil
	}
	return m, nil
}

func (m Model) handleDiffPreviewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "n", "N", "esc":
		m.templateMode = "apply_select_target"
		m.diffContent = ""
		m.diffScrollOffset = 0
		m.message = ""
		m.err = nil
	case "y", "Y":
		err := m.templateManager.ApplyTemplate(m.selectedTemplate.ID, m.selectedTargetPath)
		if err != nil {
			m.err = fmt.Errorf("应用模板失败: %w", err)
			m.message = ""
		} else {
			m.message = fmt.Sprintf("✓ 模板已成功应用到: %s", m.selectedTargetPath)
			m.err = nil
			m.templateMode = "list"
			m.selectedTemplate = nil
			m.selectedTargetPath = ""
			m.diffContent = ""
		}
	case "up", "k":
		if m.diffScrollOffset > 0 {
			m.diffScrollOffset--
		}
	case "down", "j":
		lines := strings.Split(m.diffContent, "\n")
		maxOffset := len(lines) - 10
		if maxOffset < 0 {
			maxOffset = 0
		}
		if m.diffScrollOffset < maxOffset {
			m.diffScrollOffset++
		}
	case "pgup":
		m.diffScrollOffset -= 10
		if m.diffScrollOffset < 0 {
			m.diffScrollOffset = 0
		}
	case "pgdown":
		lines := strings.Split(m.diffContent, "\n")
		maxOffset := len(lines) - 10
		if maxOffset < 0 {
			maxOffset = 0
		}
		m.diffScrollOffset += 10
		if m.diffScrollOffset > maxOffset {
			m.diffScrollOffset = maxOffset
		}
	}
	return m, nil
}

func (m Model) viewTargetSelect() string {
	var s strings.Builder

	category := template.CategoryClaudeMd
	if m.selectedTemplate != nil {
		category = m.selectedTemplate.Category
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#007AFF")).
		Padding(0, 1).
		Render(fmt.Sprintf("应用模板: %s (v%s)", m.selectedTemplate.Name, m.getVersion()))
	s.WriteString(title + "\n\n")

	targetLabel := "CLAUDE.md"
	if category == template.CategoryCodexMd {
		targetLabel = "CODEX.md"
	}
	s.WriteString(fmt.Sprintf("选择目标路径（%s）:\n\n", targetLabel))

	targets, err := template.GetTargetsForCategory(category)
	if err != nil {
		s.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF3B30")).
			Render("✗ 获取目标路径失败") + "\n")
		return s.String()
	}

	if len(targets) == 0 {
		s.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF3B30")).
			Render("✗ 没有可用的目标路径") + "\n")
		return s.String()
	}

	for i, target := range targets {
		isCursor := i == m.targetSelectCursor

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

		line := fmt.Sprintf("%s %s %s\n", arrow, styledMarker, nameText)
		s.WriteString(line)
		s.WriteString(fmt.Sprintf("         %s\n\n", descText))
	}

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8E8E93"))
	s.WriteString(helpStyle.Render("↑/↓: 选择 • Enter: 下一步 • ESC: 取消"))

	return s.String()
}

func (m Model) viewDiffPreview() string {
	var s strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#007AFF")).
		Padding(0, 1).
		Render(fmt.Sprintf("Diff 预览 (v%s)", m.getVersion()))
	s.WriteString(title + "\n\n")

	s.WriteString(fmt.Sprintf("模板: %s\n", m.selectedTemplate.Name))
	s.WriteString(fmt.Sprintf("目标: %s\n\n", m.selectedTargetPath))

	_, err := os.Stat(m.selectedTargetPath)
	fileExists := err == nil

	if !fileExists {
		s.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#34C759")).
			Render("✓ 目标文件不存在，将创建新文件") + "\n\n")

		previewLines := strings.Split(m.selectedTemplate.Content, "\n")
		maxLines := 20
		if len(previewLines) > maxLines {
			previewLines = previewLines[:maxLines]
		}
		s.WriteString("模板内容预览:\n")
		s.WriteString(strings.Join(previewLines, "\n") + "\n")
		if len(strings.Split(m.selectedTemplate.Content, "\n")) > maxLines {
			s.WriteString("\n(向下滚动查看更多...)\n")
		}

		s.WriteString("\n")
	} else {
		if m.diffContent == "" || m.diffContent == "未发现差异" {
			s.WriteString("(无差异)\n\n")
		} else {
			lines := strings.Split(m.diffContent, "\n")
			visibleLines := lines
			if m.diffScrollOffset < len(lines) {
				endLine := m.diffScrollOffset + 20
				if endLine > len(lines) {
					endLine = len(lines)
				}
				visibleLines = lines[m.diffScrollOffset:endLine]
			}

			for _, line := range visibleLines {
				styledLine := line
				if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
					styledLine = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3B30")).Render(line)
				} else if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
					styledLine = lipgloss.NewStyle().Foreground(lipgloss.Color("#34C759")).Render(line)
				} else if strings.HasPrefix(line, "@@") {
					styledLine = lipgloss.NewStyle().Foreground(lipgloss.Color("#00CED1")).Render(line)
				} else if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") {
					styledLine = lipgloss.NewStyle().Bold(true).Render(line)
				}
				s.WriteString(styledLine + "\n")
			}

			if len(lines) > 20 {
				s.WriteString("\n(向下滚动查看更多...)\n")
			}
		}

		warningStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF3B30")).
			Bold(true)
		s.WriteString("\n" + warningStyle.Render("⚠ 即将覆盖目标文件") + "\n\n")
	}

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8E8E93"))
	s.WriteString(helpStyle.Render("y: 应用 • n: 取消 • ↑/↓: 滚动 • PgUp/PgDn: 翻页"))

	return s.String()
}
