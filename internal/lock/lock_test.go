package lock

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestTryAcquireScenarios(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(path string) error
		wantAcquired  bool
		shouldUpdate  bool
		validatePID   bool
		expectSamePID bool
	}{
		{
			name:         "fresh lock",
			setup:        func(string) error { return nil },
			wantAcquired: true,
			validatePID:  true,
		},
		{
			name:         "existing lock",
			setup:        func(path string) error { return os.WriteFile(path, []byte("123"), 0600) },
			wantAcquired: false,
		},
		{
			name:         "stale lock",
			setup:        func(path string) error { return os.WriteFile(path, []byte("123"), 0600) },
			wantAcquired: true,
			shouldUpdate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			lockPath := filepath.Join(tmpDir, LockFileName)
			if err := tt.setup(lockPath); err != nil {
				t.Fatalf("准备锁文件失败: %v", err)
			}
			if tt.name == "stale lock" {
				staleTime := time.Now().Add(-StaleLockTimeout - time.Minute)
				if err := os.Chtimes(lockPath, staleTime, staleTime); err != nil {
					t.Fatalf("设置时间失败: %v", err)
				}
			}

			lock := NewLock(tmpDir)
			acquired, err := lock.TryAcquire()
			if err != nil {
				t.Fatalf("TryAcquire() error = %v", err)
			}
			if acquired != tt.wantAcquired {
				t.Fatalf("获取锁结果不匹配: %v != %v", acquired, tt.wantAcquired)
			}

			if tt.validatePID && acquired {
				pid, err := lock.GetPID()
				if err != nil {
					t.Fatalf("GetPID() error = %v", err)
				}
				if pid != os.Getpid() {
					t.Fatalf("PID 不匹配: %d", pid)
				}
			}

			if tt.shouldUpdate && acquired {
				data, err := os.ReadFile(lockPath)
				if err != nil {
					t.Fatalf("读取锁文件失败: %v", err)
				}
				if string(data) != strconv.Itoa(os.Getpid()) {
					t.Fatalf("锁文件未更新")
				}
			}
		})
	}
}

func TestForceAcquireTouchAndRelease(t *testing.T) {
	tests := []struct {
		name      string
		withFile  bool
		wantTouch bool
	}{
		{
			name:      "force acquire with existing file",
			withFile:  true,
			wantTouch: true,
		},
		{
			name:      "release without acquire",
			withFile:  false,
			wantTouch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			lockPath := filepath.Join(tmpDir, LockFileName)
			lock := NewLock(tmpDir)

			if tt.withFile {
				if err := os.WriteFile(lockPath, []byte("123"), 0600); err != nil {
					t.Fatalf("写入锁文件失败: %v", err)
				}
				if err := lock.ForceAcquire(); err != nil {
					t.Fatalf("ForceAcquire() error = %v", err)
				}

				oldTime := time.Now().Add(-time.Minute)
				if err := os.Chtimes(lockPath, oldTime, oldTime); err != nil {
					t.Fatalf("设置时间失败: %v", err)
				}
				if err := lock.Touch(); err != nil {
					t.Fatalf("Touch() error = %v", err)
				}
				infoAfter, err := os.Stat(lockPath)
				if err != nil {
					t.Fatalf("获取锁文件失败: %v", err)
				}
				if !infoAfter.ModTime().After(oldTime) {
					t.Fatalf("锁文件时间未更新")
				}
			} else {
				if err := lock.Release(); err != nil {
					t.Fatalf("Release() error = %v", err)
				}
			}
		})
	}
}

func TestGetPIDInvalid(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"invalid pid", "not-a-number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			lockPath := filepath.Join(tmpDir, LockFileName)
			if err := os.WriteFile(lockPath, []byte(tt.data), 0600); err != nil {
				t.Fatalf("写入锁文件失败: %v", err)
			}

			lock := NewLock(tmpDir)
			if _, err := lock.GetPID(); err == nil {
				t.Fatalf("期望解析 PID 失败")
			}
		})
	}
}

func TestPortableModeSkipsLock(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "portable skip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withPortableMarker(t)

			tmpDir := t.TempDir()
			lock := NewLock(tmpDir)

			acquired, err := lock.TryAcquire()
			if err != nil {
				t.Fatalf("TryAcquire() error = %v", err)
			}
			if !acquired {
				t.Fatalf("便携模式下应直接获取锁")
			}

			if err := lock.Touch(); err != nil {
				t.Fatalf("Touch() error = %v", err)
			}
			if err := lock.Release(); err != nil {
				t.Fatalf("Release() error = %v", err)
			}

			if _, err := os.Stat(filepath.Join(tmpDir, LockFileName)); err == nil {
				t.Fatalf("便携模式不应创建锁文件")
			}
		})
	}
}

func withPortableMarker(t *testing.T) {
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

func TestForceAcquirePortableMode(t *testing.T) {
	withPortableMarker(t)

	tmpDir := t.TempDir()
	lock := NewLock(tmpDir)

	if err := lock.ForceAcquire(); err != nil {
		t.Fatalf("ForceAcquire() error = %v", err)
	}

	// 便携模式下不应创建锁文件
	lockPath := filepath.Join(tmpDir, LockFileName)
	if _, err := os.Stat(lockPath); err == nil {
		t.Fatalf("便携模式不应创建锁文件")
	}
}

func TestReleaseAfterAcquire(t *testing.T) {
	tmpDir := t.TempDir()
	lock := NewLock(tmpDir)

	// 先获取锁
	acquired, err := lock.TryAcquire()
	if err != nil {
		t.Fatalf("TryAcquire() error = %v", err)
	}
	if !acquired {
		t.Fatalf("应该获取到锁")
	}

	// 验证锁文件存在
	lockPath := filepath.Join(tmpDir, LockFileName)
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("锁文件应该存在: %v", err)
	}

	// 释放锁
	if err := lock.Release(); err != nil {
		t.Fatalf("Release() error = %v", err)
	}

	// 验证锁文件已删除
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("锁文件应该被删除")
	}
}

func TestForceAcquireNewFile(t *testing.T) {
	tmpDir := t.TempDir()
	lock := NewLock(tmpDir)

	// ForceAcquire 在无现有锁时应创建新锁
	if err := lock.ForceAcquire(); err != nil {
		t.Fatalf("ForceAcquire() error = %v", err)
	}

	// 验证锁文件存在
	lockPath := filepath.Join(tmpDir, LockFileName)
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("锁文件应该存在: %v", err)
	}

	// 验证 PID
	pid, err := lock.GetPID()
	if err != nil {
		t.Fatalf("GetPID() error = %v", err)
	}
	if pid != os.Getpid() {
		t.Fatalf("PID 不匹配: %d != %d", pid, os.Getpid())
	}
}
