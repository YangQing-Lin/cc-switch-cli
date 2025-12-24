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

var (
	saveFrom string
	saveName string
)

var saveCmd = &cobra.Command{
	Use:   "save",
	Short: "Save current configuration as a template",
	Long:  `Save current CLAUDE.md configuration as a user-defined template`,
	Run: func(cmd *cobra.Command, args []string) {
		runSaveTemplate()
	},
}

func init() {
	templateCmd.AddCommand(saveCmd)
	saveCmd.Flags().StringVar(&saveFrom, "from", "", "Source path ID (project_root, global, local)")
	saveCmd.Flags().StringVar(&saveName, "name", "", "Template name")
}

func runSaveTemplate() {
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

	// 获取源路径
	var sourcePath string
	if saveFrom != "" {
		// 使用指定的源
		target, err := template.GetTargetByID(saveFrom)
		if err != nil {
			color.Red("Error: Failed to get target: %v", err)
			exitFunc(1)
		}
		if target == nil {
			color.Red("Error: Invalid source ID: %s", saveFrom)
			exitFunc(1)
		}
		sourcePath = target.Path
	} else {
		// 交互式选择源路径
		target, err := selectSourcePath()
		if err != nil {
			color.Red("Error: %v", err)
			exitFunc(1)
		}
		sourcePath = target.Path
	}

	// 检查源文件是否存在
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		color.Red("Error: Source file does not exist: %s", sourcePath)
		exitFunc(1)
	}

	// 获取模板名称
	templateName := saveName
	if templateName == "" {
		// 生成默认名称或交互式输入
		defaultName := tm.GenerateDefaultTemplateName()
		templateName = promptTemplateName(defaultName)
	}

	// 保存为模板
	id, err := tm.SaveAsTemplate(sourcePath, templateName, template.CategoryClaudeMd)
	if err != nil {
		color.Red("Error: Failed to save template: %v", err)
		exitFunc(1)
	}

	color.Green("✓ Template saved successfully")
	fmt.Printf("  ID: %s\n", color.GreenString(id))
	fmt.Printf("  Name: %s\n", templateName)
	fmt.Printf("  Category: %s\n", template.CategoryClaudeMd)
	fmt.Printf("  Source: %s\n", sourcePath)
}

// selectSourcePath 交互式选择源路径
func selectSourcePath() (*template.TemplateTarget, error) {
	targets, err := template.GetClaudeMdTargets()
	if err != nil {
		return nil, err
	}

	color.Cyan("\nSelect source path:")
	for i, target := range targets {
		// 检查文件是否存在
		exists := "✗"
		if _, err := os.Stat(target.Path); err == nil {
			exists = "✓"
		}
		fmt.Printf("  [%d] %s %s - %s\n", i+1, exists, target.Name, target.Description)
	}

	fmt.Print("\nEnter option [1-3]: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var choice int
	_, err = fmt.Sscanf(input, "%d", &choice)
	if err != nil || choice < 1 || choice > len(targets) {
		return nil, fmt.Errorf("invalid choice: %s", input)
	}

	return &targets[choice-1], nil
}

// promptTemplateName 提示输入模板名称
func promptTemplateName(defaultName string) string {
	fmt.Printf("\nEnter template name (leave empty to use default '%s'): ", defaultName)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultName
	}

	return input
}
