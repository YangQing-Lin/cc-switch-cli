package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectGeminiAuthType(t *testing.T) {
	tests := []struct {
		name     string
		provider *Provider
		expected GeminiAuthType
	}{
		{
			name: "API Key with explicit authType",
			provider: &Provider{
				Name: "Test Provider",
				SettingsConfig: map[string]interface{}{
					"authType": "gemini-api-key",
					"env": map[string]interface{}{
						"GEMINI_API_KEY": "sk-xxx",
					},
				},
			},
			expected: GeminiAuthAPIKey,
		},
		{
			name: "OAuth with explicit authType",
			provider: &Provider{
				Name: "Google",
				SettingsConfig: map[string]interface{}{
					"authType": "oauth-personal",
					"env":      map[string]interface{}{},
				},
			},
			expected: GeminiAuthOAuth,
		},
		{
			name: "OAuth detected by name",
			provider: &Provider{
				Name: "Google Official",
				SettingsConfig: map[string]interface{}{
					"env": map[string]interface{}{},
				},
			},
			expected: GeminiAuthOAuth,
		},
		{
			name: "OAuth detected by URL",
			provider: &Provider{
				Name:       "My Provider",
				WebsiteURL: "https://ai.google.dev/",
				SettingsConfig: map[string]interface{}{
					"env": map[string]interface{}{},
				},
			},
			expected: GeminiAuthOAuth,
		},
		{
			name: "API Key without explicit authType",
			provider: &Provider{
				Name: "Custom Provider",
				SettingsConfig: map[string]interface{}{
					"env": map[string]interface{}{
						"GEMINI_API_KEY":         "sk-xxx",
						"GOOGLE_GEMINI_BASE_URL": "https://api.example.com",
					},
				},
			},
			expected: GeminiAuthAPIKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectGeminiAuthType(tt.provider)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestNormalizeGeminiEnv(t *testing.T) {
	tests := []struct {
		name     string
		provider *Provider
		expected map[string]string
	}{
		{
			name: "All standard variables",
			provider: &Provider{
				SettingsConfig: map[string]interface{}{
					"env": map[string]interface{}{
						"GEMINI_API_KEY":         "sk-xxx",
						"GOOGLE_GEMINI_BASE_URL": "https://api.example.com",
						"GEMINI_MODEL":           "gemini-2.5-pro",
					},
				},
			},
			expected: map[string]string{
				"GEMINI_API_KEY":         "sk-xxx",
				"GOOGLE_GEMINI_BASE_URL": "https://api.example.com",
				"GEMINI_MODEL":           "gemini-2.5-pro",
			},
		},
		{
			name: "Empty env",
			provider: &Provider{
				SettingsConfig: map[string]interface{}{
					"env": map[string]interface{}{},
				},
			},
			expected: map[string]string{},
		},
		{
			name: "Custom variables",
			provider: &Provider{
				SettingsConfig: map[string]interface{}{
					"env": map[string]interface{}{
						"GEMINI_API_KEY": "sk-xxx",
						"CUSTOM_VAR":     "custom-value",
					},
				},
			},
			expected: map[string]string{
				"GEMINI_API_KEY": "sk-xxx",
				"CUSTOM_VAR":     "custom-value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeGeminiEnv(tt.provider)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d variables, got %d", len(tt.expected), len(result))
			}
			for key, expectedVal := range tt.expected {
				if gotVal, ok := result[key]; !ok || gotVal != expectedVal {
					t.Errorf("key %s: expected %v, got %v", key, expectedVal, gotVal)
				}
			}
		})
	}
}

func TestWriteGeminiEnvFile(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	manager := &Manager{
		customDir: tmpDir,
	}

	envMap := map[string]string{
		"GEMINI_API_KEY":         "sk-test-xxx",
		"GOOGLE_GEMINI_BASE_URL": "https://api.test.com",
		"GEMINI_MODEL":           "gemini-2.5-pro",
	}

	err := manager.writeGeminiEnvFile(envMap)
	if err != nil {
		t.Fatalf("writeGeminiEnvFile failed: %v", err)
	}

	// 验证文件存在
	envPath := filepath.Join(tmpDir, ".gemini", ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		t.Fatalf(".env file was not created")
	}

	// 读取文件内容
	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("failed to read .env file: %v", err)
	}

	contentStr := string(content)

	// 验证内容包含所有变量
	if !strings.Contains(contentStr, "GEMINI_API_KEY=sk-test-xxx") {
		t.Errorf(".env file missing GEMINI_API_KEY")
	}
	if !strings.Contains(contentStr, "GOOGLE_GEMINI_BASE_URL=https://api.test.com") {
		t.Errorf(".env file missing GOOGLE_GEMINI_BASE_URL")
	}
	if !strings.Contains(contentStr, "GEMINI_MODEL=gemini-2.5-pro") {
		t.Errorf(".env file missing GEMINI_MODEL")
	}

	// 验证文件权限 (Unix only)
	if info, err := os.Stat(envPath); err == nil && os.Getenv("GOOS") != "windows" {
		perm := info.Mode().Perm()
		if perm != 0600 {
			t.Errorf("expected file permission 0600, got %o", perm)
		}
	}
}

func TestWriteGeminiEnvFileIncremental(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	manager := &Manager{
		customDir: tmpDir,
	}

	// 先创建一个已有的 .env 文件，包含其他配置
	envDir := filepath.Join(tmpDir, ".gemini")
	if err := os.MkdirAll(envDir, 0700); err != nil {
		t.Fatalf("failed to create .gemini dir: %v", err)
	}
	envPath := filepath.Join(envDir, ".env")

	existingContent := `# 现有配置
SOME_OTHER_VAR=existing-value
GOOGLE_GEMINI_BASE_URL=https://old-api.example.com
ANOTHER_VAR=another-value
GEMINI_API_KEY=old-key
`
	if err := os.WriteFile(envPath, []byte(existingContent), 0600); err != nil {
		t.Fatalf("failed to write existing .env file: %v", err)
	}

	// 使用新的值更新
	envMap := map[string]string{
		"GEMINI_API_KEY":         "new-api-key",
		"GOOGLE_GEMINI_BASE_URL": "https://new-api.example.com",
		"GEMINI_MODEL":           "gemini-3-pro-preview",
	}

	err := manager.writeGeminiEnvFile(envMap)
	if err != nil {
		t.Fatalf("writeGeminiEnvFile failed: %v", err)
	}

	// 读取文件内容
	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("failed to read .env file: %v", err)
	}

	contentStr := string(content)

	// 验证：新值已更新
	if !strings.Contains(contentStr, "GEMINI_API_KEY=new-api-key") {
		t.Errorf(".env file should have updated GEMINI_API_KEY, got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "GOOGLE_GEMINI_BASE_URL=https://new-api.example.com") {
		t.Errorf(".env file should have updated GOOGLE_GEMINI_BASE_URL, got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "GEMINI_MODEL=gemini-3-pro-preview") {
		t.Errorf(".env file should have added GEMINI_MODEL, got: %s", contentStr)
	}

	// 验证：其他配置被保留
	if !strings.Contains(contentStr, "SOME_OTHER_VAR=existing-value") {
		t.Errorf(".env file should preserve SOME_OTHER_VAR, got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "ANOTHER_VAR=another-value") {
		t.Errorf(".env file should preserve ANOTHER_VAR, got: %s", contentStr)
	}

	// 验证：注释被保留
	if !strings.Contains(contentStr, "# 现有配置") {
		t.Errorf(".env file should preserve comments, got: %s", contentStr)
	}
}

func TestParseEnvFile(t *testing.T) {
	content := `# Comment line
GEMINI_API_KEY=test-key
GOOGLE_GEMINI_BASE_URL=https://api.example.com

SOME_VAR=value with = sign
`
	envVars, lines := parseEnvFile(content)

	// 验证解析的变量
	if envVars["GEMINI_API_KEY"] != "test-key" {
		t.Errorf("expected GEMINI_API_KEY=test-key, got %s", envVars["GEMINI_API_KEY"])
	}
	if envVars["GOOGLE_GEMINI_BASE_URL"] != "https://api.example.com" {
		t.Errorf("expected GOOGLE_GEMINI_BASE_URL=https://api.example.com, got %s", envVars["GOOGLE_GEMINI_BASE_URL"])
	}
	if envVars["SOME_VAR"] != "value with = sign" {
		t.Errorf("expected SOME_VAR=value with = sign, got %s", envVars["SOME_VAR"])
	}

	// 验证行数
	if len(lines) != 6 { // 包含最后的空行
		t.Errorf("expected 6 lines, got %d", len(lines))
	}
}

func TestExtractGeminiConfigFromProvider(t *testing.T) {
	provider := &Provider{
		Name: "Test Provider",
		SettingsConfig: map[string]interface{}{
			"authType": "gemini-api-key",
			"env": map[string]interface{}{
				"GOOGLE_GEMINI_BASE_URL": "https://api.example.com",
				"GEMINI_API_KEY":         "sk-xxx",
				"GEMINI_MODEL":           "gemini-2.5-flash",
			},
		},
	}

	baseURL, apiKey, model, authType := ExtractGeminiConfigFromProvider(provider)

	if baseURL != "https://api.example.com" {
		t.Errorf("expected baseURL 'https://api.example.com', got '%s'", baseURL)
	}
	if apiKey != "sk-xxx" {
		t.Errorf("expected apiKey 'sk-xxx', got '%s'", apiKey)
	}
	if model != "gemini-2.5-flash" {
		t.Errorf("expected model 'gemini-2.5-flash', got '%s'", model)
	}
	if authType != GeminiAuthAPIKey {
		t.Errorf("expected authType GeminiAuthAPIKey, got %v", authType)
	}
}
