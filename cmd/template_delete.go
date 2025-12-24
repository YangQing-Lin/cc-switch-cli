package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/YangQing-Lin/cc-switch-cli/internal/template"
)

var deleteForce bool

var templateDeleteCmd = &cobra.Command{
	Use:   "delete <template-id>",
	Short: "Delete a user-defined template",
	Long:  `Delete a user-defined template (builtin templates cannot be deleted)`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		templateID := args[0]
		runDeleteTemplate(templateID)
	},
}

func init() {
	templateCmd.AddCommand(templateDeleteCmd)
	templateDeleteCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "Force deletion without confirmation")
}

func runDeleteTemplate(templateID string) {
	// 获取配置路径
	homeDir, err := os.UserHomeDir()
	if err != nil {
		color.Red("Error: Failed to get home directory: %v", err)
		exitFunc(1)
	}

	configPath := filepath.Join(homeDir, ".cc-switch", "claude_templates.json")

	// 创建模板管理器
	tm, err := template.NewTemplateManager(configPath)
	if err != nil {
		color.Red("Error: Failed to initialize template manager: %v", err)
		exitFunc(1)
	}

	// 验证模板存在
	tpl, err := tm.GetTemplate(templateID)
	if err != nil {
		color.Red("Error: %v", err)
		exitFunc(1)
	}

	// 检查是否为预定义模板
	if tpl.IsBuiltin {
		color.Red("Error: Cannot delete builtin template: %s", templateID)
		exitFunc(1)
	}

	// 确认删除（除非强制）
	if !deleteForce {
		fmt.Printf("Template to delete:\n")
		fmt.Printf("  ID: %s\n", color.RedString(templateID))
		fmt.Printf("  Name: %s\n", tpl.Name)
		fmt.Printf("  Category: %s\n\n", tpl.Category)

		if !confirmDelete() {
			color.Yellow("Operation cancelled.")
			return
		}
	}

	// 删除模板
	if err := tm.DeleteTemplate(templateID); err != nil {
		color.Red("Error: Failed to delete template: %v", err)
		exitFunc(1)
	}

	color.Green("✓ Template '%s' deleted successfully", tpl.Name)
}

// confirmDelete 确认删除操作
func confirmDelete() bool {
	fmt.Print("Are you sure you want to delete this template? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.ToLower(strings.TrimSpace(input))

	return input == "y" || input == "yes"
}
