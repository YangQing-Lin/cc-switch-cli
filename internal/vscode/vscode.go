package vscode

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// VsCodeApp represents a VS Code-like application
type VsCodeApp struct {
	Name        string
	ProcessName string
	ConfigPath  string
}

// SupportedApps lists all supported VS Code-like applications
var SupportedApps = []VsCodeApp{
	{
		Name:        "VS Code",
		ProcessName: "code",
		ConfigPath:  getVsCodeConfigPath(),
	},
	{
		Name:        "Cursor",
		ProcessName: "cursor",
		ConfigPath:  getCursorConfigPath(),
	},
}

// getVsCodeConfigPath returns the VS Code configuration path
func getVsCodeConfigPath() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(home, "AppData", "Roaming", "Code", "User")
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Code", "User")
	default:
		return filepath.Join(home, ".config", "Code", "User")
	}
}

// getCursorConfigPath returns the Cursor configuration path
func getCursorConfigPath() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(home, "AppData", "Roaming", "Cursor", "User")
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Cursor", "User")
	default:
		return filepath.Join(home, ".config", "Cursor", "User")
	}
}

// IsRunning checks if a VS Code-like app is running
func IsRunning(app VsCodeApp) (bool, error) {
	switch runtime.GOOS {
	case "windows":
		return isRunningWindows(app.ProcessName)
	case "darwin":
		return isRunningMacOS(app.ProcessName)
	default:
		return isRunningLinux(app.ProcessName)
	}
}

// isRunningWindows checks if process is running on Windows
func isRunningWindows(processName string) (bool, error) {
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s.exe", processName))
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.Contains(string(output), processName), nil
}

// isRunningMacOS checks if process is running on macOS
func isRunningMacOS(processName string) (bool, error) {
	cmd := exec.Command("pgrep", "-x", processName)
	err := cmd.Run()
	return err == nil, nil
}

// isRunningLinux checks if process is running on Linux
func isRunningLinux(processName string) (bool, error) {
	cmd := exec.Command("pgrep", "-x", processName)
	err := cmd.Run()
	return err == nil, nil
}

// GetRunningApps returns a list of running VS Code-like applications
func GetRunningApps() []VsCodeApp {
	var running []VsCodeApp
	for _, app := range SupportedApps {
		if isRunning, _ := IsRunning(app); isRunning {
			running = append(running, app)
		}
	}
	return running
}

// ConfigExists checks if configuration exists for the app
func ConfigExists(app VsCodeApp) bool {
	settingsPath := filepath.Join(app.ConfigPath, "settings.json")
	_, err := os.Stat(settingsPath)
	return err == nil
}

// GetSettingsPath returns the settings.json path for the app
func GetSettingsPath(app VsCodeApp) string {
	return filepath.Join(app.ConfigPath, "settings.json")
}

// BackupSettings backs up the current settings.json
func BackupSettings(app VsCodeApp) error {
	settingsPath := GetSettingsPath(app)
	if _, err := os.Stat(settingsPath); err != nil {
		return nil // No settings to backup
	}

	backupPath := settingsPath + ".backup"
	input, err := os.ReadFile(settingsPath)
	if err != nil {
		return err
	}

	return os.WriteFile(backupPath, input, 0644)
}

// RestoreSettings restores settings.json from backup
func RestoreSettings(app VsCodeApp) error {
	settingsPath := GetSettingsPath(app)
	backupPath := settingsPath + ".backup"

	input, err := os.ReadFile(backupPath)
	if err != nil {
		return err
	}

	return os.WriteFile(settingsPath, input, 0644)
}