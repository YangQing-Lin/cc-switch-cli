package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/utils"
	"github.com/spf13/cobra"
)

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:   "restore [备份文件]",
	Short: "从备份恢复配置",
	Long: `从备份文件恢复配置。

示例:
  cc-switch restore backup.json                   # 从指定文件恢复
  cc-switch restore --latest                      # 恢复最新的备份
  cc-switch restore --list                        # 列出所有可用备份
  cc-switch restore backup.json --force           # 强制恢复（不创建当前配置的备份）`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		latest, _ := cmd.Flags().GetBool("latest")
		list, _ := cmd.Flags().GetBool("list")
		force, _ := cmd.Flags().GetBool("force")
		validate, _ := cmd.Flags().GetBool("validate")

		// 获取备份目录
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("获取用户主目录失败: %w", err)
		}
		backupDir := filepath.Join(home, ".cc-switch", "backups")

		// 列出备份
		if list {
			return listBackups(backupDir)
		}

		// 确定要恢复的备份文件
		var backupFile string
		if latest {
			// 获取最新的备份
			backupFile, err = getLatestBackup(backupDir)
			if err != nil {
				return fmt.Errorf("获取最新备份失败: %w", err)
			}
		} else if len(args) > 0 {
			backupFile = args[0]
		} else {
			return fmt.Errorf("请指定备份文件或使用 --latest 恢复最新备份")
		}

		// 检查备份文件是否存在
		if !utils.FileExists(backupFile) {
			// 如果不是绝对路径，尝试在备份目录中查找
			if !filepath.IsAbs(backupFile) {
				alternativePath := filepath.Join(backupDir, backupFile)
				if utils.FileExists(alternativePath) {
					backupFile = alternativePath
				} else {
					return fmt.Errorf("备份文件不存在: %s", backupFile)
				}
			} else {
				return fmt.Errorf("备份文件不存在: %s", backupFile)
			}
		}

		// 验证备份文件
		if validate {
			fmt.Println("验证备份文件...")
			var testConfig config.MultiAppConfig
			if err := utils.ReadJSONFile(backupFile, &testConfig); err != nil {
				return fmt.Errorf("备份文件格式无效: %w", err)
			}
			if testConfig.Version != 2 {
				return fmt.Errorf("备份文件版本不兼容: %d", testConfig.Version)
			}
			fmt.Println("✓ 备份文件验证通过")
		}

		// 获取当前配置路径
		configPath, err := config.GetConfigPath()
		if err != nil {
			return fmt.Errorf("获取配置文件路径失败: %w", err)
		}

		// 备份当前配置（除非使用 --force）
		if !force && utils.FileExists(configPath) {
			fmt.Println("备份当前配置...")
			backupPath := configPath + ".before-restore"
			if err := utils.CopyFile(configPath, backupPath); err != nil {
				return fmt.Errorf("备份当前配置失败: %w", err)
			}
			fmt.Printf("✓ 当前配置已备份到: %s\n", backupPath)
		}

		// 恢复配置
		fmt.Printf("恢复配置从: %s\n", backupFile)
		if err := utils.CopyFile(backupFile, configPath); err != nil {
			return fmt.Errorf("恢复配置失败: %w", err)
		}

		fmt.Println("✓ 配置恢复成功")

		// 显示恢复后的配置信息
		manager, err := config.NewManager()
		if err != nil {
			fmt.Printf("⚠ 无法加载恢复后的配置: %v\n", err)
		} else {
			fullConfig, _ := manager.GetConfig()
			if fullConfig != nil {
				providerCount := 0
				for _, app := range fullConfig.Apps {
					providerCount += len(app.Providers)
				}
				fmt.Printf("  应用数: %d\n", len(fullConfig.Apps))
				fmt.Printf("  供应商数: %d\n", providerCount)
			}
		}

		return nil
	},
}

// listBackups 列出所有备份
func listBackups(backupDir string) error {
	pattern := filepath.Join(backupDir, "config-backup-*.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("搜索备份文件失败: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("没有找到备份文件")
		return nil
	}

	// 获取文件信息
	type backupInfo struct {
		name string
		path string
		size int64
		time string
	}
	var backups []backupInfo

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		backups = append(backups, backupInfo{
			name: filepath.Base(file),
			path: file,
			size: info.Size(),
			time: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	// 按时间排序（最新的在前）
	for i := 0; i < len(backups)-1; i++ {
		for j := i + 1; j < len(backups); j++ {
			if backups[i].time < backups[j].time {
				backups[i], backups[j] = backups[j], backups[i]
			}
		}
	}

	// 显示备份列表
	fmt.Printf("找到 %d 个备份:\n", len(backups))
	fmt.Println("─────────────────────────────────────────────")

	for i, backup := range backups {
		marker := "  "
		if i == 0 {
			marker = "→ " // 标记最新的备份
		}
		fmt.Printf("%s%s\n", marker, backup.name)
		fmt.Printf("    大小: %.2f KB  时间: %s\n", float64(backup.size)/1024, backup.time)
	}

	fmt.Println("\n提示: 使用 'cc-switch restore --latest' 恢复最新备份")
	fmt.Println("      或 'cc-switch restore <文件名>' 恢复指定备份")

	return nil
}

// getLatestBackup 获取最新的备份文件
func getLatestBackup(backupDir string) (string, error) {
	pattern := filepath.Join(backupDir, "config-backup-*.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}

	if len(files) == 0 {
		return "", fmt.Errorf("没有找到备份文件")
	}

	// 找到最新的文件
	var latestFile string
	var latestTime int64

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		if info.ModTime().Unix() > latestTime {
			latestTime = info.ModTime().Unix()
			latestFile = file
		}
	}

	if latestFile == "" {
		return "", fmt.Errorf("无法确定最新的备份文件")
	}

	return latestFile, nil
}

func init() {
	rootCmd.AddCommand(restoreCmd)

	restoreCmd.Flags().Bool("latest", false, "恢复最新的备份")
	restoreCmd.Flags().Bool("list", false, "列出所有可用备份")
	restoreCmd.Flags().Bool("force", false, "强制恢复（不创建当前配置的备份）")
	restoreCmd.Flags().Bool("validate", true, "恢复前验证备份文件")
}
