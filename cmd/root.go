package cmd

import (
	"fmt"
	"os"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/i18n"
	"github.com/spf13/cobra"
)

var configDir string

var rootCmd = &cobra.Command{
	Use:   "cc-switch [配置名称]",
	Short: "Claude 中转站配置管理工具",
	Long: `cc-switch 是一个用于管理多个 Claude 中转站配置的命令行工具。

使用方法：
  cc-switch              列出所有配置
  cc-switch <配置名称>    切换到指定配置
  cc-switch config add   添加新配置
  cc-switch config delete 删除配置`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var manager *config.Manager
		var err error

		// 如果指定了自定义目录，使用自定义目录
		if configDir != "" {
			manager, err = config.NewManagerWithDir(configDir)
			if err != nil {
				return fmt.Errorf("初始化配置管理器失败 (自定义目录: %s): %w", configDir, err)
			}
			fmt.Printf("使用自定义配置目录: %s\n", configDir)
		} else {
			manager, err = config.NewManager()
			if err != nil {
				return fmt.Errorf("初始化配置管理器失败: %w", err)
			}
		}

		// 无参数：列出所有配置
		if len(args) == 0 {
			return listConfigs(manager)
		}

		// 单参数：切换配置
		configName := args[0]
		return switchConfig(manager, configName)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// 初始化 i18n
	_ = i18n.Init()

	// 添加全局 flag
	rootCmd.PersistentFlags().StringVar(&configDir, "dir", "", "使用自定义配置目录")

	// 自定义帮助模板
	rootCmd.SetHelpTemplate(`{{.Long}}

{{if .HasAvailableSubCommands}}可用命令:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}

{{if .HasAvailableLocalFlags}}选项:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}

使用 "{{.CommandPath}} [command] --help" 获取更多关于命令的信息。
`)
}

func listConfigs(manager *config.Manager) error {
	providers := manager.ListProviders()

	if len(providers) == 0 {
		fmt.Println("暂无配置，使用 'cc-switch config add' 添加配置")
		return nil
	}

	fmt.Println("配置列表:")
	fmt.Println("─────────")

	currentProvider := manager.GetCurrentProvider()

	for _, p := range providers {
		status := "○"
		if currentProvider != nil && p.ID == currentProvider.ID {
			status = "●"
		}

		token := config.ExtractTokenFromProvider(&p)
		baseURL := config.ExtractBaseURLFromProvider(&p)

		fmt.Printf("%s %-20s Token: %s  URL: %s",
			status,
			p.Name,
			config.MaskToken(token),
			baseURL)

		if p.Category != "" {
			fmt.Printf("  Category: %s", p.Category)
		}

		fmt.Println()
	}

	return nil
}

func switchConfig(manager *config.Manager, name string) error {
	provider, err := manager.GetProvider(name)
	if err != nil {
		return fmt.Errorf("配置不存在: %s", name)
	}

	if err := manager.SwitchProvider(name); err != nil {
		return fmt.Errorf("切换配置失败: %w", err)
	}

	token := config.ExtractTokenFromProvider(provider)
	baseURL := config.ExtractBaseURLFromProvider(provider)

	fmt.Printf("✓ 已切换到配置: %s\n", name)
	fmt.Printf("  Token: %s\n", config.MaskToken(token))
	fmt.Printf("  URL: %s\n", baseURL)

	return nil
}
