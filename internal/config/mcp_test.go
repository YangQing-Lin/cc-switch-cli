package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

func readJSONMapFile(t *testing.T, path string) map[string]interface{} {
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

func TestValidateMcpServer(t *testing.T) {
	tests := []struct {
		name    string
		server  McpServer
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid stdio",
			server: McpServer{
				ID:   "test",
				Name: "Test",
				Server: map[string]interface{}{
					"type":    "stdio",
					"command": "npx",
				},
			},
		},
		{
			name: "valid http",
			server: McpServer{
				ID:   "test",
				Name: "Test",
				Server: map[string]interface{}{
					"type": "http",
					"url":  "https://example.com/api",
				},
			},
		},
		{
			name: "valid sse",
			server: McpServer{
				ID:   "test",
				Name: "Test",
				Server: map[string]interface{}{
					"type": "sse",
					"url":  "https://example.com/api",
				},
			},
		},
		{
			name: "sse missing url",
			server: McpServer{
				ID:   "test",
				Name: "Test",
				Server: map[string]interface{}{
					"type": "sse",
				},
			},
			wantErr: true,
			errMsg:  "url",
		},
		{
			name: "sse missing host",
			server: McpServer{
				ID:   "test",
				Name: "Test",
				Server: map[string]interface{}{
					"type": "sse",
					"url":  "http://",
				},
			},
			wantErr: true,
			errMsg:  "主机",
		},
		{
			name: "sse invalid scheme",
			server: McpServer{
				ID:   "test",
				Name: "Test",
				Server: map[string]interface{}{
					"type": "sse",
					"url":  "ftp://example.com",
				},
			},
			wantErr: true,
			errMsg:  "http",
		},
		{
			name: "missing id",
			server: McpServer{
				Name: "Test",
				Server: map[string]interface{}{
					"type":    "stdio",
					"command": "npx",
				},
			},
			wantErr: true,
			errMsg:  "ID",
		},
		{
			name: "missing name",
			server: McpServer{
				ID: "test",
				Server: map[string]interface{}{
					"type":    "stdio",
					"command": "npx",
				},
			},
			wantErr: true,
			errMsg:  "名称",
		},
		{
			name: "missing type",
			server: McpServer{
				ID:   "test",
				Name: "Test",
				Server: map[string]interface{}{
					"command": "npx",
				},
			},
			wantErr: true,
			errMsg:  "type",
		},
		{
			name: "unsupported type",
			server: McpServer{
				ID:   "test",
				Name: "Test",
				Server: map[string]interface{}{
					"type": "ws",
				},
			},
			wantErr: true,
			errMsg:  "连接类型",
		},
		{
			name: "stdio missing command",
			server: McpServer{
				ID:   "test",
				Name: "Test",
				Server: map[string]interface{}{
					"type": "stdio",
				},
			},
			wantErr: true,
			errMsg:  "command",
		},
		{
			name: "http missing url",
			server: McpServer{
				ID:   "test",
				Name: "Test",
				Server: map[string]interface{}{
					"type": "http",
				},
			},
			wantErr: true,
			errMsg:  "url",
		},
		{
			name: "http missing host",
			server: McpServer{
				ID:   "test",
				Name: "Test",
				Server: map[string]interface{}{
					"type": "http",
					"url":  "http://",
				},
			},
			wantErr: true,
			errMsg:  "主机",
		},
		{
			name: "http invalid scheme",
			server: McpServer{
				ID:   "test",
				Name: "Test",
				Server: map[string]interface{}{
					"type": "http",
					"url":  "ftp://example.com",
				},
			},
			wantErr: true,
			errMsg:  "http",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMcpServer(&tt.server)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateMcpServer() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Fatalf("error %v does not contain %q", err, tt.errMsg)
			}
		})
	}
}

func TestMcpServerCRUD(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, m *Manager)
		action  func(m *Manager) error
		verify  func(t *testing.T, m *Manager)
		wantErr bool
	}{
		{
			name: "add and get",
			action: func(m *Manager) error {
				return m.AddMcpServer(McpServer{
					ID:   "srv",
					Name: "Server",
					Server: map[string]interface{}{
						"type":    "stdio",
						"command": "npx",
					},
				})
			},
			verify: func(t *testing.T, m *Manager) {
				got, err := m.GetMcpServer("srv")
				if err != nil {
					t.Fatalf("GetMcpServer() error = %v", err)
				}
				if got.Name != "Server" {
					t.Fatalf("name = %s", got.Name)
				}
			},
		},
		{
			name: "add missing id",
			action: func(m *Manager) error {
				return m.AddMcpServer(McpServer{
					Name: "Server",
					Server: map[string]interface{}{
						"type":    "stdio",
						"command": "npx",
					},
				})
			},
			wantErr: true,
		},
		{
			name: "duplicate id",
			setup: func(t *testing.T, m *Manager) {
				if err := m.AddMcpServer(McpServer{
					ID:   "dup",
					Name: "Server",
					Server: map[string]interface{}{
						"type":    "stdio",
						"command": "npx",
					},
				}); err != nil {
					t.Fatalf("setup add: %v", err)
				}
			},
			action: func(m *Manager) error {
				return m.AddMcpServer(McpServer{
					ID:   "dup",
					Name: "Server2",
					Server: map[string]interface{}{
						"type":    "stdio",
						"command": "npx",
					},
				})
			},
			wantErr: true,
		},
		{
			name: "update missing id",
			action: func(m *Manager) error {
				return m.UpdateMcpServer(McpServer{
					Name: "Updated",
					Server: map[string]interface{}{
						"type":    "stdio",
						"command": "npx",
					},
				})
			},
			wantErr: true,
		},
		{
			name: "update server",
			setup: func(t *testing.T, m *Manager) {
				if err := m.AddMcpServer(McpServer{
					ID:   "srv",
					Name: "Server",
					Server: map[string]interface{}{
						"type":    "stdio",
						"command": "npx",
					},
				}); err != nil {
					t.Fatalf("setup add: %v", err)
				}
			},
			action: func(m *Manager) error {
				return m.UpdateMcpServer(McpServer{
					ID:   "srv",
					Name: "Updated",
					Server: map[string]interface{}{
						"type":    "stdio",
						"command": "npx",
					},
				})
			},
			verify: func(t *testing.T, m *Manager) {
				got, _ := m.GetMcpServer("srv")
				if got.Name != "Updated" {
					t.Fatalf("name = %s", got.Name)
				}
			},
		},
		{
			name: "update missing",
			action: func(m *Manager) error {
				return m.UpdateMcpServer(McpServer{
					ID:   "missing",
					Name: "Updated",
					Server: map[string]interface{}{
						"type":    "stdio",
						"command": "npx",
					},
				})
			},
			wantErr: true,
		},
		{
			name: "delete server",
			setup: func(t *testing.T, m *Manager) {
				if err := m.AddMcpServer(McpServer{
					ID:   "srv",
					Name: "Server",
					Server: map[string]interface{}{
						"type":    "stdio",
						"command": "npx",
					},
				}); err != nil {
					t.Fatalf("setup add: %v", err)
				}
			},
			action: func(m *Manager) error { return m.DeleteMcpServer("srv") },
			verify: func(t *testing.T, m *Manager) {
				if _, err := m.GetMcpServer("srv"); err == nil {
					t.Fatalf("expected error after delete")
				}
			},
		},
		{
			name:    "delete missing",
			action:  func(m *Manager) error { return m.DeleteMcpServer("missing") },
			wantErr: true,
		},
		{
			name:    "toggle missing server",
			action:  func(m *Manager) error { return m.ToggleMcpApp("missing", "claude", true) },
			wantErr: true,
		},
		{
			name: "toggle app",
			setup: func(t *testing.T, m *Manager) {
				if err := m.AddMcpServer(McpServer{
					ID:   "srv",
					Name: "Server",
					Server: map[string]interface{}{
						"type":    "stdio",
						"command": "npx",
					},
				}); err != nil {
					t.Fatalf("setup add: %v", err)
				}
			},
			action: func(m *Manager) error { return m.ToggleMcpApp("srv", "claude", true) },
			verify: func(t *testing.T, m *Manager) {
				server, _ := m.GetMcpServer("srv")
				if !server.Apps.Claude {
					t.Fatalf("claude app should be enabled")
				}
			},
		},
		{
			name: "toggle codex",
			setup: func(t *testing.T, m *Manager) {
				if err := m.AddMcpServer(McpServer{
					ID:   "srv",
					Name: "Server",
					Server: map[string]interface{}{
						"type":    "stdio",
						"command": "npx",
					},
				}); err != nil {
					t.Fatalf("setup add: %v", err)
				}
			},
			action: func(m *Manager) error { return m.ToggleMcpApp("srv", "codex", true) },
			verify: func(t *testing.T, m *Manager) {
				server, _ := m.GetMcpServer("srv")
				if !server.Apps.Codex {
					t.Fatalf("codex app should be enabled")
				}
			},
		},
		{
			name: "toggle gemini",
			setup: func(t *testing.T, m *Manager) {
				if err := m.AddMcpServer(McpServer{
					ID:   "srv",
					Name: "Server",
					Server: map[string]interface{}{
						"type":    "stdio",
						"command": "npx",
					},
				}); err != nil {
					t.Fatalf("setup add: %v", err)
				}
			},
			action: func(m *Manager) error { return m.ToggleMcpApp("srv", "gemini", true) },
			verify: func(t *testing.T, m *Manager) {
				server, _ := m.GetMcpServer("srv")
				if !server.Apps.Gemini {
					t.Fatalf("gemini app should be enabled")
				}
			},
		},
		{
			name: "toggle unknown app",
			setup: func(t *testing.T, m *Manager) {
				if err := m.AddMcpServer(McpServer{
					ID:   "srv",
					Name: "Server",
					Server: map[string]interface{}{
						"type":    "stdio",
						"command": "npx",
					},
				}); err != nil {
					t.Fatalf("setup add: %v", err)
				}
			},
			action:  func(m *Manager) error { return m.ToggleMcpApp("srv", "unknown", true) },
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
				tt.verify(t, m)
			}
		})
	}
}

func TestEnsureMcpRoot(t *testing.T) {
	tests := []struct {
		name  string
		setup func(m *Manager)
	}{
		{
			name: "nil root",
			setup: func(m *Manager) {
				m.config.Mcp = nil
			},
		},
		{
			name: "nil servers",
			setup: func(m *Manager) {
				m.config.Mcp = &McpRoot{}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{config: &MultiAppConfig{}}
			if tt.setup != nil {
				tt.setup(m)
			}
			m.ensureMcpRoot()
			if m.config.Mcp == nil {
				t.Fatalf("mcp root not initialized")
			}
			if m.config.Mcp.Servers == nil {
				t.Fatalf("mcp servers not initialized")
			}
		})
	}
}

func TestListMcpServersSorted(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "sorted by id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m, err := NewManagerWithDir(dir)
			if err != nil {
				t.Fatalf("NewManagerWithDir() error = %v", err)
			}
			if err := m.AddMcpServer(McpServer{ID: "b", Name: "B", Server: map[string]interface{}{"type": "stdio", "command": "npx"}}); err != nil {
				t.Fatalf("add server: %v", err)
			}
			if err := m.AddMcpServer(McpServer{ID: "a", Name: "A", Server: map[string]interface{}{"type": "stdio", "command": "npx"}}); err != nil {
				t.Fatalf("add server: %v", err)
			}
			list := m.ListMcpServers()
			if len(list) != 2 || list[0].ID != "a" || list[1].ID != "b" {
				t.Fatalf("unexpected order: %v", list)
			}
		})
	}
}

func TestSyncMcpServerRemovals(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "remove disabled servers"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m, err := NewManagerWithDir(dir)
			if err != nil {
				t.Fatalf("NewManagerWithDir() error = %v", err)
			}

			server := McpServer{
				ID:   "srv",
				Name: "Server",
				Server: map[string]interface{}{
					"type":    "stdio",
					"command": "npx",
				},
				Apps: McpApps{Claude: false, Codex: false, Gemini: false},
			}
			m.ensureMcpRoot()
			m.config.Mcp.Servers[server.ID] = server

			claudePath, _ := m.GetClaudeSettingsPathWithDir()
			if err := os.MkdirAll(filepath.Dir(claudePath), 0700); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			claudeSettings := map[string]interface{}{"mcpServers": map[string]interface{}{"srv": server.Server}}
			claudeData, _ := json.MarshalIndent(claudeSettings, "", "  ")
			writeFile(t, claudePath, claudeData)

			codexPath, _ := m.GetCodexConfigPathWithDir()
			if err := os.MkdirAll(filepath.Dir(codexPath), 0700); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			codexConfig := map[string]interface{}{"mcp_servers": map[string]interface{}{"srv": server.Server}}
			codexData, _ := toml.Marshal(codexConfig)
			writeFile(t, codexPath, codexData)

			geminiPath, _ := m.GetGeminiSettingsPathWithDir()
			if err := os.MkdirAll(filepath.Dir(geminiPath), 0700); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			geminiSettings := map[string]interface{}{"mcpServers": map[string]interface{}{"srv": server.Server}}
			geminiData, _ := json.MarshalIndent(geminiSettings, "", "  ")
			writeFile(t, geminiPath, geminiData)

			if err := m.SyncMcpServer("srv"); err != nil {
				t.Fatalf("SyncMcpServer() error = %v", err)
			}

			claudeRaw := readJSONMapFile(t, claudePath)
			if mcpServers, ok := claudeRaw["mcpServers"].(map[string]interface{}); ok {
				if _, exists := mcpServers["srv"]; exists {
					t.Fatalf("server should be removed from claude")
				}
			}

			data, err := os.ReadFile(codexPath)
			if err != nil {
				t.Fatalf("read codex: %v", err)
			}
			var parsed map[string]interface{}
			if err := toml.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("parse codex: %v", err)
			}
			if servers, ok := parsed["mcp_servers"].(map[string]interface{}); ok {
				if _, exists := servers["srv"]; exists {
					t.Fatalf("server should be removed from codex")
				}
			}

			geminiRaw := readJSONMapFile(t, geminiPath)
			if mcpServers, ok := geminiRaw["mcpServers"].(map[string]interface{}); ok {
				if _, exists := mcpServers["srv"]; exists {
					t.Fatalf("server should be removed from gemini")
				}
			}
		})
	}
}

func TestSyncMcpServerScenarios(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, m *Manager, dir string)
		server  string
		wantErr bool
		verify  func(t *testing.T, dir string)
	}{
		{
			name:    "missing server",
			server:  "missing",
			wantErr: true,
		},
		{
			name:   "sync all apps with existing files",
			server: "srv",
			setup: func(t *testing.T, m *Manager, dir string) {
				m.ensureMcpRoot()
				m.config.Mcp.Servers["srv"] = McpServer{
					ID:   "srv",
					Name: "Server",
					Server: map[string]interface{}{
						"type":    "stdio",
						"command": "npx",
					},
					Apps: McpApps{Claude: true, Codex: true, Gemini: true},
				}

				claudePath, _ := m.GetClaudeSettingsPathWithDir()
				if err := os.MkdirAll(filepath.Dir(claudePath), 0700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				claudeSeed := `{"permissions":{"allow":[],"deny":[]},"mcpServers":{"old":{}}}`
				writeFile(t, claudePath, []byte(claudeSeed))

				codexPath, _ := m.GetCodexConfigPathWithDir()
				if err := os.MkdirAll(filepath.Dir(codexPath), 0700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				codexSeed := map[string]interface{}{
					"mcp_servers": map[string]interface{}{
						"old": map[string]interface{}{"type": "stdio", "command": "old"},
					},
				}
				codexData, _ := toml.Marshal(codexSeed)
				writeFile(t, codexPath, codexData)

				geminiPath, _ := m.GetGeminiSettingsPathWithDir()
				if err := os.MkdirAll(filepath.Dir(geminiPath), 0700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				geminiSeed := `{"security":{"auth":{"selectedType":"gemini-api-key"}},"mcpServers":{"old":{}},"extra":"keep"}`
				writeFile(t, geminiPath, []byte(geminiSeed))
			},
			verify: func(t *testing.T, dir string) {
				claudePath := filepath.Join(dir, ".claude", "settings.json")
				claudeRaw := readJSONMapFile(t, claudePath)
				claudeMcp := claudeRaw["mcpServers"].(map[string]interface{})
				if claudeMcp["srv"] == nil || claudeMcp["old"] == nil {
					t.Fatalf("claude mcpServers missing entries: %v", claudeMcp)
				}

				codexPath := filepath.Join(dir, ".codex", "config.toml")
				data, err := os.ReadFile(codexPath)
				if err != nil {
					t.Fatalf("read codex: %v", err)
				}
				var parsed map[string]interface{}
				if err := toml.Unmarshal(data, &parsed); err != nil {
					t.Fatalf("parse codex: %v", err)
				}
				codexMcp := parsed["mcp_servers"].(map[string]interface{})
				if codexMcp["srv"] == nil || codexMcp["old"] == nil {
					t.Fatalf("codex mcpServers missing entries: %v", codexMcp)
				}

				geminiPath := filepath.Join(dir, ".gemini", "settings.json")
				geminiRaw := readJSONMapFile(t, geminiPath)
				geminiMcp := geminiRaw["mcpServers"].(map[string]interface{})
				if geminiMcp["srv"] == nil || geminiMcp["old"] == nil {
					t.Fatalf("gemini mcpServers missing entries: %v", geminiMcp)
				}
				if geminiRaw["extra"] != "keep" {
					t.Fatalf("gemini extra not preserved")
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
				tt.setup(t, m, dir)
			}
			err = m.SyncMcpServer(tt.server)
			if (err != nil) != tt.wantErr {
				t.Fatalf("SyncMcpServer() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && tt.verify != nil {
				tt.verify(t, dir)
			}
		})
	}
}

func TestSyncAllMcpServersBatch(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "batch sync"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m, err := NewManagerWithDir(dir)
			if err != nil {
				t.Fatalf("NewManagerWithDir() error = %v", err)
			}
			m.ensureMcpRoot()
			m.config.Mcp.Servers["srv1"] = McpServer{
				ID:   "srv1",
				Name: "Server1",
				Server: map[string]interface{}{
					"type":    "stdio",
					"command": "npx",
				},
				Apps: McpApps{Claude: true, Codex: true, Gemini: false},
			}
			m.config.Mcp.Servers["srv2"] = McpServer{
				ID:   "srv2",
				Name: "Server2",
				Server: map[string]interface{}{
					"type": "http",
					"url":  "https://example.com",
				},
				Apps: McpApps{Claude: false, Codex: false, Gemini: true},
			}

			claudePath, _ := m.GetClaudeSettingsPathWithDir()
			if err := os.MkdirAll(filepath.Dir(claudePath), 0700); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			claudeSeed := `{"extra":"keep","mcpServers":{"old":{}}}`
			writeFile(t, claudePath, []byte(claudeSeed))

			geminiPath, _ := m.GetGeminiSettingsPathWithDir()
			if err := os.MkdirAll(filepath.Dir(geminiPath), 0700); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			geminiSeed := `{"extra":"keep","mcpServers":{"old":{}}}`
			writeFile(t, geminiPath, []byte(geminiSeed))

			if err := m.SyncAllMcpServersBatch(); err != nil {
				t.Fatalf("SyncAllMcpServersBatch() error = %v", err)
			}

			claudeRaw := readJSONMapFile(t, claudePath)
			if claudeRaw["extra"] != "keep" {
				t.Fatalf("extra field should be preserved")
			}
			claudeMcp := claudeRaw["mcpServers"].(map[string]interface{})
			if len(claudeMcp) != 1 || claudeMcp["srv1"] == nil {
				t.Fatalf("claude mcpServers mismatch: %v", claudeMcp)
			}

			codexConfigPath, err := m.GetCodexConfigPathWithDir()
			if err != nil {
				t.Fatalf("GetCodexConfigPathWithDir() error = %v", err)
			}
			mcpPath := filepath.Join(filepath.Dir(codexConfigPath), "mcp.json")
			data, err := os.ReadFile(mcpPath)
			if err != nil {
				t.Fatalf("read codex mcp.json: %v", err)
			}
			var codexRaw map[string]interface{}
			if err := json.Unmarshal(data, &codexRaw); err != nil {
				t.Fatalf("unmarshal codex mcp.json: %v", err)
			}
			codexMcp := codexRaw["mcpServers"].(map[string]interface{})
			if len(codexMcp) != 1 || codexMcp["srv1"] == nil {
				t.Fatalf("codex mcpServers mismatch: %v", codexMcp)
			}

			geminiRaw := readJSONMapFile(t, geminiPath)
			if geminiRaw["extra"] != "keep" {
				t.Fatalf("extra field should be preserved")
			}
			geminiMcp := geminiRaw["mcpServers"].(map[string]interface{})
			if len(geminiMcp) != 1 || geminiMcp["srv2"] == nil {
				t.Fatalf("gemini mcpServers mismatch: %v", geminiMcp)
			}
		})
	}
}

func TestGetMcpPresets(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "presets"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			presets := GetMcpPresets()
			if len(presets) == 0 {
				t.Fatalf("expected presets")
			}
			for _, preset := range presets {
				if preset.ID == "" {
					t.Fatalf("preset id missing")
				}
			}
		})
	}
}

func TestSyncMcpToClaude(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{name: "enabled", enabled: true},
		{name: "disabled", enabled: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m, err := NewManagerWithDir(dir)
			if err != nil {
				t.Fatalf("NewManagerWithDir() error = %v", err)
			}
			server := McpServer{
				ID:   "srv",
				Name: "Server",
				Server: map[string]interface{}{
					"type":    "stdio",
					"command": "npx",
				},
				Apps: McpApps{Claude: tt.enabled},
			}
			m.ensureMcpRoot()
			m.config.Mcp.Servers[server.ID] = server

			settingsPath, _ := m.GetClaudeSettingsPathWithDir()
			if !tt.enabled {
				if err := os.MkdirAll(filepath.Dir(settingsPath), 0700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				seed := map[string]interface{}{"mcpServers": map[string]interface{}{"srv": server.Server}}
				seedData, _ := json.MarshalIndent(seed, "", "  ")
				writeFile(t, settingsPath, seedData)
			}

			if err := m.SyncMcpToClaud("srv"); err != nil {
				t.Fatalf("SyncMcpToClaud() error = %v", err)
			}

			raw := readJSONMapFile(t, settingsPath)
			mcp := raw["mcpServers"].(map[string]interface{})
			if tt.enabled {
				if _, ok := mcp["srv"]; !ok {
					t.Fatalf("server not synced")
				}
			} else if _, ok := mcp["srv"]; ok {
				t.Fatalf("server should be removed")
			}
		})
	}
}

func TestSyncMcpToCodex(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{name: "enabled", enabled: true},
		{name: "disabled", enabled: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m, err := NewManagerWithDir(dir)
			if err != nil {
				t.Fatalf("NewManagerWithDir() error = %v", err)
			}
			server := McpServer{
				ID:   "srv",
				Name: "Server",
				Server: map[string]interface{}{
					"type":    "stdio",
					"command": "npx",
				},
				Apps: McpApps{Codex: tt.enabled},
			}
			m.ensureMcpRoot()
			m.config.Mcp.Servers[server.ID] = server

			configPath, _ := m.GetCodexConfigPathWithDir()
			if !tt.enabled {
				if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				seed := map[string]interface{}{"mcp_servers": map[string]interface{}{"srv": server.Server}}
				seedData, _ := toml.Marshal(seed)
				writeFile(t, configPath, seedData)
			}

			if err := m.SyncMcpToCodex("srv"); err != nil {
				t.Fatalf("SyncMcpToCodex() error = %v", err)
			}

			data, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("read config: %v", err)
			}
			var parsed map[string]interface{}
			if err := toml.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("parse config: %v", err)
			}
			mcp := parsed["mcp_servers"].(map[string]interface{})
			if tt.enabled {
				if _, ok := mcp["srv"]; !ok {
					t.Fatalf("server not synced")
				}
			} else if _, ok := mcp["srv"]; ok {
				t.Fatalf("server should be removed")
			}
		})
	}
}

func TestSyncMcpToGemini(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{name: "enabled", enabled: true},
		{name: "disabled", enabled: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m, err := NewManagerWithDir(dir)
			if err != nil {
				t.Fatalf("NewManagerWithDir() error = %v", err)
			}
			server := McpServer{
				ID:   "srv",
				Name: "Server",
				Server: map[string]interface{}{
					"type":    "stdio",
					"command": "npx",
				},
				Apps: McpApps{Gemini: tt.enabled},
			}
			m.ensureMcpRoot()
			m.config.Mcp.Servers[server.ID] = server

			settingsPath, _ := m.GetGeminiSettingsPathWithDir()
			if !tt.enabled {
				if err := os.MkdirAll(filepath.Dir(settingsPath), 0700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				seed := map[string]interface{}{"mcpServers": map[string]interface{}{"srv": server.Server}}
				seedData, _ := json.MarshalIndent(seed, "", "  ")
				writeFile(t, settingsPath, seedData)
			}

			if err := m.SyncMcpToGemini("srv"); err != nil {
				t.Fatalf("SyncMcpToGemini() error = %v", err)
			}

			raw := readJSONMapFile(t, settingsPath)
			mcp, _ := raw["mcpServers"].(map[string]interface{})
			if tt.enabled {
				if mcp == nil {
					t.Fatalf("mcpServers missing")
				}
				if _, ok := mcp["srv"]; !ok {
					t.Fatalf("server not synced")
				}
			} else if mcp != nil {
				if _, ok := mcp["srv"]; ok {
					t.Fatalf("server should be removed")
				}
			}
		})
	}
}

func TestSyncAllMcpServers(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "sync all"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m, err := NewManagerWithDir(dir)
			if err != nil {
				t.Fatalf("NewManagerWithDir() error = %v", err)
			}
			m.ensureMcpRoot()
			m.config.Mcp.Servers["srv1"] = McpServer{
				ID:   "srv1",
				Name: "Server1",
				Server: map[string]interface{}{
					"type":    "stdio",
					"command": "npx",
				},
				Apps: McpApps{Claude: true},
			}

			if err := m.SyncAllMcpServers(); err != nil {
				t.Fatalf("SyncAllMcpServers() error = %v", err)
			}

			settingsPath, _ := m.GetClaudeSettingsPathWithDir()
			raw := readJSONMapFile(t, settingsPath)
			mcp := raw["mcpServers"].(map[string]interface{})
			if _, ok := mcp["srv1"]; !ok {
				t.Fatalf("server not synced")
			}
		})
	}
}

func TestSyncMcpBatchWithoutExistingFiles(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "batch without existing files"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m := &Manager{customDir: dir}
			servers := map[string]interface{}{
				"srv": map[string]interface{}{"type": "stdio", "command": "npx"},
			}

			if err := m.syncMcpToClaudeBatch(servers); err != nil {
				t.Fatalf("syncMcpToClaudeBatch() error = %v", err)
			}
			if err := m.syncMcpToCodexBatch(servers); err != nil {
				t.Fatalf("syncMcpToCodexBatch() error = %v", err)
			}
			if err := m.syncMcpToGeminiBatch(servers); err != nil {
				t.Fatalf("syncMcpToGeminiBatch() error = %v", err)
			}

			claudePath, _ := m.GetClaudeSettingsPathWithDir()
			raw := readJSONMapFile(t, claudePath)
			if raw["mcpServers"] == nil {
				t.Fatalf("claude mcpServers missing")
			}

			codexPath, _ := m.GetCodexConfigPathWithDir()
			mcpPath := filepath.Join(filepath.Dir(codexPath), "mcp.json")
			if _, err := os.Stat(mcpPath); err != nil {
				t.Fatalf("codex mcp.json missing: %v", err)
			}

			geminiPath, _ := m.GetGeminiSettingsPathWithDir()
			raw = readJSONMapFile(t, geminiPath)
			if raw["mcpServers"] == nil {
				t.Fatalf("gemini mcpServers missing")
			}
		})
	}
}

func TestRemoveMcpWhenFilesMissing(t *testing.T) {
	tests := []struct {
		name string
		call func(m *Manager) error
	}{
		{
			name: "remove from claude",
			call: func(m *Manager) error { return m.RemoveMcpFromClaude("missing") },
		},
		{
			name: "remove from codex",
			call: func(m *Manager) error { return m.RemoveMcpFromCodex("missing") },
		},
		{
			name: "remove from gemini",
			call: func(m *Manager) error { return m.RemoveMcpFromGemini("missing") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m := &Manager{customDir: dir}
			if err := tt.call(m); err != nil {
				t.Fatalf("remove error = %v", err)
			}
		})
	}
}

func TestRemoveMcpWithoutServersMap(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(t *testing.T, m *Manager, dir string)
		call   func(m *Manager) error
		verify func(t *testing.T, dir string)
	}{
		{
			name: "codex no mcp_servers",
			setup: func(t *testing.T, m *Manager, dir string) {
				configPath, _ := m.GetCodexConfigPathWithDir()
				if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				writeFile(t, configPath, []byte(`model = "gpt-5"`))
			},
			call: func(m *Manager) error { return m.RemoveMcpFromCodex("missing") },
			verify: func(t *testing.T, dir string) {
				configPath := filepath.Join(dir, ".codex", "config.toml")
				data, err := os.ReadFile(configPath)
				if err != nil {
					t.Fatalf("read codex: %v", err)
				}
				if !strings.Contains(string(data), "model") {
					t.Fatalf("codex config missing model")
				}
			},
		},
		{
			name: "gemini no mcpServers",
			setup: func(t *testing.T, m *Manager, dir string) {
				settingsPath, _ := m.GetGeminiSettingsPathWithDir()
				if err := os.MkdirAll(filepath.Dir(settingsPath), 0700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				writeFile(t, settingsPath, []byte(`{"security":{"auth":{"selectedType":"gemini-api-key"}}}`))
			},
			call: func(m *Manager) error { return m.RemoveMcpFromGemini("missing") },
			verify: func(t *testing.T, dir string) {
				settingsPath := filepath.Join(dir, ".gemini", "settings.json")
				raw := readJSONMapFile(t, settingsPath)
				if raw["mcpServers"] != nil {
					t.Fatalf("unexpected mcpServers")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m := &Manager{customDir: dir}
			if tt.setup != nil {
				tt.setup(t, m, dir)
			}
			if err := tt.call(m); err != nil {
				t.Fatalf("remove error = %v", err)
			}
			if tt.verify != nil {
				tt.verify(t, dir)
			}
		})
	}
}

func TestRemoveMcpInvalidFiles(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, m *Manager, dir string)
		call  func(m *Manager) error
	}{
		{
			name: "claude invalid json",
			setup: func(t *testing.T, m *Manager, dir string) {
				settingsPath, _ := m.GetClaudeSettingsPathWithDir()
				if err := os.MkdirAll(filepath.Dir(settingsPath), 0700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				writeFile(t, settingsPath, []byte("{"))
			},
			call: func(m *Manager) error { return m.RemoveMcpFromClaude("srv") },
		},
		{
			name: "codex invalid toml",
			setup: func(t *testing.T, m *Manager, dir string) {
				configPath, _ := m.GetCodexConfigPathWithDir()
				if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				writeFile(t, configPath, []byte("invalid ="))
			},
			call: func(m *Manager) error { return m.RemoveMcpFromCodex("srv") },
		},
		{
			name: "gemini invalid json",
			setup: func(t *testing.T, m *Manager, dir string) {
				settingsPath, _ := m.GetGeminiSettingsPathWithDir()
				if err := os.MkdirAll(filepath.Dir(settingsPath), 0700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				writeFile(t, settingsPath, []byte("{"))
			},
			call: func(m *Manager) error { return m.RemoveMcpFromGemini("srv") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m := &Manager{customDir: dir}
			if tt.setup != nil {
				tt.setup(t, m, dir)
			}
			if err := tt.call(m); err == nil {
				t.Fatalf("expected remove error")
			}
		})
	}
}

func TestSyncMcpServerErrorAggregation(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "sync error"},
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
			server := McpServer{
				ID:   "srv",
				Name: "Server",
				Server: map[string]interface{}{
					"type":    "stdio",
					"command": "npx",
				},
				Apps: McpApps{Claude: true},
			}
			m.ensureMcpRoot()
			m.config.Mcp.Servers[server.ID] = server

			claudeDir := filepath.Join(dir, ".claude")
			if err := os.MkdirAll(claudeDir, 0755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			if err := os.Chmod(claudeDir, 0500); err != nil {
				t.Fatalf("chmod: %v", err)
			}
			t.Cleanup(func() {
				_ = os.Chmod(claudeDir, 0700)
			})

			if err := m.SyncMcpServer("srv"); err == nil {
				t.Fatalf("expected sync error")
			}
		})
	}
}

func TestSyncAllMcpServersPartialFailure(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "partial failure"},
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
			m.ensureMcpRoot()
			m.config.Mcp.Servers["srv"] = McpServer{
				ID:   "srv",
				Name: "Server",
				Server: map[string]interface{}{
					"type":    "stdio",
					"command": "npx",
				},
				Apps: McpApps{Claude: true},
			}

			claudeDir := filepath.Join(dir, ".claude")
			if err := os.MkdirAll(claudeDir, 0755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			if err := os.Chmod(claudeDir, 0500); err != nil {
				t.Fatalf("chmod: %v", err)
			}
			t.Cleanup(func() {
				_ = os.Chmod(claudeDir, 0700)
			})

			if err := m.SyncAllMcpServers(); err == nil {
				t.Fatalf("expected sync error")
			}
		})
	}
}

func TestSyncMcpToAppErrors(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, m *Manager, dir string)
		call  func(m *Manager) error
	}{
		{
			name: "claude invalid json",
			setup: func(t *testing.T, m *Manager, dir string) {
				m.ensureMcpRoot()
				m.config.Mcp.Servers["srv"] = McpServer{
					ID:   "srv",
					Name: "Server",
					Server: map[string]interface{}{
						"type":    "stdio",
						"command": "npx",
					},
					Apps: McpApps{Claude: true},
				}
				settingsPath, _ := m.GetClaudeSettingsPathWithDir()
				if err := os.MkdirAll(filepath.Dir(settingsPath), 0700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				writeFile(t, settingsPath, []byte("{"))
			},
			call: func(m *Manager) error { return m.SyncMcpToClaud("srv") },
		},
		{
			name: "codex invalid toml",
			setup: func(t *testing.T, m *Manager, dir string) {
				m.ensureMcpRoot()
				m.config.Mcp.Servers["srv"] = McpServer{
					ID:   "srv",
					Name: "Server",
					Server: map[string]interface{}{
						"type":    "stdio",
						"command": "npx",
					},
					Apps: McpApps{Codex: true},
				}
				configPath, _ := m.GetCodexConfigPathWithDir()
				if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				writeFile(t, configPath, []byte("invalid ="))
			},
			call: func(m *Manager) error { return m.SyncMcpToCodex("srv") },
		},
		{
			name: "gemini invalid json",
			setup: func(t *testing.T, m *Manager, dir string) {
				m.ensureMcpRoot()
				m.config.Mcp.Servers["srv"] = McpServer{
					ID:   "srv",
					Name: "Server",
					Server: map[string]interface{}{
						"type":    "stdio",
						"command": "npx",
					},
					Apps: McpApps{Gemini: true},
				}
				settingsPath, _ := m.GetGeminiSettingsPathWithDir()
				if err := os.MkdirAll(filepath.Dir(settingsPath), 0700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				writeFile(t, settingsPath, []byte("{"))
			},
			call: func(m *Manager) error { return m.SyncMcpToGemini("srv") },
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
				tt.setup(t, m, dir)
			}
			if err := tt.call(m); err == nil {
				t.Fatalf("expected sync error")
			}
		})
	}
}

func TestSyncMcpToAppMkdirErrors(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, m *Manager, dir string)
		call  func(m *Manager) error
	}{
		{
			name: "claude mkdir error",
			setup: func(t *testing.T, m *Manager, dir string) {
				m.ensureMcpRoot()
				m.config.Mcp.Servers["srv"] = McpServer{
					ID:   "srv",
					Name: "Server",
					Server: map[string]interface{}{
						"type":    "stdio",
						"command": "npx",
					},
					Apps: McpApps{Claude: true},
				}
				writeFile(t, filepath.Join(dir, ".claude"), []byte("file"))
			},
			call: func(m *Manager) error { return m.SyncMcpToClaud("srv") },
		},
		{
			name: "codex mkdir error",
			setup: func(t *testing.T, m *Manager, dir string) {
				m.ensureMcpRoot()
				m.config.Mcp.Servers["srv"] = McpServer{
					ID:   "srv",
					Name: "Server",
					Server: map[string]interface{}{
						"type":    "stdio",
						"command": "npx",
					},
					Apps: McpApps{Codex: true},
				}
				writeFile(t, filepath.Join(dir, ".codex"), []byte("file"))
			},
			call: func(m *Manager) error { return m.SyncMcpToCodex("srv") },
		},
		{
			name: "gemini mkdir error",
			setup: func(t *testing.T, m *Manager, dir string) {
				m.ensureMcpRoot()
				m.config.Mcp.Servers["srv"] = McpServer{
					ID:   "srv",
					Name: "Server",
					Server: map[string]interface{}{
						"type":    "stdio",
						"command": "npx",
					},
					Apps: McpApps{Gemini: true},
				}
				writeFile(t, filepath.Join(dir, ".gemini"), []byte("file"))
			},
			call: func(m *Manager) error { return m.SyncMcpToGemini("srv") },
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
				tt.setup(t, m, dir)
			}
			if err := tt.call(m); err == nil {
				t.Fatalf("expected sync error")
			}
		})
	}
}

func TestSyncMcpBatchErrors(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, m *Manager, dir string)
		call  func(m *Manager) error
	}{
		{
			name: "claude batch invalid json",
			setup: func(t *testing.T, m *Manager, dir string) {
				settingsPath, _ := m.GetClaudeSettingsPathWithDir()
				if err := os.MkdirAll(filepath.Dir(settingsPath), 0700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				writeFile(t, settingsPath, []byte("{"))
			},
			call: func(m *Manager) error {
				return m.syncMcpToClaudeBatch(map[string]interface{}{"srv": map[string]interface{}{"type": "stdio"}})
			},
		},
		{
			name: "gemini batch invalid json",
			setup: func(t *testing.T, m *Manager, dir string) {
				settingsPath, _ := m.GetGeminiSettingsPathWithDir()
				if err := os.MkdirAll(filepath.Dir(settingsPath), 0700); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				writeFile(t, settingsPath, []byte("{"))
			},
			call: func(m *Manager) error {
				return m.syncMcpToGeminiBatch(map[string]interface{}{"srv": map[string]interface{}{"type": "stdio"}})
			},
		},
		{
			name: "gemini batch mkdir error",
			setup: func(t *testing.T, m *Manager, dir string) {
				writeFile(t, filepath.Join(dir, ".gemini"), []byte("file"))
			},
			call: func(m *Manager) error {
				return m.syncMcpToGeminiBatch(map[string]interface{}{"srv": map[string]interface{}{"type": "stdio"}})
			},
		},
		{
			name: "codex batch mkdir error",
			setup: func(t *testing.T, m *Manager, dir string) {
				codexDir := filepath.Join(dir, ".codex")
				writeFile(t, codexDir, []byte("file"))
			},
			call: func(m *Manager) error {
				return m.syncMcpToCodexBatch(map[string]interface{}{"srv": map[string]interface{}{"type": "stdio"}})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			setTempHome(t, dir)
			m := &Manager{customDir: dir}
			if tt.setup != nil {
				tt.setup(t, m, dir)
			}
			if err := tt.call(m); err == nil {
				t.Fatalf("expected batch error")
			}
		})
	}
}
