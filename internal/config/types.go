package config

import (
	"encoding/json"
)

// CustomEndpoint 自定义端点配置（与 GUI v3.5.0+ 兼容）
type CustomEndpoint struct {
	URL      string `json:"url"`
	AddedAt  int64  `json:"addedAt"`
	LastUsed *int64 `json:"lastUsed,omitempty"`
}

// ProviderMeta 供应商元数据（与 GUI v3.5.0+ 兼容）
// 注意：此字段不写入 live 配置文件（如 ~/.config/claude_code_ext/settings.json），
// 仅保存在 ~/.cc-switch/config-cli.json 中，用于在 GUI 和 CLI 之间共享自定义端点等元数据
type ProviderMeta struct {
	CustomEndpoints map[string]CustomEndpoint `json:"custom_endpoints,omitempty"`
}

// Provider 表示单个供应商配置（与 cc-switch 完全一致）
type Provider struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	SettingsConfig map[string]interface{} `json:"settingsConfig"` // 完整的配置 JSON
	WebsiteURL     string                 `json:"websiteUrl,omitempty"`
	Category       string                 `json:"category,omitempty"`
	CreatedAt      int64                  `json:"createdAt,omitempty"` // 毫秒时间戳
	SortOrder      int                    `json:"sortOrder,omitempty"` // 列表排序序号，值越小越靠前
}

// ProviderManager 管理单个应用的所有供应商（与 cc-switch 完全一致）
type ProviderManager struct {
	Providers map[string]Provider `json:"providers"` // id -> Provider
	Current   string              `json:"current"`   // 当前激活的供应商 ID
}

// MultiAppConfig 根配置文件结构（v2 格式，与 cc-switch 完全一致）
// 注意：v2 格式将 apps 展平到顶层，而不是嵌套在 "apps" 键下
type MultiAppConfig struct {
	Version int                        `json:"version"`       // 配置版本（当前为 2）
	Apps    map[string]ProviderManager `json:"-"`             // 应用名称 -> ProviderManager (展平到顶层)
	Mcp     *McpRoot                   `json:"mcp,omitempty"` // MCP 配置
}

// OldMultiAppConfig 旧版配置文件结构（v2-old 格式，apps 嵌套在 "apps" 键下）
// 用于向后兼容
type OldMultiAppConfig struct {
	Version int                        `json:"version"` // 配置版本（当前为 2）
	Apps    map[string]ProviderManager `json:"apps"`    // 应用名称 -> ProviderManager
}

// MarshalJSON 自定义序列化，将 Apps 展平到顶层
func (c *MultiAppConfig) MarshalJSON() ([]byte, error) {
	result := make(map[string]interface{})
	result["version"] = c.Version

	// 将 Apps map 中的每个应用展平到顶层
	for appName, appManager := range c.Apps {
		result[appName] = appManager
	}

	// 添加 MCP 配置
	if c.Mcp != nil {
		result["mcp"] = c.Mcp
	}

	return json.Marshal(result)
}

// UnmarshalJSON 自定义反序列化，从顶层读取应用配置
func (c *MultiAppConfig) UnmarshalJSON(data []byte) error {
	// 先解析到 map
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// 提取 version
	if versionData, ok := raw["version"]; ok {
		if err := json.Unmarshal(versionData, &c.Version); err != nil {
			return err
		}
	}

	// 提取 MCP 配置
	if mcpData, ok := raw["mcp"]; ok {
		var mcpRoot McpRoot
		if err := json.Unmarshal(mcpData, &mcpRoot); err != nil {
			return err
		}
		c.Mcp = &mcpRoot
	}

	// 初始化 Apps map
	c.Apps = make(map[string]ProviderManager)

	// 已知的应用类型（非应用字段）
	knownFields := map[string]bool{
		"version": true,
		"mcp":     true,
	}

	// 提取所有应用配置（除了 version 和 mcp 之外的字段都视为应用）
	for key, rawData := range raw {
		if !knownFields[key] {
			var manager ProviderManager
			if err := json.Unmarshal(rawData, &manager); err != nil {
				return err
			}
			c.Apps[key] = manager
		}
	}

	return nil
}

// ClaudeEnv Claude 环境变量配置
type ClaudeEnv struct {
	AnthropicAuthToken          string `json:"ANTHROPIC_AUTH_TOKEN,omitempty"`
	AnthropicBaseURL            string `json:"ANTHROPIC_BASE_URL,omitempty"`
	AnthropicModel              string `json:"ANTHROPIC_MODEL,omitempty"`
	AnthropicDefaultHaikuModel  string `json:"ANTHROPIC_DEFAULT_HAIKU_MODEL,omitempty"`
	AnthropicDefaultSonnetModel string `json:"ANTHROPIC_DEFAULT_SONNET_MODEL,omitempty"`
	AnthropicDefaultOpusModel   string `json:"ANTHROPIC_DEFAULT_OPUS_MODEL,omitempty"`
	ClaudeCodeModel             string `json:"CLAUDE_CODE_MODEL,omitempty"`
	ClaudeCodeMaxTokens         string `json:"CLAUDE_CODE_MAX_TOKENS,omitempty"`
}

// ClaudePermissions Claude 权限配置
type ClaudePermissions struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny"`
}

// ClaudeSettings Claude 设置文件结构
type ClaudeSettings struct {
	Env         ClaudeEnv              `json:"env"`
	Permissions ClaudePermissions      `json:"permissions"`
	Model       string                 `json:"model,omitempty"`
	StatusLine  json.RawMessage        `json:"statusLine,omitempty"`
	Extra       map[string]interface{} `json:"-"`
}

// UnmarshalJSON 自定义反序列化，保存未知字段
func (c *ClaudeSettings) UnmarshalJSON(data []byte) error {
	// 1. 先解析到已知结构
	type Alias ClaudeSettings
	aux := &struct{ *Alias }{Alias: (*Alias)(c)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// 2. 解析到 map 获取所有字段
	var allFields map[string]interface{}
	if err := json.Unmarshal(data, &allFields); err != nil {
		return err
	}

	// 3. 保存未知字段
	c.Extra = make(map[string]interface{})
	knownFields := map[string]bool{
		"env": true, "permissions": true, "model": true, "statusLine": true,
	}
	for k, v := range allFields {
		if !knownFields[k] {
			c.Extra[k] = v
		}
	}

	return nil
}

// MarshalJSON 自定义序列化，合并未知字段
func (c *ClaudeSettings) MarshalJSON() ([]byte, error) {
	result := make(map[string]interface{})

	// 1. 先添加未知字段
	for k, v := range c.Extra {
		result[k] = v
	}

	// 2. 添加已知字段（覆盖同名字段）
	type Alias ClaudeSettings
	data, _ := json.Marshal((*Alias)(c))
	var tempMap map[string]interface{}
	json.Unmarshal(data, &tempMap)
	for k, v := range tempMap {
		result[k] = v
	}

	return json.Marshal(result)
}

// CodexAuthJson Codex auth.json 文件结构
type CodexAuthJson struct {
	OpenAIAPIKey string `json:"OPENAI_API_KEY"`
}

// GeminiAuthType Gemini 认证类型
type GeminiAuthType string

const (
	GeminiAuthAPIKey GeminiAuthType = "gemini-api-key" // API Key 认证
	GeminiAuthOAuth  GeminiAuthType = "oauth-personal" // OAuth 认证
)

// GeminiSecurityAuth Gemini 认证配置
type GeminiSecurityAuth struct {
	SelectedType GeminiAuthType `json:"selectedType"`
}

// GeminiSecurity Gemini 安全配置
type GeminiSecurity struct {
	Auth GeminiSecurityAuth `json:"auth"`
}

// GeminiSettings Gemini settings.json 文件结构
type GeminiSettings struct {
	Security   GeminiSecurity         `json:"security"`
	MCPServers map[string]interface{} `json:"mcpServers,omitempty"`
	Extra      map[string]interface{} `json:"-"` // 保存未知字段
}

// McpApps MCP 应用启用状态
type McpApps struct {
	Claude bool `json:"claude"`
	Codex  bool `json:"codex"`
	Gemini bool `json:"gemini"`
}

// McpServer MCP 服务器定义
type McpServer struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Server      map[string]interface{} `json:"server"` // 连接配置（type, command, args, url 等）
	Apps        McpApps                `json:"apps"`
	Description string                 `json:"description,omitempty"`
	Homepage    string                 `json:"homepage,omitempty"`
	Docs        string                 `json:"docs,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
}

// McpRoot MCP 根配置
type McpRoot struct {
	Servers map[string]McpServer `json:"servers"` // id -> McpServer
}

// UnmarshalJSON 自定义反序列化，保存未知字段
func (g *GeminiSettings) UnmarshalJSON(data []byte) error {
	// 1. 先解析到已知结构
	type Alias GeminiSettings
	aux := &struct{ *Alias }{Alias: (*Alias)(g)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// 2. 解析到 map 获取所有字段
	var allFields map[string]interface{}
	if err := json.Unmarshal(data, &allFields); err != nil {
		return err
	}

	// 3. 保存未知字段
	g.Extra = make(map[string]interface{})
	knownFields := map[string]bool{
		"security": true, "mcpServers": true,
	}
	for k, v := range allFields {
		if !knownFields[k] {
			g.Extra[k] = v
		}
	}

	return nil
}

// MarshalJSON 自定义序列化，合并未知字段
func (g *GeminiSettings) MarshalJSON() ([]byte, error) {
	result := make(map[string]interface{})

	// 1. 先添加未知字段
	for k, v := range g.Extra {
		result[k] = v
	}

	// 2. 添加已知字段（覆盖同名字段）
	type Alias GeminiSettings
	data, _ := json.Marshal((*Alias)(g))
	var tempMap map[string]interface{}
	json.Unmarshal(data, &tempMap)
	for k, v := range tempMap {
		result[k] = v
	}

	return json.Marshal(result)
}
