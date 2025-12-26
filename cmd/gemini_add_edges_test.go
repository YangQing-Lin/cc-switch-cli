package cmd

import (
	"strings"
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
)

func TestGeminiAddCmd_Edges(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	t.Run("api-key mode trims and rejects empty api key", func(t *testing.T) {
		resetFlags(geminiAddCmd)
		if err := geminiAddCmd.Flags().Set("auth-type", "api-key"); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		if err := geminiAddCmd.Flags().Set("apikey", "   "); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		if err := geminiAddCmd.Flags().Set("base-url", "https://api.example.com"); err != nil {
			t.Fatalf("set flag: %v", err)
		}
		err := geminiAddCmd.RunE(geminiAddCmd, []string{"g1"})
		if err == nil || !strings.Contains(err.Error(), "GEMINI_API_KEY") {
			t.Fatalf("expected api key error, got: %v", err)
		}
	})

	t.Run("oauth mode succeeds without api key", func(t *testing.T) {
		resetFlags(geminiAddCmd)
		if err := geminiAddCmd.Flags().Set("auth-type", "oauth"); err != nil {
			t.Fatalf("set flag: %v", err)
		}

		stdout, _ := testutil.CaptureOutput(t, func() {
			if err := geminiAddCmd.RunE(geminiAddCmd, []string{"google"}); err != nil {
				t.Fatalf("gemini add oauth: %v", err)
			}
		})
		if !strings.Contains(stdout, "OAuth") {
			t.Fatalf("expected oauth output, got: %s", stdout)
		}
	})
}
