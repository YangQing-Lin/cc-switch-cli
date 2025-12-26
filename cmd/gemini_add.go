package cmd

import (
	"fmt"
	"strings"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/spf13/cobra"
)

var geminiAddCmd = &cobra.Command{
	Use:   "add <配置名称>",
	Short: "添加新的 Gemini 配置",
	Long: `添加一个新的 Gemini CLI 配置。

Gemini 支持两种认证模式：
  1. API Key 模式（默认）：适用于第三方供应商
  2. OAuth 模式：适用于 Google 官方 Gemini API

API Key 模式配置：
  - GOOGLE_GEMINI_BASE_URL: API 端点地址
  - GEMINI_API_KEY: API 认证密钥
  - GEMINI_MODEL: 使用的模型（可选，默认 gemini-2.5-pro）

OAuth 模式配置：
  - 无需 API Key，使用 Google OAuth 认证
  - 使用 --auth-type oauth 启用

示例:
  # API Key 模式
  cc-switch gemini add mygemini --apikey sk-xxx --base-url https://api.example.com

  # OAuth 模式（Google 官方）
  cc-switch gemini add google --auth-type oauth

  # 指定模型
  cc-switch gemini add mygemini --apikey sk-xxx --model gemini-2.5-flash`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configName := args[0]

		// 从 flags 获取参数
		authTypeStr, _ := cmd.Flags().GetString("auth-type")
		apiKey, _ := cmd.Flags().GetString("apikey")
		baseURL, _ := cmd.Flags().GetString("base-url")
		model, _ := cmd.Flags().GetString("model")

		// 确定认证类型
		authType := config.GeminiAuthAPIKey
		if authTypeStr == "oauth" || authTypeStr == "oauth-personal" {
			authType = config.GeminiAuthOAuth
		}

		// OAuth 模式下不需要 API Key
		if authType == config.GeminiAuthAPIKey {
			// 交互式输入缺失的参数
			if apiKey == "" {
				token, err := promptSecret("请输入 GEMINI_API_KEY: ")
				if err != nil {
					return fmt.Errorf("读取 API Key 失败: %w", err)
				}
				apiKey = token
			}

			if baseURL == "" {
				input, _ := promptInput("请输入 GOOGLE_GEMINI_BASE_URL (默认 https://generativelanguage.googleapis.com): ")
				if input != "" {
					baseURL = input
				} else {
					baseURL = "https://generativelanguage.googleapis.com"
				}
			}
		} else {
			// OAuth 模式：设置默认值
			if baseURL == "" {
				baseURL = "https://generativelanguage.googleapis.com"
			}
		}

		// 设置默认值
		if model == "" {
			model = "gemini-2.5-pro"
		}

		// 清理输入（不管认证模式都要 trim）
		apiKey = strings.TrimSpace(apiKey)
		baseURL = strings.TrimSpace(baseURL)
		model = strings.TrimSpace(model)

		// API Key 模式下验证必填字段
		if authType == config.GeminiAuthAPIKey {
			if apiKey == "" {
				return fmt.Errorf("API Key 模式下 GEMINI_API_KEY 不能为空")
			}
			if baseURL == "" {
				return fmt.Errorf("API Key 模式下 GOOGLE_GEMINI_BASE_URL 不能为空")
			}
		}

		// 创建管理器
		manager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("初始化配置管理器失败: %w", err)
		}

		// 添加配置
		if err := manager.AddGeminiProvider(configName, baseURL, apiKey, model, authType); err != nil {
			return fmt.Errorf("添加 Gemini 配置失败: %w", err)
		}

		fmt.Printf("✓ Gemini 配置 '%s' 已添加成功\n", configName)
		if authType == config.GeminiAuthOAuth {
			fmt.Printf("  认证类型: OAuth (Google 官方)\n")
		} else {
			fmt.Printf("  认证类型: API Key\n")
			fmt.Printf("  API Key: %s\n", config.MaskToken(apiKey))
		}
		fmt.Printf("  Base URL: %s\n", baseURL)
		fmt.Printf("  Model: %s\n", model)
		fmt.Printf("\n配置已自动写入 ~/.gemini/.env 和 ~/.gemini/settings.json\n")

		return nil
	},
}

func init() {
	geminiCmd.AddCommand(geminiAddCmd)

	geminiAddCmd.Flags().String("auth-type", "api-key", "认证类型 (api-key 或 oauth)")
	geminiAddCmd.Flags().String("apikey", "", "GEMINI_API_KEY (API Key 模式)")
	geminiAddCmd.Flags().String("base-url", "", "GOOGLE_GEMINI_BASE_URL")
	geminiAddCmd.Flags().String("model", "gemini-2.5-pro", "模型名称")
}
