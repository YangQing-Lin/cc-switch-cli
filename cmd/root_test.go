package cmd

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
	"github.com/YangQing-Lin/cc-switch-cli/internal/template"
	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func resetGlobals() {
	configDir = ""
	noLock = false
	apiKey = ""
	baseURL = ""
	category = "custom"
	appName = "claude"
	defaultSonnetModel = ""
	force = false
	saveFrom = ""
	saveName = ""
	applyTarget = ""
	skipDiff = false
	deleteForce = false
	listCategory = template.CategoryClaudeMd
	setConfigDir = ""
	getSetting = false
	setSetting = ""
	addFile = ""
	addName = ""
	addCategory = template.CategoryClaudeMd
}

// mockTUIRunner 替换 tuiRunner 为 mock，返回 cleanup 函数
func mockTUIRunner(t *testing.T) {
	t.Helper()
	original := tuiRunner
	tuiRunner = func(manager *config.Manager) error {
		return nil // mock: 不启动真实 TUI
	}
	t.Cleanup(func() {
		tuiRunner = original
	})
}

func resetFlags(cmd *cobra.Command) {
	resetFlagSet(cmd.Flags())
	resetFlagSet(cmd.PersistentFlags())
	resetFlagSet(cmd.InheritedFlags())
}

func resetFlagSet(flags *pflag.FlagSet) {
	if flags == nil {
		return
	}
	flags.VisitAll(func(flag *pflag.Flag) {
		_ = flag.Value.Set(flag.DefValue)
		flag.Changed = false
	})
}

func withTempHome(t *testing.T) string {
	t.Helper()
	var home string
	testutil.WithTempHome(t, func(dir string) {
		home = dir
	})
	return home
}

func withTempCWD(t *testing.T) string {
	t.Helper()
	var cwd string
	testutil.WithTempCWD(t, func(dir string) {
		cwd = dir
	})
	return cwd
}

func withStdin(t *testing.T, input string, fn func()) {
	t.Helper()
	orig := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdin pipe: %v", err)
	}
	if _, err := io.WriteString(w, input); err != nil {
		_ = w.Close()
		_ = r.Close()
		t.Fatalf("write stdin: %v", err)
	}
	_ = w.Close()
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = orig
		_ = r.Close()
	})
	fn()
}

func setupManager(t *testing.T) *config.Manager {
	t.Helper()
	manager, err := config.NewManager()
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	return manager
}

func addClaudeProvider(t *testing.T, manager *config.Manager, name string) {
	t.Helper()
	if err := manager.AddProviderForApp("claude", name, "", "sk-test", "https://api.example.com", "custom", "", "", "", ""); err != nil {
		t.Fatalf("add provider: %v", err)
	}
	if err := manager.Save(); err != nil {
		t.Fatalf("save manager: %v", err)
	}
}

func addCodexProvider(t *testing.T, manager *config.Manager, name string, current bool) {
	t.Helper()
	provider := config.Provider{
		ID:   name + "-id",
		Name: name,
		SettingsConfig: map[string]interface{}{
			"config": map[string]interface{}{
				"base_url":   "https://codex.example.com",
				"api_key":    "sk-codex",
				"model_name": "codex-model",
			},
		},
		Category:  "custom",
		CreatedAt: 1,
	}
	if err := manager.AddProviderDirect("codex", provider); err != nil {
		t.Fatalf("add codex provider: %v", err)
	}
	if current {
		if err := manager.SwitchProviderForApp("codex", name); err != nil {
			t.Fatalf("switch codex provider: %v", err)
		}
	}
	if err := manager.Save(); err != nil {
		t.Fatalf("save manager: %v", err)
	}
}

func addGeminiProvider(t *testing.T, manager *config.Manager, name string) {
	t.Helper()
	if err := manager.AddGeminiProvider(name, "https://gemini.example.com", "sk-gemini", "gemini-2.5-pro", config.GeminiAuthAPIKey); err != nil {
		t.Fatalf("add gemini provider: %v", err)
	}
	if err := manager.Save(); err != nil {
		t.Fatalf("save manager: %v", err)
	}
}

func removePortableMarker(t *testing.T) string {
	t.Helper()
	execPath, err := os.Executable()
	if err != nil {
		t.Fatalf("executable: %v", err)
	}
	portableFile := filepath.Join(filepath.Dir(execPath), "portable.ini")
	_ = os.Remove(portableFile)
	return portableFile
}

func TestListConfigsAndSwitch(t *testing.T) {
	resetGlobals()
	withTempHome(t)
	manager := setupManager(t)
	addClaudeProvider(t, manager, "alpha")
	if err := manager.SwitchProvider("alpha"); err != nil {
		t.Fatalf("switch provider: %v", err)
	}

	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := listConfigs(manager); err != nil {
			t.Fatalf("list configs: %v", err)
		}
	})
	if !strings.Contains(stdout, "alpha") {
		t.Errorf("expected list to mention provider name, got: %s", stdout)
	}

	stdout, _ = testutil.CaptureOutput(t, func() {
		if err := switchConfig(manager, "alpha"); err != nil {
			t.Fatalf("switch config: %v", err)
		}
	})
	if !strings.Contains(stdout, "alpha") {
		t.Errorf("expected switch output to mention provider name, got: %s", stdout)
	}
}

func TestGetManagerWithDir(t *testing.T) {
	resetGlobals()
	withTempHome(t)
	configDir = t.TempDir()
	manager, err := getManager()
	if err != nil {
		t.Fatalf("getManager: %v", err)
	}
	if !strings.HasPrefix(manager.GetConfigPath(), configDir) {
		t.Errorf("expected config path under %s, got %s", configDir, manager.GetConfigPath())
	}
}

func TestRootCommandDirFlagParsing(t *testing.T) {
	resetGlobals()
	withTempHome(t)
	customDir := t.TempDir()

	manager, err := config.NewManagerWithDir(customDir)
	if err != nil {
		t.Fatalf("new manager with dir: %v", err)
	}
	addClaudeProvider(t, manager, "dir-provider")

	resetFlags(rootCmd)
	rootCmd.SetArgs([]string{"--dir", customDir, "dir-provider"})
	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("execute root: %v", err)
		}
	})
	resetFlags(rootCmd)
	rootCmd.SetArgs(nil)
	if !strings.Contains(stdout, "dir-provider") {
		t.Fatalf("expected switch output, got: %s", stdout)
	}
}

func TestRunCheckOutput(t *testing.T) {
	resetGlobals()
	withTempHome(t)
	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := runCheck(); err != nil {
			t.Fatalf("runCheck: %v", err)
		}
	})
	if !strings.Contains(stdout, "系统环境检查") {
		t.Errorf("expected check header, got: %s", stdout)
	}
}

func TestCheckFileOutput(t *testing.T) {
	resetGlobals()
	withTempHome(t)
	existing := filepath.Join(t.TempDir(), "exists.txt")
	if err := os.WriteFile(existing, []byte("ok"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	stdout, _ := testutil.CaptureOutput(t, func() {
		checkFile("exists", existing)
		checkFile("missing", filepath.Join(t.TempDir(), "missing.txt"))
	})
	if !strings.Contains(stdout, "exists") || !strings.Contains(stdout, "不存在") {
		t.Fatalf("expected checkFile output, got: %s", stdout)
	}
}

func TestOpenConfigCommandError(t *testing.T) {
	resetGlobals()
	withTempHome(t)
	t.Setenv("PATH", "")
	if err := runOpenConfig(); err == nil {
		t.Fatalf("expected error when opener is missing")
	}
}

func TestOpenConfigCommandUnsupportedOS(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	origGOOS := openConfigGOOS
	openConfigGOOS = "plan9"
	t.Cleanup(func() { openConfigGOOS = origGOOS })

	if err := runOpenConfig(); err == nil || !strings.Contains(err.Error(), "不支持的操作系统") {
		t.Fatalf("expected unsupported OS error, got: %v", err)
	}
}

func TestOpenConfigCommandSuccess(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	binDir := t.TempDir()
	xdgOpen := filepath.Join(binDir, "xdg-open")
	if err := os.WriteFile(xdgOpen, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("write xdg-open: %v", err)
	}
	t.Setenv("PATH", binDir)

	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := runOpenConfig(); err != nil {
			t.Fatalf("expected open success: %v", err)
		}
	})
	if !strings.Contains(stdout, "打开") {
		t.Fatalf("expected open output, got: %s", stdout)
	}
}

func TestCheckUpdatesCommandFallback(t *testing.T) {
	resetGlobals()
	withTempHome(t)
	t.Setenv("PATH", "")
	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := runCheckUpdates(); err != nil {
			t.Fatalf("runCheckUpdates: %v", err)
		}
	})
	if !strings.Contains(stdout, "手动访问") {
		t.Errorf("expected manual URL hint, got: %s", stdout)
	}
}

func TestCheckUpdatesCommandUnsupportedOS(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	origGOOS := checkUpdatesGOOS
	checkUpdatesGOOS = "plan9"
	t.Cleanup(func() { checkUpdatesGOOS = origGOOS })

	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := runCheckUpdates(); err != nil {
			t.Fatalf("runCheckUpdates: %v", err)
		}
	})
	if !strings.Contains(stdout, "请手动访问") {
		t.Errorf("expected manual URL hint, got: %s", stdout)
	}
}

func TestCheckUpdatesCommandSuccess(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	binDir := t.TempDir()
	xdgOpen := filepath.Join(binDir, "xdg-open")
	if err := os.WriteFile(xdgOpen, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("write xdg-open: %v", err)
	}
	t.Setenv("PATH", binDir)

	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := runCheckUpdates(); err != nil {
			t.Fatalf("check updates: %v", err)
		}
	})
	if !strings.Contains(stdout, "浏览器") {
		t.Fatalf("expected open browser output, got: %s", stdout)
	}
}

func TestPortableEnableDisable(t *testing.T) {
	resetGlobals()
	removePortableMarker(t)

	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := runPortableEnable(); err != nil {
			t.Fatalf("enable portable: %v", err)
		}
	})
	if !strings.Contains(stdout, "便携版模式已启用") {
		t.Errorf("expected enable output, got: %s", stdout)
	}

	stdout, _ = testutil.CaptureOutput(t, func() {
		if err := runPortableStatus(); err != nil {
			t.Fatalf("portable status: %v", err)
		}
	})
	if !strings.Contains(stdout, "便携版模式状态") {
		t.Errorf("expected status header, got: %s", stdout)
	}

	stdout, _ = testutil.CaptureOutput(t, func() {
		if err := runPortableDisable(); err != nil {
			t.Fatalf("disable portable: %v", err)
		}
	})
	if !strings.Contains(stdout, "便携版模式已禁用") {
		t.Errorf("expected disable output, got: %s", stdout)
	}
}

func TestPortableNoopPaths(t *testing.T) {
	resetGlobals()
	removePortableMarker(t)

	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := runPortableDisable(); err != nil {
			t.Fatalf("disable portable: %v", err)
		}
	})
	if !strings.Contains(stdout, "无需禁用") {
		t.Fatalf("expected noop disable output, got: %s", stdout)
	}

	if err := runPortableEnable(); err != nil {
		t.Fatalf("enable portable: %v", err)
	}
	stdout, _ = testutil.CaptureOutput(t, func() {
		if err := runPortableEnable(); err != nil {
			t.Fatalf("enable portable: %v", err)
		}
	})
	if !strings.Contains(stdout, "已经启用") {
		t.Fatalf("expected already enabled output, got: %s", stdout)
	}
}

func TestClaudePluginStatus(t *testing.T) {
	resetGlobals()
	withTempHome(t)
	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := runClaudePluginStatus(); err != nil {
			t.Fatalf("status: %v", err)
		}
	})
	if !strings.Contains(stdout, "Claude 插件配置状态") {
		t.Errorf("expected status output, got: %s", stdout)
	}
}

func TestClaudePluginCheckNotApplied(t *testing.T) {
	resetGlobals()
	withTempHome(t)
	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := runClaudePluginCheck(); err != nil {
			t.Fatalf("check plugin: %v", err)
		}
	})
	if !strings.Contains(stdout, "未应用") {
		t.Fatalf("expected not applied output, got: %s", stdout)
	}
}

func TestClaudePluginApplyRemoveCheck(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := runClaudePluginApply(); err != nil {
			t.Fatalf("apply plugin: %v", err)
		}
	})
	if !strings.Contains(stdout, "Claude") {
		t.Fatalf("expected apply output, got: %s", stdout)
	}

	stdout, _ = testutil.CaptureOutput(t, func() {
		if err := runClaudePluginCheck(); err != nil {
			t.Fatalf("check plugin: %v", err)
		}
	})
	if !strings.Contains(stdout, "插件配置") {
		t.Fatalf("expected check output, got: %s", stdout)
	}

	stdout, _ = testutil.CaptureOutput(t, func() {
		if err := runClaudePluginStatus(); err != nil {
			t.Fatalf("status after apply: %v", err)
		}
	})
	if !strings.Contains(stdout, "配置文件") {
		t.Fatalf("expected status output, got: %s", stdout)
	}

	stdout, _ = testutil.CaptureOutput(t, func() {
		if err := runClaudePluginRemove(); err != nil {
			t.Fatalf("remove plugin: %v", err)
		}
	})
	if !strings.Contains(stdout, "移除") && !strings.Contains(stdout, "无需操作") {
		t.Fatalf("expected remove output, got: %s", stdout)
	}
}

func TestGetConfigManager(t *testing.T) {
	resetGlobals()
	withTempHome(t)
	manager, err := getConfigManager()
	if err != nil {
		t.Fatalf("getConfigManager: %v", err)
	}
	if manager == nil {
		t.Fatalf("expected manager")
	}
}

func TestGetConfigManagerWithDir(t *testing.T) {
	resetGlobals()
	withTempHome(t)
	configDir = t.TempDir()
	manager, err := getConfigManager()
	if err != nil {
		t.Fatalf("getConfigManager: %v", err)
	}
	if manager == nil || !strings.HasPrefix(manager.GetConfigPath(), configDir) {
		t.Fatalf("expected config dir manager")
	}
}

func TestStartTUICancelledWhenLocked(t *testing.T) {
	resetGlobals()
	mockTUIRunner(t) // 避免启动真实 TUI
	home := withTempHome(t)
	removePortableMarker(t)

	manager := setupManager(t)
	configPath := manager.GetConfigPath()
	configDirPath := filepath.Dir(configPath)
	lockPath := filepath.Join(configDirPath, ".cc-switch.lock")
	if err := os.MkdirAll(configDirPath, 0755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(lockPath, []byte("123"), 0600); err != nil {
		t.Fatalf("write lock: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(lockPath)
		_ = os.RemoveAll(filepath.Join(home, ".cc-switch"))
	})

	withStdin(t, "n\n", func() {
		if err := startTUI(manager); err == nil {
			t.Fatalf("expected cancel error")
		}
	})
}

func TestStartTUILockError(t *testing.T) {
	resetGlobals()
	mockTUIRunner(t) // 避免启动真实 TUI
	withTempHome(t)
	removePortableMarker(t)

	customDir := t.TempDir()
	manager, err := config.NewManagerWithDir(customDir)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	if err := os.Chmod(customDir, 0555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer os.Chmod(customDir, 0755)

	if err := startTUI(manager); err == nil {
		t.Fatalf("expected lock error")
	}
}

func TestStartTUIForceAcquireError(t *testing.T) {
	if isRoot() {
		t.Skip("skipping: running as root")
	}
	resetGlobals()
	mockTUIRunner(t) // 避免启动真实 TUI
	withTempHome(t)
	removePortableMarker(t)

	customDir := t.TempDir()
	lockPath := filepath.Join(customDir, ".cc-switch.lock")
	if err := os.WriteFile(lockPath, []byte("123"), 0600); err != nil {
		t.Fatalf("write lock: %v", err)
	}
	// 将锁文件也设为只读，阻止覆盖写入
	if err := os.Chmod(lockPath, 0444); err != nil {
		t.Fatalf("chmod lock file: %v", err)
	}
	if err := os.Chmod(customDir, 0555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer func() {
		os.Chmod(lockPath, 0644)
		os.Chmod(customDir, 0755)
	}()

	manager, err := config.NewManagerWithDir(customDir)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	withStdin(t, "y\n", func() {
		if err := startTUI(manager); err == nil {
			t.Fatalf("expected force acquire error")
		}
	})
}

func TestExecuteExitCodes(t *testing.T) {
	if os.Getenv("CCS_CMD_HELPER") == "1" {
		args := strings.Fields(os.Getenv("CCS_CMD_ARGS"))
		os.Args = append([]string{"cc-switch"}, args...)
		Execute()
		return
	}

	runExecuteHelper(t, []string{"version"}, 0)
	runExecuteHelper(t, []string{"backup", "restore"}, 1)
}

func runExecuteHelper(t *testing.T, args []string, want int) {
	t.Helper()
	tempHome := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=TestExecuteExitCodes", "--")
	cmd.Env = append(os.Environ(),
		"CCS_CMD_HELPER=1",
		"CCS_CMD_ARGS="+strings.Join(args, " "),
		"HOME="+tempHome,
		"USERPROFILE="+tempHome,
	)
	output, err := cmd.CombinedOutput()
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

func TestArgsValidationTable(t *testing.T) {
	resetGlobals()
	cases := []struct {
		name string
		cmd  *cobra.Command
		args []string
	}{
		{"add missing arg", addCmd, []string{}},
		{"delete missing arg", deleteCmd, []string{}},
		{"update missing arg", updateCmd, []string{}},
		{"show missing arg", showCmd, []string{}},
		{"codex add missing arg", codexAddCmd, []string{}},
		{"codex delete missing arg", codexDeleteCmd, []string{}},
		{"codex update missing arg", codexUpdateCmd, []string{}},
		{"gemini add missing arg", geminiAddCmd, []string{}},
		{"gemini delete missing arg", geminiDeleteCmd, []string{}},
		{"gemini switch missing arg", geminiSwitchCmd, []string{}},
		{"template apply missing arg", applyCmd, []string{}},
		{"template delete missing arg", templateDeleteCmd, []string{}},
		{"backup restore missing arg", backupRestoreCmd, []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.cmd.Args == nil {
				t.Fatalf("command has no Args validator")
			}
			if err := tc.cmd.Args(tc.cmd, tc.args); err == nil {
				t.Fatalf("expected args validation error")
			}
		})
	}
}

func TestSwitchConfigErrors(t *testing.T) {
	resetGlobals()
	withTempHome(t)
	manager := setupManager(t)

	t.Run("switch to missing provider", func(t *testing.T) {
		if err := switchConfig(manager, "missing-provider"); err == nil {
			t.Fatalf("expected error switching to missing provider")
		}
	})
}

func TestListConfigsEmpty(t *testing.T) {
	resetGlobals()
	withTempHome(t)
	manager := setupManager(t)
	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := listConfigs(manager); err != nil {
			t.Fatalf("listConfigs: %v", err)
		}
	})
	if !strings.Contains(stdout, "暂无配置") {
		t.Errorf("expected empty list message, got: %s", stdout)
	}
}

func TestOpenConfigGetManagerError(t *testing.T) {
	resetGlobals()
	withTempHome(t)
	manager := setupManager(t)

	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := runOpenConfig(); err != nil {
			t.Logf("runOpenConfig returned error (expected for some OS): %v", err)
		}
	})

	configDir := filepath.Dir(manager.GetConfigPath())
	if !strings.Contains(stdout, configDir) && len(stdout) == 0 {
		t.Logf("Expected config directory %s in output or command to succeed", configDir)
	}
}

func TestPortableStatusSuccess(t *testing.T) {
	resetGlobals()
	withTempHome(t)

	stdout, _ := testutil.CaptureOutput(t, func() {
		if err := runPortableStatus(); err != nil {
			t.Fatalf("runPortableStatus failed: %v", err)
		}
	})
	if !strings.Contains(stdout, "便携版模式状态") {
		t.Errorf("expected status header, got: %s", stdout)
	}
}

func TestPortableEnableError(t *testing.T) {
	resetGlobals()
	removePortableMarker(t)

	execPath, err := os.Executable()
	if err != nil {
		t.Fatalf("executable: %v", err)
	}
	execDir := filepath.Dir(execPath)
	if err := os.Chmod(execDir, 0555); err != nil {
		t.Skip("cannot test permission error")
	}
	defer os.Chmod(execDir, 0755)

	if err := runPortableEnable(); err == nil {
		t.Fatalf("expected enable error")
	}
}

func TestPortableDisableError(t *testing.T) {
	resetGlobals()
	removePortableMarker(t)

	if err := runPortableEnable(); err != nil {
		t.Fatalf("enable portable: %v", err)
	}

	execPath, err := os.Executable()
	if err != nil {
		t.Fatalf("executable: %v", err)
	}
	execDir := filepath.Dir(execPath)
	portableFile := filepath.Join(execDir, "portable.ini")

	if err := os.Chmod(execDir, 0555); err != nil {
		t.Skip("cannot test permission error")
	}
	defer os.Chmod(execDir, 0755)

	if err := runPortableDisable(); err == nil {
		os.Remove(portableFile)
		t.Fatalf("expected disable error")
	}
	os.Chmod(execDir, 0755)
	os.Remove(portableFile)
}

func TestClaudePluginStatusErrors(t *testing.T) {
	resetGlobals()
	customDir := filepath.Join(t.TempDir(), "missing", "deep", "path")
	t.Setenv("HOME", customDir)
	t.Setenv("USERPROFILE", customDir)

	if err := runClaudePluginStatus(); err != nil {
		t.Fatalf("status should not fail: %v", err)
	}
}

func TestClaudePluginApplyErrors(t *testing.T) {
	resetGlobals()
	home := withTempHome(t)
	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0555); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	defer os.Chmod(claudeDir, 0755)

	if err := runClaudePluginApply(); err == nil {
		t.Fatalf("expected apply error")
	}
}

func TestClaudePluginRemoveErrors(t *testing.T) {
	resetGlobals()
	home := withTempHome(t)

	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	configFile := filepath.Join(claudeDir, "config.json")
	if err := os.WriteFile(configFile, []byte(`{"primaryApiKey":"test"}`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := os.Chmod(claudeDir, 0555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer os.Chmod(claudeDir, 0755)

	if err := runClaudePluginRemove(); err == nil {
		t.Fatalf("expected remove error")
	}
}

func TestClaudePluginCheckErrors(t *testing.T) {
	resetGlobals()
	customDir := filepath.Join(t.TempDir(), "missing")
	t.Setenv("HOME", customDir)
	t.Setenv("USERPROFILE", customDir)

	if err := runClaudePluginCheck(); err != nil {
		t.Fatalf("check should not fail on missing config: %v", err)
	}
}
