package cmd

import (
	"fmt"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "启动交互式 TUI 界面",
	Long:  `启动基于 Bubble Tea 的交互式终端用户界面，提供更友好的配置管理体验。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 初始化配置管理器
		manager, err := getConfigManager()
		if err != nil {
			return fmt.Errorf("初始化配置管理器失败: %w", err)
		}

		// 创建 TUI 模型
		model := tui.New(manager)

		// 启动 Bubble Tea 程序
		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("运行 TUI 失败: %w", err)
		}

		return nil
	},
}

// getConfigManager 获取配置管理器(复用 root.go 的逻辑)
func getConfigManager() (*config.Manager, error) {
	if configDir != "" {
		return config.NewManagerWithDir(configDir)
	}
	return config.NewManager()
}

func init() {
	rootCmd.AddCommand(uiCmd)
}
