package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/spf13/cobra"
)

// showCmd represents the show command
var showCmd = &cobra.Command{
	Use:   "show [配置名称]",
	Short: "显示供应商配置的详细信息",
	Long: `显示指定供应商配置的详细信息。

示例:
  cc-switch config show myconfig
  cc-switch config show myconfig --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		jsonFormat, _ := cmd.Flags().GetBool("json")
		appName, _ := cmd.Flags().GetString("app")

		// 创建管理器
		manager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("初始化配置管理器失败: %w", err)
		}

		// 获取配置
		provider, err := manager.GetProviderForApp(appName, name)
		if err != nil {
			return err
		}

		// 获取当前激活的配置
		currentProvider := manager.GetCurrentProviderForApp(appName)
		isCurrent := currentProvider != nil && currentProvider.ID == provider.ID

		if jsonFormat {
			// JSON 格式输出
			output := map[string]interface{}{
				"id":             provider.ID,
				"name":           provider.Name,
				"category":       provider.Category,
				"websiteUrl":     provider.WebsiteURL,
				"settingsConfig": provider.SettingsConfig,
				"createdAt":      provider.CreatedAt,
				"isCurrent":      isCurrent,
			}
			data, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				return fmt.Errorf("序列化失败: %w", err)
			}
			fmt.Println(string(data))
		} else {
			// 人类可读格式输出
			fmt.Printf("配置详情: %s\n", provider.Name)
			fmt.Println("─────────────────────────────")

			if isCurrent {
				fmt.Println("状态:     ● 当前激活")
			} else {
				fmt.Println("状态:     ○ 未激活")
			}

			fmt.Printf("ID:       %s\n", provider.ID)
			fmt.Printf("名称:     %s\n", provider.Name)

			// 提取并显示配置信息
			switch appName {
			case "claude":
				token := config.ExtractTokenFromProvider(provider)
				baseURL := config.ExtractBaseURLFromProvider(provider)
				fmt.Printf("API Token: %s\n", config.MaskToken(token))
				fmt.Printf("Base URL:  %s\n", baseURL)

				// 显示额外的环境变量（如果有）
				if envMap, ok := provider.SettingsConfig["env"].(map[string]interface{}); ok {
					if model, ok := envMap["ANTHROPIC_MODEL"].(string); ok && model != "" {
						fmt.Printf("主模型:    %s\n", model)
					}
					if model, ok := envMap["ANTHROPIC_DEFAULT_HAIKU_MODEL"].(string); ok && model != "" {
						fmt.Printf("Haiku 默认模型: %s\n", model)
					}
					if model, ok := envMap["ANTHROPIC_DEFAULT_SONNET_MODEL"].(string); ok && model != "" {
						fmt.Printf("Sonnet 默认模型: %s\n", model)
					}
					if model, ok := envMap["ANTHROPIC_DEFAULT_OPUS_MODEL"].(string); ok && model != "" {
						fmt.Printf("Opus 默认模型: %s\n", model)
					}
					if model, ok := envMap["CLAUDE_CODE_MODEL"].(string); ok && model != "" {
						fmt.Printf("Model:     %s\n", model)
					}
					if maxTokens, ok := envMap["CLAUDE_CODE_MAX_TOKENS"].(string); ok && maxTokens != "" {
						fmt.Printf("Max Tokens: %s\n", maxTokens)
					}
				}
			case "codex":
				if auth, ok := provider.SettingsConfig["auth"].(string); ok {
					fmt.Printf("Auth:      %s\n", config.MaskToken(auth))
				}
				if baseURL, ok := provider.SettingsConfig["base_url"].(string); ok {
					fmt.Printf("Base URL:  %s\n", baseURL)
				}
			}

			if provider.Category != "" {
				fmt.Printf("分类:     %s\n", provider.Category)
			}

			if provider.WebsiteURL != "" {
				fmt.Printf("网站:     %s\n", provider.WebsiteURL)
			}

			if provider.CreatedAt > 0 {
				createdTime := time.Unix(0, provider.CreatedAt*int64(time.Millisecond))
				fmt.Printf("创建时间: %s\n", createdTime.Format("2006-01-02 15:04:05"))
			}

			// 显示原始配置（调试用）
			if verbose, _ := cmd.Flags().GetBool("verbose"); verbose {
				fmt.Println("\n原始配置:")
				data, _ := json.MarshalIndent(provider.SettingsConfig, "  ", "  ")
				fmt.Println("  " + string(data))
			}
		}

		return nil
	},
}

func init() {
	configCmd.AddCommand(showCmd)

	showCmd.Flags().Bool("json", false, "以 JSON 格式输出")
	showCmd.Flags().Bool("verbose", false, "显示详细信息")
	showCmd.Flags().String("app", "claude", "应用名称 (claude 或 codex)")
}
