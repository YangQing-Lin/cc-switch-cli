package cmd

import (
	"fmt"

	"github.com/YangQing-Lin/cc-switch-cli/internal/version"
	"github.com/spf13/cobra"
)

var updateSelfCmd = &cobra.Command{
	Use:   "update",
	Short: "更新到最新版本",
	Long:  `从 GitHub Releases 下载并安装最新版本`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("当前版本: %s\n", version.GetVersion())
		fmt.Println("正在检查更新...")

		release, hasUpdate, err := version.CheckForUpdate()
		if err != nil {
			fmt.Printf("❌ 检查更新失败: %v\n", err)
			return
		}

		if !hasUpdate {
			fmt.Println("✅ 当前已是最新版本，无需更新")
			return
		}

		fmt.Printf("\n发现新版本: %s\n", release.TagName)
		fmt.Println("开始下载更新...")

		if err := version.DownloadUpdate(release); err != nil {
			fmt.Println("❌ 更新失败")
			fmt.Println()
			fmt.Println(err.Error())
			return
		}

		fmt.Printf("\n✅ 更新成功！已更新到版本 %s\n", release.TagName)
		fmt.Println("请重新运行程序以使用新版本")
	},
}

func init() {
	rootCmd.AddCommand(updateSelfCmd)
}
