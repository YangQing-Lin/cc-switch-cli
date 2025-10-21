package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/YangQing-Lin/cc-switch-cli/internal/portable"
	"github.com/YangQing-Lin/cc-switch-cli/internal/utils"
	"github.com/google/uuid"
)

// Manager 配置管理器（对应 cc-switch 的 AppState）
type Manager struct {
	config     *MultiAppConfig
	configPath string
	customDir  string // 自定义配置目录
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

// NewManagerWithDir 创建使用自定义目录的配置管理器
func NewManagerWithDir(customDir string) (*Manager, error) {
	// 验证目录
	if customDir == "" {
		return NewManager() // 使用默认目录
	}

	// 确保目录存在
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

// Load 加载配置文件（支持向后兼容和自动迁移）
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

// createDefaultConfig 创建默认配置（仅内存）
func (m *Manager) createDefaultConfig() {
	m.config = &MultiAppConfig{
		Version: 2,
		Apps: map[string]ProviderManager{
			"claude": {Providers: make(map[string]Provider), Current: ""},
			"codex":  {Providers: make(map[string]Provider), Current: ""},
		},
	}
}

// isEmptyConfig 检查配置是否为空
func (m *Manager) isEmptyConfig(data []byte) bool {
	return len(data) == 0 || string(data) == "" || string(data) == "{}"
}

// handleEmptyConfig 处理空配置文件
func (m *Manager) handleEmptyConfig() error {
	fmt.Println("配置文件为空，创建默认配置...")
	m.createDefaultConfig()
	return m.Save()
}

// loadAndMigrate 加载并迁移配置
func (m *Manager) loadAndMigrate(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return m.handleCorruptedConfig(data, err)
	}

	hasVersion, hasAppsKey := m.detectConfigVersion(raw)

	// v1 格式迁移
	if !hasVersion && !hasAppsKey {
		if err := m.migrateV1Config(data); err == nil {
			return nil
		}
	}

	// v2-old 格式迁移
	if hasAppsKey {
		if err := m.migrateV2OldConfig(data); err == nil {
			return nil
		}
	}

	// v2 格式解析
	return m.parseV2Config(data)
}

// detectConfigVersion 检测配置版本
func (m *Manager) detectConfigVersion(raw map[string]json.RawMessage) (hasVersion, hasAppsKey bool) {
	if versionRaw, exists := raw["version"]; exists {
		var version int
		if json.Unmarshal(versionRaw, &version) == nil {
			hasVersion = true
		}
	}
	_, hasAppsKey = raw["apps"]
	return
}

// handleCorruptedConfig 处理损坏的配置
func (m *Manager) handleCorruptedConfig(data []byte, err error) error {
	fmt.Printf("警告: 配置文件损坏 (%v)，将创建默认配置\n", err)
	backupPath := m.configPath + ".corrupted." + fmt.Sprintf("%d", time.Now().Unix())
	if os.WriteFile(backupPath, data, 0600) == nil {
		fmt.Printf("已备份损坏的配置到: %s\n", backupPath)
	}

	// 提示用户可以使用备份恢复
	fmt.Println()
	fmt.Println("💡 提示: 您可以使用以下命令从备份恢复配置:")
	fmt.Println("   cc-switch backup list      # 查看可用备份")
	fmt.Println("   cc-switch backup restore <backup-id>  # 恢复备份")
	fmt.Println()

	m.createDefaultConfig()
	return m.Save()
}

// migrateV1Config 迁移 v1 配置
func (m *Manager) migrateV1Config(data []byte) error {
	var v1Config ProviderManager
	if err := json.Unmarshal(data, &v1Config); err != nil || v1Config.Providers == nil {
		return err
	}

	fmt.Println("检测到 v1 配置格式，自动迁移到 v2...")
	backupPath := m.configPath + ".v1.backup." + fmt.Sprintf("%d", time.Now().Unix())
	if os.WriteFile(backupPath, data, 0600) == nil {
		fmt.Printf("已备份 v1 配置到: %s\n", backupPath)
	}

	m.config = &MultiAppConfig{
		Version: 2,
		Apps: map[string]ProviderManager{
			"claude": v1Config,
			"codex":  {Providers: make(map[string]Provider), Current: ""},
		},
	}

	if err := m.Save(); err != nil {
		return fmt.Errorf("保存迁移后的配置失败: %w", err)
	}
	fmt.Println("v1 配置迁移完成")
	return nil
}

// migrateV2OldConfig 迁移 v2-old 配置
func (m *Manager) migrateV2OldConfig(data []byte) error {
	var oldConfig OldMultiAppConfig
	if err := json.Unmarshal(data, &oldConfig); err != nil || oldConfig.Apps == nil || len(oldConfig.Apps) == 0 {
		return err
	}

	fmt.Println("检测到旧版配置格式，自动迁移到新格式...")
	if err := m.archiveOldConfig(); err != nil {
		fmt.Printf("警告: 归档旧配置失败: %v\n", err)
	}

	m.config = &MultiAppConfig{Version: 2, Apps: oldConfig.Apps}
	m.ensureProvidersInitialized()

	if err := m.Save(); err != nil {
		return fmt.Errorf("保存迁移后的配置失败: %w", err)
	}
	fmt.Println("配置迁移完成")
	return nil
}

// parseV2Config 解析 v2 配置
func (m *Manager) parseV2Config(data []byte) error {
	m.config = &MultiAppConfig{}
	if err := json.Unmarshal(data, m.config); err == nil {
		if m.config.Apps != nil && len(m.config.Apps) > 0 {
			m.ensureProvidersInitialized()
			return nil
		}

		fmt.Println("配置文件数据为空，创建默认配置...")
		m.createDefaultConfig()
		return m.Save()
	}

	// 格式不支持
	fmt.Println("警告: 配置格式不支持，将创建默认配置")
	backupPath := m.configPath + ".unsupported." + fmt.Sprintf("%d", time.Now().Unix())
	if os.WriteFile(backupPath, data, 0600) == nil {
		fmt.Printf("已备份不支持的配置到: %s\n", backupPath)
	}
	m.createDefaultConfig()
	return m.Save()
}

// ensureProvidersInitialized 确保所有 app 的 Providers map 已初始化
func (m *Manager) ensureProvidersInitialized() {
	for appName, app := range m.config.Apps {
		if app.Providers == nil {
			app.Providers = make(map[string]Provider)
			m.config.Apps[appName] = app
		}
	}
}

// Save 保存配置文件（自动备份）
func (m *Manager) Save() error {
	// 确保目录存在
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 自动备份（静默失败，不影响保存）
	if utils.FileExists(m.configPath) {
		// Import backup package at the top of the file
		// backup.CreateAutoBackup(m.configPath)
		// 暂时使用简单备份，避免循环依赖
		backupPath := m.configPath + ".bak.cli"
		data, err := os.ReadFile(m.configPath)
		if err == nil {
			_ = os.WriteFile(backupPath, data, 0600)
		}
	}

	return utils.WriteJSONFile(m.configPath, m.config, 0600)
}

// archiveOldConfig 归档旧配置文件（迁移时使用）
func (m *Manager) archiveOldConfig() error {
	if !utils.FileExists(m.configPath) {
		return nil
	}

	// 创建归档目录 ~/.cc-switch/archive
	dir := filepath.Dir(m.configPath)
	archiveDir := filepath.Join(dir, "archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return fmt.Errorf("创建归档目录失败: %w", err)
	}

	// 生成归档文件名（带时间戳）
	timestamp := time.Now().Unix()
	archivePath := filepath.Join(archiveDir, fmt.Sprintf("config.v2-old.backup.%d.json", timestamp))

	// 复制文件到归档目录
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	if err := os.WriteFile(archivePath, data, 0600); err != nil {
		return fmt.Errorf("写入归档文件失败: %w", err)
	}

	fmt.Printf("已归档旧配置: %s\n", archivePath)
	return nil
}

// AddProvider 添加新的供应商配置（默认为 Claude）
func (m *Manager) AddProvider(name, apiToken, baseURL, category string) error {
	return m.AddProviderWithWebsite("claude", name, "", apiToken, baseURL, category)
}

// AddProviderWithWebsite 添加供应商配置（支持网站URL）
func (m *Manager) AddProviderWithWebsite(appName, name, websiteURL, apiToken, baseURL, category string) error {
	return m.AddProviderForApp(appName, name, websiteURL, apiToken, baseURL, category, "", "")
}

// AddProviderForApp 为指定应用添加供应商配置
func (m *Manager) AddProviderForApp(appName, name, websiteURL, apiToken, baseURL, category, claudeModel, defaultSonnetModel string) error {
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
		envMap := map[string]interface{}{
			"ANTHROPIC_AUTH_TOKEN": apiToken,
			"ANTHROPIC_BASE_URL":   baseURL,
		}
		// 如果提供了 default-sonnet-model，添加到 env
		if defaultSonnetModel != "" {
			envMap["ANTHROPIC_DEFAULT_SONNET_MODEL"] = defaultSonnetModel
		}
		settingsConfig = map[string]interface{}{
			"env": envMap,
		}
		// 如果提供了 claudeModel，添加到顶层
		if claudeModel != "" {
			settingsConfig["model"] = claudeModel
		}
	case "codex":
		// Codex 配置格式（符合 cc-switch 的格式）
		// auth: { OPENAI_API_KEY: "..." }
		// config: TOML 字符串
		settingsConfig = map[string]interface{}{
			"auth": map[string]interface{}{
				"OPENAI_API_KEY": apiToken,
			},
			"config": generateCodexConfigTOML(name, baseURL, "gpt-5-codex"),
		}
	default:
		return fmt.Errorf("不支持的应用: %s", appName)
	}

	// 创建 Provider
	provider := Provider{
		ID:             id,
		Name:           name,
		SettingsConfig: settingsConfig,
		WebsiteURL:     websiteURL,
		Category:       category,
		CreatedAt:      time.Now().UnixMilli(),
	}

	app.Providers[id] = provider

	// 如果是第一个配置，自动设置为当前配置并立即写入 live 配置
	isFirstProvider := len(app.Providers) == 1
	if isFirstProvider {
		app.Current = id
	}

	m.config.Apps[appName] = app

	// 先保存到内存配置文件
	if err := m.Save(); err != nil {
		return err
	}

	// 如果是第一个配置，立即写入 live 配置文件
	if isFirstProvider {
		if err := m.writeProviderConfig(appName, &provider); err != nil {
			return fmt.Errorf("保存配置成功，但写入 live 配置失败: %w", err)
		}
	}

	return nil
}

// AddProviderDirect 直接添加 Provider 对象（用于导入）
func (m *Manager) AddProviderDirect(appName string, provider Provider) error {
	// 确保应用存在
	if _, exists := m.config.Apps[appName]; !exists {
		m.config.Apps[appName] = ProviderManager{
			Providers: make(map[string]Provider),
			Current:   "",
		}
	}

	app := m.config.Apps[appName]

	// 检查 ID 是否已存在
	if _, exists := app.Providers[provider.ID]; exists {
		return fmt.Errorf("Provider ID '%s' 已存在", provider.ID)
	}

	// 检查名称是否已存在
	for _, p := range app.Providers {
		if p.Name == provider.Name {
			return fmt.Errorf("配置名称 '%s' 已存在", provider.Name)
		}
	}

	// 添加 Provider
	app.Providers[provider.ID] = provider

	// 如果是第一个配置，自动设置为当前配置
	if len(app.Providers) == 1 {
		app.Current = provider.ID
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

	// 按创建时间排序，保证顺序稳定
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].CreatedAt < providers[j].CreatedAt
	})

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

// GetConfig 获取完整配置（用于导出）
func (m *Manager) GetConfig() (*MultiAppConfig, error) {
	// 返回配置的副本，避免外部修改
	configCopy := &MultiAppConfig{
		Version: m.config.Version,
		Apps:    make(map[string]ProviderManager),
	}

	for appName, appConfig := range m.config.Apps {
		providersCopy := make(map[string]Provider)
		for id, provider := range appConfig.Providers {
			providersCopy[id] = provider
		}
		configCopy.Apps[appName] = ProviderManager{
			Providers: providersCopy,
			Current:   appConfig.Current,
		}
	}

	return configCopy, nil
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
	app := m.config.Apps["codex"]
	if app.Current == "" {
		return nil
	}

	// Codex 使用两个配置文件: auth.json 和 config.toml
	authJsonPath, err := GetCodexAuthJsonPath()
	if err != nil || !utils.FileExists(authJsonPath) {
		return nil // 文件不存在，跳过回填
	}

	configPath, err := GetCodexConfigPath()
	if err != nil {
		return nil
	}

	// 读取 auth.json
	authData, err := os.ReadFile(authJsonPath)
	if err != nil {
		return err
	}

	var liveAuth CodexAuthJson
	if err := json.Unmarshal(authData, &liveAuth); err != nil {
		return err
	}

	// 读取 config.toml (可能不存在)
	var configContent string
	if utils.FileExists(configPath) {
		configData, err := os.ReadFile(configPath)
		if err == nil {
			configContent = string(configData)
		}
	}

	// 回填到当前供应商
	if currentProvider, ok := app.Providers[app.Current]; ok {
		if currentProvider.SettingsConfig == nil {
			currentProvider.SettingsConfig = make(map[string]interface{})
		}

		// 保存配置到 settingsConfig
		currentProvider.SettingsConfig["auth"] = map[string]interface{}{
			"OPENAI_API_KEY": liveAuth.OpenAIAPIKey,
		}
		currentProvider.SettingsConfig["config"] = configContent

		app.Providers[app.Current] = currentProvider
		m.config.Apps["codex"] = app
	}

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

// writeClaudeConfig 写入 Claude 配置（带回滚机制）
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

	// 创建回滚备份
	var rollbackPath string
	needRollback := false
	if utils.FileExists(settingsPath) {
		rollbackPath = settingsPath + ".rollback"
		if err := utils.CopyFile(settingsPath, rollbackPath); err != nil {
			return fmt.Errorf("创建回滚备份失败: %w", err)
		}
		needRollback = true
		defer func() {
			// 清理回滚文件
			if !needRollback && rollbackPath != "" {
				os.Remove(rollbackPath)
			}
		}()
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
		if defaultSonnetModel, ok := envMap["ANTHROPIC_DEFAULT_SONNET_MODEL"].(string); ok {
			settings.Env.AnthropicDefaultSonnetModel = defaultSonnetModel
		}
	}

	// 保存设置
	if err := utils.WriteJSONFile(settingsPath, settings, 0644); err != nil {
		// 如果写入失败，尝试恢复
		if needRollback && rollbackPath != "" {
			if restoreErr := utils.CopyFile(rollbackPath, settingsPath); restoreErr != nil {
				return fmt.Errorf("写入失败且无法恢复: 写入错误=%w, 恢复错误=%v", err, restoreErr)
			}
		}
		return fmt.Errorf("保存设置失败: %w", err)
	}

	// 成功写入，标记不需要回滚
	needRollback = false
	return nil
}

// writeCodexConfig 写入 Codex 配置（双文件事务机制）
func (m *Manager) writeCodexConfig(provider *Provider) error {
	authJsonPath, err := GetCodexAuthJsonPath()
	if err != nil {
		return fmt.Errorf("获取 Codex auth.json 路径失败: %w", err)
	}

	configPath, err := GetCodexConfigPath()
	if err != nil {
		return fmt.Errorf("获取 Codex config.toml 路径失败: %w", err)
	}

	// 确保目录存在
	dir := filepath.Dir(authJsonPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 创建回滚备份（双文件）
	var authRollbackPath, configRollbackPath string
	needRollback := false

	// 备份 auth.json
	if utils.FileExists(authJsonPath) {
		authRollbackPath = authJsonPath + ".rollback"
		if err := utils.CopyFile(authJsonPath, authRollbackPath); err != nil {
			return fmt.Errorf("创建 auth.json 回滚备份失败: %w", err)
		}
		needRollback = true
	}

	// 备份 config.toml
	if utils.FileExists(configPath) {
		configRollbackPath = configPath + ".rollback"
		if err := utils.CopyFile(configPath, configRollbackPath); err != nil {
			// 清理第一个备份
			if authRollbackPath != "" {
				os.Remove(authRollbackPath)
			}
			return fmt.Errorf("创建 config.toml 回滚备份失败: %w", err)
		}
	}

	// 清理函数
	defer func() {
		if !needRollback {
			// 成功时清理备份文件
			if authRollbackPath != "" {
				os.Remove(authRollbackPath)
			}
			if configRollbackPath != "" {
				os.Remove(configRollbackPath)
			}
		}
	}()

	// 准备 auth.json 数据
	authData := &CodexAuthJson{}
	if authMap, ok := provider.SettingsConfig["auth"].(map[string]interface{}); ok {
		if apiKey, ok := authMap["OPENAI_API_KEY"].(string); ok {
			authData.OpenAIAPIKey = apiKey
		}
	}

	// 准备 config.toml 数据（TOML 字符串）
	var configContent string
	if configStr, ok := provider.SettingsConfig["config"].(string); ok {
		configContent = configStr
	}

	// 双文件事务写入
	// 第一阶段：写入 auth.json
	authJsonData, err := json.MarshalIndent(authData, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 auth.json 失败: %w", err)
	}

	if err := utils.AtomicWriteFile(authJsonPath, authJsonData, 0644); err != nil {
		// 尝试恢复
		if needRollback && authRollbackPath != "" {
			utils.CopyFile(authRollbackPath, authJsonPath)
		}
		return fmt.Errorf("写入 auth.json 失败: %w", err)
	}

	// 第二阶段：写入 config.toml (可能为空)
	if configContent != "" {
		if err := utils.AtomicWriteFile(configPath, []byte(configContent), 0644); err != nil {
			// 恢复两个文件
			if needRollback {
				if authRollbackPath != "" {
					utils.CopyFile(authRollbackPath, authJsonPath)
				}
				if configRollbackPath != "" {
					utils.CopyFile(configRollbackPath, configPath)
				}
			}
			return fmt.Errorf("写入 config.toml 失败: %w", err)
		}
	} else {
		// 如果配置为空，删除 config.toml（如果存在）
		if utils.FileExists(configPath) {
			if err := os.Remove(configPath); err != nil {
				fmt.Printf("警告: 删除空配置文件失败: %v\n", err)
			}
		}
	}

	// 成功写入，标记不需要回滚
	needRollback = false
	return nil
}

// UpdateProvider 更新供应商配置（默认为 Claude）
func (m *Manager) UpdateProvider(oldName, newName, apiToken, baseURL, category string) error {
	return m.UpdateProviderWithWebsite("claude", oldName, newName, "", apiToken, baseURL, category)
}

// UpdateProviderWithWebsite 更新供应商配置（支持网站URL）
func (m *Manager) UpdateProviderWithWebsite(appName, oldName, newName, websiteURL, apiToken, baseURL, category string) error {
	return m.UpdateProviderForApp(appName, oldName, newName, websiteURL, apiToken, baseURL, category, "", "")
}

// UpdateProviderForApp 更新指定应用的供应商配置
func (m *Manager) UpdateProviderForApp(appName, oldName, newName, websiteURL, apiToken, baseURL, category, claudeModel, defaultSonnetModel string) error {
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
	if websiteURL != "" {
		targetProvider.WebsiteURL = websiteURL
	}
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
			// 更新 ANTHROPIC_DEFAULT_SONNET_MODEL（如果提供）
			if defaultSonnetModel != "" {
				envMap["ANTHROPIC_DEFAULT_SONNET_MODEL"] = defaultSonnetModel
			} else {
				delete(envMap, "ANTHROPIC_DEFAULT_SONNET_MODEL")
			}
		}
		// 更新顶层 model 字段（如果提供）
		if claudeModel != "" {
			targetProvider.SettingsConfig["model"] = claudeModel
		} else {
			delete(targetProvider.SettingsConfig, "model")
		}
	case "codex":
		if targetProvider.SettingsConfig == nil {
			targetProvider.SettingsConfig = make(map[string]interface{})
		}
		// 更新 auth 部分
		if _, ok := targetProvider.SettingsConfig["auth"]; !ok {
			targetProvider.SettingsConfig["auth"] = make(map[string]interface{})
		}
		if authMap, ok := targetProvider.SettingsConfig["auth"].(map[string]interface{}); ok {
			authMap["OPENAI_API_KEY"] = apiToken
		}
		// 重新生成 config TOML 字符串
		targetProvider.SettingsConfig["config"] = generateCodexConfigTOML(newName, baseURL, "gpt-5-codex")
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
	// 检测便携版模式
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

// GetClaudeSettingsPathWithDir 获取使用自定义目录的 Claude 设置文件路径
func (m *Manager) GetClaudeSettingsPathWithDir() (string, error) {
	if m.customDir == "" {
		return GetClaudeSettingsPath()
	}

	dir := filepath.Join(m.customDir, ".claude")
	settingsPath := filepath.Join(dir, "settings.json")
	return settingsPath, nil
}

// GetCodexConfigPath 获取 Codex config.toml 文件路径
func GetCodexConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}
	return filepath.Join(home, ".codex", "config.toml"), nil
}

// GetCodexAuthJsonPath 获取 Codex auth.json 文件路径
func GetCodexAuthJsonPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}
	return filepath.Join(home, ".codex", "auth.json"), nil
}

// GetCodexConfigPathWithDir 获取使用自定义目录的 Codex config.toml 文件路径
func (m *Manager) GetCodexConfigPathWithDir() (string, error) {
	if m.customDir == "" {
		return GetCodexConfigPath()
	}
	return filepath.Join(m.customDir, ".codex", "config.toml"), nil
}

// GetCodexAuthJsonPathWithDir 获取使用自定义目录的 Codex auth.json 文件路径
func (m *Manager) GetCodexAuthJsonPathWithDir() (string, error) {
	if m.customDir == "" {
		return GetCodexAuthJsonPath()
	}
	return filepath.Join(m.customDir, ".codex", "auth.json"), nil
}

// generateCodexConfigTOML 生成 Codex 的 config.toml 字符串
func generateCodexConfigTOML(providerName, baseURL, modelName string) string {
	// 清理供应商名称，确保符合 TOML 键名规范
	cleanName := strings.ToLower(providerName)
	cleanName = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return '_'
	}, cleanName)
	cleanName = strings.Trim(cleanName, "_")
	if cleanName == "" {
		cleanName = "custom"
	}

	return fmt.Sprintf(`model_provider = "%s"
model = "%s"
model_reasoning_effort = "high"
disable_response_storage = true

[model_providers.%s]
name = "%s"
base_url = "%s"
wire_api = "responses"`, cleanName, modelName, cleanName, cleanName, baseURL)
}

// ExtractTokenFromProvider 从 Provider 提取 API Token
func ExtractTokenFromProvider(p *Provider) string {
	if p == nil {
		return ""
	}

	// 尝试 Claude 格式
	if envMap, ok := p.SettingsConfig["env"].(map[string]interface{}); ok {
		if token, ok := envMap["ANTHROPIC_AUTH_TOKEN"].(string); ok {
			return token
		}
	}

	// 尝试 Codex 格式 (auth.OPENAI_API_KEY)
	if authMap, ok := p.SettingsConfig["auth"].(map[string]interface{}); ok {
		if token, ok := authMap["OPENAI_API_KEY"].(string); ok {
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

	// 尝试 Claude 格式
	if envMap, ok := p.SettingsConfig["env"].(map[string]interface{}); ok {
		if baseURL, ok := envMap["ANTHROPIC_BASE_URL"].(string); ok {
			return baseURL
		}
	}

	// 尝试 Codex 格式 (从 config TOML 字符串中提取)
	if configStr, ok := p.SettingsConfig["config"].(string); ok {
		// 使用正则表达式从 TOML 中提取 base_url
		// base_url = "https://..."
		re := regexp.MustCompile(`base_url\s*=\s*"([^"]+)"`)
		if matches := re.FindStringSubmatch(configStr); len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

// ExtractDefaultSonnetModelFromProvider 从 Provider 提取 Default Sonnet Model
func ExtractDefaultSonnetModelFromProvider(p *Provider) string {
	if p == nil {
		return ""
	}

	// 仅适用于 Claude 格式
	if envMap, ok := p.SettingsConfig["env"].(map[string]interface{}); ok {
		if model, ok := envMap["ANTHROPIC_DEFAULT_SONNET_MODEL"].(string); ok {
			return model
		}
	}

	return ""
}

// ExtractModelFromProvider 从 Provider 提取 model 字段
func ExtractModelFromProvider(p *Provider) string {
	if p == nil {
		return ""
	}

	// 仅适用于 Claude 格式（顶层 model 字段）
	if model, ok := p.SettingsConfig["model"].(string); ok {
		return model
	}

	return ""
}

// GetConfigPath 返回配置文件路径
func (m *Manager) GetConfigPath() string {
	return m.configPath
}
