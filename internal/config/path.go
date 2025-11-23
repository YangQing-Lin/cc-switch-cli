package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/YangQing-Lin/cc-switch-cli/internal/portable"
	"github.com/YangQing-Lin/cc-switch-cli/internal/utils"
)

func GetConfigPath() (string, error) {
	if portable.IsPortableMode() {
		configDir, err := portable.GetPortableConfigDir()
		if err != nil {
			return "", fmt.Errorf("获取便携版配置目录失败: %w", err)
		}
		return filepath.Join(configDir, "config.json"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}

	return filepath.Join(home, ".cc-switch", "config.json"), nil
}

func GetClaudeSettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}

	dir := filepath.Join(home, ".claude")
	settingsPath := filepath.Join(dir, "settings.json")

	if utils.FileExists(settingsPath) {
		return settingsPath, nil
	}

	legacyPath := filepath.Join(dir, "claude.json")
	if utils.FileExists(legacyPath) {
		return legacyPath, nil
	}

	return settingsPath, nil
}

func (m *Manager) GetClaudeSettingsPathWithDir() (string, error) {
	if m.customDir == "" {
		return GetClaudeSettingsPath()
	}

	dir := filepath.Join(m.customDir, ".claude")
	settingsPath := filepath.Join(dir, "settings.json")
	return settingsPath, nil
}

func GetCodexConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}
	return filepath.Join(home, ".codex", "config.toml"), nil
}

func GetCodexAuthJsonPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}
	return filepath.Join(home, ".codex", "auth.json"), nil
}

func (m *Manager) GetCodexConfigPathWithDir() (string, error) {
	if m.customDir == "" {
		return GetCodexConfigPath()
	}
	return filepath.Join(m.customDir, ".codex", "config.toml"), nil
}

func (m *Manager) GetCodexAuthJsonPathWithDir() (string, error) {
	if m.customDir == "" {
		return GetCodexAuthJsonPath()
	}
	return filepath.Join(m.customDir, ".codex", "auth.json"), nil
}

func (m *Manager) GetConfigPath() string {
	return m.configPath
}

// GetGeminiDir returns the Gemini configuration directory
func GetGeminiDir() (string, error) {
	if portable.IsPortableMode() {
		configDir, err := portable.GetPortableConfigDir()
		if err != nil {
			return "", fmt.Errorf("获取便携版配置目录失败: %w", err)
		}
		return filepath.Join(configDir, ".gemini"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}

	return filepath.Join(home, ".gemini"), nil
}

// GetGeminiEnvPath returns the Gemini .env file path
func GetGeminiEnvPath() (string, error) {
	dir, err := GetGeminiDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ".env"), nil
}

// GetGeminiSettingsPath returns the Gemini settings.json file path
func GetGeminiSettingsPath() (string, error) {
	dir, err := GetGeminiDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.json"), nil
}

// GetGeminiEnvPathWithDir returns the Gemini .env file path with custom directory support
func (m *Manager) GetGeminiEnvPathWithDir() (string, error) {
	if m.customDir == "" {
		return GetGeminiEnvPath()
	}
	return filepath.Join(m.customDir, ".gemini", ".env"), nil
}

// GetGeminiSettingsPathWithDir returns the Gemini settings.json path with custom directory support
func (m *Manager) GetGeminiSettingsPathWithDir() (string, error) {
	if m.customDir == "" {
		return GetGeminiSettingsPath()
	}
	return filepath.Join(m.customDir, ".gemini", "settings.json"), nil
}
