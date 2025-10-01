package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/YangQing-Lin/cc-switch-cli/internal/version"
	"github.com/spf13/cobra"
)

var checkUpdatesCmd = &cobra.Command{
	Use:   "check-updates",
	Short: "检查更新",
	Long:  `检查 cc-switch 是否有新版本可用`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCheckUpdates()
	},
}

func init() {
	rootCmd.AddCommand(checkUpdatesCmd)
}

func runCheckUpdates() error {
	fmt.Printf("当前版本: %s\n", version.GetVersion())
	fmt.Println("\n正在打开 GitHub Releases 页面...")

	url := "https://github.com/YangQing-Lin/cc-switch-cli/releases"

	// 根据操作系统打开浏览器
	var openCmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		openCmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		openCmd = exec.Command("open", url)
	case "linux":
		openCmd = exec.Command("xdg-open", url)
	default:
		fmt.Printf("\n请手动访问: %s\n", url)
		return nil
	}

	if err := openCmd.Start(); err != nil {
		fmt.Printf("\n自动打开失败，请手动访问: %s\n", url)
		return nil
	}

	fmt.Println("✓ 已在浏览器中打开 GitHub Releases 页面")
	return nil
}
