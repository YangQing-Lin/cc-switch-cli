package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/template"
	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func makeKey(key string) tea.KeyMsg {
	switch key {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "pgup":
		return tea.KeyMsg{Type: tea.KeyPgUp}
	case "pgdown":
		return tea.KeyMsg{Type: tea.KeyPgDown}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "delete":
		return tea.KeyMsg{Type: tea.KeyDelete}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
}

func TestHandleListKeysBranches(t *testing.T) {
	cases := []struct {
		name   string
		key    string
		setup  func(m *Model)
		verify func(t *testing.T, m Model)
	}{
		{
			name: "move_up_at_top",
			key:  "=",
			setup: func(m *Model) {
				m.cursor = 0
			},
			verify: func(t *testing.T, m Model) {
				if !strings.Contains(m.message, "顶部") {
					t.Fatalf("expected top message")
				}
			},
		},
		{
			name: "move_down_at_bottom",
			key:  "-",
			setup: func(m *Model) {
				m.cursor = len(m.providers) - 1
			},
			verify: func(t *testing.T, m Model) {
				if !strings.Contains(m.message, "底部") {
					t.Fatalf("expected bottom message")
				}
			},
		},
		{
			name: "move_up_success",
			key:  "=",
			setup: func(m *Model) {
				m.cursor = 1
			},
			verify: func(t *testing.T, m Model) {
				if !strings.Contains(m.message, "上调") {
					t.Fatalf("expected move up message")
				}
			},
		},
		{
			name: "move_down_success",
			key:  "-",
			setup: func(m *Model) {
				m.cursor = 0
			},
			verify: func(t *testing.T, m Model) {
				if !strings.Contains(m.message, "下调") {
					t.Fatalf("expected move down message")
				}
			},
		},
		{
			name: "switch_provider",
			key:  "enter",
			setup: func(m *Model) {
				m.cursor = 1
			},
			verify: func(t *testing.T, m Model) {
				if !strings.Contains(m.message, "切换") {
					t.Fatalf("expected switch message")
				}
			},
		},
		{
			name:   "copy_provider",
			key:    "C",
			verify: func(t *testing.T, _ Model) {},
		},
		{
			name: "edit_provider",
			key:  "e",
			verify: func(t *testing.T, m Model) {
				if m.mode != "edit" {
					t.Fatalf("expected edit mode")
				}
			},
		},
		{
			name: "delete_current_error",
			key:  "d",
			setup: func(m *Model) {
				m.cursor = 0
			},
			verify: func(t *testing.T, m Model) {
				if m.err == nil {
					t.Fatalf("expected delete error")
				}
			},
		},
		{
			name: "delete_non_current",
			key:  "d",
			setup: func(m *Model) {
				m.cursor = 1
			},
			verify: func(t *testing.T, m Model) {
				if m.mode != "delete" {
					t.Fatalf("expected delete mode")
				}
			},
		},
		{
			name: "refresh_list",
			key:  "r",
			verify: func(t *testing.T, m Model) {
				if !strings.Contains(m.message, "刷新") {
					t.Fatalf("expected refresh message")
				}
			},
		},
		{
			name: "switch_to_codex",
			key:  "x",
			verify: func(t *testing.T, m Model) {
				if m.currentApp != "codex" {
					t.Fatalf("expected codex app")
				}
			},
		},
		{
			name: "check_updates",
			key:  "u",
			verify: func(t *testing.T, m Model) {
				if !strings.Contains(m.message, "检查更新") {
					t.Fatalf("expected update message")
				}
			},
		},
		{
			name: "download_without_release",
			key:  "U",
			verify: func(t *testing.T, m Model) {
				if m.err == nil {
					t.Fatalf("expected update error")
				}
			},
		},
		{
			name: "template_manager_missing",
			key:  "m",
			verify: func(t *testing.T, m Model) {
				if m.err == nil {
					t.Fatalf("expected template manager error")
				}
			},
		},
		{
			name: "mcp_manager",
			key:  "M",
			verify: func(t *testing.T, m Model) {
				if m.mode != "mcp_manager" {
					t.Fatalf("expected mcp manager")
				}
			},
		},
		{
			name: "backup_create",
			key:  "b",
			verify: func(t *testing.T, m Model) {
				if m.message == "" && m.err == nil {
					t.Fatalf("expected backup result")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manager := newTestManager(t)
			addProvider(t, manager, "claude", "p1", "Claude One")
			addProvider(t, manager, "claude", "p2", "Claude Two")
			m := Model{manager: manager, mode: "list", currentApp: "claude", configPath: manager.GetConfigPath()}
			m.refreshProviders()
			if tc.setup != nil {
				tc.setup(&m)
			}
			updated, _ := m.handleListKeys(makeKey(tc.key))
			final := updated.(Model)
			tc.verify(t, final)
		})
	}
}

func TestHandleMultiColumnKeysBranches(t *testing.T) {
	cases := []struct {
		name   string
		keys   []string
		setup  func(m *Model)
		verify func(t *testing.T, m Model)
	}{
		{
			name: "navigation_and_switch",
			keys: []string{"tab", "left", "right", "down", "up", "enter"},
			verify: func(t *testing.T, m Model) {
				if m.message == "" {
					t.Fatalf("expected switch message")
				}
			},
		},
		{
			name: "move_and_delete",
			keys: []string{"=", "-", "d"},
			setup: func(m *Model) {
				m.columnCursor = 0
				m.columnCursors[0] = 1
			},
			verify: func(t *testing.T, m Model) {
				if m.mode != "delete" && m.err == nil {
					t.Fatalf("expected delete path")
				}
			},
		},
		{
			name:   "add_edit_copy",
			keys:   []string{"a", "esc", "e", "esc", "C"},
			verify: func(t *testing.T, _ Model) {},
		},
		{
			name: "backup_list_and_return",
			keys: []string{"b", "l", "esc"},
			verify: func(t *testing.T, m Model) {
				if m.mode != "list" {
					t.Fatalf("expected to return list")
				}
			},
		},
		{
			name: "template_manager",
			keys: []string{"m"},
			setup: func(m *Model) {
				m.templateManager = newTestTemplateManager(t)
			},
			verify: func(t *testing.T, m Model) {
				if m.mode != "template_manager" {
					t.Fatalf("expected template manager mode")
				}
			},
		},
		{
			name: "mcp_manager",
			keys: []string{"M"},
			verify: func(t *testing.T, m Model) {
				if m.mode != "mcp_manager" {
					t.Fatalf("expected mcp manager mode")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manager := newTestManager(t)
			addProvider(t, manager, "claude", "c1", "Claude One")
			addProvider(t, manager, "claude", "c2", "Claude Two")
			addProvider(t, manager, "codex", "x1", "Codex One")
			addProvider(t, manager, "gemini", "g1", "Gemini One")
			m := Model{manager: manager, mode: "list", viewMode: "multi", currentApp: "claude", configPath: manager.GetConfigPath()}
			m.refreshAllColumns()
			if tc.setup != nil {
				tc.setup(&m)
			}
			updated := testutil.BubbleTeaTestHelper(t, m, tc.keys)
			final := updated.(Model)
			tc.verify(t, final)
		})
	}
}

func TestFormBranchesExtra(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "selector_titles_and_labels",
			run: func(t *testing.T) {
				m := Model{currentApp: "claude"}
				if m.selectorTitle(4) == "" || m.selectorTitle(5) == "" {
					t.Fatalf("expected selector titles")
				}
				m.currentApp = "codex"
				if m.selectorTitle(4) == "" || m.selectorTitle(5) == "" {
					t.Fatalf("expected codex selector titles")
				}
				m.currentApp = "gemini"
				if m.selectorTitle(3) == "" {
					t.Fatalf("expected gemini selector title")
				}

				m.currentApp = "claude"
				labels := m.formLabels()
				if len(labels) == 0 || !strings.Contains(labels[len(labels)-1], "Opus") {
					t.Fatalf("expected claude labels")
				}
				m.currentApp = "codex"
				labels = m.formLabels()
				if len(labels) < 5 {
					t.Fatalf("expected codex labels")
				}
				m.currentApp = "gemini"
				labels = m.formLabels()
				if len(labels) != 4 {
					t.Fatalf("expected gemini labels")
				}
			},
		},
		{
			name: "token_visibility_and_update_inputs",
			run: func(t *testing.T) {
				m := Model{currentApp: "claude", mode: "add"}
				m.initForm(nil)
				m.apiTokenVisible = true
				m.applyTokenVisibility()
				m.apiTokenVisible = false
				m.applyTokenVisibility()

				m.focusIndex = 0
				updated, _ := m.updateInputs(makeKey("a"))
				m = updated.(Model)
				if m.inputs[0].Value() == "" {
					t.Fatalf("expected input update")
				}
			},
		},
		{
			name: "handle_form_keys_paths",
			run: func(t *testing.T) {
				m := Model{manager: newTestManager(t), currentApp: "claude", mode: "add"}
				m.initForm(nil)
				m.inputs[0].SetValue("Name")
				m.inputs[1].SetValue("token")
				m.inputs[2].SetValue("https://api.anthropic.com")

				_, updated, _ := m.handleFormKeys(makeKey("tab"))
				m = updated.(Model)
				_, updated, _ = m.handleFormKeys(makeKey("shift+tab"))
				m = updated.(Model)
				_, updated, _ = m.handleFormKeys(makeKey("left"))
				m = updated.(Model)
				_, updated, _ = m.handleFormKeys(makeKey("ctrl+l"))
				m = updated.(Model)
				_, updated, _ = m.handleFormKeys(makeKey("ctrl+s"))
				m = updated.(Model)
				if m.mode != "list" {
					t.Fatalf("expected submit to return list")
				}

				m.mode = "add"
				m.initForm(nil)
				m.modelSelectorActive = true
				_, updated, _ = m.handleFormKeys(makeKey("esc"))
				m = updated.(Model)
				if m.modelSelectorActive {
					t.Fatalf("expected selector close")
				}

				m.mode = "add"
				m.initForm(nil)
				_, updated, _ = m.handleFormKeys(makeKey("esc"))
				m = updated.(Model)
				if m.mode != "list" {
					t.Fatalf("expected esc to list")
				}
			},
		},
		{
			name: "selector_overlay_view",
			run: func(t *testing.T) {
				m := Model{currentApp: "codex", mode: "add", width: 80, height: 24}
				m.initForm(nil)
				m.focusIndex = 4
				m.modelSelectorActive = true
				view := m.viewForm()
				if !strings.Contains(view, "选择模型") {
					t.Fatalf("expected selector overlay")
				}
			},
		},
		{
			name: "default_sonnet_visibility",
			run: func(t *testing.T) {
				m := Model{currentApp: "claude"}
				if !m.isDefaultSonnetFieldVisible() {
					t.Fatalf("expected sonnet visible")
				}
				m.currentApp = "codex"
				if m.isDefaultSonnetFieldVisible() {
					t.Fatalf("expected sonnet hidden")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.run(t)
		})
	}
}

func TestTemplateAndMcpExtraBranches(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "template_views_and_errors",
			run: func(t *testing.T) {
				testutil.WithTempHome(t, func(home string) {
					testutil.WithTempCWD(t, func(_ string) {
						tm := newTestTemplateManager(t)
						id, _ := tm.AddTemplate("User", template.CategoryClaudeMd, "content")
						tpl, _ := tm.GetTemplate(id)

						m := Model{manager: newTestManager(t), mode: "template_manager", templateMode: "list", templateManager: tm}
						m.templates = []template.Template{{ID: "builtin", Name: "Builtin", Category: template.CategoryClaudeMd, IsBuiltin: true}, *tpl}

						updated, _ := m.handleTemplateListKeys(makeKey("p"))
						m = updated.(Model)
						if m.templateMode != "preview" {
							t.Fatalf("expected preview mode")
						}

						m.templateMode = "list"
						m.templateCursor = 0
						updated, _ = m.handleTemplateListKeys(makeKey("d"))
						m = updated.(Model)
						if m.err == nil {
							t.Fatalf("expected builtin delete error")
						}

						m.templateMode = "list"
						m.templateCursor = 1
						updated, _ = m.handleTemplateListKeys(makeKey("s"))
						m = updated.(Model)
						if m.templateMode != "save_select_source" {
							t.Fatalf("expected save source mode")
						}

						m.templateMode = "list"
						updated, _ = m.handleTemplateListKeys(makeKey("r"))
						m = updated.(Model)
						if m.message == "" {
							t.Fatalf("expected refresh message")
						}

						updated, _ = m.handleTemplateListKeys(makeKey("esc"))
						m = updated.(Model)
						if m.mode != "list" {
							t.Fatalf("expected exit to list")
						}

						m.mode = "template_manager"
						m.templateMode = "apply_select_target"
						updated, _ = m.handleTargetSelectKeys(makeKey("esc"))
						m = updated.(Model)
						if m.templateMode != "list" {
							t.Fatalf("expected cancel target select")
						}

						m.templateMode = "apply_select_target"
						m.selectedTemplate = nil
						updated, _ = m.handleTargetSelectKeys(makeKey("enter"))
						m = updated.(Model)
						if m.err == nil {
							t.Fatalf("expected missing template error")
						}

						m.selectedTemplate = &template.Template{ID: "bad", Name: "Bad", Category: "unknown"}
						updated, _ = m.handleTargetSelectKeys(makeKey("enter"))
						m = updated.(Model)
						if m.err == nil {
							t.Fatalf("expected unsupported category error")
						}

						m.selectedTemplate = tpl
						m.templateMode = "apply_preview_diff"
						m.diffContent = "line1\nline2\nline3\n"
						updated, _ = m.handleDiffPreviewKeys(makeKey("pgdown"))
						m = updated.(Model)
						updated, _ = m.handleDiffPreviewKeys(makeKey("n"))
						m = updated.(Model)
						if m.templateMode != "apply_select_target" {
							t.Fatalf("expected back to target select")
						}

						m.templateMode = "save_input_name"
						m.saveNameInput = textinput.New()
						m.saveNameInput.SetValue("X")
						view := m.viewSaveNameInput()
						if !strings.Contains(view, "模板名称") {
							t.Fatalf("expected save name view")
						}

						m.templateMode = "save_select_source"
						updated, _ = m.handleSourceSelectKeys(makeKey("esc"))
						m = updated.(Model)
						if m.templateMode != "list" {
							t.Fatalf("expected source select cancel")
						}

						m.templateMode = "save_select_source"
						updated, _ = m.handleSourceSelectKeys(makeKey("enter"))
						m = updated.(Model)
						if m.err == nil {
							t.Fatalf("expected save source error")
						}

						m.templateMode = "save_input_name"
						m.saveNameInput = textinput.New()
						updated, _ = m.handleSaveNameKeys(makeKey("esc"))
						m = updated.(Model)
						if m.templateMode != "save_select_source" {
							t.Fatalf("expected save name cancel")
						}

						m.selectedTemplate = &template.Template{ID: "p", Name: "Preview", Category: template.CategoryClaudeMd, Content: strings.Repeat("line\n", 5)}
						m.templateMode = "preview"
						updated, _ = m.handlePreviewKeys(makeKey("pgup"))
						m = updated.(Model)
						updated, _ = m.handlePreviewKeys(makeKey("esc"))
						m = updated.(Model)
						if m.templateMode != "list" {
							t.Fatalf("expected preview exit")
						}

						m.selectedTemplate = tpl
						m.targetSelectCursor = 0
						m.templateMode = "apply_select_target"
						if !strings.Contains(m.viewTargetSelect(), "选择目标路径") {
							t.Fatalf("expected target select view")
						}

						if !strings.Contains(m.viewTemplateList(), "用户模板") {
							t.Fatalf("expected template list view")
						}

						globalTarget := filepath.Join(home, ".claude", "CLAUDE.md")
						if err := os.MkdirAll(filepath.Dir(globalTarget), 0755); err != nil {
							t.Fatalf("MkdirAll() error = %v", err)
						}
						if err := os.WriteFile(globalTarget, []byte("content"), 0644); err != nil {
							t.Fatalf("WriteFile() error = %v", err)
						}
					})
				})
			},
		},
		{
			name: "mcp_apps_toggle",
			run: func(t *testing.T) {
				manager := newTestManager(t)
				server := addMcpServer(t, manager, "srv", config.McpApps{Claude: true})

				m := Model{manager: manager, mode: "mcp_manager", mcpMode: "apps_toggle", selectedMcp: &server}
				m.mcpAppsToggle = server.Apps
				updated, _ := m.handleMcpAppsToggleKeys(makeKey(" "))
				m = updated.(Model)
				updated, _ = m.handleMcpAppsToggleKeys(makeKey("enter"))
				m = updated.(Model)
				if m.message == "" {
					t.Fatalf("expected apps toggle message")
				}

				m.mcpMode = "apps_toggle"
				updated, _ = m.handleMcpAppsToggleKeys(makeKey("esc"))
				m = updated.(Model)
				if m.mcpMode != "list" {
					t.Fatalf("expected apps toggle cancel")
				}
			},
		},
		{
			name: "mcp_list_keys",
			run: func(t *testing.T) {
				manager := newTestManager(t)
				server := addMcpServer(t, manager, "srv", config.McpApps{Claude: true})
				m := Model{manager: manager, mode: "mcp_manager", mcpMode: "list"}
				m.refreshMcpServers()
				updated, _ := m.handleMcpListKeys(makeKey("e"))
				m = updated.(Model)
				if m.mcpMode != "edit" {
					t.Fatalf("expected edit mode")
				}
				m.mcpMode = "list"
				updated, _ = m.handleMcpListKeys(makeKey("d"))
				m = updated.(Model)
				if m.mcpMode != "delete" {
					t.Fatalf("expected delete mode")
				}

				m.mcpMode = "list"
				m.selectedMcp = &server
				updated, _ = m.handleMcpListKeys(makeKey("p"))
				m = updated.(Model)
				if m.mcpMode != "preset" {
					t.Fatalf("expected preset mode")
				}

				m.mcpMode = "list"
				updated, _ = m.handleMcpListKeys(makeKey("s"))
				m = updated.(Model)
				if m.message == "" {
					t.Fatalf("expected sync message")
				}

				updated, _ = m.handleMcpListKeys(makeKey("r"))
				m = updated.(Model)
				if !strings.Contains(m.message, "刷新") {
					t.Fatalf("expected refresh message")
				}

				updated, _ = m.handleMcpListKeys(makeKey("esc"))
				m = updated.(Model)
				if m.mode != "list" {
					t.Fatalf("expected exit mcp list")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.run(t)
		})
	}
}

func TestInitAndCmds(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "init_and_cmds",
			run: func(t *testing.T) {
				testutil.WithTempHome(t, func(_ string) {
					manager := newTestManager(t)
					m := New(manager)
					_ = m.Init()
					_ = tickCmd()
					_ = checkUpdateCmd()
					_ = downloadUpdateCmd(nil)
				})
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.run(t)
		})
	}
}
