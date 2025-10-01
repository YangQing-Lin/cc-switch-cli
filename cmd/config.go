package cmd

import (
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "配置管理子命令",
	Long:  "管理 Claude 中转站配置，包括添加、删除等操作",
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(addCmd)
	configCmd.AddCommand(deleteCmd)
}
