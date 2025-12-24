package cmd

import (
	"strings"
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
	"github.com/YangQing-Lin/cc-switch-cli/internal/version"
)

func TestVersionCommandOutput(t *testing.T) {
	resetGlobals()
	stdout, _ := testutil.CaptureOutput(t, func() {
		versionCmd.Run(versionCmd, []string{})
	})
	if !strings.Contains(stdout, version.GetVersion()) {
		t.Fatalf("expected version output, got: %s", stdout)
	}
}

func TestCheckUpdateCommandWithStub(t *testing.T) {
	resetGlobals()
	origCheckForUpdate := checkForUpdateFunc
	checkForUpdateFunc = func() (*version.ReleaseInfo, bool, error) {
		return &version.ReleaseInfo{
			TagName:     "v99.0.0",
			HTMLURL:     "https://example.com",
			PublishedAt: "2025-01-01",
			Body:        "notes",
			Assets:      nil,
		}, true, nil
	}
	t.Cleanup(func() { checkForUpdateFunc = origCheckForUpdate })

	stdout, _ := testutil.CaptureOutput(t, func() {
		checkUpdateCmd.Run(checkUpdateCmd, []string{})
	})
	if !strings.Contains(stdout, "发现新版本") {
		t.Fatalf("expected update output, got: %s", stdout)
	}
}

func TestUpdateSelfFromFileMissing(t *testing.T) {
	resetGlobals()
	resetFlags(updateSelfCmd)
	if err := updateSelfCmd.Flags().Set("from-file", "/non-existent-file"); err != nil {
		t.Fatalf("set from-file: %v", err)
	}
	stdout, _ := testutil.CaptureOutput(t, func() {
		updateSelfCmd.Run(updateSelfCmd, []string{})
	})
	if !strings.Contains(stdout, "文件不存在") {
		t.Fatalf("expected missing file output, got: %s", stdout)
	}
}
