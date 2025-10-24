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
	applyTarget string
	skipDiff    bool
)

var applyCmd = &cobra.Command{
	Use:   "apply <template-id>",
	Short: "Apply a template to target path",
	Long:  `Apply a template to one of the target paths (./CLAUDE.md, ~/.claude/CLAUDE.md, ./CLAUDE.local.md)`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		templateID := args[0]
		runApplyTemplate(templateID)
	},
}

func init() {
	templateCmd.AddCommand(applyCmd)
	applyCmd.Flags().StringVar(&applyTarget, "target", "", "Target path ID (project_root, global, local)")
	applyCmd.Flags().BoolVar(&skipDiff, "skip-diff", false, "Skip diff preview")
}

func runApplyTemplate(templateID string) {
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

	// 验证模板存在
	tpl, err := tm.GetTemplate(templateID)
	if err != nil {
		color.Red("Error: %v", err)
		os.Exit(1)
	}

	// 获取目标路径
	var targetPath string
	if applyTarget != "" {
		// 使用指定的目标
		target, err := template.GetTargetByID(applyTarget)
		if err != nil {
			color.Red("Error: Failed to get target: %v", err)
			os.Exit(1)
		}
		if target == nil {
			color.Red("Error: Invalid target ID: %s", applyTarget)
			os.Exit(1)
		}
		targetPath = target.Path
	} else {
		// 交互式选择目标路径
		target, err := selectTarget()
		if err != nil {
			color.Red("Error: %v", err)
			os.Exit(1)
		}
		targetPath = target.Path
	}

	// 显示 diff（除非跳过）
	if !skipDiff {
		diff, err := tm.GetDiff(templateID, targetPath)
		if err != nil {
			color.Red("Error: Failed to generate diff: %v", err)
			os.Exit(1)
		}

		color.Cyan("\n=== Diff Preview ===")
		fmt.Println(template.FormatDiffForCLI(diff))

		// 确认应用
		if !confirmApply() {
			color.Yellow("Operation cancelled.")
			return
		}
	}

	// 应用模板
	if err := tm.ApplyTemplate(templateID, targetPath); err != nil {
		color.Red("Error: Failed to apply template: %v", err)
		os.Exit(1)
	}

	color.Green("✓ Template '%s' applied to: %s", tpl.Name, targetPath)

}

// selectTarget 交互式选择目标路径
func selectTarget() (*template.TemplateTarget, error) {
	targets, err := template.GetClaudeMdTargets()
	if err != nil {
		return nil, err
	}

	color.Cyan("\nSelect target path:")
	for i, target := range targets {
		fmt.Printf("  [%d] %s - %s\n", i+1, target.Name, target.Description)
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

// confirmApply 确认应用操作
func confirmApply() bool {
	fmt.Print("\nApply this template? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.ToLower(strings.TrimSpace(input))

	return input == "y" || input == "yes"
}
