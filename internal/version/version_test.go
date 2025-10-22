package version

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGetVersion(t *testing.T) {
	version := GetVersion()
	if version == "" {
		t.Error("Expected non-empty version string")
	}

	if version != Version {
		t.Errorf("Expected version %s, got %s", Version, version)
	}
}

func TestGetBuildDate(t *testing.T) {
	buildDate := GetBuildDate()
	if buildDate == "" {
		t.Error("Expected non-empty build date")
	}
}

func TestGetGitCommit(t *testing.T) {
	gitCommit := GetGitCommit()
	if gitCommit == "" {
		t.Error("Expected non-empty git commit")
	}
}

func TestGetArchiveNameForPlatform(t *testing.T) {
	tests := []struct {
		name     string
		tagName  string
		os       string
		arch     string
		expected string
	}{
		{
			name:     "Linux AMD64",
			tagName:  "v1.2.0",
			os:       "linux",
			arch:     "amd64",
			expected: "cc-switch-cli-1.2.0-linux-amd64.tar.gz",
		},
		{
			name:     "Windows AMD64",
			tagName:  "v1.2.0",
			os:       "windows",
			arch:     "amd64",
			expected: "cc-switch-cli-1.2.0-windows-amd64.zip",
		},
		{
			name:     "Darwin ARM64",
			tagName:  "v1.2.0",
			os:       "darwin",
			arch:     "arm64",
			expected: "cc-switch-cli-1.2.0-darwin-arm64.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getArchiveNameForPlatform(tt.tagName)

			if runtime.GOOS == "windows" {
				if !strings.HasSuffix(result, ".zip") {
					t.Errorf("Expected .zip extension for Windows, got: %s", result)
				}
			} else {
				if !strings.HasSuffix(result, ".tar.gz") {
					t.Errorf("Expected .tar.gz extension for non-Windows, got: %s", result)
				}
			}

			if !strings.Contains(result, strings.TrimPrefix(tt.tagName, "v")) {
				t.Errorf("Expected version in filename, got: %s", result)
			}
		})
	}
}

func TestExtractBinaryUnsupportedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	invalidArchive := filepath.Join(tmpDir, "test.tar")

	if err := os.WriteFile(invalidArchive, []byte("invalid"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err := extractBinary(invalidArchive, tmpDir)
	if err == nil {
		t.Error("Expected error for unsupported archive format")
	}

	if !strings.Contains(err.Error(), "不支持的压缩格式") {
		t.Errorf("Expected unsupported format error, got: %v", err)
	}
}

func TestGetReleasePageURL(t *testing.T) {
	url := GetReleasePageURL()

	if url == "" {
		t.Error("Expected non-empty release page URL")
	}

	if !strings.HasPrefix(url, "https://") {
		t.Errorf("Expected HTTPS URL, got: %s", url)
	}

	expectedURL := "https://github.com/YangQing-Lin/cc-switch-cli/releases"
	if url != expectedURL {
		t.Errorf("Expected URL %s, got %s", expectedURL, url)
	}
}

func TestVersionConstants(t *testing.T) {
	if Version == "" {
		t.Error("Version constant should not be empty")
	}

	if BuildDate == "" {
		t.Error("BuildDate variable should not be empty")
	}

	if GitCommit == "" {
		t.Error("GitCommit variable should not be empty")
	}
}
