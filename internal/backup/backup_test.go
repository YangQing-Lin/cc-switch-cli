package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
)

func TestCreateBackup(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	testConfig := config.MultiAppConfig{
		Version: 2,
		Apps: map[string]config.ProviderManager{
			"claude": {
				Providers: map[string]config.Provider{
					"test": {
						ID:   "test",
						Name: "Test Provider",
					},
				},
				Current: "test",
			},
		},
	}

	data, err := json.Marshal(testConfig)
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	backupID, err := CreateBackup(configPath)
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	if backupID == "" {
		t.Fatal("Expected non-empty backup ID")
	}

	if strings.HasPrefix(backupID, AutoBackupPrefix) {
		t.Errorf("Manual backup should not have auto prefix, got: %s", backupID)
	}

	backupDir := filepath.Join(tmpDir, BackupDirName)
	backupPath := filepath.Join(backupDir, backupID+".json")

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Fatalf("Backup file not created: %s", backupPath)
	}

	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("Failed to read backup file: %v", err)
	}

	if string(backupData) != string(data) {
		t.Error("Backup content does not match original config")
	}
}

func TestCreateAutoBackup(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	testConfig := config.MultiAppConfig{
		Version: 2,
		Apps:    map[string]config.ProviderManager{},
	}

	data, err := json.Marshal(testConfig)
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	backupID, err := CreateAutoBackup(configPath)
	if err != nil {
		t.Fatalf("CreateAutoBackup failed: %v", err)
	}

	if backupID == "" {
		t.Fatal("Expected non-empty backup ID")
	}

	if !strings.HasPrefix(backupID, AutoBackupPrefix) {
		t.Errorf("Auto backup should have auto prefix, got: %s", backupID)
	}

	backupDir := filepath.Join(tmpDir, BackupDirName)
	backupPath := filepath.Join(backupDir, backupID+".json")

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Fatalf("Auto backup file not created: %s", backupPath)
	}
}

func TestCreateBackupNonExistentConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent.json")

	backupID, err := CreateBackup(configPath)
	if err != nil {
		t.Fatalf("CreateBackup should not error for non-existent config: %v", err)
	}

	if backupID != "" {
		t.Errorf("Expected empty backup ID for non-existent config, got: %s", backupID)
	}
}

func TestCleanupOldBackups(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, BackupDirName)

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup dir: %v", err)
	}

	for i := 0; i < 15; i++ {
		fileName := filepath.Join(backupDir, "backup_test_"+string(rune('A'+i))+".json")
		if err := os.WriteFile(fileName, []byte("{}"), 0600); err != nil {
			t.Fatalf("Failed to create test backup file: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := CleanupOldBackups(backupDir, MaxBackups); err != nil {
		t.Fatalf("CleanupOldBackups failed: %v", err)
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("Failed to read backup dir: %v", err)
	}

	jsonFiles := 0
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".json" {
			jsonFiles++
		}
	}

	if jsonFiles != MaxBackups {
		t.Errorf("Expected %d backup files, got %d", MaxBackups, jsonFiles)
	}
}

func TestExportConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	exportPath := filepath.Join(tmpDir, "export.json")

	testConfig := config.MultiAppConfig{
		Version: 2,
		Apps: map[string]config.ProviderManager{
			"claude": {
				Providers: map[string]config.Provider{
					"test": {
						ID:   "test",
						Name: "Test Provider",
					},
				},
				Current: "test",
			},
		},
	}

	data, err := json.Marshal(testConfig)
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	if err := ExportConfig(configPath, exportPath); err != nil {
		t.Fatalf("ExportConfig failed: %v", err)
	}

	exportData, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}

	if string(exportData) != string(data) {
		t.Error("Export content does not match original config")
	}
}

func TestImportConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	importPath := filepath.Join(tmpDir, "import.json")

	originalConfig := config.MultiAppConfig{
		Version: 2,
		Apps: map[string]config.ProviderManager{
			"claude": {
				Providers: map[string]config.Provider{
					"old": {
						ID:   "old",
						Name: "Old Provider",
					},
				},
				Current: "old",
			},
		},
	}

	newConfig := config.MultiAppConfig{
		Version: 2,
		Apps: map[string]config.ProviderManager{
			"claude": {
				Providers: map[string]config.Provider{
					"new": {
						ID:   "new",
						Name: "New Provider",
					},
				},
				Current: "new",
			},
		},
	}

	originalData, _ := json.Marshal(originalConfig)
	newData, _ := json.Marshal(newConfig)

	if err := os.WriteFile(configPath, originalData, 0600); err != nil {
		t.Fatalf("Failed to write original config: %v", err)
	}

	if err := os.WriteFile(importPath, newData, 0600); err != nil {
		t.Fatalf("Failed to write import file: %v", err)
	}

	backupID, err := ImportConfig(configPath, importPath)
	if err != nil {
		t.Fatalf("ImportConfig failed: %v", err)
	}

	if backupID == "" {
		t.Error("Expected non-empty backup ID after import")
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config after import: %v", err)
	}

	if string(configData) != string(newData) {
		t.Error("Config was not updated with imported data")
	}

	backupDir := filepath.Join(tmpDir, BackupDirName)
	backupPath := filepath.Join(backupDir, backupID+".json")
	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("Failed to read backup file: %v", err)
	}

	if string(backupData) != string(originalData) {
		t.Error("Backup does not contain original config")
	}
}

func TestListBackups(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, BackupDirName)

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup dir: %v", err)
	}

	expectedCount := 5
	for i := 0; i < expectedCount; i++ {
		fileName := filepath.Join(backupDir, "backup_test_"+string(rune('A'+i))+".json")
		if err := os.WriteFile(fileName, []byte("{}"), 0600); err != nil {
			t.Fatalf("Failed to create test backup file: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	backups, err := ListBackups(tmpDir)
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}

	if len(backups) != expectedCount {
		t.Errorf("Expected %d backups, got %d", expectedCount, len(backups))
	}

	for i := 0; i < len(backups)-1; i++ {
		if backups[i].Timestamp.Before(backups[i+1].Timestamp) {
			t.Error("Backups are not sorted by timestamp (newest first)")
			break
		}
	}
}

func TestRestoreBackup(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	backupDir := filepath.Join(tmpDir, BackupDirName)

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup dir: %v", err)
	}

	currentConfig := config.MultiAppConfig{
		Version: 2,
		Apps: map[string]config.ProviderManager{
			"claude": {
				Providers: map[string]config.Provider{
					"current": {
						ID:   "current",
						Name: "Current Provider",
					},
				},
				Current: "current",
			},
		},
	}

	oldConfig := config.MultiAppConfig{
		Version: 2,
		Apps: map[string]config.ProviderManager{
			"claude": {
				Providers: map[string]config.Provider{
					"old": {
						ID:   "old",
						Name: "Old Provider",
					},
				},
				Current: "old",
			},
		},
	}

	currentData, _ := json.Marshal(currentConfig)
	oldData, _ := json.Marshal(oldConfig)

	if err := os.WriteFile(configPath, currentData, 0600); err != nil {
		t.Fatalf("Failed to write current config: %v", err)
	}

	backupPath := filepath.Join(backupDir, "backup_20230101_120000.json")
	if err := os.WriteFile(backupPath, oldData, 0600); err != nil {
		t.Fatalf("Failed to write backup file: %v", err)
	}

	if err := RestoreBackup(configPath, backupPath); err != nil {
		t.Fatalf("RestoreBackup failed: %v", err)
	}

	restoredData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read restored config: %v", err)
	}

	if string(restoredData) != string(oldData) {
		t.Error("Config was not restored from backup")
	}
}

func TestGetDefaultExportFilename(t *testing.T) {
	filename := GetDefaultExportFilename()

	if !strings.HasPrefix(filename, "cc-switch-config-") {
		t.Errorf("Expected filename to start with 'cc-switch-config-', got: %s", filename)
	}

	if !strings.HasSuffix(filename, ".json") {
		t.Errorf("Expected filename to end with '.json', got: %s", filename)
	}

	expectedDate := time.Now().In(shanghaiLocation).Format("2006-01-02")
	if !strings.Contains(filename, expectedDate) {
		t.Errorf("Expected filename to contain date %s, got: %s", expectedDate, filename)
	}
}
