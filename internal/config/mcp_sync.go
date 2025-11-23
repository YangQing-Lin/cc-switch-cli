package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/YangQing-Lin/cc-switch-cli/internal/utils"
	"github.com/pelletier/go-toml/v2"
)

// SyncMcpToClaud 同步单个 MCP 服务器到 Claude
func (m *Manager) SyncMcpToClaud(serverID string) error {
	m.ensureMcpRoot()

	// 获取服务器配置
	server, exists := m.config.Mcp.Servers[serverID]
	if !exists {
		return fmt.Errorf("MCP 服务器不存在: %s", serverID)
	}

	// 如果未启用 Claude，移除配置
	if !server.Apps.Claude {
		return m.RemoveMcpFromClaude(serverID)
	}

	// 获取 Claude 配置文件路径
	settingsPath, err := m.GetClaudeSettingsPathWithDir()
	if err != nil {
		return fmt.Errorf("获取 Claude 配置路径失败: %w", err)
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		return fmt.Errorf("创建 Claude 配置目录失败: %w", err)
	}

	// 读取现有配置
	var settings ClaudeSettings
	if utils.FileExists(settingsPath) {
		data, err := os.ReadFile(settingsPath)
		if err != nil {
			return fmt.Errorf("读取 Claude 配置失败: %w", err)
		}
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("解析 Claude 配置失败: %w", err)
		}
	} else {
		// 初始化默认配置
		settings.Permissions.Allow = []string{}
		settings.Permissions.Deny = []string{}
	}

	// 初始化 Extra 字段（用于存储 mcpServers）
	if settings.Extra == nil {
		settings.Extra = make(map[string]interface{})
	}

	// 获取或创建 mcpServers
	var mcpServers map[string]interface{}
	if existingMcp, ok := settings.Extra["mcpServers"].(map[string]interface{}); ok {
		mcpServers = existingMcp
	} else {
		mcpServers = make(map[string]interface{})
	}

	// 添加或更新服务器配置
	mcpServers[server.ID] = server.Server
	settings.Extra["mcpServers"] = mcpServers

	// 写入配置
	if err := utils.WriteJSONFile(settingsPath, &settings, 0600); err != nil {
		return fmt.Errorf("写入 Claude 配置失败: %w", err)
	}

	return nil
}

// RemoveMcpFromClaude 从 Claude 移除 MCP 服务器
func (m *Manager) RemoveMcpFromClaude(serverID string) error {
	settingsPath, err := m.GetClaudeSettingsPathWithDir()
	if err != nil {
		return fmt.Errorf("获取 Claude 配置路径失败: %w", err)
	}

	if !utils.FileExists(settingsPath) {
		return nil // 配置文件不存在，无需移除
	}

	// 读取现有配置
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return fmt.Errorf("读取 Claude 配置失败: %w", err)
	}

	var settings ClaudeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("解析 Claude 配置失败: %w", err)
	}

	// 移除 MCP 服务器
	if settings.Extra != nil {
		if mcpServers, ok := settings.Extra["mcpServers"].(map[string]interface{}); ok {
			delete(mcpServers, serverID)
			settings.Extra["mcpServers"] = mcpServers
		}
	}

	// 写入配置
	if err := utils.WriteJSONFile(settingsPath, &settings, 0600); err != nil {
		return fmt.Errorf("写入 Claude 配置失败: %w", err)
	}

	return nil
}

// SyncMcpToCodex 同步单个 MCP 服务器到 Codex
func (m *Manager) SyncMcpToCodex(serverID string) error {
	m.ensureMcpRoot()

	// 获取服务器配置
	server, exists := m.config.Mcp.Servers[serverID]
	if !exists {
		return fmt.Errorf("MCP 服务器不存在: %s", serverID)
	}

	// 如果未启用 Codex，移除配置
	if !server.Apps.Codex {
		return m.RemoveMcpFromCodex(serverID)
	}

	// 获取 Codex 配置文件路径
	configPath, err := m.GetCodexConfigPathWithDir()
	if err != nil {
		return fmt.Errorf("获取 Codex 配置路径失败: %w", err)
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("创建 Codex 配置目录失败: %w", err)
	}

	// 读取现有配置
	var config map[string]interface{}
	if utils.FileExists(configPath) {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("读取 Codex 配置失败: %w", err)
		}
		if err := toml.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("解析 Codex 配置失败: %w", err)
		}
	} else {
		config = make(map[string]interface{})
	}

	// 获取或创建 mcp_servers
	var mcpServers map[string]interface{}
	if existingMcp, ok := config["mcp_servers"].(map[string]interface{}); ok {
		mcpServers = existingMcp
	} else {
		mcpServers = make(map[string]interface{})
	}

	// 添加或更新服务器配置
	mcpServers[server.ID] = server.Server
	config["mcp_servers"] = mcpServers

	// 写入配置
	data, err := toml.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化 Codex 配置失败: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("写入 Codex 配置失败: %w", err)
	}

	return nil
}

// RemoveMcpFromCodex 从 Codex 移除 MCP 服务器
func (m *Manager) RemoveMcpFromCodex(serverID string) error {
	configPath, err := m.GetCodexConfigPathWithDir()
	if err != nil {
		return fmt.Errorf("获取 Codex 配置路径失败: %w", err)
	}

	if !utils.FileExists(configPath) {
		return nil // 配置文件不存在，无需移除
	}

	// 读取现有配置
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取 Codex 配置失败: %w", err)
	}

	var config map[string]interface{}
	if err := toml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("解析 Codex 配置失败: %w", err)
	}

	// 移除 MCP 服务器
	if mcpServers, ok := config["mcp_servers"].(map[string]interface{}); ok {
		delete(mcpServers, serverID)
		config["mcp_servers"] = mcpServers
	}

	// 写入配置
	data, err = toml.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化 Codex 配置失败: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("写入 Codex 配置失败: %w", err)
	}

	return nil
}

// SyncMcpToGemini 同步单个 MCP 服务器到 Gemini
func (m *Manager) SyncMcpToGemini(serverID string) error {
	m.ensureMcpRoot()

	// 获取服务器配置
	server, exists := m.config.Mcp.Servers[serverID]
	if !exists {
		return fmt.Errorf("MCP 服务器不存在: %s", serverID)
	}

	// 如果未启用 Gemini，移除配置
	if !server.Apps.Gemini {
		return m.RemoveMcpFromGemini(serverID)
	}

	// 获取 Gemini 配置文件路径
	settingsPath, err := m.GetGeminiSettingsPathWithDir()
	if err != nil {
		return fmt.Errorf("获取 Gemini 配置路径失败: %w", err)
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		return fmt.Errorf("创建 Gemini 配置目录失败: %w", err)
	}

	// 读取现有配置
	var settings GeminiSettings
	if utils.FileExists(settingsPath) {
		data, err := os.ReadFile(settingsPath)
		if err != nil {
			return fmt.Errorf("读取 Gemini 配置失败: %w", err)
		}
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("解析 Gemini 配置失败: %w", err)
		}
	} else {
		// 初始化默认配置
		settings.Security.Auth.SelectedType = GeminiAuthAPIKey
	}

	// 初始化 MCPServers
	if settings.MCPServers == nil {
		settings.MCPServers = make(map[string]interface{})
	}

	// 添加或更新服务器配置
	settings.MCPServers[server.ID] = server.Server

	// 写入配置
	if err := utils.WriteJSONFile(settingsPath, &settings, 0600); err != nil {
		return fmt.Errorf("写入 Gemini 配置失败: %w", err)
	}

	return nil
}

// RemoveMcpFromGemini 从 Gemini 移除 MCP 服务器
func (m *Manager) RemoveMcpFromGemini(serverID string) error {
	settingsPath, err := m.GetGeminiSettingsPathWithDir()
	if err != nil {
		return fmt.Errorf("获取 Gemini 配置路径失败: %w", err)
	}

	if !utils.FileExists(settingsPath) {
		return nil // 配置文件不存在，无需移除
	}

	// 读取现有配置
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return fmt.Errorf("读取 Gemini 配置失败: %w", err)
	}

	var settings GeminiSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("解析 Gemini 配置失败: %w", err)
	}

	// 移除 MCP 服务器
	if settings.MCPServers != nil {
		delete(settings.MCPServers, serverID)
	}

	// 写入配置
	if err := utils.WriteJSONFile(settingsPath, &settings, 0600); err != nil {
		return fmt.Errorf("写入 Gemini 配置失败: %w", err)
	}

	return nil
}

// SyncMcpServer 同步单个 MCP 服务器到所有启用的应用
func (m *Manager) SyncMcpServer(serverID string) error {
	m.ensureMcpRoot()

	server, exists := m.config.Mcp.Servers[serverID]
	if !exists {
		return fmt.Errorf("MCP 服务器不存在: %s", serverID)
	}

	var errs []error

	// 同步到 Claude
	if server.Apps.Claude {
		if err := m.SyncMcpToClaud(serverID); err != nil {
			errs = append(errs, fmt.Errorf("同步到 Claude 失败: %w", err))
		}
	} else {
		if err := m.RemoveMcpFromClaude(serverID); err != nil {
			errs = append(errs, fmt.Errorf("从 Claude 移除失败: %w", err))
		}
	}

	// 同步到 Codex
	if server.Apps.Codex {
		if err := m.SyncMcpToCodex(serverID); err != nil {
			errs = append(errs, fmt.Errorf("同步到 Codex 失败: %w", err))
		}
	} else {
		if err := m.RemoveMcpFromCodex(serverID); err != nil {
			errs = append(errs, fmt.Errorf("从 Codex 移除失败: %w", err))
		}
	}

	// 同步到 Gemini
	if server.Apps.Gemini {
		if err := m.SyncMcpToGemini(serverID); err != nil {
			errs = append(errs, fmt.Errorf("同步到 Gemini 失败: %w", err))
		}
	} else {
		if err := m.RemoveMcpFromGemini(serverID); err != nil {
			errs = append(errs, fmt.Errorf("从 Gemini 移除失败: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("同步失败: %w", errors.Join(errs...))
	}

	return nil
}

// SyncAllMcpServers 同步所有 MCP 服务器到对应的应用
func (m *Manager) SyncAllMcpServers() error {
	m.ensureMcpRoot()

	var errs []error
	for serverID := range m.config.Mcp.Servers {
		if err := m.SyncMcpServer(serverID); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("部分同步失败: %w", errors.Join(errs...))
	}

	return nil
}

// SyncAllMcpServersBatch 批量同步所有 MCP 服务器（性能优化版）
// 相比 SyncAllMcpServers，此方法只对每个应用写入一次，而不是每个服务器写入一次
func (m *Manager) SyncAllMcpServersBatch() error {
	m.ensureMcpRoot()

	// 1. 构建每个应用的 MCP 服务器映射
	claudeServers := make(map[string]interface{})
	codexServers := make(map[string]interface{})
	geminiServers := make(map[string]interface{})

	for id, server := range m.config.Mcp.Servers {
		if server.Apps.Claude {
			claudeServers[id] = server.Server
		}
		if server.Apps.Codex {
			codexServers[id] = server.Server
		}
		if server.Apps.Gemini {
			geminiServers[id] = server.Server
		}
	}

	// 2. 一次性写入每个应用的配置
	var errs []error

	// 同步到 Claude
	if err := m.syncMcpToClaudeBatch(claudeServers); err != nil {
		errs = append(errs, fmt.Errorf("同步到 Claude 失败: %w", err))
	}

	// 同步到 Codex
	if err := m.syncMcpToCodexBatch(codexServers); err != nil {
		errs = append(errs, fmt.Errorf("同步到 Codex 失败: %w", err))
	}

	// 同步到 Gemini
	if err := m.syncMcpToGeminiBatch(geminiServers); err != nil {
		errs = append(errs, fmt.Errorf("同步到 Gemini 失败: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("部分同步失败: %w", errors.Join(errs...))
	}

	return nil
}

// syncMcpToClaudeBatch 批量同步到 Claude
func (m *Manager) syncMcpToClaudeBatch(servers map[string]interface{}) error {
	settingsPath, err := m.GetClaudeSettingsPathWithDir()
	if err != nil {
		return fmt.Errorf("获取 Claude 配置路径失败: %w", err)
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		return fmt.Errorf("创建 Claude 配置目录失败: %w", err)
	}

	// 读取现有配置
	var settings ClaudeSettings
	if utils.FileExists(settingsPath) {
		data, err := os.ReadFile(settingsPath)
		if err != nil {
			return fmt.Errorf("读取 Claude 配置失败: %w", err)
		}
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("解析 Claude 配置失败: %w", err)
		}
	} else {
		settings.Permissions.Allow = []string{}
		settings.Permissions.Deny = []string{}
	}

	// 初始化 Extra
	if settings.Extra == nil {
		settings.Extra = make(map[string]interface{})
	}

	// 一次性设置所有 MCP 服务器
	settings.Extra["mcpServers"] = servers

	// 只写入一次
	if err := utils.WriteJSONFile(settingsPath, &settings, 0600); err != nil {
		return fmt.Errorf("写入 Claude 配置失败: %w", err)
	}

	return nil
}

// syncMcpToCodexBatch 批量同步到 Codex
func (m *Manager) syncMcpToCodexBatch(servers map[string]interface{}) error {
	configPath, err := m.GetCodexConfigPathWithDir()
	if err != nil {
		return fmt.Errorf("获取 Codex 配置路径失败: %w", err)
	}

	codexDir := filepath.Dir(configPath)

	// 确保目录存在
	if err := os.MkdirAll(codexDir, 0755); err != nil {
		return fmt.Errorf("创建 Codex 配置目录失败: %w", err)
	}

	mcpPath := filepath.Join(codexDir, "mcp.json")

	// 一次性写入所有服务器
	mcpConfig := map[string]interface{}{
		"mcpServers": servers,
	}

	if err := utils.WriteJSONFile(mcpPath, mcpConfig, 0600); err != nil {
		return fmt.Errorf("写入 Codex MCP 配置失败: %w", err)
	}

	return nil
}

// syncMcpToGeminiBatch 批量同步到 Gemini
func (m *Manager) syncMcpToGeminiBatch(servers map[string]interface{}) error {
	settingsPath, err := m.GetGeminiSettingsPathWithDir()
	if err != nil {
		return fmt.Errorf("获取 Gemini 配置路径失败: %w", err)
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		return fmt.Errorf("创建 Gemini 配置目录失败: %w", err)
	}

	// 读取现有配置
	var geminiSettings map[string]interface{}
	if utils.FileExists(settingsPath) {
		data, err := os.ReadFile(settingsPath)
		if err != nil {
			return fmt.Errorf("读取 Gemini 配置失败: %w", err)
		}
		if err := json.Unmarshal(data, &geminiSettings); err != nil {
			return fmt.Errorf("解析 Gemini 配置失败: %w", err)
		}
	} else {
		geminiSettings = make(map[string]interface{})
	}

	// 一次性设置所有 MCP 服务器
	geminiSettings["mcpServers"] = servers

	// 只写入一次
	if err := utils.WriteJSONFile(settingsPath, geminiSettings, 0600); err != nil {
		return fmt.Errorf("写入 Gemini 配置失败: %w", err)
	}

	return nil
}
