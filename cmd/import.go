package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/utils"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
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
		"imported",
		"") // defaultSonnetModel

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
		return fmt.Errorf("获取 Codex config.toml 路径失败: %w", err)
	}

	authJsonPath, err := config.GetCodexAuthJsonPath()
	if err != nil {
		return fmt.Errorf("获取 Codex auth.json 路径失败: %w", err)
	}

	// 检查 auth.json 是否存在（必需）
	if !utils.FileExists(authJsonPath) {
		return fmt.Errorf("Codex auth.json 文件不存在")
	}

	// 读取 auth.json
	var authData config.CodexAuthJson
	if err := utils.ReadJSONFile(authJsonPath, &authData); err != nil {
		return fmt.Errorf("读取 auth.json 失败: %w", err)
	}

	// 读取 config.toml (可选)
	var configContent string
	if utils.FileExists(configPath) {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("读取 config.toml 失败: %w", err)
		}
		configContent = string(data)
	}

	// 验证配置
	apiKey := authData.OpenAIAPIKey
	if apiKey == "" {
		return fmt.Errorf("Codex 配置缺少 API Key")
	}

	// 从 config.toml 中提取 base_url (如果存在)
	baseURL := ""
	if configContent != "" {
		re := regexp.MustCompile(`base_url\s*=\s*"([^"]+)"`)
		if matches := re.FindStringSubmatch(configContent); len(matches) > 1 {
			baseURL = matches[1]
		}
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
	if err := manager.AddProviderForApp("codex", name, "", apiKey, baseURL, "imported", ""); err != nil {
		return fmt.Errorf("添加 Codex 配置失败: %w", err)
	}

	fmt.Printf("成功导入 Codex 配置 '%s':\n", name)
	fmt.Printf("  Token: %s\n", config.MaskToken(apiKey))
	if baseURL != "" {
		fmt.Printf("  URL: %s\n", baseURL)
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
					"backup",
					"") // defaultSonnetModel

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

	importCmd.Flags().Bool("from-live", false, "从 live 配置导入")
	importCmd.Flags().String("from-file", "", "从指定文件导入")
	importCmd.Flags().Bool("scan", false, "扫描并导入所有副本文件")
	importCmd.Flags().String("app", "claude", "应用名称 (claude 或 codex)")
	importCmd.Flags().String("name", "", "导入后的配置名称")
}