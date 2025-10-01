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
	"gopkg.in/yaml.v3"
)

// importCmd represents the import command
var importCmd = &cobra.Command{
	Use:   "import",
	Short: "从 live 配置导入供应商",
	Long: `从目标应用的 live 配置文件导入供应商配置。

支持的应用:
  - Claude: ~/.claude/settings.json
  - Codex: ~/.codex/auth.json + ~/.codex/config.toml

示例:
  cc-switch import --from-live                 # 导入当前 Claude live 配置
  cc-switch import --from-live --app codex     # 导入 Codex live 配置
  cc-switch import --from-file config.json     # 从文件导入
  cc-switch import --scan                      # 扫描并导入所有副本文件`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fromLive, _ := cmd.Flags().GetBool("from-live")
		fromFile, _ := cmd.Flags().GetString("from-file")
		scan, _ := cmd.Flags().GetBool("scan")
		appName, _ := cmd.Flags().GetString("app")
		name, _ := cmd.Flags().GetString("name")

		// 创建管理器
		manager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("初始化配置管理器失败: %w", err)
		}

		if fromLive {
			return importFromLive(manager, appName, name)
		} else if fromFile != "" {
			return importFromFile(manager, fromFile, appName)
		} else if scan {
			return scanAndImport(manager, appName)
		} else {
			return fmt.Errorf("请指定导入来源: --from-live, --from-file 或 --scan")
		}
	},
}

// importFromLive 从 live 配置导入
func importFromLive(manager *config.Manager, appName, name string) error {
	switch appName {
	case "claude":
		return importClaudeLive(manager, name)
	case "codex":
		return importCodexLive(manager, name)
	default:
		return fmt.Errorf("不支持的应用: %s", appName)
	}
}

// importClaudeLive 导入 Claude 的 live 配置
func importClaudeLive(manager *config.Manager, name string) error {
	settingsPath, err := config.GetClaudeSettingsPath()
	if err != nil {
		return fmt.Errorf("获取 Claude 设置文件路径失败: %w", err)
	}

	if !utils.FileExists(settingsPath) {
		return fmt.Errorf("Claude 设置文件不存在: %s", settingsPath)
	}

	// 读取 live 配置
	var settings config.ClaudeSettings
	if err := utils.ReadJSONFile(settingsPath, &settings); err != nil {
		return fmt.Errorf("读取 Claude 设置文件失败: %w", err)
	}

	// 验证配置
	if settings.Env.AnthropicAuthToken == "" {
		return fmt.Errorf("Claude 配置缺少 ANTHROPIC_AUTH_TOKEN")
	}

	// 默认名称
	if name == "" {
		name = "imported-" + time.Now().Format("20060102-150405")
	}

	// 检查是否已存在同样的配置
	providers := manager.ListProvidersForApp("claude")
	for _, p := range providers {
		token := config.ExtractTokenFromProvider(&p)
		if token == settings.Env.AnthropicAuthToken {
			return fmt.Errorf("相同的 API Token 已存在于配置 '%s' 中", p.Name)
		}
	}

	// 设置默认值
	baseURL := settings.Env.AnthropicBaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	// 添加配置
	err = manager.AddProviderForApp("claude", name,
		"", // websiteURL
		settings.Env.AnthropicAuthToken,
		baseURL,
		"imported")

	if err != nil {
		return fmt.Errorf("添加配置失败: %w", err)
	}

	fmt.Printf("✓ 成功从 %s 导入配置 '%s'\n", settingsPath, name)
	fmt.Printf("  Token: %s\n", config.MaskToken(settings.Env.AnthropicAuthToken))
	fmt.Printf("  URL: %s\n", baseURL)

	return nil
}

// importCodexLive 导入 Codex 的 live 配置
func importCodexLive(manager *config.Manager, name string) error {
	configPath, err := config.GetCodexConfigPath()
	if err != nil {
		return fmt.Errorf("获取 Codex config 路径失败: %w", err)
	}

	apiJsonPath, err := config.GetCodexApiJsonPath()
	if err != nil {
		return fmt.Errorf("获取 Codex api.json 路径失败: %w", err)
	}

	// 检查两个文件是否都存在
	if !utils.FileExists(configPath) && !utils.FileExists(apiJsonPath) {
		return fmt.Errorf("Codex 配置文件不存在")
	}

	// 读取 config.yaml
	var configData config.CodexConfig
	if utils.FileExists(configPath) {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("读取 config.yaml 失败: %w", err)
		}
		if err := yaml.Unmarshal(data, &configData); err != nil {
			return fmt.Errorf("解析 config.yaml 失败: %w", err)
		}
	}

	// 读取 api.json
	var apiData config.CodexApiJson
	if utils.FileExists(apiJsonPath) {
		if err := utils.ReadJSONFile(apiJsonPath, &apiData); err != nil {
			return fmt.Errorf("读取 api.json 失败: %w", err)
		}
	}

	// 合并配置（api.json 优先）
	apiKey := configData.APIKey
	if apiData.APIKey != "" {
		apiKey = apiData.APIKey
	}
	baseURL := configData.BaseURL
	if apiData.BaseURL != "" {
		baseURL = apiData.BaseURL
	}

	// 验证配置
	if apiKey == "" {
		return fmt.Errorf("Codex 配置缺少 API Key")
	}

	// 默认名称
	if name == "" {
		name = "imported-" + time.Now().Format("20060102-150405")
	}

	// 检查是否已存在同样的配置
	providers := manager.ListProvidersForApp("codex")
	for _, p := range providers {
		token := config.ExtractTokenFromProvider(&p)
		if token == apiKey {
			return fmt.Errorf("相同的 API Key 已存在于配置 '%s' 中", p.Name)
		}
	}

	// 设置默认值
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	// 添加配置
	if err := manager.AddProviderForApp("codex", name, "", apiKey, baseURL, "imported"); err != nil {
		return fmt.Errorf("添加 Codex 配置失败: %w", err)
	}

	fmt.Printf("成功导入 Codex 配置 '%s':\n", name)
	fmt.Printf("  Token: %s\n", config.MaskToken(apiKey))
	fmt.Printf("  URL: %s\n", baseURL)
	if configData.ModelName != "" {
		fmt.Printf("  Model: %s\n", configData.ModelName)
	}

	return nil
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

// scanAndImport 扫描并导入副本文件
func scanAndImport(manager *config.Manager, appName string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("获取用户主目录失败: %w", err)
	}

	importedCount := 0

	// 扫描 Claude 副本文件
	if appName == "" || appName == "claude" {
		claudeDir := filepath.Join(home, ".claude")
		pattern := filepath.Join(claudeDir, "settings-*.json")
		matches, _ := filepath.Glob(pattern)

		for _, file := range matches {
			fmt.Printf("发现副本文件: %s\n", filepath.Base(file))

			var settings config.ClaudeSettings
			if err := utils.ReadJSONFile(file, &settings); err != nil {
				fmt.Printf("  ✗ 读取失败: %v\n", err)
				continue
			}

			if settings.Env.AnthropicAuthToken == "" {
				fmt.Printf("  ✗ 缺少 API Token，跳过\n")
				continue
			}

			// 检查是否已存在
			providers := manager.ListProvidersForApp("claude")
			exists := false
			for _, p := range providers {
				token := config.ExtractTokenFromProvider(&p)
				if token == settings.Env.AnthropicAuthToken {
					fmt.Printf("  ⚠ Token 已存在于配置 '%s'，跳过\n", p.Name)
					exists = true
					break
				}
			}

			if !exists {
				// 从文件名提取时间戳作为名称
				name := "backup-" + filepath.Base(file)
				name = name[:len(name)-5] // 去掉 .json

				baseURL := settings.Env.AnthropicBaseURL
				if baseURL == "" {
					baseURL = "https://api.anthropic.com"
				}

				err = manager.AddProviderForApp("claude", name,
					"", // websiteURL
					settings.Env.AnthropicAuthToken,
					baseURL,
					"backup")

				if err != nil {
					fmt.Printf("  ✗ 导入失败: %v\n", err)
				} else {
					fmt.Printf("  ✓ 已导入为: %s\n", name)
					importedCount++
				}
			}
		}
	}

	// TODO: 扫描 Codex 副本文件

	if importedCount == 0 {
		fmt.Println("没有发现需要导入的副本文件")
	} else {
		fmt.Printf("\n扫描完成: 导入了 %d 个配置\n", importedCount)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(importCmd)

	importCmd.Flags().Bool("from-live", false, "从 live 配置导入")
	importCmd.Flags().String("from-file", "", "从指定文件导入")
	importCmd.Flags().Bool("scan", false, "扫描并导入所有副本文件")
	importCmd.Flags().String("app", "claude", "应用名称 (claude 或 codex)")
	importCmd.Flags().String("name", "", "导入后的配置名称")
}