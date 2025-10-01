package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/utils"
	"github.com/spf13/cobra"
)

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "备份配置文件",
	Long: `备份当前配置文件到指定位置。

示例:
  cc-switch backup                            # 备份到默认位置
  cc-switch backup --output my-backup.json    # 备份到指定文件
  cc-switch backup --dir ./backups            # 备份到指定目录
  cc-switch backup --keep 10                  # 只保留最近10个备份`,
	RunE: func(cmd *cobra.Command, args []string) error {
		output, _ := cmd.Flags().GetString("output")
		dir, _ := cmd.Flags().GetString("dir")
		keep, _ := cmd.Flags().GetInt("keep")
		includeClaudeSettings, _ := cmd.Flags().GetBool("include-claude")

		// 获取配置路径
		configPath, err := config.GetConfigPath()
		if err != nil {
			return fmt.Errorf("获取配置文件路径失败: %w", err)
		}

		// 检查配置文件是否存在
		if !utils.FileExists(configPath) {
			return fmt.Errorf("配置文件不存在: %s", configPath)
		}

		// 确定备份路径
		var backupPath string
		if output != "" {
			// 使用指定的输出文件
			backupPath = output
		} else {
			// 生成默认备份文件名
			timestamp := time.Now().Format("20060102-150405")
			filename := fmt.Sprintf("config-backup-%s.json", timestamp)

			if dir != "" {
				// 使用指定目录
				backupPath = filepath.Join(dir, filename)
			} else {
				// 使用默认备份目录
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("获取用户主目录失败: %w", err)
				}
				backupDir := filepath.Join(home, ".cc-switch", "backups")
				backupPath = filepath.Join(backupDir, filename)

				// 创建备份目录
				if err := os.MkdirAll(backupDir, 0755); err != nil {
					return fmt.Errorf("创建备份目录失败: %w", err)
				}
			}
		}

		// 创建备份目录（如果需要）
		backupDir := filepath.Dir(backupPath)
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			return fmt.Errorf("创建备份目录失败: %w", err)
		}

		// 复制配置文件
		if err := utils.CopyFile(configPath, backupPath); err != nil {
			return fmt.Errorf("备份配置文件失败: %w", err)
		}

		fmt.Printf("✓ 配置已备份到: %s\n", backupPath)

		// 备份 Claude 设置（如果需要）
		if includeClaudeSettings {
			claudePath, err := config.GetClaudeSettingsPath()
			if err == nil && utils.FileExists(claudePath) {
				claudeBackup := backupPath[:len(backupPath)-5] + "-claude-settings.json"
				if err := utils.CopyFile(claudePath, claudeBackup); err != nil {
					fmt.Printf("⚠ 备份 Claude 设置失败: %v\n", err)
				} else {
					fmt.Printf("✓ Claude 设置已备份到: %s\n", claudeBackup)
				}
			}
		}

		// 清理旧备份（如果设置了 keep）
		if keep > 0 && dir == "" && output == "" {
			// 只在使用默认备份目录时清理
			if err := cleanOldBackups(backupDir, keep); err != nil {
				fmt.Printf("⚠ 清理旧备份失败: %v\n", err)
			}
		}

		// 显示备份信息
		info, err := os.Stat(backupPath)
		if err == nil {
			fmt.Printf("  文件大小: %.2f KB\n", float64(info.Size())/1024)
			fmt.Printf("  备份时间: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
		}

		return nil
	},
}

// cleanOldBackups 清理旧备份文件
func cleanOldBackups(backupDir string, keep int) error {
	// 获取所有备份文件
	pattern := filepath.Join(backupDir, "config-backup-*.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	// 如果备份数量未超过限制，不需要清理
	if len(files) <= keep {
		return nil
	}

	// 获取文件信息并排序（按修改时间）
	type fileInfo struct {
		path string
		time time.Time
	}
	var fileInfos []fileInfo

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		fileInfos = append(fileInfos, fileInfo{
			path: file,
			time: info.ModTime(),
		})
	}

	// 按时间排序（最旧的在前）
	for i := 0; i < len(fileInfos)-1; i++ {
		for j := i + 1; j < len(fileInfos); j++ {
			if fileInfos[i].time.After(fileInfos[j].time) {
				fileInfos[i], fileInfos[j] = fileInfos[j], fileInfos[i]
			}
		}
	}

	// 删除最旧的文件
	toDelete := len(fileInfos) - keep
	deletedCount := 0
	for i := 0; i < toDelete && i < len(fileInfos); i++ {
		if err := os.Remove(fileInfos[i].path); err == nil {
			deletedCount++
		}
	}

	if deletedCount > 0 {
		fmt.Printf("  已清理 %d 个旧备份\n", deletedCount)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(backupCmd)

	backupCmd.Flags().StringP("output", "o", "", "指定备份文件路径")
	backupCmd.Flags().StringP("dir", "d", "", "指定备份目录")
	backupCmd.Flags().Int("keep", 0, "保留最近N个备份（0表示不限制）")
	backupCmd.Flags().Bool("include-claude", false, "同时备份 Claude 设置文件")
}