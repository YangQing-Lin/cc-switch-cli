package cmd

import (
	"github.com/spf13/cobra"
)

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage CLAUDE.md templates",
	Long:  `Manage predefined and user-defined CLAUDE.md templates`,
}

func init() {
	rootCmd.AddCommand(templateCmd)
}
