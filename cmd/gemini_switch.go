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

注意: Gemini 配置仅更新内部选中状态，不会写入任何配置文件。
需要使用 'eval $(ccs gc)' 加载环境变量到当前 shell。

示例:
  ccs gemini switch mygemini   # 切换到 mygemini 配置
  eval $(ccs gc)               # 加载切换后的配置到环境变量`,
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
			return fmt.Errorf("配置不存在: %s", configName)
		}

		// 执行切换（仅更新 Current 字段，不写文件）
		if err := manager.SwitchProviderForApp("gemini", configName); err != nil {
			return fmt.Errorf("切换配置失败: %w", err)
		}

		// 提取配置信息显示
		baseURL, apiKey, model := config.ExtractGeminiConfigFromProvider(provider)

		fmt.Printf("✓ 已切换到 Gemini 配置: %s\n", configName)
		fmt.Printf("  Base URL: %s\n", baseURL)
		fmt.Printf("  API Key: %s\n", config.MaskToken(apiKey))
		if model != "" {
			fmt.Printf("  Model: %s\n", model)
		}

		fmt.Printf("\n运行以下命令加载环境变量:\n")
		fmt.Printf("  %s\n", config.GetEnvCommandExample())

		return nil
	},
}

func init() {
	geminiCmd.AddCommand(geminiSwitchCmd)
}
