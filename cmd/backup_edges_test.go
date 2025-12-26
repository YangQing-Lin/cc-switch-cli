package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
)

func TestBackupCmd_Edges(t *testing.T) {
	t.Run("config file missing", func(t *testing.T) {
		resetGlobals()
		withTempHome(t)

		resetFlags(backupCmd)
		err := backupCmd.RunE(backupCmd, []string{})
		if err == nil || !strings.Contains(err.Error(), "配置文件不存在") {
			t.Fatalf("expected missing config error, got: %v", err)
		}
	})

	t.Run("include claude settings", func(t *testing.T) {
		resetGlobals()
		withTempHome(t)

		manager := setupManager(t)
		addClaudeProvider(t, manager, "p1")

		claudeSettingsPath, err := config.GetClaudeSettingsPath()
		if err != nil {
			t.Fatalf("get claude settings path: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(claudeSettingsPath), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		settingsContent := []byte(`{"ok":true}`)
		if err := os.WriteFile(claudeSettingsPath, settingsContent, 0644); err != nil {
			t.Fatalf("write settings: %v", err)
		}

		tmpDir := t.TempDir()
		backupPath := filepath.Join(tmpDir, "backup.json")

		resetFlags(backupCmd)
		if err := backupCmd.Flags().Set("output", backupPath); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		if err := backupCmd.Flags().Set("include-claude", "true"); err != nil {
			t.Fatalf("set flag: %v", err)
		}

		stdout, _ := testutil.CaptureOutput(t, func() {
			if err := backupCmd.RunE(backupCmd, []string{}); err != nil {
				t.Fatalf("backup: %v", err)
			}
		})
		if !strings.Contains(stdout, "Claude 设置已备份到") {
			t.Fatalf("expected claude backup output, got: %s", stdout)
		}

		claudeBackup := backupPath[:len(backupPath)-5] + "-claude-settings.json"
		got, err := os.ReadFile(claudeBackup)
		if err != nil {
			t.Fatalf("read claude backup: %v", err)
		}
		if string(got) != string(settingsContent) {
			t.Fatalf("unexpected claude backup content: %s", string(got))
		}
	})

	t.Run("keep triggers cleanup on default dir", func(t *testing.T) {
		resetGlobals()
		home := withTempHome(t)

		manager := setupManager(t)
		addClaudeProvider(t, manager, "p1")

		backupDir := filepath.Join(home, ".cc-switch", "backups")
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		older := filepath.Join(backupDir, "config-backup-20240101-010101.json")
		newer := filepath.Join(backupDir, "config-backup-20240102-010101.json")
		if err := os.WriteFile(older, []byte("{}"), 0644); err != nil {
			t.Fatalf("write: %v", err)
		}
		if err := os.WriteFile(newer, []byte("{}"), 0644); err != nil {
			t.Fatalf("write: %v", err)
		}
		_ = os.Chtimes(older, time.Now().Add(-2*time.Hour), time.Now().Add(-2*time.Hour))
		_ = os.Chtimes(newer, time.Now().Add(-1*time.Hour), time.Now().Add(-1*time.Hour))

		resetFlags(backupCmd)
		if err := backupCmd.Flags().Set("keep", "1"); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		if err := backupCmd.RunE(backupCmd, []string{}); err != nil {
			t.Fatalf("backup: %v", err)
		}

		files, err := filepath.Glob(filepath.Join(backupDir, "config-backup-*.json"))
		if err != nil {
			t.Fatalf("glob: %v", err)
		}
		if len(files) != 1 {
			t.Fatalf("expected 1 backup left, got %d: %v", len(files), files)
		}
	})
}
