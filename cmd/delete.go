package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/spf13/cobra"
)

var force bool

var deleteCmd = &cobra.Command{
	Use:   "delete <配置名称>",
	Short: "删除指定的配置",
	Long:  "删除一个已存在的 Claude 中转站配置",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configName := args[0]

		manager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("初始化配置管理器失败: %w", err)
		}

		// 检查配置是否存在
		cfg, err := manager.GetConfig(configName)
		if err != nil {
			return fmt.Errorf("配置不存在: %s", configName)
		}

		// 显示配置详情
		fmt.Println("即将删除以下配置:")
		fmt.Printf("  名称: %s\n", cfg.Name)
		fmt.Printf("  Token: %s\n", config.MaskToken(cfg.AnthropicAuthToken))
		fmt.Printf("  URL: %s\n", cfg.AnthropicBaseURL)

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
		if err := manager.DeleteConfig(configName); err != nil {
			return fmt.Errorf("删除配置失败: %w", err)
		}

		fmt.Printf("✓ 配置 '%s' 已删除成功\n", configName)
		return nil
	},
}

func init() {
	deleteCmd.Flags().BoolVarP(&force, "force", "f", false, "强制删除，不需要确认")
}
