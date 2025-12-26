package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var codexAddCmd = &cobra.Command{
	Use:   "add <配置名称>",
	Short: "添加新的 Codex 配置",
	Long: `添加一个新的 Codex CLI 配置。

Codex 配置包括：
  - API Key: 用于认证
  - Base URL: API 端点地址
  - Model: 使用的模型（可选，默认 claude-3-5-sonnet-20241022）

示例:
  cc-switch codex add mycodex --apikey sk-xxx --base-url https://api.anthropic.com
  cc-switch codex add openrouter --apikey sk-or-xxx --base-url https://openrouter.ai/api/v1`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configName := args[0]

		// 从 flags 获取参数
		apiKey, _ := cmd.Flags().GetString("apikey")
		baseURL, _ := cmd.Flags().GetString("base-url")
		modelName, _ := cmd.Flags().GetString("model")
		category, _ := cmd.Flags().GetString("category")
		websiteURL, _ := cmd.Flags().GetString("website")

		// 交互式输入缺失的参数
		if apiKey == "" {
			token, err := promptSecret("请输入 API Key: ")
			if err != nil {
				return fmt.Errorf("读取 API Key 失败: %w", err)
			}
			apiKey = token
		}

		if baseURL == "" {
			input, _ := promptInput("请输入 Base URL (默认 https://api.anthropic.com): ")
			if input != "" {
				baseURL = input
			} else {
				baseURL = "https://api.anthropic.com"
			}
		}

		// 设置默认值
		if category == "" {
			category = "custom"
		}
		if modelName == "" {
			modelName = "claude-3-5-sonnet-20241022"
		}

		// 验证输入
		apiKey = strings.TrimSpace(apiKey)
		baseURL = strings.TrimSpace(baseURL)

		if apiKey == "" {
			return fmt.Errorf("API Key 不能为空")
		}
		if baseURL == "" {
			return fmt.Errorf("Base URL 不能为空")
		}

		// 创建管理器
		manager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("初始化配置管理器失败: %w", err)
		}

		// 构建 Codex 配置
		// 生成 TOML 格式的 config.toml 内容
		configToml := fmt.Sprintf(`model_provider = "custom"
model = "%s"

[model_providers.custom]
name = "custom"
base_url = "%s"
wire_api = "responses"`, modelName, baseURL)

		provider := config.Provider{
			ID:   uuid.New().String(),
			Name: configName,
			SettingsConfig: map[string]interface{}{
				"auth": map[string]interface{}{
					"OPENAI_API_KEY": apiKey,
				},
				"config": configToml,
			},
			WebsiteURL: websiteURL,
			Category:   category,
			CreatedAt:  time.Now().UnixMilli(),
		}

		// 添加配置
		if err := manager.AddProviderDirect("codex", provider); err != nil {
			return fmt.Errorf("添加 Codex 配置失败: %w", err)
		}

		fmt.Printf("✓ Codex 配置 '%s' 已添加成功\n", configName)
		fmt.Printf("  Base URL: %s\n", baseURL)
		fmt.Printf("  Model: %s\n", modelName)
		fmt.Printf("  API Key: %s\n", config.MaskToken(apiKey))

		return nil
	},
}

func init() {
	codexCmd.AddCommand(codexAddCmd)

	codexAddCmd.Flags().String("apikey", "", "API Key")
	codexAddCmd.Flags().String("base-url", "", "Base URL")
	codexAddCmd.Flags().String("model", "claude-3-5-sonnet-20241022", "模型名称")
	codexAddCmd.Flags().String("category", "custom", "分类")
	codexAddCmd.Flags().String("website", "", "供应商网站 (可选)")
}
