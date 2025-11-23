package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/YangQing-Lin/cc-switch-cli/internal/utils"
	"github.com/google/uuid"
)

// writeGeminiConfig writes Gemini configuration to ~/.gemini/.env and settings.json
func (m *Manager) writeGeminiConfig(provider *Provider) error {
	if provider == nil {
		return fmt.Errorf("provider 不能为空")
	}

	// 检测认证类型
	authType := DetectGeminiAuthType(provider)

	// 提取环境变量
	envMap := NormalizeGeminiEnv(provider)

	// 写入 .env 文件
	if err := m.writeGeminiEnvFile(envMap); err != nil {
		return fmt.Errorf("写入 .env 文件失败: %w", err)
	}

	// 写入 settings.json 文件
	if err := m.writeGeminiSettingsFile(authType); err != nil {
		return fmt.Errorf("写入 settings.json 失败: %w", err)
	}

	return nil
}

// DetectGeminiAuthType 检测 Gemini 认证类型
func DetectGeminiAuthType(provider *Provider) GeminiAuthType {
	if provider == nil {
		return GeminiAuthAPIKey
	}

	// 检查 SettingsConfig 中的 authType 字段
	if authTypeVal, ok := provider.SettingsConfig["authType"].(string); ok {
		if authTypeVal == string(GeminiAuthOAuth) {
			return GeminiAuthOAuth
		}
	}

	// 检查是否为 Google Official (通过 partner_promotion_key 或名称)
	name := strings.ToLower(provider.Name)
	if name == "google" || name == "google official" || strings.HasPrefix(name, "google ") {
		return GeminiAuthOAuth
	}

	// 检查 WebsiteURL
	if strings.Contains(strings.ToLower(provider.WebsiteURL), "ai.google") ||
		strings.Contains(strings.ToLower(provider.WebsiteURL), "aistudio.google") {
		return GeminiAuthOAuth
	}

	// 检查环境变量：如果没有 API Key，可能是 OAuth
	envMap, ok := provider.SettingsConfig["env"].(map[string]interface{})
	if !ok {
		return GeminiAuthAPIKey
	}

	apiKey1, hasKey1 := envMap["GEMINI_API_KEY"].(string)
	apiKey2, hasKey2 := envMap["GOOGLE_GEMINI_API_KEY"].(string)

	if (!hasKey1 || apiKey1 == "") && (!hasKey2 || apiKey2 == "") {
		// 没有 API Key，可能是 OAuth
		return GeminiAuthOAuth
	}

	// 默认为 API Key 模式
	return GeminiAuthAPIKey
}

// NormalizeGeminiEnv 从 Provider 提取并规范化环境变量
func NormalizeGeminiEnv(provider *Provider) map[string]string {
	result := make(map[string]string)

	if provider == nil {
		return result
	}

	envMap, ok := provider.SettingsConfig["env"].(map[string]interface{})
	if !ok {
		return result
	}

	// 提取标准环境变量
	if val, ok := envMap["GEMINI_API_KEY"].(string); ok && val != "" {
		result["GEMINI_API_KEY"] = val
	}
	if val, ok := envMap["GOOGLE_GEMINI_API_KEY"].(string); ok && val != "" {
		result["GOOGLE_GEMINI_API_KEY"] = val
	}
	if val, ok := envMap["GOOGLE_GEMINI_BASE_URL"].(string); ok && val != "" {
		result["GOOGLE_GEMINI_BASE_URL"] = val
	}
	if val, ok := envMap["GEMINI_MODEL"].(string); ok && val != "" {
		result["GEMINI_MODEL"] = val
	}

	// 支持其他自定义环境变量
	for key, val := range envMap {
		if _, exists := result[key]; !exists {
			if strVal, ok := val.(string); ok && strVal != "" {
				result[key] = strVal
			}
		}
	}

	return result
}

// writeGeminiEnvFile 写入 Gemini .env 文件
func (m *Manager) writeGeminiEnvFile(envMap map[string]string) error {
	envPath, err := m.GetGeminiEnvPathWithDir()
	if err != nil {
		return err
	}

	// 确保目录存在
	dir := filepath.Dir(envPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 生成 .env 文件内容（KEY=VALUE 格式）
	var lines []string
	lines = append(lines, "# Gemini CLI 配置")
	lines = append(lines, "# 由 CC-Switch 自动生成")
	lines = append(lines, "")

	// 按固定顺序写入标准变量
	if val, ok := envMap["GEMINI_API_KEY"]; ok {
		lines = append(lines, fmt.Sprintf("GEMINI_API_KEY=%s", val))
	}
	if val, ok := envMap["GOOGLE_GEMINI_API_KEY"]; ok {
		lines = append(lines, fmt.Sprintf("GOOGLE_GEMINI_API_KEY=%s", val))
	}
	if val, ok := envMap["GOOGLE_GEMINI_BASE_URL"]; ok {
		lines = append(lines, fmt.Sprintf("GOOGLE_GEMINI_BASE_URL=%s", val))
	}
	if val, ok := envMap["GEMINI_MODEL"]; ok {
		lines = append(lines, fmt.Sprintf("GEMINI_MODEL=%s", val))
	}

	// 写入其他自定义变量
	standardKeys := map[string]bool{
		"GEMINI_API_KEY":         true,
		"GOOGLE_GEMINI_API_KEY":  true,
		"GOOGLE_GEMINI_BASE_URL": true,
		"GEMINI_MODEL":           true,
	}
	for key, val := range envMap {
		if !standardKeys[key] {
			lines = append(lines, fmt.Sprintf("%s=%s", key, val))
		}
	}

	content := strings.Join(lines, "\n") + "\n"

	// 原子写入，权限 0600
	return utils.AtomicWriteFile(envPath, []byte(content), 0600)
}

// writeGeminiSettingsFile 写入 Gemini settings.json 文件
func (m *Manager) writeGeminiSettingsFile(authType GeminiAuthType) error {
	settingsPath, err := m.GetGeminiSettingsPathWithDir()
	if err != nil {
		return err
	}

	// 确保目录存在
	dir := filepath.Dir(settingsPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 读取现有配置（如果存在），保留 MCP 服务器和未知字段
	var existingSettings GeminiSettings
	if utils.FileExists(settingsPath) {
		data, err := os.ReadFile(settingsPath)
		if err != nil {
			return fmt.Errorf("读取现有 settings.json 失败: %w", err)
		}
		if err := json.Unmarshal(data, &existingSettings); err != nil {
			return fmt.Errorf("解析现有 settings.json 失败: %w", err)
		}
	}

	// 创建新配置
	settings := GeminiSettings{
		Security: GeminiSecurity{
			Auth: GeminiSecurityAuth{
				SelectedType: authType,
			},
		},
		MCPServers: existingSettings.MCPServers, // 保留现有 MCP 配置
		Extra:      existingSettings.Extra,      // 保留未知字段
	}

	// 如果没有 MCP 服务器，初始化为空对象
	if settings.MCPServers == nil {
		settings.MCPServers = make(map[string]interface{})
	}

	// 如果没有 Extra，初始化为空对象
	if settings.Extra == nil {
		settings.Extra = make(map[string]interface{})
	}

	// 原子写入 JSON，权限 0600
	return utils.WriteJSONFile(settingsPath, settings, 0600)
}

// ExtractGeminiConfigFromProvider extracts Gemini configuration fields from a Provider
func ExtractGeminiConfigFromProvider(p *Provider) (baseURL, apiKey, model string, authType GeminiAuthType) {
	if p == nil {
		return
	}

	// 提取认证类型
	authType = DetectGeminiAuthType(p)

	if envMap, ok := p.SettingsConfig["env"].(map[string]interface{}); ok {
		if val, ok := envMap["GOOGLE_GEMINI_BASE_URL"].(string); ok {
			baseURL = val
		}
		// 优先使用 GEMINI_API_KEY
		if val, ok := envMap["GEMINI_API_KEY"].(string); ok {
			apiKey = val
		} else if val, ok := envMap["GOOGLE_GEMINI_API_KEY"].(string); ok {
			apiKey = val
		}
		if val, ok := envMap["GEMINI_MODEL"].(string); ok {
			model = val
		}
	}
	return
}

// AddGeminiProvider adds a new Gemini provider configuration
func (m *Manager) AddGeminiProvider(name, baseURL, apiKey, model string, authType GeminiAuthType) error {
	app, exists := m.config.Apps["gemini"]
	if !exists {
		return fmt.Errorf("应用 'gemini' 不存在")
	}

	// 检查名称是否已存在
	for _, p := range app.Providers {
		if p.Name == name {
			return fmt.Errorf("配置 '%s' 已存在", name)
		}
	}

	// 构建环境变量配置
	env := make(map[string]interface{})
	if baseURL != "" {
		env["GOOGLE_GEMINI_BASE_URL"] = baseURL
	}
	if apiKey != "" {
		env["GEMINI_API_KEY"] = apiKey
	}
	if model != "" {
		env["GEMINI_MODEL"] = model
	}

	provider := Provider{
		ID:   uuid.New().String(),
		Name: name,
		SettingsConfig: map[string]interface{}{
			"env":      env,
			"authType": string(authType),
		},
		Category:  "custom",
		CreatedAt: time.Now().UnixMilli(),
	}

	// OAuth 模式下，设置 WebsiteURL 为 Google 官方
	if authType == GeminiAuthOAuth && provider.WebsiteURL == "" {
		provider.WebsiteURL = "https://ai.google.dev/"
	}

	// 使用 AddProviderDirect 添加（自动处理 SortOrder）
	if err := m.AddProviderDirect("gemini", provider); err != nil {
		return err
	}

	// 如果这是第一个配置，自动设置为当前配置并写入 live 文件
	app = m.config.Apps["gemini"]
	if len(app.Providers) == 1 {
		app.Current = provider.ID
		m.config.Apps["gemini"] = app
		if err := m.writeGeminiConfig(&provider); err != nil {
			return fmt.Errorf("写入 live 配置失败: %w", err)
		}
		if err := m.Save(); err != nil {
			return fmt.Errorf("保存配置失败: %w", err)
		}
	}

	return nil
}

// UpdateGeminiProvider updates an existing Gemini provider configuration
func (m *Manager) UpdateGeminiProvider(oldName, newName, baseURL, apiKey, model string, authType GeminiAuthType) error {
	app, exists := m.config.Apps["gemini"]
	if !exists {
		return fmt.Errorf("应用 'gemini' 不存在")
	}

	// 查找旧配置
	var targetID string
	var oldProvider *Provider
	for id, p := range app.Providers {
		if p.Name == oldName {
			targetID = id
			provider := p
			oldProvider = &provider
			break
		}
	}

	if targetID == "" {
		return fmt.Errorf("配置 '%s' 不存在", oldName)
	}

	// 如果改名，检查新名称是否冲突
	if newName != oldName {
		for _, p := range app.Providers {
			if p.Name == newName {
				return fmt.Errorf("配置 '%s' 已存在", newName)
			}
		}
	}

	// 构建环境变量配置
	env := make(map[string]interface{})
	if baseURL != "" {
		env["GOOGLE_GEMINI_BASE_URL"] = baseURL
	}
	if apiKey != "" {
		env["GEMINI_API_KEY"] = apiKey
	}
	if model != "" {
		env["GEMINI_MODEL"] = model
	}

	// 更新配置
	oldProvider.Name = newName
	oldProvider.SettingsConfig = map[string]interface{}{
		"env":      env,
		"authType": string(authType),
	}

	// OAuth 模式下，设置 WebsiteURL 为 Google 官方
	if authType == GeminiAuthOAuth && oldProvider.WebsiteURL == "" {
		oldProvider.WebsiteURL = "https://ai.google.dev/"
	}

	app.Providers[targetID] = *oldProvider
	m.config.Apps["gemini"] = app

	// 如果修改的是当前配置，同时更新 live 文件
	if app.Current == targetID {
		if err := m.writeGeminiConfig(oldProvider); err != nil {
			return fmt.Errorf("写入 live 配置失败: %w", err)
		}
	}

	return m.Save()
}
