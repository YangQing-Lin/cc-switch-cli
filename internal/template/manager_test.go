package template

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestTemplateManager(t *testing.T) (*TemplateManager, string) {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "claude_templates.json")
	tm, err := NewTemplateManager(configPath)
	if err != nil {
		t.Fatalf("NewTemplateManager() error = %v", err)
	}
	return tm, configPath
}

func TestTemplateManagerCRUDAndPersistence(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T, tm *TemplateManager, configPath string)
	}{
		{
			name: "create_read_delete_and_defaults",
			run: func(t *testing.T, tm *TemplateManager, configPath string) {
				if _, err := os.Stat(configPath); err != nil {
					t.Fatalf("expected config file to exist: %v", err)
				}

				id, err := tm.AddTemplate("My Template", CategoryClaudeMd, "hello world")
				if err != nil {
					t.Fatalf("AddTemplate() error = %v", err)
				}
				if !strings.HasPrefix(id, "user_") {
					t.Fatalf("expected user_ prefix, got %s", id)
				}

				got, err := tm.GetTemplate(id)
				if err != nil {
					t.Fatalf("GetTemplate() error = %v", err)
				}
				if got.Content != "hello world" {
					t.Fatalf("unexpected content: %s", got.Content)
				}

				list := tm.ListTemplates(CategoryClaudeMd)
				found := false
				for _, tplt := range list {
					if tplt.ID == id && tplt.Name == "My Template" {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("expected newly added template (%s / %q) to appear in ListTemplates()", id, "My Template")
				}

				if err := tm.DeleteTemplate(id); err != nil {
					t.Fatalf("DeleteTemplate() error = %v", err)
				}
				if _, err := tm.GetTemplate(id); err == nil {
					t.Fatalf("expected error after delete")
				}

				builtinID := ""
				for _, tplt := range list {
					if tplt.IsBuiltin {
						builtinID = tplt.ID
						break
					}
				}
				if builtinID == "" {
					t.Fatalf("expected builtin template")
				}
				if err := tm.DeleteTemplate(builtinID); err == nil {
					t.Fatalf("expected error deleting builtin template")
				}
			},
		},
		{
			name: "update_by_reload",
			run: func(t *testing.T, tm *TemplateManager, configPath string) {
				id, err := tm.AddTemplate("Reloaded", CategoryClaudeMd, "before")
				if err != nil {
					t.Fatalf("AddTemplate() error = %v", err)
				}

				data, err := os.ReadFile(configPath)
				if err != nil {
					t.Fatalf("ReadFile() error = %v", err)
				}
				var cfg TemplateConfig
				if err := json.Unmarshal(data, &cfg); err != nil {
					t.Fatalf("Unmarshal() error = %v", err)
				}
				tmpl := cfg.Templates[id]
				tmpl.Content = "after"
				cfg.Templates[id] = tmpl
				data, err = json.MarshalIndent(cfg, "", "  ")
				if err != nil {
					t.Fatalf("Marshal() error = %v", err)
				}
				if err := os.WriteFile(configPath, data, 0644); err != nil {
					t.Fatalf("WriteFile() error = %v", err)
				}

				reloaded, err := NewTemplateManager(configPath)
				if err != nil {
					t.Fatalf("NewTemplateManager() error = %v", err)
				}
				updated, err := reloaded.GetTemplate(id)
				if err != nil {
					t.Fatalf("GetTemplate() error = %v", err)
				}
				if updated.Content != "after" {
					t.Fatalf("expected updated content, got %s", updated.Content)
				}
			},
		},
		{
			name: "generate_default_name",
			run: func(t *testing.T, tm *TemplateManager, _ string) {
				tm.Templates["u1"] = Template{ID: "u1", Name: "用户配置1", Category: CategoryClaudeMd}
				tm.addToCategory(CategoryClaudeMd, "u1")
				tm.Templates["u2"] = Template{ID: "u2", Name: "用户配置2", Category: CategoryClaudeMd}
				tm.addToCategory(CategoryClaudeMd, "u2")

				name := tm.GenerateDefaultTemplateName()
				if name != "用户配置3" {
					t.Fatalf("unexpected default name: %s", name)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tm, configPath := newTestTemplateManager(t)
			tc.run(t, tm, configPath)
		})
	}
}

func TestTemplateManagerFileOperations(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T, tm *TemplateManager, dir string)
	}{
		{
			name: "apply_template",
			run: func(t *testing.T, tm *TemplateManager, dir string) {
				id, err := tm.AddTemplate("Apply", CategoryClaudeMd, "apply content")
				if err != nil {
					t.Fatalf("AddTemplate() error = %v", err)
				}
				target := filepath.Join(dir, "nested", "CLAUDE.md")
				if err := tm.ApplyTemplate(id, target); err != nil {
					t.Fatalf("ApplyTemplate() error = %v", err)
				}
				data, err := os.ReadFile(target)
				if err != nil {
					t.Fatalf("ReadFile() error = %v", err)
				}
				if string(data) != "apply content" {
					t.Fatalf("unexpected content: %s", string(data))
				}
			},
		},
		{
			name: "save_as_template",
			run: func(t *testing.T, tm *TemplateManager, dir string) {
				source := filepath.Join(dir, "source.md")
				if err := os.WriteFile(source, []byte("saved"), 0644); err != nil {
					t.Fatalf("WriteFile() error = %v", err)
				}
				id, err := tm.SaveAsTemplate(source, "Saved", CategoryClaudeMd)
				if err != nil {
					t.Fatalf("SaveAsTemplate() error = %v", err)
				}
				tmpl, err := tm.GetTemplate(id)
				if err != nil {
					t.Fatalf("GetTemplate() error = %v", err)
				}
				if tmpl.Content != "saved" {
					t.Fatalf("unexpected saved content: %s", tmpl.Content)
				}
			},
		},
		{
			name: "get_diff",
			run: func(t *testing.T, tm *TemplateManager, dir string) {
				id, err := tm.AddTemplate("Diff", CategoryClaudeMd, "new line")
				if err != nil {
					t.Fatalf("AddTemplate() error = %v", err)
				}
				target := filepath.Join(dir, "target.md")
				if err := os.WriteFile(target, []byte("old line"), 0644); err != nil {
					t.Fatalf("WriteFile() error = %v", err)
				}

				diff, err := tm.GetDiff(id, target)
				if err != nil {
					t.Fatalf("GetDiff() error = %v", err)
				}
				if !strings.Contains(diff, "@@") || !strings.Contains(diff, "-old line") || !strings.Contains(diff, "+new line") {
					t.Fatalf("unexpected diff output: %s", diff)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tm, _ := newTestTemplateManager(t)
			tc.run(t, tm, t.TempDir())
		})
	}
}

func TestTemplateManagerInvalidConfig(t *testing.T) {
	cases := []struct {
		name string
		data string
	}{
		{name: "invalid_json", data: "{invalid"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "claude_templates.json")
			if err := os.WriteFile(configPath, []byte(tc.data), 0644); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}
			if _, err := NewTemplateManager(configPath); err == nil {
				t.Fatalf("expected error for invalid config")
			}
		})
	}
}

func TestTemplateManagerErrorPaths(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "add_template_save_error",
			run: func(t *testing.T) {
				dir := t.TempDir()
				blocker := filepath.Join(dir, "notadir")
				if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
					t.Fatalf("WriteFile() error = %v", err)
				}
				tm := &TemplateManager{
					Templates:  make(map[string]Template),
					Categories: make(map[string][]string),
					configPath: filepath.Join(blocker, "templates.json"),
				}
				if _, err := tm.AddTemplate("Bad", CategoryClaudeMd, "content"); err == nil {
					t.Fatalf("expected AddTemplate error")
				}
			},
		},
		{
			name: "apply_template_mkdir_error",
			run: func(t *testing.T) {
				dir := t.TempDir()
				blocker := filepath.Join(dir, "notadir")
				if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
					t.Fatalf("WriteFile() error = %v", err)
				}
				tm := &TemplateManager{
					Templates: map[string]Template{
						"t1": {ID: "t1", Name: "T1", Category: CategoryClaudeMd, Content: "x"},
					},
					Categories: make(map[string][]string),
					configPath: filepath.Join(dir, "cfg.json"),
				}
				tm.addToCategory(CategoryClaudeMd, "t1")
				target := filepath.Join(blocker, "CLAUDE.md")
				if err := tm.ApplyTemplate("t1", target); err == nil {
					t.Fatalf("expected ApplyTemplate error")
				}
			},
		},
		{
			name: "apply_template_rename_error",
			run: func(t *testing.T) {
				dir := t.TempDir()
				targetDir := filepath.Join(dir, "target")
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					t.Fatalf("MkdirAll() error = %v", err)
				}
				tm := &TemplateManager{
					Templates: map[string]Template{
						"t1": {ID: "t1", Name: "T1", Category: CategoryClaudeMd, Content: "x"},
					},
					Categories: make(map[string][]string),
					configPath: filepath.Join(dir, "cfg.json"),
				}
				tm.addToCategory(CategoryClaudeMd, "t1")
				if err := tm.ApplyTemplate("t1", targetDir); err == nil {
					t.Fatalf("expected ApplyTemplate rename error")
				}
			},
		},
		{
			name: "save_as_template_missing_source",
			run: func(t *testing.T) {
				tm, _ := newTestTemplateManager(t)
				if _, err := tm.SaveAsTemplate(filepath.Join(t.TempDir(), "missing.md"), "x", CategoryClaudeMd); err == nil {
					t.Fatalf("expected SaveAsTemplate error")
				}
			},
		},
		{
			name: "save_user_templates_rename_error",
			run: func(t *testing.T) {
				dir := t.TempDir()
				configDir := filepath.Join(dir, "configdir")
				if err := os.MkdirAll(configDir, 0755); err != nil {
					t.Fatalf("MkdirAll() error = %v", err)
				}
				tm := &TemplateManager{
					Templates: map[string]Template{
						"u1": {ID: "u1", Name: "User", Category: CategoryClaudeMd},
					},
					Categories: map[string][]string{CategoryClaudeMd: {"u1"}},
					configPath: configDir,
				}
				if err := tm.saveUserTemplates(); err == nil {
					t.Fatalf("expected saveUserTemplates rename error")
				}
			},
		},
		{
			name: "list_templates_missing_category",
			run: func(t *testing.T) {
				tm, _ := newTestTemplateManager(t)
				if got := tm.ListTemplates("missing"); len(got) != 0 {
					t.Fatalf("expected empty list, got %d", len(got))
				}
			},
		},
		{
			name: "get_template_missing",
			run: func(t *testing.T) {
				tm, _ := newTestTemplateManager(t)
				if _, err := tm.GetTemplate("missing"); err == nil {
					t.Fatalf("expected GetTemplate error")
				}
			},
		},
		{
			name: "delete_template_missing",
			run: func(t *testing.T) {
				tm, _ := newTestTemplateManager(t)
				if err := tm.DeleteTemplate("missing"); err == nil {
					t.Fatalf("expected DeleteTemplate error")
				}
			},
		},
		{
			name: "remove_from_missing_category",
			run: func(t *testing.T) {
				tm, _ := newTestTemplateManager(t)
				tm.removeFromCategory("missing", "id")
			},
		},
		{
			name: "get_diff_missing_template",
			run: func(t *testing.T) {
				tm, _ := newTestTemplateManager(t)
				if _, err := tm.GetDiff("missing", filepath.Join(t.TempDir(), "file")); err == nil {
					t.Fatalf("expected GetDiff error")
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
