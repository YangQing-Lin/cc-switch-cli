package main

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestMainExitCodes(t *testing.T) {
	if os.Getenv("CCS_HELPER_PROCESS") == "1" {
		args := strings.Fields(os.Getenv("CCS_ARGS"))
		os.Args = append([]string{"cc-switch"}, args...)
		main()
		return
	}

	runMainHelper(t, []string{"version"}, 0)
	runMainHelper(t, []string{"backup", "restore"}, 1)
}

func runMainHelper(t *testing.T, args []string, want int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestMainExitCodes", "--")
	cmd.Env = append(os.Environ(), "CCS_HELPER_PROCESS=1", "CCS_ARGS="+strings.Join(args, " "))
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("main helper timed out after 10s args=%v output: %s", args, output)
	}
	if want == 0 && err != nil {
		t.Fatalf("expected exit 0, got err %v output: %s", err, output)
	}
	if want != 0 {
		if err == nil {
			t.Fatalf("expected exit %d, got 0", want)
		}
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("expected ExitError, got %T", err)
		}
		if exitErr.ExitCode() != want {
			t.Fatalf("expected exit %d, got %d output: %s", want, exitErr.ExitCode(), output)
		}
	}
}
