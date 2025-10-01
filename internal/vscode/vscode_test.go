package vscode

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetVsCodeConfigPath(t *testing.T) {
	path := getVsCodeConfigPath()

	if path == "" {
		t.Error("getVsCodeConfigPath() returned empty path")
	}

	// 验证路径包含预期的目录结构
	switch runtime.GOOS {
	case "windows":
		if !filepath.IsAbs(path) {
			t.Errorf("路径应该是绝对路径: %s", path)
		}
		// Windows 路径应包含 AppData\Roaming\Code\User
		if !containsPathSegments(path, "AppData", "Roaming", "Code", "User") {
			t.Errorf("Windows 路径格式不正确: %s", path)
		}
	case "darwin":
		if !filepath.IsAbs(path) {
			t.Errorf("路径应该是绝对路径: %s", path)
		}
		// macOS 路径应包含 Library/Application Support/Code/User
		if !containsPathSegments(path, "Library", "Application Support", "Code", "User") {
			t.Errorf("macOS 路径格式不正确: %s", path)
		}
	default: // Linux
		if !filepath.IsAbs(path) {
			t.Errorf("路径应该是绝对路径: %s", path)
		}
		// Linux 路径应包含 .config/Code/User
		if !containsPathSegments(path, ".config", "Code", "User") {
			t.Errorf("Linux 路径格式不正确: %s", path)
		}
	}
}

func TestGetCursorConfigPath(t *testing.T) {
	path := getCursorConfigPath()

	if path == "" {
		t.Error("getCursorConfigPath() returned empty path")
	}

	// 验证路径包含预期的目录结构
	switch runtime.GOOS {
	case "windows":
		if !filepath.IsAbs(path) {
			t.Errorf("路径应该是绝对路径: %s", path)
		}
		if !containsPathSegments(path, "AppData", "Roaming", "Cursor", "User") {
			t.Errorf("Windows 路径格式不正确: %s", path)
		}
	case "darwin":
		if !filepath.IsAbs(path) {
			t.Errorf("路径应该是绝对路径: %s", path)
		}
		if !containsPathSegments(path, "Library", "Application Support", "Cursor", "User") {
			t.Errorf("macOS 路径格式不正确: %s", path)
		}
	default: // Linux
		if !filepath.IsAbs(path) {
			t.Errorf("路径应该是绝对路径: %s", path)
		}
		if !containsPathSegments(path, ".config", "Cursor", "User") {
			t.Errorf("Linux 路径格式不正确: %s", path)
		}
	}
}

func TestSupportedApps(t *testing.T) {
	if len(SupportedApps) == 0 {
		t.Error("SupportedApps should not be empty")
	}

	// 验证每个应用都有必要的字段
	for _, app := range SupportedApps {
		if app.Name == "" {
			t.Error("App name should not be empty")
		}
		if app.ProcessName == "" {
			t.Error("App process name should not be empty")
		}
		if app.ConfigPath == "" {
			t.Errorf("App %s config path should not be empty", app.Name)
		}
		if !filepath.IsAbs(app.ConfigPath) {
			t.Errorf("App %s config path should be absolute: %s", app.Name, app.ConfigPath)
		}
	}

	// 验证支持 VS Code
	found := false
	for _, app := range SupportedApps {
		if app.Name == "VS Code" {
			found = true
			if app.ProcessName != "code" {
				t.Errorf("VS Code process name should be 'code', got %s", app.ProcessName)
			}
			break
		}
	}
	if !found {
		t.Error("SupportedApps should include VS Code")
	}

	// 验证支持 Cursor
	found = false
	for _, app := range SupportedApps {
		if app.Name == "Cursor" {
			found = true
			if app.ProcessName != "cursor" {
				t.Errorf("Cursor process name should be 'cursor', got %s", app.ProcessName)
			}
			break
		}
	}
	if !found {
		t.Error("SupportedApps should include Cursor")
	}
}

func TestIsRunning(t *testing.T) {
	// 注意：此测试依赖于系统进程，可能不稳定
	// 我们只测试它不会崩溃，不测试具体结果
	for _, app := range SupportedApps {
		t.Run(app.Name, func(t *testing.T) {
			// 仅验证函数不会崩溃或panic
			_, _ = IsRunning(app)
			// 无论返回什么结果都可以，只要不出错
		})
	}
}

// 辅助函数：检查路径是否包含所有指定的段
func containsPathSegments(path string, segments ...string) bool {
	// 规范化路径分隔符
	normalizedPath := filepath.ToSlash(path)

	for _, segment := range segments {
		normalizedSegment := filepath.ToSlash(segment)
		if !contains(normalizedPath, normalizedSegment) {
			return false
		}
	}
	return true
}

// 辅助函数：字符串包含检查
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(findSubstring(s, substr) >= 0))
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
