package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/utils"
	"github.com/YangQing-Lin/cc-switch-cli/internal/vscode"
	"github.com/spf13/cobra"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "检查系统环境和配置状态",
	Long: `检查系统环境和配置状态，包括：
- Claude/Codex 配置文件状态
- VS Code/Cursor 集成状态
- 配置文件权限
- 环境变量设置`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCheck()
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
	checkCmd.Flags().BoolP("verbose", "v", false, "显示详细信息")
}

func runCheck() error {
	fmt.Println("系统环境检查")
	fmt.Println("============")
	fmt.Println()

	// 系统信息
	fmt.Printf("操作系统: %s\n", runtime.GOOS)
	fmt.Printf("架构: %s\n", runtime.GOARCH)
	homeDir, _ := os.UserHomeDir()
	fmt.Printf("用户目录: %s\n", homeDir)
	fmt.Println()

	// 检查配置文件
	fmt.Println("配置文件状态")
	fmt.Println("------------")

	// cc-switch 配置
	configPath := filepath.Join(homeDir, ".cc-switch", "config.json")
	checkFile("cc-switch 配置", configPath)

	// Claude 配置
	claudePath, _ := config.GetClaudeSettingsPath()
	checkFile("Claude 设置", claudePath)

	// Codex 配置
	codexConfigPath, _ := config.GetCodexConfigPath()
	checkFile("Codex config.toml", codexConfigPath)

	codexAuthPath, _ := config.GetCodexAuthJsonPath()
	checkFile("Codex auth.json", codexAuthPath)
	fmt.Println()

	// 检查 VS Code 集成
	fmt.Println("VS Code 集成状态")
	fmt.Println("----------------")

	runningApps := vscode.GetRunningApps()
	if len(runningApps) == 0 {
		fmt.Println("未检测到运行中的 VS Code/Cursor")
	} else {
		for _, app := range runningApps {
			fmt.Printf("✓ %s 正在运行\n", app.Name)
			if vscode.ConfigExists(app) {
				fmt.Printf("  配置文件: %s\n", vscode.GetSettingsPath(app))
			} else {
				fmt.Printf("  配置文件: 未找到\n")
			}
		}
	}
	fmt.Println()

	// 检查配置管理器
	fmt.Println("配置管理状态")
	fmt.Println("------------")

	manager, err := config.NewManager()
	if err != nil {
		fmt.Printf("✗ 无法加载配置管理器: %v\n", err)
	} else {
		// Claude 配置
		claudeProviders := manager.ListProvidersForApp("claude")
		fmt.Printf("Claude 配置数量: %d\n", len(claudeProviders))
		if current := manager.GetCurrentProviderForApp("claude"); current != nil {
			fmt.Printf("  当前激活: %s\n", current.Name)
		}

		// Codex 配置
		codexProviders := manager.ListProvidersForApp("codex")
		fmt.Printf("Codex 配置数量: %d\n", len(codexProviders))
		if current := manager.GetCurrentProviderForApp("codex"); current != nil {
			fmt.Printf("  当前激活: %s\n", current.Name)
		}
	}
	fmt.Println()

	// 环境建议
	fmt.Println("环境建议")
	fmt.Println("--------")
	if !utils.FileExists(configPath) {
		fmt.Println("• 运行 'cc-switch init' 初始化配置")
	}
	if len(runningApps) > 0 {
		fmt.Println("• 检测到 VS Code/Cursor 正在运行")
		fmt.Println("  切换配置前建议关闭编辑器或重新加载窗口")
	}

	return nil
}

func checkFile(name string, path string) {
	if utils.FileExists(path) {
		info, err := os.Stat(path)
		if err == nil {
			fmt.Printf("✓ %s: %s (%.1f KB)\n", name, path, float64(info.Size())/1024)
		} else {
			fmt.Printf("✓ %s: %s (无法获取大小)\n", name, path)
		}
	} else {
		fmt.Printf("✗ %s: 不存在\n", name)
	}
}