package cmd

import (
	"strings"
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
)

func TestInteractivePrompts(t *testing.T) {
	t.Run("codex add prompts for missing flags", func(t *testing.T) {
		resetGlobals()
		withTempHome(t)

		orig := readPassword
		readPassword = func(_ int) ([]byte, error) { return []byte("sk-codex"), nil }
		t.Cleanup(func() { readPassword = orig })

		resetFlags(codexAddCmd)
		withStdin(t, "\n", func() {
			stdout, _ := testutil.CaptureOutput(t, func() {
				if err := codexAddCmd.RunE(codexAddCmd, []string{"codex-prompt"}); err != nil {
					t.Fatalf("codex add: %v", err)
				}
			})
			if !strings.Contains(stdout, "已添加成功") || !strings.Contains(stdout, "Base URL") {
				t.Fatalf("expected success output, got: %s", stdout)
			}
		})
	})

	t.Run("gemini add prompts for missing flags (api-key mode)", func(t *testing.T) {
		resetGlobals()
		withTempHome(t)

		orig := readPassword
		readPassword = func(_ int) ([]byte, error) { return []byte("sk-gemini"), nil }
		t.Cleanup(func() { readPassword = orig })

		resetFlags(geminiAddCmd)
		withStdin(t, "\n", func() {
			stdout, _ := testutil.CaptureOutput(t, func() {
				if err := geminiAddCmd.RunE(geminiAddCmd, []string{"gemini-prompt"}); err != nil {
					t.Fatalf("gemini add: %v", err)
				}
			})
			if !strings.Contains(stdout, "已添加成功") || !strings.Contains(stdout, "API Key") {
				t.Fatalf("expected success output, got: %s", stdout)
			}
		})
	})
}
