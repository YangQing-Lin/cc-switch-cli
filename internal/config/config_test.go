package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

func setTempHome(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}

func readJSONMap(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal json %s: %v", path, err)
	}
	return out
}

func TestManagerLoadScenarios(t *testing.T) {
	tests := []struct {
		name             string
		setup            func(t *testing.T, dir, path string)
		wantFile         bool
		wantProviderName string
		wantArchive      bool
		verify           func(t *testing.T, dir, path string, m *Manager)
	}{
		{
			name:     "no existing file",
			wantFile: false,
		},
		{
			name:     "empty file",
			wantFile: true,
			setup: func(t *testing.T, dir, path string) {
				writeFile(t, path, []byte(""))
			},
			verify: func(t *testing.T, dir, path string, m *Manager) {
				if got := readJSONMap(t, path)["version"]; got != float64(2) {
					t.Fatalf("version = %v, want 2", got)
				}
			},
		},
		{
			name:     "empty object",
			wantFile: true,
			setup: func(t *testing.T, dir, path string) {
				writeFile(t, path, []byte("{}"))
			},
			verify: func(t *testing.T, dir, path string, m *Manager) {
				if got := readJSONMap(t, path)["version"]; got != float64(2) {
					t.Fatalf("version = %v, want 2", got)
				}
			},
		},
		{
			name:     "corrupted json",
			wantFile: true,
			setup: func(t *testing.T, dir, path string) {
				writeFile(t, path, []byte("{invalid"))
			},
			verify: func(t *testing.T, dir, path string, m *Manager) {
				if got := readJSONMap(t, path)["version"]; got != float64(2) {
					t.Fatalf("version = %v, want 2", got)
				}
			},
		},
		{
			name:             "migrate v1 config",
			wantFile:         true,
			wantProviderName: "Legacy",
			setup: func(t *testing.T, dir, path string) {
				v1 := ProviderManager{
					Providers: map[string]Provider{
						"p1": {
							ID:   "p1",
							Name: "Legacy",
							SettingsConfig: map[string]interface{}{
								"env": map[string]interface{}{
									"ANTHROPIC_AUTH_TOKEN": "sk-old",
									"ANTHROPIC_BASE_URL":   "https://api.example.com",
								},
							},
						},
					},
					Current: "p1",
				}
				data, err := json.MarshalIndent(v1, "", "  ")
				if err != nil {
					t.Fatalf("marshal v1: %v", err)
				}
				writeFile(t, path, data)
			},
		},
		{
			name:        "migrate v2 old config",
			wantFile:    true,
			wantArchive: true,
			setup: func(t *testing.T, dir, path string) {
				oldConfig := OldMultiAppConfig{
					Version: 2,
					Apps: map[string]ProviderManager{
						"claude": {
							Providers: map[string]Provider{
								"p1": {
									ID:   "p1",
									Name: "Legacy",
									SettingsConfig: map[string]interface{}{
										"env": map[string]interface{}{
											"ANTHROPIC_AUTH_TOKEN": "sk-old",
											"ANTHROPIC_BASE_URL":   "https://api.example.com",
										},
									},
								},
							},
							Current: "p1",
						},
					},
				}
				data, err := json.MarshalIndent(oldConfig, "", "  ")
				if err != nil {
					t.Fatalf("marshal v2 old: %v", err)
				}
				writeFile(t, path, data)
			},
			verify: func(t *testing.T, dir, path string, m *Manager) {
				raw := readJSONMap(t, path)
				if _, ok := raw["apps"]; ok {
					t.Fatalf("flattened config should not contain apps key")
				}
				if _, ok := raw["claude"]; !ok {
					t.Fatalf("flattened config should include claude key")
				}
			},
		},
		{
			name:     "v2 config without apps",
			wantFile: true,
			setup: func(t *testing.T, dir, path string) {
				writeFile(t, path, []byte(`{"version":2}`))
			},
		},
		{
			name:     "v2 config with nil providers",
			wantFile: true,
			setup: func(t *testing.T, dir, path string) {
				raw := map[string]interface{}{
					"version": 2,
					"claude": map[string]interface{}{
						"providers": nil,
						"current":   "",
					},
				}
				data, err := json.MarshalIndent(raw, "", "  ")
				if err != nil {
					t.Fatalf("marshal raw: %v", err)
				}
				writeFile(t, path, data)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			configPath := filepath.Join(dir, "config.json")
			if tt.setup != nil {
				tt.setup(t, dir, configPath)
			}

			m, err := NewManagerWithDir(dir)
			if err != nil {
				t.Fatalf("NewManagerWithDir() error = %v", err)
			}

			if tt.wantFile {
				if _, err := os.Stat(configPath); err != nil {
					t.Fatalf("expected config file: %v", err)
				}
			} else if _, err := os.Stat(configPath); err == nil {
				t.Fatalf("expected no config file")
			}

			if m.config == nil {
				t.Fatalf("manager config is nil")
			}
			if m.config.Version != 2 {
				t.Fatalf("config version = %d, want 2", m.config.Version)
			}

			requiredApps := []string{"claude", "codex", "gemini"}
			for _, app := range requiredApps {
				appConfig, ok := m.config.Apps[app]
				if !ok {
					t.Fatalf("missing app %s", app)
				}
				if appConfig.Providers == nil {
					t.Fatalf("providers map for %s is nil", app)
				}
			}

			if tt.wantProviderName != "" {
				if _, err := m.GetProviderForApp("claude", tt.wantProviderName); err != nil {
					t.Fatalf("expected provider %s: %v", tt.wantProviderName, err)
				}
			}

			if tt.wantArchive {
				archiveDir := filepath.Join(dir, "archive")
				entries, err := os.ReadDir(archiveDir)
				if err != nil {
					t.Fatalf("read archive dir: %v", err)
				}
				if len(entries) == 0 {
					t.Fatalf("expected archive backup")
				}
			}

			if tt.verify != nil {
				tt.verify(t, dir, configPath, m)
			}
		})
	}
}

func TestMultiAppConfigJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		config    MultiAppConfig
		wantKeys  []string
		wantApps  []string
		wantPrefs string
	}{
		{
			name: "marshal flattens apps",
			config: MultiAppConfig{
				Version: 2,
				Apps: map[string]ProviderManager{
					"claude": {Providers: map[string]Provider{}, Current: ""},
					"codex":  {Providers: map[string]Provider{}, Current: ""},
				},
				Mcp: &McpRoot{
					Servers: map[string]McpServer{},
				},
				Preferences: &UserPreferences{ViewMode: "multi"},
			},
			wantKeys:  []string{"version", "claude", "codex", "mcp", "preferences"},
			wantPrefs: "multi",
		},
		{
			name:     "unmarshal top-level apps",
			config:   MultiAppConfig{},
			wantApps: []string{"custom", "claude"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.name {
			case "marshal flattens apps":
				data, err := json.Marshal(&tt.config)
				if err != nil {
					t.Fatalf("marshal config: %v", err)
				}
				var raw map[string]interface{}
				if err := json.Unmarshal(data, &raw); err != nil {
					t.Fatalf("unmarshal raw: %v", err)
				}
				for _, key := range tt.wantKeys {
					if _, ok := raw[key]; !ok {
						t.Fatalf("missing key %s", key)
					}
				}
				prefs, ok := raw["preferences"].(map[string]interface{})
				if !ok || prefs["viewMode"] != tt.wantPrefs {
					t.Fatalf("preferences viewMode = %v, want %s", prefs["viewMode"], tt.wantPrefs)
				}
			case "unmarshal top-level apps":
				data := []byte(`{"version":2,"claude":{"providers":{},"current":""},"custom":{"providers":{},"current":""}}`)
				var cfg MultiAppConfig
				if err := json.Unmarshal(data, &cfg); err != nil {
					t.Fatalf("unmarshal config: %v", err)
				}
				for _, app := range tt.wantApps {
					if _, ok := cfg.Apps[app]; !ok {
						t.Fatalf("missing app %s", app)
					}
				}
			}
		})
	}
}

func TestViewModePreferences(t *testing.T) {
	tests := []struct {
		name      string
		startPref *UserPreferences
		setMode   string
		wantMode  string
	}{
		{
			name:     "default mode",
			wantMode: "single",
		},
		{
			name:      "set mode",
			startPref: &UserPreferences{ViewMode: "single"},
			setMode:   "multi",
			wantMode:  "multi",
		},
		{
			name:     "set mode with nil preferences",
			setMode:  "multi",
			wantMode: "multi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m, err := NewManagerWithDir(dir)
			if err != nil {
				t.Fatalf("NewManagerWithDir() error = %v", err)
			}
			m.config.Preferences = tt.startPref

			if tt.setMode != "" {
				if err := m.SetViewMode(tt.setMode); err != nil {
					t.Fatalf("SetViewMode() error = %v", err)
				}
			}

			if got := m.GetViewMode(); got != tt.wantMode {
				t.Fatalf("GetViewMode() = %s, want %s", got, tt.wantMode)
			}
		})
	}
}

func TestAddProviderForApp(t *testing.T) {
	tests := []struct {
		name     string
		appName  string
		provName string
		apiToken string
		baseURL  string
		category string
		setup    func(t *testing.T, m *Manager)
		wantErr  bool
		verify   func(t *testing.T, dir string, m *Manager)
	}{
		{
			name:     "add claude provider",
			appName:  "claude",
			provName: "Alpha",
			apiToken: "sk-claude",
			baseURL:  "https://api.example.com",
			category: "custom",
			verify: func(t *testing.T, dir string, m *Manager) {
				provider, err := m.GetProviderForApp("claude", "Alpha")
				if err != nil {
					t.Fatalf("GetProviderForApp() error = %v", err)
				}
				if provider.SortOrder != 1 {
					t.Fatalf("SortOrder = %d, want 1", provider.SortOrder)
				}
				settingsPath, err := m.GetClaudeSettingsPathWithDir()
				if err != nil {
					t.Fatalf("GetClaudeSettingsPathWithDir() error = %v", err)
				}
				data, err := os.ReadFile(settingsPath)
				if err != nil {
					t.Fatalf("read settings: %v", err)
				}
				var settings ClaudeSettings
				if err := json.Unmarshal(data, &settings); err != nil {
					t.Fatalf("unmarshal settings: %v", err)
				}
				if settings.Env.AnthropicAuthToken != "sk-claude" {
					t.Fatalf("token = %s, want sk-claude", settings.Env.AnthropicAuthToken)
				}
				if settings.Env.AnthropicBaseURL != "https://api.example.com" {
					t.Fatalf("base url = %s", settings.Env.AnthropicBaseURL)
				}
			},
		},
		{
			name:     "add provider when app missing",
			appName:  "claude",
			provName: "Alpha",
			apiToken: "sk-claude",
			baseURL:  "https://api.example.com",
			category: "custom",
			setup: func(t *testing.T, m *Manager) {
				delete(m.config.Apps, "claude")
			},
			verify: func(t *testing.T, dir string, m *Manager) {
				if _, ok := m.config.Apps["claude"]; !ok {
					t.Fatalf("expected claude app to be created")
				}
				if _, err := m.GetProviderForApp("claude", "Alpha"); err != nil {
					t.Fatalf("GetProviderForApp() error = %v", err)
				}
			},
		},
		{
			name:     "normalize sort order before add",
			appName:  "claude",
			provName: "Gamma",
			apiToken: "sk-claude",
			baseURL:  "https://api.example.com",
			category: "custom",
			setup: func(t *testing.T, m *Manager) {
				app := m.config.Apps["claude"]
				app.Providers["p1"] = Provider{ID: "p1", Name: "Alpha", SortOrder: 3, CreatedAt: 1}
				app.Providers["p2"] = Provider{ID: "p2", Name: "Beta"}
				m.config.Apps["claude"] = app
			},
			verify: func(t *testing.T, dir string, m *Manager) {
				app := m.config.Apps["claude"]
				if app.Providers["p1"].SortOrder != 1 || app.Providers["p2"].SortOrder != 2 {
					t.Fatalf("expected normalized sort order")
				}
			},
		},
		{
			name:     "add codex provider",
			appName:  "codex",
			provName: "Codex",
			apiToken: "sk-codex",
			baseURL:  "https://codex.example.com",
			category: "custom",
			verify: func(t *testing.T, dir string, m *Manager) {
				provider, err := m.GetProviderForApp("codex", "Codex")
				if err != nil {
					t.Fatalf("GetProviderForApp() error = %v", err)
				}
				config, ok := provider.SettingsConfig["config"].(string)
				if !ok || !strings.Contains(config, "base_url = \"https://codex.example.com\"") {
					t.Fatalf("missing base_url in config")
				}
				authPath, err := m.GetCodexAuthJsonPathWithDir()
				if err != nil {
					t.Fatalf("GetCodexAuthJsonPathWithDir() error = %v", err)
				}
				if _, err := os.Stat(authPath); err != nil {
					t.Fatalf("auth.json missing: %v", err)
				}
				configPath, err := m.GetCodexConfigPathWithDir()
				if err != nil {
					t.Fatalf("GetCodexConfigPathWithDir() error = %v", err)
				}
				if _, err := os.Stat(configPath); err != nil {
					t.Fatalf("config.toml missing: %v", err)
				}
			},
		},
		{
			name:     "duplicate name",
			appName:  "claude",
			provName: "Dup",
			apiToken: "sk-dup",
			baseURL:  "https://api.example.com",
			category: "custom",
			setup: func(t *testing.T, m *Manager) {
				if err := m.AddProviderForApp("claude", "Dup", "", "sk-old", "https://api.example.com", "custom", "", "", "", ""); err != nil {
					t.Fatalf("setup add provider: %v", err)
				}
			},
			wantErr: true,
		},
		{
			name:     "unsupported app",
			appName:  "gemini",
			provName: "Nope",
			apiToken: "sk-nope",
			baseURL:  "https://api.example.com",
			category: "custom",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m, err := NewManagerWithDir(dir)
			if err != nil {
				t.Fatalf("NewManagerWithDir() error = %v", err)
			}
			if tt.setup != nil {
				tt.setup(t, m)
			}
			err = m.AddProviderForApp(tt.appName, tt.provName, "", tt.apiToken, tt.baseURL, tt.category, "", "", "", "")
			if (err != nil) != tt.wantErr {
				t.Fatalf("AddProviderForApp() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && tt.verify != nil {
				tt.verify(t, dir, m)
			}
		})
	}
}

func TestAddProviderDirect(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, m *Manager)
		input   Provider
		wantErr bool
		verify  func(t *testing.T, m *Manager)
	}{
		{
			name: "first provider sets current",
			input: Provider{
				ID:   "p1",
				Name: "Alpha",
			},
			verify: func(t *testing.T, m *Manager) {
				app := m.config.Apps["claude"]
				if app.Current != "p1" {
					t.Fatalf("current = %s, want p1", app.Current)
				}
				p := app.Providers["p1"]
				if p.SortOrder != 1 {
					t.Fatalf("sort order = %d, want 1", p.SortOrder)
				}
			},
		},
		{
			name: "duplicate id",
			setup: func(t *testing.T, m *Manager) {
				if err := m.AddProviderDirect("claude", Provider{ID: "p1", Name: "Alpha"}); err != nil {
					t.Fatalf("setup add provider: %v", err)
				}
			},
			input:   Provider{ID: "p1", Name: "Beta"},
			wantErr: true,
		},
		{
			name: "duplicate name",
			setup: func(t *testing.T, m *Manager) {
				if err := m.AddProviderDirect("claude", Provider{ID: "p1", Name: "Alpha"}); err != nil {
					t.Fatalf("setup add provider: %v", err)
				}
			},
			input:   Provider{ID: "p2", Name: "Alpha"},
			wantErr: true,
		},
		{
			name: "preserve sort order",
			input: Provider{
				ID:        "p1",
				Name:      "Alpha",
				SortOrder: 3,
			},
			verify: func(t *testing.T, m *Manager) {
				p := m.config.Apps["claude"].Providers["p1"]
				if p.SortOrder != 3 {
					t.Fatalf("sort order = %d, want 3", p.SortOrder)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m, err := NewManagerWithDir(dir)
			if err != nil {
				t.Fatalf("NewManagerWithDir() error = %v", err)
			}
			if tt.setup != nil {
				tt.setup(t, m)
			}
			err = m.AddProviderDirect("claude", tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("AddProviderDirect() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && tt.verify != nil {
				tt.verify(t, m)
			}
		})
	}
}

func TestUpdateProviderForApp(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, m *Manager)
		appName  string
		oldName  string
		newName  string
		apiToken string
		baseURL  string
		wantErr  bool
		verify   func(t *testing.T, m *Manager)
	}{
		{
			name: "update claude provider",
			setup: func(t *testing.T, m *Manager) {
				if err := m.AddProviderForApp("claude", "Old", "", "sk-old", "https://old.example.com", "custom", "", "", "", ""); err != nil {
					t.Fatalf("setup add provider: %v", err)
				}
			},
			appName:  "claude",
			oldName:  "Old",
			newName:  "New",
			apiToken: "sk-new",
			baseURL:  "https://new.example.com",
			verify: func(t *testing.T, m *Manager) {
				p, err := m.GetProviderForApp("claude", "New")
				if err != nil {
					t.Fatalf("GetProviderForApp() error = %v", err)
				}
				env, ok := p.SettingsConfig["env"].(map[string]interface{})
				if !ok {
					t.Fatalf("env map missing")
				}
				if env["ANTHROPIC_AUTH_TOKEN"] != "sk-new" {
					t.Fatalf("token = %v", env["ANTHROPIC_AUTH_TOKEN"])
				}
			},
		},
		{
			name: "update codex provider",
			setup: func(t *testing.T, m *Manager) {
				if err := m.AddProviderForApp("codex", "Old", "", "sk-old", "https://old.example.com", "custom", "", "", "", ""); err != nil {
					t.Fatalf("setup add provider: %v", err)
				}
			},
			appName:  "codex",
			oldName:  "Old",
			newName:  "New",
			apiToken: "sk-new",
			baseURL:  "https://new.example.com",
			verify: func(t *testing.T, m *Manager) {
				p, err := m.GetProviderForApp("codex", "New")
				if err != nil {
					t.Fatalf("GetProviderForApp() error = %v", err)
				}
				baseURL := ExtractBaseURLFromProvider(p)
				if baseURL != "https://new.example.com" {
					t.Fatalf("baseURL = %s", baseURL)
				}
			},
		},
		{
			name:    "missing provider",
			appName: "claude",
			oldName: "Missing",
			newName: "New",
			wantErr: true,
		},
		{
			name: "duplicate new name",
			setup: func(t *testing.T, m *Manager) {
				if err := m.AddProviderForApp("claude", "Old", "", "sk-old", "https://old.example.com", "custom", "", "", "", ""); err != nil {
					t.Fatalf("setup add provider: %v", err)
				}
				if err := m.AddProviderForApp("claude", "New", "", "sk-new", "https://new.example.com", "custom", "", "", "", ""); err != nil {
					t.Fatalf("setup add provider: %v", err)
				}
			},
			appName: "claude",
			oldName: "Old",
			newName: "New",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m, err := NewManagerWithDir(dir)
			if err != nil {
				t.Fatalf("NewManagerWithDir() error = %v", err)
			}
			if tt.setup != nil {
				tt.setup(t, m)
			}
			err = m.UpdateProviderForApp(tt.appName, tt.oldName, tt.newName, "", tt.apiToken, tt.baseURL, "custom", "", "", "", "")
			if (err != nil) != tt.wantErr {
				t.Fatalf("UpdateProviderForApp() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && tt.verify != nil {
				tt.verify(t, m)
			}
		})
	}
}

func TestDeleteProviderForApp(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, m *Manager)
		appName string
		prov    string
		wantErr bool
	}{
		{
			name: "delete non-current",
			setup: func(t *testing.T, m *Manager) {
				if err := m.AddProviderForApp("claude", "Keep", "", "sk-1", "https://api.example.com", "custom", "", "", "", ""); err != nil {
					t.Fatalf("setup add provider: %v", err)
				}
				if err := m.AddProviderForApp("claude", "Drop", "", "sk-2", "https://api.example.com", "custom", "", "", "", ""); err != nil {
					t.Fatalf("setup add provider: %v", err)
				}
			},
			appName: "claude",
			prov:    "Drop",
		},
		{
			name: "delete current",
			setup: func(t *testing.T, m *Manager) {
				if err := m.AddProviderForApp("claude", "Current", "", "sk-1", "https://api.example.com", "custom", "", "", "", ""); err != nil {
					t.Fatalf("setup add provider: %v", err)
				}
				if err := m.AddProviderForApp("claude", "Other", "", "sk-2", "https://api.example.com", "custom", "", "", "", ""); err != nil {
					t.Fatalf("setup add provider: %v", err)
				}
			},
			appName: "claude",
			prov:    "Current",
			wantErr: true,
		},
		{
			name:    "delete missing",
			appName: "claude",
			prov:    "Missing",
			wantErr: true,
		},
		{
			name:    "delete from missing app",
			appName: "missing",
			prov:    "Anything",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m, err := NewManagerWithDir(dir)
			if err != nil {
				t.Fatalf("NewManagerWithDir() error = %v", err)
			}
			if tt.setup != nil {
				tt.setup(t, m)
			}
			err = m.DeleteProviderForApp(tt.appName, tt.prov)
			if (err != nil) != tt.wantErr {
				t.Fatalf("DeleteProviderForApp() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetProviderForApp(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, m *Manager)
		appName string
		prov    string
		wantErr bool
	}{
		{
			name: "existing provider",
			setup: func(t *testing.T, m *Manager) {
				if err := m.AddProviderForApp("claude", "Alpha", "", "sk-1", "https://api.example.com", "custom", "", "", "", ""); err != nil {
					t.Fatalf("setup add provider: %v", err)
				}
			},
			appName: "claude",
			prov:    "Alpha",
		},
		{
			name:    "missing provider",
			appName: "claude",
			prov:    "Missing",
			wantErr: true,
		},
		{
			name:    "missing app",
			appName: "missing",
			prov:    "Alpha",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m, err := NewManagerWithDir(dir)
			if err != nil {
				t.Fatalf("NewManagerWithDir() error = %v", err)
			}
			if tt.setup != nil {
				tt.setup(t, m)
			}
			_, err = m.GetProviderForApp(tt.appName, tt.prov)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetProviderForApp() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSortProvidersAndNormalize(t *testing.T) {
	tests := []struct {
		name        string
		providers   map[string]Provider
		wantOrder   []string
		wantChanged bool
	}{
		{
			name: "sort by explicit order",
			providers: map[string]Provider{
				"a": {ID: "a", Name: "A", SortOrder: 2, CreatedAt: 2},
				"b": {ID: "b", Name: "B", SortOrder: 1, CreatedAt: 1},
			},
			wantOrder:   []string{"B", "A"},
			wantChanged: false,
		},
		{
			name: "sort by created time then name",
			providers: map[string]Provider{
				"a": {ID: "a", Name: "Beta", CreatedAt: 10},
				"b": {ID: "b", Name: "Alpha", CreatedAt: 10},
				"c": {ID: "c", Name: "Gamma", CreatedAt: 0},
			},
			wantOrder:   []string{"Alpha", "Beta", "Gamma"},
			wantChanged: true,
		},
		{
			name: "sort prioritizes explicit order",
			providers: map[string]Provider{
				"a": {ID: "a", Name: "Alpha", SortOrder: 2, CreatedAt: 2},
				"b": {ID: "b", Name: "Beta", CreatedAt: 1},
			},
			wantOrder:   []string{"Alpha", "Beta"},
			wantChanged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ordered := sortProviders(tt.providers)
			if len(ordered) != len(tt.wantOrder) {
				t.Fatalf("order length = %d, want %d", len(ordered), len(tt.wantOrder))
			}
			for i, want := range tt.wantOrder {
				if ordered[i].Name != want {
					t.Fatalf("order[%d] = %s, want %s", i, ordered[i].Name, want)
				}
			}
			_, changed := normalizeSortOrder(tt.providers)
			if changed != tt.wantChanged {
				t.Fatalf("changed = %v, want %v", changed, tt.wantChanged)
			}
		})
	}
}

func TestMultiAppConfigUnmarshalErrors(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "invalid version",
			data: `{"version":"bad"}`,
		},
		{
			name: "invalid mcp",
			data: `{"version":2,"mcp":{"servers":"bad"}}`,
		},
		{
			name: "invalid preferences",
			data: `{"version":2,"preferences":"bad"}`,
		},
		{
			name: "invalid app",
			data: `{"version":2,"claude":"bad"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg MultiAppConfig
			if err := json.Unmarshal([]byte(tt.data), &cfg); err == nil {
				t.Fatalf("expected unmarshal error")
			}
		})
	}
}

func TestParseV2ConfigErrors(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "invalid json",
			data: "{invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m := &Manager{configPath: filepath.Join(dir, "config.json")}
			if err := m.parseV2Config([]byte(tt.data)); err != nil {
				t.Fatalf("parseV2Config() error = %v", err)
			}
			if m.config == nil || m.config.Version != 2 {
				t.Fatalf("config not initialized after parse")
			}
		})
	}
}

func TestMigrationEarlyReturns(t *testing.T) {
	tests := []struct {
		name    string
		call    func(m *Manager) error
		wantErr bool
	}{
		{
			name: "v1 invalid json",
			call: func(m *Manager) error {
				return m.migrateV1Config([]byte("{"))
			},
			wantErr: true,
		},
		{
			name: "v1 missing providers",
			call: func(m *Manager) error {
				return m.migrateV1Config([]byte(`{"current":""}`))
			},
		},
		{
			name: "v2 old invalid json",
			call: func(m *Manager) error {
				return m.migrateV2OldConfig([]byte("{"))
			},
			wantErr: true,
		},
		{
			name: "v2 old empty apps",
			call: func(m *Manager) error {
				return m.migrateV2OldConfig([]byte(`{"version":2,"apps":{}}`))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m := &Manager{configPath: filepath.Join(dir, "config.json")}
			err := tt.call(m)
			if (err != nil) != tt.wantErr {
				t.Fatalf("migration error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMoveProviderForApp(t *testing.T) {
	tests := []struct {
		name      string
		moveID    string
		direction int
		wantErr   bool
		wantOrder []string
		setup     func(t *testing.T, m *Manager) string
	}{
		{
			name:      "move down",
			direction: 1,
			setup: func(t *testing.T, m *Manager) string {
				if err := m.AddProviderForApp("claude", "Alpha", "", "sk-1", "https://api.example.com", "custom", "", "", "", ""); err != nil {
					t.Fatalf("setup add provider: %v", err)
				}
				if err := m.AddProviderForApp("claude", "Beta", "", "sk-2", "https://api.example.com", "custom", "", "", "", ""); err != nil {
					t.Fatalf("setup add provider: %v", err)
				}
				providers := m.ListProvidersForApp("claude")
				return providers[0].ID
			},
			wantOrder: []string{"Beta", "Alpha"},
		},
		{
			name:      "move up",
			direction: -1,
			setup: func(t *testing.T, m *Manager) string {
				if err := m.AddProviderForApp("claude", "Alpha", "", "sk-1", "https://api.example.com", "custom", "", "", "", ""); err != nil {
					t.Fatalf("setup add provider: %v", err)
				}
				if err := m.AddProviderForApp("claude", "Beta", "", "sk-2", "https://api.example.com", "custom", "", "", "", ""); err != nil {
					t.Fatalf("setup add provider: %v", err)
				}
				providers := m.ListProvidersForApp("claude")
				return providers[1].ID
			},
			wantOrder: []string{"Beta", "Alpha"},
		},
		{
			name:      "missing provider",
			direction: 1,
			setup: func(t *testing.T, m *Manager) string {
				return "missing"
			},
			wantErr: true,
		},
		{
			name:      "direction zero",
			direction: 0,
			setup: func(t *testing.T, m *Manager) string {
				if err := m.AddProviderForApp("claude", "Alpha", "", "sk-1", "https://api.example.com", "custom", "", "", "", ""); err != nil {
					t.Fatalf("setup add provider: %v", err)
				}
				providers := m.ListProvidersForApp("claude")
				return providers[0].ID
			},
		},
		{
			name:      "move out of bounds",
			direction: -1,
			setup: func(t *testing.T, m *Manager) string {
				if err := m.AddProviderForApp("claude", "Alpha", "", "sk-1", "https://api.example.com", "custom", "", "", "", ""); err != nil {
					t.Fatalf("setup add provider: %v", err)
				}
				if err := m.AddProviderForApp("claude", "Beta", "", "sk-2", "https://api.example.com", "custom", "", "", "", ""); err != nil {
					t.Fatalf("setup add provider: %v", err)
				}
				providers := m.ListProvidersForApp("claude")
				return providers[0].ID
			},
			wantOrder: []string{"Alpha", "Beta"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m, err := NewManagerWithDir(dir)
			if err != nil {
				t.Fatalf("NewManagerWithDir() error = %v", err)
			}
			id := tt.moveID
			if tt.setup != nil {
				id = tt.setup(t, m)
			}
			err = m.MoveProviderForApp("claude", id, tt.direction)
			if (err != nil) != tt.wantErr {
				t.Fatalf("MoveProviderForApp() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && len(tt.wantOrder) > 0 {
				providers := m.ListProvidersForApp("claude")
				for i, want := range tt.wantOrder {
					if providers[i].Name != want {
						t.Fatalf("order[%d] = %s, want %s", i, providers[i].Name, want)
					}
					if providers[i].SortOrder != i+1 {
						t.Fatalf("sort order[%d] = %d, want %d", i, providers[i].SortOrder, i+1)
					}
				}
			}
		})
	}
}

func TestGetConfigCopy(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "deep copy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m, err := NewManagerWithDir(dir)
			if err != nil {
				t.Fatalf("NewManagerWithDir() error = %v", err)
			}
			if err := m.AddProviderForApp("claude", "Alpha", "", "sk-1", "https://api.example.com", "custom", "", "", "", ""); err != nil {
				t.Fatalf("setup add provider: %v", err)
			}
			cfg, err := m.GetConfig()
			if err != nil {
				t.Fatalf("GetConfig() error = %v", err)
			}
			cfg.Apps["claude"].Providers["fake"] = Provider{ID: "fake", Name: "Fake"}
			if _, ok := m.config.Apps["claude"].Providers["fake"]; ok {
				t.Fatalf("config copy should not mutate manager config")
			}
		})
	}
}

func TestSwitchProviderForApp(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, m *Manager) (target string)
		appName string
		wantErr bool
		verify  func(t *testing.T, dir string, m *Manager)
	}{
		{
			name:    "switch claude",
			appName: "claude",
			setup: func(t *testing.T, m *Manager) string {
				if err := m.AddProviderForApp("claude", "Alpha", "", "sk-1", "https://api.example.com", "custom", "", "", "", ""); err != nil {
					t.Fatalf("setup add provider: %v", err)
				}
				if err := m.AddProviderForApp("claude", "Beta", "", "sk-2", "https://api2.example.com", "custom", "", "", "", ""); err != nil {
					t.Fatalf("setup add provider: %v", err)
				}
				return "Beta"
			},
			verify: func(t *testing.T, dir string, m *Manager) {
				p := m.GetCurrentProviderForApp("claude")
				if p == nil || p.Name != "Beta" {
					t.Fatalf("current provider = %v", p)
				}
				settingsPath, err := m.GetClaudeSettingsPathWithDir()
				if err != nil {
					t.Fatalf("GetClaudeSettingsPathWithDir() error = %v", err)
				}
				data, err := os.ReadFile(settingsPath)
				if err != nil {
					t.Fatalf("read settings: %v", err)
				}
				if !strings.Contains(string(data), "sk-2") {
					t.Fatalf("settings should contain new token")
				}
			},
		},
		{
			name:    "missing app",
			appName: "missing",
			setup: func(t *testing.T, m *Manager) string {
				return "Any"
			},
			wantErr: true,
		},
		{
			name:    "missing provider",
			appName: "claude",
			setup: func(t *testing.T, m *Manager) string {
				return "Missing"
			},
			wantErr: true,
		},
		{
			name:    "write failure keeps current",
			appName: "claude",
			setup: func(t *testing.T, m *Manager) string {
				if err := m.AddProviderForApp("claude", "Alpha", "", "sk-1", "https://api.example.com", "custom", "", "", "", ""); err != nil {
					t.Fatalf("setup add provider: %v", err)
				}
				if err := m.AddProviderForApp("claude", "Beta", "", "sk-2", "https://api2.example.com", "custom", "", "", "", ""); err != nil {
					t.Fatalf("setup add provider: %v", err)
				}
				settingsDir := filepath.Join(m.customDir, ".claude")
				if err := os.MkdirAll(settingsDir, 0755); err != nil {
					t.Fatalf("setup claude dir: %v", err)
				}
				if err := os.Chmod(settingsDir, 0500); err != nil {
					t.Fatalf("chmod claude dir: %v", err)
				}
				t.Cleanup(func() {
					_ = os.Chmod(settingsDir, 0700)
				})
				return "Beta"
			},
			wantErr: true,
			verify: func(t *testing.T, dir string, m *Manager) {
				if runtime.GOOS == "windows" {
					t.Skip("permission model differs on windows")
				}
				current := m.GetCurrentProviderForApp("claude")
				if current == nil || current.Name != "Alpha" {
					t.Fatalf("current should remain Alpha, got %v", current)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "write failure keeps current" && runtime.GOOS == "windows" {
				t.Skip("permission model differs on windows")
			}
			dir := t.TempDir()
			setTempHome(t, dir)
			m, err := NewManagerWithDir(dir)
			if err != nil {
				t.Fatalf("NewManagerWithDir() error = %v", err)
			}
			target := ""
			if tt.setup != nil {
				target = tt.setup(t, m)
			}
			err = m.SwitchProviderForApp(tt.appName, target)
			if (err != nil) != tt.wantErr {
				t.Fatalf("SwitchProviderForApp() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && tt.verify != nil {
				tt.verify(t, dir, m)
			}
			if err != nil && tt.verify != nil {
				tt.verify(t, dir, m)
			}
		})
	}
}

func TestGenerateCodexConfigTOML(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		baseURL  string
		model    string
		reason   string
		want     []string
	}{
		{
			name:     "sanitizes provider name",
			provider: "My Provider!",
			baseURL:  "https://api.example.com",
			model:    "gpt-5",
			reason:   "low",
			want: []string{
				"model_provider = \"my_provider\"",
				"[model_providers.my_provider]",
			},
		},
		{
			name:     "defaults for empty values",
			provider: "",
			baseURL:  "https://api.example.com",
			model:    "",
			reason:   "",
			want: []string{
				"model_provider = \"custom\"",
				"model = \"gpt-5-codex\"",
				"model_reasoning_effort = \"high\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := generateCodexConfigTOML(tt.provider, tt.baseURL, tt.model, tt.reason)
			for _, want := range tt.want {
				if !strings.Contains(config, want) {
					t.Fatalf("expected %q in config", want)
				}
			}
		})
	}
}

func TestMergeCodexConfig(t *testing.T) {
	tests := []struct {
		name     string
		existing map[string]interface{}
		ccs      map[string]interface{}
		verify   func(t *testing.T, merged map[string]interface{})
	}{
		{
			name: "preserves unknown fields and strips env_key",
			existing: map[string]interface{}{
				"custom": "keep",
				"model":  "old",
			},
			ccs: map[string]interface{}{
				"model": "new",
				"model_providers": map[string]interface{}{
					"custom": map[string]interface{}{
						"env_key": "deprecated",
						"name":    "Custom",
					},
				},
			},
			verify: func(t *testing.T, merged map[string]interface{}) {
				if merged["custom"] != "keep" {
					t.Fatalf("custom field not preserved")
				}
				if merged["model"] != "new" {
					t.Fatalf("model should be overridden")
				}
				providers := merged["model_providers"].(map[string]interface{})
				custom := providers["custom"].(map[string]interface{})
				if _, ok := custom["env_key"]; ok {
					t.Fatalf("env_key should be stripped")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged := mergeCodexConfig(tt.existing, tt.ccs)
			tt.verify(t, merged)
		})
	}
}

func TestWriteCodexConfigPreservesFields(t *testing.T) {
	tests := []struct {
		name          string
		existing      string
		configContent string
		verify        func(t *testing.T, data []byte)
		wantExact     string
	}{
		{
			name: "merge with existing",
			existing: `custom = "keep"
model = "old"
[model_providers.custom]
name = "Old"
`,
			configContent: generateCodexConfigTOML("Custom", "https://api.example.com", "gpt-5", "high"),
			verify: func(t *testing.T, data []byte) {
				var parsed map[string]interface{}
				if err := toml.Unmarshal(data, &parsed); err != nil {
					t.Fatalf("parse toml: %v", err)
				}
				if parsed["custom"] != "keep" {
					t.Fatalf("custom = %v, want keep", parsed["custom"])
				}
				if parsed["model"] != "gpt-5" {
					t.Fatalf("model = %v, want gpt-5", parsed["model"])
				}
			},
		},
		{
			name:          "fallback on invalid toml",
			existing:      "custom = \"keep\"",
			configContent: "invalid =",
			wantExact:     "invalid =",
		},
		{
			name:          "empty config removes file",
			existing:      "custom = \"keep\"",
			configContent: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m, err := NewManagerWithDir(dir)
			if err != nil {
				t.Fatalf("NewManagerWithDir() error = %v", err)
			}

			configPath, err := m.GetCodexConfigPathWithDir()
			if err != nil {
				t.Fatalf("GetCodexConfigPathWithDir() error = %v", err)
			}
			if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			if tt.existing != "" {
				writeFile(t, configPath, []byte(tt.existing))
			}

			provider := Provider{
				ID:   "p1",
				Name: "Codex",
				SettingsConfig: map[string]interface{}{
					"auth": map[string]interface{}{
						"OPENAI_API_KEY": "sk-test",
					},
					"config": tt.configContent,
				},
			}

			err = m.writeCodexConfig(&provider)
			if err != nil {
				t.Fatalf("writeCodexConfig() error = %v", err)
			}

			if tt.configContent == "" {
				if _, err := os.Stat(configPath); err == nil {
					t.Fatalf("config.toml should be removed")
				}
				return
			}

			data, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("read config.toml: %v", err)
			}
			if tt.wantExact != "" {
				if strings.TrimSpace(string(data)) != tt.wantExact {
					t.Fatalf("config content mismatch: %s", strings.TrimSpace(string(data)))
				}
				return
			}
			if tt.verify != nil {
				tt.verify(t, data)
			}
		})
	}
}

func TestClaudeSettingsUnknownFields(t *testing.T) {
	tests := []struct {
		name string
		data string
		key  string
	}{
		{
			name: "preserve extra fields",
			data: `{"env":{"ANTHROPIC_AUTH_TOKEN":"sk"},"permissions":{"allow":[],"deny":[]},"extra":{"flag":true}}`,
			key:  "extra",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var settings ClaudeSettings
			if err := json.Unmarshal([]byte(tt.data), &settings); err != nil {
				t.Fatalf("unmarshal settings: %v", err)
			}
			if settings.Extra == nil {
				t.Fatalf("expected extra fields")
			}
			if _, ok := settings.Extra[tt.key]; !ok {
				t.Fatalf("missing extra field %s", tt.key)
			}
			encoded, err := json.Marshal(&settings)
			if err != nil {
				t.Fatalf("marshal settings: %v", err)
			}
			var raw map[string]interface{}
			if err := json.Unmarshal(encoded, &raw); err != nil {
				t.Fatalf("unmarshal encoded: %v", err)
			}
			if _, ok := raw[tt.key]; !ok {
				t.Fatalf("extra field %s not preserved", tt.key)
			}
		})
	}
}

func TestPathResolution(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, home string) func()
		call    func() (string, error)
		want    func(home string) string
		wantErr bool
	}{
		{
			name: "config path normal",
			call: GetConfigPath,
			want: func(home string) string {
				return filepath.Join(home, ".cc-switch", "config.json")
			},
		},
		{
			name: "claude settings uses legacy file",
			setup: func(t *testing.T, home string) func() {
				claudeDir := filepath.Join(home, ".claude")
				if err := os.MkdirAll(claudeDir, 0700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				legacy := filepath.Join(claudeDir, "claude.json")
				writeFile(t, legacy, []byte("{}"))
				return func() {}
			},
			call: GetClaudeSettingsPath,
			want: func(home string) string {
				return filepath.Join(home, ".claude", "claude.json")
			},
		},
		{
			name: "claude settings prefers settings.json",
			setup: func(t *testing.T, home string) func() {
				claudeDir := filepath.Join(home, ".claude")
				if err := os.MkdirAll(claudeDir, 0700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				settings := filepath.Join(claudeDir, "settings.json")
				writeFile(t, settings, []byte("{}"))
				return func() {}
			},
			call: GetClaudeSettingsPath,
			want: func(home string) string {
				return filepath.Join(home, ".claude", "settings.json")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			setTempHome(t, home)
			if tt.setup != nil {
				cleanup := tt.setup(t, home)
				t.Cleanup(cleanup)
			}
			got, err := tt.call()
			if (err != nil) != tt.wantErr {
				t.Fatalf("call error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				want := tt.want(home)
				if got != want {
					t.Fatalf("path = %s, want %s", got, want)
				}
			}
		})
	}
}

func TestPortableConfigPath(t *testing.T) {
	execPath, err := os.Executable()
	if err != nil {
		t.Fatalf("Executable() error = %v", err)
	}
	execDir := filepath.Clean(filepath.Dir(execPath))
	tempDir := filepath.Clean(os.TempDir())
	rel, err := filepath.Rel(tempDir, execDir)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		t.Skipf("refusing to write portable.ini outside temp directory: execDir=%s tempDir=%s", execDir, tempDir)
	}

	portablePath := filepath.Join(execDir, "portable.ini")

	tests := []struct {
		name string
	}{
		{name: "portable mode"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.OpenFile(portablePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
			if err != nil {
				if errors.Is(err, fs.ErrExist) {
					t.Skip("portable.ini already exists")
				}
				if errors.Is(err, fs.ErrPermission) {
					t.Skipf("no permission to create portable.ini in %s: %v", execDir, err)
				}
				t.Fatalf("create portable.ini: %v", err)
			}
			if err := f.Close(); err != nil {
				t.Fatalf("close portable.ini: %v", err)
			}
			t.Cleanup(func() {
				if err := os.Remove(portablePath); err != nil && !errors.Is(err, fs.ErrNotExist) {
					t.Errorf("cleanup remove portable.ini: %v", err)
				}
			})
			path, err := GetConfigPath()
			if err != nil {
				t.Fatalf("GetConfigPath() error = %v", err)
			}
			want := filepath.Join(execDir, ".cc-switch", "config.json")
			if path != want {
				t.Fatalf("path = %s, want %s", path, want)
			}
		})
	}
}

func TestValidationAndExtractors(t *testing.T) {
	tests := []struct {
		name      string
		call      func() (string, error)
		wantErr   bool
		contains  string
		wantValue string
	}{
		{
			name: "validate provider ok",
			call: func() (string, error) {
				return "", ValidateProvider("name", "sk-123", "https://api.example.com")
			},
		},
		{
			name:     "validate provider empty name",
			call:     func() (string, error) { return "", ValidateProvider("", "sk-123", "https://api.example.com") },
			wantErr:  true,
			contains: "名称",
		},
		{
			name:     "validate provider empty token",
			call:     func() (string, error) { return "", ValidateProvider("name", "", "https://api.example.com") },
			wantErr:  true,
			contains: "Token",
		},
		{
			name:     "validate provider invalid token prefix",
			call:     func() (string, error) { return "", ValidateProvider("name", "bad-token", "https://api.example.com") },
			wantErr:  true,
			contains: "格式",
		},
		{
			name: "validate provider token too long",
			call: func() (string, error) {
				return "", ValidateProvider("name", "sk-"+strings.Repeat("x", 1001), "https://api.example.com")
			},
			wantErr:  true,
			contains: "长度",
		},
		{
			name:     "validate provider base url empty",
			call:     func() (string, error) { return "", ValidateProvider("name", "sk-123", "") },
			wantErr:  true,
			contains: "Base URL",
		},
		{
			name:     "validate provider base url missing scheme",
			call:     func() (string, error) { return "", ValidateProvider("name", "sk-123", "api.example.com") },
			wantErr:  true,
			contains: "Base URL",
		},
		{
			name:     "validate provider base url missing host",
			call:     func() (string, error) { return "", ValidateProvider("name", "sk-123", "https://") },
			wantErr:  true,
			contains: "Base URL",
		},
		{
			name:     "validate provider base url parse error",
			call:     func() (string, error) { return "", ValidateProvider("name", "sk-123", "http://[::1") },
			wantErr:  true,
			contains: "Base URL",
		},
		{
			name: "validate max tokens ok",
			call: func() (string, error) {
				return "", ValidateMaxTokens("2048")
			},
		},
		{
			name: "validate max tokens empty",
			call: func() (string, error) {
				return "", ValidateMaxTokens("")
			},
		},
		{
			name:     "validate max tokens invalid",
			call:     func() (string, error) { return "", ValidateMaxTokens("bad") },
			wantErr:  true,
			contains: "Max Tokens",
		},
		{
			name: "mask short token",
			call: func() (string, error) {
				return MaskToken("short"), nil
			},
			wantValue: "****",
		},
		{
			name: "mask long token",
			call: func() (string, error) {
				return MaskToken("sk-12345678"), nil
			},
			wantValue: "sk-1...5678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := tt.call()
			if (err != nil) != tt.wantErr {
				t.Fatalf("call error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.contains != "" && err != nil && !strings.Contains(err.Error(), tt.contains) {
				t.Fatalf("error %v does not contain %q", err, tt.contains)
			}
			if tt.wantValue != "" && value != tt.wantValue {
				t.Fatalf("value = %s, want %s", value, tt.wantValue)
			}
		})
	}

	extractorTests := []struct {
		name          string
		provider      *Provider
		wantToken     string
		wantBaseURL   string
		wantModel     string
		wantReasoning string
		wantSonnet    string
		wantHaiku     string
		wantOpus      string
		wantAnthropic string
	}{
		{
			name: "nil provider",
		},
		{
			name: "claude env",
			provider: &Provider{
				SettingsConfig: map[string]interface{}{
					"env": map[string]interface{}{
						"ANTHROPIC_AUTH_TOKEN":           "sk-1",
						"ANTHROPIC_BASE_URL":             "https://api.example.com",
						"ANTHROPIC_MODEL":                "claude-3",
						"ANTHROPIC_DEFAULT_SONNET_MODEL": "sonnet",
						"ANTHROPIC_DEFAULT_HAIKU_MODEL":  "haiku",
						"ANTHROPIC_DEFAULT_OPUS_MODEL":   "opus",
					},
				},
			},
			wantToken:     "sk-1",
			wantBaseURL:   "https://api.example.com",
			wantModel:     "claude-3",
			wantSonnet:    "sonnet",
			wantHaiku:     "haiku",
			wantOpus:      "opus",
			wantAnthropic: "claude-3",
		},
		{
			name: "codex config",
			provider: &Provider{
				SettingsConfig: map[string]interface{}{
					"auth": map[string]interface{}{
						"OPENAI_API_KEY": "sk-2",
					},
					"config": "model = \"gpt-5\"\nmodel_reasoning_effort = \"low\"\nbase_url = \"https://codex.example.com\"",
				},
			},
			wantToken:     "sk-2",
			wantBaseURL:   "https://codex.example.com",
			wantModel:     "gpt-5",
			wantReasoning: "low",
		},
		{
			name: "settings model field",
			provider: &Provider{
				SettingsConfig: map[string]interface{}{
					"model":                  "claude-2",
					"model_reasoning_effort": "medium",
				},
			},
			wantModel:     "claude-2",
			wantReasoning: "medium",
			wantAnthropic: "claude-2",
		},
	}

	for _, tt := range extractorTests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractTokenFromProvider(tt.provider); got != tt.wantToken {
				t.Fatalf("token = %s, want %s", got, tt.wantToken)
			}
			if got := ExtractBaseURLFromProvider(tt.provider); got != tt.wantBaseURL {
				t.Fatalf("baseURL = %s, want %s", got, tt.wantBaseURL)
			}
			if got := ExtractModelFromProvider(tt.provider); got != tt.wantModel {
				t.Fatalf("model = %s, want %s", got, tt.wantModel)
			}
			if got := ExtractCodexReasoningFromProvider(tt.provider); got != tt.wantReasoning {
				t.Fatalf("reasoning = %s, want %s", got, tt.wantReasoning)
			}
			if got := ExtractDefaultSonnetModelFromProvider(tt.provider); got != tt.wantSonnet {
				t.Fatalf("sonnet = %s, want %s", got, tt.wantSonnet)
			}
			if got := ExtractDefaultHaikuModelFromProvider(tt.provider); got != tt.wantHaiku {
				t.Fatalf("haiku = %s, want %s", got, tt.wantHaiku)
			}
			if got := ExtractDefaultOpusModelFromProvider(tt.provider); got != tt.wantOpus {
				t.Fatalf("opus = %s, want %s", got, tt.wantOpus)
			}
			if got := ExtractAnthropicModelFromProvider(tt.provider); got != tt.wantAnthropic {
				t.Fatalf("anthropic model = %s, want %s", got, tt.wantAnthropic)
			}
		})
	}
}

func TestConcurrentSaves(t *testing.T) {
	tests := []struct {
		name    string
		workers int
	}{
		{name: "concurrent saves", workers: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m, err := NewManagerWithDir(dir)
			if err != nil {
				t.Fatalf("NewManagerWithDir() error = %v", err)
			}
			if err := m.AddProviderForApp("claude", "Alpha", "", "sk-1", "https://api.example.com", "custom", "", "", "", ""); err != nil {
				t.Fatalf("setup add provider: %v", err)
			}

			var wg sync.WaitGroup
			errCh := make(chan error, tt.workers)
			for i := 0; i < tt.workers; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					errCh <- m.Save()
				}()
			}
			wg.Wait()
			close(errCh)

			for err := range errCh {
				if err != nil {
					t.Fatalf("Save() error = %v", err)
				}
			}

			configPath := filepath.Join(dir, "config.json")
			data, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("read config: %v", err)
			}
			var cfg MultiAppConfig
			if err := json.Unmarshal(data, &cfg); err != nil {
				t.Fatalf("config json corrupted: %v", err)
			}
		})
	}
}

func TestSavePermissionError(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "save permission error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if runtime.GOOS == "windows" {
				t.Skip("permission model differs on windows")
			}
			dir := t.TempDir()
			setTempHome(t, dir)
			m, err := NewManagerWithDir(dir)
			if err != nil {
				t.Fatalf("NewManagerWithDir() error = %v", err)
			}
			if err := m.Save(); err != nil {
				t.Fatalf("Save() error = %v", err)
			}
			if err := os.Chmod(dir, 0500); err != nil {
				t.Fatalf("chmod: %v", err)
			}
			t.Cleanup(func() {
				_ = os.Chmod(dir, 0700)
			})
			if err := m.Save(); err == nil {
				t.Fatalf("expected save error")
			}
		})
	}
}

func TestCodexConfigFileRoundTrip(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "toml round trip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m, err := NewManagerWithDir(dir)
			if err != nil {
				t.Fatalf("NewManagerWithDir() error = %v", err)
			}
			provider := Provider{
				Name: "Codex",
				SettingsConfig: map[string]interface{}{
					"auth": map[string]interface{}{
						"OPENAI_API_KEY": "sk-test",
					},
					"config": generateCodexConfigTOML("Codex", "https://api.example.com", "gpt-5", "high"),
				},
			}
			if err := m.writeCodexConfig(&provider); err != nil {
				t.Fatalf("writeCodexConfig() error = %v", err)
			}
			configPath, err := m.GetCodexConfigPathWithDir()
			if err != nil {
				t.Fatalf("GetCodexConfigPathWithDir() error = %v", err)
			}
			data, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("read config: %v", err)
			}
			var parsed map[string]interface{}
			if err := toml.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("parse toml: %v", err)
			}
			if parsed["model"] != "gpt-5" {
				t.Fatalf("model = %v", parsed["model"])
			}
		})
	}
}

func TestNewManagerDefaultPath(t *testing.T) {
	execPath, err := os.Executable()
	if err != nil {
		t.Fatalf("Executable() error = %v", err)
	}
	portablePath := filepath.Join(filepath.Dir(execPath), "portable.ini")
	if _, err := os.Stat(portablePath); err == nil {
		t.Skip("portable.ini already exists")
	}

	tests := []struct {
		name   string
		create func() (*Manager, error)
	}{
		{
			name:   "new manager",
			create: NewManager,
		},
		{
			name: "new manager with empty dir",
			create: func() (*Manager, error) {
				return NewManagerWithDir("")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			setTempHome(t, home)
			m, err := tt.create()
			if err != nil {
				t.Fatalf("create manager: %v", err)
			}
			want := filepath.Join(home, ".cc-switch", "config.json")
			if m.GetConfigPath() != want {
				t.Fatalf("config path = %s, want %s", m.GetConfigPath(), want)
			}
		})
	}
}

func TestNewManagerWithDirError(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "invalid custom dir"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			badPath := filepath.Join(dir, "not-a-dir")
			writeFile(t, badPath, []byte("file"))
			if _, err := NewManagerWithDir(badPath); err == nil {
				t.Fatalf("expected error for custom dir")
			}
		})
	}
}

func TestHomePathHelpers(t *testing.T) {
	tests := []struct {
		name string
		call func() (string, error)
		want func(home string) string
	}{
		{
			name: "codex config path",
			call: GetCodexConfigPath,
			want: func(home string) string {
				return filepath.Join(home, ".codex", "config.toml")
			},
		},
		{
			name: "codex auth path",
			call: GetCodexAuthJsonPath,
			want: func(home string) string {
				return filepath.Join(home, ".codex", "auth.json")
			},
		},
		{
			name: "gemini dir",
			call: GetGeminiDir,
			want: func(home string) string {
				return filepath.Join(home, ".gemini")
			},
		},
		{
			name: "gemini env path",
			call: GetGeminiEnvPath,
			want: func(home string) string {
				return filepath.Join(home, ".gemini", ".env")
			},
		},
		{
			name: "gemini settings path",
			call: GetGeminiSettingsPath,
			want: func(home string) string {
				return filepath.Join(home, ".gemini", "settings.json")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			setTempHome(t, home)
			got, err := tt.call()
			if err != nil {
				t.Fatalf("call error = %v", err)
			}
			if got != tt.want(home) {
				t.Fatalf("path = %s, want %s", got, tt.want(home))
			}
		})
	}
}

func TestManagerPathHelpersWithEmptyCustomDir(t *testing.T) {
	tests := []struct {
		name string
		call func(m *Manager) (string, error)
		want func(home string) string
	}{
		{
			name: "claude settings",
			call: func(m *Manager) (string, error) {
				return m.GetClaudeSettingsPathWithDir()
			},
			want: func(home string) string {
				return filepath.Join(home, ".claude", "settings.json")
			},
		},
		{
			name: "codex config",
			call: func(m *Manager) (string, error) {
				return m.GetCodexConfigPathWithDir()
			},
			want: func(home string) string {
				return filepath.Join(home, ".codex", "config.toml")
			},
		},
		{
			name: "codex auth",
			call: func(m *Manager) (string, error) {
				return m.GetCodexAuthJsonPathWithDir()
			},
			want: func(home string) string {
				return filepath.Join(home, ".codex", "auth.json")
			},
		},
		{
			name: "gemini env",
			call: func(m *Manager) (string, error) {
				return m.GetGeminiEnvPathWithDir()
			},
			want: func(home string) string {
				return filepath.Join(home, ".gemini", ".env")
			},
		},
		{
			name: "gemini settings",
			call: func(m *Manager) (string, error) {
				return m.GetGeminiSettingsPathWithDir()
			},
			want: func(home string) string {
				return filepath.Join(home, ".gemini", "settings.json")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			setTempHome(t, home)
			m := &Manager{}
			got, err := tt.call(m)
			if err != nil {
				t.Fatalf("call error = %v", err)
			}
			if got != tt.want(home) {
				t.Fatalf("path = %s, want %s", got, tt.want(home))
			}
		})
	}
}

func TestProviderWrapperMethods(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "wrapper flow"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m, err := NewManagerWithDir(dir)
			if err != nil {
				t.Fatalf("NewManagerWithDir() error = %v", err)
			}
			if err := m.AddProvider("Alpha", "sk-1", "https://api.example.com", "custom"); err != nil {
				t.Fatalf("AddProvider() error = %v", err)
			}
			if err := m.AddProviderWithWebsite("claude", "Beta", "https://example.com", "sk-2", "https://api2.example.com", "custom"); err != nil {
				t.Fatalf("AddProviderWithWebsite() error = %v", err)
			}
			if len(m.ListProviders()) != 2 {
				t.Fatalf("ListProviders() count mismatch")
			}
			provider, err := m.GetProvider("Beta")
			if err != nil {
				t.Fatalf("GetProvider() error = %v", err)
			}
			if provider.WebsiteURL != "https://example.com" {
				t.Fatalf("WebsiteURL = %s", provider.WebsiteURL)
			}
			if err := m.UpdateProvider("Alpha", "Alpha2", "sk-3", "https://api3.example.com", "custom"); err != nil {
				t.Fatalf("UpdateProvider() error = %v", err)
			}
			if err := m.UpdateProviderWithWebsite("claude", "Beta", "Beta2", "https://new.example.com", "sk-4", "https://api4.example.com", "custom"); err != nil {
				t.Fatalf("UpdateProviderWithWebsite() error = %v", err)
			}
			if err := m.SwitchProvider("Beta2"); err != nil {
				t.Fatalf("SwitchProvider() error = %v", err)
			}
			if current := m.GetCurrentProvider(); current == nil || current.Name != "Beta2" {
				t.Fatalf("current provider = %v", current)
			}
			if err := m.DeleteProvider("Alpha2"); err != nil {
				t.Fatalf("DeleteProvider() error = %v", err)
			}
			if _, err := m.GetProvider("Alpha2"); err == nil {
				t.Fatalf("expected error after delete")
			}
		})
	}
}

func TestWriteProviderConfigBranches(t *testing.T) {
	tests := []struct {
		name     string
		appName  string
		provider Provider
		wantErr  bool
		verify   func(t *testing.T, dir string)
	}{
		{
			name:    "codex",
			appName: "codex",
			provider: Provider{
				Name: "Codex",
				SettingsConfig: map[string]interface{}{
					"auth": map[string]interface{}{
						"OPENAI_API_KEY": "sk-test",
					},
					"config": generateCodexConfigTOML("Codex", "https://api.example.com", "gpt-5", "high"),
				},
			},
			verify: func(t *testing.T, dir string) {
				if _, err := os.Stat(filepath.Join(dir, ".codex", "auth.json")); err != nil {
					t.Fatalf("auth.json missing: %v", err)
				}
				if _, err := os.Stat(filepath.Join(dir, ".codex", "config.toml")); err != nil {
					t.Fatalf("config.toml missing: %v", err)
				}
			},
		},
		{
			name:    "gemini",
			appName: "gemini",
			provider: Provider{
				Name: "Gemini",
				SettingsConfig: map[string]interface{}{
					"env": map[string]interface{}{
						"GEMINI_API_KEY": "sk-xxx",
					},
				},
			},
			verify: func(t *testing.T, dir string) {
				if _, err := os.Stat(filepath.Join(dir, ".gemini", ".env")); err != nil {
					t.Fatalf("env file missing: %v", err)
				}
				if _, err := os.Stat(filepath.Join(dir, ".gemini", "settings.json")); err != nil {
					t.Fatalf("settings file missing: %v", err)
				}
			},
		},
		{
			name:    "unsupported",
			appName: "unknown",
			provider: Provider{
				Name: "Unknown",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			m := &Manager{customDir: dir}
			err := m.writeProviderConfig(tt.appName, &tt.provider)
			if (err != nil) != tt.wantErr {
				t.Fatalf("writeProviderConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && tt.verify != nil {
				tt.verify(t, dir)
			}
		})
	}
}

func TestWriteClaudeConfigPreservesExtras(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "preserve extras"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			m := &Manager{customDir: dir}
			settingsPath, err := m.GetClaudeSettingsPathWithDir()
			if err != nil {
				t.Fatalf("GetClaudeSettingsPathWithDir() error = %v", err)
			}
			if err := os.MkdirAll(filepath.Dir(settingsPath), 0700); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			seed := map[string]interface{}{
				"env": map[string]interface{}{
					"ANTHROPIC_AUTH_TOKEN": "sk-old",
				},
				"permissions": map[string]interface{}{
					"allow": []string{"tool"},
					"deny":  []string{},
				},
				"extraField": "keep",
			}
			seedData, _ := json.MarshalIndent(seed, "", "  ")
			writeFile(t, settingsPath, seedData)

			provider := Provider{
				Name: "Claude",
				SettingsConfig: map[string]interface{}{
					"env": map[string]interface{}{
						"ANTHROPIC_AUTH_TOKEN": "sk-new",
						"ANTHROPIC_BASE_URL":   "https://api.example.com",
					},
					"model": "claude-3",
				},
			}

			if err := m.writeClaudeConfig(&provider); err != nil {
				t.Fatalf("writeClaudeConfig() error = %v", err)
			}

			data, err := os.ReadFile(settingsPath)
			if err != nil {
				t.Fatalf("read settings: %v", err)
			}
			var settings ClaudeSettings
			if err := json.Unmarshal(data, &settings); err != nil {
				t.Fatalf("unmarshal settings: %v", err)
			}
			if settings.Extra["extraField"] != "keep" {
				t.Fatalf("extra field not preserved")
			}
			if settings.Env.AnthropicModel != "claude-3" {
				t.Fatalf("env model = %s", settings.Env.AnthropicModel)
			}
			if settings.Model != "claude-3" {
				t.Fatalf("model = %s", settings.Model)
			}
			if len(settings.Permissions.Allow) != 1 {
				t.Fatalf("permissions not preserved")
			}
		})
	}
}

func TestWriteClaudeConfigModelFallback(t *testing.T) {
	tests := []struct {
		name     string
		seed     string
		provider Provider
		verify   func(t *testing.T, settings ClaudeSettings)
	}{
		{
			name: "uses existing env model",
			seed: `{"env":{"ANTHROPIC_MODEL":"claude-3"},"permissions":{"allow":[],"deny":[]}}`,
			provider: Provider{
				Name:           "Claude",
				SettingsConfig: map[string]interface{}{},
			},
			verify: func(t *testing.T, settings ClaudeSettings) {
				if settings.Model != "claude-3" {
					t.Fatalf("model = %s, want claude-3", settings.Model)
				}
				if settings.Env.AnthropicModel != "claude-3" {
					t.Fatalf("env model = %s, want claude-3", settings.Env.AnthropicModel)
				}
			},
		},
		{
			name: "model without env populates env",
			seed: `{"permissions":{"allow":[],"deny":[]}}`,
			provider: Provider{
				Name: "Claude",
				SettingsConfig: map[string]interface{}{
					"model": "claude-2",
				},
			},
			verify: func(t *testing.T, settings ClaudeSettings) {
				if settings.Model != "claude-2" {
					t.Fatalf("model = %s, want claude-2", settings.Model)
				}
				if settings.Env.AnthropicModel != "claude-2" {
					t.Fatalf("env model = %s, want claude-2", settings.Env.AnthropicModel)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			m := &Manager{customDir: dir}
			settingsPath, err := m.GetClaudeSettingsPathWithDir()
			if err != nil {
				t.Fatalf("GetClaudeSettingsPathWithDir() error = %v", err)
			}
			if err := os.MkdirAll(filepath.Dir(settingsPath), 0700); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			writeFile(t, settingsPath, []byte(tt.seed))

			if err := m.writeClaudeConfig(&tt.provider); err != nil {
				t.Fatalf("writeClaudeConfig() error = %v", err)
			}
			data, err := os.ReadFile(settingsPath)
			if err != nil {
				t.Fatalf("read settings: %v", err)
			}
			var settings ClaudeSettings
			if err := json.Unmarshal(data, &settings); err != nil {
				t.Fatalf("unmarshal settings: %v", err)
			}
			if tt.verify != nil {
				tt.verify(t, settings)
			}
		})
	}
}

func TestWriteClaudeConfigErrors(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, dir string)
		provider Provider
	}{
		{
			name: "mkdir error",
			setup: func(t *testing.T, dir string) {
				writeFile(t, filepath.Join(dir, ".claude"), []byte("file"))
			},
			provider: Provider{
				Name: "Claude",
				SettingsConfig: map[string]interface{}{
					"env": map[string]interface{}{
						"ANTHROPIC_AUTH_TOKEN": "sk-test",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			m := &Manager{customDir: dir}
			if tt.setup != nil {
				tt.setup(t, dir)
			}
			if err := m.writeClaudeConfig(&tt.provider); err == nil {
				t.Fatalf("expected writeClaudeConfig error")
			}
		})
	}
}

func TestMultiAppConfigUnmarshalWithMcpPrefs(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "mcp and preferences",
			data: `{"version":2,"preferences":{"viewMode":"multi"},"mcp":{"servers":{"srv":{"id":"srv","name":"Server","server":{"type":"stdio","command":"npx"},"apps":{"claude":true,"codex":false,"gemini":false}}}},"claude":{"providers":{},"current":""}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg MultiAppConfig
			if err := json.Unmarshal([]byte(tt.data), &cfg); err != nil {
				t.Fatalf("unmarshal config: %v", err)
			}
			if cfg.Mcp == nil || cfg.Mcp.Servers["srv"].ID != "srv" {
				t.Fatalf("mcp servers not parsed")
			}
			if cfg.Preferences == nil || cfg.Preferences.ViewMode != "multi" {
				t.Fatalf("preferences not parsed")
			}
		})
	}
}

func TestGetCurrentProviderForAppMissing(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(m *Manager)
		wantNil bool
	}{
		{
			name:    "no current provider",
			wantNil: true,
		},
		{
			name: "missing current id",
			setup: func(m *Manager) {
				app := m.config.Apps["claude"]
				app.Current = "missing"
				m.config.Apps["claude"] = app
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m, err := NewManagerWithDir(dir)
			if err != nil {
				t.Fatalf("NewManagerWithDir() error = %v", err)
			}
			if tt.setup != nil {
				tt.setup(m)
			}
			if got := m.GetCurrentProviderForApp("claude"); (got == nil) != tt.wantNil {
				t.Fatalf("GetCurrentProviderForApp() = %v", got)
			}
			if got := m.ListProvidersForApp("missing"); len(got) != 0 {
				t.Fatalf("ListProvidersForApp() expected empty list")
			}
		})
	}
}

func TestSaveMkdirAllError(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "mkdir error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m, err := NewManagerWithDir(dir)
			if err != nil {
				t.Fatalf("NewManagerWithDir() error = %v", err)
			}
			badDir := filepath.Join(dir, "not-a-dir")
			writeFile(t, badDir, []byte("file"))
			m.configPath = filepath.Join(badDir, "config.json")
			if err := m.Save(); err == nil {
				t.Fatalf("expected save error")
			}
		})
	}
}
