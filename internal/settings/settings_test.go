package settings

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/utils"
)

func TestNewManager(t *testing.T) {
	// 使用临时目录测试
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	if originalHome == "" {
		originalHome = os.Getenv("USERPROFILE") // Windows
	}

	// 临时修改 HOME 环境变量
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}

	// 验证默认设置
	if manager.GetLanguage() != "zh" {
		t.Errorf("默认语言应该是 zh，实际是 %s", manager.GetLanguage())
	}

	// 验证设置文件被创建
	settingsPath := filepath.Join(tmpDir, ".cc-switch", "settings.json")
	if !utils.FileExists(settingsPath) {
		t.Error("设置文件未被创建")
	}
}

func TestSetLanguage(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	tests := []struct {
		name    string
		lang    string
		wantErr bool
	}{
		{
			name:    "设置英文",
			lang:    "en",
			wantErr: false,
		},
		{
			name:    "设置中文",
			lang:    "zh",
			wantErr: false,
		},
		{
			name:    "设置无效语言",
			lang:    "fr",
			wantErr: true,
		},
		{
			name:    "空字符串",
			lang:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.SetLanguage(tt.lang)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetLanguage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				got := manager.GetLanguage()
				if got != tt.lang {
					t.Errorf("GetLanguage() = %v, want %v", got, tt.lang)
				}

				// 重新加载验证持久化
				newManager, err := NewManager()
				if err != nil {
					t.Fatalf("重新加载失败: %v", err)
				}
				if newManager.GetLanguage() != tt.lang {
					t.Errorf("持久化后语言不匹配，got = %v, want %v", newManager.GetLanguage(), tt.lang)
				}
			}
		})
	}
}

func TestSetConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	tests := []struct {
		name      string
		configDir string
	}{
		{
			name:      "设置自定义目录",
			configDir: "/custom/config/dir",
		},
		{
			name:      "设置空字符串",
			configDir: "",
		},
		{
			name:      "设置Windows路径",
			configDir: "C:\\custom\\config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.SetConfigDir(tt.configDir)
			if err != nil {
				t.Errorf("SetConfigDir() error = %v", err)
				return
			}

			got := manager.GetConfigDir()
			if got != tt.configDir {
				t.Errorf("GetConfigDir() = %v, want %v", got, tt.configDir)
			}

			// 重新加载验证持久化
			newManager, err := NewManager()
			if err != nil {
				t.Fatalf("重新加载失败: %v", err)
			}
			if newManager.GetConfigDir() != tt.configDir {
				t.Errorf("持久化后配置目录不匹配，got = %v, want %v", newManager.GetConfigDir(), tt.configDir)
			}
		})
	}
}

func TestGet(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// 设置一些值
	if err := manager.SetLanguage("en"); err != nil {
		t.Fatalf("SetLanguage() error = %v", err)
	}
	if err := manager.SetConfigDir("/test/dir"); err != nil {
		t.Fatalf("SetConfigDir() error = %v", err)
	}

	// 获取所有设置
	settings := manager.Get()
	if settings == nil {
		t.Fatal("Get() returned nil")
	}

	if settings.Language != "en" {
		t.Errorf("settings.Language = %v, want 'en'", settings.Language)
	}
	if settings.ConfigDir != "/test/dir" {
		t.Errorf("settings.ConfigDir = %v, want '/test/dir'", settings.ConfigDir)
	}
}

func TestLoadExistingSettings(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	settingsDir := filepath.Join(tmpDir, ".cc-switch")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		t.Fatalf("创建设置目录失败: %v", err)
	}

	// 创建已存在的设置文件
	settingsPath := filepath.Join(settingsDir, "settings.json")
	content := `{
  "language": "en",
  "configDir": "/existing/dir"
}`
	if err := os.WriteFile(settingsPath, []byte(content), 0600); err != nil {
		t.Fatalf("创建设置文件失败: %v", err)
	}

	// 加载设置
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// 验证加载的设置
	if manager.GetLanguage() != "en" {
		t.Errorf("GetLanguage() = %v, want 'en'", manager.GetLanguage())
	}
	if manager.GetConfigDir() != "/existing/dir" {
		t.Errorf("GetConfigDir() = %v, want '/existing/dir'", manager.GetConfigDir())
	}
}

func TestSaveFilePermissions(t *testing.T) {
	// Windows 上跳过权限测试
	if runtime.GOOS == "windows" {
		t.Skip("跳过 Windows 上的权限测试")
	}

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	_, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	settingsPath, err := GetSettingsPath()
	if err != nil {
		t.Fatalf("GetSettingsPath() error = %v", err)
	}

	// 验证文件权限为 0600
	info, err := os.Stat(settingsPath)
	if err != nil {
		t.Fatalf("获取文件信息失败: %v", err)
	}

	perm := info.Mode().Perm()
	expectedPerm := os.FileMode(0600)
	if perm != expectedPerm {
		t.Errorf("文件权限 = %o, want %o", perm, expectedPerm)
	}
}

func TestGetSettingsPath(t *testing.T) {
	path, err := GetSettingsPath()
	if err != nil {
		t.Errorf("GetSettingsPath() error = %v", err)
	}

	if path == "" {
		t.Error("GetSettingsPath() returned empty path")
	}

	// 验证路径包含 .cc-switch/settings.json
	if !filepath.IsAbs(path) {
		t.Errorf("GetSettingsPath() should return absolute path, got %v", path)
	}

	base := filepath.Base(path)
	if base != "settings.json" {
		t.Errorf("GetSettingsPath() should end with settings.json, got %v", base)
	}

	dir := filepath.Base(filepath.Dir(path))
	if dir != ".cc-switch" {
		t.Errorf("GetSettingsPath() should be in .cc-switch directory, got %v", dir)
	}
}
