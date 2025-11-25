package tui

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/i18n"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type selectorOption struct {
	Label string
	Value string
}

var (
	claudeUnifiedModelSelectorOptions = []selectorOption{
		{Label: "claude-haiku-4-5-20251001", Value: "claude-haiku-4-5-20251001"},
		{Label: "claude-sonnet-4-5-20250929", Value: "claude-sonnet-4-5-20250929"},
		{Label: "claude-opus-4-5-20251101", Value: "claude-opus-4-5-20251101"},
	}

	codexModelSelectorOptions = []selectorOption{
		{Label: "gpt-5-codex", Value: "gpt-5-codex"},
		{Label: "gpt-5", Value: "gpt-5"},
		{Label: "gpt-5.1-codex", Value: "gpt-5.1-codex"},
		{Label: "gpt-5.1-codex-max", Value: "gpt-5.1-codex-max"},
	}

	codexReasoningSelectorOptions = []selectorOption{
		{Label: "minimal", Value: "minimal"},
		{Label: "low", Value: "low"},
		{Label: "medium", Value: "medium"},
		{Label: "high", Value: "high"},
		{Label: "xhigh", Value: "xhigh"},
	}

	geminiModelSelectorOptions = []selectorOption{
		{Label: "gemini-2.5-pro", Value: "gemini-2.5-pro"},
		{Label: "gemini-2.5-flash", Value: "gemini-2.5-flash"},
		{Label: "gemini-3-pro-preview", Value: "gemini-3-pro-preview"},
	}

	codexConfigBaseURLRegex   = regexp.MustCompile(`base_url\s*=\s*"([^"]+)"`)
	codexConfigModelRegex     = regexp.MustCompile(`model\s*=\s*"([^"]+)"`)
	codexConfigReasoningRegex = regexp.MustCompile(`model_reasoning_effort\s*=\s*"([^"]+)"`)
)

// handleFormKeys handles keys in add/edit mode
func (m Model) handleFormKeys(msg tea.KeyMsg) (bool, tea.Model, tea.Cmd) {
	// Debug: 记录按键
	keyStr := msg.String()

	// 处理选择器激活时的特殊键
	if m.modelSelectorActive {
		switch keyStr {
		case "up":
			options := m.selectorOptions(m.focusIndex)
			if len(options) == 0 {
				return true, m, nil
			}
			if m.modelSelectorCursor > 0 {
				m.modelSelectorCursor--
			}
			return true, m, nil
		case "down":
			options := m.selectorOptions(m.focusIndex)
			if len(options) == 0 {
				return true, m, nil
			}
			if m.modelSelectorCursor < len(options)-1 {
				m.modelSelectorCursor++
			}
			return true, m, nil
		case "enter":
			options := m.selectorOptions(m.focusIndex)
			if len(options) > 0 {
				selected := options[m.modelSelectorCursor]
				m.inputs[m.focusIndex].SetValue(selected.Value)
			}
			m.modelSelectorActive = false
			return true, m, nil
		case "esc", " ":
			// ESC 或 Space 都可以关闭选择器
			m.modelSelectorActive = false
			return true, m, nil
		}
	}

	// 处理选择器字段的激活（Space 键）
	if m.isSelectorField(m.focusIndex) && keyStr == " " && !m.modelSelectorActive {
		options := m.selectorOptions(m.focusIndex)
		if len(options) == 0 {
			// 没有选项时，让 Space 键正常输入空格
			return false, m, nil
		}
		m.modelSelectorActive = true
		currentValue := m.inputs[m.focusIndex].Value()
		idx := findSelectorOptionIndex(options, currentValue)
		if idx < 0 {
			idx = 0
		}
		m.modelSelectorCursor = idx
		m.err = nil
		m.message = "选择器已激活，使用方向键选择，Enter确认"
		return true, m, nil
	}

	switch msg.String() {
	case "ctrl+d":
		m.clearFormFields()
		return true, m, nil
	case "ctrl+l":
		m.apiTokenVisible = !m.apiTokenVisible
		m.applyTokenVisibility()
		if m.apiTokenVisible {
			m.message = "🔓 API Key 已显示"
		} else {
			m.message = "🔒 API Key 已隐藏"
		}
		return true, m, nil
	case "ctrl+z":
		if m.undoLastClear() {
			return true, m, nil
		}
	case "esc":
		if m.modelSelectorActive {
			m.modelSelectorActive = false
			return true, m, nil
		}
		m.mode = "list"
		m.message = ""
		m.err = nil
		return true, m, nil
	case "tab", "shift+tab":
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
		if !m.isSelectorField(m.focusIndex) || !m.modelSelectorActive {
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
		if !m.modelSelectorActive {
			m.submitForm()
			return true, m, nil
		}
	case "left", "right":
		// 让方向键传递给当前输入框，用于移动光标
		if m.focusIndex < len(m.inputs) {
			var cmd tea.Cmd
			m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)
			return true, m, cmd
		}
	}
	return false, m, nil
}

func (m Model) isSelectorField(index int) bool {
	return len(m.selectorOptions(index)) > 0
}

func (m Model) selectorOptions(index int) []selectorOption {
	switch m.currentApp {
	case "claude":
		switch index {
		case 4:
			return claudeUnifiedModelSelectorOptions
		case 5:
			return claudeUnifiedModelSelectorOptions
		case 6:
			return claudeUnifiedModelSelectorOptions
		case 7:
			return claudeUnifiedModelSelectorOptions
		}
	case "codex":
		switch index {
		case 4:
			return codexModelSelectorOptions
		case 5:
			return codexReasoningSelectorOptions
		}
	case "gemini":
		switch index {
		case 3:
			return geminiModelSelectorOptions
		}
	}
	return nil
}

func (m Model) selectorTitle(index int) string {
	switch m.currentApp {
	case "claude":
		if index == 4 {
			return "主模型"
		}
		if index == 5 {
			return "Haiku 默认模型"
		}
		if index == 6 {
			return "Sonnet 默认模型"
		}
		if index == 7 {
			return "Opus 默认模型"
		}
	case "codex":
		if index == 4 {
			return "选择模型"
		}
		if index == 5 {
			return "选择推理强度"
		}
	case "gemini":
		if index == 3 {
			return "选择模型"
		}
	}
	return ""
}

func findSelectorOptionIndex(options []selectorOption, value string) int {
	for i, option := range options {
		if option.Value == value {
			return i
		}
	}
	return -1
}

func (m Model) isReadOnlyField(index int) bool {
	return false
}

func (m *Model) applyTokenVisibility() {
	if len(m.inputs) <= 1 {
		return
	}
	if m.apiTokenVisible {
		m.inputs[1].EchoMode = textinput.EchoNormal
	} else {
		m.inputs[1].EchoMode = textinput.EchoPassword
	}
}

func (m *Model) submitForm() {
	name := m.inputs[0].Value()

	// Gemini 特殊处理：字段映射为 [Name, API Key, Base URL, Model]
	if m.currentApp == "gemini" {
		apiKey := m.inputs[1].Value()
		baseURL := m.inputs[2].Value()
		model := m.inputs[3].Value()

		if name == "" {
			m.err = errors.New(i18n.T("error.name_required"))
			return
		}

		// 检测认证类型：如果名称包含 "google" 或 "oauth"，使用 OAuth
		authType := config.GeminiAuthAPIKey
		nameLower := strings.ToLower(name)
		if strings.Contains(nameLower, "google") && !strings.Contains(nameLower, "gemini") {
			authType = config.GeminiAuthOAuth
		}

		// API Key 模式下需要验证必填字段
		if authType == config.GeminiAuthAPIKey {
			if apiKey == "" {
				m.err = errors.New("API Key 不能为空")
				return
			}
			if baseURL == "" {
				m.err = errors.New(i18n.T("error.base_url_required"))
				return
			}
			if model == "" {
				m.err = errors.New("模型不能为空")
				return
			}
		}

		var err error
		if m.mode == "edit" {
			err = m.manager.UpdateGeminiProvider(m.editName, name, baseURL, apiKey, model, authType)
		} else {
			err = m.manager.AddGeminiProvider(name, baseURL, apiKey, model, authType)
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
		return
	}

	// Claude/Codex 标准处理
	token := m.inputs[1].Value()
	baseURL := m.inputs[2].Value()
	websiteURL := m.inputs[3].Value()
	var claudeModel, defaultHaikuModel, defaultSonnetModel, defaultOpusModel string
	if m.currentApp == "claude" {
		if len(m.inputs) > 4 {
			claudeModel = m.inputs[4].Value()
		}
		if len(m.inputs) > 5 {
			defaultHaikuModel = m.inputs[5].Value()
		}
		if len(m.inputs) > 6 {
			defaultSonnetModel = m.inputs[6].Value()
		}
		if len(m.inputs) > 7 {
			defaultOpusModel = m.inputs[7].Value()
		}
	} else {
		if len(m.inputs) > 4 {
			claudeModel = m.inputs[4].Value()
		}
		if len(m.inputs) > 5 {
			defaultSonnetModel = m.inputs[5].Value()
		}
	}

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

	if m.currentApp == "codex" {
		if claudeModel == "" {
			m.err = errors.New(i18n.T("error.model_required"))
			return
		}
		if defaultSonnetModel == "" {
			m.err = errors.New(i18n.T("error.reasoning_required"))
			return
		}
	}

	var err error
	if m.mode == "edit" {
		err = m.manager.UpdateProviderForApp(m.currentApp, m.editName, name, websiteURL, token, baseURL, "custom", claudeModel, defaultHaikuModel, defaultSonnetModel, defaultOpusModel)
	} else {
		err = m.manager.AddProviderForApp(m.currentApp, name, websiteURL, token, baseURL, "custom", claudeModel, defaultHaikuModel, defaultSonnetModel, defaultOpusModel)
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

func (m Model) viewForm() string {
	var s strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#007AFF")).
		Padding(0, 1)

	if m.mode == "add" {
		s.WriteString(title.Render(fmt.Sprintf("添加新配置 (v%s)", m.getVersion())) + "\n\n")
	} else {
		s.WriteString(title.Render(fmt.Sprintf("编辑配置 (v%s)", m.getVersion())) + "\n\n")
	}

	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3B30")).Bold(true)
		s.WriteString(errStyle.Render("✗ "+m.err.Error()) + "\n\n")
	}

	labels := m.formLabels()
	for i, label := range labels {
		s.WriteString(lipgloss.NewStyle().Bold(true).Render(label+":") + "\n")
		inputView := m.inputs[i].View()
		if i == 1 && m.apiTokenVisible {
			inputView += " 👁"
		}
		if i == m.focusIndex {
			s.WriteString(lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#007AFF")).
				Render(inputView) + "\n\n")
		} else {
			s.WriteString(inputView + "\n\n")
		}
	}

	submitStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#007AFF")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 2).
		Bold(true)
	cancelStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#8E8E93")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 2)

	clearStyle := lipgloss.NewStyle()
	if m.anyFieldHasValue() {
		clearStyle = clearStyle.
			Background(lipgloss.Color("#FF9500")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 2)
	} else {
		clearStyle = clearStyle.
			Background(lipgloss.Color("#E5E5EA")).
			Foreground(lipgloss.Color("#636366")).
			Padding(0, 2)
	}

	undoStyle := lipgloss.NewStyle()
	if len(m.undoHistory) > 0 {
		undoStyle = undoStyle.
			Background(lipgloss.Color("#34C759")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 2)
	} else {
		undoStyle = undoStyle.
			Background(lipgloss.Color("#E5E5EA")).
			Foreground(lipgloss.Color("#636366")).
			Padding(0, 2)
	}

	s.WriteString(submitStyle.Render("保存 (Enter)") + " ")
	s.WriteString(cancelStyle.Render("取消 (ESC)") + " ")
	s.WriteString(clearStyle.Render("清空 (Ctrl+D)") + " ")
	s.WriteString(undoStyle.Render("回退 (Ctrl+Z)") + " ")
	keyState := "隐藏"
	if m.apiTokenVisible {
		keyState = "显示"
	}
	keyStatusStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#5856D6")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 2).
		Bold(true)
	s.WriteString(keyStatusStyle.Render(fmt.Sprintf("Key: %s (Ctrl+L)", keyState)) + "\n\n")

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8E8E93"))
	helpText := "Tab: 下一项 • Shift+Tab: 上一项"
	if m.isSelectorField(m.focusIndex) {
		helpText = "Space: 切换选项列表 • 可直接输入自定义内容"
	}
	s.WriteString(helpStyle.Render(helpText))

	baseView := s.String()

	if m.isSelectorField(m.focusIndex) && m.modelSelectorActive {
		var selectorContent strings.Builder
		selectorTitle := m.selectorTitle(m.focusIndex)
		optionsList := m.selectorOptions(m.focusIndex)

		selectorTitleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#007AFF")).
			Padding(0, 1).
			Render(selectorTitle)
		selectorContent.WriteString(selectorTitleStyle + "\n\n")

		for i, option := range optionsList {
			if i == m.modelSelectorCursor {
				selectedStyle := lipgloss.NewStyle().
					Background(lipgloss.Color("#007AFF")).
					Foreground(lipgloss.Color("#FFFFFF")).
					Bold(true).
					Padding(0, 1)
				selectorContent.WriteString(selectedStyle.Render("● "+option.Label) + "\n")
			} else {
				normalStyle := lipgloss.NewStyle().Padding(0, 1)
				selectorContent.WriteString(normalStyle.Render("○ "+option.Label) + "\n")
			}
		}

		selectorContent.WriteString("\n")
		selectorHelp := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8E8E93")).
			Render("↑/↓: 选择 • Enter: 确认 • Space/ESC: 取消")
		selectorContent.WriteString(selectorHelp)

		selectorPanel := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#007AFF")).
			Background(lipgloss.Color("#FFFFFF")).
			Padding(1, 2).
			Render(selectorContent.String())

		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			selectorPanel,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("#00000080")),
		)
	}

	return baseView
}

func (m Model) formLabels() []string {
	if m.currentApp == "gemini" {
		return []string{"配置名称", "GEMINI_API_KEY", "GOOGLE_GEMINI_BASE_URL", "GEMINI_MODEL"}
	}
	base := []string{"配置名称", "API Key", "Base URL", "网站 (可选)"}
	if m.currentApp == "claude" {
		return append(base, "主模型", "Haiku 默认模型", "Sonnet 默认模型", "Opus 默认模型")
	}
	if m.currentApp == "codex" {
		return append(base, "默认模型（必填）", "推理强度（必填）")
	}
	return append(base, "默认模型（可选）", "Default Sonnet Model (可选)")
}

func (m Model) isDefaultSonnetFieldVisible() bool {
	return m.currentApp == "claude"
}

func (m Model) isCodexReasoningFieldVisible() bool {
	return m.currentApp == "codex"
}

func (m *Model) initForm(provider *config.Provider) {
	fieldCount := 5
	switch m.currentApp {
	case "claude":
		fieldCount = 8 // Name, Token, BaseURL, Website, 主模型, Haiku, Sonnet, Opus
	case "codex":
		if m.isCodexReasoningFieldVisible() {
			fieldCount = 6
		}
	case "gemini":
		fieldCount = 4 // Gemini: Name, API Key, Base URL, Model
	}

	m.inputs = make([]textinput.Model, fieldCount)
	m.focusIndex = 0
	m.modelSelectorActive = false
	m.modelSelectorCursor = 0
	m.apiTokenVisible = false

	m.inputs[0] = textinput.New()
	switch m.currentApp {
	case "codex":
		m.inputs[0].Placeholder = "例如: OpenAI 官方"
	case "gemini":
		m.inputs[0].Placeholder = "例如: Google Gemini"
	default:
		m.inputs[0].Placeholder = "例如: Anthropic 官方"
	}
	m.inputs[0].Focus()
	m.inputs[0].CharLimit = 50
	m.inputs[0].Width = 55

	m.inputs[1] = textinput.New()
	switch m.currentApp {
	case "codex":
		m.inputs[1].Placeholder = "sk-xxxxx"
	case "gemini":
		m.inputs[1].Placeholder = "GEMINI_API_KEY"
	default:
		m.inputs[1].Placeholder = "sk-ant-xxxxx"
	}
	m.inputs[1].EchoMode = textinput.EchoPassword
	m.inputs[1].CharLimit = 500
	m.inputs[1].Width = 55
	m.applyTokenVisibility()

	m.inputs[2] = textinput.New()
	switch m.currentApp {
	case "codex":
		m.inputs[2].Placeholder = "https://pro.privnode.com/v1"
	case "gemini":
		m.inputs[2].Placeholder = "https://generativelanguage.googleapis.com"
	default:
		m.inputs[2].Placeholder = "https://api.anthropic.com"
	}
	m.inputs[2].CharLimit = 200
	m.inputs[2].Width = 55

	m.inputs[3] = textinput.New()
	if m.currentApp == "gemini" {
		m.inputs[3].Placeholder = "gemini-2.5-pro"
		m.inputs[3].CharLimit = 100
	} else {
		m.inputs[3].Placeholder = "https://example.com"
		m.inputs[3].CharLimit = 200
	}
	m.inputs[3].Width = 55

	switch m.currentApp {
	case "claude":
		m.inputs[4] = textinput.New()
		m.inputs[4].Placeholder = "claude-sonnet-4-5-20250929"
		m.inputs[4].CharLimit = 120
		m.inputs[4].Width = 55

		m.inputs[5] = textinput.New()
		m.inputs[5].Placeholder = "claude-haiku-4-5-20251001"
		m.inputs[5].CharLimit = 120
		m.inputs[5].Width = 55

		m.inputs[6] = textinput.New()
		m.inputs[6].Placeholder = "claude-sonnet-4-5-20250929"
		m.inputs[6].CharLimit = 120
		m.inputs[6].Width = 55

		m.inputs[7] = textinput.New()
		m.inputs[7].Placeholder = "claude-opus-4-5-20251101"
		m.inputs[7].CharLimit = 120
		m.inputs[7].Width = 55
	case "codex":
		if fieldCount > 4 {
			m.inputs[4] = textinput.New()
			m.inputs[4].Placeholder = "gpt-5-codex"
			m.inputs[4].CharLimit = 100
			m.inputs[4].Width = 55
		}

		if m.isCodexReasoningFieldVisible() {
			m.inputs[5] = textinput.New()
			m.inputs[5].Placeholder = "minimal/low/medium/high/xhigh"
			m.inputs[5].CharLimit = 100
			m.inputs[5].Width = 55
		}
	case "gemini":
		m.inputs[3].SetValue("gemini-2.5-pro")
	}

	if m.currentApp == "codex" && fieldCount > 4 {
		m.inputs[4].SetValue("gpt-5-codex")
		if len(m.inputs) > 5 {
			m.inputs[5].SetValue("high")
		}
	}

	if provider != nil {
		m.inputs[0].SetValue(provider.Name)

		if m.currentApp == "gemini" {
			// Gemini 特殊处理：字段映射为 [Name, API Key, Base URL, Model]
			baseURL, apiKey, model, _ := config.ExtractGeminiConfigFromProvider(provider)
			m.inputs[1].SetValue(apiKey)
			m.inputs[2].SetValue(baseURL)
			m.inputs[3].SetValue(model)
		} else {
			// Claude/Codex 标准处理
			token := config.ExtractTokenFromProvider(provider)
			baseURL := config.ExtractBaseURLFromProvider(provider)
			modelValue := config.ExtractModelFromProvider(provider)
			haikuModel := config.ExtractDefaultHaikuModelFromProvider(provider)
			sonnetModel := config.ExtractDefaultSonnetModelFromProvider(provider)
			opusModel := config.ExtractDefaultOpusModelFromProvider(provider)

			m.inputs[1].SetValue(token)
			m.inputs[2].SetValue(baseURL)
			m.inputs[3].SetValue(provider.WebsiteURL)
			if m.currentApp == "claude" {
				if len(m.inputs) > 4 {
					m.inputs[4].SetValue(modelValue)
				}
				if len(m.inputs) > 5 {
					m.inputs[5].SetValue(haikuModel)
				}
				if len(m.inputs) > 6 {
					m.inputs[6].SetValue(sonnetModel)
				}
				if len(m.inputs) > 7 {
					m.inputs[7].SetValue(opusModel)
				}
			} else if m.isCodexReasoningFieldVisible() {
				if fieldCount > 4 {
					m.inputs[4].SetValue(modelValue)
				}
				if len(m.inputs) > 5 {
					m.inputs[5].SetValue(config.ExtractCodexReasoningFromProvider(provider))
				}
			}
		}
	} else if m.copyFromProvider != nil {
		m.inputs[0].SetValue("")

		if m.currentApp == "gemini" {
			// Gemini 特殊处理：字段映射为 [Name, API Key, Base URL, Model]
			baseURL, apiKey, model, _ := config.ExtractGeminiConfigFromProvider(m.copyFromProvider)
			m.inputs[1].SetValue(apiKey)
			m.inputs[2].SetValue(baseURL)
			m.inputs[3].SetValue(model)
		} else {
			// Claude/Codex 标准处理
			token := config.ExtractTokenFromProvider(m.copyFromProvider)
			baseURL := config.ExtractBaseURLFromProvider(m.copyFromProvider)
			modelValue := config.ExtractModelFromProvider(m.copyFromProvider)
			haikuModel := config.ExtractDefaultHaikuModelFromProvider(m.copyFromProvider)
			sonnetModel := config.ExtractDefaultSonnetModelFromProvider(m.copyFromProvider)
			opusModel := config.ExtractDefaultOpusModelFromProvider(m.copyFromProvider)

			m.inputs[1].SetValue(token)
			m.inputs[2].SetValue(baseURL)
			m.inputs[3].SetValue(m.copyFromProvider.WebsiteURL)
			if m.currentApp == "claude" {
				if len(m.inputs) > 4 {
					m.inputs[4].SetValue(modelValue)
				}
				if len(m.inputs) > 5 {
					m.inputs[5].SetValue(haikuModel)
				}
				if len(m.inputs) > 6 {
					m.inputs[6].SetValue(sonnetModel)
				}
				if len(m.inputs) > 7 {
					m.inputs[7].SetValue(opusModel)
				}
			} else if m.isCodexReasoningFieldVisible() {
				if fieldCount > 4 {
					m.inputs[4].SetValue(modelValue)
				}
				if len(m.inputs) > 5 {
					m.inputs[5].SetValue(config.ExtractCodexReasoningFromProvider(m.copyFromProvider))
				}
			}
		}

		m.copyFromProvider = nil
	} else {
		if len(m.providers) == 0 {
			switch m.currentApp {
			case "claude":
				token, baseURL, primaryModel, haikuModel, sonnetModel, opusModel, loaded := m.loadLiveConfigForForm()
				if loaded {
					m.inputs[1].SetValue(token)
					m.inputs[2].SetValue(baseURL)
					if len(m.inputs) > 4 && primaryModel != "" {
						m.inputs[4].SetValue(primaryModel)
					}
					if len(m.inputs) > 5 && haikuModel != "" {
						m.inputs[5].SetValue(haikuModel)
					}
					if len(m.inputs) > 6 && sonnetModel != "" {
						m.inputs[6].SetValue(sonnetModel)
					}
					if len(m.inputs) > 7 && opusModel != "" {
						m.inputs[7].SetValue(opusModel)
					}
					m.message = "💡 已从 ~/.claude/settings.json 预填充配置"
				}
			case "codex":
				token, baseURL, modelValue, reasoningValue, loaded := m.loadCodexConfigForForm()
				if loaded {
					if token != "" {
						m.inputs[1].SetValue(token)
					}
					if baseURL != "" {
						m.inputs[2].SetValue(baseURL)
					}
					if modelValue != "" && len(m.inputs) > 4 {
						m.inputs[4].SetValue(modelValue)
					}
					if reasoningValue != "" && len(m.inputs) > 5 {
						m.inputs[5].SetValue(reasoningValue)
					}
					m.message = "💡 已从 ~/.codex/config.toml 预填充配置"
				}
			}
		}
	}
}

func (m Model) updateInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		if i == m.focusIndex && m.isReadOnlyField(i) {
			if keyMsg, ok := msg.(tea.KeyMsg); ok {
				switch keyMsg.Type {
				case tea.KeyRunes, tea.KeyBackspace, tea.KeyDelete, tea.KeySpace:
					m.err = errors.New(i18n.T("error.readonly_field"))
					continue
				}
			}
		}
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) clearFormFields() {
	if m.anyFieldHasValue() {
		state := struct {
			values []string
		}{
			values: make([]string, len(m.inputs)),
		}
		for i := range m.inputs {
			state.values[i] = m.inputs[i].Value()
		}
		m.undoHistory = append(m.undoHistory, state)
	}

	for i := range m.inputs {
		m.inputs[i].SetValue("")
	}

	if m.currentApp == "codex" {
		if len(m.inputs) > 4 {
			m.inputs[4].SetValue("gpt-5-codex")
		}
		if len(m.inputs) > 5 {
			m.inputs[5].SetValue("high")
		}
	}

	m.applyTokenVisibility()
}

func (m *Model) anyFieldHasValue() bool {
	for i := range m.inputs {
		if m.inputs[i].Value() != "" {
			return true
		}
	}
	return false
}

func (m *Model) undoLastClear() bool {
	if len(m.undoHistory) == 0 {
		return false
	}

	lastState := m.undoHistory[len(m.undoHistory)-1]
	m.undoHistory = m.undoHistory[:len(m.undoHistory)-1]

	for i := range m.inputs {
		if i < len(lastState.values) {
			m.inputs[i].SetValue(lastState.values[i])
		}
	}

	return true
}

func (m *Model) loadLiveConfigForForm() (token, baseURL, primaryModel, haikuModel, sonnetModel, opusModel string, loaded bool) {
	if m.currentApp != "claude" {
		return "", "", "", "", "", "", false
	}

	settingsPath, err := config.GetClaudeSettingsPath()
	if err != nil || !fileExists(settingsPath) {
		return "", "", "", "", "", "", false
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return "", "", "", "", "", "", false
	}

	var liveSettings config.ClaudeSettings
	if err := json.Unmarshal(data, &liveSettings); err != nil {
		return "", "", "", "", "", "", false
	}

	token = liveSettings.Env.AnthropicAuthToken
	baseURL = liveSettings.Env.AnthropicBaseURL
	primaryModel = liveSettings.Env.AnthropicModel
	haikuModel = liveSettings.Env.AnthropicDefaultHaikuModel
	sonnetModel = liveSettings.Env.AnthropicDefaultSonnetModel
	opusModel = liveSettings.Env.AnthropicDefaultOpusModel

	if token != "" || baseURL != "" || primaryModel != "" || haikuModel != "" || sonnetModel != "" || opusModel != "" {
		return token, baseURL, primaryModel, haikuModel, sonnetModel, opusModel, true
	}

	return "", "", "", "", "", "", false
}

func (m *Model) loadCodexConfigForForm() (token, baseURL, modelValue, reasoningValue string, loaded bool) {
	if m.currentApp != "codex" {
		return "", "", "", "", false
	}

	authPath, err := config.GetCodexAuthJsonPath()
	if err == nil && fileExists(authPath) {
		if data, err := os.ReadFile(authPath); err == nil {
			var auth config.CodexAuthJson
			if err := json.Unmarshal(data, &auth); err == nil {
				token = auth.OpenAIAPIKey
			}
		}
	}

	configPath, err := config.GetCodexConfigPath()
	if err == nil && fileExists(configPath) {
		if data, err := os.ReadFile(configPath); err == nil {
			configContent := string(data)
			if matches := codexConfigBaseURLRegex.FindStringSubmatch(configContent); len(matches) > 1 {
				baseURL = matches[1]
			}
			if matches := codexConfigModelRegex.FindStringSubmatch(configContent); len(matches) > 1 {
				modelValue = matches[1]
			}
			if matches := codexConfigReasoningRegex.FindStringSubmatch(configContent); len(matches) > 1 {
				reasoningValue = matches[1]
			}
		}
	}

	if token != "" || baseURL != "" || modelValue != "" || reasoningValue != "" {
		return token, baseURL, modelValue, reasoningValue, true
	}

	return "", "", "", "", false
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
