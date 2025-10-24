package cmd

import (
	"fmt"

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

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.Flags().String("app", "claude", "应用名称 (claude 或 codex)")
}
