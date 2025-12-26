package cmd

import (
	"strings"
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
)

func TestCodexCommands_Edges(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	t.Run("add trims and rejects empty api key", func(t *testing.T) {
		resetFlags(codexAddCmd)
		if err := codexAddCmd.Flags().Set("apikey", "   "); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		if err := codexAddCmd.Flags().Set("base-url", "https://api.example.com"); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		err := codexAddCmd.RunE(codexAddCmd, []string{"bad"})
		if err == nil || !strings.Contains(err.Error(), "API Key 不能为空") {
			t.Fatalf("expected api key error, got: %v", err)
		}
	})

	t.Run("add trims and rejects empty base url", func(t *testing.T) {
		resetFlags(codexAddCmd)
		if err := codexAddCmd.Flags().Set("apikey", "sk-ok"); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		if err := codexAddCmd.Flags().Set("base-url", "   "); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		err := codexAddCmd.RunE(codexAddCmd, []string{"bad"})
		if err == nil || !strings.Contains(err.Error(), "Base URL 不能为空") {
			t.Fatalf("expected base url error, got: %v", err)
		}
	})

	t.Run("add duplicate name fails", func(t *testing.T) {
		resetFlags(codexAddCmd)
		if err := codexAddCmd.Flags().Set("apikey", "sk-ok"); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		if err := codexAddCmd.Flags().Set("base-url", "https://api.example.com"); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		if err := codexAddCmd.RunE(codexAddCmd, []string{"dup"}); err != nil {
			t.Fatalf("codex add: %v", err)
		}
		err := codexAddCmd.RunE(codexAddCmd, []string{"dup"})
		if err == nil || !strings.Contains(err.Error(), "添加 Codex 配置失败") {
			t.Fatalf("expected duplicate error, got: %v", err)
		}
	})

	t.Run("update name conflict", func(t *testing.T) {
		resetFlags(codexUpdateCmd)

		manager := setupManager(t)
		addCodexProvider(t, manager, "c1", true)
		addCodexProvider(t, manager, "c2", false)
		if err := manager.Save(); err != nil {
			t.Fatalf("save: %v", err)
		}

		if err := codexUpdateCmd.Flags().Set("name", "c2"); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		if err := codexUpdateCmd.Flags().Set("apikey", "sk-new"); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		if err := codexUpdateCmd.Flags().Set("base-url", "https://api.example.com"); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		if err := codexUpdateCmd.Flags().Set("model", "m"); err != nil {
			t.Fatalf("set flag: %v", err)
		}

		err := codexUpdateCmd.RunE(codexUpdateCmd, []string{"c1"})
		if err == nil || !strings.Contains(err.Error(), "已存在") {
			t.Fatalf("expected conflict error, got: %v", err)
		}
	})

	t.Run("update trims and rejects empty api key", func(t *testing.T) {
		resetFlags(codexUpdateCmd)

		manager := setupManager(t)
		addCodexProvider(t, manager, "c3", false)
		if err := manager.Save(); err != nil {
			t.Fatalf("save: %v", err)
		}

		if err := codexUpdateCmd.Flags().Set("apikey", "   "); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		if err := codexUpdateCmd.Flags().Set("base-url", "https://api.example.com"); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		if err := codexUpdateCmd.Flags().Set("model", "m"); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		err := codexUpdateCmd.RunE(codexUpdateCmd, []string{"c3"})
		if err == nil || !strings.Contains(err.Error(), "不能为空") {
			t.Fatalf("expected required error, got: %v", err)
		}
	})

	t.Run("delete cancel and confirm", func(t *testing.T) {
		resetFlags(codexDeleteCmd)

		manager := setupManager(t)
		addCodexProvider(t, manager, "d1", false)
		addCodexProvider(t, manager, "d2", true)
		if err := manager.Save(); err != nil {
			t.Fatalf("save: %v", err)
		}

		withStdin(t, "n\n", func() {
			stdout, _ := testutil.CaptureOutput(t, func() {
				if err := codexDeleteCmd.RunE(codexDeleteCmd, []string{"d1"}); err != nil {
					t.Fatalf("delete: %v", err)
				}
			})
			if !strings.Contains(stdout, "取消删除") {
				t.Fatalf("expected cancel output, got: %s", stdout)
			}
		})

		resetFlags(codexDeleteCmd)
		withStdin(t, "y\n", func() {
			stdout, _ := testutil.CaptureOutput(t, func() {
				if err := codexDeleteCmd.RunE(codexDeleteCmd, []string{"d1"}); err != nil {
					t.Fatalf("delete: %v", err)
				}
			})
			if !strings.Contains(stdout, "已删除成功") {
				t.Fatalf("expected success output, got: %s", stdout)
			}
		})
	})
}
