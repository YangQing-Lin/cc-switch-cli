package cmd

import (
	"fmt"
	"strings"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"syscall"
)

var codexUpdateCmd = &cobra.Command{
	Use:   "update <配置名称>",
	Short: "更新现有 Codex 配置",
	Long: `更新现有的 Codex CLI 配置。

示例:
  cc-switch codex update myconfig --name newname
  cc-switch codex update myconfig --apikey sk-new-key
  cc-switch codex update myconfig --base-url https://new-api.example.com
  cc-switch codex update myconfig --model claude-opus-4-20250514`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		oldName := args[0]

		// 从命令行参数获取要更新的字段
		newName, _ := cmd.Flags().GetString("name")
		apiKey, _ := cmd.Flags().GetString("apikey")
		baseURL, _ := cmd.Flags().GetString("base-url")
		model, _ := cmd.Flags().GetString("model")
		category, _ := cmd.Flags().GetString("category")

		// 创建管理器
		manager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("初始化配置管理器失败: %w", err)
		}

		// 获取现有配置
		provider, err := manager.GetProviderForApp("codex", oldName)
		if err != nil {
			return fmt.Errorf("获取配置失败: %w", err)
		}

		// 如果没有提供新名称，使用原名称
		if newName == "" {
			newName = provider.Name
		}

		// 提取当前配置
		var currentBaseURL, currentAPIKey, currentModel string
		if configMap, ok := provider.SettingsConfig["config"].(map[string]interface{}); ok {
			if url, ok := configMap["base_url"].(string); ok {
				currentBaseURL = url
			}
			if key, ok := configMap["api_key"].(string); ok {
				currentAPIKey = key
			}
			if m, ok := configMap["model_name"].(string); ok {
				currentModel = m
			}
		}

		// 如果没有提供新的 API Key，交互式输入或保留原值
		if apiKey == "" {
			if currentAPIKey != "" {
				fmt.Printf("当前 API Key: %s\n", config.MaskToken(currentAPIKey))
				fmt.Print("输入新的 API Key (按 Enter 保留当前值): ")
				bytePassword, err := term.ReadPassword(int(syscall.Stdin))
				fmt.Println() // 换行

				if err == nil && len(bytePassword) > 0 {
					apiKey = string(bytePassword)
				} else {
					apiKey = currentAPIKey
				}
			} else {
				fmt.Print("请输入 API Key: ")
				bytePassword, err := term.ReadPassword(int(syscall.Stdin))
				fmt.Println() // 换行

				if err != nil {
					return fmt.Errorf("读取输入失败: %w", err)
				}
				apiKey = string(bytePassword)
			}
		}

		// 如果没有提供新的 Base URL，使用原值
		if baseURL == "" {
			baseURL = currentBaseURL
			if baseURL == "" {
				baseURL = "https://api.anthropic.com"
			}
		}

		// 如果没有提供新的 Model，使用原值
		if model == "" {
			model = currentModel
			if model == "" {
				model = "claude-3-5-sonnet-20241022"
			}
		}

		// 如果没有提供新的 Category，使用原值
		if category == "" {
			category = provider.Category
			if category == "" {
				category = "custom"
			}
		}

		// 清理输入
		apiKey = strings.TrimSpace(apiKey)
		baseURL = strings.TrimSpace(baseURL)
		model = strings.TrimSpace(model)

		if apiKey == "" || baseURL == "" {
			return fmt.Errorf("API Key 和 Base URL 不能为空")
		}

		// Codex 需要特殊处理，因为它有 model 参数
		// 我们需要手动更新配置
		cfgData, err := manager.GetConfig()
		if err != nil {
			return fmt.Errorf("获取配置失败: %w", err)
		}

		app, exists := cfgData.Apps["codex"]
		if !exists {
			return fmt.Errorf("Codex 应用不存在")
		}

		// 找到目标 provider
		var targetID string
		for id, p := range app.Providers {
			if p.Name == oldName {
				targetID = id
				break
			}
		}

		if targetID == "" {
			return fmt.Errorf("配置 '%s' 不存在", oldName)
		}

		// 检查新名称冲突
		if newName != oldName {
			for _, p := range app.Providers {
				if p.Name == newName {
					return fmt.Errorf("配置名称 '%s' 已存在", newName)
				}
			}
		}

		// 更新 provider
		targetProvider := app.Providers[targetID]
		targetProvider.Name = newName
		targetProvider.Category = category

		// 更新 config 部分
		if targetProvider.SettingsConfig == nil {
			targetProvider.SettingsConfig = make(map[string]interface{})
		}
		targetProvider.SettingsConfig["config"] = map[string]interface{}{
			"base_url":   baseURL,
			"api_key":    apiKey,
			"model_name": model,
		}
		// 更新 api 部分
		targetProvider.SettingsConfig["api"] = map[string]interface{}{
			"baseURL": baseURL,
			"apiKey":  apiKey,
		}

		// 保存回 app
		app.Providers[targetID] = targetProvider
		cfgData.Apps["codex"] = app

		// 如果是当前激活配置，立即应用
		if app.Current == targetID {
			// 先保存配置
			if err := manager.Save(); err != nil {
				return fmt.Errorf("保存配置失败: %w", err)
			}
			// 再切换以应用到文件系统
			if err := manager.SwitchProviderForApp("codex", newName); err != nil {
				return fmt.Errorf("应用配置失败: %w", err)
			}
		} else {
			// 只保存配置
			if err := manager.Save(); err != nil {
				return fmt.Errorf("保存配置失败: %w", err)
			}
		}

		fmt.Printf("✓ Codex 配置 '%s' 已更新成功\n", oldName)
		if oldName != newName {
			fmt.Printf("  新名称: %s\n", newName)
		}

		return nil
	},
}

func init() {
	codexCmd.AddCommand(codexUpdateCmd)

	codexUpdateCmd.Flags().String("name", "", "新的配置名称")
	codexUpdateCmd.Flags().String("apikey", "", "新的 API Key")
	codexUpdateCmd.Flags().String("base-url", "", "新的 Base URL")
	codexUpdateCmd.Flags().String("model", "", "新的模型名称")
	codexUpdateCmd.Flags().String("category", "", "新的分类")
}
