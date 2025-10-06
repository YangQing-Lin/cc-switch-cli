package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/spf13/cobra"
)

var codexDeleteCmd = &cobra.Command{
	Use:   "delete <配置名称>",
	Short: "删除指定的 Codex 配置",
	Long:  "删除一个已存在的 Codex CLI 配置",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configName := args[0]
		force, _ := cmd.Flags().GetBool("force")

		manager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("初始化配置管理器失败: %w", err)
		}

		// 检查配置是否存在
		provider, err := manager.GetProviderForApp("codex", configName)
		if err != nil {
			return fmt.Errorf("配置不存在: %s", configName)
		}

		// 提取配置信息
		var baseURL, apiKey string
		if configMap, ok := provider.SettingsConfig["config"].(map[string]interface{}); ok {
			if url, ok := configMap["base_url"].(string); ok {
				baseURL = url
			}
			if key, ok := configMap["api_key"].(string); ok {
				apiKey = key
			}
		}

		// 显示配置详情
		fmt.Println("即将删除以下 Codex 配置:")
		fmt.Printf("  名称: %s\n", provider.Name)
		fmt.Printf("  API Key: %s\n", config.MaskToken(apiKey))
		fmt.Printf("  Base URL: %s\n", baseURL)

		// 确认删除
		if !force {
			fmt.Print("\n确定要删除这个配置吗? (y/N): ")
			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("读取输入失败: %w", err)
			}

			response = strings.ToLower(strings.TrimSpace(response))
			if response != "y" && response != "yes" {
				fmt.Println("取消删除")
				return nil
			}
		}

		// 执行删除
		if err := manager.DeleteProviderForApp("codex", configName); err != nil {
			return fmt.Errorf("删除配置失败: %w", err)
		}

		fmt.Printf("✓ Codex 配置 '%s' 已删除成功\n", configName)
		return nil
	},
}

func init() {
	codexCmd.AddCommand(codexDeleteCmd)
	codexDeleteCmd.Flags().BoolP("force", "f", false, "强制删除，不需要确认")
}
