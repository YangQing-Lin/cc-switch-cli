package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/template"
	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
)

func TestTemplateCommands(t *testing.T) {
	resetGlobals()
	home := withTempHome(t)
	cwd := withTempCWD(t)

	sourcePath := filepath.Join(cwd, "CLAUDE.md")
	if err := os.WriteFile(sourcePath, []byte("source"), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	saveFrom = "project_root"
	saveName = "saved-template"
	testutil.CaptureOutput(t, func() {
		runSaveTemplate()
	})
	configPath := filepath.Join(home, ".cc-switch", "claude_templates.json")
	tm, err := template.NewTemplateManager(configPath)
	if err != nil {
		t.Fatalf("new template manager: %v", err)
	}
	foundSave := false
	for _, tpl := range tm.ListTemplates(template.CategoryClaudeMd) {
		if tpl.Name == "saved-template" {
			foundSave = true
			break
		}
	}
	if !foundSave {
		t.Fatalf("expected saved template to exist")
	}

	addFile = sourcePath
	addName = "added-template"
	addCategory = template.CategoryClaudeMd
	testutil.CaptureOutput(t, func() {
		runAddTemplate()
	})
	tm, err = template.NewTemplateManager(configPath)
	if err != nil {
		t.Fatalf("new template manager: %v", err)
	}
	foundAdd := false
	for _, tpl := range tm.ListTemplates(template.CategoryClaudeMd) {
		if tpl.Name == "added-template" {
			foundAdd = true
			break
		}
	}
	if !foundAdd {
		t.Fatalf("expected added template to exist")
	}
	applyID, err := tm.AddTemplate("apply-template", template.CategoryClaudeMd, "applied content")
	if err != nil {
		t.Fatalf("add template: %v", err)
	}

	applyTarget = "project_root"
	skipDiff = true
	testutil.CaptureOutput(t, func() {
		runApplyTemplate(applyID)
	})
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read applied file: %v", err)
	}
	if string(data) != "applied content" {
		t.Fatalf("unexpected applied content: %s", string(data))
	}

	deleteForce = true
	testutil.CaptureOutput(t, func() {
		runDeleteTemplate(applyID)
	})

	tm, err = template.NewTemplateManager(configPath)
	if err != nil {
		t.Fatalf("new template manager: %v", err)
	}
	if _, err := tm.GetTemplate(applyID); err == nil {
		t.Fatalf("expected template to be deleted")
	}

	listCategory = template.CategoryClaudeMd
	stdout, _ := testutil.CaptureOutput(t, func() {
		runListTemplates()
	})
	if !strings.Contains(stdout, "ID:") {
		t.Fatalf("expected list output to include template IDs, got: %s", stdout)
	}
}

func TestTemplateListEmptyCategory(t *testing.T) {
	resetGlobals()
	withTempHome(t)
	listCategory = "missing-category"
	testutil.CaptureOutput(t, func() {
		runListTemplates()
	})
}

func TestTemplateInteractiveHelpers(t *testing.T) {
	resetGlobals()
	home := withTempHome(t)
	cwd := withTempCWD(t)

	sourcePath := filepath.Join(cwd, "CLAUDE.md")
	if err := os.WriteFile(sourcePath, []byte("source"), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	withStdin(t, "2\n", func() {
		target, err := selectSourcePath()
		if err != nil {
			t.Fatalf("selectSourcePath: %v", err)
		}
		if target.ID != "project_root" {
			t.Fatalf("unexpected target: %s", target.ID)
		}
	})

	withStdin(t, "\n", func() {
		name := promptTemplateName("default-name")
		if name != "default-name" {
			t.Fatalf("expected default name, got %s", name)
		}
	})
	withStdin(t, "custom-name\n", func() {
		name := promptTemplateName("default-name")
		if name != "custom-name" {
			t.Fatalf("expected custom name, got %s", name)
		}
	})

	withStdin(t, "1\n", func() {
		target, err := selectTarget()
		if err != nil {
			t.Fatalf("selectTarget: %v", err)
		}
		if target.ID != "global" {
			t.Fatalf("unexpected target: %s", target.ID)
		}
	})

	withStdin(t, "y\n", func() {
		if !confirmApply() {
			t.Fatalf("expected confirmApply true")
		}
	})
	withStdin(t, "n\n", func() {
		if confirmApply() {
			t.Fatalf("expected confirmApply false")
		}
	})

	withStdin(t, "y\n", func() {
		if !confirmDelete() {
			t.Fatalf("expected confirmDelete true")
		}
	})
	withStdin(t, "n\n", func() {
		if confirmDelete() {
			t.Fatalf("expected confirmDelete false")
		}
	})

	configPath := filepath.Join(home, ".cc-switch", "claude_templates.json")
	tm, err := template.NewTemplateManager(configPath)
	if err != nil {
		t.Fatalf("new template manager: %v", err)
	}
	beforeCount := len(tm.ListTemplates(template.CategoryClaudeMd))

	saveFrom = ""
	saveName = ""
	withStdin(t, "2\n\n", func() {
		testutil.CaptureOutput(t, func() {
			runSaveTemplate()
		})
	})

	tm, err = template.NewTemplateManager(configPath)
	if err != nil {
		t.Fatalf("new template manager: %v", err)
	}
	afterCount := len(tm.ListTemplates(template.CategoryClaudeMd))
	if afterCount <= beforeCount {
		t.Fatalf("expected saved template to increase count")
	}
}

func TestTemplateApplyCancelledAndDeleteCancelled(t *testing.T) {
	resetGlobals()
	home := withTempHome(t)
	cwd := withTempCWD(t)

	targetPath := filepath.Join(cwd, "CLAUDE.md")
	if err := os.WriteFile(targetPath, []byte("original"), 0644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	configPath := filepath.Join(home, ".cc-switch", "claude_templates.json")
	tm, err := template.NewTemplateManager(configPath)
	if err != nil {
		t.Fatalf("new template manager: %v", err)
	}
	templateID, err := tm.AddTemplate("cancel-template", template.CategoryClaudeMd, "new-content")
	if err != nil {
		t.Fatalf("add template: %v", err)
	}

	applyTarget = ""
	skipDiff = false
	withStdin(t, "2\nn\n", func() {
		testutil.CaptureOutput(t, func() {
			runApplyTemplate(templateID)
		})
	})
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(data) != "original" {
		t.Fatalf("expected unchanged content, got: %s", string(data))
	}

	deleteForce = false
	withStdin(t, "n\n", func() {
		testutil.CaptureOutput(t, func() {
			runDeleteTemplate(templateID)
		})
	})
	tm, err = template.NewTemplateManager(configPath)
	if err != nil {
		t.Fatalf("new template manager: %v", err)
	}
	if _, err := tm.GetTemplate(templateID); err != nil {
		t.Fatalf("expected template to remain after cancel")
	}
}

func TestTemplateApplyConfirmedAndDeleteConfirmed(t *testing.T) {
	resetGlobals()
	home := withTempHome(t)
	cwd := withTempCWD(t)

	targetPath := filepath.Join(cwd, "CLAUDE.md")
	if err := os.WriteFile(targetPath, []byte("before"), 0644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	configPath := filepath.Join(home, ".cc-switch", "claude_templates.json")
	tm, err := template.NewTemplateManager(configPath)
	if err != nil {
		t.Fatalf("new template manager: %v", err)
	}
	templateID, err := tm.AddTemplate("confirm-template", template.CategoryClaudeMd, "after")
	if err != nil {
		t.Fatalf("add template: %v", err)
	}

	applyTarget = "project_root"
	skipDiff = false
	withStdin(t, "y\n", func() {
		testutil.CaptureOutput(t, func() {
			runApplyTemplate(templateID)
		})
	})
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(data) != "after" {
		t.Fatalf("expected applied content, got: %s", string(data))
	}

	deleteForce = false
	withStdin(t, "y\n", func() {
		testutil.CaptureOutput(t, func() {
			runDeleteTemplate(templateID)
		})
	})
	tm, err = template.NewTemplateManager(configPath)
	if err != nil {
		t.Fatalf("new template manager: %v", err)
	}
	if _, err := tm.GetTemplate(templateID); err == nil {
		t.Fatalf("expected template to be deleted")
	}
}

func TestTemplateErrorPaths(t *testing.T) {
	resetGlobals()
	origExit := exitFunc
	exitFunc = func(code int) {
		panic(code)
	}
	t.Cleanup(func() { exitFunc = origExit })

	expectExitCode := func(t *testing.T, want int) func() {
		t.Helper()
		return func() {
			r := recover()
			if r == nil {
				t.Fatalf("expected exit panic")
			}
			code, ok := r.(int)
			if !ok || code != want {
				t.Fatalf("expected exit code %d, got %#v", want, r)
			}
		}
	}

	t.Run("apply target nil", func(t *testing.T) {
		home := withTempHome(t)
		configPath := filepath.Join(home, ".cc-switch", "claude_templates.json")
		tm, err := template.NewTemplateManager(configPath)
		if err != nil {
			t.Fatalf("new template manager: %v", err)
		}
		templateID, err := tm.AddTemplate("test", template.CategoryClaudeMd, "content")
		if err != nil {
			t.Fatalf("add template: %v", err)
		}
		applyTarget = ""
		skipDiff = true
		defer expectExitCode(t, 1)()
		withStdin(t, "999\n", func() {
			runApplyTemplate(templateID)
		})
	})

	t.Run("add missing file", func(t *testing.T) {
		withTempHome(t)
		addFile = filepath.Join(t.TempDir(), "missing.md")
		addName = "missing"
		addCategory = template.CategoryClaudeMd
		defer expectExitCode(t, 1)()
		runAddTemplate()
	})

	t.Run("add read error", func(t *testing.T) {
		withTempHome(t)
		dir := t.TempDir()
		addFile = dir
		addName = "dir"
		addCategory = template.CategoryClaudeMd
		defer expectExitCode(t, 1)()
		runAddTemplate()
	})

	t.Run("add invalid config", func(t *testing.T) {
		home := withTempHome(t)
		configPath := filepath.Join(home, ".cc-switch", "claude_templates.json")
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			t.Fatalf("mkdir config dir: %v", err)
		}
		if err := os.WriteFile(configPath, []byte("{bad json"), 0644); err != nil {
			t.Fatalf("write config: %v", err)
		}
		file := filepath.Join(t.TempDir(), "ok.md")
		if err := os.WriteFile(file, []byte("ok"), 0644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		addFile = file
		addName = "bad-config"
		addCategory = template.CategoryClaudeMd
		defer expectExitCode(t, 1)()
		runAddTemplate()
	})

	t.Run("add permission error", func(t *testing.T) {
		home := withTempHome(t)
		configDir := filepath.Join(home, ".cc-switch")
		if err := os.MkdirAll(configDir, 0555); err != nil {
			t.Fatalf("mkdir config dir: %v", err)
		}
		file := filepath.Join(t.TempDir(), "ok.md")
		if err := os.WriteFile(file, []byte("ok"), 0644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		addFile = file
		addName = "no-perm"
		addCategory = template.CategoryClaudeMd
		defer expectExitCode(t, 1)()
		runAddTemplate()
	})

	t.Run("save invalid source", func(t *testing.T) {
		withTempHome(t)
		withTempCWD(t)
		saveFrom = "invalid"
		saveName = "invalid"
		defer expectExitCode(t, 1)()
		runSaveTemplate()
	})

	t.Run("save invalid selection", func(t *testing.T) {
		withTempHome(t)
		withTempCWD(t)
		saveFrom = ""
		saveName = "invalid"
		defer expectExitCode(t, 1)()
		withStdin(t, "9\n", func() {
			runSaveTemplate()
		})
	})

	t.Run("save missing file", func(t *testing.T) {
		withTempHome(t)
		withTempCWD(t)
		saveFrom = "project_root"
		saveName = "missing"
		defer expectExitCode(t, 1)()
		runSaveTemplate()
	})

	t.Run("save invalid selection", func(t *testing.T) {
		withTempHome(t)
		withTempCWD(t)
		saveFrom = ""
		saveName = "invalid"
		defer expectExitCode(t, 1)()
		withStdin(t, "9\n", func() {
			runSaveTemplate()
		})
	})

	t.Run("save invalid config", func(t *testing.T) {
		home := withTempHome(t)
		withTempCWD(t)
		configPath := filepath.Join(home, ".cc-switch", "claude_templates.json")
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			t.Fatalf("mkdir config dir: %v", err)
		}
		if err := os.WriteFile(configPath, []byte("{bad json"), 0644); err != nil {
			t.Fatalf("write config: %v", err)
		}
		saveFrom = "project_root"
		saveName = "bad-config"
		defer expectExitCode(t, 1)()
		runSaveTemplate()
	})

	t.Run("save invalid selection", func(t *testing.T) {
		withTempHome(t)
		withTempCWD(t)
		saveFrom = ""
		saveName = "invalid"
		defer expectExitCode(t, 1)()
		withStdin(t, "9\n", func() {
			runSaveTemplate()
		})
	})

	t.Run("apply invalid target", func(t *testing.T) {
		home := withTempHome(t)
		withTempCWD(t)
		configPath := filepath.Join(home, ".cc-switch", "claude_templates.json")
		tm, err := template.NewTemplateManager(configPath)
		if err != nil {
			t.Fatalf("new template manager: %v", err)
		}
		templateID, err := tm.AddTemplate("apply", template.CategoryClaudeMd, "content")
		if err != nil {
			t.Fatalf("add template: %v", err)
		}
		applyTarget = "invalid"
		defer expectExitCode(t, 1)()
		runApplyTemplate(templateID)
	})

	t.Run("apply invalid config", func(t *testing.T) {
		home := withTempHome(t)
		withTempCWD(t)
		configPath := filepath.Join(home, ".cc-switch", "claude_templates.json")
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			t.Fatalf("mkdir config dir: %v", err)
		}
		if err := os.WriteFile(configPath, []byte("{bad json"), 0644); err != nil {
			t.Fatalf("write config: %v", err)
		}
		applyTarget = "project_root"
		defer expectExitCode(t, 1)()
		runApplyTemplate("missing-template")
	})

	t.Run("apply invalid selection", func(t *testing.T) {
		withTempHome(t)
		withTempCWD(t)
		applyTarget = ""
		defer expectExitCode(t, 1)()
		withStdin(t, "9\n", func() {
			runApplyTemplate("missing-template")
		})
	})

	t.Run("apply write error", func(t *testing.T) {
		withTempHome(t)
		cwd := withTempCWD(t)
		dirTarget := filepath.Join(cwd, "CLAUDE.md")
		if err := os.MkdirAll(dirTarget, 0755); err != nil {
			t.Fatalf("mkdir target dir: %v", err)
		}
		configPath := filepath.Join(cwd, "tmp", "claude_templates.json")
		tm, err := template.NewTemplateManager(configPath)
		if err != nil {
			t.Fatalf("new template manager: %v", err)
		}
		templateID, err := tm.AddTemplate("apply", template.CategoryClaudeMd, "content")
		if err != nil {
			t.Fatalf("add template: %v", err)
		}
		applyTarget = "project_root"
		skipDiff = true
		defer expectExitCode(t, 1)()
		runApplyTemplate(templateID)
	})

	t.Run("apply invalid config", func(t *testing.T) {
		home := withTempHome(t)
		withTempCWD(t)
		configPath := filepath.Join(home, ".cc-switch", "claude_templates.json")
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			t.Fatalf("mkdir config dir: %v", err)
		}
		if err := os.WriteFile(configPath, []byte("{bad json"), 0644); err != nil {
			t.Fatalf("write config: %v", err)
		}
		applyTarget = "project_root"
		defer expectExitCode(t, 1)()
		runApplyTemplate("missing-template")
	})

	t.Run("apply missing template", func(t *testing.T) {
		withTempHome(t)
		withTempCWD(t)
		applyTarget = "project_root"
		defer expectExitCode(t, 1)()
		runApplyTemplate("missing-template")
	})

	t.Run("delete missing template", func(t *testing.T) {
		withTempHome(t)
		deleteForce = true
		defer expectExitCode(t, 1)()
		runDeleteTemplate("missing-template")
	})

	t.Run("delete builtin template", func(t *testing.T) {
		withTempHome(t)
		deleteForce = true
		defer expectExitCode(t, 1)()
		runDeleteTemplate("engineer-professional")
	})

	t.Run("delete invalid config", func(t *testing.T) {
		home := withTempHome(t)
		configPath := filepath.Join(home, ".cc-switch", "claude_templates.json")
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			t.Fatalf("mkdir config dir: %v", err)
		}
		if err := os.WriteFile(configPath, []byte("{bad json"), 0644); err != nil {
			t.Fatalf("write config: %v", err)
		}
		deleteForce = true
		defer expectExitCode(t, 1)()
		runDeleteTemplate("missing-template")
	})

	t.Run("list invalid config", func(t *testing.T) {
		home := withTempHome(t)
		configPath := filepath.Join(home, ".cc-switch", "claude_templates.json")
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			t.Fatalf("mkdir config dir: %v", err)
		}
		if err := os.WriteFile(configPath, []byte("{bad json"), 0644); err != nil {
			t.Fatalf("write config: %v", err)
		}
		defer expectExitCode(t, 1)()
		runListTemplates()
	})
}
