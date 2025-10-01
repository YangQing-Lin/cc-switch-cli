package utils

import (
	"encoding/json"
	"fmt"
	"os"
)

// FileExists 检查文件是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// WriteJSONFile 写入 JSON 文件
func WriteJSONFile(path string, data interface{}, perm os.FileMode) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 JSON 失败: %w", err)
	}

	if err := os.WriteFile(path, jsonData, perm); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

// BackupFile 备份文件
func BackupFile(path string) error {
	if !FileExists(path) {
		return nil // 原文件不存在不算错误
	}

	backupPath := path + ".backup"
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取原文件失败: %w", err)
	}

	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("创建备份失败: %w", err)
	}

	return nil
}

// ReadJSONFile 读取 JSON 文件
func ReadJSONFile(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("解析 JSON 失败: %w", err)
	}

	return nil
}
