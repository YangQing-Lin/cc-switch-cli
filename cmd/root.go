package cmd

import (
	"fmt"
	"os"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cc-switch [配置名称]",
	Short: "Claude 中转站配置管理工具",
	Long: `cc-switch 是一个用于管理多个 Claude 中转站配置的命令行工具。

使用方法：
  cc-switch              列出所有配置
  cc-switch <配置名称>    切换到指定配置
  cc-switch config add   添加新配置
  cc-switch config delete 删除配置`,
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("初始化配置管理器失败: %w", err)
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
	configs := manager.ListConfigs()

	if len(configs) == 0 {
		fmt.Println("暂无配置，使用 'cc-switch config add' 添加配置")
		return nil
	}

	fmt.Println("配置列表:")
	fmt.Println("─────────")

	currentConfig := manager.GetCurrentConfig()

	for _, cfg := range configs {
		status := "○"
		if currentConfig != nil && cfg.Name == currentConfig.Name {
			status = "●"
		}

		fmt.Printf("%s %-20s Token: %s  URL: %s",
			status,
			cfg.Name,
			config.MaskToken(cfg.AnthropicAuthToken),
			cfg.AnthropicBaseURL)

		if cfg.ClaudeCodeModel != "" {
			fmt.Printf("  Model: %s", cfg.ClaudeCodeModel)
		}

		fmt.Println()
	}

	return nil
}

func switchConfig(manager *config.Manager, name string) error {
	cfg, err := manager.GetConfig(name)
	if err != nil {
		return fmt.Errorf("配置不存在: %s", name)
	}

	if err := manager.SwitchConfig(name); err != nil {
		return fmt.Errorf("切换配置失败: %w", err)
	}

	fmt.Printf("✓ 已切换到配置: %s\n", name)
	fmt.Printf("  Token: %s\n", config.MaskToken(cfg.AnthropicAuthToken))
	fmt.Printf("  URL: %s\n", cfg.AnthropicBaseURL)

	return nil
}
