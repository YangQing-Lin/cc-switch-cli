package cmd

import (
	"os"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
)

var exitFunc = os.Exit

// getManager 获取配置管理器（考虑 --dir 参数）
func getManager() (*config.Manager, error) {
	if configDir != "" {
		return config.NewManagerWithDir(configDir)
	}
	return config.NewManager()
}
