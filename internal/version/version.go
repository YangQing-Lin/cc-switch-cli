package version

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CleanupOldUpdateDirs 清理 /tmp 目录下的历史更新临时目录
// 静默执行，失败不报错
func CleanupOldUpdateDirs() {
	tmpDir := os.TempDir()
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return // 静默失败
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		// 匹配 ccs-update-* 和 ccs-install-* 模式
		if strings.HasPrefix(name, "ccs-update-") || strings.HasPrefix(name, "ccs-install-") {
			fullPath := filepath.Join(tmpDir, name)
			_ = os.RemoveAll(fullPath) // 静默删除，忽略错误
		}
	}
}

// Version 当前版本
const Version = "1.8.5"

// BuildDate 构建日期（由编译时注入）
var BuildDate = "unknown"

// GitCommit Git 提交哈希（由编译时注入）
var GitCommit = "unknown"

// GetVersion 获取版本信息
func GetVersion() string { return Version }

// GetBuildDate 获取构建日期
func GetBuildDate() string { return BuildDate }

// GetGitCommit 获取 Git 提交哈希
func GetGitCommit() string { return GitCommit }
