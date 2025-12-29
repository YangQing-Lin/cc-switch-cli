package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
)

func main() {
	tmpDir, err := os.MkdirTemp("", "ccs-pco-010-*")
	if err != nil {
		fatalf("mkdirtemp: %v", err)
	}

	cleanup := false
	for _, arg := range os.Args[1:] {
		if arg == "--cleanup" {
			cleanup = true
		}
	}

	manager, err := config.NewManagerWithDir(tmpDir)
	if err != nil {
		fatalf("NewManagerWithDir(%s): %v", tmpDir, err)
	}

	// 准备 Providers（无需真实可用的 token/baseURL，只用于走写回路径）
	const (
		claudeName = "pco-010-claude"
		codexName  = "pco-010-codex"
		geminiName = "pco-010-gemini"
	)

	if err := manager.AddProviderForApp("claude", claudeName, "", "test-token", "https://example.invalid", "test", "claude-sonnet", "", "", ""); err != nil {
		fatalf("AddProviderForApp(claude): %v", err)
	}
	if err := manager.AddProviderForApp("codex", codexName, "", "sk-test", "https://api.example.invalid", "test", "gpt-5-codex", "high", "", ""); err != nil {
		fatalf("AddProviderForApp(codex): %v", err)
	}

	// Gemini 当前没有 AddProviderForApp，走 AddProviderDirect 注入最小 Provider。
	geminiProvider := config.Provider{
		ID:   "pco-010-gemini-id",
		Name: geminiName,
		SettingsConfig: map[string]interface{}{
			"env": map[string]interface{}{
				"GOOGLE_GEMINI_BASE_URL": "https://gemini.example.invalid",
				"GEMINI_API_KEY":         "gemini-test-key",
				"GEMINI_MODEL":           "gemini-1.5-pro",
			},
			// 留一个未知字段：用于验证“未知字段存在时，写回仍会重排 key”
			"some_unknown_field": "keep-me",
		},
	}
	if err := manager.AddProviderDirect("gemini", geminiProvider); err != nil {
		fatalf("AddProviderDirect(gemini): %v", err)
	}

	beforeCodexPath := filepath.Join(tmpDir, ".codex", "config.toml.before")
	codexPath := filepath.Join(tmpDir, ".codex", "config.toml")
	beforeClaudePath := filepath.Join(tmpDir, ".claude", "settings.json.before")
	claudePath := filepath.Join(tmpDir, ".claude", "settings.json")
	beforeGeminiPath := filepath.Join(tmpDir, ".gemini", "settings.json.before")
	geminiPath := filepath.Join(tmpDir, ".gemini", "settings.json")

	if err := os.MkdirAll(filepath.Dir(codexPath), 0o755); err != nil {
		fatalf("mkdir codex dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(claudePath), 0o755); err != nil {
		fatalf("mkdir claude dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(geminiPath), 0o755); err != nil {
		fatalf("mkdir gemini dir: %v", err)
	}

	// --- Seed “before” files（刻意构造顺序） ---
	codexBefore := codexBeforeTOML()
	claudeBefore := claudeBeforeJSON()
	geminiBefore := geminiBeforeJSON()

	if err := os.WriteFile(beforeCodexPath, []byte(codexBefore), 0o644); err != nil {
		fatalf("write before codex: %v", err)
	}
	if err := os.WriteFile(codexPath, []byte(codexBefore), 0o644); err != nil {
		fatalf("write codex: %v", err)
	}

	if err := os.WriteFile(beforeClaudePath, []byte(claudeBefore), 0o644); err != nil {
		fatalf("write before claude: %v", err)
	}
	if err := os.WriteFile(claudePath, []byte(claudeBefore), 0o644); err != nil {
		fatalf("write claude: %v", err)
	}

	if err := os.WriteFile(beforeGeminiPath, []byte(geminiBefore), 0o644); err != nil {
		fatalf("write before gemini: %v", err)
	}
	if err := os.WriteFile(geminiPath, []byte(geminiBefore), 0o644); err != nil {
		fatalf("write gemini: %v", err)
	}

	// --- Trigger write-back via SwitchProviderForApp ---
	if err := manager.SwitchProviderForApp("codex", codexName); err != nil {
		fatalf("SwitchProviderForApp(codex): %v", err)
	}
	if err := manager.SwitchProviderForApp("claude", claudeName); err != nil {
		fatalf("SwitchProviderForApp(claude): %v", err)
	}
	if err := manager.SwitchProviderForApp("gemini", geminiName); err != nil {
		fatalf("SwitchProviderForApp(gemini): %v", err)
	}

	codexAfter, err := os.ReadFile(codexPath)
	if err != nil {
		fatalf("read codex after: %v", err)
	}
	claudeAfter, err := os.ReadFile(claudePath)
	if err != nil {
		fatalf("read claude after: %v", err)
	}
	geminiAfter, err := os.ReadFile(geminiPath)
	if err != nil {
		fatalf("read gemini after: %v", err)
	}

	fmt.Printf("tmpDir=%s\n\n", tmpDir)

	// --- Codex TOML: table order ---
	fmt.Println("[Codex] config.toml table order (before -> after)")
	beforeTables := tomlTopLevelTables([]byte(codexBefore))
	afterTables := tomlTopLevelTables(codexAfter)
	fmt.Printf("  before: %s\n", strings.Join(beforeTables, " | "))
	fmt.Printf("  after : %s\n", strings.Join(afterTables, " | "))
	fmt.Printf("  index(before) sandbox_workspace_write=%d model_providers=%d\n",
		indexOf(beforeTables, "sandbox_workspace_write"),
		indexOf(beforeTables, "model_providers"),
	)
	fmt.Printf("  index(after ) sandbox_workspace_write=%d model_providers=%d\n\n",
		indexOf(afterTables, "sandbox_workspace_write"),
		indexOf(afterTables, "model_providers"),
	)

	// --- Claude/Gemini JSON: top-level key order ---
	fmt.Println("[Claude] settings.json top-level key order (before -> after)")
	beforeClaudeKeys, err := jsonTopLevelKeyOrder([]byte(claudeBefore))
	if err != nil {
		fatalf("parse claude before json: %v", err)
	}
	afterClaudeKeys, err := jsonTopLevelKeyOrder(claudeAfter)
	if err != nil {
		fatalf("parse claude after json: %v", err)
	}
	fmt.Printf("  before: %s\n", strings.Join(beforeClaudeKeys, " | "))
	fmt.Printf("  after : %s\n\n", strings.Join(afterClaudeKeys, " | "))

	fmt.Println("[Gemini] settings.json top-level key order (before -> after)")
	beforeGeminiKeys, err := jsonTopLevelKeyOrder([]byte(geminiBefore))
	if err != nil {
		fatalf("parse gemini before json: %v", err)
	}
	afterGeminiKeys, err := jsonTopLevelKeyOrder(geminiAfter)
	if err != nil {
		fatalf("parse gemini after json: %v", err)
	}
	fmt.Printf("  before: %s\n", strings.Join(beforeGeminiKeys, " | "))
	fmt.Printf("  after : %s\n", strings.Join(afterGeminiKeys, " | "))

	if cleanup {
		_ = os.RemoveAll(tmpDir)
	}
}

func fatalf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func tomlTopLevelTables(data []byte) []string {
	lines := strings.Split(string(data), "\n")
	var out []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) < 3 {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") && !strings.HasPrefix(line, "[[") {
			name := strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			out = append(out, name)
		}
	}
	return out
}

func indexOf(list []string, s string) int {
	for i, v := range list {
		if v == s {
			return i
		}
	}
	return -1
}

func jsonTopLevelKeyOrder(data []byte) ([]string, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, fmt.Errorf("expected top-level object")
	}

	var keys []string
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := tok.(string)
		if !ok {
			return nil, fmt.Errorf("expected object key, got %T", tok)
		}
		keys = append(keys, key)

		// skip value (could be nested)
		if err := skipJSONValue(dec); err != nil {
			return nil, err
		}
	}

	// consume closing '}'
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	return keys, nil
}

func skipJSONValue(dec *json.Decoder) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}

	delim, ok := tok.(json.Delim)
	if !ok {
		return nil
	}

	switch delim {
	case '{':
		for dec.More() {
			// key
			if _, err := dec.Token(); err != nil {
				return err
			}
			if err := skipJSONValue(dec); err != nil {
				return err
			}
		}
		_, err := dec.Token() // '}'
		return err
	case '[':
		for dec.More() {
			if err := skipJSONValue(dec); err != nil {
				return err
			}
		}
		_, err := dec.Token() // ']'
		return err
	default:
		return nil
	}
}

func codexBeforeTOML() string {
	// 刻意把 [sandbox_workspace_write] 放在 [model_providers] 之前，并加入注释/空行。
	// 同时用 CRLF 来模拟 Windows 换行输入（写回一般会变成 LF）。
	lines := []string{
		`# Codex config.toml (PCO-010 before)`,
		`# contains comments + blank lines + multiple tables`,
		``,
		`[sandbox_workspace_write]`,
		`enabled = true`,
		``,
		`# provider configs`,
		`[model_providers]`,
		``,
		`[model_providers.openai]`,
		`api_key = "sk-existing"`,
		`base_url = "https://api.existing.invalid"`,
		``,
		`# tail comment`,
		``,
	}
	return strings.Join(lines, "\r\n")
}

func claudeBeforeJSON() string {
	// 顶层 key 顺序刻意设置为：permissions -> env -> zzz_unknown -> model
	// 写回会经过 json.MarshalIndent（map key 排序），从而改变顺序。
	return "{\n" +
		"  \"permissions\": {\"allow\": [], \"deny\": []},\n" +
		"  \"env\": {\"ANTHROPIC_AUTH_TOKEN\": \"t\", \"ANTHROPIC_BASE_URL\": \"https://x.invalid\"},\n" +
		"  \"zzz_unknown\": {\"a\": 1},\n" +
		"  \"model\": \"claude-sonnet\"\n" +
		"}\n"
}

func geminiBeforeJSON() string {
	// 顶层 key 顺序刻意设置为：mcpServers -> security -> yyy_unknown
	return "{\n" +
		"  \"mcpServers\": {\"keep\": true},\n" +
		"  \"security\": {\"auth\": {\"selectedType\": \"gemini-api-key\"}},\n" +
		"  \"yyy_unknown\": {\"x\": 1}\n" +
		"}\n"
}
