package tui

import (
	"fmt"
	"os"
	"path/filepath"
)

var tuiExecutableFunc = os.Executable
var tuiWriteFileFunc = os.WriteFile
var tuiRemoveFileFunc = os.Remove

// enablePortableMode 启用便携模式
func (m *Model) enablePortableMode() error {
	// 获取可执行文件目录
	execPath, err := tuiExecutableFunc()
	if err != nil {
		return fmt.Errorf("获取程序路径失败: %w", err)
	}

	execDir := filepath.Dir(execPath)
	portableFile := filepath.Join(execDir, "portable.ini")

	// 创建 portable.ini 文件
	content := []byte("# CC-Switch Portable Mode\n# This file enables portable mode.\n# Delete this file to disable portable mode.\n")
	if err := tuiWriteFileFunc(portableFile, content, 0644); err != nil {
		return fmt.Errorf("创建 portable.ini 失败: %w", err)
	}

	return nil
}

// disablePortableMode 禁用便携模式
func (m *Model) disablePortableMode() error {
	// 获取可执行文件目录
	execPath, err := tuiExecutableFunc()
	if err != nil {
		return fmt.Errorf("获取程序路径失败: %w", err)
	}

	execDir := filepath.Dir(execPath)
	portableFile := filepath.Join(execDir, "portable.ini")

	// 删除 portable.ini 文件
	if err := tuiRemoveFileFunc(portableFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除 portable.ini 失败: %w", err)
	}

	return nil
}
