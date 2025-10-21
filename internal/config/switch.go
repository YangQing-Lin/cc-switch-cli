package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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
	default:
		return fmt.Errorf("不支持的应用: %s", appName)
	}
}

func (m *Manager) writeClaudeConfig(provider *Provider) error {
	settingsPath, err := GetClaudeSettingsPath()
	if err != nil {
		return fmt.Errorf("获取 Claude 设置文件路径失败: %w", err)
	}

	dir := filepath.Dir(settingsPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	var rollbackPath string
	needRollback := false
	if utils.FileExists(settingsPath) {
		rollbackPath = settingsPath + ".rollback"
		if err := utils.CopyFile(settingsPath, rollbackPath); err != nil {
			return fmt.Errorf("创建回滚备份失败: %w", err)
		}
		needRollback = true
		defer func() {
			if !needRollback && rollbackPath != "" {
				os.Remove(rollbackPath)
			}
		}()
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

	if model, ok := provider.SettingsConfig["model"].(string); ok {
		settings.Model = model
	} else {
		settings.Model = ""
	}

	if err := utils.WriteJSONFile(settingsPath, settings, 0644); err != nil {
		if needRollback && rollbackPath != "" {
			if restoreErr := utils.CopyFile(rollbackPath, settingsPath); restoreErr != nil {
				return fmt.Errorf("写入失败且无法恢复: 写入错误=%w, 恢复错误=%v", err, restoreErr)
			}
		}
		return fmt.Errorf("保存设置失败: %w", err)
	}

	needRollback = false
	return nil
}

func (m *Manager) writeCodexConfig(provider *Provider) error {
	authJsonPath, err := GetCodexAuthJsonPath()
	if err != nil {
		return fmt.Errorf("获取 Codex auth.json 路径失败: %w", err)
	}

	configPath, err := GetCodexConfigPath()
	if err != nil {
		return fmt.Errorf("获取 Codex config.toml 路径失败: %w", err)
	}

	dir := filepath.Dir(authJsonPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	var authRollbackPath, configRollbackPath string
	needRollback := false

	if utils.FileExists(authJsonPath) {
		authRollbackPath = authJsonPath + ".rollback"
		if err := utils.CopyFile(authJsonPath, authRollbackPath); err != nil {
			return fmt.Errorf("创建 auth.json 回滚备份失败: %w", err)
		}
		needRollback = true
	}

	if utils.FileExists(configPath) {
		configRollbackPath = configPath + ".rollback"
		if err := utils.CopyFile(configPath, configRollbackPath); err != nil {
			if authRollbackPath != "" {
				os.Remove(authRollbackPath)
			}
			return fmt.Errorf("创建 config.toml 回滚备份失败: %w", err)
		}
	}

	defer func() {
		if !needRollback {
			if authRollbackPath != "" {
				os.Remove(authRollbackPath)
			}
			if configRollbackPath != "" {
				os.Remove(configRollbackPath)
			}
		}
	}()

	authData := &CodexAuthJson{}
	if authMap, ok := provider.SettingsConfig["auth"].(map[string]interface{}); ok {
		if apiKey, ok := authMap["OPENAI_API_KEY"].(string); ok {
			authData.OpenAIAPIKey = apiKey
		}
	}

	var configContent string
	if configStr, ok := provider.SettingsConfig["config"].(string); ok {
		configContent = configStr
	}

	authJsonData, err := json.MarshalIndent(authData, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 auth.json 失败: %w", err)
	}

	if err := utils.AtomicWriteFile(authJsonPath, authJsonData, 0644); err != nil {
		if needRollback && authRollbackPath != "" {
			utils.CopyFile(authRollbackPath, authJsonPath)
		}
		return fmt.Errorf("写入 auth.json 失败: %w", err)
	}

	if configContent != "" {
		if err := utils.AtomicWriteFile(configPath, []byte(configContent), 0644); err != nil {
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
		if utils.FileExists(configPath) {
			if err := os.Remove(configPath); err != nil {
				fmt.Printf("警告: 删除空配置文件失败: %v\n", err)
			}
		}
	}

	needRollback = false
	return nil
}
