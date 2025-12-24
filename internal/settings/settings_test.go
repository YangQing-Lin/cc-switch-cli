package settings

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
)

func TestNewManagerDefaultsAndPermissions(t *testing.T) {
	testutil.WithTempHome(t, func(home string) {
		manager, err := NewManager()
		if err != nil {
			t.Fatalf("NewManager() error = %v", err)
		}

		if manager.GetLanguage() != "zh" {
			t.Fatalf("默认语言不正确: %s", manager.GetLanguage())
		}
		if manager.GetConfigDir() != "" {
			t.Fatalf("默认配置目录不正确: %s", manager.GetConfigDir())
		}

		settingsPath := filepath.Join(home, ".cc-switch", "settings.json")
		if _, err := os.Stat(settingsPath); err != nil {
			t.Fatalf("设置文件未创建: %v", err)
		}

		if runtime.GOOS != "windows" {
			info, err := os.Stat(settingsPath)
			if err != nil {
				t.Fatalf("获取文件信息失败: %v", err)
			}
			if info.Mode().Perm() != 0600 {
				t.Fatalf("文件权限不匹配: %o", info.Mode().Perm())
			}
		}
	})
}

func TestLoadExistingSettings(t *testing.T) {
	testutil.WithTempHome(t, func(home string) {
		settingsDir := filepath.Join(home, ".cc-switch")
		if err := os.MkdirAll(settingsDir, 0755); err != nil {
			t.Fatalf("创建目录失败: %v", err)
		}
		content := `{"language":"en","configDir":"/custom"}`
		if err := os.WriteFile(filepath.Join(settingsDir, "settings.json"), []byte(content), 0600); err != nil {
			t.Fatalf("写入设置失败: %v", err)
		}

		manager, err := NewManager()
		if err != nil {
			t.Fatalf("NewManager() error = %v", err)
		}
		if manager.GetLanguage() != "en" || manager.GetConfigDir() != "/custom" {
			t.Fatalf("加载设置失败: %+v", manager.Get())
		}
	})
}

func TestLoadCorruptedSettings(t *testing.T) {
	testutil.WithTempHome(t, func(home string) {
		settingsDir := filepath.Join(home, ".cc-switch")
		if err := os.MkdirAll(settingsDir, 0755); err != nil {
			t.Fatalf("创建目录失败: %v", err)
		}
		if err := os.WriteFile(filepath.Join(settingsDir, "settings.json"), []byte("{invalid"), 0600); err != nil {
			t.Fatalf("写入设置失败: %v", err)
		}

		_, err := NewManager()
		if err == nil {
			t.Fatalf("期望解析失败")
		}
		if !strings.Contains(err.Error(), "解析设置文件失败") {
			t.Fatalf("错误信息不匹配: %v", err)
		}
	})
}

func TestSetLanguageAndConfigDir(t *testing.T) {
	tests := []struct {
		name        string
		lang        string
		configDir   string
		wantErr     bool
		wantLang    string
		wantDir     string
		persistLang bool
	}{
		{
			name:        "set english",
			lang:        "en",
			configDir:   "/config",
			wantLang:    "en",
			wantDir:     "/config",
			persistLang: true,
		},
		{
			name:     "invalid language",
			lang:     "fr",
			wantErr:  true,
			wantLang: "zh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.WithTempHome(t, func(_ string) {
				manager, err := NewManager()
				if err != nil {
					t.Fatalf("NewManager() error = %v", err)
				}

				err = manager.SetLanguage(tt.lang)
				if (err != nil) != tt.wantErr {
					t.Fatalf("SetLanguage() error = %v, wantErr %v", err, tt.wantErr)
				}

				if tt.configDir != "" {
					if err := manager.SetConfigDir(tt.configDir); err != nil {
						t.Fatalf("SetConfigDir() error = %v", err)
					}
				}

				if manager.GetLanguage() != tt.wantLang {
					t.Fatalf("语言不匹配: %s != %s", manager.GetLanguage(), tt.wantLang)
				}
				if tt.configDir != "" && manager.GetConfigDir() != tt.wantDir {
					t.Fatalf("配置目录不匹配: %s != %s", manager.GetConfigDir(), tt.wantDir)
				}

				if tt.persistLang {
					again, err := NewManager()
					if err != nil {
						t.Fatalf("重新加载失败: %v", err)
					}
					if again.GetLanguage() != tt.wantLang {
						t.Fatalf("持久化语言不匹配: %s", again.GetLanguage())
					}
				}
			})
		})
	}
}

func TestSavePermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("跳过 Windows 权限测试")
	}
	if os.Getuid() == 0 {
		t.Skip("skipping: running as root")
	}

	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, ".cc-switch")
	if err := os.MkdirAll(readOnlyDir, 0500); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}

	manager := &Manager{
		settings:     &AppSettings{Language: "zh"},
		settingsPath: filepath.Join(readOnlyDir, "settings.json"),
	}

	if err := manager.Save(); err == nil {
		t.Fatalf("期望保存失败")
	}
}

func TestGetSettingsPath(t *testing.T) {
	testutil.WithTempHome(t, func(home string) {
		tests := []struct {
			name string
		}{
			{name: "settings path"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				path, err := GetSettingsPath()
				if err != nil {
					t.Fatalf("GetSettingsPath() error = %v", err)
				}
				if !strings.Contains(path, filepath.Join(home, ".cc-switch")) {
					t.Fatalf("路径不包含 home: %s", path)
				}
				if filepath.Base(path) != "settings.json" {
					t.Fatalf("文件名不正确: %s", path)
				}
			})
		}
	})
}

func TestGetSettings(t *testing.T) {
	testutil.WithTempHome(t, func(_ string) {
		manager, err := NewManager()
		if err != nil {
			t.Fatalf("NewManager() error = %v", err)
		}
		if err := manager.SetLanguage("en"); err != nil {
			t.Fatalf("SetLanguage() error = %v", err)
		}
		if err := manager.SetConfigDir("/dir"); err != nil {
			t.Fatalf("SetConfigDir() error = %v", err)
		}

		settings := manager.Get()
		if settings.Language != "en" || settings.ConfigDir != "/dir" {
			t.Fatalf("获取设置失败: %+v", settings)
		}
	})
}

func TestLoadUnreadableFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("跳过 Windows 权限测试")
	}
	if os.Getuid() == 0 {
		t.Skip("跳过 root 用户权限测试")
	}

	tmpDir := t.TempDir()
	settingsDir := filepath.Join(tmpDir, ".cc-switch")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	settingsPath := filepath.Join(settingsDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"language":"en"}`), 0000); err != nil {
		t.Fatalf("写入设置失败: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(settingsPath, 0644)
	})

	manager := &Manager{
		settingsPath: settingsPath,
	}
	err := manager.Load()
	if err == nil {
		t.Fatalf("期望读取失败")
	}
	if !strings.Contains(err.Error(), "读取设置文件失败") {
		t.Fatalf("错误信息不匹配: %v", err)
	}
}

func TestSaveMkdirAllFail(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("跳过 Windows 权限测试")
	}
	if os.Getuid() == 0 {
		t.Skip("跳过 root 用户权限测试")
	}

	tmpDir := t.TempDir()
	// 创建一个文件来阻止创建同名目录
	blockPath := filepath.Join(tmpDir, "blocked")
	if err := os.WriteFile(blockPath, []byte("block"), 0444); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}

	manager := &Manager{
		settings:     &AppSettings{Language: "zh"},
		settingsPath: filepath.Join(blockPath, "subdir", "settings.json"),
	}

	err := manager.Save()
	if err == nil {
		t.Fatalf("期望保存失败")
	}
	if !strings.Contains(err.Error(), "创建设置目录失败") {
		t.Fatalf("错误信息不匹配: %v", err)
	}
}
