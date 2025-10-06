package cmd

import (
	"fmt"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/spf13/cobra"
)

var codexSwitchCmd = &cobra.Command{
	Use:   "switch <配置名称>",
	Short: "切换到指定的 Codex 配置",
	Long: `切换到指定的 Codex CLI 配置，会同时更新 config.yaml 和 api.json 文件。

使用 SSOT (Single Source of Truth) 模式：
  1. 回填：保存当前配置到内存
  2. 切换：写入目标配置到文件系统
  3. 持久化：更新配置文件`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configName := args[0]

		manager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("初始化配置管理器失败: %w", err)
		}

		// 检查配置是否存在
		provider, err := manager.GetProviderForApp("codex", configName)
		if err != nil {
			return fmt.Errorf("配置不存在: %s", configName)
		}

		// 执行切换
		if err := manager.SwitchProviderForApp("codex", configName); err != nil {
			return fmt.Errorf("切换配置失败: %w", err)
		}

		// 提取配置信息显示
		var baseURL, apiKey, model string
		if configMap, ok := provider.SettingsConfig["config"].(map[string]interface{}); ok {
			if url, ok := configMap["base_url"].(string); ok {
				baseURL = url
			}
			if key, ok := configMap["api_key"].(string); ok {
				apiKey = key
			}
			if m, ok := configMap["model_name"].(string); ok {
				model = m
			}
		}

		fmt.Printf("✓ 已切换到 Codex 配置: %s\n", configName)
		fmt.Printf("  Base URL: %s\n", baseURL)
		fmt.Printf("  API Key: %s\n", config.MaskToken(apiKey))
		if model != "" {
			fmt.Printf("  Model: %s\n", model)
		}

		// 显示配置文件位置
		configPath, _ := config.GetCodexConfigPath()
		apiPath, _ := config.GetCodexApiJsonPath()
		fmt.Printf("\n已更新文件:\n")
		fmt.Printf("  - %s\n", configPath)
		fmt.Printf("  - %s\n", apiPath)

		return nil
	},
}

func init() {
	codexCmd.AddCommand(codexSwitchCmd)
}
