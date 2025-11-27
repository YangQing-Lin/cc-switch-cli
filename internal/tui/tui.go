package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/YangQing-Lin/cc-switch-cli/internal/backup"
	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/portable"
	"github.com/YangQing-Lin/cc-switch-cli/internal/template"
	"github.com/YangQing-Lin/cc-switch-cli/internal/version"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
	currentApp          string              // "claude" or "codex" or "gemini"
	appCursor           int                 // 应用选择光标 (0=Claude, 1=Codex, 2=Gemini)
	lastModTime         time.Time           // 配置文件最后修改时间
	configPath          string              // 配置文件路径
	configCorrupted     bool                // 配置文件是否损坏
	backupList          []backup.BackupInfo // 备份列表
	backupCursor        int                 // 备份列表光标
	modelSelectorActive bool                // 模型选择器是否激活（Claude 模型字段下拉）
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
		values []string
	}

	// 复制配置相关
	copyFromProvider *config.Provider // 从哪个配置复制（用于创建时预填充）

	// 更新相关
	latestRelease *version.ReleaseInfo // 最新版本信息

	// 便携模式相关
	isPortableMode bool // 是否为便携模式

	// API Token 显示状态
	apiTokenVisible bool

	// 三列视图模式相关
	viewMode        string               // "single" 或 "multi" (三列模式)
	columnCursor    int                  // 当前聚焦的列索引 (0=Claude, 1=Codex, 2=Gemini)
	columnCursors   [3]int               // 每列独立的行光标位置
	columnProviders [3][]config.Provider // 三列各自的配置列表缓存

	// MCP 管理相关字段
	mcpServers      []config.McpServer // MCP 服务器列表
	mcpCursor       int                // MCP 列表光标
	mcpMode         string             // MCP 子模式: "list", "add", "edit", "delete", "apps_toggle", "preset"
	selectedMcp     *config.McpServer  // 当前选中的 MCP 服务器
	mcpInputs       []textinput.Model  // MCP 表单输入框
	mcpFocusIndex   int                // MCP 表单焦点索引
	mcpConnType     string             // 连接类型: "stdio", "http", "sse"
	mcpAppsToggle   config.McpApps     // 应用多选状态
	mcpAppsCursor   int                // 应用多选光标 (0=Claude, 1=Codex, 2=Gemini)
	mcpPresets      []config.McpServer // MCP 预设列表
	mcpPresetCursor int                // 预设列表光标
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
		viewMode:        manager.GetViewMode(), // 从配置加载视图模式
	}
	if initErr != nil {
		m.err = initErr
	}
	m.refreshProviders()

	// 如果是三列模式，初始化所有列的配置缓存
	if m.viewMode == "multi" {
		m.refreshAllColumns()
	}

	return m
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
		case "mcp_manager":
			switch m.mcpMode {
			case "list":
				return m.handleMcpListKeys(msg)
			case "add", "edit":
				handled, newModel, cmd := m.handleMcpFormKeys(msg)
				if handled {
					return newModel, cmd
				}
				return m.updateMcpInputs(msg)
			case "delete":
				return m.handleMcpDeleteKeys(msg)
			case "apps_toggle":
				return m.handleMcpAppsToggleKeys(msg)
			case "preset":
				return m.handleMcpPresetKeys(msg)
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	switch m.mode {
	case "list":
		if m.viewMode == "multi" {
			return m.viewListMulti()
		}
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
	case "mcp_manager":
		switch m.mcpMode {
		case "list":
			return m.viewMcpList()
		case "add", "edit":
			return m.viewMcpForm()
		case "delete":
			return m.viewMcpDelete()
		case "apps_toggle":
			return m.viewMcpAppsToggle()
		case "preset":
			return m.viewMcpPreset()
		}
	}
	return ""
}
