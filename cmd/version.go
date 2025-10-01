package cmd

import (
	"fmt"

	"github.com/YangQing-Lin/cc-switch-cli/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Long:  `显示 cc-switch 的版本信息`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("cc-switch 版本: %s\n", version.GetVersion())

		if version.GetBuildDate() != "unknown" {
			fmt.Printf("构建日期: %s\n", version.GetBuildDate())
		}

		if version.GetGitCommit() != "unknown" {
			fmt.Printf("Git 提交: %s\n", version.GetGitCommit())
		}

		fmt.Println("\n项目地址: https://github.com/YangQing-Lin/cc-switch-cli")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
