package cmd

import (
	"strings"
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
	"github.com/YangQing-Lin/cc-switch-cli/internal/version"
)

func TestVersionCmd_IncludesInjectedBuildInfo(t *testing.T) {
	resetGlobals()

	origDate := version.BuildDate
	origCommit := version.GitCommit
	version.BuildDate = "2025-12-26"
	version.GitCommit = "deadbeef"
	t.Cleanup(func() {
		version.BuildDate = origDate
		version.GitCommit = origCommit
	})

	stdout, _ := testutil.CaptureOutput(t, func() {
		versionCmd.Run(versionCmd, []string{})
	})
	if !strings.Contains(stdout, "构建日期") || !strings.Contains(stdout, "Git 提交") {
		t.Fatalf("expected build info output, got: %s", stdout)
	}
}
