package config

import (
	"runtime"
	"strings"
	"testing"
)

func TestGenerateGeminiEnvExport_NoBlankLines(t *testing.T) {
	provider := &Provider{
		SettingsConfig: map[string]interface{}{
			"env": map[string]interface{}{
				"GOOGLE_GEMINI_BASE_URL": "https://example.com",
				"GEMINI_API_KEY":         "test-key",
				"GEMINI_MODEL":           "gemini-pro",
			},
		},
	}

	script, err := GenerateGeminiEnvExport(provider, "test-config", false)
	if err != nil {
		t.Fatalf("GenerateGeminiEnvExport returned error: %v", err)
	}

	if script == "" {
		t.Fatalf("expected script output, got empty string")
	}

	if strings.Contains(script, "\n\n") {
		t.Fatalf("script contains blank lines: %q", script)
	}

	lines := strings.Split(strings.TrimSuffix(script, "\n"), "\n")

	if len(lines) < 2 {
		t.Fatalf("unexpected line count: %d", len(lines))
	}

	switch runtime.GOOS {
	case "windows":
		if !strings.HasPrefix(lines[0], "$env:GOOGLE_GEMINI_BASE_URL=") {
			t.Fatalf("unexpected Windows export syntax: %s", lines[0])
		}
	default:
		if !strings.HasPrefix(lines[0], "export GOOGLE_GEMINI_BASE_URL=") {
			t.Fatalf("unexpected Unix export syntax: %s", lines[0])
		}
	}

	if !strings.HasPrefix(lines[len(lines)-2], "# ") {
		t.Fatalf("expected comment line near end, got: %s", lines[len(lines)-2])
	}

	if !strings.HasPrefix(lines[len(lines)-1], "#   ") {
		t.Fatalf("expected command hint line at end, got: %s", lines[len(lines)-1])
	}
}
