package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/utils"
	"github.com/spf13/cobra"
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "导出配置到文件",
	Long: `导出供应商配置到文件，用于备份或共享。

示例:
  cc-switch export                             # 导出到默认文件
  cc-switch export --output backup.json        # 导出到指定文件
  cc-switch export --app claude                # 只导出 Claude 配置
  cc-switch export --provider myconfig         # 只导出指定配置
  cc-switch export --format minimal            # 最小化导出（不含元数据）`,
	RunE: func(cmd *cobra.Command, args []string) error {
		output, _ := cmd.Flags().GetString("output")
		appName, _ := cmd.Flags().GetString("app")
		providerName, _ := cmd.Flags().GetString("provider")
		format, _ := cmd.Flags().GetString("format")
		pretty, _ := cmd.Flags().GetBool("pretty")

		// 创建管理器
		manager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("初始化配置管理器失败: %w", err)
		}

		// 默认输出文件名
		if output == "" {
			timestamp := time.Now().Format("20060102-150405")
			output = fmt.Sprintf("cc-switch-export-%s.json", timestamp)
		}

		// 获取完整配置
		exportConfig, err := manager.GetConfig()
		if err != nil {
			return fmt.Errorf("获取配置失败: %w", err)
		}

		// 根据参数过滤配置
		if appName != "" || providerName != "" {
			exportConfig = filterConfig(exportConfig, appName, providerName)
		}

		// 根据格式调整输出
		var exportData interface{}
		switch format {
		case "minimal":
			// 最小化格式，只包含必要信息
			exportData = convertToMinimal(exportConfig)
		case "standard":
			// 标准格式，移除敏感的运行时信息
			exportData = cleanConfig(exportConfig)
		default:
			// 完整格式
			exportData = exportConfig
		}

		// 序列化配置
		var jsonData []byte
		if pretty {
			jsonData, err = json.MarshalIndent(exportData, "", "  ")
		} else {
			jsonData, err = json.Marshal(exportData)
		}
		if err != nil {
			return fmt.Errorf("序列化配置失败: %w", err)
		}

		// 写入文件
		if err := utils.AtomicWriteFile(output, jsonData, 0644); err != nil {
			return fmt.Errorf("写入文件失败: %w", err)
		}

		// 统计信息
		stats := getExportStats(exportConfig)

		fmt.Printf("✓ 配置已导出到: %s\n", output)
		fmt.Printf("  格式: %s\n", format)
		fmt.Printf("  应用数: %d\n", len(exportConfig.Apps))
		fmt.Printf("  总配置数: %d\n", stats.totalProviders)
		if stats.claudeProviders > 0 {
			fmt.Printf("  - Claude: %d 个配置\n", stats.claudeProviders)
		}
		if stats.codexProviders > 0 {
			fmt.Printf("  - Codex: %d 个配置\n", stats.codexProviders)
		}
		fmt.Printf("  文件大小: %.2f KB\n", float64(len(jsonData))/1024)

		// 安全提醒
		if format != "minimal" {
			fmt.Println("\n⚠ 安全提醒: 导出文件包含 API Token，请妥善保管！")
		}

		return nil
	},
}

// filterConfig 过滤配置
func filterConfig(fullConfig *config.MultiAppConfig, appName, providerName string) *config.MultiAppConfig {
	filtered := &config.MultiAppConfig{
		Version: fullConfig.Version,
		Apps:    make(map[string]config.ProviderManager),
	}

	// 过滤应用
	appsToExport := []string{}
	if appName != "" {
		appsToExport = append(appsToExport, appName)
	} else {
		for app := range fullConfig.Apps {
			appsToExport = append(appsToExport, app)
		}
	}

	// 过滤供应商
	for _, app := range appsToExport {
		if appConfig, exists := fullConfig.Apps[app]; exists {
			if providerName != "" {
				// 只导出指定的供应商
				filteredProviders := make(map[string]config.Provider)
				for id, provider := range appConfig.Providers {
					if provider.Name == providerName {
						filteredProviders[id] = provider
						// 如果是当前配置，保留 current 标记
						if appConfig.Current == id {
							filtered.Apps[app] = config.ProviderManager{
								Providers: filteredProviders,
								Current:   id,
							}
						} else {
							filtered.Apps[app] = config.ProviderManager{
								Providers: filteredProviders,
								Current:   "",
							}
						}
						break
					}
				}
			} else {
				// 导出所有供应商
				filtered.Apps[app] = appConfig
			}
		}
	}

	return filtered
}

// convertToMinimal 转换为最小化格式
func convertToMinimal(fullConfig *config.MultiAppConfig) map[string]interface{} {
	minimal := make(map[string]interface{})
	minimal["version"] = fullConfig.Version

	apps := make(map[string]interface{})
	for appName, appConfig := range fullConfig.Apps {
		providers := make([]map[string]interface{}, 0)
		for _, provider := range appConfig.Providers {
			p := map[string]interface{}{
				"name":     provider.Name,
				"category": provider.Category,
			}

			// 根据应用类型提取关键信息
			if appName == "claude" {
				if envMap, ok := provider.SettingsConfig["env"].(map[string]interface{}); ok {
					p["token"] = "***masked***"
					p["baseUrl"] = envMap["ANTHROPIC_BASE_URL"]
				}
			} else if appName == "codex" {
				p["auth"] = "***masked***"
				if baseURL, ok := provider.SettingsConfig["base_url"].(string); ok {
					p["baseUrl"] = baseURL
				}
			}

			providers = append(providers, p)
		}
		apps[appName] = providers
	}
	minimal["apps"] = apps

	return minimal
}

// cleanConfig 清理配置（移除运行时信息）
func cleanConfig(fullConfig *config.MultiAppConfig) *config.MultiAppConfig {
	cleaned := &config.MultiAppConfig{
		Version: fullConfig.Version,
		Apps:    make(map[string]config.ProviderManager),
	}

	for appName, appConfig := range fullConfig.Apps {
		cleanedProviders := make(map[string]config.Provider)
		for id, provider := range appConfig.Providers {
			// 清理 Provider
			cleanedProvider := provider
			cleanedProvider.CreatedAt = 0 // 移除时间戳
			cleanedProviders[id] = cleanedProvider
		}
		cleaned.Apps[appName] = config.ProviderManager{
			Providers: cleanedProviders,
			Current:   appConfig.Current,
		}
	}

	return cleaned
}

// exportStats 导出统计
type exportStats struct {
	totalProviders  int
	claudeProviders int
	codexProviders  int
}

// getExportStats 获取导出统计
func getExportStats(fullConfig *config.MultiAppConfig) exportStats {
	stats := exportStats{}

	for appName, appConfig := range fullConfig.Apps {
		count := len(appConfig.Providers)
		stats.totalProviders += count

		switch appName {
		case "claude":
			stats.claudeProviders = count
		case "codex":
			stats.codexProviders = count
		}
	}

	return stats
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringP("output", "o", "", "输出文件路径")
	exportCmd.Flags().String("app", "", "只导出指定应用的配置")
	exportCmd.Flags().String("provider", "", "只导出指定名称的配置")
	exportCmd.Flags().String("format", "full", "导出格式 (full, standard, minimal)")
	exportCmd.Flags().Bool("pretty", true, "格式化 JSON 输出")

	// 设置 output 简写
	exportCmd.Flags().Lookup("output").Shorthand = "o"
}
