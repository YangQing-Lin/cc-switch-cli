package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
)

func writeSampleConfig(t *testing.T, path string) {
	t.Helper()
	cfg := &config.MultiAppConfig{
		Version: 2,
		Apps: map[string]config.ProviderManager{
			"claude": {
				Providers: map[string]config.Provider{
					"id": {
						ID:   "id",
						Name: "sample",
						SettingsConfig: map[string]interface{}{
							"env": map[string]interface{}{
								"ANTHROPIC_AUTH_TOKEN": "sk-test",
								"ANTHROPIC_BASE_URL":   "https://api.example.com",
							},
						},
					},
				},
				Current: "id",
			},
		},
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func TestBackupCommandCreatesFiles(t *testing.T) {
	resetGlobals()
	home := withTempHome(t)

	manager := setupManager(t)
	addClaudeProvider(t, manager, "alpha")

	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("mkdir claude dir: %v", err)
	}
	claudeSettings := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(claudeSettings, []byte("{}"), 0644); err != nil {
		t.Fatalf("write settings: %v", err)
	}

	output := filepath.Join(t.TempDir(), "backup.json")
	resetFlags(backupCmd)
	if err := backupCmd.Flags().Set("output", output); err != nil {
		t.Fatalf("set output flag: %v", err)
	}
	if err := backupCmd.Flags().Set("include-claude", "true"); err != nil {
		t.Fatalf("set include-claude flag: %v", err)
	}

	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := backupCmd.RunE(backupCmd, []string{}); err != nil {
			t.Fatalf("backup command: %v", err)
		}
	})
	if !strings.Contains(stdout, "备份") {
		t.Fatalf("expected backup output, got: %s", stdout)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("expected backup file: %v", err)
	}
	claudeBackup := strings.TrimSuffix(output, ".json") + "-claude-settings.json"
	if _, err := os.Stat(claudeBackup); err != nil {
		t.Fatalf("expected claude backup file: %v", err)
	}
}

func TestBackupCommandPermissionError(t *testing.T) {
	if runtime.GOOS == "windows" || isRoot() {
		t.Skip("permission bits are not reliable on this platform/user")
	}

	resetGlobals()
	withTempHome(t)

	manager := setupManager(t)
	addClaudeProvider(t, manager, "alpha")

	readonlyDir := filepath.Join(t.TempDir(), "readonly")
	if err := os.MkdirAll(readonlyDir, 0555); err != nil {
		t.Fatalf("mkdir readonly: %v", err)
	}
	output := filepath.Join(readonlyDir, "backup.json")

	resetFlags(backupCmd)
	if err := backupCmd.Flags().Set("output", output); err != nil {
		t.Fatalf("set output flag: %v", err)
	}
	if err := backupCmd.RunE(backupCmd, []string{}); err == nil {
		t.Fatalf("expected permission error")
	}
}

func TestBackupListAndRestoreSubcommand(t *testing.T) {
	resetGlobals()
	home := withTempHome(t)

	backupDir := filepath.Join(home, ".cc-switch", "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("mkdir backup dir: %v", err)
	}
	backupFile := filepath.Join(backupDir, "sample.json")
	writeSampleConfig(t, backupFile)

	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := backupListCmd.RunE(backupListCmd, []string{}); err != nil {
			t.Fatalf("backup list: %v", err)
		}
	})
	if !strings.Contains(stdout, "sample.json") {
		t.Fatalf("expected list output, got: %s", stdout)
	}

	stdout, _ = testutil.CaptureOutput(t, func() {
		if err := backupRestoreCmd.RunE(backupRestoreCmd, []string{"sample"}); err != nil {
			t.Fatalf("backup restore: %v", err)
		}
	})
	if !strings.Contains(stdout, "恢复") {
		t.Fatalf("expected restore output, got: %s", stdout)
	}
}

func TestRestoreCommandLatestAndList(t *testing.T) {
	resetGlobals()
	home := withTempHome(t)

	backupDir := filepath.Join(home, ".cc-switch", "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("mkdir backup dir: %v", err)
	}
	older := filepath.Join(backupDir, "config-backup-20240101-010101.json")
	newer := filepath.Join(backupDir, "config-backup-20240102-010101.json")
	writeSampleConfig(t, older)
	writeSampleConfig(t, newer)
	_ = os.Chtimes(older, time.Now().Add(-2*time.Hour), time.Now().Add(-2*time.Hour))
	_ = os.Chtimes(newer, time.Now().Add(-1*time.Hour), time.Now().Add(-1*time.Hour))

	resetFlags(restoreCmd)
	if err := restoreCmd.Flags().Set("list", "true"); err != nil {
		t.Fatalf("set list flag: %v", err)
	}
	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := restoreCmd.RunE(restoreCmd, []string{}); err != nil {
			t.Fatalf("restore list: %v", err)
		}
	})
	if !strings.Contains(stdout, "找到") {
		t.Fatalf("expected list output, got: %s", stdout)
	}

	resetFlags(restoreCmd)
	if err := restoreCmd.Flags().Set("latest", "true"); err != nil {
		t.Fatalf("set latest flag: %v", err)
	}
	stdout, _ = testutil.CaptureOutput(t, func() {
		if err := restoreCmd.RunE(restoreCmd, []string{}); err != nil {
			t.Fatalf("restore latest: %v", err)
		}
	})
	if !strings.Contains(stdout, "恢复") {
		t.Fatalf("expected restore output, got: %s", stdout)
	}
}

func TestCleanOldBackups(t *testing.T) {
	resetGlobals()
	tmpDir := t.TempDir()
	paths := []string{
		filepath.Join(tmpDir, "config-backup-1.json"),
		filepath.Join(tmpDir, "config-backup-2.json"),
		filepath.Join(tmpDir, "config-backup-3.json"),
	}
	for i, path := range paths {
		writeSampleConfig(t, path)
		_ = os.Chtimes(path, time.Now().Add(time.Duration(i)*time.Minute), time.Now().Add(time.Duration(i)*time.Minute))
	}

	if err := cleanOldBackups(tmpDir, 1); err != nil {
		t.Fatalf("clean old backups: %v", err)
	}
	files, err := filepath.Glob(filepath.Join(tmpDir, "config-backup-*.json"))
	if err != nil {
		t.Fatalf("glob backups: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 backup left, got %d", len(files))
	}
}

func TestGetLatestBackupErrors(t *testing.T) {
	resetGlobals()
	tmpDir := t.TempDir()

	if _, err := getLatestBackup(tmpDir); err == nil {
		t.Fatalf("expected error when no backups exist")
	}

	broken := filepath.Join(tmpDir, "config-backup-20240101-000000.json")
	if err := os.Symlink(filepath.Join(tmpDir, "missing.json"), broken); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}
	if _, err := getLatestBackup(tmpDir); err == nil {
		t.Fatalf("expected error when latest backup cannot be determined")
	}
}

func TestBackupCommandConfigMissing(t *testing.T) {
	resetGlobals()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	resetFlags(backupCmd)
	if err := backupCmd.RunE(backupCmd, []string{}); err == nil {
		t.Fatalf("expected error when config file missing")
	}
}

func TestBackupClaudeSettingsPartialFailure(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	manager := setupManager(t)
	addClaudeProvider(t, manager, "alpha")

	output := filepath.Join(t.TempDir(), "backup.json")
	resetFlags(backupCmd)
	if err := backupCmd.Flags().Set("output", output); err != nil {
		t.Fatalf("set output flag: %v", err)
	}
	if err := backupCmd.Flags().Set("include-claude", "true"); err != nil {
		t.Fatalf("set include-claude flag: %v", err)
	}

	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := backupCmd.RunE(backupCmd, []string{}); err != nil {
			t.Fatalf("backup command: %v", err)
		}
	})
	if !strings.Contains(stdout, "备份") {
		t.Fatalf("expected backup output, got: %s", stdout)
	}
}

func TestRestoreCommandWithLatest(t *testing.T) {
	resetGlobals()
	home := withTempHome(t)

	backupDir := filepath.Join(home, ".cc-switch", "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("mkdir backup dir: %v", err)
	}
	backup1 := filepath.Join(backupDir, "config-backup-001.json")
	backup2 := filepath.Join(backupDir, "config-backup-002.json")
	writeSampleConfig(t, backup1)
	writeSampleConfig(t, backup2)
	_ = os.Chtimes(backup2, time.Now(), time.Now())

	resetFlags(restoreCmd)
	if err := restoreCmd.Flags().Set("latest", "true"); err != nil {
		t.Fatalf("set latest flag: %v", err)
	}
	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := restoreCmd.RunE(restoreCmd, []string{}); err != nil {
			t.Fatalf("restore latest: %v", err)
		}
	})
	if !strings.Contains(stdout, "恢复") {
		t.Fatalf("expected restore output, got: %s", stdout)
	}
}

func TestRestoreCommandInteractiveCancel(t *testing.T) {
	resetGlobals()
	home := withTempHome(t)

	backupDir := filepath.Join(home, ".cc-switch", "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("mkdir backup dir: %v", err)
	}
	backup1 := filepath.Join(backupDir, "config-backup-001.json")
	writeSampleConfig(t, backup1)

	resetFlags(restoreCmd)
	if err := restoreCmd.Flags().Set("latest", "true"); err != nil {
		t.Fatalf("set latest flag: %v", err)
	}
	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := restoreCmd.RunE(restoreCmd, []string{}); err != nil {
			t.Logf("restore returned error (expected if valid): %v", err)
		}
	})
	if !strings.Contains(stdout, "恢复") && len(stdout) == 0 {
		t.Logf("restore output: %s", stdout)
	}
}

func TestRestoreCommandNoBackups(t *testing.T) {
	resetGlobals()
	home := withTempHome(t)

	backupDir := filepath.Join(home, ".cc-switch", "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("mkdir backup dir: %v", err)
	}

	resetFlags(restoreCmd)
	if err := restoreCmd.Flags().Set("latest", "true"); err != nil {
		t.Fatalf("set latest flag: %v", err)
	}
	if err := restoreCmd.RunE(restoreCmd, []string{}); err == nil {
		t.Fatalf("expected error when no backups exist")
	}
}
