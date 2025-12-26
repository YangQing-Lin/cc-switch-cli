package portable

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestIsPortableMode(t *testing.T) {
	tests := []struct {
		name       string
		setupFunc  func(t *testing.T)
		want       bool
		shouldSkip bool
	}{
		{
			name: "no marker file",
			setupFunc: func(t *testing.T) {
				if markerExists() {
					t.Skip("检测到已有 portable.ini，跳过")
				}
			},
			want: false,
		},
		{
			name: "marker file",
			setupFunc: func(t *testing.T) {
				withPortableMarker(t, false)
			},
			want: true,
		},
		{
			name: "marker directory",
			setupFunc: func(t *testing.T) {
				withPortableMarker(t, true)
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFunc(t)
			if got := IsPortableMode(); got != tt.want {
				t.Fatalf("IsPortableMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPortableConfigDir(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "config dir path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configDir, err := GetPortableConfigDir()
			if err != nil {
				t.Fatalf("GetPortableConfigDir() error = %v", err)
			}

			execPath, err := os.Executable()
			if err != nil {
				t.Fatalf("获取可执行文件失败: %v", err)
			}
			expected := filepath.Join(filepath.Dir(execPath), ".cc-switch")
			if configDir != expected {
				t.Fatalf("路径不匹配: %s != %s", configDir, expected)
			}
		})
	}
}

func TestPortableExecutableError(t *testing.T) {
	orig := portableExecutableFunc
	portableExecutableFunc = func() (string, error) { return "", errors.New("exec error") }
	t.Cleanup(func() { portableExecutableFunc = orig })

	if IsPortableMode() {
		t.Fatalf("expected IsPortableMode false on exec error")
	}
	if _, err := GetPortableConfigDir(); err == nil {
		t.Fatalf("expected GetPortableConfigDir error")
	}
}

func withPortableMarker(t *testing.T, asDir bool) {
	t.Helper()

	execPath, err := os.Executable()
	if err != nil {
		t.Fatalf("获取可执行文件失败: %v", err)
	}
	marker := filepath.Join(filepath.Dir(execPath), "portable.ini")

	// 先检查是否已存在，避免误删用户环境中的 marker
	if _, err := os.Lstat(marker); err == nil {
		t.Skip("已有 portable.ini，跳过")
	} else if !errors.Is(err, fs.ErrNotExist) {
		t.Skipf("无法检查 portable.ini: %v", err)
	}

	if asDir {
		if err := os.Mkdir(marker, 0755); err != nil {
			if errors.Is(err, fs.ErrExist) {
				t.Skip("已有 portable.ini，跳过")
			}
			t.Skipf("无法创建 portable.ini 目录: %v", err)
		}
		t.Cleanup(func() {
			if err := os.RemoveAll(marker); err != nil {
				_ = os.Chmod(marker, 0777)
				if err2 := os.RemoveAll(marker); err2 != nil {
					t.Logf("清理 portable.ini 失败: %v", err2)
				}
			}
		})
		return
	}

	f, err := os.OpenFile(marker, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			t.Skip("已有 portable.ini，跳过")
		}
		t.Skipf("无法创建 portable.ini: %v", err)
	}
	_ = f.Close()
	t.Cleanup(func() {
		if err := os.RemoveAll(marker); err != nil {
			_ = os.Chmod(marker, 0777)
			if err2 := os.RemoveAll(marker); err2 != nil {
				t.Logf("清理 portable.ini 失败: %v", err2)
			}
		}
	})
}

func markerExists() bool {
	execPath, err := os.Executable()
	if err != nil {
		return false
	}
	marker := filepath.Join(filepath.Dir(execPath), "portable.ini")
	_, err = os.Stat(marker)
	return err == nil
}
