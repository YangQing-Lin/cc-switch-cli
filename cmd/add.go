package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	apiKey   string
	baseURL  string
	category string
	appName  string
)

var addCmd = &cobra.Command{
	Use:   "add <配置名称>",
	Short: "添加新的配置",
	Long:  "添加一个新的 Claude 或 Codex 配置，可以通过命令行参数或交互式输入提供配置信息",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configName := args[0]

		// 如果没有通过参数提供，则交互式输入
		if apiKey == "" {
			token, err := promptSecret("请输入 API Token: ")
			if err != nil {
				return fmt.Errorf("读取 API Token 失败: %w", err)
			}
			apiKey = token
		}

		if baseURL == "" {
			baseURL, _ = promptInput("请输入 Base URL: ")
		}

		if category == "" && !cmd.Flags().Changed("category") {
			category, _ = promptInput("请输入 Category (可选，默认 custom): ")
		}

		// 设置默认 category
		if category == "" {
			category = "custom"
		}

		// 验证配置
		if err := config.ValidateProvider(configName, strings.TrimSpace(apiKey), strings.TrimSpace(baseURL)); err != nil {
			return err
		}

		// 添加到管理器
		manager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("初始化配置管理器失败: %w", err)
		}

		// 如果没有指定 app，默认为 claude
		if appName == "" {
			appName = "claude"
		}

		if err := manager.AddProviderForApp(
			appName,
			configName,
			"", // websiteURL - 命令行暂不支持
			strings.TrimSpace(apiKey),
			strings.TrimSpace(baseURL),
			category,
		); err != nil {
			return fmt.Errorf("添加配置失败: %w", err)
		}

		fmt.Printf("✓ 配置 '%s' 已添加到 %s 成功\n", configName, appName)
		return nil
	},
}

func init() {
	addCmd.Flags().StringVar(&apiKey, "apikey", "", "API Token")
	addCmd.Flags().StringVar(&baseURL, "base-url", "", "Base URL")
	addCmd.Flags().StringVar(&category, "category", "custom", "Provider category (official/cn_official/aggregator/third_party/custom)")
	addCmd.Flags().StringVar(&appName, "app", "claude", "Application (claude/codex)")
}

// promptSecret 提示用户输入敏感信息（隐藏输入）
func promptSecret(prompt string) (string, error) {
	fmt.Print(prompt)

	// 使用 term.ReadPassword 隐藏输入
	fd := int(os.Stdin.Fd())
	bytePassword, err := term.ReadPassword(fd)
	if err != nil {
		// 降级处理：如果隐藏输入失败，使用明文输入
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(input), nil
	}

	fmt.Println()
	return strings.TrimSpace(string(bytePassword)), nil
}

// promptInput 提示用户输入普通信息
func promptInput(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

// 为 Windows 兼容性定义 syscall.Stdin
func init() {
	if syscall.Stdin == 0 {
		// Windows 特殊处理（如果需要）
	}
}
