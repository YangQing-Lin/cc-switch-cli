package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
)

func TestAddDeleteUpdateShowCommands(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	apiKey = "sk-test"
	baseURL = "https://api.example.com"
	category = "custom"
	appName = "claude"

	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := addCmd.RunE(addCmd, []string{"demo"}); err != nil {
			t.Fatalf("add command: %v", err)
		}
	})
	if !strings.Contains(stdout, "demo") {
		t.Fatalf("expected add output to mention provider name, got: %s", stdout)
	}

	manager := setupManager(t)
	addClaudeProvider(t, manager, "to-delete")
	force = true
	stdout, _ = testutil.CaptureOutput(t, func() {
		if err := deleteCmd.RunE(deleteCmd, []string{"to-delete"}); err != nil {
			t.Fatalf("delete command: %v", err)
		}
	})
	if !strings.Contains(stdout, "to-delete") {
		t.Fatalf("expected delete output to mention provider name, got: %s", stdout)
	}

	manager = setupManager(t)
	addClaudeProvider(t, manager, "old")
	resetFlags(updateCmd)
	if err := updateCmd.Flags().Set("name", "new"); err != nil {
		t.Fatalf("set flag: %v", err)
	}
	if err := updateCmd.Flags().Set("apikey", "sk-new"); err != nil {
		t.Fatalf("set flag: %v", err)
	}
	if err := updateCmd.Flags().Set("base-url", "https://api.new"); err != nil {
		t.Fatalf("set flag: %v", err)
	}
	if err := updateCmd.Flags().Set("category", "custom"); err != nil {
		t.Fatalf("set flag: %v", err)
	}
	if err := updateCmd.Flags().Set("default-sonnet-model", "claude-3-opus"); err != nil {
		t.Fatalf("set flag: %v", err)
	}
	stdout, _ = testutil.CaptureOutput(t, func() {
		if err := updateCmd.RunE(updateCmd, []string{"old"}); err != nil {
			t.Fatalf("update command: %v", err)
		}
	})
	if !strings.Contains(stdout, "old") {
		t.Fatalf("expected update output to mention provider name, got: %s", stdout)
	}

	resetFlags(showCmd)
	if err := showCmd.Flags().Set("json", "true"); err != nil {
		t.Fatalf("set json flag: %v", err)
	}
	stdout, _ = testutil.CaptureOutput(t, func() {
		if err := showCmd.RunE(showCmd, []string{"new"}); err != nil {
			t.Fatalf("show command: %v", err)
		}
	})
	if !strings.Contains(stdout, "\"name\": \"new\"") {
		t.Fatalf("expected json output, got: %s", stdout)
	}

	resetFlags(showCmd)
	stdout, _ = testutil.CaptureOutput(t, func() {
		if err := showCmd.RunE(showCmd, []string{"new"}); err != nil {
			t.Fatalf("show command: %v", err)
		}
	})
	if !strings.Contains(stdout, "配置详情") {
		t.Fatalf("expected text output, got: %s", stdout)
	}
}

func TestExportAndImportCommands(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	manager := setupManager(t)
	addClaudeProvider(t, manager, "alpha")

	cases := []struct {
		name   string
		format string
	}{
		{"full", "full"},
		{"standard", "standard"},
		{"minimal", "minimal"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "export.json")
			resetFlags(exportCmd)
			if err := exportCmd.Flags().Set("output", output); err != nil {
				t.Fatalf("set output flag: %v", err)
			}
			if err := exportCmd.Flags().Set("format", tc.format); err != nil {
				t.Fatalf("set format flag: %v", err)
			}
			stdout, _ := testutil.CaptureOutput(t, func() {
				if err := exportCmd.RunE(exportCmd, []string{}); err != nil {
					t.Fatalf("export command: %v", err)
				}
			})
			if !strings.Contains(stdout, "export") && !strings.Contains(stdout, "导出") {
				t.Fatalf("expected export output, got: %s", stdout)
			}
			if _, err := os.Stat(output); err != nil {
				t.Fatalf("expected export file: %v", err)
			}
		})
	}

	importFile := filepath.Join(t.TempDir(), "import.json")
	importConfig := &config.MultiAppConfig{
		Version: 2,
		Apps: map[string]config.ProviderManager{
			"claude": {
				Providers: map[string]config.Provider{
					"import-id": {
						ID:   "import-id",
						Name: "imported",
						SettingsConfig: map[string]interface{}{
							"env": map[string]interface{}{
								"ANTHROPIC_AUTH_TOKEN": "sk-import",
								"ANTHROPIC_BASE_URL":   "https://api.import",
							},
						},
						Category: "custom",
					},
				},
			},
		},
	}
	data, err := json.Marshal(importConfig)
	if err != nil {
		t.Fatalf("marshal import: %v", err)
	}
	if err := os.WriteFile(importFile, data, 0644); err != nil {
		t.Fatalf("write import file: %v", err)
	}

	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := importCmd.RunE(importCmd, []string{importFile}); err != nil {
			t.Fatalf("import command: %v", err)
		}
	})
	if !strings.Contains(stdout, "导入完成") {
		t.Fatalf("expected import summary, got: %s", stdout)
	}
}

func TestFilterConfigAndMergeDuplicates(t *testing.T) {
	original := &config.MultiAppConfig{
		Version: 2,
		Apps: map[string]config.ProviderManager{
			"claude": {
				Providers: map[string]config.Provider{
					"id-1": {ID: "id-1", Name: "alpha"},
				},
				Current: "id-1",
			},
			"codex": {
				Providers: map[string]config.Provider{
					"id-2": {ID: "id-2", Name: "beta"},
				},
				Current: "id-2",
			},
		},
	}
	filtered := filterConfig(original, "claude", "alpha")
	if len(filtered.Apps) != 1 {
		t.Fatalf("expected filtered apps, got %d", len(filtered.Apps))
	}
	if len(filtered.Apps["claude"].Providers) != 1 {
		t.Fatalf("expected filtered providers")
	}

	mismatch := &config.MultiAppConfig{
		Version: 2,
		Apps: map[string]config.ProviderManager{
			"claude": {
				Providers: map[string]config.Provider{
					"id-1": {ID: "id-1", Name: "alpha"},
					"id-2": {ID: "id-2", Name: "beta"},
				},
				Current: "id-2",
			},
		},
	}
	filtered = filterConfig(mismatch, "claude", "alpha")
	if filtered.Apps["claude"].Current != "" {
		t.Fatalf("expected current to be empty when filtered provider is not current")
	}

	filtered = filterConfig(original, "claude", "missing")
	if len(filtered.Apps["claude"].Providers) != 0 {
		t.Fatalf("expected no providers for missing name")
	}

	filtered = filterConfig(original, "missing-app", "")
	if len(filtered.Apps) != 0 {
		t.Fatalf("expected no apps for missing app")
	}

	filtered = filterConfig(original, "", "")
	if len(filtered.Apps) != 2 {
		t.Fatalf("expected all apps, got %d", len(filtered.Apps))
	}

	filtered = filterConfig(original, "", "alpha")
	if len(filtered.Apps) == 0 {
		t.Fatalf("expected filtered apps for provider name")
	}

	minimal := convertToMinimal(&config.MultiAppConfig{
		Version: 2,
		Apps: map[string]config.ProviderManager{
			"codex": {
				Providers: map[string]config.Provider{
					"id-3": {
						ID:   "id-3",
						Name: "codex",
						SettingsConfig: map[string]interface{}{
							"base_url": "https://codex.example.com",
							"auth":     "sk-codex",
						},
						Category: "custom",
					},
				},
			},
		},
	})
	if _, ok := minimal["apps"]; !ok {
		t.Fatalf("expected minimal export apps")
	}

	resetGlobals()
	withTempHome(t)
	manager := setupManager(t)
	addClaudeProvider(t, manager, "dup1")
	addClaudeProvider(t, manager, "dup2")
	if err := manager.SwitchProviderForApp("claude", "dup2"); err != nil {
		t.Fatalf("switch provider: %v", err)
	}
	if err := manager.AddProviderDirect("codex", config.Provider{
		ID:   "codex-1",
		Name: "codex-dup1",
		SettingsConfig: map[string]interface{}{
			"auth": map[string]interface{}{
				"OPENAI_API_KEY": "sk-codex",
			},
		},
		Category: "custom",
	}); err != nil {
		t.Fatalf("add codex provider: %v", err)
	}
	if err := manager.AddProviderDirect("codex", config.Provider{
		ID:   "codex-2",
		Name: "codex-dup2",
		SettingsConfig: map[string]interface{}{
			"auth": map[string]interface{}{
				"OPENAI_API_KEY": "sk-codex",
			},
		},
		Category: "custom",
	}); err != nil {
		t.Fatalf("add codex provider: %v", err)
	}
	duplicates := findDuplicateProviders(manager, "claude")
	if len(duplicates) == 0 {
		t.Fatalf("expected duplicate providers")
	}
	stdout, _ := testutil.CaptureOutput(t, func() {
		if merged := mergeDuplicateProviders(manager, "claude", duplicates); merged == 0 {
			t.Fatalf("expected merged providers")
		}
	})
	if !strings.Contains(stdout, "删除重复配置") {
		t.Fatalf("expected merge output, got: %s", stdout)
	}
}

func TestImportFromFileSkipsDuplicates(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	manager := setupManager(t)
	addClaudeProvider(t, manager, "existing")

	importFile := filepath.Join(t.TempDir(), "import.json")
	importConfig := &config.MultiAppConfig{
		Version: 2,
		Apps: map[string]config.ProviderManager{
			"claude": {
				Providers: map[string]config.Provider{
					"dup": {
						ID:   "dup",
						Name: "dup",
						SettingsConfig: map[string]interface{}{
							"env": map[string]interface{}{
								"ANTHROPIC_AUTH_TOKEN": "sk-test",
								"ANTHROPIC_BASE_URL":   "https://api.example.com",
							},
						},
					},
				},
			},
		},
	}
	data, err := json.Marshal(importConfig)
	if err != nil {
		t.Fatalf("marshal import: %v", err)
	}
	if err := os.WriteFile(importFile, data, 0644); err != nil {
		t.Fatalf("write import file: %v", err)
	}

	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := importFromFile(manager, importFile, "claude"); err != nil {
			t.Fatalf("importFromFile: %v", err)
		}
	})
	if !strings.Contains(stdout, "跳过") {
		t.Fatalf("expected duplicate skip output, got: %s", stdout)
	}

	stdout, _ = testutil.CaptureOutput(t, func() {
		if err := importFromFile(manager, importFile, "missing-app"); err != nil {
			t.Fatalf("importFromFile: %v", err)
		}
	})
	if !strings.Contains(stdout, "导入完成") {
		t.Fatalf("expected import summary, got: %s", stdout)
	}
}

func TestImportFromFileMissing(t *testing.T) {
	resetGlobals()
	withTempHome(t)
	manager := setupManager(t)
	if err := importFromFile(manager, filepath.Join(t.TempDir(), "missing.json"), "claude"); err == nil {
		t.Fatalf("expected missing file error")
	}
}

func TestValidateHelpers(t *testing.T) {
	issues, warnings := validateApp("claude", config.ProviderManager{Providers: map[string]config.Provider{}}, "")
	if len(issues) != 0 || len(warnings) == 0 {
		t.Fatalf("expected warning for empty app config")
	}

	appConfig := config.ProviderManager{
		Providers: map[string]config.Provider{
			"missing-id": {
				ID:             "",
				Name:           "dup",
				SettingsConfig: nil,
			},
			"dup-id": {
				ID:   "dup-id",
				Name: "dup",
				SettingsConfig: map[string]interface{}{
					"env": map[string]interface{}{
						"ANTHROPIC_AUTH_TOKEN": "invalid",
						"ANTHROPIC_BASE_URL":   "http://example.com",
						"CLAUDE_CODE_MODEL":    "bad-model",
					},
				},
			},
		},
		Current: "missing-current",
	}
	issues, warnings = validateApp("claude", appConfig, "")
	if len(issues) == 0 || len(warnings) == 0 {
		t.Fatalf("expected issues and warnings for invalid app config")
	}

	provider := config.Provider{
		ID:   "id",
		Name: "bad",
		SettingsConfig: map[string]interface{}{
			"env": map[string]interface{}{
				"ANTHROPIC_AUTH_TOKEN": "token",
				"ANTHROPIC_BASE_URL":   "://bad-url",
				"CLAUDE_CODE_MODEL":    "unknown-model",
			},
		},
	}
	var issueList []ValidationIssue
	var warningList []ValidationIssue
	validateClaudeProvider("claude", provider, &issueList, &warningList)
	if len(issueList) == 0 {
		t.Fatalf("expected invalid base url issue")
	}
	if len(warningList) == 0 {
		t.Fatalf("expected token/model warnings")
	}

	var missingSettingsIssues []ValidationIssue
	var missingSettingsWarnings []ValidationIssue
	validateClaudeProvider("claude", config.Provider{ID: "id", Name: "none"}, &missingSettingsIssues, &missingSettingsWarnings)
	if len(missingSettingsIssues) == 0 {
		t.Fatalf("expected missing settings issue")
	}

	var missingEnvIssues []ValidationIssue
	var missingEnvWarnings []ValidationIssue
	validateClaudeProvider("claude", config.Provider{ID: "id", Name: "env", SettingsConfig: map[string]interface{}{}}, &missingEnvIssues, &missingEnvWarnings)
	if len(missingEnvIssues) == 0 {
		t.Fatalf("expected missing env issue")
	}

	var missingTokenIssues []ValidationIssue
	var missingTokenWarnings []ValidationIssue
	validateClaudeProvider("claude", config.Provider{ID: "id", Name: "token", SettingsConfig: map[string]interface{}{
		"env": map[string]interface{}{},
	}}, &missingTokenIssues, &missingTokenWarnings)
	if len(missingTokenIssues) == 0 || len(missingTokenWarnings) == 0 {
		t.Fatalf("expected missing token issue and base URL warning")
	}

	stdout, _ := testutil.CaptureOutput(t, func() {
		displayIssue(1, ValidationIssue{Level: "WARNING", App: "claude", Provider: "bad", Message: "warn", FixableMsg: "fix"}, true)
	})
	if !strings.Contains(stdout, "warn") {
		t.Fatalf("expected displayIssue output, got: %s", stdout)
	}

	if count := attemptFixes(nil, nil); count != 0 {
		t.Fatalf("expected no fixes, got %d", count)
	}

	validateCodexProvider("codex", config.Provider{}, &issueList, &warningList)
}

func TestValidateCommandFailure(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	cfg := &config.MultiAppConfig{
		Version: 2,
		Apps: map[string]config.ProviderManager{
			"claude": {
				Providers: map[string]config.Provider{
					"id-1": {
						ID:             "",
						Name:           "",
						SettingsConfig: nil,
					},
				},
				Current: "missing-current",
			},
		},
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	configPath, err := config.GetConfigPath()
	if err != nil {
		t.Fatalf("get config path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	resetFlags(validateCmd)
	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := validateCmd.RunE(validateCmd, []string{}); err == nil {
			t.Fatalf("expected validation error")
		}
	})
	if !strings.Contains(stdout, "发现") {
		t.Fatalf("expected validation output, got: %s", stdout)
	}
}

func TestSettingsCommand(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	setSetting = "language=en"
	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := runSettings([]string{}); err != nil {
			t.Fatalf("set setting: %v", err)
		}
	})
	if !strings.Contains(stdout, "language") && !strings.Contains(stdout, "语言") {
		t.Fatalf("expected set output, got: %s", stdout)
	}

	setSetting = ""
	getSetting = true
	stdout, _ = testutil.CaptureOutput(t, func() {
		if err := runSettings([]string{"language"}); err != nil {
			t.Fatalf("get setting: %v", err)
		}
	})
	if strings.TrimSpace(stdout) == "" {
		t.Fatalf("expected language output")
	}

	setSetting = "configDir=/tmp/config"
	stdout, _ = testutil.CaptureOutput(t, func() {
		if err := runSettings([]string{}); err != nil {
			t.Fatalf("set configDir: %v", err)
		}
	})
	if !strings.Contains(stdout, "配置目录") {
		t.Fatalf("expected configDir output, got: %s", stdout)
	}

	getSetting = false
	setSetting = ""
	stdout, _ = testutil.CaptureOutput(t, func() {
		if err := runSettings([]string{}); err != nil {
			t.Fatalf("list settings: %v", err)
		}
	})
	if !strings.Contains(stdout, "应用设置") {
		t.Fatalf("expected settings output, got: %s", stdout)
	}

	getSetting = true
	if err := runSettings([]string{}); err == nil {
		t.Fatalf("expected get setting error")
	}
	if err := runSettings([]string{"unknown"}); err == nil {
		t.Fatalf("expected unknown get key error")
	}
	getSetting = false

	setSetting = "unknown=1"
	if err := runSettings([]string{}); err == nil {
		t.Fatalf("expected unknown key error")
	}

	getSetting = false
	setSetting = "badformat"
	if err := runSettings([]string{}); err == nil {
		t.Fatalf("expected format error")
	}
}

func TestPromptHelpers(t *testing.T) {
	resetGlobals()
	withStdin(t, "input-value\n", func() {
		got, err := promptInput("prompt: ")
		if err != nil {
			t.Fatalf("promptInput: %v", err)
		}
		if got != "input-value" {
			t.Fatalf("expected trimmed input, got %q", got)
		}
	})
	withStdin(t, "", func() {
		if _, err := promptInput("prompt: "); err == nil {
			t.Fatalf("expected promptInput error on empty input")
		}
	})

	withStdin(t, "secret\n", func() {
		got, err := promptSecret("prompt: ")
		if err != nil {
			t.Fatalf("promptSecret: %v", err)
		}
		if got != "secret" {
			t.Fatalf("expected secret input, got %q", got)
		}
	})

	origReadPassword := readPassword
	readPassword = func(fd int) ([]byte, error) {
		return []byte("typed-secret"), nil
	}
	t.Cleanup(func() { readPassword = origReadPassword })

	got, err := promptSecret("prompt: ")
	if err != nil {
		t.Fatalf("promptSecret readPassword: %v", err)
	}
	if got != "typed-secret" {
		t.Fatalf("expected readPassword secret, got %q", got)
	}
}

func TestMigrateClaudeConfig(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	t.Run("rename", func(t *testing.T) {
		dir := t.TempDir()
		oldPath := filepath.Join(dir, "claude.json")
		newPath := filepath.Join(dir, "settings.json")
		if err := os.WriteFile(oldPath, []byte(`{"env":{"ANTHROPIC_AUTH_TOKEN":"sk-old"}}`), 0644); err != nil {
			t.Fatalf("write old config: %v", err)
		}
		if err := migrateClaudeConfig(oldPath, newPath); err != nil {
			t.Fatalf("migrateClaudeConfig: %v", err)
		}
		if _, err := os.Stat(newPath); err != nil {
			t.Fatalf("expected new config: %v", err)
		}
	})

	t.Run("merge", func(t *testing.T) {
		dir := t.TempDir()
		oldPath := filepath.Join(dir, "claude.json")
		newPath := filepath.Join(dir, "settings.json")
		if err := os.WriteFile(oldPath, []byte(`{"env":{"ANTHROPIC_AUTH_TOKEN":"sk-old","ANTHROPIC_BASE_URL":"https://old.example.com","CLAUDE_CODE_MODEL":"claude-3"}}`), 0644); err != nil {
			t.Fatalf("write old config: %v", err)
		}
		if err := os.WriteFile(newPath, []byte(`{"env":{}}`), 0644); err != nil {
			t.Fatalf("write new config: %v", err)
		}
		if err := migrateClaudeConfig(oldPath, newPath); err != nil {
			t.Fatalf("migrateClaudeConfig: %v", err)
		}
		if _, err := os.Stat(oldPath + ".backup"); err != nil {
			t.Fatalf("expected backup: %v", err)
		}
	})

	t.Run("invalid old config", func(t *testing.T) {
		dir := t.TempDir()
		oldPath := filepath.Join(dir, "claude.json")
		newPath := filepath.Join(dir, "settings.json")
		if err := os.WriteFile(oldPath, []byte("{bad json"), 0644); err != nil {
			t.Fatalf("write old config: %v", err)
		}
		if err := os.WriteFile(newPath, []byte(`{"env":{}}`), 0644); err != nil {
			t.Fatalf("write new config: %v", err)
		}
		if err := migrateClaudeConfig(oldPath, newPath); err == nil {
			t.Fatalf("expected migrate error")
		}
	})

	t.Run("invalid new config", func(t *testing.T) {
		dir := t.TempDir()
		oldPath := filepath.Join(dir, "claude.json")
		newPath := filepath.Join(dir, "settings.json")
		if err := os.WriteFile(oldPath, []byte(`{"env":{"ANTHROPIC_AUTH_TOKEN":"sk-old"}}`), 0644); err != nil {
			t.Fatalf("write old config: %v", err)
		}
		if err := os.WriteFile(newPath, []byte("{bad json"), 0644); err != nil {
			t.Fatalf("write new config: %v", err)
		}
		if err := migrateClaudeConfig(oldPath, newPath); err == nil {
			t.Fatalf("expected migrate error")
		}
	})
}

func TestMigrateExecute(t *testing.T) {
	resetGlobals()
	home := withTempHome(t)

	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("mkdir claude dir: %v", err)
	}
	oldPath := filepath.Join(claudeDir, "claude.json")
	newPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(oldPath, []byte(`{"env":{"ANTHROPIC_AUTH_TOKEN":"sk-old"}}`), 0644); err != nil {
		t.Fatalf("write old config: %v", err)
	}
	if err := os.WriteFile(newPath, []byte(`{"env":{}}`), 0644); err != nil {
		t.Fatalf("write new config: %v", err)
	}

	manager := setupManager(t)
	addClaudeProvider(t, manager, "dup1")
	addClaudeProvider(t, manager, "dup2")
	if err := manager.SwitchProviderForApp("claude", "dup2"); err != nil {
		t.Fatalf("switch provider: %v", err)
	}

	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := runMigrate(true, false); err != nil {
			t.Fatalf("runMigrate: %v", err)
		}
	})
	if !strings.Contains(stdout, "迁移完成") {
		t.Fatalf("expected migrate output, got: %s", stdout)
	}
	if _, err := os.Stat(oldPath + ".backup"); err != nil {
		t.Fatalf("expected old config backup: %v", err)
	}
}

func TestMigrateNoop(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := runMigrate(true, false); err != nil {
			t.Fatalf("runMigrate: %v", err)
		}
	})
	if !strings.Contains(stdout, "没有需要迁移") {
		t.Fatalf("expected noop output, got: %s", stdout)
	}
}

func TestMigrateCancelled(t *testing.T) {
	resetGlobals()
	home := withTempHome(t)

	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("mkdir claude dir: %v", err)
	}
	oldPath := filepath.Join(claudeDir, "claude.json")
	if err := os.WriteFile(oldPath, []byte(`{"env":{"ANTHROPIC_AUTH_TOKEN":"sk-old"}}`), 0644); err != nil {
		t.Fatalf("write old config: %v", err)
	}

	withStdin(t, "n\n", func() {
		stdout, _ := testutil.CaptureOutput(t, func() {
			if err := runMigrate(false, false); err != nil {
				t.Fatalf("runMigrate: %v", err)
			}
		})
		if !strings.Contains(stdout, "迁移已取消") {
			t.Fatalf("expected cancel output, got: %s", stdout)
		}
	})
}

func TestConfigDirCommand(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	setConfigDir = "/tmp/custom"
	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := runConfigDir(); err != nil {
			t.Fatalf("runConfigDir: %v", err)
		}
	})
	if !strings.Contains(stdout, "CC_SWITCH_DIR") {
		t.Fatalf("expected hint output, got: %s", stdout)
	}

	setConfigDir = ""
	stdout, _ = testutil.CaptureOutput(t, func() {
		if err := runConfigDir(); err != nil {
			t.Fatalf("runConfigDir: %v", err)
		}
	})
	if !strings.Contains(stdout, "配置目录") {
		t.Fatalf("expected config dir output, got: %s", stdout)
	}
}

func TestMigrateDryRun(t *testing.T) {
	resetGlobals()
	home := withTempHome(t)

	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	oldPath := filepath.Join(claudeDir, "claude.json")
	if err := os.WriteFile(oldPath, []byte(`{"env":{"ANTHROPIC_AUTH_TOKEN":"sk-test"}}`), 0644); err != nil {
		t.Fatalf("write old config: %v", err)
	}

	manager := setupManager(t)
	addClaudeProvider(t, manager, "dup1")
	addClaudeProvider(t, manager, "dup2")
	if err := manager.Save(); err != nil {
		t.Fatalf("save manager: %v", err)
	}

	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := runMigrate(true, true); err != nil {
			t.Fatalf("runMigrate: %v", err)
		}
	})
	if !strings.Contains(stdout, "模拟") {
		t.Fatalf("expected dry-run output, got: %s", stdout)
	}
}

func TestCodexCommands(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	resetFlags(codexAddCmd)
	if err := codexAddCmd.Flags().Set("apikey", "sk-codex"); err != nil {
		t.Fatalf("set flag: %v", err)
	}
	if err := codexAddCmd.Flags().Set("base-url", "https://codex.example.com"); err != nil {
		t.Fatalf("set flag: %v", err)
	}
	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := codexAddCmd.RunE(codexAddCmd, []string{"codex-one"}); err != nil {
			t.Fatalf("codex add: %v", err)
		}
	})
	if !strings.Contains(stdout, "codex-one") {
		t.Fatalf("expected add output, got: %s", stdout)
	}

	manager := setupManager(t)
	addCodexProvider(t, manager, "codex-two", true)

	stdout, _ = testutil.CaptureOutput(t, func() {
		if err := codexListCmd.RunE(codexListCmd, []string{}); err != nil {
			t.Fatalf("codex list: %v", err)
		}
	})
	if !strings.Contains(stdout, "codex-two") {
		t.Fatalf("expected list output, got: %s", stdout)
	}

	stdout, _ = testutil.CaptureOutput(t, func() {
		if err := codexSwitchCmd.RunE(codexSwitchCmd, []string{"codex-two"}); err != nil {
			t.Fatalf("codex switch: %v", err)
		}
	})
	if !strings.Contains(stdout, "codex-two") {
		t.Fatalf("expected switch output, got: %s", stdout)
	}

	resetFlags(codexUpdateCmd)
	if err := codexUpdateCmd.Flags().Set("apikey", "sk-updated"); err != nil {
		t.Fatalf("set flag: %v", err)
	}
	if err := codexUpdateCmd.Flags().Set("base-url", "https://codex.updated"); err != nil {
		t.Fatalf("set flag: %v", err)
	}
	if err := codexUpdateCmd.Flags().Set("model", "codex-model"); err != nil {
		t.Fatalf("set flag: %v", err)
	}
	stdout, _ = testutil.CaptureOutput(t, func() {
		if err := codexUpdateCmd.RunE(codexUpdateCmd, []string{"codex-two"}); err != nil {
			t.Fatalf("codex update: %v", err)
		}
	})
	if !strings.Contains(stdout, "codex-two") {
		t.Fatalf("expected update output, got: %s", stdout)
	}

	manager = setupManager(t)
	if err := manager.SwitchProviderForApp("codex", "codex-one"); err != nil {
		t.Fatalf("switch codex provider: %v", err)
	}

	resetFlags(codexDeleteCmd)
	if err := codexDeleteCmd.Flags().Set("force", "true"); err != nil {
		t.Fatalf("set flag: %v", err)
	}
	stdout, _ = testutil.CaptureOutput(t, func() {
		if err := codexDeleteCmd.RunE(codexDeleteCmd, []string{"codex-two"}); err != nil {
			t.Fatalf("codex delete: %v", err)
		}
	})
	if !strings.Contains(stdout, "codex-two") {
		t.Fatalf("expected delete output, got: %s", stdout)
	}
}

func TestGeminiCommands(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	resetFlags(geminiAddCmd)
	if err := geminiAddCmd.Flags().Set("apikey", "sk-gemini"); err != nil {
		t.Fatalf("set flag: %v", err)
	}
	if err := geminiAddCmd.Flags().Set("base-url", "https://gemini.example.com"); err != nil {
		t.Fatalf("set flag: %v", err)
	}
	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := geminiAddCmd.RunE(geminiAddCmd, []string{"gemini-one"}); err != nil {
			t.Fatalf("gemini add: %v", err)
		}
	})
	if !strings.Contains(stdout, "gemini-one") {
		t.Fatalf("expected add output, got: %s", stdout)
	}

	manager := setupManager(t)
	addGeminiProvider(t, manager, "gemini-two")

	stdout, _ = testutil.CaptureOutput(t, func() {
		if err := geminiListCmd.RunE(geminiListCmd, []string{}); err != nil {
			t.Fatalf("gemini list: %v", err)
		}
	})
	if !strings.Contains(stdout, "gemini-two") {
		t.Fatalf("expected list output, got: %s", stdout)
	}

	stdout, _ = testutil.CaptureOutput(t, func() {
		if err := geminiSwitchCmd.RunE(geminiSwitchCmd, []string{"gemini-two"}); err != nil {
			t.Fatalf("gemini switch: %v", err)
		}
	})
	if !strings.Contains(stdout, "gemini-two") {
		t.Fatalf("expected switch output, got: %s", stdout)
	}

	stdout, _ = testutil.CaptureOutput(t, func() {
		if err := geminiEnvCmd.RunE(geminiEnvCmd, []string{}); err != nil {
			t.Fatalf("gemini env: %v", err)
		}
	})
	if !strings.Contains(stdout, "Gemini CLI") {
		t.Fatalf("expected env output, got: %s", stdout)
	}

	manager = setupManager(t)
	if err := manager.SwitchProviderForApp("gemini", "gemini-one"); err != nil {
		t.Fatalf("switch gemini provider: %v", err)
	}

	withStdin(t, "y\n", func() {
		stdout, _ = testutil.CaptureOutput(t, func() {
			if err := geminiDeleteCmd.RunE(geminiDeleteCmd, []string{"gemini-two"}); err != nil {
				t.Fatalf("gemini delete: %v", err)
			}
		})
	})
	if !strings.Contains(stdout, "gemini-two") {
		t.Fatalf("expected delete output, got: %s", stdout)
	}

	stdout, _ = testutil.CaptureOutput(t, func() {
		if err := geminiCmd.RunE(geminiCmd, []string{}); err != nil {
			t.Fatalf("gemini help: %v", err)
		}
	})
	if !strings.Contains(stdout, "Gemini") {
		t.Fatalf("expected help output, got: %s", stdout)
	}
}
