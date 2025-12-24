package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/template"
)

func TestDisplayWidth(t *testing.T) {
	cases := []struct {
		name string
		text string
		want int
	}{
		{name: "ascii", text: "abc", want: 3},
		{name: "chinese", text: "中文", want: 4},
		{name: "mixed", text: "a中", want: 3},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := displayWidth(tc.text); got != tc.want {
				t.Fatalf("displayWidth(%q) = %d, want %d", tc.text, got, tc.want)
			}
		})
	}
}

func TestHelperMappings(t *testing.T) {
	cases := []struct {
		name       string
		col        int
		wantApp    string
		currentApp string
		wantCat    string
	}{
		{name: "claude", col: 0, wantApp: "claude", currentApp: "claude", wantCat: template.CategoryClaudeMd},
		{name: "codex", col: 1, wantApp: "codex", currentApp: "codex", wantCat: template.CategoryCodexMd},
		{name: "gemini", col: 2, wantApp: "gemini", currentApp: "gemini", wantCat: ""},
		{name: "default", col: 9, wantApp: "claude", currentApp: "unknown", wantCat: template.CategoryClaudeMd},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := Model{currentApp: tc.currentApp}
			if got := m.columnToAppName(tc.col); got != tc.wantApp {
				t.Fatalf("columnToAppName(%d) = %s, want %s", tc.col, got, tc.wantApp)
			}
			if got := m.currentTemplateCategory(); got != tc.wantCat {
				t.Fatalf("currentTemplateCategory() = %s, want %s", got, tc.wantCat)
			}
		})
	}
}

func TestTemplateCategoryDisplay(t *testing.T) {
	cases := []struct {
		name     string
		category string
		want     string
	}{
		{name: "codex", category: template.CategoryCodexMd, want: "Codex 指南 (CODEX.md)"},
		{name: "claude", category: template.CategoryClaudeMd, want: "Claude 指南 (CLAUDE.md)"},
		{name: "custom", category: "custom", want: "custom"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := Model{}
			if got := m.templateCategoryDisplay(tc.category); got != tc.want {
				t.Fatalf("templateCategoryDisplay(%s) = %s, want %s", tc.category, got, tc.want)
			}
		})
	}
}

func TestRefreshHelpers(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "refresh_providers_and_view_mode",
			run: func(t *testing.T) {
				manager := newTestManager(t)
				addProvider(t, manager, "claude", "p1", "Claude One")
				m := Model{manager: manager, currentApp: "claude"}
				m.refreshProviders()
				if len(m.providers) != 1 {
					t.Fatalf("expected providers to refresh")
				}

				m.viewMode = "multi"
				m.saveViewModePreference()
				if got := manager.GetViewMode(); got != "multi" {
					t.Fatalf("expected view mode multi, got %s", got)
				}
			},
		},
		{
			name: "refresh_all_columns_and_sync",
			run: func(t *testing.T) {
				manager := newTestManager(t)
				addProvider(t, manager, "claude", "c1", "Claude One")
				addProvider(t, manager, "codex", "x1", "Codex One")
				addProvider(t, manager, "gemini", "g1", "Gemini One")
				m := Model{manager: manager}
				m.columnCursors = [3]int{5, 5, 5}
				m.refreshAllColumns()
				if len(m.columnProviders[0]) == 0 || len(m.columnProviders[1]) == 0 || len(m.columnProviders[2]) == 0 {
					t.Fatalf("expected column providers to refresh")
				}
				if m.columnCursors[0] != 0 || m.columnCursors[1] != 0 || m.columnCursors[2] != 0 {
					t.Fatalf("expected column cursors to clamp")
				}

				m.currentApp = "codex"
				m.cursor = 1
				m.syncColumnCursors()
				if m.columnCursor != 1 || m.columnCursors[1] != 1 || m.desiredRow != 1 {
					t.Fatalf("expected cursors to sync")
				}
			},
		},
		{
			name: "refresh_templates_and_mcp",
			run: func(t *testing.T) {
				manager := newTestManager(t)
				m := Model{manager: manager, currentApp: "claude", templateManager: newTestTemplateManager(t)}
				if _, err := m.templateManager.AddTemplate("t", template.CategoryClaudeMd, "content"); err != nil {
					t.Fatalf("AddTemplate() error = %v", err)
				}
				m.refreshTemplates()
				if len(m.templates) == 0 {
					t.Fatalf("expected templates to refresh")
				}

				addMcpServer(t, manager, "srv", config.McpApps{Claude: true})
				m.refreshMcpServers()
				if len(m.mcpServers) == 0 {
					t.Fatalf("expected mcp servers to refresh")
				}
			},
		},
		{
			name: "sync_mod_time",
			run: func(t *testing.T) {
				manager := newTestManager(t)
				addProvider(t, manager, "claude", "p1", "Claude One")
				m := Model{manager: manager, configPath: manager.GetConfigPath()}
				m.syncModTime()
				if m.lastModTime.IsZero() {
					t.Fatalf("expected lastModTime to update")
				}
				if m.lastModTime.After(time.Now().Add(time.Minute)) {
					t.Fatalf("unexpected mod time")
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

func TestGetCategoryBadge(t *testing.T) {
	cases := []struct {
		name     string
		category string
		want     string
	}{
		{name: "official", category: "official", want: "官方"},
		{name: "cn_official", category: "cn_official", want: "国产官方"},
		{name: "aggregator", category: "aggregator", want: "聚合"},
		{name: "third_party", category: "third_party", want: "第三方"},
		{name: "custom", category: "custom", want: "自定义"},
		{name: "unknown", category: "unknown", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := GetCategoryBadge(tc.category)
			if tc.want == "" {
				if got != "" {
					t.Fatalf("expected empty badge, got %s", got)
				}
				return
			}
			if !strings.Contains(got, tc.want) {
				t.Fatalf("expected badge to contain %s, got %s", tc.want, got)
			}
		})
	}
}
