package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
)

// skipPermissionTest 检查是否应该跳过权限测试（root用户或Windows）
func skipPermissionTest(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("跳过 Windows 权限测试")
	}
	if os.Getuid() == 0 {
		t.Skip("跳过 root 用户权限测试")
	}
}

func TestCreateBackup(t *testing.T) {
	tmpDir := t.TempDir()

	configPath := filepath.Join(tmpDir, "config.json")
	data := writeConfigFile(t, configPath, sampleConfig("current"))

	// 创建只读目录（目录不可写，导致无法创建备份子目录）
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	readOnlyConfig := filepath.Join(readOnlyDir, "config.json")
	_ = writeConfigFile(t, readOnlyConfig, sampleConfig("readonly"))
	if runtime.GOOS != "windows" {
		// 设置目录为不可写，这样创建 backups 子目录时会失败
		if err := os.Chmod(readOnlyDir, 0555); err != nil {
			t.Fatalf("设置目录权限失败: %v", err)
		}
		t.Cleanup(func() {
			_ = os.Chmod(readOnlyDir, 0755)
		})
	}

	unreadablePath := filepath.Join(tmpDir, "unreadable.json")
	_ = writeConfigFile(t, unreadablePath, sampleConfig("unreadable"))
	if runtime.GOOS != "windows" {
		if err := os.Chmod(unreadablePath, 0000); err != nil {
			t.Fatalf("设置文件权限失败: %v", err)
		}
		t.Cleanup(func() {
			_ = os.Chmod(unreadablePath, 0600)
		})
	}

	tests := []struct {
		name      string
		path      string
		wantID    bool
		wantErr   bool
		skipOnWin bool
		verify    bool
	}{
		{
			name:   "create backup",
			path:   configPath,
			wantID: true,
			verify: true,
		},
		{
			name:   "missing config",
			path:   filepath.Join(tmpDir, "missing.json"),
			wantID: false,
		},
		{
			name:      "permission denied",
			path:      readOnlyConfig,
			wantErr:   true,
			skipOnWin: true,
		},
		{
			name:      "unreadable file",
			path:      unreadablePath,
			wantErr:   true,
			skipOnWin: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnWin {
				skipPermissionTest(t)
			}

			backupID, err := CreateBackup(tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("CreateBackup() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantID && backupID == "" {
				t.Fatalf("期望返回备份 ID")
			}
			if !tt.wantID && backupID != "" {
				t.Fatalf("未期望的备份 ID: %s", backupID)
			}

			if !tt.verify {
				return
			}

			if !strings.HasPrefix(backupID, backupPrefix) {
				t.Fatalf("备份前缀不匹配: %s", backupID)
			}

			backupPath := filepath.Join(filepath.Dir(tt.path), BackupDirName, backupID+".json")
			if _, err := os.Stat(backupPath); err != nil {
				t.Fatalf("备份文件不存在: %v", err)
			}

			backupData, err := os.ReadFile(backupPath)
			if err != nil {
				t.Fatalf("读取备份失败: %v", err)
			}
			if string(backupData) != string(data) {
				t.Fatalf("备份内容不匹配")
			}
		})
	}
}

func TestCleanupOldBackups(t *testing.T) {
	setupBackupDir := func(t *testing.T, backupCount int) string {
		t.Helper()

		tmpDir := t.TempDir()
		backupDir := filepath.Join(tmpDir, BackupDirName)

		if err := os.MkdirAll(backupDir, 0755); err != nil {
			t.Fatalf("创建备份目录失败: %v", err)
		}

		now := time.Now()
		for i := 0; i < backupCount; i++ {
			name := string(rune('a'+i)) + ".json"
			// 更早的文件使用更旧的时间戳，确保排序稳定
			createBackupFile(t, backupDir, name, now.Add(time.Duration(-(backupCount-i))*time.Hour))
		}

		// 添加非 .json 文件和子目录来测试过滤逻辑
		if err := os.WriteFile(filepath.Join(backupDir, "note.txt"), []byte("ignore"), 0644); err != nil {
			t.Fatalf("写入文件失败: %v", err)
		}
		if err := os.MkdirAll(filepath.Join(backupDir, "subdir"), 0755); err != nil {
			t.Fatalf("创建子目录失败: %v", err)
		}

		return backupDir
	}

	tests := []struct {
		name      string
		setupDir  func(t *testing.T) string
		retain    int
		wantCount int
	}{
		{"retain 0", func(t *testing.T) string { return setupBackupDir(t, 3) }, 0, 3},
		{"retain 2", func(t *testing.T) string { return setupBackupDir(t, 3) }, 2, 2},
		{"dir missing", func(t *testing.T) string { return filepath.Join(t.TempDir(), "missing") }, 2, 0},
		// 测试 len(backups) <= retain 分支
		{"retain more than exists", func(t *testing.T) string { return setupBackupDir(t, 2) }, 10, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backupDir := tt.setupDir(t)
			if err := CleanupOldBackups(backupDir, tt.retain); err != nil {
				t.Fatalf("CleanupOldBackups() error = %v", err)
			}
			if tt.wantCount == 0 {
				return
			}

			entries, err := os.ReadDir(backupDir)
			if err != nil {
				t.Fatalf("读取目录失败: %v", err)
			}
			count := 0
			for _, entry := range entries {
				if filepath.Ext(entry.Name()) == ".json" {
					count++
				}
			}
			if count != tt.wantCount {
				t.Fatalf("备份数量不匹配: %d != %d", count, tt.wantCount)
			}
		})
	}
}

func TestExportImportRestoreConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	exportPath := filepath.Join(tmpDir, "export.json")
	importPath := filepath.Join(tmpDir, "import.json")
	restorePath := filepath.Join(tmpDir, "restore.json")

	data := writeConfigFile(t, configPath, sampleConfig("export"))
	_ = writeConfigFile(t, importPath, sampleConfig("import"))
	_ = writeConfigFile(t, restorePath, sampleConfig("restore"))

	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	readOnlyConfig := filepath.Join(readOnlyDir, "config.json")
	_ = writeConfigFile(t, readOnlyConfig, sampleConfig("readonly"))
	if runtime.GOOS != "windows" {
		// 设置配置文件为只读，这样写入操作会失败
		if err := os.Chmod(readOnlyConfig, 0444); err != nil {
			t.Fatalf("设置文件权限失败: %v", err)
		}
		t.Cleanup(func() {
			_ = os.Chmod(readOnlyConfig, 0644)
		})
	}

	devFullAvailable := supportsDevFull()

	tests := []struct {
		name      string
		action    func() error
		wantErr   bool
		errSubstr string
		skipOnWin bool
		skip      bool
	}{
		{
			name: "export success",
			action: func() error {
				return ExportConfig(configPath, exportPath)
			},
		},
		{
			name: "export corrupted config",
			action: func() error {
				corrupt := filepath.Join(tmpDir, "corrupt.json")
				if err := os.WriteFile(corrupt, []byte("{invalid"), 0600); err != nil {
					return err
				}
				return ExportConfig(corrupt, filepath.Join(tmpDir, "out.json"))
			},
			wantErr:   true,
			errSubstr: "corrupted",
		},
		{
			name: "export disk full",
			action: func() error {
				return ExportConfig(configPath, "/dev/full")
			},
			wantErr: true,
			skip:    !devFullAvailable,
		},
		{
			name: "import success",
			action: func() error {
				_, err := ImportConfig(configPath, importPath)
				return err
			},
		},
		{
			name: "import invalid file",
			action: func() error {
				corrupt := filepath.Join(tmpDir, "bad.json")
				if err := os.WriteFile(corrupt, []byte("{invalid"), 0600); err != nil {
					return err
				}
				_, err := ImportConfig(configPath, corrupt)
				return err
			},
			wantErr:   true,
			errSubstr: "invalid",
		},
		{
			name: "import permission denied",
			action: func() error {
				_, err := ImportConfig(readOnlyConfig, importPath)
				return err
			},
			wantErr:   true,
			skipOnWin: true,
		},
		{
			name: "restore success",
			action: func() error {
				return RestoreBackup(configPath, restorePath)
			},
		},
		{
			name: "restore corrupted backup",
			action: func() error {
				corrupt := filepath.Join(tmpDir, "bad-backup.json")
				if err := os.WriteFile(corrupt, []byte("{invalid"), 0600); err != nil {
					return err
				}
				return RestoreBackup(configPath, corrupt)
			},
			wantErr:   true,
			errSubstr: "corrupted",
		},
		{
			name: "restore permission denied",
			action: func() error {
				return RestoreBackup(readOnlyConfig, restorePath)
			},
			wantErr:   true,
			skipOnWin: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip("跳过不支持的测试环境")
			}
			if tt.skipOnWin {
				skipPermissionTest(t)
			}

			err := tt.action()
			if (err != nil) != tt.wantErr {
				t.Fatalf("action error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.errSubstr != "" && (err == nil || !strings.Contains(err.Error(), tt.errSubstr)) {
				t.Fatalf("错误信息不匹配: %v", err)
			}
		})
	}

	// 验证导出内容与原始配置一致
	if got, err := os.ReadFile(exportPath); err == nil {
		if string(got) != string(data) {
			t.Fatalf("导出内容不匹配")
		}
		if runtime.GOOS != "windows" {
			info, err := os.Stat(exportPath)
			if err != nil {
				t.Fatalf("读取导出文件失败: %v", err)
			}
			if info.Mode().Perm() != 0600 {
				t.Fatalf("导出文件权限不匹配: %o", info.Mode().Perm())
			}
		}
	}
}

func TestListBackups(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, BackupDirName)

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("创建备份目录失败: %v", err)
	}

	createBackupFile(t, backupDir, "old.json", time.Now().Add(-2*time.Hour))
	createBackupFile(t, backupDir, "new.json", time.Now().Add(-1*time.Hour))
	if err := os.WriteFile(filepath.Join(backupDir, "note.txt"), []byte("ignore"), 0600); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}

	tests := []struct {
		name      string
		configDir string
		wantCount int
	}{
		{"list backups", tmpDir, 2},
		{"missing dir", filepath.Join(tmpDir, "missing"), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backups, err := ListBackups(tt.configDir)
			if err != nil {
				t.Fatalf("ListBackups() error = %v", err)
			}
			if len(backups) != tt.wantCount {
				t.Fatalf("备份数量不匹配: %d", len(backups))
			}
			if len(backups) > 1 && !backups[0].Timestamp.After(backups[1].Timestamp) {
				t.Fatalf("备份排序错误")
			}
		})
	}
}

func TestGetDefaultExportFilename(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"default filename"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := GetDefaultExportFilename()
			if !strings.HasPrefix(filename, "cc-switch-config-") {
				t.Fatalf("前缀不匹配: %s", filename)
			}
			if !strings.HasSuffix(filename, ".json") {
				t.Fatalf("后缀不匹配: %s", filename)
			}
			expectedDate := time.Now().In(shanghaiLocation).Format("2006-01-02")
			if !strings.Contains(filename, expectedDate) {
				t.Fatalf("日期不匹配: %s", filename)
			}
		})
	}
}

func TestCleanupManualBackups(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, BackupDirName)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("创建备份目录失败: %v", err)
	}

	createBackupFile(t, backupDir, backupPrefix+"old.json", time.Now().Add(-3*time.Hour))
	createBackupFile(t, backupDir, backupPrefix+"new.json", time.Now().Add(-1*time.Hour))
	createBackupFile(t, backupDir, "other.json", time.Now().Add(-2*time.Hour))

	tests := []struct {
		name      string
		retain    int
		wantCount int
	}{
		{name: "retain 1", retain: 1, wantCount: 1},
		{name: "retain 0", retain: 0, wantCount: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupManualBackups(backupDir, tt.retain)
			count := countManualBackups(t, backupDir)
			if count != tt.wantCount {
				t.Fatalf("备份数量不匹配: %d != %d", count, tt.wantCount)
			}
		})
	}
}

func writeConfigFile(t *testing.T, path string, cfg config.MultiAppConfig) []byte {
	t.Helper()
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("序列化配置失败: %v", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("写入配置失败: %v", err)
	}
	return data
}

func sampleConfig(providerID string) config.MultiAppConfig {
	return config.MultiAppConfig{
		Version: 2,
		Apps: map[string]config.ProviderManager{
			"claude": {
				Providers: map[string]config.Provider{
					providerID: {ID: providerID, Name: "Test " + providerID},
				},
				Current: providerID,
			},
		},
	}
}

func createBackupFile(t *testing.T, dir, name string, modTime time.Time) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
		t.Fatalf("创建备份失败: %v", err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatalf("设置时间失败: %v", err)
	}
}

func countManualBackups(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读取目录失败: %v", err)
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		if strings.HasPrefix(entry.Name(), backupPrefix) {
			count++
		}
	}
	return count
}

func supportsDevFull() bool {
	if runtime.GOOS == "windows" {
		return false
	}
	if _, err := os.Stat("/dev/full"); err != nil {
		return false
	}
	file, err := os.OpenFile("/dev/full", os.O_WRONLY, 0600)
	if err != nil {
		return false
	}
	file.Close()
	return true
}
