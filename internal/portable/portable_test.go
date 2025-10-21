package portable

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsPortableMode(t *testing.T) {
	result := IsPortableMode()

	if result {
		portableFile := getExpectedPortableFilePath(t)
		if _, err := os.Stat(portableFile); os.IsNotExist(err) {
			t.Errorf("IsPortableMode returned true but portable.ini does not exist at: %s", portableFile)
		}
	}
}

func TestIsPortableModeWithFile(t *testing.T) {
	tmpDir := t.TempDir()

	fakeExe := filepath.Join(tmpDir, "fake_exe")
	if err := os.WriteFile(fakeExe, []byte("fake"), 0755); err != nil {
		t.Fatalf("Failed to create fake executable: %v", err)
	}

	portableFile := filepath.Join(tmpDir, "portable.ini")
	if err := os.WriteFile(portableFile, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create portable.ini: %v", err)
	}

	if _, err := os.Stat(portableFile); err != nil {
		t.Errorf("portable.ini should exist: %v", err)
	}
}

func TestIsPortableModeWithDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	fakeExe := filepath.Join(tmpDir, "fake_exe")
	if err := os.WriteFile(fakeExe, []byte("fake"), 0755); err != nil {
		t.Fatalf("Failed to create fake executable: %v", err)
	}

	portableDir := filepath.Join(tmpDir, "portable.ini")
	if err := os.MkdirAll(portableDir, 0755); err != nil {
		t.Fatalf("Failed to create portable.ini directory: %v", err)
	}

	info, err := os.Stat(portableDir)
	if err != nil {
		t.Fatalf("Failed to stat portable.ini: %v", err)
	}

	if !info.IsDir() {
		t.Error("portable.ini should be a directory in this test")
	}
}

func TestGetPortableConfigDir(t *testing.T) {
	configDir, err := GetPortableConfigDir()
	if err != nil {
		t.Fatalf("GetPortableConfigDir failed: %v", err)
	}

	if configDir == "" {
		t.Error("Expected non-empty config directory")
	}

	if !filepath.IsAbs(configDir) {
		t.Errorf("Expected absolute path, got: %s", configDir)
	}

	if !filepath.IsAbs(configDir) || filepath.Base(configDir) != ".cc-switch" {
		t.Errorf("Expected config dir to end with '.cc-switch', got: %s", configDir)
	}
}

func TestGetPortableConfigDirStructure(t *testing.T) {
	configDir, err := GetPortableConfigDir()
	if err != nil {
		t.Fatalf("GetPortableConfigDir failed: %v", err)
	}

	execPath, err := os.Executable()
	if err != nil {
		t.Fatalf("Failed to get executable path: %v", err)
	}

	execDir := filepath.Dir(execPath)
	expectedDir := filepath.Join(execDir, ".cc-switch")

	if configDir != expectedDir {
		t.Errorf("Expected config dir %s, got %s", expectedDir, configDir)
	}
}

func getExpectedPortableFilePath(t *testing.T) string {
	t.Helper()

	execPath, err := os.Executable()
	if err != nil {
		t.Fatalf("Failed to get executable path: %v", err)
	}

	execDir := filepath.Dir(execPath)
	return filepath.Join(execDir, "portable.ini")
}
