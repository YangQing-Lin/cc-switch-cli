package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectGeminiAuthType(t *testing.T) {
	tests := []struct {
		name     string
		provider *Provider
		want     GeminiAuthType
	}{
		{
			name:     "nil provider",
			provider: nil,
			want:     GeminiAuthAPIKey,
		},
		{
			name: "explicit api key",
			provider: &Provider{
				Name: "Custom",
				SettingsConfig: map[string]interface{}{
					"authType": "gemini-api-key",
					"env": map[string]interface{}{
						"GEMINI_API_KEY": "sk-xxx",
					},
				},
			},
			want: GeminiAuthAPIKey,
		},
		{
			name: "explicit oauth",
			provider: &Provider{
				Name: "Google",
				SettingsConfig: map[string]interface{}{
					"authType": "oauth-personal",
					"env":      map[string]interface{}{},
				},
			},
			want: GeminiAuthOAuth,
		},
		{
			name: "oauth by name",
			provider: &Provider{
				Name: "Google Official",
				SettingsConfig: map[string]interface{}{
					"env": map[string]interface{}{},
				},
			},
			want: GeminiAuthOAuth,
		},
		{
			name: "oauth by url",
			provider: &Provider{
				Name:       "Custom",
				WebsiteURL: "https://ai.google.dev/",
				SettingsConfig: map[string]interface{}{
					"env": map[string]interface{}{},
				},
			},
			want: GeminiAuthOAuth,
		},
		{
			name: "oauth by missing api key",
			provider: &Provider{
				Name: "Custom",
				SettingsConfig: map[string]interface{}{
					"env": map[string]interface{}{},
				},
			},
			want: GeminiAuthOAuth,
		},
		{
			name: "api key default",
			provider: &Provider{
				Name: "Custom",
				SettingsConfig: map[string]interface{}{
					"env": map[string]interface{}{
						"GEMINI_API_KEY": "sk-xxx",
					},
				},
			},
			want: GeminiAuthAPIKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectGeminiAuthType(tt.provider); got != tt.want {
				t.Fatalf("DetectGeminiAuthType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeGeminiEnv(t *testing.T) {
	tests := []struct {
		name     string
		provider *Provider
		want     map[string]string
	}{
		{
			name:     "nil provider",
			provider: nil,
			want:     map[string]string{},
		},
		{
			name: "standard variables",
			provider: &Provider{
				SettingsConfig: map[string]interface{}{
					"env": map[string]interface{}{
						"GEMINI_API_KEY":         "sk-xxx",
						"GOOGLE_GEMINI_BASE_URL": "https://api.example.com",
						"GEMINI_MODEL":           "gemini-2.5-pro",
					},
				},
			},
			want: map[string]string{
				"GEMINI_API_KEY":         "sk-xxx",
				"GOOGLE_GEMINI_BASE_URL": "https://api.example.com",
				"GEMINI_MODEL":           "gemini-2.5-pro",
			},
		},
		{
			name: "custom variables",
			provider: &Provider{
				SettingsConfig: map[string]interface{}{
					"env": map[string]interface{}{
						"GEMINI_API_KEY": "sk-xxx",
						"CUSTOM_VAR":     "custom-value",
					},
				},
			},
			want: map[string]string{
				"GEMINI_API_KEY": "sk-xxx",
				"CUSTOM_VAR":     "custom-value",
			},
		},
		{
			name: "empty env",
			provider: &Provider{
				SettingsConfig: map[string]interface{}{
					"env": map[string]interface{}{},
				},
			},
			want: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeGeminiEnv(tt.provider)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for key, wantVal := range tt.want {
				if gotVal := got[key]; gotVal != wantVal {
					t.Fatalf("%s = %s, want %s", key, gotVal, wantVal)
				}
			}
		})
	}
}

func TestParseEnvFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantVars map[string]string
	}{
		{
			name:    "basic env file",
			content: "# Comment\nGEMINI_API_KEY=test-key\nGOOGLE_GEMINI_BASE_URL=https://api.example.com\n\nSOME_VAR=value with = sign\n",
			wantVars: map[string]string{
				"GEMINI_API_KEY":         "test-key",
				"GOOGLE_GEMINI_BASE_URL": "https://api.example.com",
				"SOME_VAR":               "value with = sign",
			},
		},
		{
			name:    "whitespace around key and value",
			content: "  GEMINI_API_KEY =  test-key  \nGOOGLE_GEMINI_BASE_URL= https://api.example.com \n",
			wantVars: map[string]string{
				"GEMINI_API_KEY":         "test-key",
				"GOOGLE_GEMINI_BASE_URL": "https://api.example.com",
			},
		},
		{
			name:    "export prefix",
			content: "export GEMINI_API_KEY=test-key\nexport\tGOOGLE_GEMINI_BASE_URL=https://api.example.com\n",
			wantVars: map[string]string{
				"GEMINI_API_KEY":         "test-key",
				"GOOGLE_GEMINI_BASE_URL": "https://api.example.com",
			},
		},
		{
			name:     "ignore empty key",
			content:  "=value\n",
			wantVars: map[string]string{},
		},
		{
			name:     "ignore lines without equals",
			content:  "NO_EQUALS\n",
			wantVars: map[string]string{},
		},
		{
			name:    "ignore comment lines",
			content: "   # leading comment\n# comment\nKEY=value\n",
			wantVars: map[string]string{
				"KEY": "value",
			},
		},
		{
			name:    "quoted values",
			content: "GEMINI_API_KEY=\"test key\"\nSINGLE='a b'\n",
			wantVars: map[string]string{
				"GEMINI_API_KEY": "test key",
				"SINGLE":         "a b",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars, lines := parseEnvFile(tt.content)
			if wantLines := len(strings.Split(tt.content, "\n")); len(lines) != wantLines {
				t.Fatalf("lines = %d, want %d", len(lines), wantLines)
			}
			if len(vars) != len(tt.wantVars) {
				t.Fatalf("vars len = %d, want %d", len(vars), len(tt.wantVars))
			}
			for key, wantVal := range tt.wantVars {
				if got := vars[key]; got != wantVal {
					t.Fatalf("%s = %s, want %s", key, got, wantVal)
				}
			}
		})
	}
}

func TestWriteGeminiEnvFile(t *testing.T) {
	tests := []struct {
		name        string
		seedContent string
		envMap      map[string]string
		verify      func(t *testing.T, path string)
	}{
		{
			name: "create new file",
			envMap: map[string]string{
				"GEMINI_API_KEY":         "sk-test-xxx",
				"GOOGLE_GEMINI_BASE_URL": "https://api.test.com",
				"GEMINI_MODEL":           "gemini-2.5-pro",
			},
			verify: func(t *testing.T, path string) {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read env file: %v", err)
				}
				content := string(data)
				if !strings.Contains(content, "GEMINI_API_KEY=sk-test-xxx") {
					t.Fatalf("missing GEMINI_API_KEY")
				}
				if !strings.Contains(content, "GOOGLE_GEMINI_BASE_URL=https://api.test.com") {
					t.Fatalf("missing GOOGLE_GEMINI_BASE_URL")
				}
				if !strings.Contains(content, "GEMINI_MODEL=gemini-2.5-pro") {
					t.Fatalf("missing GEMINI_MODEL")
				}
			},
		},
		{
			name:        "incremental update",
			seedContent: "# Existing\nSOME_OTHER_VAR=existing-value\nGOOGLE_GEMINI_BASE_URL=https://old.example.com\nANOTHER_VAR=another-value\nGEMINI_API_KEY=old-key\n",
			envMap: map[string]string{
				"GEMINI_API_KEY":         "new-api-key",
				"GOOGLE_GEMINI_BASE_URL": "https://new.example.com",
				"GEMINI_MODEL":           "gemini-3-pro-preview",
			},
			verify: func(t *testing.T, path string) {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read env file: %v", err)
				}
				content := string(data)
				checks := []string{
					"GEMINI_API_KEY=new-api-key",
					"GOOGLE_GEMINI_BASE_URL=https://new.example.com",
					"GEMINI_MODEL=gemini-3-pro-preview",
					"SOME_OTHER_VAR=existing-value",
					"ANOTHER_VAR=another-value",
					"# Existing",
				}
				for _, check := range checks {
					if !strings.Contains(content, check) {
						t.Fatalf("missing %q", check)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			manager := &Manager{customDir: dir}
			if tt.seedContent != "" {
				envDir := filepath.Join(dir, ".gemini")
				if err := os.MkdirAll(envDir, 0700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				envPath := filepath.Join(envDir, ".env")
				writeFile(t, envPath, []byte(tt.seedContent))
			}

			if err := manager.writeGeminiEnvFile(tt.envMap); err != nil {
				t.Fatalf("writeGeminiEnvFile() error = %v", err)
			}

			envPath := filepath.Join(dir, ".gemini", ".env")
			if _, err := os.Stat(envPath); err != nil {
				t.Fatalf("env file missing: %v", err)
			}
			if tt.verify != nil {
				tt.verify(t, envPath)
			}
		})
	}
}

func TestWriteGeminiSettingsFile(t *testing.T) {
	tests := []struct {
		name     string
		seed     string
		authType GeminiAuthType
		verify   func(t *testing.T, path string)
	}{
		{
			name:     "new file",
			authType: GeminiAuthAPIKey,
			verify: func(t *testing.T, path string) {
				data := readJSONMap(t, path)
				security := data["security"].(map[string]interface{})
				auth := security["auth"].(map[string]interface{})
				if auth["selectedType"] != string(GeminiAuthAPIKey) {
					t.Fatalf("authType = %v", auth["selectedType"])
				}
			},
		},
		{
			name:     "preserves extra fields",
			seed:     `{"security":{"auth":{"selectedType":"gemini-api-key"}},"mcpServers":{"old":{}},"extra":"keep"}`,
			authType: GeminiAuthOAuth,
			verify: func(t *testing.T, path string) {
				data := readJSONMap(t, path)
				security := data["security"].(map[string]interface{})
				auth := security["auth"].(map[string]interface{})
				if auth["selectedType"] != string(GeminiAuthOAuth) {
					t.Fatalf("authType = %v", auth["selectedType"])
				}
				mcp := data["mcpServers"].(map[string]interface{})
				if _, ok := mcp["old"]; !ok {
					t.Fatalf("mcpServers not preserved")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			manager := &Manager{customDir: dir}
			settingsPath := filepath.Join(dir, ".gemini", "settings.json")
			if err := os.MkdirAll(filepath.Dir(settingsPath), 0700); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			if tt.seed != "" {
				writeFile(t, settingsPath, []byte(tt.seed))
			}
			if err := manager.writeGeminiSettingsFile(tt.authType); err != nil {
				t.Fatalf("writeGeminiSettingsFile() error = %v", err)
			}
			if tt.verify != nil {
				tt.verify(t, settingsPath)
			}
		})
	}
}

func TestGeminiSettingsUnknownFields(t *testing.T) {
	tests := []struct {
		name string
		data string
		key  string
	}{
		{
			name: "preserve extra",
			data: `{"security":{"auth":{"selectedType":"gemini-api-key"}},"extra":{"flag":true}}`,
			key:  "extra",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var settings GeminiSettings
			if err := json.Unmarshal([]byte(tt.data), &settings); err != nil {
				t.Fatalf("unmarshal settings: %v", err)
			}
			if settings.Extra == nil || settings.Extra[tt.key] == nil {
				t.Fatalf("extra field missing")
			}
			data, err := json.Marshal(&settings)
			if err != nil {
				t.Fatalf("marshal settings: %v", err)
			}
			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("unmarshal raw: %v", err)
			}
			if _, ok := raw[tt.key]; !ok {
				t.Fatalf("extra field not preserved")
			}
		})
	}
}

func TestExtractGeminiConfigFromProvider(t *testing.T) {
	tests := []struct {
		name      string
		provider  *Provider
		wantBase  string
		wantKey   string
		wantModel string
		wantAuth  GeminiAuthType
	}{
		{
			name:     "nil provider",
			provider: nil,
			wantAuth: "",
		},
		{
			name: "extract config",
			provider: &Provider{
				Name: "Test",
				SettingsConfig: map[string]interface{}{
					"authType": "gemini-api-key",
					"env": map[string]interface{}{
						"GOOGLE_GEMINI_BASE_URL": "https://api.example.com",
						"GEMINI_API_KEY":         "sk-xxx",
						"GEMINI_MODEL":           "gemini-2.5-flash",
					},
				},
			},
			wantBase:  "https://api.example.com",
			wantKey:   "sk-xxx",
			wantModel: "gemini-2.5-flash",
			wantAuth:  GeminiAuthAPIKey,
		},
		{
			name: "fallback google api key",
			provider: &Provider{
				Name: "Test",
				SettingsConfig: map[string]interface{}{
					"env": map[string]interface{}{
						"GOOGLE_GEMINI_BASE_URL": "https://api.example.com",
						"GOOGLE_GEMINI_API_KEY":  "sk-google",
					},
				},
			},
			wantBase: "https://api.example.com",
			wantKey:  "sk-google",
			wantAuth: GeminiAuthAPIKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL, apiKey, model, authType := ExtractGeminiConfigFromProvider(tt.provider)
			if baseURL != tt.wantBase {
				t.Fatalf("baseURL = %s, want %s", baseURL, tt.wantBase)
			}
			if apiKey != tt.wantKey {
				t.Fatalf("apiKey = %s, want %s", apiKey, tt.wantKey)
			}
			if model != tt.wantModel {
				t.Fatalf("model = %s, want %s", model, tt.wantModel)
			}
			if authType != tt.wantAuth {
				t.Fatalf("authType = %v, want %v", authType, tt.wantAuth)
			}
		})
	}
}

func TestGeminiProviderCRUD(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, m *Manager)
		action  func(m *Manager) error
		verify  func(t *testing.T, dir string, m *Manager)
		wantErr bool
	}{
		{
			name: "add provider writes live files",
			action: func(m *Manager) error {
				return m.AddGeminiProvider("Gemini", "https://api.example.com", "sk-xxx", "gemini-2.5-pro", GeminiAuthAPIKey)
			},
			verify: func(t *testing.T, dir string, m *Manager) {
				envPath := filepath.Join(dir, ".gemini", ".env")
				if _, err := os.Stat(envPath); err != nil {
					t.Fatalf("env file missing: %v", err)
				}
				settingsPath := filepath.Join(dir, ".gemini", "settings.json")
				if _, err := os.Stat(settingsPath); err != nil {
					t.Fatalf("settings file missing: %v", err)
				}
				current := m.GetCurrentProviderForApp("gemini")
				if current == nil || current.Name != "Gemini" {
					t.Fatalf("current provider mismatch: %v", current)
				}
			},
		},
		{
			name: "add oauth sets website",
			action: func(m *Manager) error {
				return m.AddGeminiProvider("Google", "https://api.example.com", "", "", GeminiAuthOAuth)
			},
			verify: func(t *testing.T, dir string, m *Manager) {
				provider, err := m.GetProviderForApp("gemini", "Google")
				if err != nil {
					t.Fatalf("GetProviderForApp() error = %v", err)
				}
				if provider.WebsiteURL != "https://ai.google.dev/" {
					t.Fatalf("WebsiteURL = %s", provider.WebsiteURL)
				}
			},
		},
		{
			name: "missing app",
			setup: func(t *testing.T, m *Manager) {
				delete(m.config.Apps, "gemini")
			},
			action:  func(m *Manager) error { return m.AddGeminiProvider("Gemini", "", "", "", GeminiAuthAPIKey) },
			wantErr: true,
		},
		{
			name: "update provider",
			setup: func(t *testing.T, m *Manager) {
				if err := m.AddGeminiProvider("Old", "https://old.example.com", "sk-old", "gemini-1", GeminiAuthAPIKey); err != nil {
					t.Fatalf("setup add provider: %v", err)
				}
			},
			action: func(m *Manager) error {
				return m.UpdateGeminiProvider("Old", "New", "https://new.example.com", "sk-new", "gemini-2", GeminiAuthOAuth)
			},
			verify: func(t *testing.T, dir string, m *Manager) {
				provider, err := m.GetProviderForApp("gemini", "New")
				if err != nil {
					t.Fatalf("GetProviderForApp() error = %v", err)
				}
				if provider.WebsiteURL == "" {
					t.Fatalf("WebsiteURL should be set for oauth")
				}
			},
		},
		{
			name: "duplicate name",
			setup: func(t *testing.T, m *Manager) {
				if err := m.AddGeminiProvider("Dup", "", "", "", GeminiAuthAPIKey); err != nil {
					t.Fatalf("setup add provider: %v", err)
				}
			},
			action:  func(m *Manager) error { return m.AddGeminiProvider("Dup", "", "", "", GeminiAuthAPIKey) },
			wantErr: true,
		},
		{
			name:    "update missing",
			action:  func(m *Manager) error { return m.UpdateGeminiProvider("Missing", "New", "", "", "", GeminiAuthAPIKey) },
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
			err = tt.action(m)
			if (err != nil) != tt.wantErr {
				t.Fatalf("action error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && tt.verify != nil {
				tt.verify(t, dir, m)
			}
		})
	}
}

func TestWriteGeminiConfig(t *testing.T) {
	tests := []struct {
		name     string
		provider *Provider
		setup    func(t *testing.T, dir string)
		wantErr  bool
		verify   func(t *testing.T, dir string)
	}{
		{
			name:     "nil provider",
			provider: nil,
			wantErr:  true,
		},
		{
			name: "settings file invalid",
			provider: &Provider{
				Name: "Gemini",
				SettingsConfig: map[string]interface{}{
					"env": map[string]interface{}{
						"GEMINI_API_KEY": "sk-xxx",
					},
				},
			},
			setup: func(t *testing.T, dir string) {
				settingsPath := filepath.Join(dir, ".gemini", "settings.json")
				if err := os.MkdirAll(filepath.Dir(settingsPath), 0700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				writeFile(t, settingsPath, []byte("{"))
			},
			wantErr: true,
		},
		{
			name: "env dir invalid",
			provider: &Provider{
				Name: "Gemini",
				SettingsConfig: map[string]interface{}{
					"env": map[string]interface{}{
						"GEMINI_API_KEY": "sk-xxx",
					},
				},
			},
			setup: func(t *testing.T, dir string) {
				writeFile(t, filepath.Join(dir, ".gemini"), []byte("file"))
			},
			wantErr: true,
		},
		{
			name: "valid provider",
			provider: &Provider{
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.setup != nil {
				tt.setup(t, dir)
			}
			manager := &Manager{customDir: dir}
			err := manager.writeGeminiConfig(tt.provider)
			if (err != nil) != tt.wantErr {
				t.Fatalf("writeGeminiConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && tt.verify != nil {
				tt.verify(t, dir)
			}
		})
	}
}

func TestWriteGeminiSettingsFileError(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, dir string, manager *Manager)
	}{
		{
			name: "invalid json",
			setup: func(t *testing.T, dir string, manager *Manager) {
				settingsPath := filepath.Join(dir, ".gemini", "settings.json")
				if err := os.MkdirAll(filepath.Dir(settingsPath), 0700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				writeFile(t, settingsPath, []byte("{"))
			},
		},
		{
			name: "mkdir error",
			setup: func(t *testing.T, dir string, manager *Manager) {
				writeFile(t, filepath.Join(dir, ".gemini"), []byte("file"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			manager := &Manager{customDir: dir}
			if tt.setup != nil {
				tt.setup(t, dir, manager)
			}
			if err := manager.writeGeminiSettingsFile(GeminiAuthAPIKey); err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func TestGeminiSettingsUnmarshalErrors(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "invalid json",
			data: "{",
		},
		{
			name: "invalid security",
			data: `{"security":"bad"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var settings GeminiSettings
			if err := json.Unmarshal([]byte(tt.data), &settings); err == nil {
				t.Fatalf("expected unmarshal error")
			}
		})
	}
}
