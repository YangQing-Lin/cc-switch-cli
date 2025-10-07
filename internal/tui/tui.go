package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/i18n"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model TUI 主模型
type Model struct {
	manager    *config.Manager
	providers  []config.Provider
	cursor     int
	width      int
	height     int
	err        error
	message    string
	mode       string // "list", "add", "edit", "delete", "app_select"
	editName   string // 正在编辑的配置名称
	deleteName string // 要删除的配置名称
	inputs     []textinput.Model
	focusIndex int
	currentApp string // "claude" or "codex"
	appCursor  int    // 应用选择光标 (0=Claude, 1=Codex)
}

// New 创建新的 TUI 模型
func New(manager *config.Manager) Model {
	m := Model{
		manager:    manager,
		mode:       "list",
		currentApp: "claude", // 默认显示 Claude
		appCursor:  0,
	}
	m.refreshProviders()
	return m
}

func (m *Model) refreshProviders() {
	m.providers = m.manager.ListProvidersForApp(m.currentApp)
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

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
			if current == nil || provider.ID != current.ID {
				err := m.manager.SwitchProviderForApp(m.currentApp, provider.Name)
				if err != nil {
					m.err = err
					m.message = ""
				} else {
					m.message = i18n.T("success.switched_to") + ": " + provider.Name
					m.err = nil
					m.refreshProviders()
				}
			}
		}
	case "a":
		m.mode = "add"
		m.editName = ""
		m.initForm(nil)
		return m, textinput.Blink
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
	}
	return m, nil
}

// Form handlers - 返回 (handled, model, cmd)
func (m Model) handleFormKeys(msg tea.KeyMsg) (bool, tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = "list"
		m.message = ""
		m.err = nil
		return true, m, nil
	case "tab", "shift+tab", "up", "down":
		if msg.String() == "up" || msg.String() == "shift+tab" {
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
	case "enter", "ctrl+s":
		m.submitForm()
		return true, m, nil
	}
	// 未处理,返回 false
	return false, m, nil
}

func (m *Model) submitForm() {
	name := m.inputs[0].Value()
	websiteURL := m.inputs[1].Value()
	token := m.inputs[2].Value()
	baseURL := m.inputs[3].Value()
	defaultSonnetModel := m.inputs[4].Value()

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
		err = m.manager.UpdateProviderForApp(m.currentApp, m.editName, name, websiteURL, token, baseURL, "custom", defaultSonnetModel)
	} else {
		// Add provider
		err = m.manager.AddProviderForApp(m.currentApp, name, websiteURL, token, baseURL, "custom", defaultSonnetModel)
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

	// Title with current app indicator
	appName := "Claude Code"
	if m.currentApp == "codex" {
		appName = "Codex CLI"
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#007AFF")).
		Padding(0, 1).
		Render(fmt.Sprintf("CC Switch - %s 配置管理", appName))
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
		"e: 编辑",
		"d: 删除",
		"t: 切换应用",
		"c: Claude",
		"x: Codex",
		"r: 刷新",
		"q: 退出",
	}
	s.WriteString(helpStyle.Render(strings.Join(helps, " • ")))

	return s.String()
}

// viewAppSelect renders the app selection screen
func (m Model) viewAppSelect() string {
	var s strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#007AFF")).
		Padding(0, 1).
		Render("选择应用")
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
	var s strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#007AFF")).
		Padding(0, 1)

	if m.mode == "add" {
		s.WriteString(title.Render("添加新配置") + "\n\n")
	} else {
		s.WriteString(title.Render("编辑配置") + "\n\n")
	}

	// Error
	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3B30")).Bold(true)
		s.WriteString(errStyle.Render("✗ "+m.err.Error()) + "\n\n")
	}

	// Form
	labels := []string{"配置名称", "网站 (可选)", "API Token", "Base URL", "Default Sonnet Model (可选)"}
	for i, label := range labels {
		s.WriteString(lipgloss.NewStyle().Bold(true).Render(label+":") + "\n")
		if i == m.focusIndex {
			s.WriteString(lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#007AFF")).
				Render(m.inputs[i].View()) + "\n\n")
		} else {
			s.WriteString(m.inputs[i].View() + "\n\n")
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

	s.WriteString(submitStyle.Render("保存 (Enter)") + " ")
	s.WriteString(cancelStyle.Render("取消 (ESC)") + "\n\n")

	// Help
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8E8E93"))
	s.WriteString(helpStyle.Render("Tab: 下一项 • Shift+Tab: 上一项"))

	return s.String()
}

func (m Model) viewDelete() string {
	var s strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#007AFF")).
		Padding(0, 1).
		Render("确认删除")
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

// Helper functions
func (m *Model) initForm(provider *config.Provider) {
	m.inputs = make([]textinput.Model, 5)
	m.focusIndex = 0

	// Name
	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "例如: Anthropic 官方"
	m.inputs[0].Focus()
	m.inputs[0].CharLimit = 50
	m.inputs[0].Width = 50

	// Website URL
	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "https://example.com"
	m.inputs[1].CharLimit = 200
	m.inputs[1].Width = 50

	// API Token
	m.inputs[2] = textinput.New()
	m.inputs[2].Placeholder = "sk-ant-xxxxx"
	m.inputs[2].EchoMode = textinput.EchoPassword
	m.inputs[2].CharLimit = 500
	m.inputs[2].Width = 50

	// Base URL
	m.inputs[3] = textinput.New()
	m.inputs[3].Placeholder = "https://api.anthropic.com"
	m.inputs[3].CharLimit = 200
	m.inputs[3].Width = 50

	// Default Sonnet Model (optional, Claude only)
	m.inputs[4] = textinput.New()
	m.inputs[4].Placeholder = "例如: claude-3-5-sonnet-20241022 (可选)"
	m.inputs[4].CharLimit = 100
	m.inputs[4].Width = 50

	// Fill existing data
	if provider != nil {
		m.inputs[0].SetValue(provider.Name)
		m.inputs[1].SetValue(provider.WebsiteURL)

		token := config.ExtractTokenFromProvider(provider)
		baseURL := config.ExtractBaseURLFromProvider(provider)
		defaultSonnetModel := config.ExtractDefaultSonnetModelFromProvider(provider)

		m.inputs[2].SetValue(token)
		m.inputs[3].SetValue(baseURL)
		m.inputs[4].SetValue(defaultSonnetModel)
	}
}

func (m Model) updateInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}
