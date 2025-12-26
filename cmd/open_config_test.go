package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
)

func TestRunOpenConfig_UnsupportedOS(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	orig := openConfigGOOS
	openConfigGOOS = "plan9"
	t.Cleanup(func() { openConfigGOOS = orig })

	if err := runOpenConfig(); err == nil {
		t.Fatalf("expected error for unsupported os")
	}
}

func TestRunOpenConfig_StartError(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	origExec := openConfigExecCommand
	openConfigExecCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("definitely-not-exist-ccs-open-config")
	}
	t.Cleanup(func() {
		openConfigExecCommand = origExec
	})

	for _, goos := range []string{"linux", "darwin", "windows"} {
		origGOOS := openConfigGOOS
		openConfigGOOS = goos
		err := runOpenConfig()
		openConfigGOOS = origGOOS
		if err == nil || !strings.Contains(err.Error(), "打开文件管理器失败") {
			t.Fatalf("expected start error for %s, got: %v", goos, err)
		}
	}
}

func TestRunOpenConfig_Success(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	origExec := openConfigExecCommand
	openConfigExecCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("true")
	}
	t.Cleanup(func() { openConfigExecCommand = origExec })

	for _, goos := range []string{"linux", "darwin", "windows"} {
		origGOOS := openConfigGOOS
		openConfigGOOS = goos

		stdout, _ := testutil.CaptureOutput(t, func() {
			if err := runOpenConfig(); err != nil {
				t.Fatalf("unexpected error for %s: %v", goos, err)
			}
		})

		openConfigGOOS = origGOOS

		if !strings.Contains(stdout, "已在文件管理器中打开配置目录") {
			t.Fatalf("expected success output for %s, got: %s", goos, stdout)
		}
	}
}
