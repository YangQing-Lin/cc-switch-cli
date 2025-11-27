package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/YangQing-Lin/cc-switch-cli/internal/backup"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// handleBackupListKeys 处理备份列表模式的键盘事件
func (m Model) handleBackupListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.mode = "list"
		m.message = ""
		m.err = nil
	case "up", "k":
		if len(m.backupList) > 0 {
			if m.backupCursor > 0 {
				m.backupCursor--
			} else {
				m.backupCursor = len(m.backupList) - 1
			}
		}
	case "down", "j":
		if len(m.backupList) > 0 {
			if m.backupCursor < len(m.backupList)-1 {
				m.backupCursor++
			} else {
				m.backupCursor = 0
			}
		}
	case "enter":
		if len(m.backupList) > 0 {
			selectedBackup := m.backupList[m.backupCursor]

			// 恢复备份
			err := backup.RestoreBackup(m.configPath, selectedBackup.Path)
			if err != nil {
				m.err = fmt.Errorf("恢复备份失败: %w", err)
				m.message = ""
			} else {
				m.message = fmt.Sprintf("配置已从备份恢复: %s", filepath.Base(selectedBackup.Path))
				m.err = nil

				// 重新加载配置
				if err := m.manager.Load(); err != nil {
					m.err = fmt.Errorf("重新加载配置失败: %w", err)
					m.message = ""
				} else {
					m.refreshProviders()
					m.syncModTime()
				}

				// 返回列表模式
				m.mode = "list"
			}
		}
	}
	return m, nil
}

// viewBackupList 渲染备份列表视图
func (m Model) viewBackupList() string {
	var s strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#007AFF")).
		Padding(0, 1).
		Render(fmt.Sprintf("备份列表 (v%s)", m.getVersion()))
	s.WriteString(title + "\n\n")

	// Status message
	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3B30")).Bold(true)
		s.WriteString(errStyle.Render("✗ "+m.err.Error()) + "\n\n")
	} else if m.message != "" {
		msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#34C759")).Bold(true)
		s.WriteString(msgStyle.Render("✓ "+m.message) + "\n\n")
	}

	// Backup list
	if len(m.backupList) == 0 {
		s.WriteString("暂无备份文件\n\n")
	} else {
		for i, b := range m.backupList {
			isCursor := i == m.backupCursor

			// 备份文件名
			filename := filepath.Base(b.Path)

			displayName := filename

			if isCursor {
				displayName = lipgloss.NewStyle().
					Background(lipgloss.Color("#007AFF")).
					Foreground(lipgloss.Color("#FFFFFF")).
					Bold(true).
					Padding(0, 1).
					Render(displayName)
			} else {
				displayName = lipgloss.NewStyle().
					Padding(0, 1).
					Render(displayName)
			}

			// 显示时间和大小
			timeStr := b.Timestamp.Format("2006-01-02 15:04:05")
			sizeStr := fmt.Sprintf("%.2f KB", float64(b.Size)/1024)
			info := fmt.Sprintf("  %s | %s", timeStr, sizeStr)

			s.WriteString(fmt.Sprintf("%d. %s\n%s\n", i+1, displayName, info))
		}
	}

	// Help
	s.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8E8E93"))
	s.WriteString(helpStyle.Render("↑/↓: 选择 • Enter: 恢复 • ESC: 返回"))

	return s.String()
}
