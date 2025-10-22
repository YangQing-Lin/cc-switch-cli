package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/YangQing-Lin/cc-switch-cli/internal/backup"
	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/i18n"
	"github.com/YangQing-Lin/cc-switch-cli/internal/portable"
	"github.com/YangQing-Lin/cc-switch-cli/internal/template"
	"github.com/YangQing-Lin/cc-switch-cli/internal/version"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model TUI 主模型
type Model struct {
	manager             *config.Manager
	providers           []config.Provider
	cursor              int
	width               int
	height              int
	err                 error
	message             string
	mode                string // "list", "add", "edit", "delete", "app_select", "backup_list", "template_manager"
	editName            string // 正在编辑的配置名称
	deleteName          string // 要删除的配置名称
	inputs              []textinput.Model
	focusIndex          int
	currentApp          string              // "claude" or "codex"
	appCursor           int                 // 应用选择光标 (0=Claude, 1=Codex)
	lastModTime         time.Time           // 配置文件最后修改时间
	configPath          string              // 配置文件路径
	configCorrupted     bool                // 配置文件是否损坏
	backupList          []backup.BackupInfo // 备份列表
	backupCursor        int                 // 备份列表光标
	modelSelectorActive bool                // 模型选择器是否激活（Claude model 或 Default Sonnet Model）
	modelSelectorCursor int                 // 模型选择器光标位置

	// 模板管理相关字段
	templateManager *template.TemplateManager // 模板管理器实例
	templates       []template.Template       // 当前显示的模板列表
	templateCursor  int                       // 模板列表光标位置
	templateMode    string                    // 模板子模式状态机

	// 应用模板流程
	selectedTemplate   *template.Template // 当前选中的模板
	targetSelectCursor int                // 目标路径选择光标（0-2）
	selectedTargetPath string             // 选中的目标路径（绝对路径）
	diffContent        string             // 生成的 diff 内容
	diffScrollOffset   int                // diff 查看滚动偏移量

	// 保存模板流程
	sourceSelectCursor int             // 源路径选择光标（0-2）
	selectedSourcePath string          // 选中的源路径
	saveNameInput      textinput.Model // 模板名称输入框

	// 预览流程
	previewScrollOffset int // 预览内容滚动偏移量

	// 撤销历史栈（用于清空后的回退操作）
	undoHistory []struct {
		name       string
		token      string
		baseURL    string
		websiteURL string
		modelValue string
		extraValue string
	}

	// 复制配置相关
	copyFromProvider *config.Provider // 从哪个配置复制（用于创建时预填充）

	// 更新相关
	latestRelease *version.ReleaseInfo // 最新版本信息

	// 便携模式相关
	isPortableMode bool // 是否为便携模式

	// API Token 显示状态
	apiTokenVisible bool
}

// tickMsg is sent on every tick for config refresh
type tickMsg time.Time

// updateCheckMsg is sent when update check completes
type updateCheckMsg struct {
	release   *version.ReleaseInfo
	hasUpdate bool
	err       error
}

// updateDownloadMsg is sent when update download completes
type updateDownloadMsg struct {
	err error
}

// New 创建新的 TUI 模型
func New(manager *config.Manager) Model {
	configPath := manager.GetConfigPath()
	var modTime time.Time
	if info, err := os.Stat(configPath); err == nil {
		modTime = info.ModTime()
	}

	// 初始化模板管理器
	homeDir, _ := os.UserHomeDir()
	templateConfigPath := filepath.Join(homeDir, ".cc-switch", "claude_templates.json")
	templateManager, err := template.NewTemplateManager(templateConfigPath)
	var initErr error
	if err != nil {
		initErr = fmt.Errorf("初始化模板管理器失败: %w", err)
	}

	m := Model{
		manager:         manager,
		mode:            "list",
		currentApp:      "claude",
		appCursor:       0,
		configPath:      configPath,
		lastModTime:     modTime,
		templateManager: templateManager,
		isPortableMode:  portable.IsPortableMode(),
	}
	if initErr != nil {
		m.err = initErr
	}
	m.refreshProviders()
	return m
}

func (m *Model) refreshProviders() {
	m.providers = m.manager.ListProvidersForApp(m.currentApp)
}

func (m *Model) refreshTemplates() {
	if m.templateManager == nil {
		return
	}
	allTemplates := m.templateManager.ListTemplates(m.currentTemplateCategory())
	m.templates = allTemplates
}

func (m Model) currentTemplateCategory() string {
	if m.currentApp == "codex" {
		return template.CategoryCodexMd
	}
	return template.CategoryClaudeMd
}

func (m Model) templateCategoryDisplay(category string) string {
	switch category {
	case template.CategoryCodexMd:
		return "Codex 指南 (CODEX.md)"
	case template.CategoryClaudeMd:
		return "Claude 指南 (CLAUDE.md)"
	default:
		return category
	}
}

// syncModTime updates the cached modification time after internal config changes
func (m *Model) syncModTime() {
	if info, err := os.Stat(m.configPath); err == nil {
		m.lastModTime = info.ModTime()
	}
}

// checkConfigChanges checks if config file has been modified externally
func (m *Model) checkConfigChanges() {
	info, err := os.Stat(m.configPath)
	if err != nil {
		// Config file doesn't exist or can't be accessed
		if !m.configCorrupted {
			m.configCorrupted = true
			m.err = errors.New("配置文件不可访问，请使用 'backup restore' 命令恢复")
			m.message = ""
		}
		return
	}

	modTime := info.ModTime()
	if modTime.After(m.lastModTime) {
		// Config file was modified externally, reload
		m.lastModTime = modTime

		// Try to reload config
		if err := m.manager.Load(); err != nil {
			m.configCorrupted = true
			m.err = fmt.Errorf("配置文件损坏: %v。请使用 'backup restore' 恢复", err)
			m.message = ""
		} else {
			m.configCorrupted = false
			m.refreshProviders()
			m.message = "配置已从外部更新刷新"
			m.err = nil
		}
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
	)
}

// tickCmd returns a command that sends a tick message every 2 seconds
func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// checkUpdateCmd returns a command that checks for updates
func checkUpdateCmd() tea.Cmd {
	return func() tea.Msg {
		release, hasUpdate, err := version.CheckForUpdate()
		return updateCheckMsg{
			release:   release,
			hasUpdate: hasUpdate,
			err:       err,
		}
	}
}

// downloadUpdateCmd returns a command that downloads and installs update
func downloadUpdateCmd(release *version.ReleaseInfo) tea.Cmd {
	return func() tea.Msg {
		err := version.DownloadUpdate(release)
		return updateDownloadMsg{
			err: err,
		}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		// Only check config file changes in list mode
		if m.mode == "list" {
			m.checkConfigChanges()
		}
		return m, tickCmd()

	case updateCheckMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("检查更新失败: %w", msg.err)
			m.message = ""
			m.latestRelease = nil
		} else if msg.hasUpdate {
			m.message = fmt.Sprintf("发现新版本 %s！按 'U' 键下载更新", msg.release.TagName)
			m.err = nil
			m.latestRelease = msg.release
		} else {
			m.message = "当前已是最新版本"
			m.err = nil
			m.latestRelease = nil
		}

	case updateDownloadMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("更新失败: %w", msg.err)
			m.message = ""
		} else {
			m.message = "✓ 更新成功！请退出并重新运行程序"
			m.err = nil
			m.latestRelease = nil
		}

	case tea.KeyMsg:
		switch m.mode {
		case "list":
			return m.handleListKeys(msg)
		case "add", "edit":
			// 先处理特殊键,再更新输入框
			handled, newModel, cmd := m.handleFormKeys(msg)
			if handled {
				return newModel, cmd
			}
			// 未被特殊键处理,继续更新输入框
			return m.updateInputs(msg)
		case "delete":
			return m.handleDeleteKeys(msg)
		case "app_select":
			return m.handleAppSelectKeys(msg)
		case "backup_list":
			return m.handleBackupListKeys(msg)
		case "template_manager":
			switch m.templateMode {
			case "list":
				return m.handleTemplateListKeys(msg)
			case "apply_select_target":
				return m.handleTargetSelectKeys(msg)
			case "apply_preview_diff":
				return m.handleDiffPreviewKeys(msg)
			case "save_select_source":
				return m.handleSourceSelectKeys(msg)
			case "save_input_name":
				return m.handleSaveNameKeys(msg)
			case "preview":
				return m.handlePreviewKeys(msg)
			case "delete_confirm":
				return m.handleDeleteConfirmKeys(msg)
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	switch m.mode {
	case "list":
		return m.viewList()
	case "add", "edit":
		return m.viewForm()
	case "delete":
		return m.viewDelete()
	case "app_select":
		return m.viewAppSelect()
	case "backup_list":
		return m.viewBackupList()
	case "template_manager":
		switch m.templateMode {
		case "list":
			return m.viewTemplateList()
		case "apply_select_target":
			return m.viewTargetSelect()
		case "apply_preview_diff":
			return m.viewDiffPreview()
		case "save_select_source":
			return m.viewSourceSelect()
		case "save_input_name":
			return m.viewSaveNameInput()
		case "preview":
			return m.viewTemplatePreview()
		case "delete_confirm":
			return m.viewTemplateDelete()
		}
	}
	return ""
}

// List view handlers
func (m Model) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.providers)-1 {
			m.cursor++
		}
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
		// Toggle between Claude and Codex
		if m.currentApp == "claude" {
			m.currentApp = "codex"
		} else {
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

// Delete handlers
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

// App select handlers
func (m Model) handleAppSelectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.mode = "list"
	case "up", "k":
		if m.appCursor > 0 {
			m.appCursor--
		}
	case "down", "j":
		if m.appCursor < 1 {
			m.appCursor++
		}
	case "enter":
		if m.appCursor == 0 {
			m.currentApp = "claude"
		} else {
			m.currentApp = "codex"
		}
		m.cursor = 0
		m.refreshProviders()
		m.mode = "list"
		m.message = fmt.Sprintf("切换到 %s", m.currentApp)
	}
	return m, nil
}

// View renderers
func (m Model) viewList() string {
	var s strings.Builder

	// Title with current app indicator and version
	appName := "Claude Code"
	if m.currentApp == "codex" {
		appName = "Codex CLI"
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
		"Enter: 切换",
		"a: 添加",
		"C: 复制",
		"e: 编辑",
		"d: 删除",
		"b: 备份",
		"l: 备份列表",
		"m: 模板管理",
		"t: 切换应用",
		"c: Claude",
		"x: Codex",
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

// viewAppSelect renders the app selection screen
func (m Model) viewAppSelect() string {
	var s strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#007AFF")).
		Padding(0, 1).
		Render(fmt.Sprintf("选择应用 (v%s)", m.getVersion()))
	s.WriteString(title + "\n\n")

	apps := []string{"Claude Code", "Codex CLI"}
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

// viewBackupList renders the backup list view
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

			// 标记自动备份
			backupType := ""
			if strings.HasPrefix(filename, backup.AutoBackupPrefix) {
				backupType = " (自动)"
			} else {
				backupType = " (手动)"
			}

			displayName := filename + backupType

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

// handleBackupListKeys handles keys in backup list mode
func (m Model) handleBackupListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.mode = "list"
		m.message = ""
		m.err = nil
	case "up", "k":
		if m.backupCursor > 0 {
			m.backupCursor--
		}
	case "down", "j":
		if m.backupCursor < len(m.backupList)-1 {
			m.backupCursor++
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

// getVersion 获取版本号
func (m Model) getVersion() string {
	return version.GetVersion()
}

// enablePortableMode 启用便携模式
func (m *Model) enablePortableMode() error {
	// 获取可执行文件目录
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取程序路径失败: %w", err)
	}

	execDir := filepath.Dir(execPath)
	portableFile := filepath.Join(execDir, "portable.ini")

	// 创建 portable.ini 文件
	content := []byte("# CC-Switch Portable Mode\n# This file enables portable mode.\n# Delete this file to disable portable mode.\n")
	if err := os.WriteFile(portableFile, content, 0644); err != nil {
		return fmt.Errorf("创建 portable.ini 失败: %w", err)
	}

	return nil
}

// disablePortableMode 禁用便携模式
func (m *Model) disablePortableMode() error {
	// 获取可执行文件目录
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取程序路径失败: %w", err)
	}

	execDir := filepath.Dir(execPath)
	portableFile := filepath.Join(execDir, "portable.ini")

	// 删除 portable.ini 文件
	if err := os.Remove(portableFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除 portable.ini 失败: %w", err)
	}

	return nil
}
