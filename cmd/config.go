package cmd

import (
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:     "config",
	Aliases: []string{"cfg"},
	Short:   "配置管理子命令",
	Long: `管理 Claude 中转站配置，包括添加、删除等操作

简化用法:
  ccs cfg add <name> ...       # 添加配置
  ccs cfg delete <name>        # 删除配置`,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(addCmd)
	configCmd.AddCommand(deleteCmd)
}
