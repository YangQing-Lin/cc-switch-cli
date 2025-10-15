package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFileExists(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	// 测试不存在的文件
	nonExistentPath := filepath.Join(tmpDir, "nonexistent.txt")
	if FileExists(nonExistentPath) {
		t.Errorf("FileExists() 对不存在的文件应返回 false")
	}

	// 创建文件
	existingPath := filepath.Join(tmpDir, "existing.txt")
	if err := os.WriteFile(existingPath, []byte("test"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 测试存在的文件
	if !FileExists(existingPath) {
		t.Errorf("FileExists() 对存在的文件应返回 true")
	}
}

func TestAtomicWriteFile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name      string
		path      string
		data      []byte
		perm      os.FileMode
		wantErr   bool
		skipOnWin bool // Windows 上跳过权限测试
	}{
		{
			name:    "写入新文件",
			path:    filepath.Join(tmpDir, "new.txt"),
			data:    []byte("hello world"),
			perm:    0644,
			wantErr: false,
		},
		{
			name:    "覆盖已存在文件",
			path:    filepath.Join(tmpDir, "existing.txt"),
			data:    []byte("updated content"),
			perm:    0644,
			wantErr: false,
		},
		{
			name:      "设置严格权限",
			path:      filepath.Join(tmpDir, "secure.txt"),
			data:      []byte("secret"),
			perm:      0600,
			wantErr:   false,
			skipOnWin: true,
		},
		{
			name:    "默认权限（0）",
			path:    filepath.Join(tmpDir, "default.txt"),
			data:    []byte("default"),
			perm:    0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnWin && runtime.GOOS == "windows" {
				t.Skip("跳过 Windows 上的权限测试")
			}

			// 如果是覆盖测试，先创建原文件
			if tt.name == "覆盖已存在文件" {
				if err := os.WriteFile(tt.path, []byte("old content"), 0644); err != nil {
					t.Fatalf("创建原文件失败: %v", err)
				}
			}

			// 执行原子写入
			err := AtomicWriteFile(tt.path, tt.data, tt.perm)
			if (err != nil) != tt.wantErr {
				t.Errorf("AtomicWriteFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// 验证文件内容
			content, err := os.ReadFile(tt.path)
			if err != nil {
				t.Fatalf("读取文件失败: %v", err)
			}
			if string(content) != string(tt.data) {
				t.Errorf("文件内容不匹配\n期望: %s\n实际: %s", tt.data, content)
			}

			// 验证文件权限（非Windows）
			if runtime.GOOS != "windows" && tt.perm != 0 {
				info, err := os.Stat(tt.path)
				if err != nil {
					t.Fatalf("获取文件信息失败: %v", err)
				}
				actualPerm := info.Mode().Perm()
				if actualPerm != tt.perm {
					t.Errorf("文件权限不匹配\n期望: %o\n实际: %o", tt.perm, actualPerm)
				}
			}

			// 验证临时文件已清理
			tmpFiles, _ := filepath.Glob(filepath.Join(tmpDir, ".tmp-*"))
			if len(tmpFiles) > 0 {
				t.Errorf("临时文件未清理: %v", tmpFiles)
			}
		})
	}
}

func TestWriteJSONFile(t *testing.T) {
	tmpDir := t.TempDir()

	type testData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name    string
		path    string
		data    interface{}
		perm    os.FileMode
		wantErr bool
	}{
		{
			name:    "写入有效JSON",
			path:    filepath.Join(tmpDir, "valid.json"),
			data:    testData{Name: "test", Value: 42},
			perm:    0644,
			wantErr: false,
		},
		{
			name:    "写入带权限的JSON",
			path:    filepath.Join(tmpDir, "secure.json"),
			data:    map[string]string{"key": "value"},
			perm:    0600,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WriteJSONFile(tt.path, tt.data, tt.perm)
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteJSONFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// 验证文件存在
			if !FileExists(tt.path) {
				t.Errorf("JSON文件未创建")
			}

			// 验证可以读回JSON
			var result interface{}
			err = ReadJSONFile(tt.path, &result)
			if err != nil {
				t.Errorf("读取JSON失败: %v", err)
			}
		})
	}
}

func TestReadJSONFile(t *testing.T) {
	tmpDir := t.TempDir()

	type testData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name:    "有效JSON",
			content: `{"name":"test","value":42}`,
			wantErr: false,
		},
		{
			name:    "无效JSON",
			content: `{invalid json}`,
			wantErr: true,
		},
		{
			name:    "空文件",
			content: ``,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmpDir, tt.name+".json")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatalf("创建测试文件失败: %v", err)
			}

			var result testData
			err := ReadJSONFile(path, &result)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadJSONFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && (result.Name != "test" || result.Value != 42) {
				t.Errorf("读取的数据不正确: %+v", result)
			}
		})
	}

	// 测试读取不存在的文件
	t.Run("文件不存在", func(t *testing.T) {
		var result testData
		err := ReadJSONFile(filepath.Join(tmpDir, "nonexistent.json"), &result)
		if err == nil {
			t.Errorf("读取不存在的文件应该返回错误")
		}
	})
}

func TestBackupFile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		fileExists bool
		wantErr    bool
	}{
		{
			name:       "备份已存在文件",
			fileExists: true,
			wantErr:    false,
		},
		{
			name:       "备份不存在的文件",
			fileExists: false,
			wantErr:    false, // 不存在不算错误
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmpDir, tt.name+".txt")
			backupPath := path + ".backup"

			// 创建原文件
			if tt.fileExists {
				if err := os.WriteFile(path, []byte("original content"), 0644); err != nil {
					t.Fatalf("创建原文件失败: %v", err)
				}
			}

			// 执行备份
			err := BackupFile(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("BackupFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 验证备份结果
			if tt.fileExists {
				if !FileExists(backupPath) {
					t.Errorf("备份文件未创建")
				}

				// 验证备份内容
				content, err := os.ReadFile(backupPath)
				if err != nil {
					t.Fatalf("读取备份文件失败: %v", err)
				}
				if string(content) != "original content" {
					t.Errorf("备份内容不匹配")
				}
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		srcContent string
		srcPerm    os.FileMode
		wantErr    bool
	}{
		{
			name:       "复制普通文件",
			srcContent: "test content",
			srcPerm:    0644,
			wantErr:    false,
		},
		{
			name:       "复制带权限的文件",
			srcContent: "secret",
			srcPerm:    0600,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcPath := filepath.Join(tmpDir, "src_"+tt.name)
			dstPath := filepath.Join(tmpDir, "dst_"+tt.name)

			// 创建源文件
			if err := os.WriteFile(srcPath, []byte(tt.srcContent), tt.srcPerm); err != nil {
				t.Fatalf("创建源文件失败: %v", err)
			}

			// 执行复制
			err := CopyFile(srcPath, dstPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("CopyFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// 验证目标文件存在
			if !FileExists(dstPath) {
				t.Errorf("目标文件未创建")
			}

			// 验证内容
			content, err := os.ReadFile(dstPath)
			if err != nil {
				t.Fatalf("读取目标文件失败: %v", err)
			}
			if string(content) != tt.srcContent {
				t.Errorf("复制内容不匹配\n期望: %s\n实际: %s", tt.srcContent, content)
			}

			// 验证权限（非Windows）
			if runtime.GOOS != "windows" {
				srcInfo, _ := os.Stat(srcPath)
				dstInfo, _ := os.Stat(dstPath)
				if srcInfo.Mode().Perm() != dstInfo.Mode().Perm() {
					t.Errorf("权限未正确复制")
				}
			}
		})
	}

	// 测试复制不存在的文件
	t.Run("源文件不存在", func(t *testing.T) {
		err := CopyFile(
			filepath.Join(tmpDir, "nonexistent.txt"),
			filepath.Join(tmpDir, "dest.txt"),
		)
		if err == nil {
			t.Errorf("复制不存在的文件应该返回错误")
		}
	})
}
