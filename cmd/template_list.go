package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/YangQing-Lin/cc-switch-cli/internal/template"
)

var listCategory string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available templates",
	Long:  `List all predefined and user-defined templates`,
	Run: func(cmd *cobra.Command, args []string) {
		runListTemplates()
	},
}

func init() {
	templateCmd.AddCommand(listCmd)
	listCmd.Flags().StringVar(&listCategory, "category", template.CategoryClaudeMd, "Filter by category")
}

func runListTemplates() {
	// 获取配置路径
	homeDir, err := os.UserHomeDir()
	if err != nil {
		color.Red("Error: Failed to get home directory: %v", err)
		os.Exit(1)
	}

	configPath := filepath.Join(homeDir, ".cc-switch", "claude_templates.json")

	// 创建模板管理器
	tm, err := template.NewTemplateManager(configPath)
	if err != nil {
		color.Red("Error: Failed to initialize template manager: %v", err)
		os.Exit(1)
	}

	// 获取模板列表
	templates := tm.ListTemplates(listCategory)

	if len(templates) == 0 {
		color.Yellow("No templates found in category: %s", listCategory)
		return
	}

	// 分组显示：预定义 vs 用户自定义
	builtinTemplates := []template.Template{}
	userTemplates := []template.Template{}

	for _, t := range templates {
		if t.IsBuiltin {
			builtinTemplates = append(builtinTemplates, t)
		} else {
			userTemplates = append(userTemplates, t)
		}
	}

	// 显示预定义模板
	if len(builtinTemplates) > 0 {
		color.Cyan("\n=== Builtin Templates ===")
		for _, t := range builtinTemplates {
			fmt.Printf("  ID: %s\n", color.GreenString(t.ID))
			fmt.Printf("  Name: %s\n", t.Name)
			fmt.Printf("  Category: %s\n\n", t.Category)
		}
	}

	// 显示用户自定义模板
	if len(userTemplates) > 0 {
		color.Cyan("=== User Templates ===")
		for _, t := range userTemplates {
			fmt.Printf("  ID: %s\n", color.GreenString(t.ID))
			fmt.Printf("  Name: %s\n", t.Name)
			fmt.Printf("  Category: %s\n\n", t.Category)
		}
	}

	color.Green("Total: %d templates", len(templates))
}
