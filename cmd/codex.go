package cmd

import (
	"github.com/spf13/cobra"
)

var codexCmd = &cobra.Command{
	Use:   "codex",
	Short: "Codex CLI 配置管理",
	Long: `管理 Codex CLI 配置，包括添加、切换、删除等操作。

Codex 使用双配置文件：
  - config.yaml: 模型配置
  - api.json: API 认证信息

示例:
  cc-switch codex add mycodex --apikey sk-xxx --base-url https://api.example.com
  cc-switch codex switch mycodex
  cc-switch codex list`,
}

func init() {
	rootCmd.AddCommand(codexCmd)
}
