package cmd

import (
	"fmt"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/spf13/cobra"
)

var geminiSwitchCmd = &cobra.Command{
	Use:   "switch <配置名称>",
	Short: "切换到指定的 Gemini 配置",
	Long: `切换到指定的 Gemini CLI 配置。

配置将自动写入 ~/.gemini/.env 和 ~/.gemini/settings.json 文件。
Gemini CLI 会自动检测配置变化，无需重启或手动加载环境变量。

示例:
  ccs gemini switch mygemini   # 切换到 mygemini 配置
  ccs gemini switch google     # 切换到 google 配置（OAuth 模式）`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configName := args[0]

		manager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("初始化配置管理器失败: %w", err)
		}

		// 检查配置是否存在
		provider, err := manager.GetProviderForApp("gemini", configName)
		if err != nil {
			return fmt.Errorf("获取配置失败: %w", err)
		}

		// 执行切换（写入 .env 和 settings.json）
		if err := manager.SwitchProviderForApp("gemini", configName); err != nil {
			return fmt.Errorf("切换配置失败: %w", err)
		}

		// 提取配置信息显示
		baseURL, apiKey, model, authType := config.ExtractGeminiConfigFromProvider(provider)

		fmt.Printf("✓ 已切换到 Gemini 配置: %s\n", configName)
		if authType == config.GeminiAuthOAuth {
			fmt.Printf("  认证类型: OAuth (Google 官方)\n")
		} else {
			fmt.Printf("  认证类型: API Key\n")
			fmt.Printf("  API Key: %s\n", config.MaskToken(apiKey))
		}
		fmt.Printf("  Base URL: %s\n", baseURL)
		if model != "" {
			fmt.Printf("  Model: %s\n", model)
		}

		fmt.Printf("\n配置已写入:\n")
		fmt.Printf("  ~/.gemini/.env\n")
		fmt.Printf("  ~/.gemini/settings.json\n")
		fmt.Printf("\nGemini CLI 会自动检测配置变化，无需重启\n")

		return nil
	},
}

func init() {
	geminiCmd.AddCommand(geminiSwitchCmd)
}
