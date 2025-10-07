package cmd

import (
	"fmt"

	"github.com/YangQing-Lin/cc-switch-cli/internal/claude"
	"github.com/spf13/cobra"
)

// claudePluginCmd represents the claude-plugin command
var claudePluginCmd = &cobra.Command{
	Use:     "claude-plugin [status|apply|remove|check]",
	Aliases: []string{"plugin", "pl"},
	Short:   "Claude 插件配置管理",
	Long: `Claude 插件配置管理命令。

Claude 插件集成说明:
  - 用于管理 ~/.claude/config.json 文件
  - 应用配置后，写入固定的 primaryApiKey 字段
  - 移除配置时，只删除 primaryApiKey 字段，保留其他配置
  - 适用于需要使用第三方 API 服务的场景

使用方法:
  cc-switch claude-plugin status    # 查看配置状态
  cc-switch claude-plugin apply     # 应用配置
  cc-switch claude-plugin remove    # 移除配置
  cc-switch claude-plugin check     # 检查是否已应用

简化用法:
  ccs plugin status                 # 查看状态（推荐）
  ccs plugin apply                  # 应用配置（推荐）
  ccs plugin remove                 # 移除配置（推荐）
  ccs pl status                     # 最短形式`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 无参数时默认显示状态
		if len(args) == 0 {
			return runClaudePluginStatus()
		}
		return nil
	},
}

// claudePluginStatusCmd represents the claude-plugin status command
var claudePluginStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看 Claude 插件配置状态",
	Long:  `显示 Claude 插件配置文件是否存在及其路径`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runClaudePluginStatus()
	},
}

// claudePluginApplyCmd represents the claude-plugin apply command
var claudePluginApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "应用 Claude 插件配置",
	Long: `应用 Claude 插件配置，写入固定的 primaryApiKey 字段。
适用于使用第三方 API 服务时。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runClaudePluginApply()
	},
}

// claudePluginRemoveCmd represents the claude-plugin remove command
var claudePluginRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "移除 Claude 插件配置",
	Long: `移除 Claude 插件配置，只删除 primaryApiKey 字段。
适用于切换回官方 API 服务时。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runClaudePluginRemove()
	},
}

// claudePluginCheckCmd represents the claude-plugin check command
var claudePluginCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "检查 Claude 插件配置是否已应用",
	Long:  `检查 Claude 插件配置是否包含正确的 primaryApiKey 字段`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runClaudePluginCheck()
	},
}

func init() {
	rootCmd.AddCommand(claudePluginCmd)
	claudePluginCmd.AddCommand(claudePluginStatusCmd)
	claudePluginCmd.AddCommand(claudePluginApplyCmd)
	claudePluginCmd.AddCommand(claudePluginRemoveCmd)
	claudePluginCmd.AddCommand(claudePluginCheckCmd)
}

func runClaudePluginStatus() error {
	fmt.Println("Claude 插件配置状态")
	fmt.Println("===================")
	fmt.Println()

	exists, path, err := claude.ClaudeConfigStatus()
	if err != nil {
		return fmt.Errorf("获取配置状态失败: %w", err)
	}

	fmt.Printf("配置文件: %s\n", path)
	if exists {
		fmt.Println("文件状态: ✓ 存在")

		// 检查是否已应用
		applied, err := claude.IsClaudePluginApplied()
		if err != nil {
			fmt.Printf("检查应用状态失败: %v\n", err)
		} else if applied {
			fmt.Println("配置状态: ✓ 已应用（由 cc-switch 管理）")
		} else {
			fmt.Println("配置状态: ○ 存在但未应用")
		}
	} else {
		fmt.Println("文件状态: ✗ 不存在")
	}

	return nil
}

func runClaudePluginApply() error {
	applied, err := claude.ApplyClaudePlugin()
	if err != nil {
		return fmt.Errorf("应用配置失败: %w", err)
	}

	if applied {
		fmt.Println("✓ Claude 插件配置已应用")
		path, _ := claude.GetClaudeConfigPath()
		fmt.Printf("  配置文件: %s\n", path)
		fmt.Println()
		fmt.Println("说明:")
		fmt.Println("  - 已写入 primaryApiKey 字段")
		fmt.Println("  - 第三方 API 服务现在可以正常工作")
	} else {
		fmt.Println("Claude 插件配置已经应用，无需重复操作")
	}

	return nil
}

func runClaudePluginRemove() error {
	removed, err := claude.RemoveClaudePlugin()
	if err != nil {
		return fmt.Errorf("移除配置失败: %w", err)
	}

	if removed {
		fmt.Println("✓ Claude 插件配置已移除")
		path, _ := claude.GetClaudeConfigPath()
		fmt.Printf("  配置文件: %s\n", path)
		fmt.Println()
		fmt.Println("说明:")
		fmt.Println("  - 已删除 primaryApiKey 字段")
		fmt.Println("  - 其他配置字段已保留")
		fmt.Println("  - 官方 API 服务现在可以正常工作")
	} else {
		fmt.Println("Claude 插件配置未应用或已移除，无需操作")
	}

	return nil
}

func runClaudePluginCheck() error {
	applied, err := claude.IsClaudePluginApplied()
	if err != nil {
		return fmt.Errorf("检查配置失败: %w", err)
	}

	if applied {
		fmt.Println("✓ Claude 插件配置已正确应用")
		fmt.Println("  状态: 由 cc-switch 管理")
	} else {
		fmt.Println("✗ Claude 插件配置未应用")
		fmt.Println("  提示: 使用 'cc-switch claude-plugin apply' 应用配置")
	}

	return nil
}
