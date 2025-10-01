package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// CreateTempDir 创建临时测试目录
func CreateTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "cc-switch-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	return dir
}

// CreateTempFile 创建临时测试文件
func CreateTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	return path
}

// AssertFileExists 断言文件存在
func AssertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("文件不存在: %s", path)
	}
}

// AssertFileNotExists 断言文件不存在
func AssertFileNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Errorf("文件不应该存在: %s", path)
	}
}

// AssertFileContent 断言文件内容
func AssertFileContent(t *testing.T, path, expected string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}
	if string(content) != expected {
		t.Errorf("文件内容不匹配\n期望: %s\n实际: %s", expected, string(content))
	}
}

// AssertFileMode 断言文件权限（仅在非Windows系统）
func AssertFileMode(t *testing.T, path string, expected os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("获取文件信息失败: %v", err)
	}
	actual := info.Mode().Perm()
	if actual != expected {
		t.Errorf("文件权限不匹配\n期望: %o\n实际: %o", expected, actual)
	}
}
