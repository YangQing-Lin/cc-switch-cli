package i18n

import (
	"testing"
)

func TestSetLanguage(t *testing.T) {
	tests := []struct {
		name string
		lang string
		want string
	}{
		{
			name: "设置英文",
			lang: "en",
			want: "en",
		},
		{
			name: "设置中文",
			lang: "zh",
			want: "zh",
		},
		{
			name: "设置无效语言",
			lang: "fr",
			want: "zh", // 应保持不变
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 重置为默认语言
			currentLanguage = "zh"

			SetLanguage(tt.lang)
			got := GetLanguage()

			if got != tt.want {
				t.Errorf("GetLanguage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetLanguage(t *testing.T) {
	// 重置为默认
	currentLanguage = "zh"

	if got := GetLanguage(); got != "zh" {
		t.Errorf("GetLanguage() = %v, want %v", got, "zh")
	}

	SetLanguage("en")
	if got := GetLanguage(); got != "en" {
		t.Errorf("GetLanguage() after SetLanguage('en') = %v, want %v", got, "en")
	}
}

func TestT(t *testing.T) {
	tests := []struct {
		name string
		lang string
		key  string
		args []interface{}
		want string
	}{
		{
			name: "中文翻译",
			lang: "zh",
			key:  "success",
			want: "成功",
		},
		{
			name: "英文翻译",
			lang: "en",
			key:  "success",
			want: "Success",
		},
		{
			name: "不存在的key",
			lang: "zh",
			key:  "nonexistent_key",
			want: "nonexistent_key", // 返回key本身
		},
		{
			name: "不带参数的翻译",
			lang: "zh",
			key:  "provider_added",
			want: "配置添加成功",
		},
		{
			name: "无效语言降级到中文",
			lang: "invalid",
			key:  "success",
			want: "成功", // 降级到中文
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 手动设置语言（包括无效语言）
			if tt.lang == "invalid" {
				currentLanguage = "invalid"
			} else {
				SetLanguage(tt.lang)
			}

			var got string
			if len(tt.args) > 0 {
				got = T(tt.key, tt.args...)
			} else {
				got = T(tt.key)
			}

			if got != tt.want {
				t.Errorf("T(%v) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestTf(t *testing.T) {
	// 重置语言
	currentLanguage = "zh"

	// Tf 应该和 T 功能相同
	result := Tf("success")
	expected := T("success")

	if result != expected {
		t.Errorf("Tf('success') = %v, want %v", result, expected)
	}
}

func TestMessages(t *testing.T) {
	// 验证所有中文key都有对应的英文翻译
	zhKeys := make(map[string]bool)
	for key := range messages["zh"] {
		zhKeys[key] = true
	}

	for key := range messages["en"] {
		if !zhKeys[key] {
			t.Errorf("英文翻译key %v 在中文中不存在", key)
		}
	}

	// 验证所有英文key都有对应的中文翻译
	for key := range messages["zh"] {
		if _, ok := messages["en"][key]; !ok {
			t.Errorf("中文翻译key %v 在英文中不存在", key)
		}
	}
}
