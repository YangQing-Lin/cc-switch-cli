package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/YangQing-Lin/cc-switch-cli/internal/utils"
)

// Manager 配置管理器
type Manager struct {
	store      *ConfigStore
	configPath string
}

// NewManager 创建配置管理器
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

// Load 加载配置文件
func (m *Manager) Load() error {
	if !utils.FileExists(m.configPath) {
		// 配置文件不存在，创建空配置
		m.store = &ConfigStore{
			Configs:   []Config{},
			UpdatedAt: time.Now(),
		}
		return m.Save()
	}

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	m.store = &ConfigStore{}
	if err := json.Unmarshal(data, m.store); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	return nil
}

// Save 保存配置文件
func (m *Manager) Save() error {
	m.store.UpdatedAt = time.Now()

	// 确保目录存在
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	return utils.WriteJSONFile(m.configPath, m.store, 0600)
}

// AddConfig 添加新配置
func (m *Manager) AddConfig(config Config) error {
	// 检查配置是否已存在
	for _, c := range m.store.Configs {
		if c.Name == config.Name {
			return fmt.Errorf("配置 '%s' 已存在", config.Name)
		}
	}

	// 设置时间戳
	now := time.Now()
	config.CreatedAt = now
	config.UpdatedAt = now

	m.store.Configs = append(m.store.Configs, config)

	// 如果是第一个配置，自动设置为当前配置
	if len(m.store.Configs) == 1 {
		m.store.CurrentConfig = config.Name
	}

	return m.Save()
}

// DeleteConfig 删除配置
func (m *Manager) DeleteConfig(name string) error {
	index := -1
	for i, c := range m.store.Configs {
		if c.Name == name {
			index = i
			break
		}
	}

	if index == -1 {
		return fmt.Errorf("配置 '%s' 不存在", name)
	}

	// 删除配置
	m.store.Configs = append(m.store.Configs[:index], m.store.Configs[index+1:]...)

	// 如果删除的是当前配置，清除当前配置
	if m.store.CurrentConfig == name {
		m.store.CurrentConfig = ""
	}

	return m.Save()
}

// GetConfig 获取指定配置
func (m *Manager) GetConfig(name string) (*Config, error) {
	for _, c := range m.store.Configs {
		if c.Name == name {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("配置 '%s' 不存在", name)
}

// ListConfigs 列出所有配置
func (m *Manager) ListConfigs() []Config {
	return m.store.Configs
}

// GetCurrentConfig 获取当前激活的配置
func (m *Manager) GetCurrentConfig() *Config {
	if m.store.CurrentConfig == "" {
		return nil
	}

	for _, c := range m.store.Configs {
		if c.Name == m.store.CurrentConfig {
			return &c
		}
	}

	return nil
}

// SwitchConfig 切换到指定配置
func (m *Manager) SwitchConfig(name string) error {
	cfg, err := m.GetConfig(name)
	if err != nil {
		return err
	}

	// 获取 Claude 设置文件路径
	settingsPath, err := GetClaudeSettingsPath()
	if err != nil {
		return fmt.Errorf("获取 Claude 设置文件路径失败: %w", err)
	}

	// 备份当前设置
	if err := utils.BackupFile(settingsPath); err != nil {
		return fmt.Errorf("备份设置文件失败: %w", err)
	}

	// 读取现有设置（如果存在）
	settings := &ClaudeSettings{
		Permissions: ClaudePermissions{
			Allow: []string{},
			Deny:  []string{},
		},
	}

	if utils.FileExists(settingsPath) {
		data, err := os.ReadFile(settingsPath)
		if err != nil {
			return fmt.Errorf("读取设置文件失败: %w", err)
		}

		if err := json.Unmarshal(data, settings); err != nil {
			// 如果解析失败，使用默认设置
			settings = &ClaudeSettings{
				Permissions: ClaudePermissions{
					Allow: []string{},
					Deny:  []string{},
				},
			}
		}
	}

	// 更新环境变量配置
	settings.Env = ClaudeEnv{
		AnthropicAuthToken:  cfg.AnthropicAuthToken,
		AnthropicBaseURL:    cfg.AnthropicBaseURL,
		ClaudeCodeModel:     cfg.ClaudeCodeModel,
		ClaudeCodeMaxTokens: cfg.ClaudeCodeMaxTokens,
	}

	// 保存设置
	if err := utils.WriteJSONFile(settingsPath, settings, 0644); err != nil {
		return fmt.Errorf("保存设置文件失败: %w", err)
	}

	// 更新当前配置
	m.store.CurrentConfig = name
	return m.Save()
}

// Validate 验证配置的有效性
func (c *Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("配置名称不能为空")
	}

	if c.AnthropicAuthToken == "" {
		return fmt.Errorf("API Token 不能为空")
	}

	// 验证 Token 格式
	if !strings.HasPrefix(c.AnthropicAuthToken, "sk-") {
		return fmt.Errorf("API Token 格式错误，应以 'sk-' 开头")
	}

	if c.AnthropicBaseURL == "" {
		return fmt.Errorf("Base URL 不能为空")
	}

	// 验证 URL 格式
	if _, err := url.Parse(c.AnthropicBaseURL); err != nil {
		return fmt.Errorf("无效的 Base URL: %w", err)
	}

	// 验证 MaxTokens 是否为数字
	if c.ClaudeCodeMaxTokens != "" {
		if _, err := strconv.Atoi(c.ClaudeCodeMaxTokens); err != nil {
			return fmt.Errorf("Max Tokens 必须是数字")
		}
	}

	return nil
}

// MaskToken 脱敏显示 Token
func MaskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

// GetConfigPath 获取配置文件路径
func GetConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}

	return filepath.Join(home, ".cc-switch", "configs.json"), nil
}

// GetClaudeSettingsPath 获取 Claude 设置文件路径
func GetClaudeSettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}

	return filepath.Join(home, ".claude", "settings.json"), nil
}
