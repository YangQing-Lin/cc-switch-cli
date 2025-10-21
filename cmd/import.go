package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/utils"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// importCmd represents the import command
var importCmd = &cobra.Command{
	Use:   "import",
	Short: "从文件导入配置",
	Long: `从 JSON 文件导入供应商配置。

示例:
  cc-switch import config.json     # 从文件导入`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("请指定要导入的文件")
		}
		appName, _ := cmd.Flags().GetString("app")

		// 创建管理器
		manager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("初始化配置管理器失败: %w", err)
		}

		return importFromFile(manager, args[0], appName)
	},
}

// importFromFile 从文件导入配置
func importFromFile(manager *config.Manager, filePath, appName string) error {
	if !utils.FileExists(filePath) {
		return fmt.Errorf("文件不存在: %s", filePath)
	}

	// 读取文件
	var importConfig config.MultiAppConfig
	if err := utils.ReadJSONFile(filePath, &importConfig); err != nil {
		return fmt.Errorf("读取导入文件失败: %w", err)
	}

	// 创建自动备份（在导入前）
	configPath := manager.GetConfigPath()
	if utils.FileExists(configPath) {
		backupID, err := createAutoBackup(configPath)
		if err != nil {
			fmt.Printf("⚠ 创建备份失败: %v\n", err)
		} else if backupID != "" {
			fmt.Printf("✓ 已创建备份: %s\n", backupID)
		}
	}

	// 导入配置
	importedCount := 0
	skippedCount := 0

	// 处理指定应用或所有应用
	appsToImport := []string{}
	if appName != "" {
		appsToImport = append(appsToImport, appName)
	} else {
		for app := range importConfig.Apps {
			appsToImport = append(appsToImport, app)
		}
	}

	for _, app := range appsToImport {
		appConfig, exists := importConfig.Apps[app]
		if !exists {
			continue
		}

		// 获取现有配置用于去重
		existingProviders := manager.ListProvidersForApp(app)
		existingTokens := make(map[string]string)
		for _, p := range existingProviders {
			if app == "claude" {
				token := config.ExtractTokenFromProvider(&p)
				if token != "" {
					existingTokens[token] = p.Name
				}
			}
		}

		// 导入每个供应商
		for _, provider := range appConfig.Providers {
			// 检查是否已存在
			skip := false
			if app == "claude" {
				token := config.ExtractTokenFromProvider(&provider)
				if existingName, exists := existingTokens[token]; exists {
					fmt.Printf("⚠ 跳过重复配置: %s (与 '%s' 相同)\n", provider.Name, existingName)
					skippedCount++
					skip = true
				}
			}

			if !skip {
				// 生成新 ID
				newID := uuid.New().String()
				provider.ID = newID

				// 如果名称冲突，添加后缀
				originalName := provider.Name
				suffix := 1
				for {
					nameExists := false
					for _, existing := range existingProviders {
						if existing.Name == provider.Name {
							nameExists = true
							break
						}
					}
					if !nameExists {
						break
					}
					provider.Name = fmt.Sprintf("%s-%d", originalName, suffix)
					suffix++
				}

				// 添加到管理器
				if err := manager.AddProviderDirect(app, provider); err != nil {
					fmt.Printf("✗ 导入 %s 失败: %v\n", provider.Name, err)
				} else {
					fmt.Printf("✓ 导入配置: %s\n", provider.Name)
					importedCount++
				}
			}
		}
	}

	fmt.Printf("\n导入完成: %d 个配置已导入, %d 个配置已跳过\n", importedCount, skippedCount)
	return nil
}

// createAutoBackup 创建自动备份（匹配 GUI 的备份格式）
func createAutoBackup(configPath string) (string, error) {
	// 生成时间戳 (格式: backup_YYYYMMDD_HHMMSS)
	timestamp := time.Now().UTC().Format("20060102_150405")
	backupID := fmt.Sprintf("backup_%s", timestamp)

	// 创建备份目录
	configDir := filepath.Dir(configPath)
	backupDir := filepath.Join(configDir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("创建备份目录失败: %w", err)
	}

	// 创建备份文件
	backupPath := filepath.Join(backupDir, backupID+".json")
	if err := utils.CopyFile(configPath, backupPath); err != nil {
		return "", fmt.Errorf("复制配置文件失败: %w", err)
	}

	// 清理旧备份（保留最近10个）
	cleanupAutoBackups(backupDir, 10)

	return backupID, nil
}

// cleanupAutoBackups 清理旧的自动备份
func cleanupAutoBackups(backupDir string, maxBackups int) {
	pattern := filepath.Join(backupDir, "backup_*.json")
	files, err := filepath.Glob(pattern)
	if err != nil || len(files) <= maxBackups {
		return
	}

	// 获取文件信息并排序
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
	toDelete := len(fileInfos) - maxBackups
	for i := 0; i < toDelete && i < len(fileInfos); i++ {
		os.Remove(fileInfos[i].path)
	}
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.Flags().String("app", "claude", "应用名称 (claude 或 codex)")
}
