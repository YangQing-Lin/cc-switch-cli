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
		"success":        "Success",
		"failed":         "Failed",
		"error":          "Error",
		"warning":        "Warning",

		// Provider operations
		"provider_added":     "Provider added successfully",
		"provider_updated":   "Provider updated successfully",
		"provider_deleted":   "Provider deleted successfully",
		"provider_switched":  "Switched to provider",
		"provider_not_found": "Provider not found",

		// Config operations
		"config_imported":   "Configuration imported successfully",
		"config_exported":   "Configuration exported successfully",
		"config_backed_up":  "Configuration backed up successfully",
		"config_restored":   "Configuration restored successfully",

		// Validation
		"validation_success": "Configuration validation passed",
		"validation_failed":  "Configuration validation failed",

		// File operations
		"file_not_found":    "File not found",
		"directory_created": "Directory created",
		"directory_opened":  "Opened configuration directory in file manager",
	},
	"zh": {
		// Common
		"success":        "成功",
		"failed":         "失败",
		"error":          "错误",
		"warning":        "警告",

		// Provider operations
		"provider_added":     "配置添加成功",
		"provider_updated":   "配置更新成功",
		"provider_deleted":   "配置删除成功",
		"provider_switched":  "已切换到配置",
		"provider_not_found": "未找到配置",

		// Config operations
		"config_imported":   "配置导入成功",
		"config_exported":   "配置导出成功",
		"config_backed_up":  "配置备份成功",
		"config_restored":   "配置恢复成功",

		// Validation
		"validation_success": "配置验证通过",
		"validation_failed":  "配置验证失败",

		// File operations
		"file_not_found":    "文件未找到",
		"directory_created": "目录已创建",
		"directory_opened":  "已在文件管理器中打开配置目录",
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
