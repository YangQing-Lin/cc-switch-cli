package config

import (
	"encoding/json"
)

// Provider 表示单个供应商配置（与 cc-switch 完全一致）
type Provider struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	SettingsConfig map[string]interface{} `json:"settingsConfig"` // 完整的配置 JSON
	WebsiteURL     string                 `json:"websiteUrl,omitempty"`
	Category       string                 `json:"category,omitempty"`
	CreatedAt      int64                  `json:"createdAt,omitempty"` // 毫秒时间戳
}

// ProviderManager 管理单个应用的所有供应商（与 cc-switch 完全一致）
type ProviderManager struct {
	Providers map[string]Provider `json:"providers"` // id -> Provider
	Current   string              `json:"current"`   // 当前激活的供应商 ID
}

// MultiAppConfig 根配置文件结构（与 cc-switch 完全一致）
type MultiAppConfig struct {
	Version int                         `json:"version"`         // 配置版本（当前为 2）
	Apps    map[string]ProviderManager  `json:"-"`               // 使用 flatten 方式
	Claude  ProviderManager             `json:"claude"`          // Claude 应用
	Codex   ProviderManager             `json:"codex"`           // Codex 应用
}

// ClaudeEnv Claude 环境变量配置
type ClaudeEnv struct {
	AnthropicAuthToken  string `json:"ANTHROPIC_AUTH_TOKEN,omitempty"`
	AnthropicBaseURL    string `json:"ANTHROPIC_BASE_URL,omitempty"`
	ClaudeCodeModel     string `json:"CLAUDE_CODE_MODEL,omitempty"`
	ClaudeCodeMaxTokens string `json:"CLAUDE_CODE_MAX_TOKENS,omitempty"`
}

// ClaudePermissions Claude 权限配置
type ClaudePermissions struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny"`
}

// ClaudeSettings Claude 设置文件结构
type ClaudeSettings struct {
	Env         ClaudeEnv               `json:"env"`
	Permissions ClaudePermissions       `json:"permissions"`
	Model       string                  `json:"model,omitempty"`
	StatusLine  json.RawMessage         `json:"statusLine,omitempty"`
	Extra       map[string]interface{}  `json:"-"`
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
