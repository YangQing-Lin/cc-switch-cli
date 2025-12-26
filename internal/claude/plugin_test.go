package claude

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
)

func TestClaudeConfigPaths(t *testing.T) {
	testutil.WithTempHome(t, func(home string) {
		tests := []struct {
			name string
		}{
			{name: "config path"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				path, err := GetClaudeConfigPath()
				if err != nil {
					t.Fatalf("GetClaudeConfigPath() error = %v", err)
				}
				expected := filepath.Join(home, claudeDir, claudeConfigFile)
				if path != expected {
					t.Fatalf("路径不匹配: %s != %s", path, expected)
				}
			})
		}
	})
}

func TestEnsureClaudeDirExists(t *testing.T) {
	testutil.WithTempHome(t, func(home string) {
		if err := EnsureClaudeDirExists(); err != nil {
			t.Fatalf("EnsureClaudeDirExists() error = %v", err)
		}

		path := filepath.Join(home, claudeDir)
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			t.Fatalf("目录不存在: %v", err)
		}
	})
}

func TestReadClaudeConfig(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantErr    bool
		wantEmpty  bool
		shouldFile bool
	}{
		{name: "missing file", wantEmpty: true},
		{name: "valid config", body: `{"foo":"bar"}`, shouldFile: true},
		{name: "corrupted config", body: "{invalid", shouldFile: true, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.WithTempHome(t, func(home string) {
				if tt.shouldFile {
					dir := filepath.Join(home, claudeDir)
					if err := os.MkdirAll(dir, 0755); err != nil {
						t.Fatalf("创建目录失败: %v", err)
					}
					if err := os.WriteFile(filepath.Join(dir, claudeConfigFile), []byte(tt.body), 0600); err != nil {
						t.Fatalf("写入配置失败: %v", err)
					}
				}

				config, err := ReadClaudeConfig()
				if (err != nil) != tt.wantErr {
					t.Fatalf("ReadClaudeConfig() error = %v, wantErr %v", err, tt.wantErr)
				}
				if tt.wantErr {
					return
				}
				if tt.wantEmpty && len(config) != 0 {
					t.Fatalf("期望空配置")
				}
				if !tt.wantEmpty && config["foo"] != "bar" {
					t.Fatalf("配置内容不匹配: %+v", config)
				}
			})
		})
	}
}

func TestWriteClaudeConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    ClaudeConfig
		wantErr   bool
		skipOnWin bool
	}{
		{
			name:   "write config",
			config: ClaudeConfig{"foo": "bar"},
		},
		{
			name:      "permission denied",
			config:    ClaudeConfig{"foo": "bar"},
			wantErr:   true,
			skipOnWin: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.WithTempHome(t, func(home string) {
				if tt.skipOnWin && runtime.GOOS == "windows" {
					t.Skip("跳过 Windows 权限测试")
				}

				if tt.wantErr {
					if err := os.Chmod(home, 0500); err != nil {
						t.Fatalf("设置权限失败: %v", err)
					}
					t.Cleanup(func() {
						_ = os.Chmod(home, 0755)
					})
				}

				err := WriteClaudeConfig(tt.config)
				if (err != nil) != tt.wantErr {
					t.Fatalf("WriteClaudeConfig() error = %v, wantErr %v", err, tt.wantErr)
				}
				if tt.wantErr {
					return
				}

				path := filepath.Join(home, claudeDir, claudeConfigFile)
				content, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("读取配置失败: %v", err)
				}
				if !strings.HasSuffix(string(content), "\n") {
					t.Fatalf("配置文件应以换行结尾")
				}

				var decoded map[string]interface{}
				if err := json.Unmarshal(content, &decoded); err != nil {
					t.Fatalf("解析配置失败: %v", err)
				}
				if decoded["foo"] != "bar" {
					t.Fatalf("配置内容不匹配: %+v", decoded)
				}

				if runtime.GOOS != "windows" {
					info, err := os.Stat(path)
					if err != nil {
						t.Fatalf("读取文件信息失败: %v", err)
					}
					if info.Mode().Perm() != 0600 {
						t.Fatalf("权限不匹配: %o", info.Mode().Perm())
					}
				}
			})
		})
	}
}

func TestApplyAndRemoveClaudePlugin(t *testing.T) {
	tests := []struct {
		name       string
		initial    ClaudeConfig
		apply      bool
		wantChange bool
		remove     bool
		wantExists bool
	}{
		{
			name:       "apply fresh config",
			initial:    ClaudeConfig{},
			apply:      true,
			wantChange: true,
			remove:     true,
			wantExists: false,
		},
		{
			name:       "apply existing config",
			initial:    ClaudeConfig{"foo": "bar"},
			apply:      true,
			wantChange: true,
			remove:     true,
			wantExists: true,
		},
		{
			name:       "already applied",
			initial:    ClaudeConfig{"primaryApiKey": managedAPIKey},
			apply:      true,
			wantChange: false,
			remove:     false,
			wantExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.WithTempHome(t, func(home string) {
				if len(tt.initial) > 0 {
					if err := WriteClaudeConfig(tt.initial); err != nil {
						t.Fatalf("写入初始配置失败: %v", err)
					}
				}

				if tt.apply {
					changed, err := ApplyClaudePlugin()
					if err != nil {
						t.Fatalf("ApplyClaudePlugin() error = %v", err)
					}
					if changed != tt.wantChange {
						t.Fatalf("ApplyClaudePlugin() changed = %v, want %v", changed, tt.wantChange)
					}
				}

				if tt.remove {
					removed, err := RemoveClaudePlugin()
					if err != nil {
						t.Fatalf("RemoveClaudePlugin() error = %v", err)
					}
					if !removed {
						t.Fatalf("期望移除成功")
					}
				}

				path := filepath.Join(home, claudeDir, claudeConfigFile)
				_, err := os.Stat(path)
				exists := err == nil
				if exists != tt.wantExists {
					t.Fatalf("配置文件存在性不匹配: %v != %v", exists, tt.wantExists)
				}
			})
		})
	}
}

func TestRemoveClaudePluginScenarios(t *testing.T) {
	tests := []struct {
		name       string
		initial    ClaudeConfig
		wantChange bool
		wantExists bool
	}{
		{
			name:       "missing config",
			initial:    nil,
			wantChange: false,
			wantExists: false,
		},
		{
			name:       "no primary key",
			initial:    ClaudeConfig{"foo": "bar"},
			wantChange: false,
			wantExists: true,
		},
		{
			name:       "only primary key",
			initial:    ClaudeConfig{"primaryApiKey": managedAPIKey},
			wantChange: true,
			wantExists: false,
		},
		{
			name:       "primary key with other fields",
			initial:    ClaudeConfig{"primaryApiKey": managedAPIKey, "foo": "bar"},
			wantChange: true,
			wantExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.WithTempHome(t, func(home string) {
				if tt.initial != nil {
					if err := WriteClaudeConfig(tt.initial); err != nil {
						t.Fatalf("写入初始配置失败: %v", err)
					}
				}

				changed, err := RemoveClaudePlugin()
				if err != nil {
					t.Fatalf("RemoveClaudePlugin() error = %v", err)
				}
				if changed != tt.wantChange {
					t.Fatalf("RemoveClaudePlugin() changed = %v, want %v", changed, tt.wantChange)
				}

				path := filepath.Join(home, claudeDir, claudeConfigFile)
				_, err = os.Stat(path)
				exists := err == nil
				if exists != tt.wantExists {
					t.Fatalf("配置文件存在性不匹配: %v != %v", exists, tt.wantExists)
				}
			})
		})
	}
}

func TestIsClaudePluginAppliedAndStatus(t *testing.T) {
	tests := []struct {
		name        string
		config      ClaudeConfig
		wantApplied bool
		wantExists  bool
	}{
		{
			name:        "no config",
			config:      nil,
			wantApplied: false,
			wantExists:  false,
		},
		{
			name:        "applied",
			config:      ClaudeConfig{"primaryApiKey": managedAPIKey},
			wantApplied: true,
			wantExists:  true,
		},
		{
			name:        "not applied",
			config:      ClaudeConfig{"primaryApiKey": "other"},
			wantApplied: false,
			wantExists:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.WithTempHome(t, func(home string) {
				if tt.config != nil {
					if err := WriteClaudeConfig(tt.config); err != nil {
						t.Fatalf("写入配置失败: %v", err)
					}
				}

				applied, err := IsClaudePluginApplied()
				if err != nil {
					t.Fatalf("IsClaudePluginApplied() error = %v", err)
				}
				if applied != tt.wantApplied {
					t.Fatalf("应用状态不匹配: %v != %v", applied, tt.wantApplied)
				}

				exists, path, err := ClaudeConfigStatus()
				if err != nil {
					t.Fatalf("ClaudeConfigStatus() error = %v", err)
				}
				if exists != tt.wantExists {
					t.Fatalf("存在性不匹配: %v != %v", exists, tt.wantExists)
				}
				if !strings.HasSuffix(path, filepath.Join(claudeDir, claudeConfigFile)) {
					t.Fatalf("路径不正确: %s", path)
				}
				if exists {
					if _, err := os.Stat(path); err != nil {
						t.Fatalf("配置文件不存在: %v", err)
					}
				}
			})
		})
	}
}

func TestClaudePlugin_UserHomeDirError(t *testing.T) {
	orig := claudeUserHomeDirFunc
	claudeUserHomeDirFunc = func() (string, error) { return "", errors.New("no home") }
	t.Cleanup(func() { claudeUserHomeDirFunc = orig })

	if _, err := GetClaudeConfigPath(); err == nil {
		t.Fatalf("expected GetClaudeConfigPath error")
	}
	if err := EnsureClaudeDirExists(); err == nil {
		t.Fatalf("expected EnsureClaudeDirExists error")
	}
	if _, _, err := ClaudeConfigStatus(); err == nil {
		t.Fatalf("expected ClaudeConfigStatus error")
	}
}

func TestWriteClaudeConfig_Errors(t *testing.T) {
	testutil.WithTempHome(t, func(_ string) {
		t.Run("marshal error", func(t *testing.T) {
			err := WriteClaudeConfig(ClaudeConfig{"bad": make(chan int)})
			if err == nil || !strings.Contains(err.Error(), "序列化 Claude 配置失败") {
				t.Fatalf("expected marshal error, got: %v", err)
			}
		})

		t.Run("atomic write error", func(t *testing.T) {
			orig := claudeAtomicWriteFileFunc
			claudeAtomicWriteFileFunc = func(string, []byte, os.FileMode) error {
				return errors.New("write boom")
			}
			t.Cleanup(func() { claudeAtomicWriteFileFunc = orig })

			err := WriteClaudeConfig(ClaudeConfig{"foo": "bar"})
			if err == nil || !strings.Contains(err.Error(), "写入 Claude 配置失败") {
				t.Fatalf("expected write error, got: %v", err)
			}
		})
	})
}

func TestRemoveClaudePlugin_RemoveFileError(t *testing.T) {
	testutil.WithTempHome(t, func(_ string) {
		if err := WriteClaudeConfig(ClaudeConfig{"primaryApiKey": managedAPIKey}); err != nil {
			t.Fatalf("write config: %v", err)
		}

		orig := claudeRemoveFileFunc
		claudeRemoveFileFunc = func(string) error { return errors.New("rm boom") }
		t.Cleanup(func() { claudeRemoveFileFunc = orig })

		_, err := RemoveClaudePlugin()
		if err == nil || !strings.Contains(err.Error(), "删除空配置文件失败") {
			t.Fatalf("expected remove error, got: %v", err)
		}
	})
}

func TestReadClaudeConfig_ReadFileError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 权限测试不稳定，跳过")
	}
	testutil.WithTempHome(t, func(home string) {
		dir := filepath.Join(home, claudeDir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("创建目录失败: %v", err)
		}
		cfgPath := filepath.Join(dir, claudeConfigFile)
		if err := os.WriteFile(cfgPath, []byte(`{"foo":"bar"}`), 0600); err != nil {
			t.Fatalf("写入配置失败: %v", err)
		}
		if err := os.Chmod(cfgPath, 0000); err != nil {
			t.Fatalf("设置权限失败: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(cfgPath, 0600) })

		_, err := ReadClaudeConfig()
		if err == nil || !strings.Contains(err.Error(), "读取 Claude 配置失败") {
			t.Fatalf("expected read error, got: %v", err)
		}
	})
}

func TestApplyClaudePlugin_Errors(t *testing.T) {
	t.Run("read error", func(t *testing.T) {
		testutil.WithTempHome(t, func(home string) {
			dir := filepath.Join(home, claudeDir)
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatalf("创建目录失败: %v", err)
			}
			if err := os.WriteFile(filepath.Join(dir, claudeConfigFile), []byte("{bad"), 0600); err != nil {
				t.Fatalf("写入配置失败: %v", err)
			}

			_, err := ApplyClaudePlugin()
			if err == nil || !strings.Contains(err.Error(), "解析 Claude 配置失败") {
				t.Fatalf("expected parse error, got: %v", err)
			}
		})
	})

	t.Run("write error", func(t *testing.T) {
		testutil.WithTempHome(t, func(_ string) {
			orig := claudeAtomicWriteFileFunc
			claudeAtomicWriteFileFunc = func(string, []byte, os.FileMode) error { return errors.New("write boom") }
			t.Cleanup(func() { claudeAtomicWriteFileFunc = orig })

			_, err := ApplyClaudePlugin()
			if err == nil || !strings.Contains(err.Error(), "写入 Claude 配置失败") {
				t.Fatalf("expected write error, got: %v", err)
			}
		})
	})
}

func TestRemoveClaudePlugin_Errors(t *testing.T) {
	t.Run("read error", func(t *testing.T) {
		testutil.WithTempHome(t, func(home string) {
			dir := filepath.Join(home, claudeDir)
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatalf("创建目录失败: %v", err)
			}
			if err := os.WriteFile(filepath.Join(dir, claudeConfigFile), []byte("{bad"), 0600); err != nil {
				t.Fatalf("写入配置失败: %v", err)
			}

			_, err := RemoveClaudePlugin()
			if err == nil || !strings.Contains(err.Error(), "解析 Claude 配置失败") {
				t.Fatalf("expected parse error, got: %v", err)
			}
		})
	})

	t.Run("write error", func(t *testing.T) {
		testutil.WithTempHome(t, func(_ string) {
			if err := WriteClaudeConfig(ClaudeConfig{"primaryApiKey": managedAPIKey, "foo": "bar"}); err != nil {
				t.Fatalf("write config: %v", err)
			}

			orig := claudeAtomicWriteFileFunc
			claudeAtomicWriteFileFunc = func(string, []byte, os.FileMode) error { return errors.New("write boom") }
			t.Cleanup(func() { claudeAtomicWriteFileFunc = orig })

			_, err := RemoveClaudePlugin()
			if err == nil || !strings.Contains(err.Error(), "写入 Claude 配置失败") {
				t.Fatalf("expected write error, got: %v", err)
			}
		})
	})
}

func TestIsClaudePluginApplied_TypeMismatch(t *testing.T) {
	testutil.WithTempHome(t, func(_ string) {
		if err := WriteClaudeConfig(ClaudeConfig{"primaryApiKey": 123}); err != nil {
			t.Fatalf("write config: %v", err)
		}
		applied, err := IsClaudePluginApplied()
		if err != nil {
			t.Fatalf("IsClaudePluginApplied error: %v", err)
		}
		if applied {
			t.Fatalf("expected not applied for non-string primaryApiKey")
		}
	})
}

func TestClaudeFunctions_UserHomeDirError(t *testing.T) {
	orig := claudeUserHomeDirFunc
	claudeUserHomeDirFunc = func() (string, error) { return "", errors.New("no home") }
	t.Cleanup(func() { claudeUserHomeDirFunc = orig })

	if _, err := ReadClaudeConfig(); err == nil {
		t.Fatalf("expected ReadClaudeConfig error")
	}
	if err := WriteClaudeConfig(ClaudeConfig{"foo": "bar"}); err == nil {
		t.Fatalf("expected WriteClaudeConfig error")
	}
	if _, err := ApplyClaudePlugin(); err == nil {
		t.Fatalf("expected ApplyClaudePlugin error")
	}
	if _, err := RemoveClaudePlugin(); err == nil {
		t.Fatalf("expected RemoveClaudePlugin error")
	}
	if _, err := IsClaudePluginApplied(); err == nil {
		t.Fatalf("expected IsClaudePluginApplied error")
	}
}
