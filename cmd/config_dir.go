package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	setConfigDir string
)

var configDirCmd = &cobra.Command{
	Use:   "config-dir",
	Short: "管理配置目录",
	Long:  `显示当前配置目录路径，或设置自定义配置目录`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigDir()
	},
}

func init() {
	rootCmd.AddCommand(configDirCmd)
	configDirCmd.Flags().StringVar(&setConfigDir, "set", "", "设置自定义配置目录路径")
}

func runConfigDir() error {
	manager, err := getManager()
	if err != nil {
		return err
	}

	configPath := manager.GetConfigPath()
	configDir := filepath.Dir(configPath)

	// 如果是设置目录
	if setConfigDir != "" {
		fmt.Printf("⚠ 注意: 设置自定义配置目录需要使用 --dir 全局参数\n\n")
		fmt.Printf("示例:\n")
		fmt.Printf("  cc-switch --dir \"%s\" list\n", setConfigDir)
		fmt.Printf("  cc-switch --dir \"%s\" switch <provider>\n\n", setConfigDir)
		fmt.Printf("或者设置环境变量:\n")
		fmt.Printf("  export CC_SWITCH_DIR=\"%s\"\n", setConfigDir)
		return nil
	}

	// 显示当前配置目录
	fmt.Printf("配置目录: %s\n", configDir)
	fmt.Printf("配置文件: %s\n", configPath)

	return nil
}
