package config

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
)

// writeGeminiConfig does nothing for Gemini since it uses environment variables only
func (m *Manager) writeGeminiConfig(provider *Provider) error {
	// Gemini 不写配置文件，仅通过 export 语句加载环境变量
	return nil
}

// GetEnvCommandExample 返回当前系统加载环境变量的命令示例
func GetEnvCommandExample() string {
	switch runtime.GOOS {
	case "windows":
		// 优先 PowerShell (Windows 10+ 默认)
		return "ccs.exe gc | Invoke-Expression"
	default: // linux, darwin
		return "eval $(ccs gc)"
	}
}

// GenerateGeminiEnvExport generates environment variable export statements for the given Gemini provider
// quiet: if true, omit comments (for piping to Invoke-Expression)
func GenerateGeminiEnvExport(provider *Provider, configName string, quiet bool) (string, error) {
	if provider == nil {
		return "", fmt.Errorf("provider 不能为空")
	}

	var baseURL, apiKey, model string

	// 从 SettingsConfig 提取环境变量
	if envMap, ok := provider.SettingsConfig["env"].(map[string]interface{}); ok {
		if val, ok := envMap["GOOGLE_GEMINI_BASE_URL"].(string); ok {
			baseURL = val
		}
		if val, ok := envMap["GEMINI_API_KEY"].(string); ok {
			apiKey = val
		}
		if val, ok := envMap["GEMINI_MODEL"].(string); ok {
			model = val
		}
	}

	// 按系统生成对应格式
	var scriptLines []string
	switch runtime.GOOS {
	case "windows":
		// PowerShell 格式
		if baseURL != "" {
			scriptLines = append(scriptLines, fmt.Sprintf("$env:GOOGLE_GEMINI_BASE_URL=%s", shellQuote(baseURL)))
		}
		if apiKey != "" {
			scriptLines = append(scriptLines, fmt.Sprintf("$env:GEMINI_API_KEY=%s", shellQuote(apiKey)))
		}
		if model != "" {
			scriptLines = append(scriptLines, fmt.Sprintf("$env:GEMINI_MODEL=%s", shellQuote(model)))
		}
	default: // linux, darwin
		// Unix export 格式
		if baseURL != "" {
			scriptLines = append(scriptLines, fmt.Sprintf("export GOOGLE_GEMINI_BASE_URL=%s", shellQuote(baseURL)))
		}
		if apiKey != "" {
			scriptLines = append(scriptLines, fmt.Sprintf("export GEMINI_API_KEY=%s", shellQuote(apiKey)))
		}
		if model != "" {
			scriptLines = append(scriptLines, fmt.Sprintf("export GEMINI_MODEL=%s", shellQuote(model)))
		}
	}

	// 添加提示文本（除非 quiet 模式）
	if !quiet && configName != "" {
		scriptLines = append(scriptLines,
			fmt.Sprintf("# 运行以下命令加载 %s 配置:", configName),
			fmt.Sprintf("#   %s", GetEnvCommandExample()),
		)
	}

	if len(scriptLines) == 0 {
		return "", nil
	}

	return strings.Join(scriptLines, "\n") + "\n", nil
}

// shellQuote quotes a string for safe use in shell export statements
func shellQuote(s string) string {
	switch runtime.GOOS {
	case "windows":
		// PowerShell 必须始终使用双引号,转义内部双引号和反引号
		escaped := strings.ReplaceAll(s, "`", "``")
		escaped = strings.ReplaceAll(escaped, "\"", "`\"")
		return fmt.Sprintf("\"%s\"", escaped)
	default:
		// Unix 使用单引号
		if strings.ContainsAny(s, " \t\n\"'$`\\!*?[](){};<>|&~") {
			// 转义单引号: ' -> '\''
			escaped := strings.ReplaceAll(s, "'", "'\\''")
			return fmt.Sprintf("'%s'", escaped)
		}
		// 简单字符串直接返回
		return s
	}
}

// ExtractGeminiConfigFromProvider extracts Gemini configuration fields from a Provider
func ExtractGeminiConfigFromProvider(p *Provider) (baseURL, apiKey, model string) {
	if p == nil {
		return
	}

	if envMap, ok := p.SettingsConfig["env"].(map[string]interface{}); ok {
		if val, ok := envMap["GOOGLE_GEMINI_BASE_URL"].(string); ok {
			baseURL = val
		}
		if val, ok := envMap["GEMINI_API_KEY"].(string); ok {
			apiKey = val
		}
		if val, ok := envMap["GEMINI_MODEL"].(string); ok {
			model = val
		}
	}
	return
}

// AddGeminiProvider adds a new Gemini provider configuration
func (m *Manager) AddGeminiProvider(name, baseURL, apiKey, model string) error {
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

	provider := Provider{
		ID:   uuid.New().String(),
		Name: name,
		SettingsConfig: map[string]interface{}{
			"env": map[string]interface{}{
				"GOOGLE_GEMINI_BASE_URL": baseURL,
				"GEMINI_API_KEY":         apiKey,
				"GEMINI_MODEL":           model,
			},
		},
		Category:  "custom",
		CreatedAt: time.Now().UnixMilli(),
	}

	// 使用 AddProviderDirect 添加（自动处理 SortOrder）
	return m.AddProviderDirect("gemini", provider)
}

// UpdateGeminiProvider updates an existing Gemini provider configuration
func (m *Manager) UpdateGeminiProvider(oldName, newName, baseURL, apiKey, model string) error {
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

	// 更新配置
	oldProvider.Name = newName
	oldProvider.SettingsConfig = map[string]interface{}{
		"env": map[string]interface{}{
			"GOOGLE_GEMINI_BASE_URL": baseURL,
			"GEMINI_API_KEY":         apiKey,
			"GEMINI_MODEL":           model,
		},
	}

	app.Providers[targetID] = *oldProvider
	m.config.Apps["gemini"] = app

	return m.Save()
}
