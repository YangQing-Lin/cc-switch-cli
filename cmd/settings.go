package cmd

import (
	"fmt"
	"strings"

	"github.com/YangQing-Lin/cc-switch-cli/internal/settings"
	"github.com/spf13/cobra"
)

var (
	getSetting bool
	setSetting string
)

var settingsCmd = &cobra.Command{
	Use:   "settings [key]",
	Short: "管理应用设置",
	Long: `管理 cc-switch 应用设置

示例:
  cc-switch settings                     # 显示所有设置
  cc-switch settings --get language     # 获取语言设置
  cc-switch settings --set language=en  # 设置语言为英文
  cc-switch settings --set configDir=/custom/path  # 设置自定义配置目录`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSettings(args)
	},
}

func init() {
	rootCmd.AddCommand(settingsCmd)
	settingsCmd.Flags().BoolVar(&getSetting, "get", false, "获取指定设置项的值")
	settingsCmd.Flags().StringVar(&setSetting, "set", "", "设置项 (格式: key=value)")
}

func runSettings(args []string) error {
	manager, err := settings.NewManager()
	if err != nil {
		return err
	}

	// 设置模式
	if setSetting != "" {
		parts := strings.SplitN(setSetting, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("设置格式错误，应为: key=value")
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "language":
			if err := manager.SetLanguage(value); err != nil {
				return err
			}
			fmt.Printf("✓ 语言设置已更新为: %s\n", value)
		case "configDir":
			if err := manager.SetConfigDir(value); err != nil {
				return err
			}
			fmt.Printf("✓ 配置目录已更新为: %s\n", value)
		default:
			return fmt.Errorf("未知的设置项: %s (支持: language, configDir)", key)
		}
		return nil
	}

	// 获取模式
	if getSetting {
		if len(args) == 0 {
			return fmt.Errorf("请指定要获取的设置项名称")
		}

		key := args[0]
		switch key {
		case "language":
			fmt.Println(manager.GetLanguage())
		case "configDir":
			fmt.Println(manager.GetConfigDir())
		default:
			return fmt.Errorf("未知的设置项: %s (支持: language, configDir)", key)
		}
		return nil
	}

	// 显示所有设置
	s := manager.Get()
	fmt.Println("应用设置:")
	fmt.Printf("  语言 (language):    %s\n", s.Language)
	if s.ConfigDir != "" {
		fmt.Printf("  配置目录 (configDir): %s\n", s.ConfigDir)
	} else {
		fmt.Printf("  配置目录 (configDir): (使用默认)\n")
	}

	return nil
}
