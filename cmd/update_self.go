package cmd

import (
	"fmt"
	"os"

	"github.com/YangQing-Lin/cc-switch-cli/internal/version"
	"github.com/spf13/cobra"
)

var updateSelfCmd = &cobra.Command{
	Use:   "update",
	Short: "更新到最新版本",
	Long:  `从 GitHub Releases 下载并安装最新版本，或从本地文件安装`,
	Run: func(cmd *cobra.Command, args []string) {
		fromFile, _ := cmd.Flags().GetString("from-file")
		force, _ := cmd.Flags().GetBool("force")

		// 从本地文件安装
		if fromFile != "" {
			fmt.Printf("当前版本: %s\n", version.GetVersion())
			fmt.Printf("正在从本地文件安装: %s\n", fromFile)

			// 检查文件是否存在
			if _, err := os.Stat(fromFile); os.IsNotExist(err) {
				fmt.Printf("❌ 文件不存在: %s\n", fromFile)
				return
			}

			// 安装二进制文件
			if err := version.InstallBinary(fromFile, force); err != nil {
				fmt.Println("❌ 安装失败")
				fmt.Println()
				fmt.Println(err.Error())
				return
			}

			fmt.Println("\n✅ 安装成功！")
			fmt.Println("请重新运行程序以使用新版本")
			return
		}

		// 在线更新（原有逻辑）
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

	updateSelfCmd.Flags().StringP("from-file", "f", "", "从本地文件安装（支持 .tar.gz, .zip 或裸二进制）")
	updateSelfCmd.Flags().Bool("force", false, "跳过平台验证（高级用户）")
}
