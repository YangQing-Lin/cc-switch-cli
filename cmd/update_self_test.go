package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
	"github.com/YangQing-Lin/cc-switch-cli/internal/version"
)

func TestUpdateSelfCmd(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	t.Run("from-file install failure", func(t *testing.T) {
		resetFlags(updateSelfCmd)
		tmp := t.TempDir()
		fromFile := filepath.Join(tmp, "bin")
		if err := os.WriteFile(fromFile, []byte("bin"), 0644); err != nil {
			t.Fatalf("write file: %v", err)
		}

		orig := updateSelfInstallBinaryFunc
		updateSelfInstallBinaryFunc = func(_ string, _ bool) error { return errors.New("boom") }
		t.Cleanup(func() { updateSelfInstallBinaryFunc = orig })

		if err := updateSelfCmd.Flags().Set("from-file", fromFile); err != nil {
			t.Fatalf("set flag: %v", err)
		}

		stdout, _ := testutil.CaptureOutput(t, func() {
			updateSelfCmd.Run(updateSelfCmd, []string{})
		})
		if !strings.Contains(stdout, "安装失败") || !strings.Contains(stdout, "boom") {
			t.Fatalf("expected install failure output, got: %s", stdout)
		}
	})

	t.Run("from-file install success", func(t *testing.T) {
		resetFlags(updateSelfCmd)
		tmp := t.TempDir()
		fromFile := filepath.Join(tmp, "bin")
		if err := os.WriteFile(fromFile, []byte("bin"), 0644); err != nil {
			t.Fatalf("write file: %v", err)
		}

		called := false
		orig := updateSelfInstallBinaryFunc
		updateSelfInstallBinaryFunc = func(path string, force bool) error {
			called = true
			if path != fromFile || force {
				t.Fatalf("unexpected args: %s %v", path, force)
			}
			return nil
		}
		t.Cleanup(func() { updateSelfInstallBinaryFunc = orig })

		if err := updateSelfCmd.Flags().Set("from-file", fromFile); err != nil {
			t.Fatalf("set flag: %v", err)
		}

		stdout, _ := testutil.CaptureOutput(t, func() {
			updateSelfCmd.Run(updateSelfCmd, []string{})
		})
		if !called || !strings.Contains(stdout, "安装成功") {
			t.Fatalf("expected success output, got: %s", stdout)
		}
	})

	t.Run("online no update", func(t *testing.T) {
		resetFlags(updateSelfCmd)

		orig := updateSelfCheckForUpdateFunc
		updateSelfCheckForUpdateFunc = func() (*version.ReleaseInfo, bool, error) {
			return nil, false, nil
		}
		t.Cleanup(func() { updateSelfCheckForUpdateFunc = orig })

		stdout, _ := testutil.CaptureOutput(t, func() {
			updateSelfCmd.Run(updateSelfCmd, []string{})
		})
		if !strings.Contains(stdout, "无需更新") {
			t.Fatalf("expected no-update output, got: %s", stdout)
		}
	})

	t.Run("online check error", func(t *testing.T) {
		resetFlags(updateSelfCmd)

		orig := updateSelfCheckForUpdateFunc
		updateSelfCheckForUpdateFunc = func() (*version.ReleaseInfo, bool, error) {
			return nil, false, errors.New("nope")
		}
		t.Cleanup(func() { updateSelfCheckForUpdateFunc = orig })

		stdout, _ := testutil.CaptureOutput(t, func() {
			updateSelfCmd.Run(updateSelfCmd, []string{})
		})
		if !strings.Contains(stdout, "检查更新失败") || !strings.Contains(stdout, "nope") {
			t.Fatalf("expected check error output, got: %s", stdout)
		}
	})

	t.Run("online update download failure", func(t *testing.T) {
		resetFlags(updateSelfCmd)

		origCheck := updateSelfCheckForUpdateFunc
		origDownload := updateSelfDownloadUpdateFunc
		updateSelfCheckForUpdateFunc = func() (*version.ReleaseInfo, bool, error) {
			return &version.ReleaseInfo{TagName: "v99.0.0"}, true, nil
		}
		updateSelfDownloadUpdateFunc = func(_ *version.ReleaseInfo) error { return errors.New("download boom") }
		t.Cleanup(func() {
			updateSelfCheckForUpdateFunc = origCheck
			updateSelfDownloadUpdateFunc = origDownload
		})

		stdout, _ := testutil.CaptureOutput(t, func() {
			updateSelfCmd.Run(updateSelfCmd, []string{})
		})
		if !strings.Contains(stdout, "更新失败") || !strings.Contains(stdout, "download boom") {
			t.Fatalf("expected download failure output, got: %s", stdout)
		}
	})

	t.Run("online update success", func(t *testing.T) {
		resetFlags(updateSelfCmd)

		origCheck := updateSelfCheckForUpdateFunc
		origDownload := updateSelfDownloadUpdateFunc
		updateSelfCheckForUpdateFunc = func() (*version.ReleaseInfo, bool, error) {
			return &version.ReleaseInfo{TagName: "v99.0.0"}, true, nil
		}
		updateSelfDownloadUpdateFunc = func(_ *version.ReleaseInfo) error { return nil }
		t.Cleanup(func() {
			updateSelfCheckForUpdateFunc = origCheck
			updateSelfDownloadUpdateFunc = origDownload
		})

		stdout, _ := testutil.CaptureOutput(t, func() {
			updateSelfCmd.Run(updateSelfCmd, []string{})
		})
		if !strings.Contains(stdout, "更新成功") || !strings.Contains(stdout, "v99.0.0") {
			t.Fatalf("expected update success output, got: %s", stdout)
		}
	})
}
