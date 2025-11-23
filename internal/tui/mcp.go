package tui

import (
	"fmt"
	"strings"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-shellwords"
)

// MCP 列表视图
func (m Model) viewMcpList() string {
	var s strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#007AFF")).
		Padding(0, 1).
		Render(fmt.Sprintf("CC Switch CLI v%s - MCP 服务器管理", m.getVersion()))
	s.WriteString(title + "\n\n")

	// Status message
	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3B30")).Bold(true)
		s.WriteString(errStyle.Render("✗ "+m.err.Error()) + "\n\n")
	} else if m.message != "" {
		msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#34C759")).Bold(true)
		s.WriteString(msgStyle.Render("✓ "+m.message) + "\n\n")
	}

	// MCP 服务器列表
	if len(m.mcpServers) == 0 {
		s.WriteString("暂无 MCP 服务器，按 'a' 添加新服务器或 'p' 查看预设列表\n\n")
	} else {
		for i, server := range m.mcpServers {
			isCursor := i == m.mcpCursor

			// 渲染 marker
			marker := "○"
			if isCursor {
				marker = "●"
			}
			markerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#007AFF"))
			styledMarker := markerStyle.Render(marker)

			// 渲染服务器名称
			nameText := fmt.Sprintf("%s - %s", server.ID, server.Name)
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

			// 应用标签
			var appTags []string
			if server.Apps.Claude {
				appTags = append(appTags, "[C]")
			}
			if server.Apps.Codex {
				appTags = append(appTags, "[X]")
			}
			if server.Apps.Gemini {
				appTags = append(appTags, "[G]")
			}
			appsText := strings.Join(appTags, " ")
			if len(appTags) == 0 {
				appsText = "[ ]"
			}

			// 描述
			description := server.Description
			if description == "" {
				description = "无描述"
			}

			// 组合
			line := fmt.Sprintf("%s %s  %s", styledMarker, nameText, appsText)
			s.WriteString(line + "\n")
			s.WriteString(fmt.Sprintf("  %s\n", description))
			s.WriteString("\n")
		}
	}

	// Help
	s.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8E8E93"))
	helps := []string{
		"↑/↓: 选择",
		"Enter: 切换应用",
		"a: 添加",
		"e: 编辑",
		"d: 删除",
		"p: 预设列表",
		"s: 同步所有",
		"r: 刷新",
		"ESC: 返回",
	}
	s.WriteString(helpStyle.Render(strings.Join(helps, " • ")))

	return s.String()
}

// MCP 列表按键处理
func (m Model) handleMcpListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.mode = "list"
		m.message = ""
		m.err = nil
	case "up", "k":
		if len(m.mcpServers) > 0 {
			if m.mcpCursor > 0 {
				m.mcpCursor--
			} else {
				m.mcpCursor = len(m.mcpServers) - 1
			}
		}
	case "down", "j":
		if len(m.mcpServers) > 0 {
			if m.mcpCursor < len(m.mcpServers)-1 {
				m.mcpCursor++
			} else {
				m.mcpCursor = 0
			}
		}
	case "enter":
		if len(m.mcpServers) > 0 {
			// 进入应用多选模式
			m.selectedMcp = &m.mcpServers[m.mcpCursor]
			m.mcpAppsToggle = m.selectedMcp.Apps
			m.mcpAppsCursor = 0
			m.mcpMode = "apps_toggle"
			m.message = ""
			m.err = nil
		}
	case "a":
		// 添加 MCP 服务器
		m.mcpMode = "add"
		m.selectedMcp = nil
		m.initMcpForm(nil)
		return m, textinput.Blink
	case "e":
		if len(m.mcpServers) > 0 {
			// 编辑 MCP 服务器
			server := m.mcpServers[m.mcpCursor]
			m.mcpMode = "edit"
			m.selectedMcp = &server
			m.initMcpForm(&server)
			return m, textinput.Blink
		}
	case "d":
		if len(m.mcpServers) > 0 {
			// 删除 MCP 服务器
			m.selectedMcp = &m.mcpServers[m.mcpCursor]
			m.mcpMode = "delete"
			m.message = ""
			m.err = nil
		}
	case "p":
		// 预设列表
		m.mcpPresets = config.GetMcpPresets()
		m.mcpPresetCursor = 0
		m.mcpMode = "preset"
		m.message = ""
		m.err = nil
	case "s":
		// 同步所有 MCP 服务器
		if err := m.manager.SyncAllMcpServers(); err != nil {
			m.err = fmt.Errorf("同步失败: %w", err)
			m.message = ""
		} else {
			m.message = "✓ 所有 MCP 服务器已同步"
			m.err = nil
		}
	case "r":
		// 刷新列表
		m.refreshMcpServers()
		m.message = "列表已刷新"
		m.err = nil
	}
	return m, nil
}

// 初始化 MCP 表单
func (m *Model) initMcpForm(server *config.McpServer) {
	m.mcpInputs = make([]textinput.Model, 6)
	m.mcpFocusIndex = 0

	// ID
	m.mcpInputs[0] = textinput.New()
	m.mcpInputs[0].Placeholder = "server-id"
	m.mcpInputs[0].Focus()
	m.mcpInputs[0].PromptStyle = focusedStyle
	m.mcpInputs[0].TextStyle = focusedStyle
	m.mcpInputs[0].Width = 50

	// Name
	m.mcpInputs[1] = textinput.New()
	m.mcpInputs[1].Placeholder = "Server Name"
	m.mcpInputs[1].Width = 50

	// Command (stdio)
	m.mcpInputs[2] = textinput.New()
	m.mcpInputs[2].Placeholder = "uvx"
	m.mcpInputs[2].Width = 50

	// Args (stdio)
	m.mcpInputs[3] = textinput.New()
	m.mcpInputs[3].Placeholder = "mcp-server-fetch"
	m.mcpInputs[3].Width = 50

	// URL (http/sse)
	m.mcpInputs[4] = textinput.New()
	m.mcpInputs[4].Placeholder = "http://localhost:3000/mcp"
	m.mcpInputs[4].Width = 50

	// Description
	m.mcpInputs[5] = textinput.New()
	m.mcpInputs[5].Placeholder = "服务器描述（可选）"
	m.mcpInputs[5].Width = 50

	// 如果是编辑模式，填充现有数据
	if server != nil {
		m.mcpInputs[0].SetValue(server.ID)
		m.mcpInputs[0].Blur() // ID 不可编辑

		// 编辑模式下从第二个输入框开始
		m.mcpFocusIndex = 1
		m.mcpInputs[1].Focus()
		m.mcpInputs[1].PromptStyle = focusedStyle
		m.mcpInputs[1].TextStyle = focusedStyle
		m.mcpInputs[1].SetValue(server.Name)
		m.mcpInputs[5].SetValue(server.Description)

		// 根据 server 类型设置连接类型和字段
		if connType, ok := server.Server["type"].(string); ok {
			m.mcpConnType = connType
			switch connType {
			case "stdio":
				if cmd, ok := server.Server["command"].(string); ok {
					m.mcpInputs[2].SetValue(cmd)
				}
				if args, ok := server.Server["args"].([]interface{}); ok {
					var argsStr []string
					for _, arg := range args {
						if argStr, ok := arg.(string); ok {
							argsStr = append(argsStr, argStr)
						}
					}
					m.mcpInputs[3].SetValue(strings.Join(argsStr, " "))
				}
			case "http", "sse":
				if url, ok := server.Server["url"].(string); ok {
					m.mcpInputs[4].SetValue(url)
				}
			}
		}
	} else {
		// 默认 stdio 类型
		m.mcpConnType = "stdio"
	}
}

// MCP 表单视图
func (m Model) viewMcpForm() string {
	var s strings.Builder

	mode := "添加"
	if m.mcpMode == "edit" {
		mode = "编辑"
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#007AFF")).
		Padding(0, 1).
		Render(fmt.Sprintf("%s MCP 服务器", mode))
	s.WriteString(title + "\n\n")

	// Status message
	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3B30")).Bold(true)
		s.WriteString(errStyle.Render("✗ "+m.err.Error()) + "\n\n")
	}

	// ID
	s.WriteString("服务器 ID:\n")
	s.WriteString(m.mcpInputs[0].View() + "\n\n")

	// Name
	s.WriteString("服务器名称:\n")
	s.WriteString(m.mcpInputs[1].View() + "\n\n")

	// 连接类型选择
	s.WriteString("连接类型:\n")
	connTypes := []string{"stdio", "http", "sse"}
	for i, ct := range connTypes {
		marker := "○"
		style := lipgloss.NewStyle()
		if ct == m.mcpConnType {
			marker = "●"
			style = style.Foreground(lipgloss.Color("#007AFF")).Bold(true)
		}
		if i > 0 {
			s.WriteString("  ")
		}
		s.WriteString(style.Render(fmt.Sprintf("%s %s", marker, ct)))
	}
	s.WriteString("\n\n")

	// 根据连接类型显示不同字段
	switch m.mcpConnType {
	case "stdio":
		s.WriteString("命令 (stdio):\n")
		s.WriteString(m.mcpInputs[2].View() + "\n\n")
		s.WriteString("参数 (用空格分隔):\n")
		s.WriteString(m.mcpInputs[3].View() + "\n\n")
	case "http", "sse":
		s.WriteString("URL:\n")
		s.WriteString(m.mcpInputs[4].View() + "\n\n")
	}

	// Description
	s.WriteString("描述 (可选):\n")
	s.WriteString(m.mcpInputs[5].View() + "\n\n")

	// Help
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8E8E93"))
	helps := []string{
		"Tab: 下一个",
		"Shift+Tab: 上一个",
		"Ctrl+T: 切换连接类型",
		"Enter/Ctrl+S: 保存",
		"ESC: 取消",
	}
	s.WriteString(helpStyle.Render(strings.Join(helps, " • ")))

	return s.String()
}

// MCP 表单按键处理
func (m Model) handleMcpFormKeys(msg tea.KeyMsg) (bool, tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+s", "enter":
		// 保存
		if err := m.saveMcpForm(); err != nil {
			m.err = err
			m.message = ""
			return true, m, nil
		}

		// 保存成功
		m.refreshMcpServers()
		m.syncModTime()

		// 先捕获操作类型
		action := m.mcpMode
		verb := map[string]string{"add": "添加", "edit": "更新"}[action]

		m.mcpMode = "list"
		m.message = fmt.Sprintf("✓ MCP 服务器已%s", verb)
		m.err = nil
		return true, m, nil

	case "esc":
		m.mcpMode = "list"
		m.message = ""
		m.err = nil
		return true, m, nil

	case "ctrl+t":
		// 切换连接类型
		switch m.mcpConnType {
		case "stdio":
			m.mcpConnType = "http"
		case "http":
			m.mcpConnType = "sse"
		case "sse":
			m.mcpConnType = "stdio"
		}
		return true, m, nil
	}

	return false, m, nil
}

// 保存 MCP 表单
func (m *Model) saveMcpForm() error {
	// 构建服务器配置
	server := config.McpServer{
		ID:          strings.TrimSpace(m.mcpInputs[0].Value()),
		Name:        strings.TrimSpace(m.mcpInputs[1].Value()),
		Description: strings.TrimSpace(m.mcpInputs[5].Value()),
		Server:      make(map[string]interface{}),
	}

	// 设置连接类型
	server.Server["type"] = m.mcpConnType

	// 根据连接类型设置字段
	switch m.mcpConnType {
	case "stdio":
		command := strings.TrimSpace(m.mcpInputs[2].Value())
		argsStr := strings.TrimSpace(m.mcpInputs[3].Value())
		server.Server["command"] = command
		if argsStr != "" {
			// 使用 shellwords 解析参数，支持引号
			args, err := shellwords.Parse(argsStr)
			if err != nil {
				return fmt.Errorf("解析参数失败: %w", err)
			}
			interfaceArgs := make([]interface{}, len(args))
			for i, arg := range args {
				interfaceArgs[i] = arg
			}
			server.Server["args"] = interfaceArgs
		}
	case "http", "sse":
		url := strings.TrimSpace(m.mcpInputs[4].Value())
		server.Server["url"] = url
	}

	// 保留原有的应用状态（如果是编辑模式）
	if m.mcpMode == "edit" && m.selectedMcp != nil {
		server.Apps = m.selectedMcp.Apps
	}

	// 添加或更新
	if m.mcpMode == "add" {
		if err := m.manager.AddMcpServer(server); err != nil {
			return err
		}
	} else {
		if err := m.manager.UpdateMcpServer(server); err != nil {
			return err
		}
	}

	// 保存配置
	if err := m.manager.Save(); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}

	// 同步到对应的应用
	if err := m.manager.SyncMcpServer(server.ID); err != nil {
		return fmt.Errorf("同步失败: %w", err)
	}

	return nil
}

// 更新 MCP 输入框
func (m Model) updateMcpInputs(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg.String() {
	case "tab", "down":
		m.mcpFocusIndex++
		if m.mcpFocusIndex >= len(m.mcpInputs) {
			// 编辑模式下跳过ID输入框
			if m.mcpMode == "edit" {
				m.mcpFocusIndex = 1
			} else {
				m.mcpFocusIndex = 0
			}
		}
		// 编辑模式下跳过ID输入框
		if m.mcpMode == "edit" && m.mcpFocusIndex == 0 {
			m.mcpFocusIndex = 1
		}
	case "shift+tab", "up":
		m.mcpFocusIndex--
		if m.mcpFocusIndex < 0 {
			m.mcpFocusIndex = len(m.mcpInputs) - 1
		}
		// 编辑模式下跳过ID输入框
		if m.mcpMode == "edit" && m.mcpFocusIndex == 0 {
			m.mcpFocusIndex = len(m.mcpInputs) - 1
		}
	}

	// 更新焦点
	for i := range m.mcpInputs {
		if i == m.mcpFocusIndex {
			cmds = append(cmds, m.mcpInputs[i].Focus())
			m.mcpInputs[i].PromptStyle = focusedStyle
			m.mcpInputs[i].TextStyle = focusedStyle
		} else {
			m.mcpInputs[i].Blur()
			m.mcpInputs[i].PromptStyle = noStyle
			m.mcpInputs[i].TextStyle = noStyle
		}
	}

	// 更新当前输入框
	var cmd tea.Cmd
	m.mcpInputs[m.mcpFocusIndex], cmd = m.mcpInputs[m.mcpFocusIndex].Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// MCP 删除确认视图
func (m Model) viewMcpDelete() string {
	var s strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#007AFF")).
		Padding(0, 1).
		Render("确认删除 MCP 服务器")
	s.WriteString(title + "\n\n")

	if m.selectedMcp != nil {
		msg := fmt.Sprintf("确定要删除 MCP 服务器 '%s' 吗？", m.selectedMcp.Name)
		s.WriteString(msg + "\n\n")

		warning := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF3B30")).
			Bold(true).
			Render("⚠ 此操作将从所有应用的配置中移除该服务器！")
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
	}

	return s.String()
}

// MCP 删除按键处理
func (m Model) handleMcpDeleteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.selectedMcp != nil {
			// 删除服务器
			if err := m.manager.DeleteMcpServer(m.selectedMcp.ID); err != nil {
				m.err = fmt.Errorf("删除失败: %w", err)
				m.message = ""
			} else {
				// 保存配置
				if err := m.manager.Save(); err != nil {
					m.err = fmt.Errorf("保存配置失败: %w", err)
					m.message = ""
				} else {
					// 从所有应用移除
					var syncErrs []error
					if err := m.manager.RemoveMcpFromClaude(m.selectedMcp.ID); err != nil {
						syncErrs = append(syncErrs, fmt.Errorf("Claude: %w", err))
					}
					if err := m.manager.RemoveMcpFromCodex(m.selectedMcp.ID); err != nil {
						syncErrs = append(syncErrs, fmt.Errorf("Codex: %w", err))
					}
					if err := m.manager.RemoveMcpFromGemini(m.selectedMcp.ID); err != nil {
						syncErrs = append(syncErrs, fmt.Errorf("Gemini: %w", err))
					}

					if len(syncErrs) > 0 {
						// 显示警告但仍标记为成功删除
						m.message = fmt.Sprintf("⚠ MCP 服务器已删除，但部分同步失败: %v", syncErrs)
						m.err = nil
					} else {
						m.message = "✓ MCP 服务器已删除"
						m.err = nil
					}

					m.refreshMcpServers()
					if m.mcpCursor >= len(m.mcpServers) && m.mcpCursor > 0 {
						m.mcpCursor--
					}
					m.syncModTime()
				}
			}
		}
		m.mcpMode = "list"
		m.selectedMcp = nil
	case "n", "N", "esc":
		m.mcpMode = "list"
		m.selectedMcp = nil
		m.message = ""
		m.err = nil
	}
	return m, nil
}

// MCP 应用多选视图
func (m Model) viewMcpAppsToggle() string {
	var s strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#007AFF")).
		Padding(0, 1).
		Render("选择应用")
	s.WriteString(title + "\n\n")

	if m.selectedMcp != nil {
		s.WriteString(fmt.Sprintf("为 MCP 服务器 '%s' 选择要启用的应用：\n\n", m.selectedMcp.Name))

		apps := []struct {
			name    string
			enabled *bool
		}{
			{"Claude Code", &m.mcpAppsToggle.Claude},
			{"Codex CLI", &m.mcpAppsToggle.Codex},
			{"Gemini CLI", &m.mcpAppsToggle.Gemini},
		}

		for i, app := range apps {
			isCursor := i == m.mcpAppsCursor

			marker := "○"
			checkbox := "[ ]"
			if *app.enabled {
				checkbox = "[✓]"
			}

			style := lipgloss.NewStyle()
			if isCursor {
				marker = "●"
				style = style.
					Background(lipgloss.Color("#007AFF")).
					Foreground(lipgloss.Color("#FFFFFF")).
					Bold(true).
					Padding(0, 1)
			} else {
				style = style.Padding(0, 1)
			}

			line := fmt.Sprintf("%s %s %s", marker, checkbox, style.Render(app.name))
			s.WriteString(line + "\n")
		}

		s.WriteString("\n")
		helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8E8E93"))
		s.WriteString(helpStyle.Render("↑/↓: 选择 • Space: 切换 • Enter: 保存 • ESC: 取消"))
	}

	return s.String()
}

// MCP 应用多选按键处理
func (m Model) handleMcpAppsToggleKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.mcpMode = "list"
		m.selectedMcp = nil
		m.message = ""
		m.err = nil
	case "up", "k":
		if m.mcpAppsCursor > 0 {
			m.mcpAppsCursor--
		} else {
			m.mcpAppsCursor = 2
		}
	case "down", "j":
		if m.mcpAppsCursor < 2 {
			m.mcpAppsCursor++
		} else {
			m.mcpAppsCursor = 0
		}
	case " ":
		// 切换当前选项
		switch m.mcpAppsCursor {
		case 0:
			m.mcpAppsToggle.Claude = !m.mcpAppsToggle.Claude
		case 1:
			m.mcpAppsToggle.Codex = !m.mcpAppsToggle.Codex
		case 2:
			m.mcpAppsToggle.Gemini = !m.mcpAppsToggle.Gemini
		}
	case "enter":
		if m.selectedMcp != nil {
			// 检测是否为新服务器
			existingServer, err := m.manager.GetMcpServer(m.selectedMcp.ID)
			if err != nil {
				// 服务器不存在,使用预设配置添加
				m.selectedMcp.Apps = m.mcpAppsToggle
				if err := m.manager.AddMcpServer(*m.selectedMcp); err != nil {
					m.err = fmt.Errorf("添加失败: %w", err)
					m.message = ""
					return m, nil
				}
			} else {
				// 服务器已存在,只更新Apps字段,保持其他配置不变
				existingServer.Apps = m.mcpAppsToggle
				if err := m.manager.UpdateMcpServer(*existingServer); err != nil {
					m.err = fmt.Errorf("更新失败: %w", err)
					m.message = ""
					return m, nil
				}
			}

			// 保存配置
			if err := m.manager.Save(); err != nil {
				m.err = fmt.Errorf("保存配置失败: %w", err)
				m.message = ""
			} else {
				// 同步到对应应用
				if err := m.manager.SyncMcpServer(m.selectedMcp.ID); err != nil {
					m.err = fmt.Errorf("同步失败: %w", err)
					m.message = ""
				} else {
					m.message = "✓ 应用状态已更新并同步"
					m.err = nil
					m.refreshMcpServers()
					m.syncModTime()
				}
			}

			m.mcpMode = "list"
			m.selectedMcp = nil
		}
	}
	return m, nil
}

// MCP 预设列表视图
func (m Model) viewMcpPreset() string {
	var s strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#007AFF")).
		Padding(0, 1).
		Render("MCP 预设服务器")
	s.WriteString(title + "\n\n")

	for i, preset := range m.mcpPresets {
		isCursor := i == m.mcpPresetCursor

		marker := "○"
		if isCursor {
			marker = "●"
		}
		markerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#007AFF"))
		styledMarker := markerStyle.Render(marker)

		nameText := fmt.Sprintf("%s - %s", preset.ID, preset.Name)
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

		// 连接信息
		connInfo := ""
		if connType, ok := preset.Server["type"].(string); ok {
			switch connType {
			case "stdio":
				if cmd, ok := preset.Server["command"].(string); ok {
					connInfo = fmt.Sprintf("stdio: %s", cmd)
					if args, ok := preset.Server["args"].([]interface{}); ok {
						var argsStr []string
						for _, arg := range args {
							if argStr, ok := arg.(string); ok {
								argsStr = append(argsStr, argStr)
							}
						}
						connInfo += " " + strings.Join(argsStr, " ")
					}
				}
			case "http", "sse":
				if url, ok := preset.Server["url"].(string); ok {
					connInfo = fmt.Sprintf("%s: %s", connType, url)
				}
			}
		}

		line := fmt.Sprintf("%s %s", styledMarker, nameText)
		s.WriteString(line + "\n")
		s.WriteString(fmt.Sprintf("  %s\n", preset.Description))
		s.WriteString(fmt.Sprintf("  %s\n", connInfo))
		s.WriteString("\n")
	}

	// Help
	s.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8E8E93"))
	s.WriteString(helpStyle.Render("↑/↓: 选择 • Enter: 添加此预设 • ESC: 返回"))

	return s.String()
}

// MCP 预设列表按键处理
func (m Model) handleMcpPresetKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.mcpMode = "list"
		m.message = ""
		m.err = nil
	case "up", "k":
		if len(m.mcpPresets) > 0 {
			if m.mcpPresetCursor > 0 {
				m.mcpPresetCursor--
			} else {
				m.mcpPresetCursor = len(m.mcpPresets) - 1
			}
		}
	case "down", "j":
		if len(m.mcpPresets) > 0 {
			if m.mcpPresetCursor < len(m.mcpPresets)-1 {
				m.mcpPresetCursor++
			} else {
				m.mcpPresetCursor = 0
			}
		}
	case "enter":
		if len(m.mcpPresets) > 0 {
			// 选择预设，进入应用选择
			preset := m.mcpPresets[m.mcpPresetCursor]
			m.selectedMcp = &preset
			m.mcpAppsToggle = config.McpApps{Claude: false, Codex: false, Gemini: false}
			m.mcpAppsCursor = 0
			m.mcpMode = "apps_toggle"
			m.message = ""
			m.err = nil
		}
	}
	return m, nil
}
