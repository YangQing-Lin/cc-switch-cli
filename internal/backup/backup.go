package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/i18n"
)

const (
	// MaxBackups is the maximum number of backups to keep
	MaxBackups = 10
	// MaxAutoBackups is the maximum number of auto backups to keep
	MaxAutoBackups = 5
	// BackupDirName is the name of the backup directory
	BackupDirName = "backups"
	// AutoBackupPrefix is the prefix for auto backup files
	AutoBackupPrefix = "auto_"
)

// BackupInfo contains information about a backup file
type BackupInfo struct {
	Path      string
	Timestamp time.Time
	Size      int64
}

// CreateBackup creates a timestamped backup of the current config file
// Returns the backup ID (timestamp portion) or empty string if source doesn't exist
func CreateBackup(configPath string) (string, error) {
	return createBackup(configPath, false)
}

// CreateAutoBackup creates an automatic backup (called by Save())
// Returns the backup ID or empty string if source doesn't exist
func CreateAutoBackup(configPath string) (string, error) {
	return createBackup(configPath, true)
}

// createBackup internal function to create backup (manual or auto)
func createBackup(configPath string, isAuto bool) (string, error) {
	// Check if source config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return "", nil
	}

	// Generate timestamp and backup ID
	timestamp := time.Now().UTC().Format("20060102_150405")
	var backupID string
	if isAuto {
		backupID = fmt.Sprintf("%sbackup_%s", AutoBackupPrefix, timestamp)
	} else {
		backupID = fmt.Sprintf("backup_%s", timestamp)
	}

	// Create backup directory
	configDir := filepath.Dir(configPath)
	backupDir := filepath.Join(configDir, BackupDirName)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create backup file path
	backupPath := filepath.Join(backupDir, backupID+".json")

	// Read source config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read config file: %w", err)
	}

	// Write backup file
	if err := os.WriteFile(backupPath, data, 0600); err != nil {
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}

	// Cleanup old backups (different limits for auto vs manual)
	if isAuto {
		cleanupAutoBackups(backupDir, MaxAutoBackups)
	} else {
		cleanupManualBackups(backupDir, MaxBackups)
	}

	return backupID, nil
}

// cleanupAutoBackups removes old auto backup files
func cleanupAutoBackups(backupDir string, retain int) {
	cleanupBackupsByPrefix(backupDir, AutoBackupPrefix, retain)
}

// cleanupManualBackups removes old manual backup files
func cleanupManualBackups(backupDir string, retain int) {
	cleanupBackupsByPrefix(backupDir, "backup_", retain)
}

// cleanupBackupsByPrefix removes old backup files with given prefix
func cleanupBackupsByPrefix(backupDir, prefix string, retain int) {
	if retain == 0 {
		return
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		// Only process files with matching prefix
		if !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}

		fullPath := filepath.Join(backupDir, entry.Name())
		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}

		backups = append(backups, BackupInfo{
			Path:      fullPath,
			Timestamp: info.ModTime(),
			Size:      info.Size(),
		})
	}

	if len(backups) <= retain {
		return
	}

	// Sort by time (oldest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.Before(backups[j].Timestamp)
	})

	// Delete oldest
	for i := 0; i < len(backups)-retain; i++ {
		os.Remove(backups[i].Path)
	}
}

// CleanupOldBackups removes old backup files, keeping only the most recent 'retain' files
func CleanupOldBackups(backupDir string, retain int) error {
	if retain == 0 {
		return nil // Skip cleanup if retain is 0
	}

	// Check if directory exists
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return nil // Nothing to clean up
	}

	// Read all files in backup directory
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	// Collect backup files with their modification times
	var backups []BackupInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process .json files
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		fullPath := filepath.Join(backupDir, entry.Name())
		info, err := os.Stat(fullPath)
		if err != nil {
			continue // Skip files we can't stat
		}

		backups = append(backups, BackupInfo{
			Path:      fullPath,
			Timestamp: info.ModTime(),
			Size:      info.Size(),
		})
	}

	// Check if cleanup is needed
	if len(backups) <= retain {
		return nil // No cleanup needed
	}

	// Sort by modification time (oldest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.Before(backups[j].Timestamp)
	})

	// Calculate how many to remove
	removeCount := len(backups) - retain

	// Delete oldest files
	for i := 0; i < removeCount; i++ {
		if err := os.Remove(backups[i].Path); err != nil {
			// Log warning but continue
			fmt.Fprintf(os.Stderr, "Warning: failed to delete old backup %s: %v\n",
				backups[i].Path, err)
		}
	}

	return nil
}

// ExportConfig exports the current configuration to a file
func ExportConfig(configPath, outputPath string) error {
	// Read current config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Validate it's valid JSON
	var config config.MultiAppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("config file is corrupted: %w", err)
	}

	// Write to output file
	if err := os.WriteFile(outputPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	return nil
}

// ImportConfig imports configuration from a file with automatic backup
// Returns the backup ID created before import
func ImportConfig(configPath, importPath string) (string, error) {
	// Read import file
	importData, err := os.ReadFile(importPath)
	if err != nil {
		return "", fmt.Errorf("failed to read import file: %w", err)
	}

	// Validate configuration structure
	var newConfig config.MultiAppConfig
	if err := json.Unmarshal(importData, &newConfig); err != nil {
		return "", fmt.Errorf("invalid configuration file: %w", err)
	}

	// Create automatic backup before import
	backupID, err := CreateBackup(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	// Write new configuration
	if err := os.WriteFile(configPath, importData, 0600); err != nil {
		return backupID, fmt.Errorf("failed to write configuration: %w", err)
	}

	return backupID, nil
}

// ListBackups returns a list of all backup files sorted by timestamp (newest first)
func ListBackups(configDir string) ([]BackupInfo, error) {
	backupDir := filepath.Join(configDir, BackupDirName)

	// Check if directory exists
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return []BackupInfo{}, nil // Return empty list if no backups exist
	}

	// Read all files in backup directory
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	// Collect backup files
	var backups []BackupInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process .json files
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		fullPath := filepath.Join(backupDir, entry.Name())
		info, err := os.Stat(fullPath)
		if err != nil {
			continue // Skip files we can't stat
		}

		backups = append(backups, BackupInfo{
			Path:      fullPath,
			Timestamp: info.ModTime(),
			Size:      info.Size(),
		})
	}

	// Sort by modification time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

// RestoreBackup restores configuration from a backup file
func RestoreBackup(configPath, backupPath string) error {
	// Read backup file
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Validate configuration structure
	var config config.MultiAppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("backup file is corrupted: %w", err)
	}

	// Create a backup of current config before restoring
	backupID, err := CreateBackup(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to backup current config: %v\n", err)
	} else if backupID != "" {
		fmt.Printf("%s: %s\n", i18n.T("backup.created"), backupID)
	}

	// Write restored configuration
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write configuration: %w", err)
	}

	return nil
}

// GetDefaultExportFilename returns a default filename for config export
func GetDefaultExportFilename() string {
	date := time.Now().Format("2006-01-02")
	return fmt.Sprintf("cc-switch-config-%s.json", date)
}
