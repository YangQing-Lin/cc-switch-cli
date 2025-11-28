package tui

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/YangQing-Lin/cc-switch-cli/internal/backup"
	"github.com/YangQing-Lin/cc-switch-cli/internal/i18n"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// handleListKeys 处理列表模式的键盘事件
func (m Model) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// 三列模式委托给 handleMultiColumnKeys
	if m.viewMode == "multi" {
		return m.handleMultiColumnKeys(msg)
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "v":
		// 切换到三列模式
		m.viewMode = "multi"
		m.syncColumnCursors()
		m.saveViewModePreference()
		m.message = "切换到三列视图"
		m.err = nil
		return m, nil
	case "up", "k":
		if len(m.providers) > 0 {
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(m.providers) - 1
			}
		}
	case "down", "j":
		if len(m.providers) > 0 {
			if m.cursor < len(m.providers)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}
		}
	case "=":
		if len(m.providers) == 0 {
			break
		}
		if m.cursor == 0 {
			m.message = "已在顶部，无法继续上调"
			m.err = nil
			break
		}
		provider := m.providers[m.cursor]
		if err := m.manager.MoveProviderForApp(m.currentApp, provider.ID, -1); err != nil {
			m.err = err
			m.message = ""
			break
		}
		targetID := provider.ID
		m.refreshProviders()
		m.syncModTime()
		m.err = nil
		m.message = "↑ 顺序已上调: " + provider.Name
		newIndex := m.cursor - 1
		for idx, p := range m.providers {
			if p.ID == targetID {
				newIndex = idx
				break
			}
		}
		if newIndex < 0 {
			newIndex = 0
		}
		m.cursor = newIndex
	case "-":
		if len(m.providers) == 0 {
			break
		}
		if m.cursor >= len(m.providers)-1 {
			m.message = "已在底部，无法继续下调"
			m.err = nil
			break
		}
		provider := m.providers[m.cursor]
		if err := m.manager.MoveProviderForApp(m.currentApp, provider.ID, 1); err != nil {
			m.err = err
			m.message = ""
			break
		}
		targetID := provider.ID
		m.refreshProviders()
		m.syncModTime()
		m.err = nil
		m.message = "↓ 顺序已下调: " + provider.Name
		newIndex := m.cursor + 1
		for idx, p := range m.providers {
			if p.ID == targetID {
				newIndex = idx
				break
			}
		}
		if newIndex > len(m.providers)-1 {
			newIndex = len(m.providers) - 1
		}
		m.cursor = newIndex
	case "enter":
		if len(m.providers) > 0 {
			provider := m.providers[m.cursor]
			current := m.manager.GetCurrentProviderForApp(m.currentApp)

			// 判断是切换还是覆盖
			isSwitch := current == nil || provider.ID != current.ID

			// 无论是否已激活，都执行切换操作（如果已激活则是覆盖）
			err := m.manager.SwitchProviderForApp(m.currentApp, provider.Name)
			if err != nil {
				m.err = err
				m.message = ""
			} else {
				if isSwitch {
					m.message = i18n.T("success.switched_to") + ": " + provider.Name
				} else {
					m.message = "✓ 已覆盖 live 配置: " + provider.Name
				}
				m.err = nil
				m.refreshProviders()
				m.syncModTime()
			}
		}
	case "a":
		m.mode = "add"
		m.editName = ""
		m.copyFromProvider = nil // 清空复制源
		m.initForm(nil)
		return m, textinput.Blink
	case "C":
		// 复制当前选中的配置并进入创建模式
		if len(m.providers) > 0 {
			provider := m.providers[m.cursor]
			m.mode = "add"
			m.editName = ""
			m.copyFromProvider = &provider // 设置复制源
			m.initForm(nil)
			return m, textinput.Blink
		}
	case "e":
		if len(m.providers) > 0 {
			provider := m.providers[m.cursor]
			m.mode = "edit"
			m.editName = provider.Name
			m.initForm(&provider)
			return m, textinput.Blink
		}
	case "d":
		if len(m.providers) > 0 {
			provider := m.providers[m.cursor]
			current := m.manager.GetCurrentProviderForApp(m.currentApp)
			if current != nil && provider.ID == current.ID {
				m.err = errors.New(i18n.T("error.cannot_delete_current"))
				m.message = ""
			} else {
				m.mode = "delete"
				m.deleteName = provider.Name
			}
		}
	case "r":
		m.refreshProviders()
		m.message = "列表已刷新"
		m.err = nil
		return m, tea.ClearScreen
	case "t":
		// Toggle between Claude, Codex and Gemini
		switch m.currentApp {
		case "claude":
			m.currentApp = "codex"
		case "codex":
			m.currentApp = "gemini"
		case "gemini":
			m.currentApp = "claude"
		default:
			m.currentApp = "claude"
		}
		m.cursor = 0
		m.refreshProviders()
		m.message = fmt.Sprintf("切换到 %s", m.currentApp)
		m.err = nil
	case "c":
		// Switch to Claude
		if m.currentApp != "claude" {
			m.currentApp = "claude"
			m.cursor = 0
			m.refreshProviders()
			m.message = "切换到 Claude"
			m.err = nil
		}
	case "x":
		// Switch to Codex
		if m.currentApp != "codex" {
			m.currentApp = "codex"
			m.cursor = 0
			m.refreshProviders()
			m.message = "切换到 Codex"
			m.err = nil
		}
	case "g":
		// Switch to Gemini
		if m.currentApp != "gemini" {
			m.currentApp = "gemini"
			m.cursor = 0
			m.refreshProviders()
			m.message = "切换到 Gemini"
			m.err = nil
		}
	case "b":
		// Create backup
		backupID, err := backup.CreateBackup(m.configPath)
		if err != nil {
			m.err = fmt.Errorf("创建备份失败: %w", err)
			m.message = ""
		} else if backupID == "" {
			m.err = errors.New("配置文件不存在，无法创建备份")
			m.message = ""
		} else {
			m.message = "备份已创建: " + backupID
			m.err = nil
		}
	case "l":
		// List backups
		configDir := filepath.Dir(m.configPath)
		backups, err := backup.ListBackups(configDir)
		if err != nil {
			m.err = fmt.Errorf("读取备份列表失败: %w", err)
			m.message = ""
		} else {
			m.backupList = backups
			m.backupCursor = 0
			m.mode = "backup_list"
			m.message = ""
			m.err = nil
		}
	case "m":
		// Template manager
		if m.templateManager == nil {
			m.err = errors.New("模板管理器未初始化")
			m.message = ""
		} else {
			m.mode = "template_manager"
			m.templateMode = "list"
			m.refreshTemplates()
			m.templateCursor = 0
			m.message = ""
			m.err = nil
		}
	case "M":
		// MCP manager
		m.mode = "mcp_manager"
		m.mcpMode = "list"
		m.refreshMcpServers()
		m.mcpCursor = 0
		m.message = ""
		m.err = nil
	case "u":
		// Check for updates
		m.message = "正在检查更新..."
		m.err = nil
		return m, checkUpdateCmd()
	case "U":
		// Download and install update (if available)
		if m.latestRelease != nil {
			m.message = "正在下载更新..."
			m.err = nil
			return m, downloadUpdateCmd(m.latestRelease)
		} else {
			m.err = errors.New("没有可用的更新，请先按 'u' 检查更新")
			m.message = ""
		}
	case "p":
		// Toggle portable mode
		if m.isPortableMode {
			// Disable portable mode
			err := m.disablePortableMode()
			if err != nil {
				m.err = fmt.Errorf("禁用便携模式失败: %w", err)
				m.message = ""
			} else {
				m.isPortableMode = false
				m.message = "✓ 便携模式已禁用"
				m.err = nil
			}
		} else {
			// Enable portable mode
			err := m.enablePortableMode()
			if err != nil {
				m.err = fmt.Errorf("启用便携模式失败: %w", err)
				m.message = ""
			} else {
				m.isPortableMode = true
				m.message = "✓ 便携模式已启用"
				m.err = nil
			}
		}
	}
	return m, nil
}

// handleMultiColumnKeys 处理三列模式的键盘事件
func (m Model) handleMultiColumnKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "v":
		// 切换回单列模式
		m.viewMode = "single"
		m.currentApp = m.columnToAppName(m.columnCursor)
		m.cursor = m.columnCursors[m.columnCursor]
		m.refreshProviders()
		m.saveViewModePreference()
		m.message = "切换到单列视图"
		m.err = nil
		return m, nil

	case "tab":
		// Tab 切换到下一列
		if m.columnCursor < 2 {
			m.columnCursor++
		} else {
			m.columnCursor = 0
		}

	case "left", "h":
		// 切换列
		if m.columnCursor > 0 {
			m.columnCursor--
		} else {
			m.columnCursor = 2 // 循环
		}
		// 使用期望行位置：若目标列行数不足则显示在最后一行，但不更新 desiredRow
		targetLen := len(m.columnProviders[m.columnCursor])
		if targetLen > 0 {
			if m.desiredRow >= targetLen {
				m.columnCursors[m.columnCursor] = targetLen - 1
			} else {
				m.columnCursors[m.columnCursor] = m.desiredRow
			}
		}

	case "right":
		// 切换列
		if m.columnCursor < 2 {
			m.columnCursor++
		} else {
			m.columnCursor = 0 // 循环
		}
		// 使用期望行位置：若目标列行数不足则显示在最后一行，但不更新 desiredRow
		targetLen := len(m.columnProviders[m.columnCursor])
		if targetLen > 0 {
			if m.desiredRow >= targetLen {
				m.columnCursors[m.columnCursor] = targetLen - 1
			} else {
				m.columnCursors[m.columnCursor] = m.desiredRow
			}
		}

	case "up", "k":
		col := m.columnCursor
		if len(m.columnProviders[col]) > 0 {
			if m.columnCursors[col] > 0 {
				m.columnCursors[col]--
			} else {
				m.columnCursors[col] = len(m.columnProviders[col]) - 1 // 循环
			}
			// 更新期望行位置
			m.desiredRow = m.columnCursors[col]
		}

	case "down", "j":
		col := m.columnCursor
		if len(m.columnProviders[col]) > 0 {
			if m.columnCursors[col] < len(m.columnProviders[col])-1 {
				m.columnCursors[col]++
			} else {
				m.columnCursors[col] = 0 // 循环
			}
			// 更新期望行位置
			m.desiredRow = m.columnCursors[col]
		}

	case "enter":
		col := m.columnCursor
		if len(m.columnProviders[col]) > 0 {
			provider := m.columnProviders[col][m.columnCursors[col]]
			appName := m.columnToAppName(col)
			err := m.manager.SwitchProviderForApp(appName, provider.Name)
			if err != nil {
				m.err = err
				m.message = ""
			} else {
				m.message = fmt.Sprintf("已切换 %s 配置: %s", appName, provider.Name)
				m.err = nil
				m.refreshAllColumns()
			}
		}

	case "a":
		// 添加配置到当前列对应的应用
		m.currentApp = m.columnToAppName(m.columnCursor)
		m.mode = "add"
		m.editName = ""
		m.copyFromProvider = nil
		m.initForm(nil)
		return m, textinput.Blink

	case "e":
		col := m.columnCursor
		if len(m.columnProviders[col]) > 0 {
			provider := m.columnProviders[col][m.columnCursors[col]]
			m.currentApp = m.columnToAppName(col)
			m.mode = "edit"
			m.editName = provider.Name
			m.initForm(&provider)
			return m, textinput.Blink
		}

	case "d":
		col := m.columnCursor
		if len(m.columnProviders[col]) > 0 {
			provider := m.columnProviders[col][m.columnCursors[col]]
			appName := m.columnToAppName(col)
			current := m.manager.GetCurrentProviderForApp(appName)
			if current != nil && provider.ID == current.ID {
				m.err = errors.New(i18n.T("error.cannot_delete_current"))
				m.message = ""
			} else {
				m.currentApp = appName
				m.mode = "delete"
				m.deleteName = provider.Name
			}
		}

	case "=":
		col := m.columnCursor
		providers := m.columnProviders[col]
		if len(providers) == 0 {
			break
		}
		cursor := m.columnCursors[col]
		if cursor == 0 {
			m.message = "已在顶部，无法继续上调"
			m.err = nil
			break
		}
		provider := providers[cursor]
		appName := m.columnToAppName(col)
		if err := m.manager.MoveProviderForApp(appName, provider.ID, -1); err != nil {
			m.err = err
			m.message = ""
			break
		}
		targetID := provider.ID
		m.refreshAllColumns()
		m.syncModTime()
		m.err = nil
		m.message = "↑ 顺序已上调: " + provider.Name
		// 更新光标位置
		newIndex := cursor - 1
		for idx, p := range m.columnProviders[col] {
			if p.ID == targetID {
				newIndex = idx
				break
			}
		}
		if newIndex < 0 {
			newIndex = 0
		}
		m.columnCursors[col] = newIndex

	case "-":
		col := m.columnCursor
		providers := m.columnProviders[col]
		if len(providers) == 0 {
			break
		}
		cursor := m.columnCursors[col]
		if cursor >= len(providers)-1 {
			m.message = "已在底部，无法继续下调"
			m.err = nil
			break
		}
		provider := providers[cursor]
		appName := m.columnToAppName(col)
		if err := m.manager.MoveProviderForApp(appName, provider.ID, 1); err != nil {
			m.err = err
			m.message = ""
			break
		}
		targetID := provider.ID
		m.refreshAllColumns()
		m.syncModTime()
		m.err = nil
		m.message = "↓ 顺序已下调: " + provider.Name
		// 更新光标位置
		newIndex := cursor + 1
		for idx, p := range m.columnProviders[col] {
			if p.ID == targetID {
				newIndex = idx
				break
			}
		}
		if newIndex > len(m.columnProviders[col])-1 {
			newIndex = len(m.columnProviders[col]) - 1
		}
		m.columnCursors[col] = newIndex

	case "C":
		// 复制当前选中的配置并进入创建模式
		col := m.columnCursor
		if len(m.columnProviders[col]) > 0 {
			provider := m.columnProviders[col][m.columnCursors[col]]
			m.currentApp = m.columnToAppName(col)
			m.mode = "add"
			m.editName = ""
			m.copyFromProvider = &provider
			m.initForm(nil)
			return m, textinput.Blink
		}

	case "b":
		// Create backup
		backupID, err := backup.CreateBackup(m.configPath)
		if err != nil {
			m.err = fmt.Errorf("创建备份失败: %w", err)
			m.message = ""
		} else if backupID == "" {
			m.err = errors.New("配置文件不存在，无法创建备份")
			m.message = ""
		} else {
			m.message = "备份已创建: " + backupID
			m.err = nil
		}

	case "l":
		// List backups
		configDir := filepath.Dir(m.configPath)
		backups, err := backup.ListBackups(configDir)
		if err != nil {
			m.err = fmt.Errorf("读取备份列表失败: %w", err)
			m.message = ""
		} else {
			m.backupList = backups
			m.backupCursor = 0
			m.mode = "backup_list"
			m.message = ""
			m.err = nil
		}

	case "m":
		// Template manager
		if m.templateManager == nil {
			m.err = errors.New("模板管理器未初始化")
			m.message = ""
		} else {
			m.currentApp = m.columnToAppName(m.columnCursor)
			m.mode = "template_manager"
			m.templateMode = "list"
			m.refreshTemplates()
			m.templateCursor = 0
			m.message = ""
			m.err = nil
		}

	case "M":
		// MCP manager
		m.mode = "mcp_manager"
		m.mcpMode = "list"
		m.refreshMcpServers()
		m.mcpCursor = 0
		m.message = ""
		m.err = nil

	case "p":
		// Toggle portable mode
		if m.isPortableMode {
			err := m.disablePortableMode()
			if err != nil {
				m.err = fmt.Errorf("禁用便携模式失败: %w", err)
				m.message = ""
			} else {
				m.isPortableMode = false
				m.message = "✓ 便携模式已禁用"
				m.err = nil
			}
		} else {
			err := m.enablePortableMode()
			if err != nil {
				m.err = fmt.Errorf("启用便携模式失败: %w", err)
				m.message = ""
			} else {
				m.isPortableMode = true
				m.message = "✓ 便携模式已启用"
				m.err = nil
			}
		}

	case "u":
		// Check for updates
		m.message = "正在检查更新..."
		m.err = nil
		return m, checkUpdateCmd()

	case "U":
		// Download and install update (if available)
		if m.latestRelease != nil {
			m.message = "正在下载更新..."
			m.err = nil
			return m, downloadUpdateCmd(m.latestRelease)
		} else {
			m.err = errors.New("没有可用的更新，请先按 'u' 检查更新")
			m.message = ""
		}

	case "r":
		m.refreshAllColumns()
		m.message = "列表已刷新"
		m.err = nil
		return m, tea.ClearScreen
	}

	return m, nil
}

// viewList 渲染单列列表视图
func (m Model) viewList() string {
	var s strings.Builder

	// Title with current app indicator and version
	appName := "Claude Code"
	switch m.currentApp {
	case "codex":
		appName = "Codex CLI"
	case "gemini":
		appName = "Gemini CLI"
	}

	// 添加便携模式标识
	portableIndicator := ""
	if m.isPortableMode {
		portableIndicator = " (便携版)"
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#007AFF")).
		Padding(0, 1).
		Render(fmt.Sprintf("CC Switch CLI v%s - %s 配置管理%s", m.getVersion(), appName, portableIndicator))
	s.WriteString(title + "\n\n")

	// Status message
	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3B30")).Bold(true)
		s.WriteString(errStyle.Render("✗ "+m.err.Error()) + "\n\n")
	} else if m.message != "" {
		msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#34C759")).Bold(true)
		s.WriteString(msgStyle.Render("✓ "+m.message) + "\n\n")
	}

	// Provider list
	if len(m.providers) == 0 {
		s.WriteString(fmt.Sprintf("暂无 %s 配置，按 'a' 添加新配置\n\n", appName))
	} else {
		current := m.manager.GetCurrentProviderForApp(m.currentApp)
		for i, p := range m.providers {
			// 判断是否是当前激活的配置
			isActive := current != nil && p.ID == current.ID
			isCursor := i == m.cursor

			// 分开渲染 marker 和名称，避免样式覆盖
			marker := "○"
			markerStyle := lipgloss.NewStyle()
			if isActive {
				markerStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#34C759")).
					Bold(true)
				marker = "●"
			}
			styledMarker := markerStyle.Render(marker)

			// 渲染名称
			nameText := p.Name
			if isCursor {
				if isActive {
					// 当前激活 + 光标选中 = 绿色背景 + 白色文字
					nameText = lipgloss.NewStyle().
						Background(lipgloss.Color("#34C759")).
						Foreground(lipgloss.Color("#FFFFFF")).
						Bold(true).
						Padding(0, 1).
						Render(nameText)
				} else {
					// 仅光标选中 = 蓝色背景 + 白色文字
					nameText = lipgloss.NewStyle().
						Background(lipgloss.Color("#007AFF")).
						Foreground(lipgloss.Color("#FFFFFF")).
						Bold(true).
						Padding(0, 1).
						Render(nameText)
				}
			} else {
				nameText = lipgloss.NewStyle().
					Padding(0, 1).
					Render(nameText)
			}

			// 组合 marker 和名称
			line := fmt.Sprintf("%s %s", styledMarker, nameText)
			s.WriteString(line + "\n")
		}
	}

	// Help
	s.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8E8E93"))
	helps := []string{
		"↑/↓: 选择",
		"=: 上调",
		"-: 下调",
		"Enter: 切换",
		"a: 添加",
		"C: 复制",
		"e: 编辑",
		"d: 删除",
		"b: 备份",
		"l: 备份列表",
		"m: 模板管理",
		"M: MCP管理",
		"t: 切换应用",
		"c: Claude",
		"x: Codex",
		"g: Gemini",
		"v: 三列视图",
		"p: 便携模式",
		"u: 检查更新",
		"U: 执行更新",
		"r: 刷新",
		"q: 退出",
	}

	const itemsPerLine = 5
	var helpLines []string
	for i := 0; i < len(helps); i += itemsPerLine {
		end := i + itemsPerLine
		if end > len(helps) {
			end = len(helps)
		}
		helpLines = append(helpLines, strings.Join(helps[i:end], " • "))
	}
	s.WriteString(helpStyle.Render(strings.Join(helpLines, "\n")))

	return s.String()
}

// viewListMulti 三列视图渲染（表格形式）
func (m Model) viewListMulti() string {
	var s strings.Builder

	// 便携模式标识
	portableIndicator := ""
	if m.isPortableMode {
		portableIndicator = " (便携版)"
	}

	// 标题
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#007AFF")).
		Render(fmt.Sprintf("CC Switch CLI v%s - 三列视图%s", m.getVersion(), portableIndicator))
	s.WriteString(title + "\n\n")

	// 状态消息
	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3B30")).Bold(true)
		s.WriteString(errStyle.Render("✗ "+m.err.Error()) + "\n\n")
	} else if m.message != "" {
		msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#34C759")).Bold(true)
		s.WriteString(msgStyle.Render("✓ "+m.message) + "\n\n")
	}

	// 计算每列宽度（基于最长配置名称的显示宽度 + 2）
	appTitles := []string{"Claude Code", "Codex CLI", "Gemini CLI"}
	colWidths := make([]int, 3)
	for i := 0; i < 3; i++ {
		// 最小宽度为标题显示宽度
		maxWidth := displayWidth(appTitles[i])
		for _, p := range m.columnProviders[i] {
			w := displayWidth(p.Name)
			if w > maxWidth {
				maxWidth = w
			}
		}
		// 如果该列为空，确保至少能显示"暂无配置"
		if len(m.columnProviders[i]) == 0 {
			emptyHint := displayWidth("暂无配置")
			if emptyHint > maxWidth {
				maxWidth = emptyHint
			}
		}
		colWidths[i] = maxWidth + 2 // +2 为左右各留 1 字符空隙
		if colWidths[i] < 10 {
			colWidths[i] = 10
		}
	}

	// 计算最大行数
	maxRows := 0
	for i := 0; i < 3; i++ {
		if len(m.columnProviders[i]) > maxRows {
			maxRows = len(m.columnProviders[i])
		}
	}
	if maxRows == 0 {
		maxRows = 1 // 至少显示一行（空提示）
	}

	// 表格边框字符
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#3A3A3C"))

	// 渲染表头边框
	s.WriteString(borderStyle.Render(m.renderTableBorder(colWidths, "top")) + "\n")

	// 渲染表头
	s.WriteString(m.renderTableHeader(appTitles, colWidths) + "\n")

	// 渲染表头分隔线
	s.WriteString(borderStyle.Render(m.renderTableBorder(colWidths, "middle")) + "\n")

	// 渲染数据行
	for row := 0; row < maxRows; row++ {
		s.WriteString(m.renderTableRow(row, colWidths) + "\n")
	}

	// 渲染底部边框
	s.WriteString(borderStyle.Render(m.renderTableBorder(colWidths, "bottom")) + "\n")

	// 底部帮助信息
	s.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8E8E93"))
	helps := []string{
		"Tab/←→: 切换列",
		"↑/↓: 选择",
		"=: 上调",
		"-: 下调",
		"Enter: 切换",
		"a: 添加",
		"C: 复制",
		"e: 编辑",
		"d: 删除",
		"b: 备份",
		"l: 备份列表",
		"m: 模板管理",
		"M: MCP管理",
		"p: 便携模式",
		"u: 检查更新",
		"U: 执行更新",
		"r: 刷新",
		"v: 单列模式",
		"q: 退出",
	}

	const itemsPerLine = 5
	var helpLines []string
	for i := 0; i < len(helps); i += itemsPerLine {
		end := i + itemsPerLine
		if end > len(helps) {
			end = len(helps)
		}
		helpLines = append(helpLines, strings.Join(helps[i:end], " • "))
	}
	s.WriteString(helpStyle.Render(strings.Join(helpLines, "\n")))

	return s.String()
}

// renderTableBorder 渲染表格边框
func (m Model) renderTableBorder(colWidths []int, position string) string {
	var left, mid, right, fill string
	switch position {
	case "top":
		left, mid, right, fill = "┌", "┬", "┐", "─"
	case "middle":
		left, mid, right, fill = "├", "┼", "┤", "─"
	case "bottom":
		left, mid, right, fill = "└", "┴", "┘", "─"
	}

	var parts []string
	for i, w := range colWidths {
		if i == 0 {
			parts = append(parts, left)
		}
		parts = append(parts, strings.Repeat(fill, w+2)) // +2 为单元格内边距
		if i < len(colWidths)-1 {
			parts = append(parts, mid)
		} else {
			parts = append(parts, right)
		}
	}
	return strings.Join(parts, "")
}

// renderTableHeader 渲染表头
func (m Model) renderTableHeader(titles []string, colWidths []int) string {
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#3A3A3C"))
	var cells []string

	for i, title := range titles {
		isActive := i == m.columnCursor
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#007AFF"))
		if isActive {
			titleStyle = titleStyle.Underline(true)
		}

		// 居中对齐（使用显示宽度）
		titleWidth := displayWidth(title)
		padding := colWidths[i] - titleWidth
		leftPad := padding / 2
		rightPad := padding - leftPad
		cell := strings.Repeat(" ", leftPad) + titleStyle.Render(title) + strings.Repeat(" ", rightPad)
		cells = append(cells, " "+cell+" ")
	}

	return borderStyle.Render("│") + strings.Join(cells, borderStyle.Render("│")) + borderStyle.Render("│")
}

// renderTableRow 渲染数据行
func (m Model) renderTableRow(row int, colWidths []int) string {
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#3A3A3C"))
	var cells []string

	for col := 0; col < 3; col++ {
		providers := m.columnProviders[col]
		width := colWidths[col]
		isActiveColumn := col == m.columnCursor
		appName := m.columnToAppName(col)
		current := m.manager.GetCurrentProviderForApp(appName)

		var cellContent string
		var contentWidth int

		if row < len(providers) {
			p := providers[row]
			isActive := current != nil && p.ID == current.ID
			isCursor := isActiveColumn && row == m.columnCursors[col]

			marker := "○"
			markerStyle := lipgloss.NewStyle()
			if isActive {
				markerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#34C759")).Bold(true)
				marker = "●"
			}

			nameStyle := lipgloss.NewStyle()
			if isCursor {
				if isActive {
					nameStyle = nameStyle.Background(lipgloss.Color("#34C759")).Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
				} else {
					nameStyle = nameStyle.Background(lipgloss.Color("#007AFF")).Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
				}
			}

			name := p.Name
			nameWidth := displayWidth(name)

			// 截断过长名称
			if nameWidth > width-2 {
				// 逐字符截断直到宽度合适
				truncated := ""
				truncWidth := 0
				for _, r := range name {
					rw := 1
					if unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) ||
						unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r) ||
						(r >= 0xFF00 && r <= 0xFFEF) {
						rw = 2
					}
					if truncWidth+rw+3 > width-2 { // 留3个字符给"..."
						break
					}
					truncated += string(r)
					truncWidth += rw
				}
				name = truncated + "..."
				nameWidth = truncWidth + 3
			}

			cellContent = markerStyle.Render(marker) + " " + nameStyle.Render(name)
			contentWidth = 2 + nameWidth // marker(1) + space(1) + name
		} else if row == 0 && len(providers) == 0 {
			// 空列提示
			hint := "暂无配置"
			hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8E8E93")).Italic(true)
			cellContent = hintStyle.Render(hint)
			contentWidth = displayWidth(hint)
		} else {
			// 空单元格
			cellContent = ""
			contentWidth = 0
		}

		// 填充空格对齐
		if contentWidth < width {
			cellContent += strings.Repeat(" ", width-contentWidth)
		}

		cells = append(cells, " "+cellContent+" ")
	}

	return borderStyle.Render("│") + strings.Join(cells, borderStyle.Render("│")) + borderStyle.Render("│")
}
