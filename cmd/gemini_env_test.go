package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
)

func TestGeminiEnvCmd(t *testing.T) {
	resetGlobals()
	home := withTempHome(t)

	manager := setupManager(t)
	if err := manager.AddGeminiProvider("g1", "https://api.example.com", "sk-gemini", "gemini-2.5-pro", config.GeminiAuthAPIKey); err != nil {
		t.Fatalf("add gemini provider: %v", err)
	}

	geminiDir := filepath.Join(home, ".gemini")
	if err := os.MkdirAll(geminiDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(geminiDir, ".env"), []byte("GEMINI_API_KEY=sk-gemini\n"), 0644); err != nil {
		t.Fatalf("write env: %v", err)
	}
	if err := os.WriteFile(filepath.Join(geminiDir, "settings.json"), []byte(""), 0644); err != nil {
		t.Fatalf("write settings: %v", err)
	}

	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := geminiEnvCmd.RunE(geminiEnvCmd, []string{}); err != nil {
			t.Fatalf("gemini env: %v", err)
		}
	})
	if !strings.Contains(stdout, "Gemini CLI 配置文件") || !strings.Contains(stdout, "当前选中配置") {
		t.Fatalf("unexpected output: %s", stdout)
	}
	if !strings.Contains(stdout, "GEMINI_API_KEY") || !strings.Contains(stdout, "(空文件)") {
		t.Fatalf("expected env content and empty settings marker, got: %s", stdout)
	}
}

func TestGeminiEnvCmd_MissingAndReadError(t *testing.T) {
	t.Run("missing files", func(t *testing.T) {
		resetGlobals()
		withTempHome(t)

		stdout, _ := testutil.CaptureOutput(t, func() {
			if err := geminiEnvCmd.RunE(geminiEnvCmd, []string{}); err != nil {
				t.Fatalf("gemini env: %v", err)
			}
		})
		if !strings.Contains(stdout, "状态: 不存在") {
			t.Fatalf("expected missing marker, got: %s", stdout)
		}
	})

	t.Run("read error", func(t *testing.T) {
		resetGlobals()
		home := withTempHome(t)

		geminiDir := filepath.Join(home, ".gemini")
		if err := os.MkdirAll(filepath.Join(geminiDir, ".env"), 0755); err != nil {
			t.Fatalf("mkdir env dir: %v", err)
		}

		stdout, _ := testutil.CaptureOutput(t, func() {
			if err := geminiEnvCmd.RunE(geminiEnvCmd, []string{}); err != nil {
				t.Fatalf("gemini env: %v", err)
			}
		})
		if !strings.Contains(stdout, "无法读取") {
			t.Fatalf("expected read error marker, got: %s", stdout)
		}
	})
}
