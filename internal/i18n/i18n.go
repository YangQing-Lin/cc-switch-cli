package i18n

import (
	"fmt"

	"github.com/YangQing-Lin/cc-switch-cli/internal/settings"
)

var currentLanguage = "zh" // 默认中文

// Message 多语言消息定义
var messages = map[string]map[string]string{
	"en": {
		// Common
		"success": "Success",
		"failed":  "Failed",
		"error":   "Error",
		"warning": "Warning",

		// Provider operations
		"provider_added":     "Provider added successfully",
		"provider_updated":   "Provider updated successfully",
		"provider_deleted":   "Provider deleted successfully",
		"provider_switched":  "Switched to provider",
		"provider_not_found": "Provider not found",

		// Config operations
		"config_imported":  "Configuration imported successfully",
		"config_exported":  "Configuration exported successfully",
		"config_backed_up": "Configuration backed up successfully",
		"config_restored":  "Configuration restored successfully",

		// Validation
		"validation_success": "Configuration validation passed",
		"validation_failed":  "Configuration validation failed",

		// File operations
		"file_not_found":    "File not found",
		"directory_created": "Directory created",
		"directory_opened":  "Opened configuration directory in file manager",

		// TUI specific
		"error.cannot_delete_current":     "Cannot delete currently active configuration",
		"error.switch_failed":             "Failed to switch configuration",
		"error.name_required":             "Configuration name is required",
		"error.token_required":            "API Token is required",
		"error.base_url_required":         "API Base URL is required",
		"error.invalid_config":            "Invalid configuration",
		"error.update_failed":             "Failed to update configuration",
		"error.add_failed":                "Failed to add configuration",
		"error.delete_failed":             "Failed to delete configuration",
		"error.readonly_field":            "This field is read-only, use → to select from predefined options",
		"success.switched_to":             "Switched to",
		"success.provider_updated":        "Configuration updated successfully",
		"success.provider_added":          "Configuration added successfully",
		"success.provider_deleted":        "Configuration deleted",
		"warning.apply_vscode_failed":     "Failed to apply to VSCode",
		"confirm.delete_provider_message": "Are you sure you want to delete configuration '%s'?",
	},
	"zh": {
		// Common
		"success": "成功",
		"failed":  "失败",
		"error":   "错误",
		"warning": "警告",

		// Provider operations
		"provider_added":     "配置添加成功",
		"provider_updated":   "配置更新成功",
		"provider_deleted":   "配置删除成功",
		"provider_switched":  "已切换到配置",
		"provider_not_found": "未找到配置",

		// Config operations
		"config_imported":  "配置导入成功",
		"config_exported":  "配置导出成功",
		"config_backed_up": "配置备份成功",
		"config_restored":  "配置恢复成功",

		// Validation
		"validation_success": "配置验证通过",
		"validation_failed":  "配置验证失败",

		// File operations
		"file_not_found":    "文件未找到",
		"directory_created": "目录已创建",
		"directory_opened":  "已在文件管理器中打开配置目录",

		// TUI specific
		"error.cannot_delete_current":     "无法删除当前激活的配置",
		"error.switch_failed":             "切换配置失败",
		"error.name_required":             "配置名称不能为空",
		"error.token_required":            "API Token 不能为空",
		"error.base_url_required":         "API 基础 URL 不能为空",
		"error.invalid_config":            "配置格式无效",
		"error.update_failed":             "更新配置失败",
		"error.add_failed":                "添加配置失败",
		"error.delete_failed":             "删除配置失败",
		"error.readonly_field":            "此字段为只读，请使用 → 键从预定义选项中选择",
		"success.switched_to":             "已切换到",
		"success.provider_updated":        "配置更新成功",
		"success.provider_added":          "配置添加成功",
		"success.provider_deleted":        "配置已删除",
		"warning.apply_vscode_failed":     "应用到 VSCode 失败",
		"confirm.delete_provider_message": "确定要删除配置 '%s' 吗？",
	},
}

// Init 初始化语言设置
func Init() error {
	manager, err := settings.NewManager()
	if err != nil {
		// 如果加载设置失败，使用默认语言
		return nil
	}

	lang := manager.GetLanguage()
	if lang == "en" || lang == "zh" {
		currentLanguage = lang
	}

	return nil
}

// SetLanguage 设置当前语言
func SetLanguage(lang string) {
	if lang == "en" || lang == "zh" {
		currentLanguage = lang
	}
}

// GetLanguage 获取当前语言
func GetLanguage() string {
	return currentLanguage
}

// T 翻译消息 (Translation)
func T(key string, args ...interface{}) string {
	langMessages, ok := messages[currentLanguage]
	if !ok {
		langMessages = messages["zh"] // 降级到中文
	}

	msg, ok := langMessages[key]
	if !ok {
		return key // 如果找不到翻译，返回 key 本身
	}

	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}

	return msg
}

// Tf 翻译消息并格式化 (Translation with format)
func Tf(key string, args ...interface{}) string {
	return T(key, args...)
}
