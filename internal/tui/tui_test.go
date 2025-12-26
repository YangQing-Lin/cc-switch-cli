package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/YangQing-Lin/cc-switch-cli/internal/backup"
	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/template"
	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
	"github.com/YangQing-Lin/cc-switch-cli/internal/version"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func TestModelNewUpdateAndView(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "new_and_update_messages",
			run: func(t *testing.T) {
				testutil.WithTempHome(t, func(_ string) {
					manager := newTestManager(t)
					m := New(manager)
					if m.mode != "list" || m.currentApp != "claude" {
						t.Fatalf("unexpected defaults: mode=%s app=%s", m.mode, m.currentApp)
					}

					updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
					um := updated.(Model)
					if um.width != 100 || um.height != 40 {
						t.Fatalf("unexpected size: %dx%d", um.width, um.height)
					}

					updated, _ = um.Update(updateCheckMsg{hasUpdate: false})
					um = updated.(Model)
					if !strings.Contains(um.message, "最新") {
						t.Fatalf("expected latest message, got %s", um.message)
					}

					release := &version.ReleaseInfo{TagName: "v9.9.9"}
					updated, _ = um.Update(updateCheckMsg{hasUpdate: true, release: release})
					um = updated.(Model)
					if !strings.Contains(um.message, "v9.9.9") {
						t.Fatalf("expected update message, got %s", um.message)
					}

					updated, _ = um.Update(updateCheckMsg{err: fmt.Errorf("boom")})
					um = updated.(Model)
					if um.err == nil {
						t.Fatalf("expected error on update check")
					}

					updated, _ = um.Update(updateDownloadMsg{err: fmt.Errorf("download")})
					um = updated.(Model)
					if um.err == nil {
						t.Fatalf("expected error on update download")
					}

					updated, _ = um.Update(updateDownloadMsg{})
					um = updated.(Model)
					if !strings.Contains(um.message, "更新成功") {
						t.Fatalf("expected success message, got %s", um.message)
					}
				})
			},
		},
		{
			name: "view_dispatch",
			run: func(t *testing.T) {
				manager := newTestManager(t)
				addProvider(t, manager, "claude", "p1", "Claude One")
				m := Model{manager: manager, currentApp: "claude", mode: "list", configPath: manager.GetConfigPath()}
				m.refreshProviders()
				if !strings.Contains(m.View(), "配置管理") {
					t.Fatalf("expected list view")
				}

				m.viewMode = "multi"
				m.refreshAllColumns()
				if !strings.Contains(m.View(), "三列视图") {
					t.Fatalf("expected multi view")
				}

				m.viewMode = "single"
				m.mode = "add"
				m.initForm(nil)
				if !strings.Contains(m.View(), "添加新配置") {
					t.Fatalf("expected add view")
				}

				m.mode = "delete"
				m.deleteName = "Claude One"
				if !strings.Contains(m.View(), "确认删除") {
					t.Fatalf("expected delete view")
				}

				m.mode = "app_select"
				if !strings.Contains(m.View(), "选择应用") {
					t.Fatalf("expected app select view")
				}

				m.mode = "backup_list"
				if !strings.Contains(m.View(), "备份列表") {
					t.Fatalf("expected backup view")
				}

				m.templateManager = newTestTemplateManager(t)
				m.mode = "template_manager"
				m.templateMode = "list"
				m.templates = []template.Template{{ID: "t1", Name: "T1", Category: template.CategoryClaudeMd}}
				if !strings.Contains(m.View(), "模板管理") {
					t.Fatalf("expected template list view")
				}

				m.templateMode = "delete_confirm"
				m.selectedTemplate = &template.Template{ID: "t1", Name: "T1", Category: template.CategoryClaudeMd}
				if !strings.Contains(m.View(), "确认删除") {
					t.Fatalf("expected template delete view")
				}

				m.mode = "mcp_manager"
				m.mcpMode = "list"
				if !strings.Contains(m.View(), "MCP 服务器管理") {
					t.Fatalf("expected mcp list view")
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

func TestUpdateCommands_NoNetwork(t *testing.T) {
	t.Run("checkUpdateCmd", func(t *testing.T) {
		orig := checkForUpdateFunc
		checkForUpdateFunc = func() (*version.ReleaseInfo, bool, error) {
			return &version.ReleaseInfo{TagName: "v1.2.3"}, true, nil
		}
		t.Cleanup(func() { checkForUpdateFunc = orig })

		msg := checkUpdateCmd()()
		got, ok := msg.(updateCheckMsg)
		if !ok {
			t.Fatalf("expected updateCheckMsg, got %T", msg)
		}
		if got.err != nil || !got.hasUpdate || got.release == nil || got.release.TagName != "v1.2.3" {
			t.Fatalf("unexpected msg: %+v", got)
		}
	})

	t.Run("checkUpdateCmd error", func(t *testing.T) {
		orig := checkForUpdateFunc
		checkForUpdateFunc = func() (*version.ReleaseInfo, bool, error) {
			return nil, false, errors.New("boom")
		}
		t.Cleanup(func() { checkForUpdateFunc = orig })

		msg := checkUpdateCmd()()
		got := msg.(updateCheckMsg)
		if got.err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("downloadUpdateCmd passes release", func(t *testing.T) {
		var received *version.ReleaseInfo
		orig := downloadUpdateFunc
		downloadUpdateFunc = func(r *version.ReleaseInfo) error {
			received = r
			return errors.New("download boom")
		}
		t.Cleanup(func() { downloadUpdateFunc = orig })

		release := &version.ReleaseInfo{TagName: "v9.9.9"}
		msg := downloadUpdateCmd(release)()
		got := msg.(updateDownloadMsg)
		if received != release {
			t.Fatalf("expected release pointer passed through")
		}
		if got.err == nil || !strings.Contains(got.err.Error(), "download boom") {
			t.Fatalf("expected download error, got %v", got.err)
		}
	})
}

func TestViewDispatch_ExtraSubModes(t *testing.T) {
	testutil.WithTempHome(t, func(_ string) {
		manager := newTestManager(t)
		tm := newTestTemplateManager(t)

		m := Model{
			manager:         manager,
			templateManager: tm,
			mode:            "template_manager",
			currentApp:      "claude",
			selectedTemplate: &template.Template{
				ID:       "t1",
				Name:     "T1",
				Category: template.CategoryClaudeMd,
				Content:  "line1\nline2\n",
			},
		}

		m.templateMode = "apply_select_target"
		if v := m.View(); !strings.Contains(v, "选择目标路径") {
			t.Fatalf("expected target select view")
		}

		m.templateMode = "apply_preview_diff"
		m.selectedTargetPath = filepath.Join(t.TempDir(), "missing.md")
		if v := m.View(); !strings.Contains(v, "Diff 预览") || !strings.Contains(v, "将创建新文件") {
			t.Fatalf("expected diff preview view")
		}

		m.templateMode = "save_select_source"
		if v := m.View(); !strings.Contains(v, "保存为模板") {
			t.Fatalf("expected source select view")
		}

		m.templateMode = "save_input_name"
		m.selectedSourcePath = "CLAUDE.md"
		m.saveNameInput = textinput.New()
		if v := m.View(); !strings.Contains(v, "模板名称") {
			t.Fatalf("expected save name view")
		}

		m.templateMode = "preview"
		if v := m.View(); !strings.Contains(v, "预览:") {
			t.Fatalf("expected template preview view")
		}

		m.mode = "mcp_manager"
		m.selectedMcp = &config.McpServer{ID: "s1", Name: "Server One"}

		m.mcpMode = "add"
		m.initMcpForm(nil)
		if v := m.View(); !strings.Contains(v, "添加 MCP 服务器") {
			t.Fatalf("expected mcp form view")
		}

		m.mcpMode = "delete"
		if v := m.View(); !strings.Contains(v, "确认删除") {
			t.Fatalf("expected mcp delete view")
		}

		m.mcpMode = "apps_toggle"
		if v := m.View(); !strings.Contains(v, "选择应用") {
			t.Fatalf("expected mcp apps view")
		}

		m.mcpMode = "preset"
		m.mcpPresets = []config.McpServer{
			{ID: "p1", Name: "Preset One", Description: "desc", Server: map[string]any{"type": "http", "url": "http://localhost"}},
		}
		if v := m.View(); !strings.Contains(v, "预设服务器") {
			t.Fatalf("expected mcp preset view")
		}
	})
}

func TestListModeKeySequences(t *testing.T) {
	cases := []struct {
		name   string
		keys   []string
		setup  func(m *Model)
		verify func(t *testing.T, m Model)
	}{
		{
			name: "enter_add_mode",
			keys: []string{"a"},
			setup: func(m *Model) {
				m.currentApp = "claude"
			},
			verify: func(t *testing.T, m Model) {
				if m.mode != "add" || len(m.inputs) == 0 {
					t.Fatalf("expected add mode")
				}
			},
		},
		{
			name: "toggle_multi_view",
			keys: []string{"v"},
			verify: func(t *testing.T, m Model) {
				if m.viewMode != "multi" {
					t.Fatalf("expected multi view")
				}
				if !strings.Contains(m.message, "三列") {
					t.Fatalf("expected message for multi view")
				}
			},
		},
		{
			name: "toggle_app",
			keys: []string{"t"},
			verify: func(t *testing.T, m Model) {
				if m.currentApp != "codex" {
					t.Fatalf("expected app codex, got %s", m.currentApp)
				}
			},
		},
		{
			name: "enter_backup_list",
			keys: []string{"l"},
			setup: func(m *Model) {
				if _, err := backup.CreateBackup(m.configPath); err != nil {
					t.Fatalf("CreateBackup() error = %v", err)
				}
			},
			verify: func(t *testing.T, m Model) {
				if m.mode != "backup_list" {
					t.Fatalf("expected backup_list mode")
				}
				if len(m.backupList) == 0 {
					t.Fatalf("expected backups")
				}
			},
		},
		{
			name: "enter_template_manager",
			keys: []string{"m"},
			setup: func(m *Model) {
				m.templateManager = newTestTemplateManager(t)
			},
			verify: func(t *testing.T, m Model) {
				if m.mode != "template_manager" || m.templateMode != "list" {
					t.Fatalf("expected template manager list")
				}
			},
		},
		{
			name: "enter_mcp_manager",
			keys: []string{"M"},
			verify: func(t *testing.T, m Model) {
				if m.mode != "mcp_manager" || m.mcpMode != "list" {
					t.Fatalf("expected mcp manager list")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manager := newTestManager(t)
			addProvider(t, manager, "claude", "p1", "Claude One")
			m := Model{
				manager:    manager,
				mode:       "list",
				currentApp: "claude",
				configPath: manager.GetConfigPath(),
			}
			m.refreshProviders()
			if tc.setup != nil {
				tc.setup(&m)
			}
			updated := testutil.BubbleTeaTestHelper(t, m, tc.keys)
			final := updated.(Model)
			tc.verify(t, final)
		})
	}
}

func TestMultiColumnKeys(t *testing.T) {
	cases := []struct {
		name   string
		keys   []string
		verify func(t *testing.T, m Model)
	}{
		{
			name: "switch_columns_and_enter",
			keys: []string{"right", "down", "enter", "v"},
			verify: func(t *testing.T, m Model) {
				if m.viewMode != "single" {
					t.Fatalf("expected single view after v")
				}
				if m.message == "" {
					t.Fatalf("expected switch message")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manager := newTestManager(t)
			addProvider(t, manager, "claude", "c1", "Claude One")
			addProvider(t, manager, "codex", "x1", "Codex One")
			addProvider(t, manager, "gemini", "g1", "Gemini One")

			m := Model{manager: manager, mode: "list", viewMode: "multi", currentApp: "claude", configPath: manager.GetConfigPath()}
			m.refreshAllColumns()
			updated := testutil.BubbleTeaTestHelper(t, m, tc.keys)
			final := updated.(Model)
			tc.verify(t, final)
		})
	}
}

func TestFormKeyHandlingAndSubmit(t *testing.T) {
	cases := []struct {
		name   string
		setup  func(m *Model)
		keys   []string
		verify func(t *testing.T, m Model)
	}{
		{
			name: "selector_activation",
			setup: func(m *Model) {
				m.currentApp = "codex"
				m.mode = "add"
				m.initForm(nil)
				m.focusIndex = 4
				m.width = 80
				m.height = 24
			},
			keys: []string{" ", "down", "enter"},
			verify: func(t *testing.T, m Model) {
				if m.modelSelectorActive {
					t.Fatalf("expected selector to close")
				}
				if m.inputs[4].Value() == "" {
					t.Fatalf("expected selector to set value")
				}
			},
		},
		{
			name: "clear_and_undo",
			setup: func(m *Model) {
				m.currentApp = "claude"
				m.mode = "add"
				m.initForm(nil)
				m.inputs[0].SetValue("Name")
				m.inputs[1].SetValue("token")
			},
			keys: []string{"ctrl+d", "ctrl+z"},
			verify: func(t *testing.T, m Model) {
				if m.inputs[0].Value() != "Name" || m.inputs[1].Value() != "token" {
					t.Fatalf("expected undo to restore values")
				}
			},
		},
		{
			name: "submit_missing_name",
			setup: func(m *Model) {
				m.currentApp = "claude"
				m.mode = "add"
				m.initForm(nil)
				m.inputs[1].SetValue("token")
				m.inputs[2].SetValue("https://api.anthropic.com")
			},
			keys: []string{"enter"},
			verify: func(t *testing.T, m Model) {
				if m.err == nil {
					t.Fatalf("expected error for missing name")
				}
			},
		},
		{
			name: "submit_success_claude",
			setup: func(m *Model) {
				m.currentApp = "claude"
				m.mode = "add"
				m.initForm(nil)
				m.inputs[0].SetValue("Claude One")
				m.inputs[1].SetValue("token")
				m.inputs[2].SetValue("https://api.anthropic.com")
			},
			keys: []string{"enter"},
			verify: func(t *testing.T, m Model) {
				if m.mode != "list" || m.err != nil {
					t.Fatalf("expected successful submit")
				}
				if !strings.Contains(m.message, "配置添加成功") {
					t.Fatalf("unexpected message: %s", m.message)
				}
			},
		},
		{
			name: "submit_gemini_oauth",
			setup: func(m *Model) {
				m.currentApp = "gemini"
				m.mode = "add"
				m.initForm(nil)
				m.inputs[0].SetValue("google oauth")
			},
			keys: []string{"enter"},
			verify: func(t *testing.T, m Model) {
				if m.err != nil {
					t.Fatalf("expected OAuth submit success: %v", m.err)
				}
				if m.mode != "list" {
					t.Fatalf("expected to return to list")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manager := newTestManager(t)
			m := Model{manager: manager}
			if tc.setup != nil {
				tc.setup(&m)
			}
			updated := testutil.BubbleTeaTestHelper(t, m, tc.keys)
			final := updated.(Model)
			tc.verify(t, final)
		})
	}
}

func TestFormLoadConfigAndReadonly(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "load_claude_and_codex",
			run: func(t *testing.T) {
				testutil.WithTempHome(t, func(home string) {
					claudePath := filepath.Join(home, ".claude", "settings.json")
					if err := os.MkdirAll(filepath.Dir(claudePath), 0755); err != nil {
						t.Fatalf("MkdirAll() error = %v", err)
					}
					claudeData := []byte(`{"env":{"ANTHROPIC_AUTH_TOKEN":"t","ANTHROPIC_BASE_URL":"u","ANTHROPIC_MODEL":"m"}}`)
					if err := os.WriteFile(claudePath, claudeData, 0644); err != nil {
						t.Fatalf("WriteFile() error = %v", err)
					}

					codexAuth := filepath.Join(home, ".codex", "auth.json")
					if err := os.MkdirAll(filepath.Dir(codexAuth), 0755); err != nil {
						t.Fatalf("MkdirAll() error = %v", err)
					}
					if err := os.WriteFile(codexAuth, []byte(`{"OPENAI_API_KEY":"sk"}`), 0644); err != nil {
						t.Fatalf("WriteFile() error = %v", err)
					}

					codexConfig := filepath.Join(home, ".codex", "config.toml")
					codexData := []byte("base_url = \"https://example.com\"\nmodel = \"gpt-5\"\nmodel_reasoning_effort = \"high\"\n")
					if err := os.WriteFile(codexConfig, codexData, 0644); err != nil {
						t.Fatalf("WriteFile() error = %v", err)
					}

					m := Model{currentApp: "claude"}
					token, baseURL, model, _, _, _, loaded := m.loadLiveConfigForForm()
					if !loaded || token != "t" || baseURL != "u" || model != "m" {
						t.Fatalf("unexpected claude values")
					}

					m.currentApp = "codex"
					token, baseURL, model, reasoning, loaded := m.loadCodexConfigForForm()
					if !loaded || token != "sk" || baseURL != "https://example.com" || model != "gpt-5" || reasoning != "high" {
						t.Fatalf("unexpected codex values")
					}
				})
			},
		},
		{
			name: "readonly_field_update",
			run: func(t *testing.T) {
				m := Model{}
				if m.isReadOnlyField(0) {
					t.Fatalf("expected no readonly fields")
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

func TestDeleteAndAppSelect(t *testing.T) {
	cases := []struct {
		name   string
		setup  func(m *Model)
		keys   []string
		verify func(t *testing.T, m Model)
	}{
		{
			name: "delete_provider_success",
			setup: func(m *Model) {
				addProvider(t, m.manager, "claude", "p1", "Claude One")
				addProvider(t, m.manager, "claude", "p2", "Claude Two")
				m.refreshProviders()
				m.mode = "delete"
				m.deleteName = "Claude Two"
			},
			keys: []string{"y"},
			verify: func(t *testing.T, m Model) {
				if m.mode != "list" || m.err != nil {
					t.Fatalf("expected delete success")
				}
			},
		},
		{
			name: "app_select",
			setup: func(m *Model) {
				m.mode = "app_select"
			},
			keys: []string{"down", "enter"},
			verify: func(t *testing.T, m Model) {
				if m.currentApp != "codex" || m.mode != "list" {
					t.Fatalf("expected app switch to codex")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := Model{manager: newTestManager(t), currentApp: "claude"}
			if tc.setup != nil {
				tc.setup(&m)
			}
			updated := testutil.BubbleTeaTestHelper(t, m, tc.keys)
			final := updated.(Model)
			tc.verify(t, final)
		})
	}
}

func TestBackupListFlow(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "restore_backup",
			run: func(t *testing.T) {
				manager := newTestManager(t)
				addProvider(t, manager, "claude", "p1", "Claude One")
				configPath := manager.GetConfigPath()

				id, err := backup.CreateBackup(configPath)
				if err != nil {
					t.Fatalf("CreateBackup() error = %v", err)
				}
				if id == "" {
					t.Fatalf("expected backup id")
				}

				backups, err := backup.ListBackups(filepath.Dir(configPath))
				if err != nil || len(backups) == 0 {
					t.Fatalf("expected backups list")
				}

				m := Model{manager: manager, mode: "backup_list", configPath: configPath, backupList: backups}
				updated := testutil.BubbleTeaTestHelper(t, m, []string{"enter"})
				final := updated.(Model)
				if final.mode != "list" {
					t.Fatalf("expected to return to list")
				}
				if final.err != nil {
					t.Fatalf("unexpected restore error: %v", final.err)
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

func TestTemplateFlows(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "apply_and_save_and_delete",
			run: func(t *testing.T) {
				testutil.WithTempHome(t, func(home string) {
					testutil.WithTempCWD(t, func(cwd string) {
						tm := newTestTemplateManager(t)
						id, err := tm.AddTemplate("User", template.CategoryClaudeMd, "content")
						if err != nil {
							t.Fatalf("AddTemplate() error = %v", err)
						}
						tpl, _ := tm.GetTemplate(id)

						m := Model{manager: newTestManager(t), mode: "template_manager", templateMode: "list", templateManager: tm}
						m.templates = []template.Template{*tpl}
						m.templateCursor = 0

						updated, _ := m.handleTemplateListKeys(tea.KeyMsg{Type: tea.KeyEnter})
						m = updated.(Model)
						if m.templateMode != "apply_select_target" {
							t.Fatalf("expected apply_select_target")
						}

						globalTarget := filepath.Join(home, ".claude", "CLAUDE.md")
						if err := os.MkdirAll(filepath.Dir(globalTarget), 0755); err != nil {
							t.Fatalf("MkdirAll() error = %v", err)
						}
						if err := os.WriteFile(globalTarget, []byte("old"), 0644); err != nil {
							t.Fatalf("WriteFile() error = %v", err)
						}

						updated, _ = m.handleTargetSelectKeys(tea.KeyMsg{Type: tea.KeyEnter})
						m = updated.(Model)
						if m.templateMode != "apply_preview_diff" || m.diffContent == "" {
							t.Fatalf("expected diff preview")
						}

						updated, _ = m.handleDiffPreviewKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
						m = updated.(Model)
						if m.templateMode != "list" {
							t.Fatalf("expected to return to list")
						}
						data, err := os.ReadFile(globalTarget)
						if err != nil {
							t.Fatalf("ReadFile() error = %v", err)
						}
						if string(data) != "content" {
							t.Fatalf("expected applied content")
						}

						m.templateMode = "save_select_source"
						updated, _ = m.handleSourceSelectKeys(tea.KeyMsg{Type: tea.KeyEnter})
						m = updated.(Model)
						if m.templateMode != "save_input_name" {
							t.Fatalf("expected save_input_name")
						}

						m.saveNameInput.SetValue("Saved")
						updated, _ = m.handleSaveNameKeys(tea.KeyMsg{Type: tea.KeyEnter})
						m = updated.(Model)
						if m.templateMode != "list" {
							t.Fatalf("expected save to return list")
						}

						m.selectedTemplate = tpl
						m.templateMode = "delete_confirm"
						updated, _ = m.handleDeleteConfirmKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
						m = updated.(Model)
						if m.templateMode != "list" {
							t.Fatalf("expected delete return list")
						}

						m.selectedTemplate = &template.Template{ID: "p", Name: "Preview", Category: template.CategoryClaudeMd, Content: strings.Repeat("line\n", 30)}
						m.templateMode = "preview"
						updated, _ = m.handlePreviewKeys(tea.KeyMsg{Type: tea.KeyDown})
						m = updated.(Model)
						if m.previewScrollOffset == 0 {
							t.Fatalf("expected preview scroll")
						}

						view := m.viewTemplatePreview()
						if !strings.Contains(view, "预览") || !strings.Contains(view, "Preview") {
							t.Fatalf("unexpected preview view")
						}

						if !strings.Contains(m.viewSourceSelect(), "保存为模板") {
							t.Fatalf("expected source select view")
						}

						m.selectedTemplate = tpl
						m.selectedTargetPath = globalTarget
						m.diffContent = "--- a\n+++ b\n@@\n-old\n+new\n"
						if !strings.Contains(m.viewDiffPreview(), "Diff 预览") {
							t.Fatalf("expected diff preview view")
						}
					})
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

func TestMcpFlows(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "list_form_delete_preset",
			run: func(t *testing.T) {
				manager := newTestManager(t)
				server := addMcpServer(t, manager, "srv", config.McpApps{Claude: true})

				m := Model{manager: manager, mode: "mcp_manager", mcpMode: "list"}
				m.refreshMcpServers()
				if !strings.Contains(m.viewMcpList(), "MCP 服务器管理") {
					t.Fatalf("expected mcp list view")
				}

				updated, _ := m.handleMcpListKeys(tea.KeyMsg{Type: tea.KeyEnter})
				m = updated.(Model)
				if m.mcpMode != "apps_toggle" {
					t.Fatalf("expected apps toggle mode")
				}

				m.mcpMode = "list"
				updated, _ = m.handleMcpListKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
				m = updated.(Model)
				if m.mcpMode != "add" {
					t.Fatalf("expected add mode")
				}
				m.mcpInputs[0].SetValue("id")
				m.mcpInputs[1].SetValue("Name")
				m.mcpInputs[2].SetValue("npx")
				m.mcpInputs[3].SetValue("mcp-server-fetch")
				_, updatedModel, _ := m.handleMcpFormKeys(tea.KeyMsg{Type: tea.KeyEnter})
				m = updatedModel.(Model)
				if m.mcpMode != "list" || m.err != nil {
					t.Fatalf("expected save success")
				}

				m.selectedMcp = &server
				m.mcpMode = "delete"
				updated, _ = m.handleMcpDeleteKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
				m = updated.(Model)
				if m.mcpMode != "list" {
					t.Fatalf("expected delete return list")
				}

				m.mcpPresets = config.GetMcpPresets()
				m.mcpMode = "preset"
				updated, _ = m.handleMcpPresetKeys(tea.KeyMsg{Type: tea.KeyEnter})
				m = updated.(Model)
				if m.mcpMode != "apps_toggle" {
					t.Fatalf("expected apps toggle from preset")
				}

				m.mcpMode = "add"
				m.initMcpForm(nil)
				m.mcpConnType = "stdio"
				m.mcpInputs[0].SetValue("bad")
				m.mcpInputs[1].SetValue("bad")
				m.mcpInputs[2].SetValue("cmd")
				m.mcpInputs[3].SetValue("\"")
				if err := m.saveMcpForm(); err == nil {
					t.Fatalf("expected parse error")
				}

				m.mcpMode = "edit"
				m.initMcpForm(&server)
				m.mcpFocusIndex = 1
				updated, _ = m.updateMcpInputs(tea.KeyMsg{Type: tea.KeyTab})
				m = updated.(Model)
				if m.mcpFocusIndex == 0 {
					t.Fatalf("expected focus skip ID in edit mode")
				}

				if !strings.Contains(m.viewMcpForm(), "MCP 服务器") {
					t.Fatalf("expected mcp form view")
				}
				if !strings.Contains(m.viewMcpDelete(), "确认删除") {
					t.Fatalf("expected mcp delete view")
				}
				if !strings.Contains(m.viewMcpAppsToggle(), "选择应用") {
					t.Fatalf("expected mcp apps view")
				}
				if !strings.Contains(m.viewMcpPreset(), "预设服务器") {
					t.Fatalf("expected mcp preset view")
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

func TestPortableToggle(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "enable_disable_portable",
			run: func(t *testing.T) {
				manager := newTestManager(t)
				m := Model{manager: manager, mode: "list"}
				updated, _ := m.handleListKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
				m = updated.(Model)
				if !m.isPortableMode && m.err != nil {
					t.Fatalf("expected toggle success or no error")
				}
				updated, _ = m.handleListKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
				m = updated.(Model)
				if m.isPortableMode {
					t.Fatalf("expected portable mode disabled")
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

func TestPortableModeErrors(t *testing.T) {
	t.Run("executable error", func(t *testing.T) {
		orig := tuiExecutableFunc
		tuiExecutableFunc = func() (string, error) { return "", errors.New("no exe") }
		t.Cleanup(func() { tuiExecutableFunc = orig })

		m := Model{}
		if err := m.enablePortableMode(); err == nil {
			t.Fatalf("expected enablePortableMode error")
		}
		if err := m.disablePortableMode(); err == nil {
			t.Fatalf("expected disablePortableMode error")
		}
	})

	t.Run("write file error", func(t *testing.T) {
		origExec := tuiExecutableFunc
		origWrite := tuiWriteFileFunc
		tuiExecutableFunc = func() (string, error) { return filepath.Join(t.TempDir(), "ccs"), nil }
		tuiWriteFileFunc = func(string, []byte, os.FileMode) error { return errors.New("write boom") }
		t.Cleanup(func() {
			tuiExecutableFunc = origExec
			tuiWriteFileFunc = origWrite
		})

		m := Model{}
		if err := m.enablePortableMode(); err == nil || !strings.Contains(err.Error(), "创建 portable.ini 失败") {
			t.Fatalf("expected write error, got %v", err)
		}
	})

	t.Run("remove not exist ignored", func(t *testing.T) {
		origExec := tuiExecutableFunc
		origRemove := tuiRemoveFileFunc
		tuiExecutableFunc = func() (string, error) { return filepath.Join(t.TempDir(), "ccs"), nil }
		tuiRemoveFileFunc = func(string) error { return os.ErrNotExist }
		t.Cleanup(func() {
			tuiExecutableFunc = origExec
			tuiRemoveFileFunc = origRemove
		})

		m := Model{}
		if err := m.disablePortableMode(); err != nil {
			t.Fatalf("expected ignore not-exist, got %v", err)
		}
	})

	t.Run("remove error", func(t *testing.T) {
		origExec := tuiExecutableFunc
		origRemove := tuiRemoveFileFunc
		tuiExecutableFunc = func() (string, error) { return filepath.Join(t.TempDir(), "ccs"), nil }
		tuiRemoveFileFunc = func(string) error { return errors.New("rm boom") }
		t.Cleanup(func() {
			tuiExecutableFunc = origExec
			tuiRemoveFileFunc = origRemove
		})

		m := Model{}
		if err := m.disablePortableMode(); err == nil || !strings.Contains(err.Error(), "删除 portable.ini 失败") {
			t.Fatalf("expected remove error, got %v", err)
		}
	})
}

func TestViewSnippets(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "list_and_multi_views",
			run: func(t *testing.T) {
				manager := newTestManager(t)
				addProvider(t, manager, "claude", "p1", strings.Repeat("长名字", 10))
				m := Model{manager: manager, currentApp: "claude", configPath: manager.GetConfigPath()}
				m.refreshProviders()
				view := m.viewList()
				if !strings.Contains(view, "配置管理") {
					t.Fatalf("expected list view")
				}

				m.viewMode = "multi"
				m.refreshAllColumns()
				view = m.viewListMulti()
				if !strings.Contains(view, "三列视图") || !strings.Contains(view, "┌") {
					t.Fatalf("expected multi view table")
				}
			},
		},
		{
			name: "backup_and_delete_views",
			run: func(t *testing.T) {
				m := Model{backupList: []backup.BackupInfo{{Path: "a", Timestamp: time.Now(), Size: 1024}}}
				if !strings.Contains(m.viewBackupList(), "备份列表") {
					t.Fatalf("expected backup view")
				}

				m.deleteName = "Claude One"
				if !strings.Contains(m.viewDelete(), "删除") {
					t.Fatalf("expected delete view")
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
