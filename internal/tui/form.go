package tui

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/i18n"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// handleFormKeys handles keys in add/edit mode
func (m Model) handleFormKeys(msg tea.KeyMsg) (bool, tea.Model, tea.Cmd) {
	if m.focusIndex == 4 || m.focusIndex == 5 {
		switch msg.String() {
		case "right":
			if !m.modelSelectorActive {
				m.modelSelectorActive = true
				m.modelSelectorCursor = 0
				return true, m, nil
			}
		case "left":
			if m.modelSelectorActive {
				m.modelSelectorActive = false
				return true, m, nil
			}
		case "up":
			if m.modelSelectorActive {
				if m.modelSelectorCursor > 0 {
					m.modelSelectorCursor--
				}
				return true, m, nil
			}
		case "down":
			if m.modelSelectorActive {
				var maxCursor int
				if m.focusIndex == 4 {
					maxCursor = len(predefinedClaudeModels) - 1
				} else {
					maxCursor = len(predefinedModels) - 1
				}
				if m.modelSelectorCursor < maxCursor {
					m.modelSelectorCursor++
				}
				return true, m, nil
			}
		case "enter":
			if m.modelSelectorActive {
				var selectedModel string
				if m.focusIndex == 4 {
					selectedModel = predefinedClaudeModels[m.modelSelectorCursor]
				} else {
					if m.modelSelectorCursor == 0 {
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
		m.clearFormFields()
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
		if !m.modelSelectorActive {
			m.submitForm()
			return true, m, nil
		}
	}
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
		err = m.manager.UpdateProviderForApp(m.currentApp, m.editName, name, websiteURL, token, baseURL, "custom", claudeModel, defaultSonnetModel)
	} else {
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

	labels := []string{"配置名称", "API Token", "Base URL", "网站 (可选)", "默认模型（可选）", "Default Sonnet Model (可选)"}
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
	s.WriteString(clearStyle.Render("清空内容 (Ctrl+D)") + " ")
	s.WriteString(undoStyle.Render("回退 (Ctrl+Z)") + "\n\n")

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8E8E93"))
	helpText := "Tab: 下一项 • Shift+Tab: 上一项"
	if m.focusIndex == 4 || m.focusIndex == 5 {
		helpText = "→: 显示模型选项 • ←: 隐藏模型选项 • Tab: 下一项"
	}
	s.WriteString(helpStyle.Render(helpText))

	if (m.focusIndex == 4 || m.focusIndex == 5) && m.modelSelectorActive {
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

		selectorPanel := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#007AFF")).
			Padding(1, 2).
			Render(selectorContent.String())

		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			s.String(),
			"  ",
			selectorPanel,
		)
	}

	return s.String()
}

func (m *Model) initForm(provider *config.Provider) {
	m.inputs = make([]textinput.Model, 6)
	m.focusIndex = 0

	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "例如: Anthropic 官方"
	m.inputs[0].Focus()
	m.inputs[0].CharLimit = 50
	m.inputs[0].Width = 50

	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "sk-ant-xxxxx"
	m.inputs[1].EchoMode = textinput.EchoPassword
	m.inputs[1].CharLimit = 500
	m.inputs[1].Width = 50

	m.inputs[2] = textinput.New()
	m.inputs[2].Placeholder = "https://api.anthropic.com"
	m.inputs[2].CharLimit = 200
	m.inputs[2].Width = 50

	m.inputs[3] = textinput.New()
	m.inputs[3].Placeholder = "https://example.com"
	m.inputs[3].CharLimit = 200
	m.inputs[3].Width = 50

	m.inputs[4] = textinput.New()
	m.inputs[4].Placeholder = "Default (recommended)"
	m.inputs[4].CharLimit = 100
	m.inputs[4].Width = 50

	m.inputs[5] = textinput.New()
	m.inputs[5].Placeholder = "例如: claude-3-5-sonnet-20241022 (可选)"
	m.inputs[5].CharLimit = 100
	m.inputs[5].Width = 50

	if provider != nil {
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
		m.inputs[0].SetValue("")

		token := config.ExtractTokenFromProvider(m.copyFromProvider)
		baseURL := config.ExtractBaseURLFromProvider(m.copyFromProvider)
		claudeModel := config.ExtractModelFromProvider(m.copyFromProvider)
		defaultSonnetModel := config.ExtractDefaultSonnetModelFromProvider(m.copyFromProvider)

		m.inputs[1].SetValue(token)
		m.inputs[2].SetValue(baseURL)
		m.inputs[3].SetValue(m.copyFromProvider.WebsiteURL)
		m.inputs[4].SetValue(claudeModel)
		m.inputs[5].SetValue(defaultSonnetModel)

		m.copyFromProvider = nil
	} else {
		if len(m.providers) == 0 {
			token, baseURL, defaultModel, loaded := m.loadLiveConfigForForm()
			if loaded {
				m.inputs[1].SetValue(token)
				m.inputs[2].SetValue(baseURL)
				if defaultModel != "" {
					m.inputs[4].SetValue(defaultModel)
				}
				m.message = "💡 已从 ~/.claude/settings.json 预填充配置"
			}
		}
	}
}

func (m Model) updateInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		if i == m.focusIndex && i == 4 {
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

	m.inputs[0].SetValue("")
	m.inputs[1].SetValue("")
	m.inputs[2].SetValue("")
	m.inputs[3].SetValue("")
	m.inputs[4].SetValue("")
	m.inputs[5].SetValue("")
}

func (m *Model) anyFieldHasValue() bool {
	return m.inputs[0].Value() != "" ||
		m.inputs[1].Value() != "" ||
		m.inputs[2].Value() != "" ||
		m.inputs[3].Value() != "" ||
		m.inputs[4].Value() != "" ||
		m.inputs[5].Value() != ""
}

func (m *Model) undoLastClear() bool {
	if len(m.undoHistory) == 0 {
		return false
	}

	lastState := m.undoHistory[len(m.undoHistory)-1]
	m.undoHistory = m.undoHistory[:len(m.undoHistory)-1]

	m.inputs[0].SetValue(lastState.name)
	m.inputs[1].SetValue(lastState.token)
	m.inputs[2].SetValue(lastState.baseURL)
	m.inputs[3].SetValue(lastState.websiteURL)
	m.inputs[4].SetValue(lastState.claudeModel)
	m.inputs[5].SetValue(lastState.defaultSonnet)

	return true
}

func (m *Model) loadLiveConfigForForm() (token, baseURL, defaultModel string, loaded bool) {
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

	token = liveSettings.Env.AnthropicAuthToken
	baseURL = liveSettings.Env.AnthropicBaseURL
	defaultModel = liveSettings.Env.AnthropicDefaultSonnetModel

	if token != "" && baseURL != "" {
		return token, baseURL, defaultModel, true
	}

	return "", "", "", false
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
