package config

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateMcpServer(t *testing.T) {
	tests := []struct {
		name    string
		server  McpServer
		wantErr bool
		errMsg  string
	}{
		{
			name: "有效的stdio类型",
			server: McpServer{
				ID:   "test-server",
				Name: "Test Server",
				Server: map[string]interface{}{
					"type":    "stdio",
					"command": "npx",
				},
			},
			wantErr: false,
		},
		{
			name: "有效的http类型",
			server: McpServer{
				ID:   "test-server",
				Name: "Test Server",
				Server: map[string]interface{}{
					"type": "http",
					"url":  "https://example.com/api",
				},
			},
			wantErr: false,
		},
		{
			name: "有效的sse类型",
			server: McpServer{
				ID:   "test-server",
				Name: "Test Server",
				Server: map[string]interface{}{
					"type": "sse",
					"url":  "http://localhost:8080/sse",
				},
			},
			wantErr: false,
		},
		{
			name: "ID为空",
			server: McpServer{
				ID:   "",
				Name: "Test Server",
				Server: map[string]interface{}{
					"type":    "stdio",
					"command": "npx",
				},
			},
			wantErr: true,
			errMsg:  "ID 不能为空",
		},
		{
			name: "Name为空",
			server: McpServer{
				ID:   "test-server",
				Name: "",
				Server: map[string]interface{}{
					"type":    "stdio",
					"command": "npx",
				},
			},
			wantErr: true,
			errMsg:  "名称不能为空",
		},
		{
			name: "缺少type字段",
			server: McpServer{
				ID:   "test-server",
				Name: "Test Server",
				Server: map[string]interface{}{
					"command": "npx",
				},
			},
			wantErr: true,
			errMsg:  "缺少 type 字段",
		},
		{
			name: "不支持的连接类型",
			server: McpServer{
				ID:   "test-server",
				Name: "Test Server",
				Server: map[string]interface{}{
					"type": "websocket",
				},
			},
			wantErr: true,
			errMsg:  "不支持的连接类型",
		},
		{
			name: "stdio类型缺少command",
			server: McpServer{
				ID:   "test-server",
				Name: "Test Server",
				Server: map[string]interface{}{
					"type": "stdio",
				},
			},
			wantErr: true,
			errMsg:  "需要非空 command 字段",
		},
		{
			name: "stdio类型command为空字符串",
			server: McpServer{
				ID:   "test-server",
				Name: "Test Server",
				Server: map[string]interface{}{
					"type":    "stdio",
					"command": "   ",
				},
			},
			wantErr: true,
			errMsg:  "需要非空 command 字段",
		},
		{
			name: "http类型缺少url",
			server: McpServer{
				ID:   "test-server",
				Name: "Test Server",
				Server: map[string]interface{}{
					"type": "http",
				},
			},
			wantErr: true,
			errMsg:  "需要非空 url 字段",
		},
		{
			name: "http类型url为空字符串",
			server: McpServer{
				ID:   "test-server",
				Name: "Test Server",
				Server: map[string]interface{}{
					"type": "http",
					"url":  "  ",
				},
			},
			wantErr: true,
			errMsg:  "需要非空 url 字段",
		},
		{
			name: "http类型url使用非HTTP协议",
			server: McpServer{
				ID:   "test-server",
				Name: "Test Server",
				Server: map[string]interface{}{
					"type": "http",
					"url":  "ftp://example.com/file",
				},
			},
			wantErr: true,
			errMsg:  "必须使用 http 或 https 协议",
		},
		{
			name: "http类型url为相对路径",
			server: McpServer{
				ID:   "test-server",
				Name: "Test Server",
				Server: map[string]interface{}{
					"type": "http",
					"url":  "/api/endpoint",
				},
			},
			wantErr: true,
			errMsg:  "必须使用 http 或 https 协议",
		},
		{
			name: "http类型url缺少主机",
			server: McpServer{
				ID:   "test-server",
				Name: "Test Server",
				Server: map[string]interface{}{
					"type": "http",
					"url":  "http:///api",
				},
			},
			wantErr: true,
			errMsg:  "缺少主机地址",
		},
		{
			name: "sse类型url使用非HTTP协议",
			server: McpServer{
				ID:   "test-server",
				Name: "Test Server",
				Server: map[string]interface{}{
					"type": "sse",
					"url":  "ws://example.com/sse",
				},
			},
			wantErr: true,
			errMsg:  "必须使用 http 或 https 协议",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMcpServer(&tt.server)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMcpServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateMcpServer() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestMcpServerCRUD(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManagerWithDir(tmpDir)
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}

	// 测试添加
	server := McpServer{
		ID:   "test-mcp",
		Name: "Test MCP Server",
		Server: map[string]interface{}{
			"type":    "stdio",
			"command": "npx",
			"args":    []interface{}{"@modelcontextprotocol/server-test"},
		},
		Apps: McpApps{
			Claude: true,
			Codex:  false,
			Gemini: false,
		},
	}

	if err := manager.AddMcpServer(server); err != nil {
		t.Errorf("AddMcpServer() error = %v", err)
	}

	// 测试获取
	got, err := manager.GetMcpServer("test-mcp")
	if err != nil {
		t.Errorf("GetMcpServer() error = %v", err)
	}
	if got.ID != server.ID || got.Name != server.Name {
		t.Errorf("GetMcpServer() got = %v, want %v", got, server)
	}

	// 测试更新
	server.Name = "Updated MCP Server"
	if err := manager.UpdateMcpServer(server); err != nil {
		t.Errorf("UpdateMcpServer() error = %v", err)
	}

	got, _ = manager.GetMcpServer("test-mcp")
	if got.Name != "Updated MCP Server" {
		t.Errorf("UpdateMcpServer() name = %q, want %q", got.Name, "Updated MCP Server")
	}

	// 测试列表
	list := manager.ListMcpServers()
	if len(list) != 1 {
		t.Errorf("ListMcpServers() len = %d, want 1", len(list))
	}

	// 测试删除
	if err := manager.DeleteMcpServer("test-mcp"); err != nil {
		t.Errorf("DeleteMcpServer() error = %v", err)
	}

	_, err = manager.GetMcpServer("test-mcp")
	if err == nil {
		t.Error("GetMcpServer() should return error after delete")
	}
}

func TestMcpServerDuplicateID(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManagerWithDir(tmpDir)
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}

	server := McpServer{
		ID:   "duplicate-id",
		Name: "First Server",
		Server: map[string]interface{}{
			"type":    "stdio",
			"command": "npx",
		},
	}

	if err := manager.AddMcpServer(server); err != nil {
		t.Fatalf("第一次添加失败: %v", err)
	}

	// 尝试添加相同ID
	server.Name = "Second Server"
	err = manager.AddMcpServer(server)
	if err == nil {
		t.Error("AddMcpServer() should return error for duplicate ID")
	}
	if err != nil && !strings.Contains(err.Error(), "已存在") {
		t.Errorf("AddMcpServer() error = %v, want error containing '已存在'", err)
	}
}

func TestErrorsJoinInSync(t *testing.T) {
	// 测试 errors.Join 是否正确工作
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")

	joined := errors.Join(err1, err2)

	// 验证可以使用 errors.Is 检查
	if !errors.Is(joined, err1) {
		t.Error("errors.Is should return true for err1")
	}
	if !errors.Is(joined, err2) {
		t.Error("errors.Is should return true for err2")
	}
}
