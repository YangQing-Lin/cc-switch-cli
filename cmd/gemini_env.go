package cmd

import (
	"fmt"
	"os"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/spf13/cobra"
)

var geminiEnvCmd = &cobra.Command{
	Use:   "env",
	Short: "查看 Gemini 配置文件路径和内容",
	Long: `查看 Gemini CLI 配置文件的路径和内容。

配置文件位置：
  ~/.gemini/.env           - 环境变量配置
  ~/.gemini/settings.json  - 认证模式和 MCP 服务器配置

示例:
  ccs gemini env         # 查看配置文件路径和内容
  ccs gc env             # 简写方式`,
	Aliases: []string{"show", "view"},
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("初始化配置管理器失败: %w", err)
		}

		// 获取配置文件路径
		envPath, err := manager.GetGeminiEnvPathWithDir()
		if err != nil {
			return fmt.Errorf("获取 .env 路径失败: %w", err)
		}

		settingsPath, err := manager.GetGeminiSettingsPathWithDir()
		if err != nil {
			return fmt.Errorf("获取 settings.json 路径失败: %w", err)
		}

		fmt.Println("Gemini CLI 配置文件:")
		fmt.Println("────────────────────────────────────────")

		// 显示 .env 文件
		fmt.Printf(".env 文件: %s\n", envPath)
		if _, err := os.Stat(envPath); os.IsNotExist(err) {
			fmt.Println("  状态: 不存在")
			fmt.Println("  提示: 使用 'ccs gemini switch <配置名>' 来创建配置文件")
		} else {
			content, err := os.ReadFile(envPath)
			if err != nil {
				fmt.Printf("  状态: 无法读取 (%v)\n", err)
			} else {
				fmt.Println("  内容:")
				if len(content) == 0 {
					fmt.Println("    (空文件)")
				} else {
					fmt.Println(string(content))
				}
			}
		}

		fmt.Println()

		// 显示 settings.json 文件
		fmt.Printf("settings.json 文件: %s\n", settingsPath)
		if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
			fmt.Println("  状态: 不存在")
		} else {
			content, err := os.ReadFile(settingsPath)
			if err != nil {
				fmt.Printf("  状态: 无法读取 (%v)\n", err)
			} else {
				fmt.Println("  内容:")
				if len(content) == 0 {
					fmt.Println("    (空文件)")
				} else {
					fmt.Println(string(content))
				}
			}
		}

		fmt.Println("────────────────────────────────────────")

		// 显示当前选中的配置
		currentProvider := manager.GetCurrentProviderForApp("gemini")
		if currentProvider != nil {
			fmt.Printf("\n当前选中配置: %s\n", currentProvider.Name)
		} else {
			fmt.Println("\n未选中任何配置")
		}

		return nil
	},
}

func init() {
	geminiCmd.AddCommand(geminiEnvCmd)

	// gemini 命令的默认行为：显示配置列表
	geminiCmd.RunE = func(cmd *cobra.Command, args []string) error {
		// 如果没有子命令，显示帮助信息
		return cmd.Help()
	}
}
