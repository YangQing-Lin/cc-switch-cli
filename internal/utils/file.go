package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// FileExists 检查文件是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// WriteJSONFile 原子写入 JSON 文件
func WriteJSONFile(path string, data interface{}, perm os.FileMode) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 JSON 失败: %w", err)
	}

	// 实现原子写入
	return AtomicWriteFile(path, jsonData, perm)
}

// AtomicWriteFile 原子写入文件（写入临时文件然后重命名）
func AtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	// 创建临时文件
	tmpFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpPath := tmpFile.Name()

	// 确保清理临时文件
	defer func() {
		if FileExists(tmpPath) {
			os.Remove(tmpPath)
		}
	}()

	// 写入数据到临时文件
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("写入临时文件失败: %w", err)
	}

	// 关闭文件
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("关闭临时文件失败: %w", err)
	}

	// 设置权限
	if err := os.Chmod(tmpPath, perm); err != nil {
		// Windows 可能不支持某些权限，忽略错误
		if runtime.GOOS != "windows" {
			return fmt.Errorf("设置文件权限失败: %w", err)
		}
	}

	// 保留原文件权限（如果存在）
	if FileExists(path) && runtime.GOOS != "windows" {
		if stat, err := os.Stat(path); err == nil {
			os.Chmod(tmpPath, stat.Mode())
		}
	}

	// Windows 需要先删除目标文件
	if runtime.GOOS == "windows" && FileExists(path) {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("删除原文件失败: %w", err)
		}
	}

	// 原子重命名
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("重命名文件失败: %w", err)
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

// CopyFile 复制文件
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer sourceFile.Close()

	// 获取源文件权限
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("获取源文件信息失败: %w", err)
	}

	// 创建目标文件
	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer destFile.Close()

	// 复制内容
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("复制文件内容失败: %w", err)
	}

	// 设置相同的权限
	if err := os.Chmod(dst, sourceInfo.Mode()); err != nil {
		// Windows 可能不支持某些权限，忽略错误
		if runtime.GOOS != "windows" {
			return fmt.Errorf("设置文件权限失败: %w", err)
		}
	}

	return nil
}
