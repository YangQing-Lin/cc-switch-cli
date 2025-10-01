package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/utils"
	"github.com/spf13/cobra"
)

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "迁移和合并配置文件",
	Long: `迁移和合并配置文件，支持：
- 合并 claude.json 到 settings.json
- 合并重复的配置项
- 清理无效的配置`,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		return runMigrate(force, dryRun)
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.Flags().BoolP("force", "f", false, "强制迁移，不提示确认")
	migrateCmd.Flags().Bool("dry-run", false, "模拟运行，不实际修改文件")
}

func runMigrate(force bool, dryRun bool) error {
	if dryRun {
		fmt.Println("【模拟运行模式】不会实际修改任何文件")
		fmt.Println()
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("获取用户目录失败: %w", err)
	}

	// 检查是否需要迁移 claude.json -> settings.json
	claudeDir := filepath.Join(homeDir, ".claude")
	oldPath := filepath.Join(claudeDir, "claude.json")
	newPath := filepath.Join(claudeDir, "settings.json")

	migrationNeeded := false
	var actions []string

	// 检查 Claude 配置迁移
	if utils.FileExists(oldPath) {
		if utils.FileExists(newPath) {
			actions = append(actions, fmt.Sprintf("合并 claude.json 到 settings.json"))
			migrationNeeded = true
		} else {
			actions = append(actions, fmt.Sprintf("重命名 claude.json 为 settings.json"))
			migrationNeeded = true
		}
	}

	// 检查配置管理器中的重复项
	manager, err := config.NewManager()
	if err == nil {
		// 检查 Claude 重复配置
		claudeDuplicates := findDuplicateProviders(manager, "claude")
		if len(claudeDuplicates) > 0 {
			actions = append(actions, fmt.Sprintf("发现 %d 个重复的 Claude 配置", len(claudeDuplicates)))
			migrationNeeded = true
		}

		// 检查 Codex 重复配置
		codexDuplicates := findDuplicateProviders(manager, "codex")
		if len(codexDuplicates) > 0 {
			actions = append(actions, fmt.Sprintf("发现 %d 个重复的 Codex 配置", len(codexDuplicates)))
			migrationNeeded = true
		}
	}

	if !migrationNeeded {
		fmt.Println("✓ 没有需要迁移的配置")
		return nil
	}

	// 显示迁移计划
	fmt.Println("迁移计划")
	fmt.Println("========")
	for i, action := range actions {
		fmt.Printf("%d. %s\n", i+1, action)
	}
	fmt.Println()

	// 确认迁移
	if !force && !dryRun {
		fmt.Print("确认执行迁移？(y/N): ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "y" && confirm != "Y" {
			fmt.Println("迁移已取消")
			return nil
		}
	}

	// 执行迁移
	fmt.Println("\n开始迁移...")

	// 1. 迁移 claude.json -> settings.json
	if utils.FileExists(oldPath) {
		if dryRun {
			fmt.Println("[模拟] 迁移 Claude 配置文件")
		} else {
			if err := migrateClaudeConfig(oldPath, newPath); err != nil {
				return fmt.Errorf("迁移 Claude 配置失败: %w", err)
			}
			fmt.Println("✓ 成功迁移 Claude 配置文件")
		}
	}

	// 2. 合并重复配置
	if manager != nil {
		// 合并 Claude 重复项
		claudeDuplicates := findDuplicateProviders(manager, "claude")
		if len(claudeDuplicates) > 0 {
			if dryRun {
				fmt.Printf("[模拟] 合并 %d 个重复的 Claude 配置\n", len(claudeDuplicates))
			} else {
				count := mergeDuplicateProviders(manager, "claude", claudeDuplicates)
				if count > 0 {
					fmt.Printf("✓ 成功合并 %d 个重复的 Claude 配置\n", count)
				}
			}
		}

		// 合并 Codex 重复项
		codexDuplicates := findDuplicateProviders(manager, "codex")
		if len(codexDuplicates) > 0 {
			if dryRun {
				fmt.Printf("[模拟] 合并 %d 个重复的 Codex 配置\n", len(codexDuplicates))
			} else {
				count := mergeDuplicateProviders(manager, "codex", codexDuplicates)
				if count > 0 {
					fmt.Printf("✓ 成功合并 %d 个重复的 Codex 配置\n", count)
				}
			}
		}

		// 保存配置
		if !dryRun {
			if err := manager.Save(); err != nil {
				return fmt.Errorf("保存配置失败: %w", err)
			}
		}
	}

	if dryRun {
		fmt.Println("\n【模拟运行完成】实际运行时将执行上述操作")
	} else {
		fmt.Println("\n✓ 迁移完成")
	}

	return nil
}

// migrateClaudeConfig 迁移 Claude 配置文件
func migrateClaudeConfig(oldPath, newPath string) error {
	// 如果新文件不存在，直接重命名
	if !utils.FileExists(newPath) {
		return os.Rename(oldPath, newPath)
	}

	// 如果新文件存在，需要合并
	var oldConfig, newConfig config.ClaudeSettings

	// 读取旧配置
	if err := utils.ReadJSONFile(oldPath, &oldConfig); err != nil {
		return fmt.Errorf("读取旧配置失败: %w", err)
	}

	// 读取新配置
	if err := utils.ReadJSONFile(newPath, &newConfig); err != nil {
		return fmt.Errorf("读取新配置失败: %w", err)
	}

	// 合并配置（优先保留新配置的值）
	if newConfig.Env.AnthropicAuthToken == "" && oldConfig.Env.AnthropicAuthToken != "" {
		newConfig.Env.AnthropicAuthToken = oldConfig.Env.AnthropicAuthToken
	}
	if newConfig.Env.AnthropicBaseURL == "" && oldConfig.Env.AnthropicBaseURL != "" {
		newConfig.Env.AnthropicBaseURL = oldConfig.Env.AnthropicBaseURL
	}
	if newConfig.Env.ClaudeCodeModel == "" && oldConfig.Env.ClaudeCodeModel != "" {
		newConfig.Env.ClaudeCodeModel = oldConfig.Env.ClaudeCodeModel
	}

	// 保存合并后的配置
	if err := utils.WriteJSONFile(newPath, &newConfig, 0644); err != nil {
		return fmt.Errorf("保存合并配置失败: %w", err)
	}

	// 备份并删除旧文件
	backupPath := oldPath + ".backup"
	if err := os.Rename(oldPath, backupPath); err != nil {
		return fmt.Errorf("备份旧配置失败: %w", err)
	}

	return nil
}

// findDuplicateProviders 查找重复的供应商配置
func findDuplicateProviders(manager *config.Manager, appName string) map[string][]config.Provider {
	duplicates := make(map[string][]config.Provider)
	providers := manager.ListProvidersForApp(appName)

	// 按 API Token 分组
	tokenMap := make(map[string][]config.Provider)
	for _, p := range providers {
		token := config.ExtractTokenFromProvider(&p)
		if token != "" {
			tokenMap[token] = append(tokenMap[token], p)
		}
	}

	// 找出重复项
	for token, provs := range tokenMap {
		if len(provs) > 1 {
			duplicates[token] = provs
		}
	}

	return duplicates
}

// mergeDuplicateProviders 合并重复的供应商配置
func mergeDuplicateProviders(manager *config.Manager, appName string, duplicates map[string][]config.Provider) int {
	mergedCount := 0

	for _, provs := range duplicates {
		if len(provs) <= 1 {
			continue
		}

		// 保留第一个，删除其余的
		keepProvider := provs[0]
		currentProvider := manager.GetCurrentProviderForApp(appName)

		for i := 1; i < len(provs); i++ {
			// 如果要删除的是当前激活的，先切换到保留的那个
			if currentProvider != nil && currentProvider.ID == provs[i].ID {
				manager.SwitchProviderForApp(appName, keepProvider.Name)
			}

			// 删除重复项
			if err := manager.DeleteProviderForApp(appName, provs[i].Name); err == nil {
				mergedCount++
				fmt.Printf("  删除重复配置: %s\n", provs[i].Name)
			}
		}
	}

	return mergedCount
}