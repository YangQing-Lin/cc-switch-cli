package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/YangQing-Lin/cc-switch-cli/internal/utils"
)

const (
	claudeDir        = ".claude"
	claudeConfigFile = "config.json"
	managedAPIKey    = "any" // 固定值，用于标记由 cc-switch 管理
)

// ClaudeConfig 表示 Claude 插件配置文件结构
type ClaudeConfig map[string]interface{}

var claudeUserHomeDirFunc = os.UserHomeDir
var claudeAtomicWriteFileFunc = utils.AtomicWriteFile
var claudeRemoveFileFunc = os.Remove

// GetClaudeConfigPath 获取 Claude 插件配置文件路径
func GetClaudeConfigPath() (string, error) {
	home, err := claudeUserHomeDirFunc()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}

	return filepath.Join(home, claudeDir, claudeConfigFile), nil
}

// EnsureClaudeDirExists 确保 Claude 配置目录存在
func EnsureClaudeDirExists() error {
	home, err := claudeUserHomeDirFunc()
	if err != nil {
		return fmt.Errorf("获取用户主目录失败: %w", err)
	}

	dir := filepath.Join(home, claudeDir)
	if !utils.FileExists(dir) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建 Claude 配置目录失败: %w", err)
		}
	}

	return nil
}

// ReadClaudeConfig 读取 Claude 插件配置
func ReadClaudeConfig() (ClaudeConfig, error) {
	path, err := GetClaudeConfigPath()
	if err != nil {
		return nil, err
	}

	if !utils.FileExists(path) {
		return make(ClaudeConfig), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取 Claude 配置失败: %w", err)
	}

	var config ClaudeConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析 Claude 配置失败: %w", err)
	}

	return config, nil
}

// WriteClaudeConfig 写入 Claude 插件配置
func WriteClaudeConfig(config ClaudeConfig) error {
	if err := EnsureClaudeDirExists(); err != nil {
		return err
	}

	path, err := GetClaudeConfigPath()
	if err != nil {
		return err
	}

	// 格式化 JSON 输出
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 Claude 配置失败: %w", err)
	}

	// 添加换行符
	data = append(data, '\n')

	// 写入文件（权限 0600，保护敏感信息）
	if err := claudeAtomicWriteFileFunc(path, data, 0600); err != nil {
		return fmt.Errorf("写入 Claude 配置失败: %w", err)
	}

	return nil
}

// ApplyClaudePlugin 应用 Claude 插件配置
// 写入固定的 primaryApiKey 字段
func ApplyClaudePlugin() (bool, error) {
	config, err := ReadClaudeConfig()
	if err != nil {
		return false, err
	}

	// 检查是否已经应用
	if val, ok := config["primaryApiKey"]; ok {
		if strVal, ok := val.(string); ok && strVal == managedAPIKey {
			return false, nil // 已经应用，无需重复
		}
	}

	// 设置固定值
	config["primaryApiKey"] = managedAPIKey

	if err := WriteClaudeConfig(config); err != nil {
		return false, err
	}

	return true, nil
}

// RemoveClaudePlugin 移除 Claude 插件配置
// 只删除 primaryApiKey 字段，保留其他字段
func RemoveClaudePlugin() (bool, error) {
	path, err := GetClaudeConfigPath()
	if err != nil {
		return false, err
	}

	if !utils.FileExists(path) {
		return false, nil // 文件不存在，无需移除
	}

	config, err := ReadClaudeConfig()
	if err != nil {
		return false, err
	}

	// 检查是否存在 primaryApiKey
	if _, exists := config["primaryApiKey"]; !exists {
		return false, nil // 不存在，无需移除
	}

	// 删除 primaryApiKey 字段
	delete(config, "primaryApiKey")

	// 如果配置为空，删除文件
	if len(config) == 0 {
		if err := claudeRemoveFileFunc(path); err != nil && !os.IsNotExist(err) {
			return false, fmt.Errorf("删除空配置文件失败: %w", err)
		}
		return true, nil
	}

	// 否则写回修改后的配置
	if err := WriteClaudeConfig(config); err != nil {
		return false, err
	}

	return true, nil
}

// IsClaudePluginApplied 检查 Claude 插件配置是否已应用
func IsClaudePluginApplied() (bool, error) {
	config, err := ReadClaudeConfig()
	if err != nil {
		return false, err
	}

	if val, ok := config["primaryApiKey"]; ok {
		if strVal, ok := val.(string); ok && strVal == managedAPIKey {
			return true, nil
		}
	}

	return false, nil
}

// ClaudeConfigStatus 获取 Claude 插件配置状态
func ClaudeConfigStatus() (exists bool, path string, err error) {
	path, err = GetClaudeConfigPath()
	if err != nil {
		return false, "", err
	}

	exists = utils.FileExists(path)
	return exists, path, nil
}
