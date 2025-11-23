package config

import (
	"fmt"
	neturl "net/url"
	"sort"
	"strings"
)

// ensureMcpRoot 确保 MCP 配置已初始化
func (m *Manager) ensureMcpRoot() {
	if m.config.Mcp == nil {
		m.config.Mcp = &McpRoot{
			Servers: make(map[string]McpServer),
		}
	}
	if m.config.Mcp.Servers == nil {
		m.config.Mcp.Servers = make(map[string]McpServer)
	}
}

// ListMcpServers 列出所有 MCP 服务器，按 ID 排序
func (m *Manager) ListMcpServers() []McpServer {
	m.ensureMcpRoot()

	servers := make([]McpServer, 0, len(m.config.Mcp.Servers))
	for _, server := range m.config.Mcp.Servers {
		servers = append(servers, server)
	}

	// 按 ID 排序
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].ID < servers[j].ID
	})

	return servers
}

// GetMcpServer 获取指定 ID 的 MCP 服务器
func (m *Manager) GetMcpServer(id string) (*McpServer, error) {
	m.ensureMcpRoot()

	server, exists := m.config.Mcp.Servers[id]
	if !exists {
		return nil, fmt.Errorf("MCP 服务器不存在: %s", id)
	}

	return &server, nil
}

// AddMcpServer 添加 MCP 服务器
func (m *Manager) AddMcpServer(server McpServer) error {
	m.ensureMcpRoot()

	// 验证 ID
	if server.ID == "" {
		return fmt.Errorf("MCP 服务器 ID 不能为空")
	}

	// 检查 ID 是否已存在
	if _, exists := m.config.Mcp.Servers[server.ID]; exists {
		return fmt.Errorf("MCP 服务器 ID 已存在: %s", server.ID)
	}

	// 验证服务器配置
	if err := validateMcpServer(&server); err != nil {
		return fmt.Errorf("验证失败: %w", err)
	}

	// 添加服务器
	m.config.Mcp.Servers[server.ID] = server

	return nil
}

// UpdateMcpServer 更新 MCP 服务器
func (m *Manager) UpdateMcpServer(server McpServer) error {
	m.ensureMcpRoot()

	// 验证 ID
	if server.ID == "" {
		return fmt.Errorf("MCP 服务器 ID 不能为空")
	}

	// 检查 ID 是否存在
	if _, exists := m.config.Mcp.Servers[server.ID]; !exists {
		return fmt.Errorf("MCP 服务器不存在: %s", server.ID)
	}

	// 验证服务器配置
	if err := validateMcpServer(&server); err != nil {
		return fmt.Errorf("验证失败: %w", err)
	}

	// 更新服务器
	m.config.Mcp.Servers[server.ID] = server

	return nil
}

// DeleteMcpServer 删除 MCP 服务器
func (m *Manager) DeleteMcpServer(id string) error {
	m.ensureMcpRoot()

	// 检查 ID 是否存在
	if _, exists := m.config.Mcp.Servers[id]; !exists {
		return fmt.Errorf("MCP 服务器不存在: %s", id)
	}

	// 删除服务器
	delete(m.config.Mcp.Servers, id)

	return nil
}

// ToggleMcpApp 切换 MCP 服务器的应用启用状态
func (m *Manager) ToggleMcpApp(serverID, appName string, enabled bool) error {
	m.ensureMcpRoot()

	// 获取服务器
	server, exists := m.config.Mcp.Servers[serverID]
	if !exists {
		return fmt.Errorf("MCP 服务器不存在: %s", serverID)
	}

	// 更新应用启用状态
	switch strings.ToLower(appName) {
	case "claude":
		server.Apps.Claude = enabled
	case "codex":
		server.Apps.Codex = enabled
	case "gemini":
		server.Apps.Gemini = enabled
	default:
		return fmt.Errorf("未知的应用名称: %s", appName)
	}

	// 保存更新
	m.config.Mcp.Servers[serverID] = server

	return nil
}

// validateMcpServer 验证 MCP 服务器配置
func validateMcpServer(server *McpServer) error {
	// 验证 ID
	if server.ID == "" {
		return fmt.Errorf("服务器 ID 不能为空")
	}

	// 验证名称
	if server.Name == "" {
		return fmt.Errorf("服务器名称不能为空")
	}

	// 验证服务器配置
	if server.Server == nil || len(server.Server) == 0 {
		return fmt.Errorf("服务器配置不能为空")
	}

	// 验证连接类型
	connType, ok := server.Server["type"].(string)
	if !ok {
		return fmt.Errorf("服务器配置缺少 type 字段")
	}

	// 根据连接类型验证必需字段
	switch connType {
	case "stdio":
		// stdio 需要 command 字段
		command, ok := server.Server["command"].(string)
		if !ok || strings.TrimSpace(command) == "" {
			return fmt.Errorf("stdio 类型需要非空 command 字段")
		}

	case "http", "sse":
		// http/sse 需要 url 字段
		urlStr, ok := server.Server["url"].(string)
		if !ok || strings.TrimSpace(urlStr) == "" {
			return fmt.Errorf("%s 类型需要非空 url 字段", connType)
		}

		// 验证 URL 格式
		parsedURL, err := neturl.Parse(strings.TrimSpace(urlStr))
		if err != nil {
			return fmt.Errorf("%s 类型的 url 格式无效: %w", connType, err)
		}

		// 验证 scheme 必须是 http 或 https
		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			return fmt.Errorf("%s 类型的 url 必须使用 http 或 https 协议", connType)
		}

		// 验证 Host 非空
		if parsedURL.Host == "" {
			return fmt.Errorf("%s 类型的 url 缺少主机地址", connType)
		}

	default:
		return fmt.Errorf("不支持的连接类型: %s", connType)
	}

	return nil
}

// GetMcpPresets 获取 MCP 预设服务器列表
func GetMcpPresets() []McpServer {
	return []McpServer{
		{
			ID:          "fetch",
			Name:        "mcp-server-fetch",
			Description: "HTTP fetch tool for web requests",
			Homepage:    "https://github.com/modelcontextprotocol/servers",
			Docs:        "https://github.com/modelcontextprotocol/servers/tree/main/src/fetch",
			Tags:        []string{"stdio", "http", "web"},
			Server: map[string]interface{}{
				"type":    "stdio",
				"command": "uvx",
				"args":    []interface{}{"mcp-server-fetch"},
			},
			Apps: McpApps{Claude: false, Codex: false, Gemini: false},
		},
		{
			ID:          "time",
			Name:        "@modelcontextprotocol/server-time",
			Description: "Current time and timezone information",
			Homepage:    "https://github.com/modelcontextprotocol/servers",
			Docs:        "https://github.com/modelcontextprotocol/servers/tree/main/src/time",
			Tags:        []string{"stdio", "time", "utility"},
			Server: map[string]interface{}{
				"type":    "stdio",
				"command": "npx",
				"args":    []interface{}{"-y", "@modelcontextprotocol/server-time"},
			},
			Apps: McpApps{Claude: false, Codex: false, Gemini: false},
		},
		{
			ID:          "memory",
			Name:        "@modelcontextprotocol/server-memory",
			Description: "Knowledge graph memory system",
			Homepage:    "https://github.com/modelcontextprotocol/servers",
			Docs:        "https://github.com/modelcontextprotocol/servers/tree/main/src/memory",
			Tags:        []string{"stdio", "memory", "graph"},
			Server: map[string]interface{}{
				"type":    "stdio",
				"command": "npx",
				"args":    []interface{}{"-y", "@modelcontextprotocol/server-memory"},
			},
			Apps: McpApps{Claude: false, Codex: false, Gemini: false},
		},
		{
			ID:          "sequential-thinking",
			Name:        "@modelcontextprotocol/server-sequential-thinking",
			Description: "Extended thinking and reasoning capabilities",
			Homepage:    "https://github.com/modelcontextprotocol/servers",
			Docs:        "https://github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking",
			Tags:        []string{"stdio", "thinking", "reasoning"},
			Server: map[string]interface{}{
				"type":    "stdio",
				"command": "npx",
				"args":    []interface{}{"-y", "@modelcontextprotocol/server-sequential-thinking"},
			},
			Apps: McpApps{Claude: false, Codex: false, Gemini: false},
		},
		{
			ID:          "filesystem",
			Name:        "@modelcontextprotocol/server-filesystem",
			Description: "Secure file system access",
			Homepage:    "https://github.com/modelcontextprotocol/servers",
			Docs:        "https://github.com/modelcontextprotocol/servers/tree/main/src/filesystem",
			Tags:        []string{"stdio", "filesystem", "files"},
			Server: map[string]interface{}{
				"type":    "stdio",
				"command": "npx",
				"args":    []interface{}{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
			},
			Apps: McpApps{Claude: false, Codex: false, Gemini: false},
		},
	}
}
