package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/YangQing-Lin/cc-switch-cli/internal/utils"
)

// AppSettings 应用设置
type AppSettings struct {
	Language  string `json:"language"`  // 语言: "en" 或 "zh"
	ConfigDir string `json:"configDir"` // 自定义配置目录
}

// Manager 设置管理器
type Manager struct {
	settings     *AppSettings
	settingsPath string
}

// NewManager 创建设置管理器
func NewManager() (*Manager, error) {
	settingsPath, err := GetSettingsPath()
	if err != nil {
		return nil, fmt.Errorf("获取设置文件路径失败: %w", err)
	}

	manager := &Manager{
		settingsPath: settingsPath,
	}

	if err := manager.Load(); err != nil {
		return nil, err
	}

	return manager, nil
}

// GetSettingsPath 获取设置文件路径
func GetSettingsPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}

	return filepath.Join(homeDir, ".cc-switch", "settings.json"), nil
}

// Load 加载设置文件
func (m *Manager) Load() error {
	// 如果设置文件不存在，创建默认设置
	if !utils.FileExists(m.settingsPath) {
		m.settings = &AppSettings{
			Language:  "zh", // 默认中文
			ConfigDir: "",
		}
		return m.Save()
	}

	data, err := os.ReadFile(m.settingsPath)
	if err != nil {
		return fmt.Errorf("读取设置文件失败: %w", err)
	}

	m.settings = &AppSettings{}
	if err := json.Unmarshal(data, m.settings); err != nil {
		return fmt.Errorf("解析设置文件失败: %w", err)
	}

	return nil
}

// Save 保存设置文件
func (m *Manager) Save() error {
	// 确保目录存在
	dir := filepath.Dir(m.settingsPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建设置目录失败: %w", err)
	}

	return utils.WriteJSONFile(m.settingsPath, m.settings, 0600)
}

// GetLanguage 获取语言设置
func (m *Manager) GetLanguage() string {
	return m.settings.Language
}

// SetLanguage 设置语言
func (m *Manager) SetLanguage(language string) error {
	if language != "en" && language != "zh" {
		return fmt.Errorf("不支持的语言: %s (支持: en, zh)", language)
	}
	m.settings.Language = language
	return m.Save()
}

// GetConfigDir 获取自定义配置目录
func (m *Manager) GetConfigDir() string {
	return m.settings.ConfigDir
}

// SetConfigDir 设置自定义配置目录
func (m *Manager) SetConfigDir(configDir string) error {
	m.settings.ConfigDir = configDir
	return m.Save()
}

// Get 获取所有设置
func (m *Manager) Get() *AppSettings {
	return m.settings
}
