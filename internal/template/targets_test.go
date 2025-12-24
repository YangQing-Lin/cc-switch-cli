package template

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
)

func TestTargetsForCategoryAndLookup(t *testing.T) {
	cases := []struct {
		name     string
		category string
	}{
		{name: "claude", category: CategoryClaudeMd},
		{name: "codex", category: CategoryCodexMd},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.WithTempHome(t, func(home string) {
				testutil.WithTempCWD(t, func(cwd string) {
					targets, err := GetTargetsForCategory(tc.category)
					if err != nil {
						t.Fatalf("GetTargetsForCategory() error = %v", err)
					}
					if len(targets) != 3 {
						t.Fatalf("expected 3 targets, got %d", len(targets))
					}

					var wantGlobal, wantProject, wantLocal string
					if tc.category == CategoryClaudeMd {
						wantGlobal = filepath.Join(home, ".claude", "CLAUDE.md")
						wantProject = filepath.Join(cwd, "CLAUDE.md")
						wantLocal = filepath.Join(cwd, "CLAUDE.local.md")
					} else {
						wantGlobal = filepath.Join(home, ".codex", "CODEX.md")
						wantProject = filepath.Join(cwd, "CODEX.md")
						wantLocal = filepath.Join(cwd, "CODEX.local.md")
					}

					if targets[0].Path != wantGlobal || targets[1].Path != wantProject || targets[2].Path != wantLocal {
						t.Fatalf("unexpected target paths: %#v", targets)
					}

					byID, err := GetTargetByCategory(tc.category, "project_root")
					if err != nil {
						t.Fatalf("GetTargetByCategory() error = %v", err)
					}
					if byID == nil || byID.Path != wantProject {
						t.Fatalf("unexpected project target: %#v", byID)
					}
				})
			})
		})
	}
}

func TestTargetsEdgeCases(t *testing.T) {
	cases := []struct {
		name     string
		category string
		id       string
		wantErr  bool
		wantNil  bool
	}{
		{name: "unsupported_category", category: "unknown", wantErr: true},
		{name: "unsupported_category_with_id", category: "unknown", id: "global", wantErr: true},
		{name: "missing_id", category: CategoryClaudeMd, id: "missing", wantNil: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.WithTempHome(t, func(_ string) {
				testutil.WithTempCWD(t, func(_ string) {
					if tc.wantErr {
						if tc.id != "" {
							if _, err := GetTargetByCategory(tc.category, tc.id); err == nil {
								t.Fatalf("expected error for unsupported category")
							}
							return
						}
						if _, err := GetTargetsForCategory(tc.category); err == nil {
							t.Fatalf("expected error for unsupported category")
						}
						return
					}

					got, err := GetTargetByCategory(tc.category, tc.id)
					if err != nil {
						t.Fatalf("GetTargetByCategory() error = %v", err)
					}
					if tc.wantNil && got != nil {
						t.Fatalf("expected nil target, got %#v", got)
					}
				})
			})
		})
	}
}

func TestGetTargetByID(t *testing.T) {
	cases := []struct {
		name    string
		id      string
		wantNil bool
	}{
		{name: "global", id: "global"},
		{name: "project_root", id: "project_root"},
		{name: "missing", id: "missing", wantNil: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.WithTempHome(t, func(_ string) {
				testutil.WithTempCWD(t, func(_ string) {
					got, err := GetTargetByID(tc.id)
					if err != nil {
						t.Fatalf("GetTargetByID() error = %v", err)
					}
					if tc.wantNil {
						if got != nil {
							t.Fatalf("expected nil target for id %s, got %#v", tc.id, got)
						}
						return
					}
					if got == nil {
						t.Fatalf("expected target for id %s", tc.id)
					}
				})
			})
		})
	}
}

func TestTargetsCwdError(t *testing.T) {
	// On Windows, the current working directory is typically locked by the process and cannot be deleted,
	// which makes this test (that relies on deleting the current working directory to force os.Getwd() to fail)
	// unreliable.
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows: cannot reliably delete current working directory to trigger Getwd failure")
	}

	cases := []struct {
		name string
		call func() ([]TemplateTarget, error)
	}{
		{name: "claude", call: GetClaudeMdTargets},
		{name: "codex", call: GetCodexMdTargets},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			orig, err := os.Getwd()
			if err != nil {
				t.Fatalf("Getwd() error = %v", err)
			}
			dir := t.TempDir()
			if err := os.Chdir(dir); err != nil {
				t.Fatalf("Chdir() error = %v", err)
			}
			if err := os.RemoveAll(dir); err != nil {
				t.Fatalf("RemoveAll() error = %v", err)
			}
			t.Cleanup(func() {
				_ = os.Chdir(orig)
			})

			if _, err := tc.call(); err == nil {
				t.Fatalf("expected Getwd error")
			}
		})
	}
}
