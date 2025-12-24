package tui

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/template"
)

func newTestManager(t *testing.T) *config.Manager {
	t.Helper()
	manager, err := config.NewManagerWithDir(t.TempDir())
	if err != nil {
		t.Fatalf("NewManagerWithDir() error = %v", err)
	}
	return manager
}

func newTestTemplateManager(t *testing.T) *template.TemplateManager {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "templates.json")
	tm, err := template.NewTemplateManager(configPath)
	if err != nil {
		t.Fatalf("NewTemplateManager() error = %v", err)
	}
	return tm
}

func addProvider(t *testing.T, manager *config.Manager, appName, id, name string) config.Provider {
	t.Helper()
	provider := config.Provider{
		ID:             id,
		Name:           name,
		Category:       "custom",
		CreatedAt:      time.Now().UnixMilli(),
		SettingsConfig: map[string]interface{}{},
	}
	switch appName {
	case "claude":
		provider.SettingsConfig["env"] = map[string]interface{}{
			"ANTHROPIC_AUTH_TOKEN":           "token",
			"ANTHROPIC_BASE_URL":             "https://api.anthropic.com",
			"ANTHROPIC_MODEL":                "claude-3",
			"ANTHROPIC_DEFAULT_HAIKU_MODEL":  "haiku",
			"ANTHROPIC_DEFAULT_SONNET_MODEL": "sonnet",
			"ANTHROPIC_DEFAULT_OPUS_MODEL":   "opus",
		}
	case "codex":
		provider.SettingsConfig["auth"] = map[string]interface{}{
			"OPENAI_API_KEY": "test-api-key-12345",
		}
		provider.SettingsConfig["config"] = "model = \"gpt-5\"\nmodel_provider = \"openai\"\nmodel_reasoning_effort = \"high\"\n"
	case "gemini":
		provider.SettingsConfig["env"] = map[string]interface{}{
			"GEMINI_API_KEY":         "gemini-key",
			"GOOGLE_GEMINI_BASE_URL": "https://generativelanguage.googleapis.com",
			"GEMINI_MODEL":           "gemini-2.5-pro",
		}
		provider.SettingsConfig["authType"] = string(config.GeminiAuthAPIKey)
	}

	if err := manager.AddProviderDirect(appName, provider); err != nil {
		t.Fatalf("AddProviderDirect(%s) error = %v", appName, err)
	}

	return provider
}

func addMcpServer(t *testing.T, manager *config.Manager, id string, apps config.McpApps) config.McpServer {
	t.Helper()
	server := config.McpServer{
		ID:          id,
		Name:        "Server " + id,
		Description: "desc",
		Apps:        apps,
		Server: map[string]interface{}{
			"type":    "stdio",
			"command": "npx",
			"args":    []interface{}{"mcp-server-fetch"},
		},
	}
	if err := manager.AddMcpServer(server); err != nil {
		t.Fatalf("AddMcpServer() error = %v", err)
	}
	if err := manager.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	return server
}
