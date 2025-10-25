package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var geminiAddCmd = &cobra.Command{
	Use:   "add <配置名称>",
	Short: "添加新的 Gemini 配置",
	Long: `添加一个新的 Gemini CLI 配置。

Gemini 配置包括：
  - GOOGLE_GEMINI_BASE_URL: API 端点地址
  - GEMINI_API_KEY: API 认证密钥
  - GEMINI_MODEL: 使用的模型（可选，默认 gemini-2.5-pro）

示例:
  cc-switch gemini add mygemini --apikey sk-xxx --base-url https://generativelanguage.googleapis.com
  cc-switch gemini add google --apikey sk-xxx --model gemini-2.5-flash`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configName := args[0]

		// 从 flags 获取参数
		apiKey, _ := cmd.Flags().GetString("apikey")
		baseURL, _ := cmd.Flags().GetString("base-url")
		model, _ := cmd.Flags().GetString("model")
		category, _ := cmd.Flags().GetString("category")
		websiteURL, _ := cmd.Flags().GetString("website")

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

		if model == "" && !cmd.Flags().Changed("model") {
			input, _ := promptInput("请输入模型名称 (默认 gemini-2.5-pro): ")
			if input != "" {
				model = input
			} else {
				model = "gemini-2.5-pro"
			}
		}

		// 设置默认值
		if category == "" {
			category = "custom"
		}
		if model == "" {
			model = "gemini-2.5-pro"
		}

		// 验证输入
		apiKey = strings.TrimSpace(apiKey)
		baseURL = strings.TrimSpace(baseURL)
		model = strings.TrimSpace(model)

		if apiKey == "" {
			return fmt.Errorf("GEMINI_API_KEY 不能为空")
		}
		if baseURL == "" {
			return fmt.Errorf("GOOGLE_GEMINI_BASE_URL 不能为空")
		}

		// 创建管理器
		manager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("初始化配置管理器失败: %w", err)
		}

		// 构建 Gemini 配置
		provider := config.Provider{
			ID:   uuid.New().String(),
			Name: configName,
			SettingsConfig: map[string]interface{}{
				"env": map[string]interface{}{
					"GOOGLE_GEMINI_BASE_URL": baseURL,
					"GEMINI_API_KEY":         apiKey,
					"GEMINI_MODEL":           model,
				},
			},
			WebsiteURL: websiteURL,
			Category:   category,
			CreatedAt:  time.Now().UnixMilli(),
		}

		// 添加配置
		if err := manager.AddProviderDirect("gemini", provider); err != nil {
			return fmt.Errorf("添加 Gemini 配置失败: %w", err)
		}

		fmt.Printf("✓ Gemini 配置 '%s' 已添加成功\n", configName)
		fmt.Printf("  Base URL: %s\n", baseURL)
		fmt.Printf("  Model: %s\n", model)
		fmt.Printf("  API Key: %s\n", config.MaskToken(apiKey))
		fmt.Printf("\n使用 'eval $(ccs gc %s)' 加载此配置到环境变量\n", configName)

		return nil
	},
}

func init() {
	geminiCmd.AddCommand(geminiAddCmd)

	geminiAddCmd.Flags().String("apikey", "", "GEMINI_API_KEY")
	geminiAddCmd.Flags().String("base-url", "", "GOOGLE_GEMINI_BASE_URL")
	geminiAddCmd.Flags().String("model", "gemini-2.5-pro", "模型名称")
	geminiAddCmd.Flags().String("category", "custom", "分类")
	geminiAddCmd.Flags().String("website", "", "供应商网站 (可选)")
}
