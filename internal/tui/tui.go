package tui

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/YangQing-Lin/cc-switch-cli/internal/backup"
	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/i18n"
	"github.com/YangQing-Lin/cc-switch-cli/internal/template"
	"github.com/YangQing-Lin/cc-switch-cli/internal/version"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// 预定义的 model 选项
var predefinedClaudeModels = []string{
	"", // 不填（默认推荐）
	"opus",
	"opusplan",
}

// 预定义的 Default Sonnet Model 选项
var predefinedModels = []string{
	"清空",
	"claude-sonnet-4-5-20250929",
	"claude-sonnet-4-20250514",
}

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
		name          string
		token         string
		baseURL       string
		websiteURL    string
		claudeModel   string
		defaultSonnet string
	}

	// 复制配置相关
	copyFromProvider *config.Provider // 从哪个配置复制（用于创建时预填充）

	// 更新相关
	latestRelease *version.ReleaseInfo // 最新版本信息
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
	allTemplates := m.templateManager.ListTemplates(template.CategoryClaudeMd)
	m.templates = allTemplates
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
	}
	return m, nil
}

// Form handlers - 返回 (handled, model, cmd)
func (m Model) handleFormKeys(msg tea.KeyMsg) (bool, tea.Model, tea.Cmd) {
	// 当焦点在 model 字段（索引 4）或 Default Sonnet Model 字段（索引 5）时，处理模型选择器
	if m.focusIndex == 4 || m.focusIndex == 5 {
		switch msg.String() {
		case "right":
			// 激活模型选择器
			if !m.modelSelectorActive {
				m.modelSelectorActive = true
				m.modelSelectorCursor = 0
				return true, m, nil
			}
		case "left":
			// 取消激活模型选择器
			if m.modelSelectorActive {
				m.modelSelectorActive = false
				return true, m, nil
			}
		case "up":
			// 如果模型选择器激活，上移光标
			if m.modelSelectorActive {
				if m.modelSelectorCursor > 0 {
					m.modelSelectorCursor--
				}
				return true, m, nil
			}
		case "down":
			// 如果模型选择器激活，下移光标
			if m.modelSelectorActive {
				// 动态获取选项数量
				var maxCursor int
				if m.focusIndex == 4 {
					maxCursor = len(predefinedClaudeModels) - 1 // model 字段
				} else {
					maxCursor = len(predefinedModels) - 1 // Default Sonnet Model
				}
				if m.modelSelectorCursor < maxCursor {
					m.modelSelectorCursor++
				}
				return true, m, nil
			}
		case "enter":
			// 如果模型选择器激活，选择模型并填入
			if m.modelSelectorActive {
				var selectedModel string
				if m.focusIndex == 4 {
					selectedModel = predefinedClaudeModels[m.modelSelectorCursor]
				} else {
					// Default Sonnet Model 字段
					if m.modelSelectorCursor == 0 {
						// 选择"清空"选项，清空输入框
						selectedModel = ""
					} else {
						selectedModel = predefinedModels[m.modelSelectorCursor]
					}
				}
				m.inputs[m.focusIndex].SetValue(selectedModel)
				m.modelSelectorActive = false
				return true, m, nil
			}
		}
	}

	switch msg.String() {
	case "ctrl+d":
		// 清空当前表单的所有内容（通用功能）
		m.clearFormFields()
		return true, m, nil
	case "ctrl+z":
		// 回退上一次清空操作
		if m.undoLastClear() {
			return true, m, nil
		}
	case "esc":
		// 如果模型选择器激活，先关闭选择器
		if m.modelSelectorActive {
			m.modelSelectorActive = false
			return true, m, nil
		}
		// 否则返回列表
		m.mode = "list"
		m.message = ""
		m.err = nil
		return true, m, nil
	case "tab", "shift+tab":
		// Tab 切换字段时，关闭模型选择器
		m.modelSelectorActive = false
		if msg.String() == "shift+tab" {
			m.focusIndex--
		} else {
			m.focusIndex++
		}
		if m.focusIndex >= len(m.inputs) {
			m.focusIndex = 0
		} else if m.focusIndex < 0 {
			m.focusIndex = len(m.inputs) - 1
		}
		cmds := make([]tea.Cmd, len(m.inputs))
		for i := range m.inputs {
			if i == m.focusIndex {
				cmds[i] = m.inputs[i].Focus()
			} else {
				m.inputs[i].Blur()
			}
		}
		return true, m, tea.Batch(cmds...)
	case "up", "down":
		// 如果不是在模型字段，或者选择器未激活，正常切换字段
		if (m.focusIndex != 4 && m.focusIndex != 5) || !m.modelSelectorActive {
			m.modelSelectorActive = false
			if msg.String() == "up" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}
			if m.focusIndex >= len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs) - 1
			}
			cmds := make([]tea.Cmd, len(m.inputs))
			for i := range m.inputs {
				if i == m.focusIndex {
					cmds[i] = m.inputs[i].Focus()
				} else {
					m.inputs[i].Blur()
				}
			}
			return true, m, tea.Batch(cmds...)
		}
	case "ctrl+s":
		m.submitForm()
		return true, m, nil
	case "enter":
		// 如果不在模型选择器中，Enter 提交表单
		if !m.modelSelectorActive {
			m.submitForm()
			return true, m, nil
		}
	}
	// 未处理,返回 false
	return false, m, nil
}

func (m *Model) submitForm() {
	name := m.inputs[0].Value()
	token := m.inputs[1].Value()
	baseURL := m.inputs[2].Value()
	websiteURL := m.inputs[3].Value()
	claudeModel := m.inputs[4].Value()
	defaultSonnetModel := m.inputs[5].Value()

	if name == "" {
		m.err = errors.New(i18n.T("error.name_required"))
		return
	}
	if token == "" {
		m.err = errors.New(i18n.T("error.token_required"))
		return
	}
	if baseURL == "" {
		m.err = errors.New(i18n.T("error.base_url_required"))
		return
	}

	var err error
	if m.mode == "edit" {
		// Update provider
		err = m.manager.UpdateProviderForApp(m.currentApp, m.editName, name, websiteURL, token, baseURL, "custom", claudeModel, defaultSonnetModel)
	} else {
		// Add provider
		err = m.manager.AddProviderForApp(m.currentApp, name, websiteURL, token, baseURL, "custom", claudeModel, defaultSonnetModel)
	}

	if err != nil {
		m.err = err
		m.message = ""
	} else {
		if m.mode == "edit" {
			m.message = i18n.T("success.provider_updated")
		} else {
			m.message = i18n.T("success.provider_added")
		}
		m.err = nil
		m.mode = "list"
		m.refreshProviders()
		m.syncModTime()
	}
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

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#007AFF")).
		Padding(0, 1).
		Render(fmt.Sprintf("CC Switch CLI v%s - %s 配置管理", m.getVersion(), appName))
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

func (m Model) viewForm() string {
	var formContent strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#007AFF")).
		Padding(0, 1)

	if m.mode == "add" {
		formContent.WriteString(title.Render(fmt.Sprintf("添加新配置 (v%s)", m.getVersion())) + "\n\n")
	} else {
		formContent.WriteString(title.Render(fmt.Sprintf("编辑配置 (v%s)", m.getVersion())) + "\n\n")
	}

	// Error
	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3B30")).Bold(true)
		formContent.WriteString(errStyle.Render("✗ "+m.err.Error()) + "\n\n")
	}

	// Form (必填字段在前，可选字段在后)
	labels := []string{"配置名称", "API Token", "Base URL", "网站 (可选)", "默认模型（可选）", "Default Sonnet Model (可选)"}
	for i, label := range labels {
		formContent.WriteString(lipgloss.NewStyle().Bold(true).Render(label+":") + "\n")
		if i == m.focusIndex {
			formContent.WriteString(lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#007AFF")).
				Render(m.inputs[i].View()) + "\n\n")
		} else {
			formContent.WriteString(m.inputs[i].View() + "\n\n")
		}
	}

	// Buttons
	submitStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#007AFF")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 2).
		Bold(true)
	cancelStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#8E8E93")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 2)

	// 清空按钮（始终显示，根据是否有内容决定样式）
	clearStyle := lipgloss.NewStyle()
	if m.anyFieldHasValue() {
		// 有内容，高亮显示
		clearStyle = clearStyle.
			Background(lipgloss.Color("#FF9500")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 2)
	} else {
		// 无内容，灰色显示
		clearStyle = clearStyle.
			Background(lipgloss.Color("#3A3A3C")).
			Foreground(lipgloss.Color("#636366")).
			Padding(0, 2)
	}

	// 回退按钮（始终显示，根据是否有历史记录决定样式）
	undoStyle := lipgloss.NewStyle()
	if len(m.undoHistory) > 0 {
		// 有历史记录，高亮显示
		undoStyle = undoStyle.
			Background(lipgloss.Color("#34C759")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 2)
	} else {
		// 无历史记录，灰色显示
		undoStyle = undoStyle.
			Background(lipgloss.Color("#3A3A3C")).
			Foreground(lipgloss.Color("#636366")).
			Padding(0, 2)
	}

	formContent.WriteString(submitStyle.Render("保存 (Enter)") + " ")
	formContent.WriteString(cancelStyle.Render("取消 (ESC)") + " ")
	formContent.WriteString(clearStyle.Render("清空内容 (Ctrl+D)") + " ")
	formContent.WriteString(undoStyle.Render("回退 (Ctrl+Z)") + "\n\n")

	// Help
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8E8E93"))
	helpText := "Tab: 下一项 • Shift+Tab: 上一项"
	if m.focusIndex == 4 || m.focusIndex == 5 {
		helpText = "→: 显示模型选项 • ←: 隐藏模型选项 • Tab: 下一项"
	}
	formContent.WriteString(helpStyle.Render(helpText))

	// 如果焦点在 model 或 Default Sonnet Model 字段，且选择器激活，显示侧边栏
	if (m.focusIndex == 4 || m.focusIndex == 5) && m.modelSelectorActive {
		// 构建模型选择器面板
		var selectorContent strings.Builder
		var selectorTitle string
		var optionsList []string

		if m.focusIndex == 4 {
			selectorTitle = "选择模型"
			optionsList = []string{"Default (recommended)", "Opus", "Opus Plan Mode"}
		} else {
			selectorTitle = "预定义模型"
			optionsList = []string{"清空", "claude-sonnet-4-5-20250929", "claude-sonnet-4-20250514"}
		}

		selectorTitleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#007AFF")).
			Padding(0, 1).
			Render(selectorTitle)
		selectorContent.WriteString(selectorTitleStyle + "\n\n")

		for i, model := range optionsList {
			if i == m.modelSelectorCursor {
				// 高亮选中的模型
				selectedStyle := lipgloss.NewStyle().
					Background(lipgloss.Color("#007AFF")).
					Foreground(lipgloss.Color("#FFFFFF")).
					Bold(true).
					Padding(0, 1)
				selectorContent.WriteString(selectedStyle.Render("● "+model) + "\n")
			} else {
				normalStyle := lipgloss.NewStyle().Padding(0, 1)
				selectorContent.WriteString(normalStyle.Render("○ "+model) + "\n")
			}
		}

		selectorContent.WriteString("\n")
		selectorHelp := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8E8E93")).
			Render("↑/↓: 选择 • Enter: 确认")
		selectorContent.WriteString(selectorHelp)

		// 给选择器面板添加边框
		selectorPanel := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#007AFF")).
			Padding(1, 2).
			Render(selectorContent.String())

		// 使用 JoinHorizontal 将表单和选择器并排显示
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			formContent.String(),
			"  ", // 间距
			selectorPanel,
		)
	}

	return formContent.String()
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

// Helper functions

// clearFormFields 清空表单所有字段并保存到历史栈
func (m *Model) clearFormFields() {
	// 只有在有值的情况下才保存到历史栈
	if m.anyFieldHasValue() {
		m.undoHistory = append(m.undoHistory, struct {
			name          string
			token         string
			baseURL       string
			websiteURL    string
			claudeModel   string
			defaultSonnet string
		}{
			name:          m.inputs[0].Value(),
			token:         m.inputs[1].Value(),
			baseURL:       m.inputs[2].Value(),
			websiteURL:    m.inputs[3].Value(),
			claudeModel:   m.inputs[4].Value(),
			defaultSonnet: m.inputs[5].Value(),
		})
	}

	// 清空所有字段（包括配置名称）
	m.inputs[0].SetValue("") // Name
	m.inputs[1].SetValue("") // Token
	m.inputs[2].SetValue("") // Base URL
	m.inputs[3].SetValue("") // Website URL
	m.inputs[4].SetValue("") // Model
	m.inputs[5].SetValue("") // Default Sonnet Model
}

// anyFieldHasValue 检查表单是否有任何非空字段
func (m *Model) anyFieldHasValue() bool {
	// 检查所有字段是否有值
	return m.inputs[0].Value() != "" || // Name
		m.inputs[1].Value() != "" || // Token
		m.inputs[2].Value() != "" || // Base URL
		m.inputs[3].Value() != "" || // Website URL
		m.inputs[4].Value() != "" || // Model
		m.inputs[5].Value() != "" // Default Sonnet Model
}

// undoLastClear 回退上一次清空操作
func (m *Model) undoLastClear() bool {
	if len(m.undoHistory) == 0 {
		return false // 没有历史记录
	}

	// 从栈顶取出最后一次的值
	lastState := m.undoHistory[len(m.undoHistory)-1]
	m.undoHistory = m.undoHistory[:len(m.undoHistory)-1]

	// 恢复所有字段
	m.inputs[0].SetValue(lastState.name)
	m.inputs[1].SetValue(lastState.token)
	m.inputs[2].SetValue(lastState.baseURL)
	m.inputs[3].SetValue(lastState.websiteURL)
	m.inputs[4].SetValue(lastState.claudeModel)
	m.inputs[5].SetValue(lastState.defaultSonnet)

	return true
}

// loadLiveConfigForForm 从 live 配置文件加载配置（用于表单自动填充）
func (m *Model) loadLiveConfigForForm() (token, baseURL, defaultModel string, loaded bool) {
	// 只支持 Claude 应用的自动加载
	if m.currentApp != "claude" {
		return "", "", "", false
	}

	settingsPath, err := config.GetClaudeSettingsPath()
	if err != nil || !fileExists(settingsPath) {
		return "", "", "", false
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return "", "", "", false
	}

	var liveSettings config.ClaudeSettings
	if err := json.Unmarshal(data, &liveSettings); err != nil {
		return "", "", "", false
	}

	// 提取配置
	token = liveSettings.Env.AnthropicAuthToken
	baseURL = liveSettings.Env.AnthropicBaseURL
	defaultModel = liveSettings.Env.AnthropicDefaultSonnetModel

	// 只有 token 和 baseURL 都存在才算成功加载
	if token != "" && baseURL != "" {
		return token, baseURL, defaultModel, true
	}

	return "", "", "", false
}

// fileExists 检查文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (m *Model) initForm(provider *config.Provider) {
	m.inputs = make([]textinput.Model, 6)
	m.focusIndex = 0

	// Name (必填)
	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "例如: Anthropic 官方"
	m.inputs[0].Focus()
	m.inputs[0].CharLimit = 50
	m.inputs[0].Width = 50

	// API Token (必填)
	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "sk-ant-xxxxx"
	m.inputs[1].EchoMode = textinput.EchoPassword
	m.inputs[1].CharLimit = 500
	m.inputs[1].Width = 50

	// Base URL (必填)
	m.inputs[2] = textinput.New()
	m.inputs[2].Placeholder = "https://api.anthropic.com"
	m.inputs[2].CharLimit = 200
	m.inputs[2].Width = 50

	// Website URL (可选)
	m.inputs[3] = textinput.New()
	m.inputs[3].Placeholder = "https://example.com"
	m.inputs[3].CharLimit = 200
	m.inputs[3].Width = 50

	// Model (可选)
	m.inputs[4] = textinput.New()
	m.inputs[4].Placeholder = "Default (recommended)"
	m.inputs[4].CharLimit = 100
	m.inputs[4].Width = 50

	// Default Sonnet Model (可选)
	m.inputs[5] = textinput.New()
	m.inputs[5].Placeholder = "例如: claude-3-5-sonnet-20241022 (可选)"
	m.inputs[5].CharLimit = 100
	m.inputs[5].Width = 50

	// Fill existing data
	if provider != nil {
		// 编辑模式：使用现有配置
		m.inputs[0].SetValue(provider.Name)

		token := config.ExtractTokenFromProvider(provider)
		baseURL := config.ExtractBaseURLFromProvider(provider)
		claudeModel := config.ExtractModelFromProvider(provider)
		defaultSonnetModel := config.ExtractDefaultSonnetModelFromProvider(provider)

		m.inputs[1].SetValue(token)
		m.inputs[2].SetValue(baseURL)
		m.inputs[3].SetValue(provider.WebsiteURL)
		m.inputs[4].SetValue(claudeModel)
		m.inputs[5].SetValue(defaultSonnetModel)
	} else if m.copyFromProvider != nil {
		// 复制模式：从 copyFromProvider 预填充（除了名称）
		// 名称留空，让用户输入新名称（避免重复）
		m.inputs[0].SetValue("") // 名称留空

		token := config.ExtractTokenFromProvider(m.copyFromProvider)
		baseURL := config.ExtractBaseURLFromProvider(m.copyFromProvider)
		claudeModel := config.ExtractModelFromProvider(m.copyFromProvider)
		defaultSonnetModel := config.ExtractDefaultSonnetModelFromProvider(m.copyFromProvider)

		m.inputs[1].SetValue(token)
		m.inputs[2].SetValue(baseURL)
		m.inputs[3].SetValue(m.copyFromProvider.WebsiteURL)
		m.inputs[4].SetValue(claudeModel)
		m.inputs[5].SetValue(defaultSonnetModel)

		// 复制完成后清空 copyFromProvider，避免影响后续操作
		m.copyFromProvider = nil
	} else {
		// 创建模式：只有当前应用没有配置时才自动加载
		if len(m.providers) == 0 {
			token, baseURL, defaultModel, loaded := m.loadLiveConfigForForm()
			if loaded {
				m.inputs[1].SetValue(token)
				m.inputs[2].SetValue(baseURL)
				if defaultModel != "" {
					m.inputs[4].SetValue(defaultModel)
				}
			}
		}
	}
}

func (m Model) updateInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

// getVersion 获取版本号
func (m Model) getVersion() string {
	return version.GetVersion()
}
