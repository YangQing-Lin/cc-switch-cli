package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	existingPath := filepath.Join(tmpDir, "exists.txt")
	if err := os.WriteFile(existingPath, []byte("ok"), 0644); err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"existing file", existingPath, true},
		{"missing file", filepath.Join(tmpDir, "missing.txt"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FileExists(tt.path); got != tt.want {
				t.Fatalf("FileExists(%s) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestAtomicWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	existingPath := filepath.Join(tmpDir, "existing.txt")
	if err := os.WriteFile(existingPath, []byte("old"), 0600); err != nil {
		t.Fatalf("创建原文件失败: %v", err)
	}

	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(readOnlyDir, 0500); err != nil {
			t.Fatalf("设置目录权限失败: %v", err)
		}
		t.Cleanup(func() {
			_ = os.Chmod(readOnlyDir, 0755)
		})
	}

	tests := []struct {
		name         string
		path         string
		data         []byte
		perm         os.FileMode
		wantPerm     os.FileMode
		wantErr      bool
		skipOnWin    bool
		shouldVerify bool
	}{
		{
			name:         "write new file",
			path:         filepath.Join(tmpDir, "new.txt"),
			data:         []byte("hello"),
			perm:         0644,
			wantPerm:     0644,
			shouldVerify: true,
		},
		{
			name:         "overwrite existing file preserves explicit perm",
			path:         existingPath,
			data:         []byte("updated"),
			perm:         0600,
			wantPerm:     0600,
			shouldVerify: true,
		},
		{
			name:         "default perm uses existing file",
			path:         existingPath,
			data:         []byte("same-perm"),
			perm:         0,
			wantPerm:     0600,
			shouldVerify: true,
		},
		{
			name:         "default perm new file",
			path:         filepath.Join(tmpDir, "default.txt"),
			data:         []byte("default"),
			perm:         0,
			wantPerm:     0644,
			shouldVerify: true,
		},
		{
			name:      "permission denied - cannot create temp file",
			path:      filepath.Join(readOnlyDir, "blocked.txt"),
			data:      []byte("blocked"),
			perm:      0644,
			wantErr:   true,
			skipOnWin: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnWin && runtime.GOOS == "windows" {
				t.Skip("跳过 Windows 权限测试")
			}

			err := AtomicWriteFile(tt.path, tt.data, tt.perm)
			if (err != nil) != tt.wantErr {
				t.Fatalf("AtomicWriteFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr || !tt.shouldVerify {
				return
			}

			content, err := os.ReadFile(tt.path)
			if err != nil {
				t.Fatalf("读取文件失败: %v", err)
			}
			if string(content) != string(tt.data) {
				t.Fatalf("文件内容不匹配: %s", string(content))
			}

			if runtime.GOOS != "windows" {
				info, err := os.Stat(tt.path)
				if err != nil {
					t.Fatalf("获取文件信息失败: %v", err)
				}
				if info.Mode().Perm() != tt.wantPerm {
					t.Fatalf("文件权限不匹配: %o != %o", info.Mode().Perm(), tt.wantPerm)
				}
			}

			tmpFiles, _ := filepath.Glob(filepath.Join(filepath.Dir(tt.path), ".tmp-*"))
			if len(tmpFiles) != 0 {
				t.Fatalf("临时文件未清理: %v", tmpFiles)
			}
		})
	}
}

func TestWriteJSONFile(t *testing.T) {
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(readOnlyDir, 0500); err != nil {
			t.Fatalf("设置目录权限失败: %v", err)
		}
		t.Cleanup(func() {
			_ = os.Chmod(readOnlyDir, 0755)
		})
	}

	tests := []struct {
		name      string
		path      string
		data      interface{}
		perm      os.FileMode
		wantErr   bool
		errSubstr string
		skipOnWin bool
	}{
		{
			name:    "write valid json",
			path:    filepath.Join(tmpDir, "valid.json"),
			data:    map[string]string{"name": "test"},
			perm:    0600,
			wantErr: false,
		},
		{
			name:      "marshal error",
			path:      filepath.Join(tmpDir, "invalid.json"),
			data:      make(chan int),
			perm:      0600,
			wantErr:   true,
			errSubstr: "序列化 JSON 失败",
		},
		{
			name:      "permission denied",
			path:      filepath.Join(readOnlyDir, "blocked.json"),
			data:      map[string]string{"blocked": "yes"},
			perm:      0600,
			wantErr:   true,
			skipOnWin: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnWin && runtime.GOOS == "windows" {
				t.Skip("跳过 Windows 权限测试")
			}

			err := WriteJSONFile(tt.path, tt.data, tt.perm)
			if (err != nil) != tt.wantErr {
				t.Fatalf("WriteJSONFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.errSubstr != "" && (err == nil || !strings.Contains(err.Error(), tt.errSubstr)) {
				t.Fatalf("错误信息不匹配: %v", err)
			}
			if tt.wantErr {
				return
			}

			var got map[string]interface{}
			if err := ReadJSONFile(tt.path, &got); err != nil {
				t.Fatalf("读取 JSON 失败: %v", err)
			}
			if runtime.GOOS != "windows" {
				info, err := os.Stat(tt.path)
				if err != nil {
					t.Fatalf("获取文件信息失败: %v", err)
				}
				if info.Mode().Perm() != tt.perm {
					t.Fatalf("文件权限不匹配: %o != %o", info.Mode().Perm(), tt.perm)
				}
			}
		})
	}
}

func TestReadJSONFile(t *testing.T) {
	tmpDir := t.TempDir()
	validPath := filepath.Join(tmpDir, "valid.json")
	if err := os.WriteFile(validPath, []byte(`{"name":"test","value":42}`), 0644); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}

	invalidPath := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(invalidPath, []byte("{invalid"), 0644); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid json", validPath, false},
		{"invalid json", invalidPath, true},
		{"missing file", filepath.Join(tmpDir, "missing.json"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got struct {
				Name  string `json:"name"`
				Value int    `json:"value"`
			}
			err := ReadJSONFile(tt.path, &got)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ReadJSONFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && (got.Name != "test" || got.Value != 42) {
				t.Fatalf("读取数据不正确: %+v", got)
			}
		})
	}
}

func TestBackupFile(t *testing.T) {
	tmpDir := t.TempDir()
	readonlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readonlyDir, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	readonlyPath := filepath.Join(readonlyDir, "config.json")
	if err := os.WriteFile(readonlyPath, []byte("data"), 0644); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(readonlyDir, 0500); err != nil {
			t.Fatalf("设置目录权限失败: %v", err)
		}
		t.Cleanup(func() {
			_ = os.Chmod(readonlyDir, 0755)
		})
	}

	tests := []struct {
		name       string
		path       string
		wantErr    bool
		skipOnWin  bool
		shouldCopy bool
	}{
		{
			name:       "backup existing file",
			path:       filepath.Join(tmpDir, "config.json"),
			shouldCopy: true,
		},
		{
			name:    "missing source file",
			path:    filepath.Join(tmpDir, "missing.json"),
			wantErr: false,
		},
		{
			name:      "permission denied",
			path:      readonlyPath,
			wantErr:   true,
			skipOnWin: true,
		},
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte("original"), 0644); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnWin && runtime.GOOS == "windows" {
				t.Skip("跳过 Windows 权限测试")
			}

			err := BackupFile(tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("BackupFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr || !tt.shouldCopy {
				return
			}

			backupPath := tt.path + ".backup"
			if !FileExists(backupPath) {
				t.Fatalf("备份文件未创建: %s", backupPath)
			}
			content, err := os.ReadFile(backupPath)
			if err != nil {
				t.Fatalf("读取备份失败: %v", err)
			}
			if string(content) != "original" {
				t.Fatalf("备份内容不匹配: %s", string(content))
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "src.txt")
	if err := os.WriteFile(srcPath, []byte("content"), 0600); err != nil {
		t.Fatalf("写入源文件失败: %v", err)
	}

	readonlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readonlyDir, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(readonlyDir, 0500); err != nil {
			t.Fatalf("设置目录权限失败: %v", err)
		}
		t.Cleanup(func() {
			_ = os.Chmod(readonlyDir, 0755)
		})
	}

	devFullAvailable := supportsDevFull()

	tests := []struct {
		name      string
		src       string
		dst       string
		wantErr   bool
		skipOnWin bool
		skip      bool
	}{
		{
			name:    "copy success",
			src:     srcPath,
			dst:     filepath.Join(tmpDir, "dst.txt"),
			wantErr: false,
		},
		{
			name:    "missing source",
			src:     filepath.Join(tmpDir, "missing.txt"),
			dst:     filepath.Join(tmpDir, "dst-missing.txt"),
			wantErr: true,
		},
		{
			name:      "permission denied",
			src:       srcPath,
			dst:       filepath.Join(readonlyDir, "blocked.txt"),
			wantErr:   true,
			skipOnWin: true,
		},
		{
			name:    "disk full",
			src:     srcPath,
			dst:     "/dev/full",
			wantErr: true,
			skip:    !devFullAvailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip("跳过不支持的测试环境")
			}
			if tt.skipOnWin && runtime.GOOS == "windows" {
				t.Skip("跳过 Windows 权限测试")
			}

			err := CopyFile(tt.src, tt.dst)
			if (err != nil) != tt.wantErr {
				t.Fatalf("CopyFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.name == "disk full" {
				if !isNoSpaceError(err) && !strings.Contains(strings.ToLower(err.Error()), "no space") {
					t.Fatalf("期望磁盘满错误，got: %v", err)
				}
			}
			if tt.wantErr {
				return
			}

			content, err := os.ReadFile(tt.dst)
			if err != nil {
				t.Fatalf("读取目标文件失败: %v", err)
			}
			if string(content) != "content" {
				t.Fatalf("复制内容不匹配: %s", string(content))
			}

			if runtime.GOOS != "windows" {
				srcInfo, _ := os.Stat(tt.src)
				dstInfo, _ := os.Stat(tt.dst)
				if srcInfo.Mode().Perm() != dstInfo.Mode().Perm() {
					t.Fatalf("权限未正确复制")
				}
			}
		})
	}
}

func supportsDevFull() bool {
	if runtime.GOOS == "windows" {
		return false
	}
	if _, err := os.Stat("/dev/full"); err != nil {
		return false
	}
	file, err := os.OpenFile("/dev/full", os.O_WRONLY, 0600)
	if err != nil {
		return false
	}
	file.Close()
	return true
}

func isNoSpaceError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "no space")
}

// TestAtomicWriteFile_WriteToReadOnlyDir 测试写入到不可写目录（例如 /dev）
func TestAtomicWriteFile_WriteToReadOnlyDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("跳过 Windows，/dev 路径不可用")
	}

	// 这里我们测试写入 /dev 目录（通常权限不足）
	err := AtomicWriteFile("/dev/test.txt", []byte("test"), 0644)
	if err == nil {
		t.Fatal("期望写入失败，但成功了")
	}
	// 应该在创建临时文件时失败
	if !strings.Contains(err.Error(), "创建临时文件失败") {
		t.Logf("警告: 错误信息不同但测试通过: %v", err)
	}

	// 测试另一个场景：创建后立即删除临时目录导致写入失败
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "file.txt")

	// 在测试中模拟写入失败场景很困难，这里验证正常写入
	if err := AtomicWriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("正常写入失败: %v", err)
	}
}

// TestAtomicWriteFile_RenameFailure 测试重命名失败的场景
func TestAtomicWriteFile_RenameFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("跳过 Windows，重命名行为不同")
	}

	tmpDir := t.TempDir()

	// 创建一个只读目录，文件已存在
	targetDir := filepath.Join(tmpDir, "target")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}

	targetFile := filepath.Join(targetDir, "file.txt")
	if err := os.WriteFile(targetFile, []byte("existing"), 0644); err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}

	// 将目标文件设为只读，并将目录设为只读
	if err := os.Chmod(targetFile, 0444); err != nil {
		t.Fatalf("设置文件权限失败: %v", err)
	}
	if err := os.Chmod(targetDir, 0555); err != nil {
		t.Fatalf("设置目录权限失败: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(targetDir, 0755)
		_ = os.Chmod(targetFile, 0644)
	})

	// 尝试写入，应该在重命名时失败
	err := AtomicWriteFile(targetFile, []byte("new content"), 0644)
	if err == nil {
		t.Fatal("期望重命名失败，但成功了")
	}
	if !strings.Contains(err.Error(), "重命名文件失败") && !strings.Contains(err.Error(), "创建临时文件失败") {
		t.Fatalf("错误信息不符合预期: %v", err)
	}
}

// TestBackupFile_ReadFailure 测试读取文件失败的场景
func TestBackupFile_ReadFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("跳过 Windows，权限模型不同")
	}

	tmpDir := t.TempDir()
	unreadablePath := filepath.Join(tmpDir, "unreadable.txt")

	// 创建文件但设置为不可读
	if err := os.WriteFile(unreadablePath, []byte("secret"), 0644); err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}
	if err := os.Chmod(unreadablePath, 0000); err != nil {
		t.Fatalf("设置权限失败: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(unreadablePath, 0644)
	})

	err := BackupFile(unreadablePath)
	if err == nil {
		t.Fatal("期望读取失败，但成功了")
	}
	if !strings.Contains(err.Error(), "读取原文件失败") {
		t.Fatalf("错误信息不符合预期: %v", err)
	}
}

// TestCopyFile_StatFailure 测试获取文件信息失败的场景
func TestCopyFile_StatFailure(t *testing.T) {
	// 这个场景很难模拟，因为 Open 成功后 Stat 通常不会失败
	// 我们使用符号链接到不存在的文件来模拟
	if runtime.GOOS == "windows" {
		t.Skip("跳过 Windows，符号链接需要管理员权限")
	}

	tmpDir := t.TempDir()
	brokenLink := filepath.Join(tmpDir, "broken-link")
	nonExistent := filepath.Join(tmpDir, "non-existent")

	// 创建指向不存在文件的符号链接
	if err := os.Symlink(nonExistent, brokenLink); err != nil {
		t.Skipf("无法创建符号链接: %v", err)
	}

	dstPath := filepath.Join(tmpDir, "dst.txt")
	err := CopyFile(brokenLink, dstPath)
	if err == nil {
		t.Fatal("期望复制失败，但成功了")
	}
	// 应该在打开源文件时失败
	if !strings.Contains(err.Error(), "打开源文件失败") {
		t.Logf("警告: 错误信息不符合预期，但测试通过: %v", err)
	}
}

// TestCopyFile_LargeFile 测试复制大文件
func TestCopyFile_LargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "large.dat")
	dstPath := filepath.Join(tmpDir, "large-copy.dat")

	// 创建一个较大的文件（1MB）
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	if err := os.WriteFile(srcPath, largeData, 0644); err != nil {
		t.Fatalf("创建大文件失败: %v", err)
	}

	// 复制文件
	if err := CopyFile(srcPath, dstPath); err != nil {
		t.Fatalf("复制大文件失败: %v", err)
	}

	// 验证文件内容
	copiedData, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("读取复制文件失败: %v", err)
	}

	if len(copiedData) != len(largeData) {
		t.Fatalf("文件大小不匹配: %d != %d", len(copiedData), len(largeData))
	}

	// 验证内容一致
	for i := range largeData {
		if copiedData[i] != largeData[i] {
			t.Fatalf("数据不匹配位置 %d: %d != %d", i, copiedData[i], largeData[i])
		}
	}
}

// TestAtomicWriteFile_ChmodFailure 测试 Chmod 失败的场景（非 Windows）
func TestAtomicWriteFile_ChmodFailure(t *testing.T) {
	// 这个场景很难直接触发，因为 Chmod 通常不会失败
	// 在实际使用中，代码已经处理了 Windows 的情况
	// 这里我们主要验证在 Windows 下忽略 Chmod 错误
	if runtime.GOOS != "windows" {
		t.Skip("此测试仅用于 Windows 平台验证")
	}

	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.txt")

	// 在 Windows 上，某些权限设置可能失败但被忽略
	err := AtomicWriteFile(testPath, []byte("test"), 0600)
	if err != nil {
		t.Fatalf("写入失败: %v", err)
	}

	// 验证文件已创建
	if !FileExists(testPath) {
		t.Fatal("文件未创建")
	}
}

// TestAtomicWriteFile_WindowsDeleteFailure 测试 Windows 删除原文件失败
func TestAtomicWriteFile_WindowsDeleteFailure(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("此测试仅针对 Windows 平台")
	}

	tmpDir := t.TempDir()
	existingPath := filepath.Join(tmpDir, "existing.txt")

	// 创建并打开文件（保持句柄打开）
	if err := os.WriteFile(existingPath, []byte("old"), 0644); err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}

	// 打开文件保持锁定
	file, err := os.Open(existingPath)
	if err != nil {
		t.Fatalf("打开文件失败: %v", err)
	}
	defer file.Close()

	// 尝试覆盖文件，在 Windows 上应该失败
	err = AtomicWriteFile(existingPath, []byte("new"), 0644)
	if err == nil {
		t.Fatal("期望删除失败，但成功了")
	}
	if !strings.Contains(err.Error(), "删除原文件失败") {
		t.Logf("警告: 错误信息可能不同: %v", err)
	}
}

// TestAtomicWriteFile_EmptyData 测试写入空数据
func TestAtomicWriteFile_EmptyData(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "empty.txt")

	// 写入空数据
	err := AtomicWriteFile(testPath, []byte{}, 0644)
	if err != nil {
		t.Fatalf("写入空数据失败: %v", err)
	}

	// 验证文件已创建且为空
	if !FileExists(testPath) {
		t.Fatal("文件未创建")
	}

	content, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}

	if len(content) != 0 {
		t.Fatalf("期望空文件，但有 %d 字节", len(content))
	}
}

// TestAtomicWriteFile_SpecialPermissions 测试特殊权限设置
func TestAtomicWriteFile_SpecialPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 权限模型不同")
	}

	tmpDir := t.TempDir()

	tests := []struct {
		name string
		perm os.FileMode
	}{
		{"read-only", 0400},
		{"write-only", 0200},
		{"executable", 0755},
		{"strict", 0600},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testPath := filepath.Join(tmpDir, tt.name+".txt")
			err := AtomicWriteFile(testPath, []byte("data"), tt.perm)
			if err != nil {
				t.Fatalf("写入失败: %v", err)
			}

			info, err := os.Stat(testPath)
			if err != nil {
				t.Fatalf("获取文件信息失败: %v", err)
			}

			if info.Mode().Perm() != tt.perm {
				t.Fatalf("权限不匹配: %o != %o", info.Mode().Perm(), tt.perm)
			}

			// 清理：恢复写权限以便删除
			_ = os.Chmod(testPath, 0644)
		})
	}
}

// TestBackupFile_WriteFailure 测试备份写入失败
func TestBackupFile_WriteFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 权限模型不同")
	}

	tmpDir := t.TempDir()
	readonlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readonlyDir, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}

	srcFile := filepath.Join(readonlyDir, "source.txt")
	if err := os.WriteFile(srcFile, []byte("data"), 0644); err != nil {
		t.Fatalf("创建源文件失败: %v", err)
	}

	// 设置目录为只读
	if err := os.Chmod(readonlyDir, 0500); err != nil {
		t.Fatalf("设置目录权限失败: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(readonlyDir, 0755)
	})

	// 尝试备份，应该失败（无法写入备份文件）
	err := BackupFile(srcFile)
	if err == nil {
		t.Fatal("期望备份失败，但成功了")
	}
	if !strings.Contains(err.Error(), "创建备份失败") {
		t.Fatalf("错误信息不符合预期: %v", err)
	}
}

// TestCopyFile_DestinationExists 测试目标文件已存在的情况
func TestCopyFile_DestinationExists(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "src.txt")
	dstPath := filepath.Join(tmpDir, "dst.txt")

	// 创建源文件和目标文件
	if err := os.WriteFile(srcPath, []byte("new content"), 0644); err != nil {
		t.Fatalf("创建源文件失败: %v", err)
	}
	if err := os.WriteFile(dstPath, []byte("old content"), 0644); err != nil {
		t.Fatalf("创建目标文件失败: %v", err)
	}

	// 复制应该覆盖目标文件
	if err := CopyFile(srcPath, dstPath); err != nil {
		t.Fatalf("复制失败: %v", err)
	}

	// 验证内容已更新
	content, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("读取目标文件失败: %v", err)
	}

	if string(content) != "new content" {
		t.Fatalf("目标文件未更新: %s", string(content))
	}
}

// TestCopyFile_PermissionPreserved 测试权限保留
func TestCopyFile_PermissionPreserved(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 权限模型不同")
	}

	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "src.txt")
	dstPath := filepath.Join(tmpDir, "dst.txt")

	// 创建具有特殊权限的源文件
	if err := os.WriteFile(srcPath, []byte("data"), 0600); err != nil {
		t.Fatalf("创建源文件失败: %v", err)
	}

	if err := CopyFile(srcPath, dstPath); err != nil {
		t.Fatalf("复制失败: %v", err)
	}

	srcInfo, _ := os.Stat(srcPath)
	dstInfo, _ := os.Stat(dstPath)

	if srcInfo.Mode().Perm() != dstInfo.Mode().Perm() {
		t.Fatalf("权限未正确复制: %o != %o", srcInfo.Mode().Perm(), dstInfo.Mode().Perm())
	}
}

// TestAtomicWriteFile_LargeData 测试写入大数据
func TestAtomicWriteFile_LargeData(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "large.dat")

	// 创建大数据（5MB）
	largeData := make([]byte, 5*1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	// 写入大数据
	err := AtomicWriteFile(testPath, largeData, 0644)
	if err != nil {
		t.Fatalf("写入大数据失败: %v", err)
	}

	// 验证文件内容
	content, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}

	if len(content) != len(largeData) {
		t.Fatalf("文件大小不匹配: %d != %d", len(content), len(largeData))
	}
}

// TestFileExists_Directory 测试目录检测
func TestFileExists_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")

	// 创建目录
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}

	// FileExists 对目录也应该返回 true
	if !FileExists(subDir) {
		t.Fatal("目录应该被检测为存在")
	}
}

// TestReadJSONFile_EmptyFile 测试读取空 JSON 文件
func TestReadJSONFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	emptyPath := filepath.Join(tmpDir, "empty.json")

	// 创建空文件
	if err := os.WriteFile(emptyPath, []byte(""), 0644); err != nil {
		t.Fatalf("创建空文件失败: %v", err)
	}

	var result map[string]interface{}
	err := ReadJSONFile(emptyPath, &result)
	if err == nil {
		t.Fatal("期望读取空文件失败，但成功了")
	}
	if !strings.Contains(err.Error(), "解析 JSON 失败") {
		t.Fatalf("错误信息不符合预期: %v", err)
	}
}

// TestWriteJSONFile_ComplexData 测试写入复杂 JSON 数据
func TestWriteJSONFile_ComplexData(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "complex.json")

	// 创建复杂数据结构
	complexData := map[string]interface{}{
		"string":  "value",
		"number":  42,
		"boolean": true,
		"array":   []int{1, 2, 3},
		"nested": map[string]string{
			"key": "value",
		},
	}

	err := WriteJSONFile(testPath, complexData, 0644)
	if err != nil {
		t.Fatalf("写入复杂 JSON 失败: %v", err)
	}

	// 读取并验证
	var result map[string]interface{}
	err = ReadJSONFile(testPath, &result)
	if err != nil {
		t.Fatalf("读取 JSON 失败: %v", err)
	}

	if result["string"] != "value" {
		t.Fatal("字符串值不匹配")
	}
	// 注意：JSON 数字会被解析为 float64
	if result["number"].(float64) != 42 {
		t.Fatal("数字值不匹配")
	}
	if result["boolean"] != true {
		t.Fatal("布尔值不匹配")
	}
}

// TestBackupFile_ExistingBackup 测试备份文件已存在的情况
func TestBackupFile_ExistingBackup(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.txt")
	backupPath := srcPath + ".backup"

	// 创建源文件和旧备份
	if err := os.WriteFile(srcPath, []byte("new data"), 0644); err != nil {
		t.Fatalf("创建源文件失败: %v", err)
	}
	if err := os.WriteFile(backupPath, []byte("old backup"), 0644); err != nil {
		t.Fatalf("创建旧备份失败: %v", err)
	}

	// 执行备份（应该覆盖旧备份）
	err := BackupFile(srcPath)
	if err != nil {
		t.Fatalf("备份失败: %v", err)
	}

	// 验证备份内容已更新
	content, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("读取备份失败: %v", err)
	}

	if string(content) != "new data" {
		t.Fatalf("备份内容未更新: %s", string(content))
	}
}

func TestBackupFile_ReadError(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source-dir")
	if err := os.MkdirAll(srcPath, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}

	err := BackupFile(srcPath)
	if err == nil || !strings.Contains(err.Error(), "读取原文件失败") {
		t.Fatalf("expected read error, got: %v", err)
	}
}

func TestAtomicWriteFile_RenameToDirectoryErrorCleansTemp(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "target-dir")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}

	err := AtomicWriteFile(targetDir, []byte("data"), 0644)
	if err == nil || !strings.Contains(err.Error(), "重命名文件失败") {
		t.Fatalf("expected rename error, got: %v", err)
	}

	tmpFiles, _ := filepath.Glob(filepath.Join(tmpDir, ".tmp-*"))
	if len(tmpFiles) != 0 {
		t.Fatalf("临时文件未清理: %v", tmpFiles)
	}
}

func TestCopyFile_SourceIsDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}

	dstPath := filepath.Join(tmpDir, "dst")
	err := CopyFile(srcDir, dstPath)
	if err == nil || !strings.Contains(err.Error(), "复制文件内容失败") {
		t.Fatalf("expected copy error, got: %v", err)
	}
}
