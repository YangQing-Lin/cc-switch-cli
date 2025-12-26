package cmd

import (
	"strings"
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
)

func TestUpdateConfigCommand_ErrorBranches(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	manager := setupManager(t)
	addClaudeProvider(t, manager, "old")

	t.Run("missing provider", func(t *testing.T) {
		resetFlags(updateCmd)
		if err := updateCmd.Flags().Set("app", "claude"); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		if err := updateCmd.Flags().Set("apikey", "sk-new"); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		if err := updateCmd.Flags().Set("base-url", "https://api.example.com"); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		err := updateCmd.RunE(updateCmd, []string{"missing"})
		if err == nil || !strings.Contains(err.Error(), "获取配置失败") {
			t.Fatalf("expected get error, got: %v", err)
		}
	})

	t.Run("invalid base url", func(t *testing.T) {
		resetFlags(updateCmd)
		if err := updateCmd.Flags().Set("app", "claude"); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		if err := updateCmd.Flags().Set("apikey", "sk-new"); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		if err := updateCmd.Flags().Set("base-url", "not-a-url"); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		err := updateCmd.RunE(updateCmd, []string{"old"})
		if err == nil || !strings.Contains(err.Error(), "无效的 Base URL") {
			t.Fatalf("expected url error, got: %v", err)
		}
	})

	t.Run("rename prints new name", func(t *testing.T) {
		resetFlags(updateCmd)
		if err := updateCmd.Flags().Set("app", "claude"); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		if err := updateCmd.Flags().Set("apikey", "sk-new"); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		if err := updateCmd.Flags().Set("base-url", "https://api.example.com"); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		if err := updateCmd.Flags().Set("name", "new"); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		stdout, _ := testutil.CaptureOutput(t, func() {
			if err := updateCmd.RunE(updateCmd, []string{"old"}); err != nil {
				t.Fatalf("update: %v", err)
			}
		})
		if !strings.Contains(stdout, "新名称") {
			t.Fatalf("expected rename output, got: %s", stdout)
		}
	})
}
