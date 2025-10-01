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
	"github.com/google/uuid"
)

// Manager 配置管理器（对应 cc-switch 的 AppState）
type Manager struct {
	config     *MultiAppConfig
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
		// 配置文件不存在，创建默认配置
		m.config = &MultiAppConfig{
			Version: 2,
			Claude: ProviderManager{
				Providers: make(map[string]Provider),
				Current:   "",
			},
			Codex: ProviderManager{
				Providers: make(map[string]Provider),
				Current:   "",
			},
		}
		return m.Save()
	}

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	m.config = &MultiAppConfig{}
	if err := json.Unmarshal(data, m.config); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 确保 map 已初始化
	if m.config.Claude.Providers == nil {
		m.config.Claude.Providers = make(map[string]Provider)
	}
	if m.config.Codex.Providers == nil {
		m.config.Codex.Providers = make(map[string]Provider)
	}

	return nil
}

// Save 保存配置文件（与 cc-switch 保持一致，先创建备份）
func (m *Manager) Save() error {
	// 确保目录存在
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 创建备份（如果文件已存在）
	if utils.FileExists(m.configPath) {
		backupPath := m.configPath + ".bak"
		data, err := os.ReadFile(m.configPath)
		if err == nil {
			_ = os.WriteFile(backupPath, data, 0600)
		}
	}

	return utils.WriteJSONFile(m.configPath, m.config, 0600)
}

// AddProvider 添加新的供应商配置（Claude）
func (m *Manager) AddProvider(name, apiToken, baseURL, category string) error {
	// 检查配置是否已存在（通过名称）
	for _, p := range m.config.Claude.Providers {
		if p.Name == name {
			return fmt.Errorf("配置 '%s' 已存在", name)
		}
	}

	// 生成唯一 ID
	id := uuid.New().String()

	// 创建 settingsConfig（完全匹配 cc-switch 格式）
	settingsConfig := map[string]interface{}{
		"env": map[string]interface{}{
			"ANTHROPIC_AUTH_TOKEN": apiToken,
			"ANTHROPIC_BASE_URL":   baseURL,
		},
	}

	// 创建 Provider
	provider := Provider{
		ID:             id,
		Name:           name,
		SettingsConfig: settingsConfig,
		WebsiteURL:     "",
		Category:       category,
		CreatedAt:      time.Now().UnixMilli(),
	}

	m.config.Claude.Providers[id] = provider

	// 如果是第一个配置，自动设置为当前配置
	if len(m.config.Claude.Providers) == 1 {
		m.config.Claude.Current = id
	}

	return m.Save()
}

// DeleteProvider 删除供应商配置
func (m *Manager) DeleteProvider(name string) error {
	// 找到对应的 provider
	var targetID string
	for id, p := range m.config.Claude.Providers {
		if p.Name == name {
			targetID = id
			break
		}
	}

	if targetID == "" {
		return fmt.Errorf("配置 '%s' 不存在", name)
	}

	// 不能删除当前激活的配置
	if m.config.Claude.Current == targetID {
		return fmt.Errorf("不能删除当前激活的配置，请先切换到其他配置")
	}

	// 删除配置
	delete(m.config.Claude.Providers, targetID)

	return m.Save()
}

// GetProvider 获取指定配置
func (m *Manager) GetProvider(name string) (*Provider, error) {
	for _, p := range m.config.Claude.Providers {
		if p.Name == name {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("配置 '%s' 不存在", name)
}

// ListProviders 列出所有供应商配置
func (m *Manager) ListProviders() []Provider {
	providers := make([]Provider, 0, len(m.config.Claude.Providers))
	for _, p := range m.config.Claude.Providers {
		providers = append(providers, p)
	}
	return providers
}

// GetCurrentProvider 获取当前激活的供应商
func (m *Manager) GetCurrentProvider() *Provider {
	if m.config.Claude.Current == "" {
		return nil
	}

	if p, ok := m.config.Claude.Providers[m.config.Claude.Current]; ok {
		return &p
	}

	return nil
}

// SwitchProvider 切换到指定供应商（实现 SSOT 三步流程）
func (m *Manager) SwitchProvider(name string) error {
	// 找到对应的 provider
	var targetID string
	var targetProvider *Provider
	for id, p := range m.config.Claude.Providers {
		if p.Name == name {
			targetID = id
			provider := p
			targetProvider = &provider
			break
		}
	}

	if targetID == "" {
		return fmt.Errorf("配置 '%s' 不存在", name)
	}

	// SSOT 步骤 1: 回填（Backfill）当前配置
	// 如果有当前激活的配置，将 live 配置回填到内存中的当前供应商
	if m.config.Claude.Current != "" {
		settingsPath, err := GetClaudeSettingsPath()
		if err == nil && utils.FileExists(settingsPath) {
			// 读取当前 live 配置
			data, err := os.ReadFile(settingsPath)
			if err == nil {
				var liveSettings ClaudeSettings
				if err := json.Unmarshal(data, &liveSettings); err == nil {
					// 回填到当前供应商
					if currentProvider, ok := m.config.Claude.Providers[m.config.Claude.Current]; ok {
						// 更新 settingsConfig
						if envMap, ok := currentProvider.SettingsConfig["env"].(map[string]interface{}); ok {
							envMap["ANTHROPIC_AUTH_TOKEN"] = liveSettings.Env.AnthropicAuthToken
							envMap["ANTHROPIC_BASE_URL"] = liveSettings.Env.AnthropicBaseURL
							if liveSettings.Env.ClaudeCodeModel != "" {
								envMap["CLAUDE_CODE_MODEL"] = liveSettings.Env.ClaudeCodeModel
							}
							if liveSettings.Env.ClaudeCodeMaxTokens != "" {
								envMap["CLAUDE_CODE_MAX_TOKENS"] = liveSettings.Env.ClaudeCodeMaxTokens
							}
						}
						m.config.Claude.Providers[m.config.Claude.Current] = currentProvider
					}
				}
			}
		}
	}

	// SSOT 步骤 2: 切换（Switch）写入目标配置
	settingsPath, err := GetClaudeSettingsPath()
	if err != nil {
		return fmt.Errorf("获取 Claude 设置文件路径失败: %w", err)
	}

	// 备份当前设置
	if err := utils.BackupFile(settingsPath); err != nil {
		return fmt.Errorf("备份设置文件失败: %w", err)
	}

	// 读取现有设置（保留其他字段）
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

	// 从 settingsConfig 提取 env 并写入
	if envMap, ok := targetProvider.SettingsConfig["env"].(map[string]interface{}); ok {
		settings.Env = ClaudeEnv{}
		if token, ok := envMap["ANTHROPIC_AUTH_TOKEN"].(string); ok {
			settings.Env.AnthropicAuthToken = token
		}
		if baseURL, ok := envMap["ANTHROPIC_BASE_URL"].(string); ok {
			settings.Env.AnthropicBaseURL = baseURL
		}
		if model, ok := envMap["CLAUDE_CODE_MODEL"].(string); ok {
			settings.Env.ClaudeCodeModel = model
		}
		if maxTokens, ok := envMap["CLAUDE_CODE_MAX_TOKENS"].(string); ok {
			settings.Env.ClaudeCodeMaxTokens = maxTokens
		}
	}

	// 保存设置
	if err := utils.WriteJSONFile(settingsPath, settings, 0644); err != nil {
		return fmt.Errorf("保存设置文件失败: %w", err)
	}

	// SSOT 步骤 3: 持久化（Persist）更新当前供应商 ID
	m.config.Claude.Current = targetID
	return m.Save()
}

// ValidateProvider 验证供应商配置的有效性
func ValidateProvider(name, apiToken, baseURL string) error {
	if name == "" {
		return fmt.Errorf("配置名称不能为空")
	}

	if apiToken == "" {
		return fmt.Errorf("API Token 不能为空")
	}

	// 验证 Token 格式（支持 Anthropic 和第三方格式）
	if !strings.HasPrefix(apiToken, "sk-") && !strings.HasPrefix(apiToken, "88_") {
		return fmt.Errorf("API Token 格式错误，应以 'sk-' 或 '88_' 开头")
	}

	if baseURL == "" {
		return fmt.Errorf("Base URL 不能为空")
	}

	// 验证 URL 格式
	if _, err := url.Parse(baseURL); err != nil {
		return fmt.Errorf("无效的 Base URL: %w", err)
	}

	return nil
}

// ValidateMaxTokens 验证 MaxTokens 是否为有效数字
func ValidateMaxTokens(maxTokens string) error {
	if maxTokens == "" {
		return nil
	}
	if _, err := strconv.Atoi(maxTokens); err != nil {
		return fmt.Errorf("Max Tokens 必须是数字")
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

// GetConfigPath 获取配置文件路径（与 cc-switch 一致）
func GetConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}

	return filepath.Join(home, ".cc-switch", "config.json"), nil
}

// GetClaudeSettingsPath 获取 Claude 设置文件路径
func GetClaudeSettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}

	dir := filepath.Join(home, ".claude")
	settingsPath := filepath.Join(dir, "settings.json")

	// 优先使用新版文件名
	if utils.FileExists(settingsPath) {
		return settingsPath, nil
	}

	// 兼容旧版文件名 claude.json
	legacyPath := filepath.Join(dir, "claude.json")
	if utils.FileExists(legacyPath) {
		return legacyPath, nil
	}

	// 默认使用 settings.json
	return settingsPath, nil
}

// ExtractTokenFromProvider 从 Provider 提取 API Token
func ExtractTokenFromProvider(p *Provider) string {
	if p == nil {
		return ""
	}
	if envMap, ok := p.SettingsConfig["env"].(map[string]interface{}); ok {
		if token, ok := envMap["ANTHROPIC_AUTH_TOKEN"].(string); ok {
			return token
		}
	}
	return ""
}

// ExtractBaseURLFromProvider 从 Provider 提取 Base URL
func ExtractBaseURLFromProvider(p *Provider) string {
	if p == nil {
		return ""
	}
	if envMap, ok := p.SettingsConfig["env"].(map[string]interface{}); ok {
		if baseURL, ok := envMap["ANTHROPIC_BASE_URL"].(string); ok {
			return baseURL
		}
	}
	return ""
}
