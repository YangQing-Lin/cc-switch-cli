package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
)

func TestShowCmd_JSONAndVerbose(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	manager := setupManager(t)
	addClaudeProvider(t, manager, "p1")

	t.Run("json output", func(t *testing.T) {
		resetFlags(showCmd)
		if err := showCmd.Flags().Set("app", "claude"); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		if err := showCmd.Flags().Set("json", "true"); err != nil {
			t.Fatalf("set flag: %v", err)
		}

		stdout, _ := testutil.CaptureOutput(t, func() {
			if err := showCmd.RunE(showCmd, []string{"p1"}); err != nil {
				t.Fatalf("show: %v", err)
			}
		})

		var got map[string]any
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("unmarshal json: %v\n%s", err, stdout)
		}
		if got["name"] != "p1" {
			t.Fatalf("unexpected name: %v", got["name"])
		}
	})

	t.Run("verbose output", func(t *testing.T) {
		resetFlags(showCmd)
		if err := showCmd.Flags().Set("app", "claude"); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		if err := showCmd.Flags().Set("verbose", "true"); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		stdout, _ := testutil.CaptureOutput(t, func() {
			if err := showCmd.RunE(showCmd, []string{"p1"}); err != nil {
				t.Fatalf("show: %v", err)
			}
		})
		if !strings.Contains(stdout, "原始配置") {
			t.Fatalf("expected verbose output, got: %s", stdout)
		}
	})

	t.Run("codex branch output", func(t *testing.T) {
		resetGlobals()
		withTempHome(t)

		manager := setupManager(t)
		p := config.Provider{
			ID:   "codex-id",
			Name: "c1",
			SettingsConfig: map[string]interface{}{
				"auth":     "sk-codex",
				"base_url": "https://api.example.com",
			},
			Category: "custom",
		}
		if err := manager.AddProviderDirect("codex", p); err != nil {
			t.Fatalf("add codex provider: %v", err)
		}

		resetFlags(showCmd)
		if err := showCmd.Flags().Set("app", "codex"); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		stdout, _ := testutil.CaptureOutput(t, func() {
			if err := showCmd.RunE(showCmd, []string{"c1"}); err != nil {
				t.Fatalf("show: %v", err)
			}
		})
		if !strings.Contains(stdout, "Auth:") || !strings.Contains(stdout, "Base URL:") {
			t.Fatalf("expected codex fields, got: %s", stdout)
		}
	})
}
