package portable

import (
	"os"
	"path/filepath"
)

var portableExecutableFunc = os.Executable

// IsPortableMode 检测是否为便携版模式
// 便携版模式：在程序所在目录下存在 portable.ini 文件
func IsPortableMode() bool {
	execPath, err := portableExecutableFunc()
	if err != nil {
		return false
	}

	execDir := filepath.Dir(execPath)
	portableFile := filepath.Join(execDir, "portable.ini")

	info, err := os.Stat(portableFile)
	if err != nil {
		return false
	}

	return !info.IsDir()
}

// GetPortableConfigDir 获取便携版配置目录
// 便携版模式下，配置目录为程序所在目录下的 .cc-switch 子目录
func GetPortableConfigDir() (string, error) {
	execPath, err := portableExecutableFunc()
	if err != nil {
		return "", err
	}

	execDir := filepath.Dir(execPath)
	configDir := filepath.Join(execDir, ".cc-switch")

	return configDir, nil
}
