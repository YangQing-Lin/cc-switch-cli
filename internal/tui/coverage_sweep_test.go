package tui

import (
	"encoding/json"
	"errors"
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

func TestHandleListKeysSweep(t *testing.T) {
	manager := newTestManager(t)
	addProvider(t, manager, "claude", "p1", "Claude One")
	addProvider(t, manager, "claude", "p2", "Claude Two")
	addProvider(t, manager, "codex", "x1", "Codex One")
	if err := manager.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	m := Model{manager: manager, mode: "list", currentApp: "claude", configPath: manager.GetConfigPath(), templateManager: newTestTemplateManager(t)}
	m.refreshProviders()

	cases := []struct {
		name   string
		key    string
		setup  func(t *testing.T, m *Model)
		verify func(t *testing.T, before Model, after Model, cmd tea.Cmd)
	}{
		{
			name:  "up",
			key:   "up",
			setup: func(_ *testing.T, m *Model) { m.cursor = 1 },
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.mode != "list" || after.cursor != 0 {
					t.Fatalf("expected list mode and cursor=0, got mode=%q cursor=%d", after.mode, after.cursor)
				}
			},
		},
		{
			name:  "down",
			key:   "down",
			setup: func(_ *testing.T, m *Model) { m.cursor = 0 },
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.mode != "list" || after.cursor != 1 {
					t.Fatalf("expected list mode and cursor=1, got mode=%q cursor=%d", after.mode, after.cursor)
				}
			},
		},
		{
			name:  "enter",
			key:   "enter",
			setup: func(_ *testing.T, m *Model) { m.cursor = 1 },
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.mode != "list" || after.err != nil || after.message == "" {
					t.Fatalf("expected list mode, message set, err nil; got mode=%q msg=%q err=%v", after.mode, after.message, after.err)
				}
			},
		},
		{
			name: "add",
			key:  "a",
			verify: func(t *testing.T, _ Model, after Model, cmd tea.Cmd) {
				if after.mode != "add" || len(after.inputs) == 0 || cmd == nil {
					t.Fatalf("expected add mode with inputs and cmd, got mode=%q inputs=%d cmd=%v", after.mode, len(after.inputs), cmd)
				}
			},
		},
		{
			name: "copy",
			key:  "C",
			verify: func(t *testing.T, _ Model, after Model, cmd tea.Cmd) {
				if after.mode != "add" || len(after.inputs) < 3 || cmd == nil {
					t.Fatalf("expected add mode with inputs and cmd, got mode=%q inputs=%d cmd=%v", after.mode, len(after.inputs), cmd)
				}
				// initForm 会消费 copyFromProvider 并清空它
				if after.copyFromProvider != nil {
					t.Fatalf("expected copyFromProvider cleared, got %v", after.copyFromProvider)
				}
				if after.inputs[1].Value() != "token" {
					t.Fatalf("expected copied token to prefill, got %q", after.inputs[1].Value())
				}
			},
		},
		{
			name: "edit",
			key:  "e",
			verify: func(t *testing.T, _ Model, after Model, cmd tea.Cmd) {
				if after.mode != "edit" || after.editName == "" || len(after.inputs) == 0 || cmd == nil {
					t.Fatalf("expected edit mode with editName/inputs and cmd, got mode=%q editName=%q inputs=%d cmd=%v", after.mode, after.editName, len(after.inputs), cmd)
				}
			},
		},
		{
			name: "delete",
			key:  "d",
			setup: func(t *testing.T, m *Model) {
				m.refreshProviders()
				current := m.manager.GetCurrentProviderForApp(m.currentApp)
				if current != nil {
					for i, p := range m.providers {
						if p.ID != current.ID {
							m.cursor = i
							return
						}
					}
				}
				if len(m.providers) > 0 {
					m.cursor = 0
				} else {
					t.Fatalf("expected providers for delete test")
				}
			},
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.mode == "delete" {
					if after.deleteName == "" {
						t.Fatalf("expected deleteName to be set")
					}
					return
				}
				// 如果命中了“不能删除当前配置”的分支，则应返回 list 且 err 非空
				if after.mode != "list" || after.err == nil {
					t.Fatalf("expected delete mode or list mode with err set, got mode=%q err=%v", after.mode, after.err)
				}
			},
		},
		{
			name: "refresh",
			key:  "r",
			verify: func(t *testing.T, _ Model, after Model, cmd tea.Cmd) {
				if after.mode != "list" || after.err != nil || after.message == "" || cmd == nil {
					t.Fatalf("expected list mode, message set, err nil and cmd, got mode=%q msg=%q err=%v cmd=%v", after.mode, after.message, after.err, cmd)
				}
			},
		},
		{
			name: "toggle_app",
			key:  "t",
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.mode != "list" || after.currentApp != "codex" || after.cursor != 0 {
					t.Fatalf("expected currentApp=codex and cursor=0, got app=%q cursor=%d mode=%q", after.currentApp, after.cursor, after.mode)
				}
			},
		},
		{
			name:  "switch_claude",
			key:   "c",
			setup: func(_ *testing.T, m *Model) { m.currentApp = "codex" },
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.currentApp != "claude" || after.mode != "list" {
					t.Fatalf("expected currentApp=claude in list mode, got app=%q mode=%q", after.currentApp, after.mode)
				}
			},
		},
		{
			name: "switch_codex",
			key:  "x",
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.currentApp != "codex" || after.mode != "list" {
					t.Fatalf("expected currentApp=codex in list mode, got app=%q mode=%q", after.currentApp, after.mode)
				}
			},
		},
		{
			name: "switch_gemini",
			key:  "g",
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.currentApp != "gemini" || after.mode != "list" {
					t.Fatalf("expected currentApp=gemini in list mode, got app=%q mode=%q", after.currentApp, after.mode)
				}
			},
		},
		{
			name: "backup",
			key:  "b",
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.mode != "list" || (after.message == "" && after.err == nil) {
					t.Fatalf("expected backup result (message or err) in list mode, got mode=%q msg=%q err=%v", after.mode, after.message, after.err)
				}
			},
		},
		{
			name: "backup_list",
			key:  "l",
			setup: func(t *testing.T, m *Model) {
				backupID, err := backup.CreateBackup(m.configPath)
				if err != nil {
					t.Fatalf("CreateBackup() error = %v", err)
				}
				if backupID == "" {
					t.Fatalf("CreateBackup() returned empty backupID")
				}
			},
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.mode != "backup_list" || after.err != nil {
					t.Fatalf("expected backup_list mode with err nil, got mode=%q err=%v", after.mode, after.err)
				}
			},
		},
		{
			name: "template_manager",
			key:  "m",
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.mode != "template_manager" || after.templateMode != "list" || after.err != nil {
					t.Fatalf("expected template_manager(list) with err nil, got mode=%q templateMode=%q err=%v", after.mode, after.templateMode, after.err)
				}
			},
		},
		{
			name:  "template_manager_error",
			key:   "m",
			setup: func(_ *testing.T, m *Model) { m.templateManager = nil },
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.mode != "list" || after.err == nil {
					t.Fatalf("expected list mode with err set, got mode=%q err=%v", after.mode, after.err)
				}
			},
		},
		{
			name: "mcp_manager",
			key:  "M",
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.mode != "mcp_manager" || after.mcpMode != "list" || after.err != nil {
					t.Fatalf("expected mcp_manager(list) with err nil, got mode=%q mcpMode=%q err=%v", after.mode, after.mcpMode, after.err)
				}
			},
		},
		{
			name: "check_update",
			key:  "u",
			verify: func(t *testing.T, _ Model, after Model, cmd tea.Cmd) {
				if after.mode != "list" || after.err != nil || after.message == "" || cmd == nil {
					t.Fatalf("expected list mode, message set, err nil and cmd, got mode=%q msg=%q err=%v cmd=%v", after.mode, after.message, after.err, cmd)
				}
			},
		},
		{
			name: "download_no_release",
			key:  "U",
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.mode != "list" || after.err == nil {
					t.Fatalf("expected list mode with err set, got mode=%q err=%v", after.mode, after.err)
				}
			},
		},
		{
			name:  "download_with_release",
			key:   "U",
			setup: func(_ *testing.T, m *Model) { m.latestRelease = &version.ReleaseInfo{TagName: "v0"} },
			verify: func(t *testing.T, _ Model, after Model, cmd tea.Cmd) {
				if after.mode != "list" || after.err != nil || after.message == "" || cmd == nil {
					t.Fatalf("expected list mode, message set, err nil and cmd, got mode=%q msg=%q err=%v cmd=%v", after.mode, after.message, after.err, cmd)
				}
			},
		},
		{
			name: "toggle_portable",
			key:  "p",
			verify: func(t *testing.T, before Model, after Model, _ tea.Cmd) {
				if after.mode != "list" || (after.isPortableMode == before.isPortableMode && after.err == nil) {
					t.Fatalf("expected portable mode to toggle or err set, got before=%v after=%v err=%v", before.isPortableMode, after.isPortableMode, after.err)
				}
			},
		},
		{
			name: "toggle_multi",
			key:  "v",
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.mode != "list" || after.viewMode != "multi" || after.err != nil {
					t.Fatalf("expected list mode with viewMode=multi and err nil, got mode=%q viewMode=%q err=%v", after.mode, after.viewMode, after.err)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mm := m
			mm.mode = "list"
			mm.viewMode = "single"
			mm.refreshProviders()
			if tc.setup != nil {
				tc.setup(t, &mm)
			}
			updatedModel, cmd := mm.handleListKeys(makeKey(tc.key))
			updated, ok := updatedModel.(Model)
			if !ok {
				t.Fatalf("expected Model, got %T", updatedModel)
			}
			tc.verify(t, mm, updated, cmd)
		})
	}
}

func TestHandleMultiColumnKeysSweep(t *testing.T) {
	manager := newTestManager(t)
	addProvider(t, manager, "claude", "c1", "Claude One")
	addProvider(t, manager, "claude", "c2", "Claude Two")
	addProvider(t, manager, "codex", "x1", "Codex One")
	addProvider(t, manager, "gemini", "g1", "Gemini One")
	if err := manager.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	m := Model{manager: manager, mode: "list", viewMode: "multi", currentApp: "claude", configPath: manager.GetConfigPath(), templateManager: newTestTemplateManager(t)}
	m.refreshAllColumns()

	cases := []struct {
		name   string
		key    string
		setup  func(t *testing.T, m *Model)
		verify func(t *testing.T, before Model, after Model, cmd tea.Cmd)
	}{
		{
			name: "tab",
			key:  "tab",
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.viewMode != "multi" || after.mode != "list" || after.columnCursor != 1 {
					t.Fatalf("expected multi list mode and columnCursor=1, got mode=%q viewMode=%q columnCursor=%d", after.mode, after.viewMode, after.columnCursor)
				}
			},
		},
		{
			name: "left",
			key:  "left",
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.viewMode != "multi" || after.mode != "list" || after.columnCursor != 2 {
					t.Fatalf("expected multi list mode and columnCursor=2, got mode=%q viewMode=%q columnCursor=%d", after.mode, after.viewMode, after.columnCursor)
				}
			},
		},
		{
			name: "right",
			key:  "right",
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.viewMode != "multi" || after.mode != "list" || after.columnCursor != 1 {
					t.Fatalf("expected multi list mode and columnCursor=1, got mode=%q viewMode=%q columnCursor=%d", after.mode, after.viewMode, after.columnCursor)
				}
			},
		},
		{
			name: "up",
			key:  "up",
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.viewMode != "multi" || after.mode != "list" || after.columnCursors[after.columnCursor] == 0 {
					t.Fatalf("expected cursor to move within current column, got col=%d row=%d", after.columnCursor, after.columnCursors[after.columnCursor])
				}
			},
		},
		{
			name: "down",
			key:  "down",
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.viewMode != "multi" || after.mode != "list" || after.columnCursors[after.columnCursor] == 0 {
					t.Fatalf("expected cursor to move within current column, got col=%d row=%d", after.columnCursor, after.columnCursors[after.columnCursor])
				}
			},
		},
		{
			name: "enter",
			key:  "enter",
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.mode != "list" || after.viewMode != "multi" || (after.message == "" && after.err == nil) {
					t.Fatalf("expected switch result (message or err) in multi list mode, got mode=%q viewMode=%q msg=%q err=%v", after.mode, after.viewMode, after.message, after.err)
				}
			},
		},
		{
			name: "add",
			key:  "a",
			verify: func(t *testing.T, _ Model, after Model, cmd tea.Cmd) {
				if after.mode != "add" || after.currentApp != "claude" || len(after.inputs) == 0 || cmd == nil {
					t.Fatalf("expected add mode for claude with inputs and cmd, got mode=%q app=%q inputs=%d cmd=%v", after.mode, after.currentApp, len(after.inputs), cmd)
				}
			},
		},
		{
			name: "edit",
			key:  "e",
			verify: func(t *testing.T, _ Model, after Model, cmd tea.Cmd) {
				if after.mode != "edit" || after.editName == "" || len(after.inputs) == 0 || cmd == nil {
					t.Fatalf("expected edit mode with editName/inputs and cmd, got mode=%q editName=%q inputs=%d cmd=%v", after.mode, after.editName, len(after.inputs), cmd)
				}
			},
		},
		{
			name: "delete",
			key:  "d",
			setup: func(t *testing.T, m *Model) {
				col := m.columnCursor
				appName := m.columnToAppName(col)
				current := m.manager.GetCurrentProviderForApp(appName)
				if current != nil {
					for i, p := range m.columnProviders[col] {
						if p.ID != current.ID {
							m.columnCursors[col] = i
							return
						}
					}
				}
				if len(m.columnProviders[col]) > 0 {
					m.columnCursors[col] = 0
				} else {
					t.Fatalf("expected providers for delete test")
				}
			},
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.mode == "delete" {
					if after.deleteName == "" {
						t.Fatalf("expected deleteName to be set")
					}
					return
				}
				// 如果命中了“不能删除当前配置”的分支，则应返回 list 且 err 非空
				if after.mode != "list" || after.err == nil {
					t.Fatalf("expected delete mode or list mode with err set, got mode=%q err=%v", after.mode, after.err)
				}
			},
		},
		{
			name:  "move_up",
			key:   "=",
			setup: func(_ *testing.T, m *Model) { m.columnCursors[m.columnCursor] = 1 },
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.mode != "list" || after.err != nil || after.message == "" {
					t.Fatalf("expected move up message with err nil, got mode=%q msg=%q err=%v", after.mode, after.message, after.err)
				}
			},
		},
		{
			name: "move_down",
			key:  "-",
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.mode != "list" || after.err != nil || after.message == "" {
					t.Fatalf("expected move down message with err nil, got mode=%q msg=%q err=%v", after.mode, after.message, after.err)
				}
			},
		},
		{
			name: "copy",
			key:  "C",
			verify: func(t *testing.T, _ Model, after Model, cmd tea.Cmd) {
				if after.mode != "add" || len(after.inputs) < 3 || cmd == nil {
					t.Fatalf("expected add mode with inputs and cmd, got mode=%q inputs=%d cmd=%v", after.mode, len(after.inputs), cmd)
				}
				// initForm 会消费 copyFromProvider 并清空它
				if after.copyFromProvider != nil {
					t.Fatalf("expected copyFromProvider cleared, got %v", after.copyFromProvider)
				}
				if after.inputs[1].Value() != "token" {
					t.Fatalf("expected copied token to prefill, got %q", after.inputs[1].Value())
				}
			},
		},
		{
			name: "backup",
			key:  "b",
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.mode != "list" || (after.message == "" && after.err == nil) {
					t.Fatalf("expected backup result (message or err) in list mode, got mode=%q msg=%q err=%v", after.mode, after.message, after.err)
				}
			},
		},
		{
			name: "backup_list",
			key:  "l",
			setup: func(t *testing.T, m *Model) {
				backupID, err := backup.CreateBackup(m.configPath)
				if err != nil {
					t.Fatalf("CreateBackup() error = %v", err)
				}
				if backupID == "" {
					t.Fatalf("CreateBackup() returned empty backupID")
				}
			},
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.mode != "backup_list" || after.err != nil {
					t.Fatalf("expected backup_list mode with err nil, got mode=%q err=%v", after.mode, after.err)
				}
			},
		},
		{
			name: "template_manager",
			key:  "m",
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.mode != "template_manager" || after.templateMode != "list" || after.err != nil {
					t.Fatalf("expected template_manager(list) with err nil, got mode=%q templateMode=%q err=%v", after.mode, after.templateMode, after.err)
				}
			},
		},
		{
			name: "mcp_manager",
			key:  "M",
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.mode != "mcp_manager" || after.mcpMode != "list" || after.err != nil {
					t.Fatalf("expected mcp_manager(list) with err nil, got mode=%q mcpMode=%q err=%v", after.mode, after.mcpMode, after.err)
				}
			},
		},
		{
			name: "portable",
			key:  "p",
			verify: func(t *testing.T, before Model, after Model, _ tea.Cmd) {
				if after.mode != "list" || after.viewMode != "multi" || (after.isPortableMode == before.isPortableMode && after.err == nil) {
					t.Fatalf("expected portable mode to toggle or err set, got before=%v after=%v err=%v", before.isPortableMode, after.isPortableMode, after.err)
				}
			},
		},
		{
			name: "check_update",
			key:  "u",
			verify: func(t *testing.T, _ Model, after Model, cmd tea.Cmd) {
				if after.mode != "list" || after.err != nil || after.message == "" || cmd == nil {
					t.Fatalf("expected list mode, message set, err nil and cmd, got mode=%q msg=%q err=%v cmd=%v", after.mode, after.message, after.err, cmd)
				}
			},
		},
		{
			name: "download_no_release",
			key:  "U",
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.mode != "list" || after.err == nil {
					t.Fatalf("expected list mode with err set, got mode=%q err=%v", after.mode, after.err)
				}
			},
		},
		{
			name: "refresh",
			key:  "r",
			verify: func(t *testing.T, _ Model, after Model, cmd tea.Cmd) {
				if after.mode != "list" || after.err != nil || after.message == "" || cmd == nil {
					t.Fatalf("expected list mode, message set, err nil and cmd, got mode=%q msg=%q err=%v cmd=%v", after.mode, after.message, after.err, cmd)
				}
			},
		},
		{
			name: "switch_view",
			key:  "v",
			verify: func(t *testing.T, _ Model, after Model, _ tea.Cmd) {
				if after.mode != "list" || after.viewMode != "single" || after.err != nil {
					t.Fatalf("expected list mode with viewMode=single and err nil, got mode=%q viewMode=%q err=%v", after.mode, after.viewMode, after.err)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mm := m
			mm.mode = "list"
			mm.viewMode = "multi"
			mm.columnCursor = 0
			mm.refreshAllColumns()
			if tc.setup != nil {
				tc.setup(t, &mm)
			}
			updatedModel, cmd := mm.handleMultiColumnKeys(makeKey(tc.key))
			updated, ok := updatedModel.(Model)
			if !ok {
				t.Fatalf("expected Model, got %T", updatedModel)
			}
			tc.verify(t, mm, updated, cmd)
		})
	}
}

func TestHandleListKeysErrorBranches(t *testing.T) {
	manager := newTestManager(t)
	addProvider(t, manager, "claude", "p1", "Claude One")
	m := Model{manager: manager, mode: "list", currentApp: "claude", configPath: manager.GetConfigPath()}

	m.providers = []config.Provider{
		{ID: "p1", Name: "Claude One"},
		{ID: "missing", Name: "Missing"},
	}
	m.cursor = 1
	updated, _ := m.handleListKeys(makeKey("="))
	m = updated.(Model)
	if m.err == nil {
		t.Fatalf("expected move up error")
	}
	m.providers = []config.Provider{
		{ID: "missing", Name: "Missing"},
		{ID: "p1", Name: "Claude One"},
	}
	m.cursor = 0
	updated, _ = m.handleListKeys(makeKey("-"))
	m = updated.(Model)
	if m.err == nil {
		t.Fatalf("expected move down error")
	}
	updated, _ = m.handleListKeys(makeKey("enter"))
	m = updated.(Model)
	if m.err == nil {
		t.Fatalf("expected switch error")
	}

	badBackupDir := t.TempDir()
	backupPath := filepath.Join(badBackupDir, backup.BackupDirName)
	_ = os.WriteFile(backupPath, []byte("not a dir"), 0644)
	m.configPath = filepath.Join(badBackupDir, "config.json")
	updated, _ = m.handleListKeys(makeKey("l"))
	m = updated.(Model)
	if m.err == nil {
		t.Fatalf("expected backup list error")
	}

	updated, _ = m.handleListKeys(makeKey("b"))
	m = updated.(Model)
	if m.err == nil {
		t.Fatalf("expected backup error")
	}
}

func TestHandleMultiColumnKeysErrorBranches(t *testing.T) {
	manager := newTestManager(t)
	m := Model{manager: manager, mode: "list", viewMode: "multi", currentApp: "claude", configPath: manager.GetConfigPath()}
	m.columnProviders[0] = []config.Provider{
		{ID: "p1", Name: "Claude One"},
		{ID: "missing", Name: "Missing"},
	}
	m.columnCursor = 0
	m.columnCursors[0] = 1

	updated, _ := m.handleMultiColumnKeys(makeKey("="))
	m = updated.(Model)
	if m.err == nil {
		t.Fatalf("expected move error")
	}
	m.columnProviders[0] = []config.Provider{
		{ID: "missing", Name: "Missing"},
		{ID: "p1", Name: "Claude One"},
	}
	m.columnCursors[0] = 0
	updated, _ = m.handleMultiColumnKeys(makeKey("enter"))
	m = updated.(Model)
	if m.err == nil {
		t.Fatalf("expected switch error")
	}

	badBackupDir := t.TempDir()
	backupPath := filepath.Join(badBackupDir, backup.BackupDirName)
	_ = os.WriteFile(backupPath, []byte("not a dir"), 0644)
	m.configPath = filepath.Join(badBackupDir, "config.json")
	updated, _ = m.handleMultiColumnKeys(makeKey("l"))
	m = updated.(Model)
	if m.err == nil {
		t.Fatalf("expected backup list error")
	}
}

func TestSubmitFormVariants(t *testing.T) {
	cases := []struct {
		name    string
		setup   func(m *Model)
		wantErr bool
	}{
		{
			name: "codex_missing_model",
			setup: func(m *Model) {
				m.currentApp = "codex"
				m.mode = "add"
				m.initForm(nil)
				m.inputs[0].SetValue("Codex")
				m.inputs[1].SetValue("token")
				m.inputs[2].SetValue("https://api.example.com")
				if len(m.inputs) > 4 {
					m.inputs[4].SetValue("")
				}
				m.inputs[5].SetValue("high")
			},
			wantErr: true,
		},
		{
			name: "codex_missing_reasoning",
			setup: func(m *Model) {
				m.currentApp = "codex"
				m.mode = "add"
				m.initForm(nil)
				m.inputs[0].SetValue("Codex")
				m.inputs[1].SetValue("token")
				m.inputs[2].SetValue("https://api.example.com")
				m.inputs[4].SetValue("gpt-5")
				if len(m.inputs) > 5 {
					m.inputs[5].SetValue("")
				}
			},
			wantErr: true,
		},
		{
			name: "gemini_missing_fields",
			setup: func(m *Model) {
				m.currentApp = "gemini"
				m.mode = "add"
				m.initForm(nil)
				m.inputs[0].SetValue("Gemini")
			},
			wantErr: true,
		},
		{
			name: "edit_provider",
			setup: func(m *Model) {
				m.currentApp = "claude"
				addProvider(t, m.manager, "claude", "p1", "Claude One")
				m.refreshProviders()
				m.currentApp = "claude"
				m.mode = "edit"
				m.editName = "Claude One"
				provider := m.providers[0]
				m.initForm(&provider)
				m.inputs[0].SetValue("Claude Renamed")
				m.inputs[1].SetValue("token")
				m.inputs[2].SetValue("https://api.anthropic.com")
			},
			wantErr: false,
		},
		{
			name: "codex_success",
			setup: func(m *Model) {
				m.currentApp = "codex"
				m.mode = "add"
				m.initForm(nil)
				m.inputs[0].SetValue("Codex")
				m.inputs[1].SetValue("token")
				m.inputs[2].SetValue("https://api.example.com")
				m.inputs[4].SetValue("gpt-5")
				if len(m.inputs) > 5 {
					m.inputs[5].SetValue("high")
				}
			},
			wantErr: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := Model{manager: newTestManager(t)}
			m.refreshProviders()
			tc.setup(&m)
			m.submitForm()
			if tc.wantErr && m.err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func TestUpdateAndViewDispatchSweep(t *testing.T) {
	manager := newTestManager(t)
	addProvider(t, manager, "claude", "p1", "Claude One")
	if err := manager.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	m := Model{manager: manager, configPath: manager.GetConfigPath(), templateManager: newTestTemplateManager(t)}

	modes := []struct {
		name             string
		setup            func(*Model)
		wantMode         string
		wantTemplateMode string
		wantMcpMode      string
	}{
		{name: "list", setup: func(m *Model) { m.mode = "list" }, wantMode: "list"},
		{name: "add", setup: func(m *Model) { m.currentApp = "claude"; m.mode = "add"; m.initForm(nil) }, wantMode: "list"},
		{name: "edit", setup: func(m *Model) { m.currentApp = "claude"; m.mode = "edit"; m.initForm(nil) }, wantMode: "list"},
		{name: "delete", setup: func(m *Model) { m.mode = "delete"; m.deleteName = "Claude" }, wantMode: "list"},
		{name: "app_select", setup: func(m *Model) { m.mode = "app_select" }, wantMode: "list"},
		{name: "backup_list", setup: func(m *Model) { m.mode = "backup_list" }, wantMode: "list"},
		{name: "template_list", setup: func(m *Model) { m.mode = "template_manager"; m.templateMode = "list" }, wantMode: "list"},
		{name: "template_preview", setup: func(m *Model) {
			m.mode = "template_manager"
			m.templateMode = "preview"
			m.selectedTemplate = &template.Template{Name: "T", Content: "c"}
		}, wantMode: "template_manager", wantTemplateMode: "list"},
		{name: "template_apply", setup: func(m *Model) {
			m.mode = "template_manager"
			m.templateMode = "apply_preview_diff"
			m.selectedTemplate = &template.Template{Name: "T", Content: "c"}
		}, wantMode: "template_manager", wantTemplateMode: "apply_select_target"},
		{name: "template_save", setup: func(m *Model) {
			m.mode = "template_manager"
			m.templateMode = "save_input_name"
			m.saveNameInput = textinput.New()
		}, wantMode: "template_manager", wantTemplateMode: "save_select_source"},
		{name: "template_delete", setup: func(m *Model) {
			m.mode = "template_manager"
			m.templateMode = "delete_confirm"
			m.selectedTemplate = &template.Template{Name: "T"}
		}, wantMode: "template_manager", wantTemplateMode: "list"},
		{name: "mcp_list", setup: func(m *Model) { m.mode = "mcp_manager"; m.mcpMode = "list" }, wantMode: "list"},
		{name: "mcp_form", setup: func(m *Model) { m.mode = "mcp_manager"; m.mcpMode = "add"; m.initMcpForm(nil) }, wantMode: "mcp_manager", wantMcpMode: "list"},
		{name: "mcp_delete", setup: func(m *Model) {
			m.mode = "mcp_manager"
			m.mcpMode = "delete"
			m.selectedMcp = &config.McpServer{Name: "X"}
		}, wantMode: "mcp_manager", wantMcpMode: "list"},
		{name: "mcp_apps", setup: func(m *Model) {
			m.mode = "mcp_manager"
			m.mcpMode = "apps_toggle"
			m.selectedMcp = &config.McpServer{Name: "X"}
		}, wantMode: "mcp_manager", wantMcpMode: "list"},
		{name: "mcp_preset", setup: func(m *Model) { m.mode = "mcp_manager"; m.mcpMode = "preset"; m.mcpPresets = config.GetMcpPresets() }, wantMode: "mcp_manager", wantMcpMode: "list"},
	}

	for _, tc := range modes {
		t.Run(tc.name, func(t *testing.T) {
			mm := m
			mm.refreshProviders()
			tc.setup(&mm)
			updatedModel, _ := mm.Update(makeKey("esc"))
			updated, ok := updatedModel.(Model)
			if !ok {
				t.Fatalf("expected Model, got %T", updatedModel)
			}
			if updated.mode != tc.wantMode {
				t.Fatalf("expected mode=%q, got %q", tc.wantMode, updated.mode)
			}
			if tc.wantTemplateMode != "" && updated.templateMode != tc.wantTemplateMode {
				t.Fatalf("expected templateMode=%q, got %q", tc.wantTemplateMode, updated.templateMode)
			}
			if tc.wantMcpMode != "" && updated.mcpMode != tc.wantMcpMode {
				t.Fatalf("expected mcpMode=%q, got %q", tc.wantMcpMode, updated.mcpMode)
			}
			view := updated.View()
			if view == "" {
				t.Fatalf("expected non-empty view")
			}
		})
	}
}

func TestBackupListKeysSweep(t *testing.T) {
	manager := newTestManager(t)
	addProvider(t, manager, "claude", "p1", "Claude One")
	if err := manager.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	configPath := manager.GetConfigPath()
	backupID, err := backup.CreateBackup(configPath)
	if err != nil {
		t.Fatalf("CreateBackup() error = %v", err)
	}
	if backupID == "" {
		t.Fatalf("CreateBackup() returned empty backupID")
	}
	backups, err := backup.ListBackups(filepath.Dir(configPath))
	if err != nil {
		t.Fatalf("ListBackups() error = %v", err)
	}

	m := Model{manager: manager, mode: "backup_list", configPath: configPath, backupList: backups}
	keys := []string{"up", "down", "esc"}
	for _, key := range keys {
		updated := testutil.BubbleTeaTestHelper(t, m, []string{key})
		m = updated.(Model)
	}
}

func TestMcpFormKeysExtra(t *testing.T) {
	manager := newTestManager(t)
	m := Model{manager: manager, mode: "mcp_manager", mcpMode: "add"}
	m.initMcpForm(nil)
	m.mcpInputs[0].SetValue("id")
	m.mcpInputs[1].SetValue("Name")
	m.mcpInputs[2].SetValue("npx")
	m.mcpInputs[3].SetValue("mcp-server-fetch")

	_, updated, _ := m.handleMcpFormKeys(makeKey("ctrl+t"))
	m = updated.(Model)
	_, updated, _ = m.handleMcpFormKeys(makeKey("esc"))
	m = updated.(Model)
	_, _, _ = m.handleMcpFormKeys(makeKey("ctrl+s"))
}

func TestHandleFormKeysSelectorNavigation(t *testing.T) {
	m := Model{manager: newTestManager(t), currentApp: "codex", mode: "add"}
	m.initForm(nil)
	m.focusIndex = 4

	_, updated, _ := m.handleFormKeys(makeKey(" "))
	m = updated.(Model)
	_, updated, _ = m.handleFormKeys(makeKey("down"))
	m = updated.(Model)
	_, updated, _ = m.handleFormKeys(makeKey("up"))
	m = updated.(Model)
	_, updated, _ = m.handleFormKeys(makeKey("enter"))
	m = updated.(Model)
	if m.modelSelectorActive {
		t.Fatalf("expected selector to close")
	}

	m.focusIndex = 0
	_, updated, _ = m.handleFormKeys(makeKey("down"))
	m = updated.(Model)
	_, updated, _ = m.handleFormKeys(makeKey("up"))
	m = updated.(Model)
}

func TestTemplateSaveBranches(t *testing.T) {
	testutil.WithTempHome(t, func(home string) {
		testutil.WithTempCWD(t, func(cwd string) {
			_ = home
			_ = cwd
			tm := newTestTemplateManager(t)
			id, _ := tm.AddTemplate("User", template.CategoryClaudeMd, "content")
			tpl, _ := tm.GetTemplate(id)
			m := Model{manager: newTestManager(t), templateManager: tm, selectedTemplate: tpl}
			m.templateMode = "save_select_source"

			updated, _ := m.handleSourceSelectKeys(makeKey("up"))
			m = updated.(Model)
			updated, _ = m.handleSourceSelectKeys(makeKey("down"))
			m = updated.(Model)

			m.templateMode = "save_input_name"
			m.saveNameInput = textinput.New()
			updated, _ = m.handleSaveNameKeys(makeKey("enter"))
			m = updated.(Model)

			m.selectedTemplate = tpl
			m.selectedTargetPath = filepath.Join(cwd, "CLAUDE.md")
			m.diffContent = "未发现差异"
			_ = m.viewDiffPreview()
		})
	})
}

func TestListMultiRowVariants(t *testing.T) {
	manager := newTestManager(t)
	addProvider(t, manager, "claude", "p1", strings.Repeat("长名字", 20))
	m := Model{manager: manager, currentApp: "claude", viewMode: "multi"}
	m.refreshAllColumns()
	m.columnProviders[1] = nil
	m.columnProviders[2] = nil
	_ = m.viewListMulti()
}

func TestFormSelectorOptionsCoverage(t *testing.T) {
	cases := []struct {
		name  string
		app   string
		index int
	}{
		{name: "claude_primary", app: "claude", index: 4},
		{name: "claude_haiku", app: "claude", index: 5},
		{name: "claude_sonnet", app: "claude", index: 6},
		{name: "claude_opus", app: "claude", index: 7},
		{name: "codex_model", app: "codex", index: 4},
		{name: "codex_reasoning", app: "codex", index: 5},
		{name: "gemini_model", app: "gemini", index: 3},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := Model{currentApp: tc.app}
			opts := m.selectorOptions(tc.index)
			if tc.app != "claude" && tc.app != "codex" && tc.app != "gemini" {
				return
			}
			if opts == nil {
				t.Fatalf("expected selector options")
			}
		})
	}
}

func TestHandlePreviewKeysSweep(t *testing.T) {
	m := Model{templateMode: "preview", selectedTemplate: &template.Template{Content: strings.Repeat("line\n", 40)}}
	keys := []string{"down", "up", "pgdown", "pgup", "q"}
	for _, key := range keys {
		updated, _ := m.handlePreviewKeys(makeKey(key))
		m = updated.(Model)
	}
}

func TestHandleDeleteKeysSweep(t *testing.T) {
	manager := newTestManager(t)
	addProvider(t, manager, "claude", "p1", "Claude One")
	addProvider(t, manager, "claude", "p2", "Claude Two")
	m := Model{manager: manager, mode: "delete", currentApp: "claude", deleteName: "Claude Two"}
	keys := []string{"n", "y"}
	for _, key := range keys {
		updated, _ := m.handleDeleteKeys(makeKey(key))
		m = updated.(Model)
	}
}

func TestHandleAppSelectKeysSweep(t *testing.T) {
	m := Model{mode: "app_select", manager: newTestManager(t)}
	keys := []string{"up", "down", "esc", "enter"}
	for _, key := range keys {
		updated, _ := m.handleAppSelectKeys(makeKey(key))
		m = updated.(Model)
	}
}

func TestHandleMcpPresetKeysSweep(t *testing.T) {
	m := Model{mcpMode: "preset", mcpPresets: config.GetMcpPresets()}
	keys := []string{"up", "down", "enter", "esc"}
	for _, key := range keys {
		updated, _ := m.handleMcpPresetKeys(makeKey(key))
		m = updated.(Model)
	}
}

func TestHandleMcpDeleteKeysSweep(t *testing.T) {
	manager := newTestManager(t)
	server := addMcpServer(t, manager, "srv", config.McpApps{Claude: true})
	m := Model{manager: manager, mcpMode: "delete", selectedMcp: &server}
	keys := []string{"n", "y"}
	for _, key := range keys {
		updated, _ := m.handleMcpDeleteKeys(makeKey(key))
		m = updated.(Model)
	}
}

func TestHandleMcpDeleteKeysMissing(t *testing.T) {
	manager := newTestManager(t)
	m := Model{manager: manager, mcpMode: "delete", selectedMcp: &config.McpServer{ID: "missing"}}
	updated, _ := m.handleMcpDeleteKeys(makeKey("y"))
	m = updated.(Model)
	if m.err == nil {
		t.Fatalf("expected delete error")
	}
}

func TestHandleMcpAppsToggleKeysSweep(t *testing.T) {
	manager := newTestManager(t)
	server := addMcpServer(t, manager, "srv", config.McpApps{Claude: true})
	m := Model{manager: manager, mcpMode: "apps_toggle", selectedMcp: &server}
	m.mcpAppsToggle = server.Apps
	keys := []string{"up", "down", " ", "enter"}
	for _, key := range keys {
		updated, _ := m.handleMcpAppsToggleKeys(makeKey(key))
		m = updated.(Model)
	}
}

func TestHandleSourceSelectKeysExisting(t *testing.T) {
	testutil.WithTempHome(t, func(home string) {
		testutil.WithTempCWD(t, func(_ string) {
			path := filepath.Join(home, ".claude", "CLAUDE.md")
			_ = os.MkdirAll(filepath.Dir(path), 0755)
			_ = os.WriteFile(path, []byte("content"), 0644)

			tm := newTestTemplateManager(t)
			m := Model{templateManager: tm, currentApp: "claude", templateMode: "save_select_source"}
			keys := []string{"down", "up", "enter"}
			for _, key := range keys {
				updated, _ := m.handleSourceSelectKeys(makeKey(key))
				m = updated.(Model)
			}
		})
	})
}

func TestUpdateInputsReadonlyPath(t *testing.T) {
	m := Model{currentApp: "claude", mode: "add"}
	m.initForm(nil)
	m.focusIndex = 0
	_, _ = m.updateInputs(makeKey("backspace"))
	_, _ = m.updateInputs(makeKey("delete"))
}

func TestTickCmdExecution(t *testing.T) {
	cmd := tickCmd()
	if cmd == nil {
		t.Fatalf("expected tick cmd")
	}
	_ = cmd()
}

func TestBackupViewEmptyList(t *testing.T) {
	m := Model{backupList: nil}
	_ = m.viewBackupList()
}

func TestMcpViewEmptyList(t *testing.T) {
	m := Model{mcpServers: nil}
	_ = m.viewMcpList()
}

func TestRenderTableRowEmpty(t *testing.T) {
	manager := newTestManager(t)
	m := Model{manager: manager}
	m.columnProviders = [3][]config.Provider{{}, {}, {}}
	_ = m.renderTableRow(0, []int{10, 10, 10})
}

func TestRenderTableRowTruncate(t *testing.T) {
	manager := newTestManager(t)
	addProvider(t, manager, "claude", "p1", strings.Repeat("长名字", 20))
	m := Model{manager: manager}
	m.columnProviders[0] = manager.ListProvidersForApp("claude")
	m.columnCursor = 0
	m.columnCursors[0] = 0
	_ = m.renderTableRow(0, []int{10, 10, 10})
	_ = m.renderTableRow(3, []int{10, 10, 10})
}

func TestTemplateViewDiffNoFile(t *testing.T) {
	m := Model{selectedTemplate: &template.Template{Name: "T", Content: strings.Repeat("line\n", 30)}, selectedTargetPath: "missing"}
	_ = m.viewDiffPreview()
}

func TestTemplateViewSaveNameDefault(t *testing.T) {
	m := Model{templateManager: newTestTemplateManager(t), templateMode: "save_input_name"}
	m.saveNameInput = textinput.New()
	_ = m.viewSaveNameInput()
}

func TestMcpFormSaveHttp(t *testing.T) {
	manager := newTestManager(t)
	m := Model{manager: manager, mcpMode: "add"}
	m.initMcpForm(nil)
	m.mcpConnType = "http"
	m.mcpInputs[0].SetValue("id")
	m.mcpInputs[1].SetValue("Name")
	m.mcpInputs[4].SetValue("https://example.com/mcp")
	_ = m.saveMcpForm()
}

func TestFormSelectorSpaceNoOptions(t *testing.T) {
	m := Model{currentApp: "claude", mode: "add"}
	m.initForm(nil)
	m.focusIndex = 0
	_, updated, _ := m.handleFormKeys(makeKey(" "))
	_ = updated.(Model)
}

func TestTemplateApplyScrollKeys(t *testing.T) {
	m := Model{templateMode: "apply_preview_diff", diffContent: strings.Repeat("line\n", 40), selectedTemplate: &template.Template{Name: "T"}}
	keys := []string{"up", "down", "pgup", "pgdown"}
	for _, key := range keys {
		updated, _ := m.handleDiffPreviewKeys(makeKey(key))
		m = updated.(Model)
	}
}

func TestFormClearUndoEmpty(t *testing.T) {
	testutil.WithTempHome(t, func(_ string) {
		m := Model{currentApp: "claude"}
		m.initForm(nil)
		for i := range m.inputs {
			m.inputs[i].SetValue("")
		}
		m.clearFormFields()
		if m.undoLastClear() {
			t.Fatalf("expected no undo history")
		}
	})
}

func TestMcpUpdateInputsShiftTab(t *testing.T) {
	m := Model{mcpMode: "add"}
	m.initMcpForm(nil)
	m.mcpFocusIndex = 0
	updated, _ := m.updateMcpInputs(makeKey("shift+tab"))
	_ = updated.(Model)
}

func TestTemplateListViewEmpty(t *testing.T) {
	m := Model{mode: "template_manager", templateMode: "list", templates: nil}
	_ = m.viewTemplateList()
}

func TestFormLabelsDefaults(t *testing.T) {
	m := Model{currentApp: "unknown"}
	labels := m.formLabels()
	if len(labels) == 0 {
		t.Fatalf("expected default labels")
	}
}

func TestViewFormEmpty(t *testing.T) {
	m := Model{currentApp: "claude", mode: "add"}
	m.initForm(nil)
	_ = m.viewForm()
}

func TestViewTargetSelectNoTargets(t *testing.T) {
	m := Model{selectedTemplate: &template.Template{Name: "T", Category: "unknown"}, targetSelectCursor: 0}
	_ = m.viewTargetSelect()
}

func TestHandleMcpAppsToggleKeysNewServer(t *testing.T) {
	manager := newTestManager(t)
	preset := config.GetMcpPresets()[0]
	m := Model{manager: manager, mcpMode: "apps_toggle", selectedMcp: &preset}
	m.mcpAppsToggle = config.McpApps{Claude: true}
	_, _ = m.handleMcpAppsToggleKeys(makeKey("enter"))
}

func TestHandleBackupListKeysEmpty(t *testing.T) {
	m := Model{mode: "backup_list", backupList: nil}
	keys := []string{"up", "down", "esc"}
	for _, key := range keys {
		updated, _ := m.handleBackupListKeys(makeKey(key))
		m = updated.(Model)
	}
}

func TestViewListEmptyProviders(t *testing.T) {
	m := Model{currentApp: "claude"}
	_ = m.viewList()
}

func TestViewListMultiEmpty(t *testing.T) {
	m := Model{manager: newTestManager(t), viewMode: "multi"}
	_ = m.viewListMulti()
}

func TestMcpPresetViewEmpty(t *testing.T) {
	m := Model{mcpPresets: nil}
	_ = m.viewMcpPreset()
}

func TestUpdateWithWindowSize(t *testing.T) {
	m := Model{}
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	_ = updated.(Model)
}

func TestUpdateTickMsg(t *testing.T) {
	m := Model{}
	updated, _ := m.Update(tickMsg(time.Now()))
	_ = updated.(Model)
}

func TestUpdateKeyMsgRouting(t *testing.T) {
	manager := newTestManager(t)
	m := Model{manager: manager, mode: "list"}
	updated, _ := m.Update(makeKey("q"))
	_ = updated.(Model)
}

func TestHandleTargetSelectKeysBranches(t *testing.T) {
	testutil.WithTempHome(t, func(_ string) {
		testutil.WithTempCWD(t, func(_ string) {
			tm := newTestTemplateManager(t)
			templateID, err := tm.AddTemplate("T", template.CategoryClaudeMd, "content")
			if err != nil {
				t.Fatalf("AddTemplate() error = %v", err)
			}
			tpl, err := tm.GetTemplate(templateID)
			if err != nil {
				t.Fatalf("GetTemplate() error = %v", err)
			}

			cases := []struct {
				name   string
				key    string
				setup  func(*Model)
				verify func(*testing.T, Model)
			}{
				{
					name: "esc_back_to_list",
					key:  "esc",
					verify: func(t *testing.T, m Model) {
						if m.templateMode != "list" || m.selectedTemplate != nil {
							t.Fatalf("expected back to list")
						}
					},
				},
				{
					name:  "no_template_selected",
					key:   "enter",
					setup: func(m *Model) { m.selectedTemplate = nil },
					verify: func(t *testing.T, m Model) {
						if m.err == nil {
							t.Fatalf("expected error")
						}
					},
				},
				{
					name: "invalid_category",
					key:  "enter",
					setup: func(m *Model) {
						m.selectedTemplate = &template.Template{ID: "bad", Name: "Bad", Category: "unknown"}
					},
					verify: func(t *testing.T, m Model) {
						if m.err == nil {
							t.Fatalf("expected category error")
						}
					},
				},
				{
					name: "missing_template_id",
					key:  "enter",
					setup: func(m *Model) {
						m.selectedTemplate = &template.Template{ID: "missing", Name: "Missing", Category: template.CategoryClaudeMd}
					},
					verify: func(t *testing.T, m Model) {
						if m.err == nil {
							t.Fatalf("expected diff error")
						}
					},
				},
				{
					name:  "cursor_clamp_and_down",
					key:   "down",
					setup: func(m *Model) { m.targetSelectCursor = 99 },
					verify: func(t *testing.T, m Model) {
						targets, _ := template.GetTargetsForCategory(template.CategoryClaudeMd)
						if len(targets) == 0 {
							t.Fatalf("expected targets")
						}
						if m.targetSelectCursor < 0 || m.targetSelectCursor >= len(targets) {
							t.Fatalf("cursor out of range")
						}
					},
				},
				{
					name:  "cursor_up_wrap",
					key:   "up",
					setup: func(m *Model) { m.targetSelectCursor = 0 },
					verify: func(t *testing.T, m Model) {
						targets, _ := template.GetTargetsForCategory(template.CategoryClaudeMd)
						if len(targets) == 0 {
							t.Fatalf("expected targets")
						}
						if m.targetSelectCursor != len(targets)-1 {
							t.Fatalf("expected wrap to last")
						}
					},
				},
				{
					name: "enter_success",
					key:  "enter",
					verify: func(t *testing.T, m Model) {
						if m.templateMode != "apply_preview_diff" {
							t.Fatalf("expected apply preview mode")
						}
						if m.diffContent == "" {
							t.Fatalf("expected diff content")
						}
					},
				},
			}

			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					m := Model{
						mode:             "template_manager",
						templateMode:     "apply_select_target",
						templateManager:  tm,
						selectedTemplate: tpl,
					}
					if tc.setup != nil {
						tc.setup(&m)
					}
					updated, _ := m.handleTargetSelectKeys(makeKey(tc.key))
					m = updated.(Model)
					tc.verify(t, m)
				})
			}
		})
	})
}

func TestHandleSourceSelectKeysBranches(t *testing.T) {
	testutil.WithTempHome(t, func(home string) {
		testutil.WithTempCWD(t, func(_ string) {
			tm := newTestTemplateManager(t)

			m := Model{mode: "template_manager", templateMode: "save_select_source", currentApp: "claude", templateManager: tm}
			updated, _ := m.handleSourceSelectKeys(makeKey("esc"))
			m = updated.(Model)
			if m.templateMode != "list" {
				t.Fatalf("expected back to list")
			}

			m = Model{mode: "template_manager", templateMode: "save_select_source", currentApp: "claude", templateManager: tm}
			updated, _ = m.handleSourceSelectKeys(makeKey("enter"))
			m = updated.(Model)
			if m.err == nil {
				t.Fatalf("expected missing file error")
			}

			targetPath := filepath.Join(home, ".claude", "CLAUDE.md")
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				t.Fatalf("MkdirAll() error = %v", err)
			}
			if err := os.WriteFile(targetPath, []byte("content"), 0644); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			m = Model{mode: "template_manager", templateMode: "save_select_source", currentApp: "claude", templateManager: tm, sourceSelectCursor: 1}
			updated = testutil.BubbleTeaTestHelper(t, m, []string{"down", "enter"})
			m = updated.(Model)
			if m.templateMode != "save_input_name" {
				t.Fatalf("expected save input mode")
			}
			if m.selectedSourcePath == "" {
				t.Fatalf("expected selected source path")
			}
		})
	})
}

func TestHandleTemplateListKeysBranches(t *testing.T) {
	tm := newTestTemplateManager(t)
	m := Model{mode: "template_manager", templateMode: "list", currentApp: "claude", templateManager: tm}
	m.refreshTemplates()
	if len(m.templates) == 0 {
		t.Fatalf("expected templates")
	}

	builtinIndex := -1
	for i, tpl := range m.templates {
		if tpl.IsBuiltin {
			builtinIndex = i
			break
		}
	}
	if builtinIndex == -1 {
		t.Fatalf("expected builtin template")
	}

	cases := []struct {
		name   string
		key    string
		setup  func(*Model)
		verify func(*testing.T, Model)
	}{
		{
			name: "preview",
			key:  "p",
			verify: func(t *testing.T, m Model) {
				if m.templateMode != "preview" || m.selectedTemplate == nil {
					t.Fatalf("expected preview mode")
				}
			},
		},
		{
			name: "save",
			key:  "s",
			verify: func(t *testing.T, m Model) {
				if m.templateMode != "save_select_source" {
					t.Fatalf("expected save mode")
				}
			},
		},
		{
			name:  "delete_builtin",
			key:   "d",
			setup: func(m *Model) { m.templateCursor = builtinIndex },
			verify: func(t *testing.T, m Model) {
				if m.err == nil {
					t.Fatalf("expected delete error")
				}
			},
		},
		{
			name: "refresh",
			key:  "r",
			verify: func(t *testing.T, m Model) {
				if m.message == "" {
					t.Fatalf("expected refresh message")
				}
			},
		},
		{
			name: "escape",
			key:  "esc",
			verify: func(t *testing.T, m Model) {
				if m.mode != "list" {
					t.Fatalf("expected list mode")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mm := m
			mm.mode = "template_manager"
			mm.templateMode = "list"
			mm.refreshTemplates()
			if tc.setup != nil {
				tc.setup(&mm)
			}
			updated, _ := mm.handleTemplateListKeys(makeKey(tc.key))
			mm = updated.(Model)
			tc.verify(t, mm)
		})
	}
}

func TestHandleMcpListKeysExtra(t *testing.T) {
	manager := newTestManager(t)
	m := Model{manager: manager, mode: "mcp_manager", mcpMode: "list"}

	updated, _ := m.handleMcpListKeys(makeKey("s"))
	m = updated.(Model)
	if m.message == "" {
		t.Fatalf("expected sync message")
	}

	updated, _ = m.handleMcpListKeys(makeKey("p"))
	m = updated.(Model)
	if m.mcpMode != "preset" || len(m.mcpPresets) == 0 {
		t.Fatalf("expected preset list")
	}
}

func TestHandleMcpAppsToggleKeysEsc(t *testing.T) {
	manager := newTestManager(t)
	server := addMcpServer(t, manager, "srv-esc", config.McpApps{Claude: true})
	m := Model{manager: manager, mcpMode: "apps_toggle", selectedMcp: &server}
	updated, _ := m.handleMcpAppsToggleKeys(makeKey("esc"))
	m = updated.(Model)
	if m.mcpMode != "list" || m.selectedMcp != nil {
		t.Fatalf("expected exit apps toggle")
	}
}

func TestViewMcpListDescriptions(t *testing.T) {
	m := Model{
		mcpServers: []config.McpServer{
			{
				ID:          "srv1",
				Name:        "Server One",
				Description: "",
				Apps:        config.McpApps{Claude: true},
			},
			{
				ID:          "srv2",
				Name:        "Server Two",
				Description: "desc",
				Apps:        config.McpApps{Codex: true},
			},
		},
		mcpCursor: 0,
	}
	view := m.viewMcpList()
	if !strings.Contains(view, "无描述") {
		t.Fatalf("expected fallback description")
	}
}

func TestSelectorTitleCoverage(t *testing.T) {
	cases := []struct {
		app   string
		index int
	}{
		{app: "claude", index: 4},
		{app: "claude", index: 5},
		{app: "claude", index: 6},
		{app: "claude", index: 7},
		{app: "codex", index: 4},
		{app: "codex", index: 5},
		{app: "gemini", index: 3},
		{app: "unknown", index: 1},
	}

	for _, tc := range cases {
		m := Model{currentApp: tc.app}
		_ = m.selectorTitle(tc.index)
	}
}

func TestClearFormFieldsAndUndo(t *testing.T) {
	m := Model{currentApp: "codex"}
	m.initForm(nil)
	m.inputs[0].SetValue("Name")
	m.inputs[1].SetValue("token")
	m.inputs[2].SetValue("https://api.example.com")

	m.clearFormFields()
	if len(m.undoHistory) == 0 {
		t.Fatalf("expected undo history")
	}
	if len(m.inputs) > 5 {
		if m.inputs[4].Value() == "" || m.inputs[5].Value() == "" {
			t.Fatalf("expected codex defaults")
		}
	}
	if !m.undoLastClear() {
		t.Fatalf("expected undo success")
	}
	if m.inputs[0].Value() != "Name" {
		t.Fatalf("expected restore values")
	}
}

func TestLoadLiveAndCodexConfigForForm(t *testing.T) {
	testutil.WithTempHome(t, func(home string) {
		claudePath := filepath.Join(home, ".claude", "settings.json")
		if err := os.MkdirAll(filepath.Dir(claudePath), 0755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		claudeSettings := config.ClaudeSettings{
			Env: config.ClaudeEnv{
				AnthropicAuthToken:          "token",
				AnthropicBaseURL:            "https://api.anthropic.com",
				AnthropicModel:              "claude-3",
				AnthropicDefaultHaikuModel:  "haiku",
				AnthropicDefaultSonnetModel: "sonnet",
				AnthropicDefaultOpusModel:   "opus",
			},
		}
		data, err := json.Marshal(claudeSettings)
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}
		if err := os.WriteFile(claudePath, data, 0644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		m := Model{currentApp: "claude"}
		token, baseURL, primaryModel, haikuModel, sonnetModel, opusModel, loaded := m.loadLiveConfigForForm()
		if !loaded || token == "" || baseURL == "" || primaryModel == "" || haikuModel == "" || sonnetModel == "" || opusModel == "" {
			t.Fatalf("expected claude config values")
		}

		codexAuthPath := filepath.Join(home, ".codex", "auth.json")
		if err := os.MkdirAll(filepath.Dir(codexAuthPath), 0755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		if err := os.WriteFile(codexAuthPath, []byte(`{"OPENAI_API_KEY":"test-api-key-12345"}`), 0644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		codexConfigPath := filepath.Join(home, ".codex", "config.toml")
		codexConfig := strings.Join([]string{
			`base_url = "https://api.example.com"`,
			`model = "gpt-5"`,
			`model_reasoning_effort = "high"`,
		}, "\n")
		if err := os.WriteFile(codexConfigPath, []byte(codexConfig), 0644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		m = Model{currentApp: "codex"}
		token, baseURL, modelValue, reasoningValue, loaded := m.loadCodexConfigForForm()
		if !loaded || token == "" || baseURL == "" || modelValue == "" || reasoningValue == "" {
			t.Fatalf("expected codex config values")
		}
	})
}

func TestViewListActiveAndMessage(t *testing.T) {
	manager := newTestManager(t)
	p1 := addProvider(t, manager, "claude", "p1", "Claude One")
	_ = addProvider(t, manager, "claude", "p2", "Claude Two")
	if err := manager.SwitchProviderForApp("claude", p1.Name); err != nil {
		t.Fatalf("SwitchProviderForApp() error = %v", err)
	}

	m := Model{
		manager:         manager,
		currentApp:      "claude",
		providers:       manager.ListProvidersForApp("claude"),
		cursor:          0,
		message:         "ok",
		isPortableMode:  true,
		configPath:      manager.GetConfigPath(),
		templateManager: newTestTemplateManager(t),
	}
	view := m.viewList()
	if !strings.Contains(view, "便携版") {
		t.Fatalf("expected portable indicator")
	}
	if !strings.Contains(view, "●") {
		t.Fatalf("expected active marker")
	}
}

func TestViewListMultiActiveAndError(t *testing.T) {
	manager := newTestManager(t)
	addProvider(t, manager, "claude", "p1", "Claude One")
	addProvider(t, manager, "codex", "c1", "Codex One")
	addProvider(t, manager, "gemini", "g1", "Gemini One")
	if err := manager.SwitchProviderForApp("codex", "Codex One"); err != nil {
		t.Fatalf("SwitchProviderForApp() error = %v", err)
	}

	m := Model{
		manager:        manager,
		viewMode:       "multi",
		columnCursor:   1,
		isPortableMode: true,
		err:            errors.New("boom"),
	}
	m.refreshAllColumns()
	view := m.viewListMulti()
	if !strings.Contains(view, "便携版") || !strings.Contains(view, "✗") {
		t.Fatalf("expected error banner and portable flag")
	}
}

func TestHandleTemplateListKeysNavigation(t *testing.T) {
	tm := newTestTemplateManager(t)
	m := Model{mode: "template_manager", templateMode: "list", currentApp: "claude", templateManager: tm}
	m.refreshTemplates()
	if len(m.templates) < 2 {
		t.Fatalf("expected multiple templates")
	}

	m.templateCursor = 0
	updated, _ := m.handleTemplateListKeys(makeKey("down"))
	m = updated.(Model)
	updated, _ = m.handleTemplateListKeys(makeKey("up"))
	m = updated.(Model)

	updated, _ = m.handleTemplateListKeys(makeKey("enter"))
	m = updated.(Model)
	if m.templateMode != "apply_select_target" || m.selectedTemplate == nil {
		t.Fatalf("expected select target mode")
	}
}

func TestHandleBackupListKeysRestoreError(t *testing.T) {
	manager := newTestManager(t)
	m := Model{
		manager:    manager,
		mode:       "backup_list",
		configPath: manager.GetConfigPath(),
		backupList: []backup.BackupInfo{{Path: filepath.Join(t.TempDir(), "missing.json")}},
	}
	updated, _ := m.handleBackupListKeys(makeKey("enter"))
	m = updated.(Model)
	if m.err == nil {
		t.Fatalf("expected restore error")
	}
}

func TestHandleMcpAppsToggleKeysAddError(t *testing.T) {
	manager := newTestManager(t)
	server := config.McpServer{
		ID:     "bad",
		Name:   "",
		Server: map[string]interface{}{"type": "stdio"},
	}
	m := Model{
		manager:       manager,
		mcpMode:       "apps_toggle",
		selectedMcp:   &server,
		mcpAppsToggle: config.McpApps{Claude: true},
	}
	updated, _ := m.handleMcpAppsToggleKeys(makeKey("enter"))
	m = updated.(Model)
	if m.err == nil {
		t.Fatalf("expected add error")
	}
}

func TestHandleMcpAppsToggleKeysSaveError(t *testing.T) {
	manager := newTestManager(t)
	server := addMcpServer(t, manager, "save-err", config.McpApps{Claude: true})

	configDir := filepath.Dir(manager.GetConfigPath())
	_ = os.RemoveAll(configDir)
	if err := os.WriteFile(configDir, []byte("not a dir"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	m := Model{
		manager:       manager,
		mcpMode:       "apps_toggle",
		selectedMcp:   &server,
		mcpAppsToggle: server.Apps,
	}
	updated, _ := m.handleMcpAppsToggleKeys(makeKey("enter"))
	m = updated.(Model)
	if m.err == nil {
		t.Fatalf("expected save error")
	}
}

func TestHandleMcpDeleteKeysSaveError(t *testing.T) {
	manager := newTestManager(t)
	server := addMcpServer(t, manager, "delete-err", config.McpApps{Claude: true})

	configDir := filepath.Dir(manager.GetConfigPath())
	_ = os.RemoveAll(configDir)
	if err := os.WriteFile(configDir, []byte("not a dir"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	m := Model{manager: manager, mcpMode: "delete", selectedMcp: &server}
	updated, _ := m.handleMcpDeleteKeys(makeKey("y"))
	m = updated.(Model)
	if m.err == nil {
		t.Fatalf("expected save error")
	}
}

func TestHandleMcpListKeysAllBranches(t *testing.T) {
	manager := newTestManager(t)
	_ = addMcpServer(t, manager, "srv1", config.McpApps{Claude: true})
	_ = addMcpServer(t, manager, "srv2", config.McpApps{Codex: true})

	base := Model{manager: manager, mode: "mcp_manager", mcpMode: "list"}
	base.refreshMcpServers()
	if len(base.mcpServers) < 2 {
		t.Fatalf("expected mcp servers")
	}

	cases := []struct {
		name   string
		key    string
		setup  func(*Model)
		verify func(*testing.T, Model)
	}{
		{
			name:  "up_wrap",
			key:   "up",
			setup: func(m *Model) { m.mcpCursor = 0 },
			verify: func(t *testing.T, m Model) {
				if m.mcpCursor != len(m.mcpServers)-1 {
					t.Fatalf("expected wrap to last")
				}
			},
		},
		{
			name:  "down_wrap",
			key:   "down",
			setup: func(m *Model) { m.mcpCursor = len(m.mcpServers) - 1 },
			verify: func(t *testing.T, m Model) {
				if m.mcpCursor != 0 {
					t.Fatalf("expected wrap to first")
				}
			},
		},
		{
			name: "enter_apps_toggle",
			key:  "enter",
			verify: func(t *testing.T, m Model) {
				if m.mcpMode != "apps_toggle" || m.selectedMcp == nil {
					t.Fatalf("expected apps toggle")
				}
			},
		},
		{
			name: "add",
			key:  "a",
			verify: func(t *testing.T, m Model) {
				if m.mcpMode != "add" {
					t.Fatalf("expected add mode")
				}
			},
		},
		{
			name: "edit",
			key:  "e",
			verify: func(t *testing.T, m Model) {
				if m.mcpMode != "edit" || m.selectedMcp == nil {
					t.Fatalf("expected edit mode")
				}
			},
		},
		{
			name: "delete",
			key:  "d",
			verify: func(t *testing.T, m Model) {
				if m.mcpMode != "delete" || m.selectedMcp == nil {
					t.Fatalf("expected delete mode")
				}
			},
		},
		{
			name: "preset",
			key:  "p",
			verify: func(t *testing.T, m Model) {
				if m.mcpMode != "preset" || len(m.mcpPresets) == 0 {
					t.Fatalf("expected presets")
				}
			},
		},
		{
			name: "sync",
			key:  "s",
			verify: func(t *testing.T, m Model) {
				if m.message == "" {
					t.Fatalf("expected sync message")
				}
			},
		},
		{
			name: "refresh",
			key:  "r",
			verify: func(t *testing.T, m Model) {
				if m.message == "" {
					t.Fatalf("expected refresh message")
				}
			},
		},
		{
			name: "escape",
			key:  "esc",
			verify: func(t *testing.T, m Model) {
				if m.mode != "list" {
					t.Fatalf("expected list mode")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := base
			m.mode = "mcp_manager"
			m.mcpMode = "list"
			m.refreshMcpServers()
			if tc.setup != nil {
				tc.setup(&m)
			}
			updated, _ := m.handleMcpListKeys(makeKey(tc.key))
			m = updated.(Model)
			tc.verify(t, m)
		})
	}
}

func TestHandleSourceSelectKeysWrap(t *testing.T) {
	testutil.WithTempHome(t, func(home string) {
		testutil.WithTempCWD(t, func(cwd string) {
			tm := newTestTemplateManager(t)
			globalPath := filepath.Join(home, ".claude", "CLAUDE.md")
			localPath := filepath.Join(cwd, "CLAUDE.local.md")
			if err := os.MkdirAll(filepath.Dir(globalPath), 0755); err != nil {
				t.Fatalf("MkdirAll() error = %v", err)
			}
			if err := os.WriteFile(globalPath, []byte("global"), 0644); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}
			if err := os.WriteFile(localPath, []byte("local"), 0644); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			m := Model{mode: "template_manager", templateMode: "save_select_source", currentApp: "claude", templateManager: tm, sourceSelectCursor: 2}
			updated, _ := m.handleSourceSelectKeys(makeKey("up"))
			m = updated.(Model)
			if m.sourceSelectCursor != 0 {
				t.Fatalf("expected move to existing above")
			}

			updated, _ = m.handleSourceSelectKeys(makeKey("down"))
			m = updated.(Model)
			if m.sourceSelectCursor != 2 {
				t.Fatalf("expected move to existing below")
			}
		})
	})
}

func TestViewBackupListMessage(t *testing.T) {
	m := Model{
		message: "ok",
		backupList: []backup.BackupInfo{
			{Path: filepath.Join(t.TempDir(), "backup.json"), Size: 1024},
		},
	}
	view := m.viewBackupList()
	if !strings.Contains(view, "✓") {
		t.Fatalf("expected success banner")
	}
}

func TestNewTemplateManagerInitError(t *testing.T) {
	testutil.WithTempHome(t, func(home string) {
		blocked := filepath.Join(home, ".cc-switch")
		if err := os.WriteFile(blocked, []byte("not a dir"), 0644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		manager := newTestManager(t)
		m := New(manager)
		if m.err == nil {
			t.Fatalf("expected init error")
		}
	})
}

func TestSubmitFormClaudeErrorsAndSuccess(t *testing.T) {
	cases := []struct {
		name     string
		setup    func(m *Model)
		wantErr  bool
		wantMode string
	}{
		{
			name: "claude_missing_name",
			setup: func(m *Model) {
				m.currentApp = "claude"
				m.mode = "add"
				m.providers = []config.Provider{{ID: "existing"}}
				m.initForm(nil)
				m.inputs[1].SetValue("token")
				m.inputs[2].SetValue("https://api.anthropic.com")
			},
			wantErr: true,
		},
		{
			name: "claude_missing_token",
			setup: func(m *Model) {
				m.currentApp = "claude"
				m.mode = "add"
				m.providers = []config.Provider{{ID: "existing"}}
				m.initForm(nil)
				m.inputs[0].SetValue("Claude")
				m.inputs[2].SetValue("https://api.anthropic.com")
			},
			wantErr: true,
		},
		{
			name: "claude_missing_base_url",
			setup: func(m *Model) {
				m.currentApp = "claude"
				m.mode = "add"
				m.providers = []config.Provider{{ID: "existing"}}
				m.initForm(nil)
				m.inputs[0].SetValue("Claude")
				m.inputs[1].SetValue("token")
			},
			wantErr: true,
		},
		{
			name: "claude_success",
			setup: func(m *Model) {
				m.currentApp = "claude"
				m.mode = "add"
				m.providers = []config.Provider{{ID: "existing"}}
				m.initForm(nil)
				m.inputs[0].SetValue("Claude")
				m.inputs[1].SetValue("token")
				m.inputs[2].SetValue("https://api.anthropic.com")
			},
			wantErr:  false,
			wantMode: "list",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manager := newTestManager(t)
			m := Model{manager: manager}
			tc.setup(&m)
			m.submitForm()
			if tc.wantErr && m.err == nil {
				t.Fatalf("expected error")
			}
			if !tc.wantErr && m.err != nil {
				t.Fatalf("unexpected error: %v", m.err)
			}
			if tc.wantMode != "" && m.mode != tc.wantMode {
				t.Fatalf("expected mode %s", tc.wantMode)
			}
		})
	}
}

func TestSubmitFormGeminiOAuth(t *testing.T) {
	testutil.WithTempHome(t, func(_ string) {
		manager := newTestManager(t)
		m := Model{manager: manager, currentApp: "gemini", mode: "add"}
		m.initForm(nil)
		m.inputs[0].SetValue("google oauth")
		m.submitForm()
		if m.err != nil {
			t.Fatalf("unexpected error: %v", m.err)
		}
		if m.mode != "list" {
			t.Fatalf("expected list mode")
		}
	})
}

func TestSubmitFormGeminiMissingDetails(t *testing.T) {
	manager := newTestManager(t)
	cases := []struct {
		name  string
		setup func(m *Model)
	}{
		{
			name: "gemini_missing_base_url",
			setup: func(m *Model) {
				m.currentApp = "gemini"
				m.mode = "add"
				m.initForm(nil)
				m.inputs[0].SetValue("Gemini")
				m.inputs[1].SetValue("key")
				m.inputs[2].SetValue("")
			},
		},
		{
			name: "gemini_missing_model",
			setup: func(m *Model) {
				m.currentApp = "gemini"
				m.mode = "add"
				m.initForm(nil)
				m.inputs[0].SetValue("Gemini")
				m.inputs[1].SetValue("key")
				m.inputs[2].SetValue("https://api.example.com")
				m.inputs[3].SetValue("")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := Model{manager: manager}
			tc.setup(&m)
			m.submitForm()
			if m.err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func TestSubmitFormEditModeMultiView(t *testing.T) {
	manager := newTestManager(t)
	provider := addProvider(t, manager, "claude", "edit1", "Claude Edit")
	m := Model{manager: manager, currentApp: "claude", mode: "edit", editName: provider.Name, viewMode: "multi"}
	m.initForm(&provider)
	m.inputs[0].SetValue("Claude Edit")
	m.inputs[1].SetValue("token")
	m.inputs[2].SetValue("https://api.anthropic.com")
	m.submitForm()
	if m.err != nil || m.mode != "list" {
		t.Fatalf("expected edit success")
	}
	if len(m.columnProviders[0]) == 0 {
		t.Fatalf("expected columns refreshed")
	}
}

func TestSubmitFormCodexMissingBasics(t *testing.T) {
	manager := newTestManager(t)
	cases := []struct {
		name  string
		setup func(m *Model)
	}{
		{
			name: "codex_missing_name",
			setup: func(m *Model) {
				m.currentApp = "codex"
				m.mode = "add"
				m.providers = []config.Provider{{ID: "existing"}}
				m.initForm(nil)
				m.inputs[1].SetValue("token")
				m.inputs[2].SetValue("https://api.example.com")
				m.inputs[4].SetValue("gpt-5")
				m.inputs[5].SetValue("high")
			},
		},
		{
			name: "codex_missing_token",
			setup: func(m *Model) {
				m.currentApp = "codex"
				m.mode = "add"
				m.providers = []config.Provider{{ID: "existing"}}
				m.initForm(nil)
				m.inputs[0].SetValue("Codex")
				m.inputs[2].SetValue("https://api.example.com")
				m.inputs[4].SetValue("gpt-5")
				m.inputs[5].SetValue("high")
			},
		},
		{
			name: "codex_missing_base_url",
			setup: func(m *Model) {
				m.currentApp = "codex"
				m.mode = "add"
				m.providers = []config.Provider{{ID: "existing"}}
				m.initForm(nil)
				m.inputs[0].SetValue("Codex")
				m.inputs[1].SetValue("token")
				m.inputs[4].SetValue("gpt-5")
				m.inputs[5].SetValue("high")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := Model{manager: manager}
			tc.setup(&m)
			m.submitForm()
			if m.err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func TestApplyTokenVisibilityToggle(t *testing.T) {
	m := Model{currentApp: "claude"}
	m.initForm(nil)

	m.apiTokenVisible = true
	m.applyTokenVisibility()
	if m.inputs[1].EchoMode != textinput.EchoNormal {
		t.Fatalf("expected visible token")
	}

	m.apiTokenVisible = false
	m.applyTokenVisibility()
	if m.inputs[1].EchoMode != textinput.EchoPassword {
		t.Fatalf("expected hidden token")
	}
}

func TestFindSelectorOptionIndex(t *testing.T) {
	if findSelectorOptionIndex(claudePrimaryModelSelectorOptions, "opus") == -1 {
		t.Fatalf("expected to find option")
	}
	if findSelectorOptionIndex(claudePrimaryModelSelectorOptions, "missing") != -1 {
		t.Fatalf("expected missing option")
	}
}

func TestRefreshTemplatesNoManager(t *testing.T) {
	m := Model{}
	m.refreshTemplates()
}

func TestSyncColumnCursorsGemini(t *testing.T) {
	manager := newTestManager(t)
	addProvider(t, manager, "gemini", "g1", "Gemini One")
	m := Model{manager: manager, currentApp: "gemini", cursor: 0}
	m.syncColumnCursors()
	if m.columnCursor != 2 {
		t.Fatalf("expected gemini column cursor")
	}
}

func TestInitFormCopyFromProvider(t *testing.T) {
	manager := newTestManager(t)
	provider := addProvider(t, manager, "claude", "copy1", "Claude Copy")
	m := Model{currentApp: "claude", copyFromProvider: &provider}
	m.initForm(nil)
	if m.copyFromProvider != nil {
		t.Fatalf("expected copyFromProvider cleared")
	}
	if m.inputs[1].Value() == "" {
		t.Fatalf("expected token copied")
	}
}

func TestHandleSourceSelectKeysUnsupportedCategory(t *testing.T) {
	m := Model{mode: "template_manager", templateMode: "save_select_source", currentApp: "gemini", templateManager: newTestTemplateManager(t)}
	updated, _ := m.handleSourceSelectKeys(makeKey("enter"))
	m = updated.(Model)
	if m.err == nil {
		t.Fatalf("expected error for unsupported category")
	}
}

func TestViewSourceSelectUnsupportedCategory(t *testing.T) {
	m := Model{currentApp: "gemini", templateManager: newTestTemplateManager(t)}
	view := m.viewSourceSelect()
	if !strings.Contains(view, "✗ 获取源路径失败") {
		t.Fatalf("expected error message")
	}
}

func TestViewSourceSelectWithTargets(t *testing.T) {
	testutil.WithTempHome(t, func(home string) {
		testutil.WithTempCWD(t, func(_ string) {
			path := filepath.Join(home, ".claude", "CLAUDE.md")
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatalf("MkdirAll() error = %v", err)
			}
			if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			m := Model{currentApp: "claude", templateManager: newTestTemplateManager(t), sourceSelectCursor: 0}
			view := m.viewSourceSelect()
			if !strings.Contains(view, "选择源文件") {
				t.Fatalf("expected source select view")
			}
		})
	})
}

func TestViewSaveNameInputMessageAndError(t *testing.T) {
	m := Model{
		templateManager:    newTestTemplateManager(t),
		selectedSourcePath: "/tmp/source.md",
		message:            "ok",
		err:                errors.New("boom"),
	}
	m.saveNameInput = textinput.New()
	view := m.viewSaveNameInput()
	if !strings.Contains(view, "ok") || !strings.Contains(view, "✗") {
		t.Fatalf("expected message and error")
	}
}

func TestApplyTokenVisibilityEarlyReturn(t *testing.T) {
	m := Model{}
	m.applyTokenVisibility()
}

func TestLoadLiveConfigForFormNonClaude(t *testing.T) {
	m := Model{currentApp: "codex"}
	_, _, _, _, _, _, loaded := m.loadLiveConfigForForm()
	if loaded {
		t.Fatalf("expected not loaded")
	}
}

func TestLoadLiveConfigForFormInvalidAndEmpty(t *testing.T) {
	testutil.WithTempHome(t, func(home string) {
		settingsPath := filepath.Join(home, ".claude", "settings.json")
		if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		if err := os.WriteFile(settingsPath, []byte("{bad"), 0644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		m := Model{currentApp: "claude"}
		_, _, _, _, _, _, loaded := m.loadLiveConfigForForm()
		if loaded {
			t.Fatalf("expected not loaded")
		}

		emptySettings := config.ClaudeSettings{}
		data, err := json.Marshal(emptySettings)
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}
		if err := os.WriteFile(settingsPath, data, 0644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		_, _, _, _, _, _, loaded = m.loadLiveConfigForForm()
		if loaded {
			t.Fatalf("expected empty settings not loaded")
		}
	})
}

func TestHandleMcpAppsToggleKeysSyncError(t *testing.T) {
	manager := newTestManager(t)
	server := addMcpServer(t, manager, "sync-err", config.McpApps{Claude: true})

	claudeDir := filepath.Join(filepath.Dir(manager.GetConfigPath()), ".claude")
	if err := os.WriteFile(claudeDir, []byte("not a dir"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	m := Model{
		manager:       manager,
		mcpMode:       "apps_toggle",
		selectedMcp:   &server,
		mcpAppsToggle: server.Apps,
	}
	updated, _ := m.handleMcpAppsToggleKeys(makeKey("enter"))
	m = updated.(Model)
	if m.err == nil {
		t.Fatalf("expected sync error")
	}
}

func TestUpdateMcpInputsEditModeSkipID(t *testing.T) {
	manager := newTestManager(t)
	server := addMcpServer(t, manager, "edit", config.McpApps{Claude: true})

	m := Model{manager: manager, mcpMode: "edit"}
	m.initMcpForm(&server)
	m.mcpFocusIndex = 1

	updated, _ := m.updateMcpInputs(makeKey("shift+tab"))
	m = updated.(Model)
	if m.mcpFocusIndex == 0 {
		t.Fatalf("expected skip id field")
	}

	updated, _ = m.updateMcpInputs(makeKey("tab"))
	m = updated.(Model)
	if m.mcpFocusIndex == 0 {
		t.Fatalf("expected skip id field")
	}
}

func TestHandleMcpDeleteKeysSyncErrors(t *testing.T) {
	manager := newTestManager(t)
	server := addMcpServer(t, manager, "sync-delete", config.McpApps{Claude: true, Codex: true, Gemini: true})

	claudePath, err := manager.GetClaudeSettingsPathWithDir()
	if err != nil {
		t.Fatalf("GetClaudeSettingsPathWithDir() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(claudePath), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(claudePath, []byte("{bad"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	codexPath, err := manager.GetCodexConfigPathWithDir()
	if err != nil {
		t.Fatalf("GetCodexConfigPathWithDir() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(codexPath), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(codexPath, []byte("bad"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	geminiPath, err := manager.GetGeminiSettingsPathWithDir()
	if err != nil {
		t.Fatalf("GetGeminiSettingsPathWithDir() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(geminiPath), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(geminiPath, []byte("{bad"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	m := Model{manager: manager, mcpMode: "delete", selectedMcp: &server}
	updated, _ := m.handleMcpDeleteKeys(makeKey("y"))
	m = updated.(Model)
	if m.message == "" || m.err != nil {
		t.Fatalf("expected warning message with nil error")
	}
}

func TestHandleMcpDeleteKeysSuccess(t *testing.T) {
	manager := newTestManager(t)
	server := addMcpServer(t, manager, "delete-ok", config.McpApps{Claude: true})
	m := Model{manager: manager, mcpMode: "delete", selectedMcp: &server}
	updated, _ := m.handleMcpDeleteKeys(makeKey("y"))
	m = updated.(Model)
	if m.err != nil || !strings.Contains(m.message, "✓") {
		t.Fatalf("expected delete success message")
	}
	if m.selectedMcp != nil || m.mcpMode != "list" {
		t.Fatalf("expected reset selection")
	}
}

func TestHandleMultiColumnKeysDownloadWithRelease(t *testing.T) {
	manager := newTestManager(t)
	addProvider(t, manager, "claude", "c1", "Claude One")
	m := Model{
		manager:       manager,
		mode:          "list",
		viewMode:      "multi",
		currentApp:    "claude",
		configPath:    manager.GetConfigPath(),
		latestRelease: &version.ReleaseInfo{TagName: "v0"},
	}
	m.refreshAllColumns()
	_, _ = m.handleMultiColumnKeys(makeKey("U"))
}

func TestPortableModeHelpers(t *testing.T) {
	m := Model{}
	if err := m.enablePortableMode(); err != nil {
		t.Fatalf("enablePortableMode() error = %v", err)
	}
	if err := m.disablePortableMode(); err != nil {
		t.Fatalf("disablePortableMode() error = %v", err)
	}
}

func TestHandleMultiColumnKeysPortableDisable(t *testing.T) {
	manager := newTestManager(t)
	m := Model{manager: manager, mode: "list", viewMode: "multi", isPortableMode: true}
	m.refreshAllColumns()
	updated, _ := m.handleMultiColumnKeys(makeKey("p"))
	m = updated.(Model)
	if m.isPortableMode {
		t.Fatalf("expected portable mode disabled")
	}
}

func TestHandleMultiColumnKeysQuit(t *testing.T) {
	m := Model{viewMode: "multi"}
	_, cmd := m.handleMultiColumnKeys(makeKey("q"))
	if cmd == nil {
		t.Fatalf("expected quit cmd")
	}
}

func TestHandleMcpAppsToggleKeysWrap(t *testing.T) {
	manager := newTestManager(t)
	server := addMcpServer(t, manager, "wrap", config.McpApps{Claude: true})
	m := Model{manager: manager, mcpMode: "apps_toggle", selectedMcp: &server}
	m.mcpAppsCursor = 0
	updated, _ := m.handleMcpAppsToggleKeys(makeKey("up"))
	m = updated.(Model)
	if m.mcpAppsCursor != 2 {
		t.Fatalf("expected wrap to bottom")
	}
	updated, _ = m.handleMcpAppsToggleKeys(makeKey("down"))
	m = updated.(Model)
	if m.mcpAppsCursor != 0 {
		t.Fatalf("expected wrap to top")
	}
	updated, _ = m.handleMcpAppsToggleKeys(makeKey("q"))
	m = updated.(Model)
	if m.mcpMode != "list" {
		t.Fatalf("expected exit apps toggle")
	}
}

func TestSaveMcpFormErrors(t *testing.T) {
	manager := newTestManager(t)

	m := Model{manager: manager, mcpMode: "add"}
	m.initMcpForm(nil)
	m.mcpConnType = "stdio"
	m.mcpInputs[0].SetValue("id")
	m.mcpInputs[1].SetValue("Name")
	m.mcpInputs[2].SetValue("cmd")
	m.mcpInputs[3].SetValue("\"")
	if err := m.saveMcpForm(); err == nil {
		t.Fatalf("expected args parse error")
	}

	m = Model{manager: manager, mcpMode: "add"}
	m.initMcpForm(nil)
	m.mcpConnType = "http"
	m.mcpInputs[0].SetValue("id2")
	m.mcpInputs[1].SetValue("Name")
	m.mcpInputs[4].SetValue("not a url")
	if err := m.saveMcpForm(); err == nil {
		t.Fatalf("expected url validation error")
	}

	m = Model{manager: manager, mcpMode: "edit"}
	m.initMcpForm(nil)
	m.mcpConnType = "stdio"
	m.mcpInputs[0].SetValue("missing")
	m.mcpInputs[1].SetValue("Name")
	m.mcpInputs[2].SetValue("cmd")
	if err := m.saveMcpForm(); err == nil {
		t.Fatalf("expected update error")
	}
}
