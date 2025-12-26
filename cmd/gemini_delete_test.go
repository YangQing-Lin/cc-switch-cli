package cmd

import (
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
)

func TestGeminiDeleteCmd(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	manager := setupManager(t)
	if err := manager.AddGeminiProvider("g1", "https://api.example.com", "sk-gemini", "gemini-2.5-pro", config.GeminiAuthAPIKey); err != nil {
		t.Fatalf("add gemini provider: %v", err)
	}
	if err := manager.AddGeminiProvider("g2", "https://api.example.com", "sk-gemini", "gemini-2.5-pro", config.GeminiAuthAPIKey); err != nil {
		t.Fatalf("add gemini provider: %v", err)
	}
	// 删除当前激活配置会失败，先切到另一个 provider
	if err := manager.SwitchProviderForApp("gemini", "g2"); err != nil {
		t.Fatalf("switch provider: %v", err)
	}

	t.Run("cancel", func(t *testing.T) {
		withStdin(t, "n\n", func() {
			stdout, _ := testutil.CaptureOutput(t, func() {
				if err := geminiDeleteCmd.RunE(geminiDeleteCmd, []string{"g1"}); err != nil {
					t.Fatalf("gemini delete: %v", err)
				}
			})
			if stdout == "" {
				t.Fatalf("expected output")
			}
		})

		verify := setupManager(t)
		if _, err := verify.GetProviderForApp("gemini", "g1"); err != nil {
			t.Fatalf("expected provider still exists: %v", err)
		}
	})

	t.Run("confirm", func(t *testing.T) {
		withStdin(t, "y\n", func() {
			stdout, _ := testutil.CaptureOutput(t, func() {
				if err := geminiDeleteCmd.RunE(geminiDeleteCmd, []string{"g1"}); err != nil {
					t.Fatalf("gemini delete: %v", err)
				}
			})
			if stdout == "" {
				t.Fatalf("expected output")
			}
		})

		verify := setupManager(t)
		if _, err := verify.GetProviderForApp("gemini", "g1"); err == nil {
			t.Fatalf("expected provider deleted")
		}
	})
}
