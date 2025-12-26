package cmd

import (
	"strings"
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
)

func TestDeleteCommand(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	manager := setupManager(t)
	addClaudeProvider(t, manager, "p1")
	addClaudeProvider(t, manager, "p2")
	if err := manager.SwitchProvider("p2"); err != nil {
		t.Fatalf("switch provider: %v", err)
	}
	if err := manager.Save(); err != nil {
		t.Fatalf("save manager: %v", err)
	}

	t.Run("missing provider", func(t *testing.T) {
		resetFlags(deleteCmd)
		force = true
		stdout, _ := testutil.CaptureOutput(t, func() {
			err := deleteCmd.RunE(deleteCmd, []string{"missing"})
			if err == nil || !strings.Contains(err.Error(), "配置不存在") {
				t.Fatalf("expected missing error, got: %v", err)
			}
		})
		if stdout != "" {
			t.Fatalf("expected no stdout on early error, got: %s", stdout)
		}
	})

	t.Run("cancel", func(t *testing.T) {
		resetFlags(deleteCmd)
		force = false
		withStdin(t, "n\n", func() {
			stdout, _ := testutil.CaptureOutput(t, func() {
				if err := deleteCmd.RunE(deleteCmd, []string{"p1"}); err != nil {
					t.Fatalf("delete: %v", err)
				}
			})
			if !strings.Contains(stdout, "取消删除") {
				t.Fatalf("expected cancel output, got: %s", stdout)
			}
		})

		verify := setupManager(t)
		if _, err := verify.GetProvider("p1"); err != nil {
			t.Fatalf("expected provider still exists: %v", err)
		}
	})

	t.Run("confirm", func(t *testing.T) {
		resetFlags(deleteCmd)
		force = false
		withStdin(t, "y\n", func() {
			stdout, _ := testutil.CaptureOutput(t, func() {
				if err := deleteCmd.RunE(deleteCmd, []string{"p1"}); err != nil {
					t.Fatalf("delete: %v", err)
				}
			})
			if !strings.Contains(stdout, "已删除成功") {
				t.Fatalf("expected success output, got: %s", stdout)
			}
		})

		verify := setupManager(t)
		if _, err := verify.GetProvider("p1"); err == nil {
			t.Fatalf("expected provider deleted")
		}
	})
}
