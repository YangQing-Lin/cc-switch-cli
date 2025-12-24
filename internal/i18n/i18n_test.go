package i18n

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name           string
		settingsBody   string
		wantLang       string
		shouldWriteCfg bool
	}{
		{
			name:           "load english settings",
			settingsBody:   `{"language":"en"}`,
			wantLang:       "en",
			shouldWriteCfg: true,
		},
		{
			name:           "invalid language fallback",
			settingsBody:   `{"language":"fr"}`,
			wantLang:       "zh",
			shouldWriteCfg: true,
		},
		{
			name:           "corrupted settings fallback",
			settingsBody:   "{invalid",
			wantLang:       "zh",
			shouldWriteCfg: true,
		},
		{
			name:           "missing settings fallback",
			wantLang:       "zh",
			shouldWriteCfg: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.WithTempHome(t, func(home string) {
				currentLanguage = "zh"
				t.Cleanup(func() {
					currentLanguage = "zh"
				})

				if tt.shouldWriteCfg {
					settingsDir := filepath.Join(home, ".cc-switch")
					if err := os.MkdirAll(settingsDir, 0755); err != nil {
						t.Fatalf("创建目录失败: %v", err)
					}
					if err := os.WriteFile(filepath.Join(settingsDir, "settings.json"), []byte(tt.settingsBody), 0600); err != nil {
						t.Fatalf("写入设置失败: %v", err)
					}
				}

				if err := Init(); err != nil {
					t.Fatalf("Init() error = %v", err)
				}
				if GetLanguage() != tt.wantLang {
					t.Fatalf("语言不匹配: %s != %s", GetLanguage(), tt.wantLang)
				}
			})
		})
	}
}

func TestSetLanguage(t *testing.T) {
	tests := []struct {
		name    string
		lang    string
		want    string
		initial string
	}{
		{"set english", "en", "en", "zh"},
		{"set chinese", "zh", "zh", "en"},
		{"invalid keeps", "fr", "zh", "zh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currentLanguage = tt.initial
			t.Cleanup(func() {
				currentLanguage = "zh"
			})

			SetLanguage(tt.lang)
			if got := GetLanguage(); got != tt.want {
				t.Fatalf("GetLanguage() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestTranslate(t *testing.T) {
	tests := []struct {
		name string
		lang string
		key  string
		args []interface{}
		want string
	}{
		{
			name: "chinese translation",
			lang: "zh",
			key:  "success",
			want: "成功",
		},
		{
			name: "english translation",
			lang: "en",
			key:  "success",
			want: "Success",
		},
		{
			name: "missing key returns key",
			lang: "zh",
			key:  "missing.key",
			want: "missing.key",
		},
		{
			name: "fallback to chinese",
			lang: "invalid",
			key:  "success",
			want: "成功",
		},
		{
			name: "formatted message",
			lang: "en",
			key:  "confirm.delete_provider_message",
			args: []interface{}{"demo"},
			want: "Are you sure you want to delete configuration 'demo'?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.lang == "invalid" {
				currentLanguage = "invalid"
			} else {
				currentLanguage = "zh"
				SetLanguage(tt.lang)
			}
			t.Cleanup(func() {
				currentLanguage = "zh"
			})

			got := T(tt.key, tt.args...)
			if got != tt.want {
				t.Fatalf("T(%s) = %s, want %s", tt.key, got, tt.want)
			}
			if Tf(tt.key, tt.args...) != got {
				t.Fatalf("Tf() 不匹配")
			}
		})
	}
}

func TestMessagesCoverage(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
	}{
		{"zh -> en", "zh", "en"},
		{"en -> zh", "en", "zh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for key := range messages[tt.from] {
				if _, ok := messages[tt.to][key]; !ok {
					t.Fatalf("缺少翻译 key: %s", key)
				}
			}
		})
	}
}
