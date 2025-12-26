package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/template"
	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
	"github.com/fatih/color"
)

func TestRunListTemplates(t *testing.T) {
	resetGlobals()
	home := withTempHome(t)

	configPath := filepath.Join(home, ".cc-switch", "claude_templates.json")
	tm, err := template.NewTemplateManager(configPath)
	if err != nil {
		t.Fatalf("new template manager: %v", err)
	}
	if _, err := tm.AddTemplate("user one", template.CategoryClaudeMd, "content"); err != nil {
		t.Fatalf("add template: %v", err)
	}

	origCategory := listCategory
	t.Cleanup(func() { listCategory = origCategory })

	t.Run("builtin and user sections", func(t *testing.T) {
		listCategory = template.CategoryClaudeMd
		stdout, _ := testutil.CaptureOutput(t, func() {
			orig := color.Output
			color.Output = os.Stdout
			defer func() { color.Output = orig }()

			runListTemplates()
		})
		if !strings.Contains(stdout, "Builtin Templates") || !strings.Contains(stdout, "User Templates") {
			t.Fatalf("expected both sections, got: %s", stdout)
		}
	})

	t.Run("empty category", func(t *testing.T) {
		listCategory = "missing-category"
		stdout, _ := testutil.CaptureOutput(t, func() {
			orig := color.Output
			color.Output = os.Stdout
			defer func() { color.Output = orig }()

			runListTemplates()
		})
		if !strings.Contains(stdout, "No templates found") {
			t.Fatalf("expected empty output, got: %s", stdout)
		}
	})
}
