package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/YangQing-Lin/cc-switch-cli/internal/template"
)

var (
	addFile     string
	addName     string
	addCategory string
)

var templateAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a custom template from file",
	Long:  `Add a custom template by importing content from a file`,
	Run: func(cmd *cobra.Command, args []string) {
		runAddTemplate()
	},
}

func init() {
	templateCmd.AddCommand(templateAddCmd)
	templateAddCmd.Flags().StringVar(&addFile, "file", "", "Path to template file (required)")
	templateAddCmd.Flags().StringVar(&addName, "name", "", "Template name (required)")
	templateAddCmd.Flags().StringVar(&addCategory, "category", template.CategoryClaudeMd, "Template category")
	if err := templateAddCmd.MarkFlagRequired("file"); err != nil {
		panic(err)
	}
	if err := templateAddCmd.MarkFlagRequired("name"); err != nil {
		panic(err)
	}
}

func runAddTemplate() {
	// 验证文件存在
	if _, err := os.Stat(addFile); os.IsNotExist(err) {
		color.Red("Error: File does not exist: %s", addFile)
		exitFunc(1)
	}

	// 读取文件内容
	content, err := os.ReadFile(addFile)
	if err != nil {
		color.Red("Error: Failed to read file: %v", err)
		exitFunc(1)
	}

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

	// 添加模板
	id, err := tm.AddTemplate(addName, addCategory, string(content))
	if err != nil {
		color.Red("Error: Failed to add template: %v", err)
		exitFunc(1)
	}

	color.Green("✓ Template added successfully")
	fmt.Printf("  ID: %s\n", color.GreenString(id))
	fmt.Printf("  Name: %s\n", addName)
	fmt.Printf("  Category: %s\n", addCategory)
	fmt.Printf("  Source: %s\n", addFile)
}
