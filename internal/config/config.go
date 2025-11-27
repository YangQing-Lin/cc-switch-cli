package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/YangQing-Lin/cc-switch-cli/internal/utils"
)

type Manager struct {
	config     *MultiAppConfig
	configPath string
	customDir  string
}

func NewManager() (*Manager, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, fmt.Errorf("获取配置文件路径失败: %w", err)
	}

	manager := &Manager{
		configPath: configPath,
	}

	if err := manager.Load(); err != nil {
		return nil, err
	}

	return manager, nil
}

func NewManagerWithDir(customDir string) (*Manager, error) {
	if customDir == "" {
		return NewManager()
	}

	if err := os.MkdirAll(customDir, 0755); err != nil {
		return nil, fmt.Errorf("创建自定义目录失败: %w", err)
	}

	configPath := filepath.Join(customDir, "config.json")
	manager := &Manager{
		configPath: configPath,
		customDir:  customDir,
	}

	if err := manager.Load(); err != nil {
		return nil, err
	}

	return manager, nil
}

func (m *Manager) Load() error {
	if !utils.FileExists(m.configPath) {
		m.createDefaultConfig()
		return nil
	}

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	if m.isEmptyConfig(data) {
		return m.handleEmptyConfig()
	}

	return m.loadAndMigrate(data)
}

func (m *Manager) createDefaultConfig() {
	m.config = &MultiAppConfig{
		Version: 2,
		Apps: map[string]ProviderManager{
			"claude": {Providers: make(map[string]Provider), Current: ""},
			"codex":  {Providers: make(map[string]Provider), Current: ""},
			"gemini": {Providers: make(map[string]Provider), Current: ""},
		},
	}
}

func (m *Manager) Save() error {
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	return utils.WriteJSONFile(m.configPath, m.config, 0600)
}

// GetViewMode 获取视图模式偏好
func (m *Manager) GetViewMode() string {
	if m.config.Preferences == nil || m.config.Preferences.ViewMode == "" {
		return "single"
	}
	return m.config.Preferences.ViewMode
}

// SetViewMode 设置视图模式偏好
func (m *Manager) SetViewMode(mode string) error {
	if m.config.Preferences == nil {
		m.config.Preferences = &UserPreferences{}
	}
	m.config.Preferences.ViewMode = mode
	return m.Save()
}
