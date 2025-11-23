package cmd

import (
	"fmt"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/spf13/cobra"
)

var geminiListCmd = &cobra.Command{
	Use:     "list",
	Short:   "列出所有 Gemini 配置",
	Aliases: []string{"ls"},
	Long: `列出所有 Gemini CLI 配置，显示编号、名称、认证类型和详细信息。

示例:
  ccs gemini list
  ccs g ls`,
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("初始化配置管理器失败: %w", err)
		}

		providers := manager.ListProvidersForApp("gemini")

		if len(providers) == 0 {
			fmt.Println("暂无 Gemini 配置，使用 'ccs gemini add <配置名>' 添加配置")
			return nil
		}

		fmt.Println("Gemini 配置列表:")
		fmt.Println("────────────────────────────────────────────────────────────────")

		currentProvider := manager.GetCurrentProviderForApp("gemini")

		for idx, p := range providers {
			// 编号从 1 开始
			number := idx + 1
			status := " "
			if currentProvider != nil && p.ID == currentProvider.ID {
				status = "●" // 当前选中
			}

			baseURL, apiKey, model, authType := config.ExtractGeminiConfigFromProvider(&p)

			// 格式: [编号] 状态 名称
			fmt.Printf("[%d] %s %-20s", number, status, p.Name)

			// 显示认证类型
			if authType == config.GeminiAuthOAuth {
				fmt.Printf(" [OAuth] ")
			} else {
				fmt.Printf(" [API Key] ")
			}

			// 显示详细信息
			if model != "" {
				fmt.Printf("Model: %-20s", model)
			}
			if baseURL != "" {
				fmt.Printf(" URL: %s", baseURL)
			}
			if authType == config.GeminiAuthAPIKey && apiKey != "" {
				fmt.Printf(" Key: %s", config.MaskToken(apiKey))
			}

			fmt.Println()
		}

		fmt.Println("────────────────────────────────────────────────────────────────")
		fmt.Printf("共 %d 个配置\n", len(providers))

		if currentProvider != nil {
			fmt.Printf("\n当前选中: %s\n", currentProvider.Name)
			fmt.Println("配置文件位置: ~/.gemini/.env 和 ~/.gemini/settings.json")
		} else {
			fmt.Println("\n未选中任何配置，使用 'ccs gemini switch <配置名>' 切换")
		}

		return nil
	},
}

func init() {
	geminiCmd.AddCommand(geminiListCmd)
}
