package cmd

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/spf13/cobra"
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "验证配置文件的完整性和有效性",
	Long: `验证配置文件的完整性和有效性。

检查项目包括:
  - 配置文件格式
  - 必需字段是否存在
  - API Token 格式
  - Base URL 格式
  - ID 唯一性
  - 名称唯一性
  - Current 引用有效性

示例:
  cc-switch validate                   # 验证所有配置
  cc-switch validate --app claude      # 只验证 Claude 配置
  cc-switch validate --provider name   # 验证特定配置
  cc-switch validate --fix             # 尝试修复发现的问题`,
	RunE: func(cmd *cobra.Command, args []string) error {
		appName, _ := cmd.Flags().GetString("app")
		providerName, _ := cmd.Flags().GetString("provider")
		fix, _ := cmd.Flags().GetBool("fix")
		verbose, _ := cmd.Flags().GetBool("verbose")

		// 创建管理器
		manager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("初始化配置管理器失败: %w", err)
		}

		// 获取完整配置
		fullConfig, err := manager.GetConfig()
		if err != nil {
			return fmt.Errorf("获取配置失败: %w", err)
		}

		// 验证结果
		issues := []ValidationIssue{}
		warnings := []ValidationIssue{}

		// 基础验证
		if fullConfig.Version != 2 {
			issues = append(issues, ValidationIssue{
				Level:   "ERROR",
				Message: fmt.Sprintf("配置版本不正确: 期望 2, 实际 %d", fullConfig.Version),
			})
		}

		if fullConfig.Apps == nil || len(fullConfig.Apps) == 0 {
			warnings = append(warnings, ValidationIssue{
				Level:   "WARNING",
				Message: "配置文件中没有任何应用",
			})
		}

		// 验证每个应用
		appsToValidate := []string{}
		if appName != "" {
			appsToValidate = append(appsToValidate, appName)
		} else {
			for app := range fullConfig.Apps {
				appsToValidate = append(appsToValidate, app)
			}
		}

		for _, app := range appsToValidate {
			appConfig, exists := fullConfig.Apps[app]
			if !exists {
				issues = append(issues, ValidationIssue{
					Level:   "ERROR",
					App:     app,
					Message: fmt.Sprintf("应用 '%s' 不存在", app),
				})
				continue
			}

			// 验证应用配置
			appIssues, appWarnings := validateApp(app, appConfig, providerName)
			issues = append(issues, appIssues...)
			warnings = append(warnings, appWarnings...)
		}

		// 显示结果
		totalIssues := len(issues) + len(warnings)

		if totalIssues == 0 {
			fmt.Println("✓ 配置验证通过，未发现问题")
			return nil
		}

		// 显示错误
		if len(issues) > 0 {
			fmt.Printf("\n发现 %d 个错误:\n", len(issues))
			fmt.Println("─────────────")
			for i, issue := range issues {
				displayIssue(i+1, issue, verbose)
			}
		}

		// 显示警告
		if len(warnings) > 0 {
			fmt.Printf("\n发现 %d 个警告:\n", len(warnings))
			fmt.Println("─────────────")
			for i, warning := range warnings {
				displayIssue(i+1, warning, verbose)
			}
		}

		// 尝试修复
		if fix && len(issues) > 0 {
			fmt.Println("\n尝试修复问题...")
			fixedCount := attemptFixes(manager, issues)
			if fixedCount > 0 {
				fmt.Printf("✓ 已修复 %d 个问题\n", fixedCount)
			} else {
				fmt.Println("✗ 无法自动修复任何问题")
			}
		}

		// 如果有错误，返回非零退出码
		if len(issues) > 0 {
			return fmt.Errorf("配置验证失败，发现 %d 个错误", len(issues))
		}

		return nil
	},
}

// ValidationIssue 验证问题
type ValidationIssue struct {
	Level      string // ERROR, WARNING
	App        string
	Provider   string
	Message    string
	FixableMsg string // 可修复的提示
}

// validateApp 验证应用配置
func validateApp(appName string, appConfig config.ProviderManager, filterProvider string) ([]ValidationIssue, []ValidationIssue) {
	issues := []ValidationIssue{}
	warnings := []ValidationIssue{}

	// 检查是否有供应商
	if len(appConfig.Providers) == 0 {
		warnings = append(warnings, ValidationIssue{
			Level:   "WARNING",
			App:     appName,
			Message: fmt.Sprintf("应用 '%s' 没有任何供应商配置", appName),
		})
		return issues, warnings
	}

	// 检查 current 引用
	if appConfig.Current != "" {
		if _, exists := appConfig.Providers[appConfig.Current]; !exists {
			issues = append(issues, ValidationIssue{
				Level:      "ERROR",
				App:        appName,
				Message:    fmt.Sprintf("Current 引用了不存在的供应商 ID: %s", appConfig.Current),
				FixableMsg: "可以通过 --fix 清除无效引用",
			})
		}
	}

	// 检查每个供应商
	nameMap := make(map[string]string) // name -> id
	for id, provider := range appConfig.Providers {
		// 过滤
		if filterProvider != "" && provider.Name != filterProvider {
			continue
		}

		// 检查 ID
		if id == "" {
			issues = append(issues, ValidationIssue{
				Level:    "ERROR",
				App:      appName,
				Provider: provider.Name,
				Message:  "供应商缺少 ID",
			})
		}

		// 检查名称
		if provider.Name == "" {
			issues = append(issues, ValidationIssue{
				Level:    "ERROR",
				App:      appName,
				Provider: id,
				Message:  "供应商缺少名称",
			})
		} else {
			// 检查名称唯一性
			if existingID, exists := nameMap[provider.Name]; exists {
				issues = append(issues, ValidationIssue{
					Level:    "ERROR",
					App:      appName,
					Provider: provider.Name,
					Message:  fmt.Sprintf("名称 '%s' 重复 (ID: %s 和 %s)", provider.Name, id, existingID),
				})
			}
			nameMap[provider.Name] = id
		}

		// 根据应用类型验证配置
		switch appName {
		case "claude":
			validateClaudeProvider(appName, provider, &issues, &warnings)
		case "codex":
			validateCodexProvider(appName, provider, &issues, &warnings)
		}
	}

	return issues, warnings
}

// validateClaudeProvider 验证 Claude 供应商配置
func validateClaudeProvider(appName string, provider config.Provider, issues, warnings *[]ValidationIssue) {
	// 检查 settingsConfig
	if provider.SettingsConfig == nil {
		*issues = append(*issues, ValidationIssue{
			Level:    "ERROR",
			App:      appName,
			Provider: provider.Name,
			Message:  "缺少 settingsConfig",
		})
		return
	}

	// 检查 env
	envMap, ok := provider.SettingsConfig["env"].(map[string]interface{})
	if !ok {
		*issues = append(*issues, ValidationIssue{
			Level:    "ERROR",
			App:      appName,
			Provider: provider.Name,
			Message:  "settingsConfig 缺少 env 字段",
		})
		return
	}

	// 检查 API Token
	token, ok := envMap["ANTHROPIC_AUTH_TOKEN"].(string)
	if !ok || token == "" {
		*issues = append(*issues, ValidationIssue{
			Level:    "ERROR",
			App:      appName,
			Provider: provider.Name,
			Message:  "缺少 ANTHROPIC_AUTH_TOKEN",
		})
	} else {
		// 验证 Token 格式
		if !strings.HasPrefix(token, "sk-") && !strings.HasPrefix(token, "88_") {
			*warnings = append(*warnings, ValidationIssue{
				Level:    "WARNING",
				App:      appName,
				Provider: provider.Name,
				Message:  fmt.Sprintf("API Token 格式可能不正确: %s", config.MaskToken(token)),
			})
		}
	}

	// 检查 Base URL
	baseURL, ok := envMap["ANTHROPIC_BASE_URL"].(string)
	if !ok || baseURL == "" {
		*warnings = append(*warnings, ValidationIssue{
			Level:      "WARNING",
			App:        appName,
			Provider:   provider.Name,
			Message:    "缺少 ANTHROPIC_BASE_URL",
			FixableMsg: "可以通过 --fix 设置默认值",
		})
	} else {
		// 验证 URL 格式
		if _, err := url.Parse(baseURL); err != nil {
			*issues = append(*issues, ValidationIssue{
				Level:    "ERROR",
				App:      appName,
				Provider: provider.Name,
				Message:  fmt.Sprintf("无效的 Base URL: %s", baseURL),
			})
		}
	}

	// 检查可选字段
	if model, ok := envMap["CLAUDE_CODE_MODEL"].(string); ok && model != "" {
		// 验证模型名称
		validModels := []string{"opus", "sonnet", "haiku", "claude-3-opus", "claude-3-sonnet", "claude-3-haiku"}
		valid := false
		for _, validModel := range validModels {
			if strings.Contains(strings.ToLower(model), validModel) {
				valid = true
				break
			}
		}
		if !valid {
			*warnings = append(*warnings, ValidationIssue{
				Level:    "WARNING",
				App:      appName,
				Provider: provider.Name,
				Message:  fmt.Sprintf("模型名称可能不正确: %s", model),
			})
		}
	}
}

// validateCodexProvider 验证 Codex 供应商配置
func validateCodexProvider(appName string, provider config.Provider, issues, warnings *[]ValidationIssue) {
	// TODO: 实现 Codex 验证
}

// displayIssue 显示问题
func displayIssue(index int, issue ValidationIssue, verbose bool) {
	prefix := "  "
	if issue.Level == "ERROR" {
		prefix = "✗ "
	} else if issue.Level == "WARNING" {
		prefix = "⚠ "
	}

	location := ""
	if issue.App != "" {
		location = fmt.Sprintf("[%s", issue.App)
		if issue.Provider != "" {
			location += fmt.Sprintf("/%s", issue.Provider)
		}
		location += "] "
	}

	fmt.Printf("%s%d. %s%s\n", prefix, index, location, issue.Message)

	if verbose && issue.FixableMsg != "" {
		fmt.Printf("     💡 %s\n", issue.FixableMsg)
	}
}

// attemptFixes 尝试修复问题
func attemptFixes(manager *config.Manager, issues []ValidationIssue) int {
	fixedCount := 0

	// TODO: 实现自动修复逻辑
	// - 清除无效的 current 引用
	// - 添加缺少的默认值
	// - 生成缺失的 ID

	return fixedCount
}

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().String("app", "", "只验证指定应用")
	validateCmd.Flags().String("provider", "", "只验证指定供应商")
	validateCmd.Flags().Bool("fix", false, "尝试自动修复发现的问题")
	validateCmd.Flags().BoolP("verbose", "v", false, "显示详细信息")
}
