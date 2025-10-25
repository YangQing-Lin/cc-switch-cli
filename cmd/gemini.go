package cmd

import (
	"github.com/spf13/cobra"
)

var geminiCmd = &cobra.Command{
	Use:     "gemini",
	Aliases: []string{"gc", "g"},
	Short:   "Gemini CLI 配置管理",
	Long: `管理 Gemini CLI 配置，包括添加、切换、删除等操作。

Gemini 使用环境变量配置：
  - GOOGLE_GEMINI_BASE_URL: API 端点地址
  - GEMINI_API_KEY: API 认证密钥
  - GEMINI_MODEL: 使用的模型

示例:
  cc-switch gemini add mygemini --apikey sk-xxx --base-url https://generativelanguage.googleapis.com
  cc-switch gc                  # 输出当前选中配置的 export 语句
  cc-switch gc 3                # 输出编号为 3 的配置的 export 语句
  eval $(cc-switch gc)          # 加载环境变量到当前 shell

简化用法:
  ccs gc                        # 输出当前配置的 export 语句（推荐）
  ccs gc 2                      # 输出编号为 2 的配置
  ccs g add <name> ...          # 添加配置
  ccs g list                    # 列出所有配置及编号`,
}

func init() {
	rootCmd.AddCommand(geminiCmd)
}
