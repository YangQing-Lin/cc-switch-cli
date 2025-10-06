package cmd

import (
	"fmt"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/spf13/cobra"
)

var codexListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有 Codex 配置",
	Long:  "显示所有已保存的 Codex CLI 配置列表",
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("初始化配置管理器失败: %w", err)
		}

		providers := manager.ListProvidersForApp("codex")

		if len(providers) == 0 {
			fmt.Println("暂无 Codex 配置，使用 'cc-switch codex add' 添加配置")
			return nil
		}

		fmt.Println("Codex 配置列表:")
		fmt.Println("─────────────────────────────")

		currentProvider := manager.GetCurrentProviderForApp("codex")

		for _, p := range providers {
			status := "○"
			if currentProvider != nil && p.ID == currentProvider.ID {
				status = "●"
			}

			// 从 config 部分提取信息
			var baseURL, apiKey, model string
			if configMap, ok := p.SettingsConfig["config"].(map[string]interface{}); ok {
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

			fmt.Printf("%s %-20s\n", status, p.Name)
			fmt.Printf("   API Key: %s\n", config.MaskToken(apiKey))
			fmt.Printf("   Base URL: %s\n", baseURL)
			if model != "" {
				fmt.Printf("   Model: %s\n", model)
			}
			if p.Category != "" {
				fmt.Printf("   Category: %s\n", p.Category)
			}
			fmt.Println()
		}

		return nil
	},
}

func init() {
	codexCmd.AddCommand(codexListCmd)
}
