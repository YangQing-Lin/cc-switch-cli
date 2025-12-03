package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/YangQing-Lin/cc-switch-cli/internal/utils"
)

func (m *Manager) SwitchProvider(name string) error {
	return m.SwitchProviderForApp("claude", name)
}

func (m *Manager) SwitchProviderForApp(appName, name string) error {
	app, exists := m.config.Apps[appName]
	if !exists {
		return fmt.Errorf("应用 '%s' 不存在", appName)
	}

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

	if err := m.writeProviderConfig(appName, targetProvider); err != nil {
		return fmt.Errorf("写入配置失败: %w", err)
	}

	app.Current = targetID
	m.config.Apps[appName] = app
	return m.Save()
}

func (m *Manager) writeProviderConfig(appName string, provider *Provider) error {
	switch appName {
	case "claude":
		return m.writeClaudeConfig(provider)
	case "codex":
		return m.writeCodexConfig(provider)
	case "gemini":
		return m.writeGeminiConfig(provider)
	default:
		return fmt.Errorf("不支持的应用: %s", appName)
	}
}

func (m *Manager) writeClaudeConfig(provider *Provider) error {
	settingsPath, err := m.GetClaudeSettingsPathWithDir()
	if err != nil {
		return fmt.Errorf("获取 Claude 设置文件路径失败: %w", err)
	}

	dir := filepath.Dir(settingsPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	settings := &ClaudeSettings{
		Permissions: ClaudePermissions{
			Allow: []string{},
			Deny:  []string{},
		},
	}

	if utils.FileExists(settingsPath) {
		data, err := os.ReadFile(settingsPath)
		if err == nil {
			json.Unmarshal(data, settings)
		}
	}

	if envMap, ok := provider.SettingsConfig["env"].(map[string]interface{}); ok {
		settings.Env = ClaudeEnv{}
		if token, ok := envMap["ANTHROPIC_AUTH_TOKEN"].(string); ok {
			settings.Env.AnthropicAuthToken = token
		}
		if baseURL, ok := envMap["ANTHROPIC_BASE_URL"].(string); ok {
			settings.Env.AnthropicBaseURL = baseURL
		}
		if model, ok := envMap["ANTHROPIC_MODEL"].(string); ok {
			settings.Env.AnthropicModel = model
		}
		if defaultHaikuModel, ok := envMap["ANTHROPIC_DEFAULT_HAIKU_MODEL"].(string); ok {
			settings.Env.AnthropicDefaultHaikuModel = defaultHaikuModel
		}
		if model, ok := envMap["CLAUDE_CODE_MODEL"].(string); ok {
			settings.Env.ClaudeCodeModel = model
		}
		if maxTokens, ok := envMap["CLAUDE_CODE_MAX_TOKENS"].(string); ok {
			settings.Env.ClaudeCodeMaxTokens = maxTokens
		}
		if defaultOpusModel, ok := envMap["ANTHROPIC_DEFAULT_OPUS_MODEL"].(string); ok {
			settings.Env.AnthropicDefaultOpusModel = defaultOpusModel
		}
		if defaultSonnetModel, ok := envMap["ANTHROPIC_DEFAULT_SONNET_MODEL"].(string); ok {
			settings.Env.AnthropicDefaultSonnetModel = defaultSonnetModel
		}
	}

	if model, ok := provider.SettingsConfig["model"].(string); ok {
		settings.Model = model
	} else if settings.Env.AnthropicModel != "" {
		settings.Model = settings.Env.AnthropicModel
	} else {
		settings.Model = ""
	}

	if settings.Env.AnthropicModel == "" && settings.Model != "" {
		settings.Env.AnthropicModel = settings.Model
	}

	if err := utils.WriteJSONFile(settingsPath, settings, 0644); err != nil {
		return fmt.Errorf("保存设置失败: %w", err)
	}

	return nil
}

func (m *Manager) writeCodexConfig(provider *Provider) error {
	authJsonPath, err := m.GetCodexAuthJsonPathWithDir()
	if err != nil {
		return fmt.Errorf("获取 Codex auth.json 路径失败: %w", err)
	}

	configPath, err := m.GetCodexConfigPathWithDir()
	if err != nil {
		return fmt.Errorf("获取 Codex config.toml 路径失败: %w", err)
	}

	dir := filepath.Dir(authJsonPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	authData := &CodexAuthJson{}
	if authMap, ok := provider.SettingsConfig["auth"].(map[string]interface{}); ok {
		if apiKey, ok := authMap["OPENAI_API_KEY"].(string); ok {
			authData.OpenAIAPIKey = apiKey
		}
	}

	authJsonData, err := json.MarshalIndent(authData, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 auth.json 失败: %w", err)
	}

	if err := utils.AtomicWriteFile(authJsonPath, authJsonData, 0644); err != nil {
		return fmt.Errorf("写入 auth.json 失败: %w", err)
	}

	// 获取 CCS 配置字符串
	configContent, _ := provider.SettingsConfig["config"].(string)

	// 如果 CCS 配置为空，删除 config.toml
	if configContent == "" {
		if utils.FileExists(configPath) {
			if err := os.Remove(configPath); err != nil {
				fmt.Printf("警告: 删除空配置文件失败: %v\n", err)
			}
		}
		return nil
	}

	// 解析 CCS 配置
	var ccsConfig map[string]interface{}
	if err := toml.Unmarshal([]byte(configContent), &ccsConfig); err != nil {
		// 解析失败，回退到完全覆盖
		return utils.AtomicWriteFile(configPath, []byte(configContent), 0644)
	}

	// 读取现有配置（如果存在）
	existingConfig := make(map[string]interface{})
	if utils.FileExists(configPath) {
		data, err := os.ReadFile(configPath)
		if err == nil {
			toml.Unmarshal(data, &existingConfig) // 忽略解析错误
		}
	}

	// 合并配置
	mergedConfig := mergeCodexConfig(existingConfig, ccsConfig)

	// 序列化并写入
	data, err := toml.Marshal(mergedConfig)
	if err != nil {
		return fmt.Errorf("序列化 config.toml 失败: %w", err)
	}

	return utils.AtomicWriteFile(configPath, data, 0644)
}

// mergeCodexConfig 合并 Codex 配置，CCS 管理的字段覆盖，其他字段保留
func mergeCodexConfig(existing, ccs map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// 1. 复制 existing 中所有字段到 result
	for k, v := range existing {
		result[k] = v
	}

	// 2. 用 CCS 配置覆盖管理的字段
	ccsFields := []string{
		"model_provider",
		"model",
		"model_reasoning_effort",
		"disable_response_storage",
	}
	for _, field := range ccsFields {
		if val, ok := ccs[field]; ok {
			result[field] = val
		}
	}

	// 3. 合并 model_providers 段
	if ccsProviders, ok := ccs["model_providers"].(map[string]interface{}); ok {
		existingProviders := make(map[string]interface{})
		if ep, ok := result["model_providers"].(map[string]interface{}); ok {
			existingProviders = ep
		}

		// CCS 的 provider 段覆盖同名段，同时过滤废弃字段
		for name, config := range ccsProviders {
			if providerConfig, ok := config.(map[string]interface{}); ok {
				// 过滤掉 env_key（已废弃）
				filteredConfig := make(map[string]interface{})
				for k, v := range providerConfig {
					if k != "env_key" {
						filteredConfig[k] = v
					}
				}
				existingProviders[name] = filteredConfig
			} else {
				existingProviders[name] = config
			}
		}
		result["model_providers"] = existingProviders
	}

	return result
}
