package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/utils"
	"github.com/spf13/cobra"
)

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "备份配置文件",
	Long: `备份、列出和恢复配置文件。

子命令:
  cc-switch backup               # 创建新备份
  cc-switch backup list          # 列出所有备份
  cc-switch backup restore <id>  # 从备份恢复

备份示例:
  cc-switch backup                            # 备份到默认位置
  cc-switch backup --output my-backup.json    # 备份到指定文件
  cc-switch backup --dir ./backups            # 备份到指定目录`,
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
	sort.Slice(fileInfos, func(i, j int) bool {
		return fileInfos[i].time.Before(fileInfos[j].time)
	})

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

// backupListCmd lists all backups
var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有备份",
	Long:  `列出所有可用的配置备份文件，包括自动备份和手动备份。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 获取配置目录
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("获取用户主目录失败: %w", err)
		}

		backupDir := filepath.Join(home, ".cc-switch", "backups")
		if !utils.FileExists(backupDir) {
			fmt.Println("没有找到任何备份文件")
			return nil
		}

		// 获取所有备份文件
		pattern := filepath.Join(backupDir, "*.json")
		files, err := filepath.Glob(pattern)
		if err != nil {
			return fmt.Errorf("读取备份目录失败: %w", err)
		}

		if len(files) == 0 {
			fmt.Println("没有找到任何备份文件")
			return nil
		}

		// 获取文件信息并排序
		type backupInfo struct {
			name    string
			path    string
			size    int64
			modTime time.Time
		}
		var backups []backupInfo

		for _, file := range files {
			info, err := os.Stat(file)
			if err != nil {
				continue
			}
			backups = append(backups, backupInfo{
				name:    filepath.Base(file),
				path:    file,
				size:    info.Size(),
				modTime: info.ModTime(),
			})
		}

		// 按时间排序（最新的在前）
		sort.Slice(backups, func(i, j int) bool {
			return backups[i].modTime.After(backups[j].modTime)
		})

		// 显示备份列表
		fmt.Printf("找到 %d 个备份文件:\n\n", len(backups))
		for i, backup := range backups {
			fmt.Printf("%d. %s\n", i+1, backup.name)
			fmt.Printf("   时间: %s\n", backup.modTime.Format("2006-01-02 15:04:05"))
			fmt.Printf("   大小: %.2f KB\n", float64(backup.size)/1024)
			fmt.Printf("   路径: %s\n\n", backup.path)
		}

		return nil
	},
}

// backupRestoreCmd restores from a backup
var backupRestoreCmd = &cobra.Command{
	Use:   "restore <backup-name>",
	Short: "从备份恢复配置",
	Long: `从指定的备份文件恢复配置。

示例:
  cc-switch backup restore backup_20251006_230307.json
  cc-switch backup restore backup_20251006_230307`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		backupName := args[0]

		// 如果没有 .json 后缀，添加它
		if filepath.Ext(backupName) != ".json" {
			backupName += ".json"
		}

		// 获取备份目录
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("获取用户主目录失败: %w", err)
		}

		backupPath := filepath.Join(home, ".cc-switch", "backups", backupName)
		if !utils.FileExists(backupPath) {
			return fmt.Errorf("备份文件不存在: %s", backupPath)
		}

		// 验证备份文件
		var testConfig config.MultiAppConfig
		if err := utils.ReadJSONFile(backupPath, &testConfig); err != nil {
			return fmt.Errorf("备份文件格式无效: %w", err)
		}

		// 获取当前配置路径
		configPath, err := config.GetConfigPath()
		if err != nil {
			return fmt.Errorf("获取配置文件路径失败: %w", err)
		}

		// 在恢复前创建当前配置的备份
		if utils.FileExists(configPath) {
			timestamp := time.Now().UTC().Format("20060102_150405")
			preRestoreBackup := fmt.Sprintf("backup_%s_pre-restore.json", timestamp)
			preRestorePath := filepath.Join(filepath.Dir(backupPath), preRestoreBackup)

			if err := utils.CopyFile(configPath, preRestorePath); err != nil {
				fmt.Printf("⚠ 警告: 无法创建恢复前备份: %v\n", err)
			} else {
				fmt.Printf("✓ 已创建恢复前备份: %s\n", preRestoreBackup)
			}
		}

		// 恢复配置
		if err := utils.CopyFile(backupPath, configPath); err != nil {
			return fmt.Errorf("恢复配置失败: %w", err)
		}

		fmt.Printf("✓ 配置已从备份恢复: %s\n", backupName)
		fmt.Printf("  配置文件: %s\n", configPath)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupRestoreCmd)

	backupCmd.Flags().StringP("output", "o", "", "指定备份文件路径")
	backupCmd.Flags().StringP("dir", "d", "", "指定备份目录")
	backupCmd.Flags().Int("keep", 0, "保留最近N个备份（0表示不限制）")
	backupCmd.Flags().Bool("include-claude", false, "同时备份 Claude 设置文件")
}