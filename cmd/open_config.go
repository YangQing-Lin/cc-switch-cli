package cmd

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

var openConfigCmd = &cobra.Command{
	Use:   "open-config",
	Short: "打开配置文件夹",
	Long:  `在系统文件管理器中打开 cc-switch 配置文件夹`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runOpenConfig()
	},
}

func init() {
	rootCmd.AddCommand(openConfigCmd)
}

func runOpenConfig() error {
	manager, err := getManager()
	if err != nil {
		return err
	}

	// 获取配置目录
	configDir := filepath.Dir(manager.GetConfigPath())

	fmt.Printf("配置目录: %s\n", configDir)

	// 根据操作系统打开文件管理器
	var openCmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		openCmd = exec.Command("explorer", configDir)
	case "darwin":
		openCmd = exec.Command("open", configDir)
	case "linux":
		// 尝试多个 Linux 文件管理器
		openCmd = exec.Command("xdg-open", configDir)
	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}

	if err := openCmd.Start(); err != nil {
		return fmt.Errorf("打开文件管理器失败: %w", err)
	}

	fmt.Println("✓ 已在文件管理器中打开配置目录")
	return nil
}
