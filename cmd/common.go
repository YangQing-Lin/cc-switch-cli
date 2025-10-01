package cmd

import (
	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
)

// getManager 获取配置管理器（考虑 --dir 参数）
func getManager() (*config.Manager, error) {
	if configDir != "" {
		return config.NewManagerWithDir(configDir)
	}
	return config.NewManager()
}
