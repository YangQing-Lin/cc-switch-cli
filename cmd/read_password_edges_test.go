package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
)

func TestReadPasswordInjection(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	t.Run("claude update keeps current token on empty input", func(t *testing.T) {
		manager := setupManager(t)
		addClaudeProvider(t, manager, "p")

		orig := cmdReadPasswordFunc
		cmdReadPasswordFunc = func(_ int) ([]byte, error) { return []byte(""), nil }
		t.Cleanup(func() { cmdReadPasswordFunc = orig })

		resetFlags(updateCmd)
		if err := updateCmd.Flags().Set("app", "claude"); err != nil {
			t.Fatalf("set flag: %v", err)
		}

		stdout, _ := testutil.CaptureOutput(t, func() {
			if err := updateCmd.RunE(updateCmd, []string{"p"}); err != nil {
				t.Fatalf("update: %v", err)
			}
		})
		if !strings.Contains(stdout, "当前 API Token") {
			t.Fatalf("expected prompt output, got: %s", stdout)
		}
	})

	t.Run("claude update read error when no current token", func(t *testing.T) {
		manager := setupManager(t)
		p := config.Provider{
			ID:   "no-token-id",
			Name: "no-token",
			SettingsConfig: map[string]interface{}{
				"env": map[string]interface{}{
					"ANTHROPIC_BASE_URL": "https://api.example.com",
				},
			},
		}
		if err := manager.AddProviderDirect("claude", p); err != nil {
			t.Fatalf("add provider: %v", err)
		}

		orig := cmdReadPasswordFunc
		cmdReadPasswordFunc = func(_ int) ([]byte, error) { return nil, errors.New("no tty") }
		t.Cleanup(func() { cmdReadPasswordFunc = orig })

		resetFlags(updateCmd)
		if err := updateCmd.Flags().Set("app", "claude"); err != nil {
			t.Fatalf("set flag: %v", err)
		}

		err := updateCmd.RunE(updateCmd, []string{"no-token"})
		if err == nil || !strings.Contains(err.Error(), "读取输入失败") {
			t.Fatalf("expected read error, got: %v", err)
		}
	})

	t.Run("codex update keeps current key on empty input", func(t *testing.T) {
		manager := setupManager(t)
		addCodexProvider(t, manager, "c", false)
		if err := manager.Save(); err != nil {
			t.Fatalf("save: %v", err)
		}

		orig := cmdReadPasswordFunc
		cmdReadPasswordFunc = func(_ int) ([]byte, error) { return []byte(""), nil }
		t.Cleanup(func() { cmdReadPasswordFunc = orig })

		resetFlags(codexUpdateCmd)
		stdout, _ := testutil.CaptureOutput(t, func() {
			if err := codexUpdateCmd.RunE(codexUpdateCmd, []string{"c"}); err != nil {
				t.Fatalf("codex update: %v", err)
			}
		})
		if !strings.Contains(stdout, "当前 API Key") {
			t.Fatalf("expected prompt output, got: %s", stdout)
		}
	})

	t.Run("codex update read error when no current key", func(t *testing.T) {
		manager := setupManager(t)
		p := config.Provider{
			ID:   "no-key-id",
			Name: "no-key",
			SettingsConfig: map[string]interface{}{
				"config": map[string]interface{}{
					"base_url":   "https://api.example.com",
					"model_name": "m",
				},
			},
		}
		if err := manager.AddProviderDirect("codex", p); err != nil {
			t.Fatalf("add provider: %v", err)
		}

		orig := cmdReadPasswordFunc
		cmdReadPasswordFunc = func(_ int) ([]byte, error) { return nil, errors.New("no tty") }
		t.Cleanup(func() { cmdReadPasswordFunc = orig })

		resetFlags(codexUpdateCmd)
		err := codexUpdateCmd.RunE(codexUpdateCmd, []string{"no-key"})
		if err == nil || !strings.Contains(err.Error(), "读取输入失败") {
			t.Fatalf("expected read error, got: %v", err)
		}
	})
}
