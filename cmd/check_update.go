package cmd

import (
	"fmt"

	"github.com/YangQing-Lin/cc-switch-cli/internal/version"
	"github.com/spf13/cobra"
)

var checkForUpdateFunc = version.CheckForUpdate

var checkUpdateCmd = &cobra.Command{
	Use:   "check-update",
	Short: "检查是否有新版本",
	Long:  `检查 GitHub Releases 是否有新版本可用`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("当前版本: %s\n", version.GetVersion())
		fmt.Println("正在检查更新...")

		release, hasUpdate, err := checkForUpdateFunc()
		if err != nil {
			fmt.Printf("❌ 检查更新失败: %v\n", err)
			return
		}

		if hasUpdate {
			fmt.Printf("\n🎉 发现新版本: %s\n", release.TagName)
			fmt.Printf("发布时间: %s\n", release.PublishedAt)
			fmt.Printf("下载地址: %s\n", release.HTMLURL)
			if release.Body != "" {
				fmt.Printf("\n更新说明:\n%s\n", release.Body)
			}
			fmt.Printf("\n运行以下命令更新到最新版本:\n  ccs update\n")
		} else {
			fmt.Println("✅ 当前已是最新版本")
		}
	},
}

func init() {
	rootCmd.AddCommand(checkUpdateCmd)
}
