package cmd

import (
	"github.com/spf13/cobra"
)

var codexCmd = &cobra.Command{
	Use:     "codex",
	Aliases: []string{"cx", "c"},
	Short:   "Codex CLI 配置管理",
	Long: `管理 Codex CLI 配置，包括添加、切换、删除等操作。

Codex 使用双配置文件：
  - config.yaml: 模型配置
  - api.json: API 认证信息

示例:
  cc-switch codex add mycodex --apikey sk-xxx --base-url https://api.example.com
  cc-switch codex switch mycodex
  cc-switch codex list

简化用法:
  ccs cx add <name> ...        # 添加配置（推荐）
  ccs cx switch <name>         # 切换配置（推荐）
  ccs cx list                  # 列出配置（推荐）
  ccs c add <name> ...         # 最短形式`,
}

func init() {
	rootCmd.AddCommand(codexCmd)
}
