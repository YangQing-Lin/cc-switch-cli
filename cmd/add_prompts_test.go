package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
)

func TestAddCmd_Prompts(t *testing.T) {
	t.Run("fallback to plaintext when ReadPassword fails", func(t *testing.T) {
		resetGlobals()
		withTempHome(t)

		orig := readPassword
		readPassword = func(_ int) ([]byte, error) { return nil, errors.New("no tty") }
		t.Cleanup(func() { readPassword = orig })

		resetFlags(addCmd)
		withStdin(t, "sk-fallback\nhttps://api.example.com\n", func() {
			stdout, _ := testutil.CaptureOutput(t, func() {
				if err := addCmd.RunE(addCmd, []string{"cfg"}); err != nil {
					t.Fatalf("add: %v", err)
				}
			})
			if !strings.Contains(stdout, "已添加到") {
				t.Fatalf("expected success output, got: %s", stdout)
			}
		})
	})

	t.Run("uses ReadPassword when available", func(t *testing.T) {
		resetGlobals()
		withTempHome(t)

		orig := readPassword
		readPassword = func(_ int) ([]byte, error) { return []byte("sk-term"), nil }
		t.Cleanup(func() { readPassword = orig })

		resetFlags(addCmd)
		withStdin(t, "https://api.example.com\n", func() {
			stdout, _ := testutil.CaptureOutput(t, func() {
				if err := addCmd.RunE(addCmd, []string{"cfg2"}); err != nil {
					t.Fatalf("add: %v", err)
				}
			})
			if !strings.Contains(stdout, "cfg2") {
				t.Fatalf("expected output contains name, got: %s", stdout)
			}
		})
	})
}
