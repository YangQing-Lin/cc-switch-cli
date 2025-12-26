package cmd

import (
	"fmt"
	"syscall"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update [配置名称]",
	Short: "更新现有供应商配置",
	Long: `更新现有的供应商配置。

示例:
  cc-switch config update myconfig --name newname
  cc-switch config update myconfig --apikey sk-new-token
  cc-switch config update myconfig --base-url https://new-api.example.com`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		oldName := args[0]

		// 从命令行参数获取要更新的字段
		newName, _ := cmd.Flags().GetString("name")
		apiKey, _ := cmd.Flags().GetString("apikey")
		baseURL, _ := cmd.Flags().GetString("base-url")
		category, _ := cmd.Flags().GetString("category")
		appName, _ := cmd.Flags().GetString("app")
		defaultSonnetModel, _ := cmd.Flags().GetString("default-sonnet-model")

		// 创建管理器
		manager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("初始化配置管理器失败: %w", err)
		}

		// 获取现有配置
		provider, err := manager.GetProviderForApp(appName, oldName)
		if err != nil {
			return fmt.Errorf("获取配置失败: %w", err)
		}

		// 如果没有提供新名称，使用原名称
		if newName == "" {
			newName = provider.Name
		}

		// 如果没有提供新的 API Key，使用原值或交互式输入
		if apiKey == "" {
			currentToken := config.ExtractTokenFromProvider(provider)
			if currentToken != "" {
				fmt.Printf("当前 API Token: %s\n", config.MaskToken(currentToken))
				fmt.Print("输入新的 API Token (按 Enter 保留当前值): ")
				bytePassword, err := cmdReadPasswordFunc(int(syscall.Stdin))
				fmt.Println() // 换行

				if err == nil && len(bytePassword) > 0 {
					apiKey = string(bytePassword)
				} else {
					apiKey = currentToken
				}
			} else {
				fmt.Print("请输入 API Token: ")
				bytePassword, err := cmdReadPasswordFunc(int(syscall.Stdin))
				fmt.Println() // 换行

				if err != nil {
					return fmt.Errorf("读取输入失败: %w", err)
				}
				apiKey = string(bytePassword)
			}
		}

		// 如果没有提供新的 Base URL，使用原值
		if baseURL == "" {
			baseURL = config.ExtractBaseURLFromProvider(provider)
			if baseURL == "" {
				baseURL = "https://api.anthropic.com"
			}
		}

		// 如果没有提供新的 Category，使用原值
		if category == "" {
			category = provider.Category
			if category == "" {
				category = "custom"
			}
		}

		// 验证输入
		if err := config.ValidateProvider(newName, apiKey, baseURL); err != nil {
			return err
		}

		// 更新配置
		if err := manager.UpdateProviderForApp(appName, oldName, newName, "", apiKey, baseURL, category, "", "", defaultSonnetModel, ""); err != nil {
			return err
		}

		fmt.Printf("✓ 配置 '%s' 已更新成功\n", oldName)
		if oldName != newName {
			fmt.Printf("  新名称: %s\n", newName)
		}

		return nil
	},
}

func init() {
	configCmd.AddCommand(updateCmd)

	updateCmd.Flags().String("name", "", "新的配置名称")
	updateCmd.Flags().String("apikey", "", "新的 API Token")
	updateCmd.Flags().String("base-url", "", "新的 Base URL")
	updateCmd.Flags().String("category", "", "新的分类")
	updateCmd.Flags().String("app", "claude", "应用名称 (claude 或 codex)")
	updateCmd.Flags().String("default-sonnet-model", "", "Default Sonnet model (optional, for Claude only)")
}
