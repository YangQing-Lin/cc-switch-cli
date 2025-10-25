package cmd

import (
	"fmt"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/spf13/cobra"
)

var geminiDeleteCmd = &cobra.Command{
	Use:   "delete <配置名称>",
	Short: "删除指定的 Gemini 配置",
	Long: `删除指定的 Gemini CLI 配置。

示例:
  ccs gemini delete mygemini`,
	Aliases: []string{"rm", "remove"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configName := args[0]

		manager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("初始化配置管理器失败: %w", err)
		}

		// 检查配置是否存在
		_, err = manager.GetProviderForApp("gemini", configName)
		if err != nil {
			return fmt.Errorf("配置不存在: %s", configName)
		}

		// 确认删除
		fmt.Printf("确定要删除 Gemini 配置 '%s' 吗? (y/N): ", configName)
		var confirm string
		fmt.Scanln(&confirm)

		if confirm != "y" && confirm != "Y" {
			fmt.Println("已取消删除")
			return nil
		}

		// 执行删除
		if err := manager.DeleteProviderForApp("gemini", configName); err != nil {
			return fmt.Errorf("删除配置失败: %w", err)
		}

		fmt.Printf("✓ Gemini 配置 '%s' 已删除\n", configName)

		return nil
	},
}

func init() {
	geminiCmd.AddCommand(geminiDeleteCmd)
}
