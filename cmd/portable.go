package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/portable"
	"github.com/spf13/cobra"
)

// portableCmd represents the portable command
var portableCmd = &cobra.Command{
	Use:     "portable [status|enable|disable|on|off]",
	Aliases: []string{"port", "p"},
	Short:   "便携版模式管理",
	Long: `便携版模式管理命令。

便携版模式说明:
  - 在程序所在目录下放置 portable.ini 文件即可启用便携版模式
  - 便携版模式下，配置文件存储在程序目录的 .cc-switch 子目录中
  - 适用于 USB 便携设备或不想污染用户主目录的场景

使用方法:
  cc-switch portable [status]  # 查看便携版状态（默认）
  cc-switch portable enable    # 启用便携版模式（创建 portable.ini）
  cc-switch portable disable   # 禁用便携版模式（删除 portable.ini）
  cc-switch portable on        # 启用便携版模式（简写）
  cc-switch portable off       # 禁用便携版模式（简写）

简化用法:
  ccs port                     # 查看状态
  ccs port on                  # 启用
  ccs port off                 # 禁用
  ccs p                        # 查看状态（最短）
  ccs p on                     # 启用（最短）`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 无参数时默认显示状态
		if len(args) == 0 {
			return runPortableStatus()
		}
		return nil
	},
}

// portableStatusCmd represents the portable status command
var portableStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看便携版模式状态",
	Long:  `显示当前是否为便携版模式以及配置文件位置`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPortableStatus()
	},
}

// portableEnableCmd represents the portable enable command
var portableEnableCmd = &cobra.Command{
	Use:     "enable",
	Aliases: []string{"on"},
	Short:   "启用便携版模式",
	Long: `启用便携版模式，在程序所在目录创建 portable.ini 文件。
启用后，配置将存储在程序目录而非用户主目录。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPortableEnable()
	},
}

// portableDisableCmd represents the portable disable command
var portableDisableCmd = &cobra.Command{
	Use:     "disable",
	Aliases: []string{"off"},
	Short:   "禁用便携版模式",
	Long: `禁用便携版模式，删除程序所在目录的 portable.ini 文件。
注意：已有的便携版配置不会被删除，需要手动清理。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPortableDisable()
	},
}

func init() {
	rootCmd.AddCommand(portableCmd)
	portableCmd.AddCommand(portableStatusCmd)
	portableCmd.AddCommand(portableEnableCmd)
	portableCmd.AddCommand(portableDisableCmd)
}

func runPortableStatus() error {
	fmt.Println("便携版模式状态")
	fmt.Println("==============")
	fmt.Println()

	isPortable := portable.IsPortableMode()
	if isPortable {
		fmt.Println("✓ 便携版模式：已启用")
	} else {
		fmt.Println("✗ 便携版模式：未启用")
	}
	fmt.Println()

	// 显示可执行文件路径
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		fmt.Printf("程序目录: %s\n", execDir)
		portableFile := filepath.Join(execDir, "portable.ini")
		fmt.Printf("标记文件: %s", portableFile)
		if isPortable {
			fmt.Printf(" (存在)\n")
		} else {
			fmt.Printf(" (不存在)\n")
		}
	}
	fmt.Println()

	// 显示配置文件路径
	configPath, err := config.GetConfigPath()
	if err != nil {
		return fmt.Errorf("获取配置路径失败: %w", err)
	}
	fmt.Printf("配置文件: %s\n", configPath)

	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); err == nil {
		fmt.Println("配置状态: 已存在")
	} else if os.IsNotExist(err) {
		fmt.Println("配置状态: 不存在")
	} else {
		fmt.Printf("配置状态: 无法确定 (%v)\n", err)
	}

	return nil
}

func runPortableEnable() error {
	// 检查是否已经是便携版模式
	if portable.IsPortableMode() {
		fmt.Println("便携版模式已经启用")
		return runPortableStatus()
	}

	// 获取可执行文件目录
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取程序路径失败: %w", err)
	}

	execDir := filepath.Dir(execPath)
	portableFile := filepath.Join(execDir, "portable.ini")

	// 创建 portable.ini 文件
	content := []byte("# CC-Switch Portable Mode\n# This file enables portable mode.\n# Delete this file to disable portable mode.\n")
	if err := os.WriteFile(portableFile, content, 0644); err != nil {
		return fmt.Errorf("创建 portable.ini 失败: %w", err)
	}

	fmt.Println("✓ 便携版模式已启用")
	fmt.Printf("  标记文件: %s\n", portableFile)
	fmt.Println()

	// 显示配置目录
	configDir, err := portable.GetPortableConfigDir()
	if err != nil {
		return fmt.Errorf("获取便携版配置目录失败: %w", err)
	}
	fmt.Printf("配置目录: %s\n", configDir)
	fmt.Println()
	fmt.Println("提示：")
	fmt.Println("  - 配置文件现在将存储在程序目录中")
	fmt.Println("  - 如需迁移现有配置，请使用 'cc-switch export' 和 'cc-switch import' 命令")

	return nil
}

func runPortableDisable() error {
	// 检查是否为便携版模式
	if !portable.IsPortableMode() {
		fmt.Println("便携版模式未启用，无需禁用")
		return nil
	}

	// 获取可执行文件目录
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取程序路径失败: %w", err)
	}

	execDir := filepath.Dir(execPath)
	portableFile := filepath.Join(execDir, "portable.ini")

	// 删除 portable.ini 文件
	if err := os.Remove(portableFile); err != nil {
		return fmt.Errorf("删除 portable.ini 失败: %w", err)
	}

	fmt.Println("✓ 便携版模式已禁用")
	fmt.Printf("  已删除: %s\n", portableFile)
	fmt.Println()

	// 显示新的配置路径
	configPath, err := config.GetConfigPath()
	if err == nil {
		fmt.Printf("配置文件将使用: %s\n", configPath)
	}
	fmt.Println()
	fmt.Println("提示：")
	fmt.Println("  - 便携版配置文件仍然保留在程序目录中")
	fmt.Println("  - 如需迁移配置，请使用 'cc-switch export' 和 'cc-switch import' 命令")

	return nil
}
