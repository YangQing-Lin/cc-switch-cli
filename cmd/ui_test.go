package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
)

func TestUICmd(t *testing.T) {
	t.Run("runs tui via tuiRunner", func(t *testing.T) {
		resetGlobals()
		withTempHome(t)
		noLock = true

		called := false
		orig := tuiRunner
		tuiRunner = func(_ *config.Manager) error {
			called = true
			return nil
		}
		t.Cleanup(func() { tuiRunner = orig })

		if err := uiCmd.RunE(uiCmd, []string{}); err != nil {
			t.Fatalf("ui cmd: %v", err)
		}
		if !called {
			t.Fatalf("expected tuiRunner called")
		}
	})

	t.Run("config dir error", func(t *testing.T) {
		resetGlobals()
		tmpHome := withTempHome(t)

		// 指向一个文件（不是目录），触发 NewManagerWithDir 失败
		configDir = tmpHome + "/not-a-dir"
		if err := os.WriteFile(configDir, []byte("x"), 0644); err != nil {
			t.Fatalf("write file: %v", err)
		}

		err := uiCmd.RunE(uiCmd, []string{})
		if err == nil || !strings.Contains(err.Error(), "初始化配置管理器失败") {
			t.Fatalf("expected init error, got: %v", err)
		}
	})
}
