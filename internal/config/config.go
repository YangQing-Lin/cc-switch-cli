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
			Apps: map[string]ProviderManager{
				"claude": {
					Providers: make(map[string]Provider),
					Current:   "",
				},
				"codex": {
					Providers: make(map[string]Provider),
					Current:   "",
				},
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

	// 确保 Apps map 已初始化
	if m.config.Apps == nil {
		m.config.Apps = make(map[string]ProviderManager)
	}

	// 确保每个 app 的 Providers map 已初始化
	for appName, app := range m.config.Apps {
		if app.Providers == nil {
			app.Providers = make(map[string]Provider)
			m.config.Apps[appName] = app
		}
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

// AddProvider 添加新的供应商配置（默认为 Claude）
func (m *Manager) AddProvider(name, apiToken, baseURL, category string) error {
	return m.AddProviderForApp("claude", name, apiToken, baseURL, category)
}

// AddProviderForApp 为指定应用添加供应商配置
func (m *Manager) AddProviderForApp(appName, name, apiToken, baseURL, category string) error {
	// 确保应用存在
	if _, exists := m.config.Apps[appName]; !exists {
		m.config.Apps[appName] = ProviderManager{
			Providers: make(map[string]Provider),
			Current:   "",
		}
	}

	app := m.config.Apps[appName]

	// 检查配置是否已存在（通过名称）
	for _, p := range app.Providers {
		if p.Name == name {
			return fmt.Errorf("配置 '%s' 已存在", name)
		}
	}

	// 生成唯一 ID
	id := uuid.New().String()

	// 创建 settingsConfig（根据应用类型）
	var settingsConfig map[string]interface{}

	switch appName {
	case "claude":
		settingsConfig = map[string]interface{}{
			"env": map[string]interface{}{
				"ANTHROPIC_AUTH_TOKEN": apiToken,
				"ANTHROPIC_BASE_URL":   baseURL,
			},
		}
	case "codex":
		// Codex 配置格式
		settingsConfig = map[string]interface{}{
			"auth": apiToken,
			"base_url": baseURL,
		}
	default:
		return fmt.Errorf("不支持的应用: %s", appName)
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

	app.Providers[id] = provider

	// 如果是第一个配置，自动设置为当前配置
	if len(app.Providers) == 1 {
		app.Current = id
	}

	m.config.Apps[appName] = app
	return m.Save()
}

// DeleteProvider 删除供应商配置（默认为 Claude）
func (m *Manager) DeleteProvider(name string) error {
	return m.DeleteProviderForApp("claude", name)
}

// DeleteProviderForApp 删除指定应用的供应商配置
func (m *Manager) DeleteProviderForApp(appName, name string) error {
	app, exists := m.config.Apps[appName]
	if !exists {
		return fmt.Errorf("应用 '%s' 不存在", appName)
	}

	// 找到对应的 provider
	var targetID string
	for id, p := range app.Providers {
		if p.Name == name {
			targetID = id
			break
		}
	}

	if targetID == "" {
		return fmt.Errorf("配置 '%s' 不存在", name)
	}

	// 不能删除当前激活的配置
	if app.Current == targetID {
		return fmt.Errorf("不能删除当前激活的配置，请先切换到其他配置")
	}

	// 删除配置
	delete(app.Providers, targetID)
	m.config.Apps[appName] = app

	return m.Save()
}

// GetProvider 获取指定配置（默认为 Claude）
func (m *Manager) GetProvider(name string) (*Provider, error) {
	return m.GetProviderForApp("claude", name)
}

// GetProviderForApp 获取指定应用的指定配置
func (m *Manager) GetProviderForApp(appName, name string) (*Provider, error) {
	app, exists := m.config.Apps[appName]
	if !exists {
		return nil, fmt.Errorf("应用 '%s' 不存在", appName)
	}

	for _, p := range app.Providers {
		if p.Name == name {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("配置 '%s' 不存在", name)
}

// ListProviders 列出所有供应商配置（默认为 Claude）
func (m *Manager) ListProviders() []Provider {
	return m.ListProvidersForApp("claude")
}

// ListProvidersForApp 列出指定应用的所有供应商配置
func (m *Manager) ListProvidersForApp(appName string) []Provider {
	app, exists := m.config.Apps[appName]
	if !exists {
		return []Provider{}
	}

	providers := make([]Provider, 0, len(app.Providers))
	for _, p := range app.Providers {
		providers = append(providers, p)
	}
	return providers
}

// GetCurrentProvider 获取当前激活的供应商（默认为 Claude）
func (m *Manager) GetCurrentProvider() *Provider {
	return m.GetCurrentProviderForApp("claude")
}

// GetCurrentProviderForApp 获取指定应用的当前激活供应商
func (m *Manager) GetCurrentProviderForApp(appName string) *Provider {
	app, exists := m.config.Apps[appName]
	if !exists || app.Current == "" {
		return nil
	}

	if p, ok := app.Providers[app.Current]; ok {
		return &p
	}

	return nil
}

// SwitchProvider 切换到指定供应商（默认为 Claude）
func (m *Manager) SwitchProvider(name string) error {
	return m.SwitchProviderForApp("claude", name)
}

// SwitchProviderForApp 切换指定应用的供应商（实现 SSOT 三步流程）
func (m *Manager) SwitchProviderForApp(appName, name string) error {
	app, exists := m.config.Apps[appName]
	if !exists {
		return fmt.Errorf("应用 '%s' 不存在", appName)
	}

	// 找到对应的 provider
	var targetID string
	var targetProvider *Provider
	for id, p := range app.Providers {
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
	if err := m.backfillCurrentConfig(appName); err != nil {
		// 回填失败不应阻止切换，只记录警告
		fmt.Printf("警告: 回填当前配置失败: %v\n", err)
	}

	// SSOT 步骤 2: 切换（Switch）写入目标配置
	if err := m.writeProviderConfig(appName, targetProvider); err != nil {
		return fmt.Errorf("写入配置失败: %w", err)
	}

	// SSOT 步骤 3: 持久化（Persist）更新当前供应商 ID
	app.Current = targetID
	m.config.Apps[appName] = app
	return m.Save()
}

// backfillCurrentConfig 回填当前 live 配置到内存
func (m *Manager) backfillCurrentConfig(appName string) error {
	app, exists := m.config.Apps[appName]
	if !exists || app.Current == "" {
		return nil // 没有当前配置，无需回填
	}

	switch appName {
	case "claude":
		return m.backfillClaudeConfig()
	case "codex":
		return m.backfillCodexConfig()
	default:
		return fmt.Errorf("不支持的应用: %s", appName)
	}
}

// backfillClaudeConfig 回填 Claude 的 live 配置
func (m *Manager) backfillClaudeConfig() error {
	app := m.config.Apps["claude"]
	if app.Current == "" {
		return nil
	}

	settingsPath, err := GetClaudeSettingsPath()
	if err != nil || !utils.FileExists(settingsPath) {
		return nil // 文件不存在，跳过回填
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return err
	}

	var liveSettings ClaudeSettings
	if err := json.Unmarshal(data, &liveSettings); err != nil {
		return err
	}

	// 回填到当前供应商
	if currentProvider, ok := app.Providers[app.Current]; ok {
		if currentProvider.SettingsConfig == nil {
			currentProvider.SettingsConfig = make(map[string]interface{})
		}
		if _, ok := currentProvider.SettingsConfig["env"]; !ok {
			currentProvider.SettingsConfig["env"] = make(map[string]interface{})
		}

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
		app.Providers[app.Current] = currentProvider
		m.config.Apps["claude"] = app
	}

	return nil
}

// backfillCodexConfig 回填 Codex 的 live 配置
func (m *Manager) backfillCodexConfig() error {
	// TODO: 实现 Codex 配置回填
	return nil
}

// writeProviderConfig 写入供应商配置到目标应用
func (m *Manager) writeProviderConfig(appName string, provider *Provider) error {
	switch appName {
	case "claude":
		return m.writeClaudeConfig(provider)
	case "codex":
		return m.writeCodexConfig(provider)
	default:
		return fmt.Errorf("不支持的应用: %s", appName)
	}
}

// writeClaudeConfig 写入 Claude 配置
func (m *Manager) writeClaudeConfig(provider *Provider) error {
	settingsPath, err := GetClaudeSettingsPath()
	if err != nil {
		return fmt.Errorf("获取 Claude 设置文件路径失败: %w", err)
	}

	// 确保目录存在
	dir := filepath.Dir(settingsPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 备份当前设置
	if utils.FileExists(settingsPath) {
		if err := utils.BackupFile(settingsPath); err != nil {
			return fmt.Errorf("备份设置文件失败: %w", err)
		}
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
		if err == nil {
			json.Unmarshal(data, settings) // 忽略错误，使用默认值
		}
	}

	// 从 settingsConfig 提取 env 并写入
	if envMap, ok := provider.SettingsConfig["env"].(map[string]interface{}); ok {
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
	return utils.WriteJSONFile(settingsPath, settings, 0644)
}

// writeCodexConfig 写入 Codex 配置
func (m *Manager) writeCodexConfig(provider *Provider) error {
	// TODO: 实现 Codex 配置写入（需要双文件事务）
	return fmt.Errorf("Codex 支持尚未实现")
}

// UpdateProvider 更新供应商配置（默认为 Claude）
func (m *Manager) UpdateProvider(oldName, newName, apiToken, baseURL, category string) error {
	return m.UpdateProviderForApp("claude", oldName, newName, apiToken, baseURL, category)
}

// UpdateProviderForApp 更新指定应用的供应商配置
func (m *Manager) UpdateProviderForApp(appName, oldName, newName, apiToken, baseURL, category string) error {
	app, exists := m.config.Apps[appName]
	if !exists {
		return fmt.Errorf("应用 '%s' 不存在", appName)
	}

	// 找到要更新的 provider
	var targetID string
	var targetProvider Provider
	for id, p := range app.Providers {
		if p.Name == oldName {
			targetID = id
			targetProvider = p
			break
		}
	}

	if targetID == "" {
		return fmt.Errorf("配置 '%s' 不存在", oldName)
	}

	// 检查新名称是否与其他配置冲突
	if newName != oldName {
		for _, p := range app.Providers {
			if p.Name == newName {
				return fmt.Errorf("配置名称 '%s' 已存在", newName)
			}
		}
	}

	// 更新配置
	targetProvider.Name = newName
	if category != "" {
		targetProvider.Category = category
	}

	// 更新 settingsConfig
	switch appName {
	case "claude":
		if targetProvider.SettingsConfig == nil {
			targetProvider.SettingsConfig = make(map[string]interface{})
		}
		if _, ok := targetProvider.SettingsConfig["env"]; !ok {
			targetProvider.SettingsConfig["env"] = make(map[string]interface{})
		}
		if envMap, ok := targetProvider.SettingsConfig["env"].(map[string]interface{}); ok {
			envMap["ANTHROPIC_AUTH_TOKEN"] = apiToken
			envMap["ANTHROPIC_BASE_URL"] = baseURL
		}
	case "codex":
		if targetProvider.SettingsConfig == nil {
			targetProvider.SettingsConfig = make(map[string]interface{})
		}
		targetProvider.SettingsConfig["auth"] = apiToken
		targetProvider.SettingsConfig["base_url"] = baseURL
	}

	// 保存更新后的配置
	app.Providers[targetID] = targetProvider
	m.config.Apps[appName] = app

	// 如果更新的是当前激活的配置，立即应用到 live
	if app.Current == targetID {
		if err := m.writeProviderConfig(appName, &targetProvider); err != nil {
			return fmt.Errorf("更新 live 配置失败: %w", err)
		}
	}

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
