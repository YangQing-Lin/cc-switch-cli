package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()

	manager, err := NewManagerWithDir(tmpDir)
	if err != nil {
		t.Fatalf("NewManagerWithDir() error = %v", err)
	}

	if manager == nil {
		t.Fatal("NewManagerWithDir() returned nil")
	}

	// 注意: Load() 现在不立即创建文件,首次添加配置时才创建
	// 这是为了避免与 cc-switch UI 产生并发写入竞争
	// 验证管理器可用即可
	providers := manager.ListProvidersForApp("claude")
	if providers == nil {
		t.Error("ListProvidersForApp 应该返回空列表而不是 nil")
	}
}

func TestAddProvider(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManagerWithDir(tmpDir)
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}

	tests := []struct {
		name     string
		appName  string
		provName string
		apiToken string
		baseURL  string
		category string
		wantErr  bool
	}{
		{
			name:     "添加Claude提供商",
			appName:  "claude",
			provName: "Test Provider",
			apiToken: "test-token-123",
			baseURL:  "",
			category: "test",
			wantErr:  false,
		},
		{
			name:     "添加Codex提供商",
			appName:  "codex",
			provName: "Codex Test",
			apiToken: "codex-key-456",
			baseURL:  "https://api.example.com",
			category: "custom",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.AddProviderForApp(tt.appName, tt.provName, "", tt.apiToken, tt.baseURL, tt.category, "", "", "", "")
			if (err != nil) != tt.wantErr {
				t.Errorf("AddProviderForApp() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// 验证提供商已添加
				provider, err := manager.GetProviderForApp(tt.appName, tt.provName)
				if err != nil {
					t.Errorf("获取提供商失败: %v", err)
					return
				}

				if provider.Name != tt.provName {
					t.Errorf("提供商名称不匹配, got = %v, want %v", provider.Name, tt.provName)
				}
			}
		})
	}
}

func TestListProviders(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManagerWithDir(tmpDir)
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}

	// 初始应该为空
	providers := manager.ListProvidersForApp("claude")
	if len(providers) != 0 {
		t.Errorf("初始提供商数量应为0, got = %d", len(providers))
	}

	// 添加多个提供商
	manager.AddProviderForApp("claude", "Provider 1", "", "token1", "", "test", "", "", "", "")
	manager.AddProviderForApp("claude", "Provider 2", "", "token2", "", "test", "", "", "", "")
	manager.AddProviderForApp("claude", "Provider 3", "", "token3", "", "test", "", "", "", "")

	// 验证数量
	providers = manager.ListProvidersForApp("claude")
	if len(providers) != 3 {
		t.Errorf("提供商数量 = %d, want 3", len(providers))
	}
}

func TestGetProvider(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManagerWithDir(tmpDir)
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}

	// 添加提供商
	err = manager.AddProviderForApp("claude", "Test Provider", "", "test-token", "", "test", "", "", "", "")
	if err != nil {
		t.Fatalf("添加提供商失败: %v", err)
	}

	// 获取存在的提供商
	provider, err := manager.GetProviderForApp("claude", "Test Provider")
	if err != nil {
		t.Errorf("GetProviderForApp() error = %v", err)
	}
	if provider == nil {
		t.Fatal("GetProviderForApp() returned nil")
	}
	if provider.Name != "Test Provider" {
		t.Errorf("provider.Name = %v, want 'Test Provider'", provider.Name)
	}

	// 获取不存在的提供商
	_, err = manager.GetProviderForApp("claude", "Nonexistent")
	if err == nil {
		t.Error("获取不存在的提供商应该返回错误")
	}
}

func TestDeleteProvider(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManagerWithDir(tmpDir)
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}

	// 添加两个提供商
	manager.AddProviderForApp("claude", "Provider 1", "", "token1", "", "test", "", "", "", "")
	manager.AddProviderForApp("claude", "To Delete", "", "token2", "", "test", "", "", "", "")

	// 验证提供商存在
	providers := manager.ListProvidersForApp("claude")
	if len(providers) != 2 {
		t.Fatalf("提供商数量 = %d, want 2", len(providers))
	}

	// 删除非当前提供商
	err = manager.DeleteProviderForApp("claude", "To Delete")
	if err != nil {
		t.Errorf("DeleteProviderForApp() error = %v", err)
	}

	// 验证提供商已删除
	providers = manager.ListProvidersForApp("claude")
	if len(providers) != 1 {
		t.Errorf("删除后提供商数量 = %d, want 1", len(providers))
	}

	// 删除不存在的提供商
	err = manager.DeleteProviderForApp("claude", "Nonexistent")
	if err == nil {
		t.Error("删除不存在的提供商应该返回错误")
	}
}

func TestUpdateProvider(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManagerWithDir(tmpDir)
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}

	// 添加提供商
	manager.AddProviderForApp("claude", "Original", "", "old-token", "", "test", "", "", "", "")

	// 更新提供商
	err = manager.UpdateProviderForApp("claude", "Original", "New Name", "", "new-token", "", "updated", "", "", "", "")
	if err != nil {
		t.Errorf("UpdateProviderForApp() error = %v", err)
	}

	// 验证更新
	provider, err := manager.GetProviderForApp("claude", "New Name")
	if err != nil {
		t.Fatalf("获取更新后的提供商失败: %v", err)
	}

	if provider.Name != "New Name" {
		t.Errorf("provider.Name = %v, want 'New Name'", provider.Name)
	}

	// 验证旧名称不存在
	_, err = manager.GetProviderForApp("claude", "Original")
	if err == nil {
		t.Error("旧名称应该不存在")
	}
}

func TestConfigPersistence(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建第一个管理器并添加数据
	manager1, err := NewManagerWithDir(tmpDir)
	if err != nil {
		t.Fatalf("创建管理器1失败: %v", err)
	}

	manager1.AddProviderForApp("claude", "Persistent", "", "token123", "", "test", "", "", "", "")

	// 创建第二个管理器（模拟重启）
	manager2, err := NewManagerWithDir(tmpDir)
	if err != nil {
		t.Fatalf("创建管理器2失败: %v", err)
	}

	// 验证数据持久化
	provider, err := manager2.GetProviderForApp("claude", "Persistent")
	if err != nil {
		t.Errorf("持久化数据读取失败: %v", err)
	}
	if provider == nil {
		t.Fatal("持久化数据丢失")
	}
	if provider.Name != "Persistent" {
		t.Errorf("持久化数据不匹配, got = %v", provider.Name)
	}
}

func TestConfigFileFormat(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManagerWithDir(tmpDir)
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}

	// 添加提供商
	manager.AddProviderForApp("claude", "Test", "", "token", "", "test", "", "", "", "")

	// 读取配置文件
	configPath := filepath.Join(tmpDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("读取配置文件失败: %v", err)
	}

	// 解析配置文件
	var config MultiAppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("解析配置文件失败: %v", err)
	}

	// 验证版本
	if config.Version != 2 {
		t.Errorf("config.Version = %d, want 2", config.Version)
	}

	// 验证结构
	if config.Apps == nil {
		t.Fatal("config.Apps is nil")
	}

	claudeApp, ok := config.Apps["claude"]
	if !ok {
		t.Fatal("config.Apps['claude'] not found")
	}

	if len(claudeApp.Providers) != 1 {
		t.Errorf("claude providers count = %d, want 1", len(claudeApp.Providers))
	}
}

func TestProviderID(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManagerWithDir(tmpDir)
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}

	// 添加多个提供商
	manager.AddProviderForApp("claude", "Provider 1", "", "token1", "", "test", "", "", "", "")
	manager.AddProviderForApp("claude", "Provider 2", "", "token2", "", "test", "", "", "", "")

	// 获取提供商
	p1, _ := manager.GetProviderForApp("claude", "Provider 1")
	p2, _ := manager.GetProviderForApp("claude", "Provider 2")

	// 验证ID唯一性
	if p1.ID == p2.ID {
		t.Error("提供商ID应该唯一")
	}

	// 验证ID非空
	if p1.ID == "" || p2.ID == "" {
		t.Error("提供商ID不应该为空")
	}
}

func TestMultiAppIsolation(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManagerWithDir(tmpDir)
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}

	// 为不同应用添加同名提供商
	manager.AddProviderForApp("claude", "Provider 1", "", "claude-token-1", "", "test", "", "", "", "")
	manager.AddProviderForApp("claude", "Same Name", "", "claude-token", "", "test", "", "", "", "")
	manager.AddProviderForApp("codex", "Same Name", "", "codex-token", "", "test", "", "", "", "")

	// 验证它们是独立的
	claudeProvider, err := manager.GetProviderForApp("claude", "Same Name")
	if err != nil {
		t.Fatalf("获取Claude提供商失败: %v", err)
	}

	codexProvider, err := manager.GetProviderForApp("codex", "Same Name")
	if err != nil {
		t.Fatalf("获取Codex提供商失败: %v", err)
	}

	// 验证ID不同
	if claudeProvider.ID == codexProvider.ID {
		t.Error("不同应用的提供商应该有不同的ID")
	}

	// 验证删除一个不影响另一个（删除非当前提供商）
	err = manager.DeleteProviderForApp("claude", "Same Name")
	if err != nil {
		t.Logf("删除Claude提供商: %v (可能是当前激活的)", err)
	}

	// 验证Codex提供商仍然存在
	_, err = manager.GetProviderForApp("codex", "Same Name")
	if err != nil {
		t.Error("Codex提供商应该仍然存在")
	}

	// 验证Claude和Codex的提供商数量独立
	claudeProviders := manager.ListProvidersForApp("claude")
	codexProviders := manager.ListProvidersForApp("codex")

	t.Logf("Claude提供商数量: %d", len(claudeProviders))
	t.Logf("Codex提供商数量: %d", len(codexProviders))

	if len(codexProviders) != 1 {
		t.Errorf("Codex提供商数量 = %d, want 1", len(codexProviders))
	}
}

// TestConfigMigration 测试从旧格式迁移到新格式
func TestConfigMigration(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// 1. 创建旧格式的配置文件（apps 嵌套在 "apps" 键下）
	oldConfig := OldMultiAppConfig{
		Version: 2,
		Apps: map[string]ProviderManager{
			"claude": {
				Providers: map[string]Provider{
					"test-id": {
						ID:   "test-id",
						Name: "Test Provider",
						SettingsConfig: map[string]interface{}{
							"env": map[string]interface{}{
								"ANTHROPIC_AUTH_TOKEN": "sk-test123",
								"ANTHROPIC_BASE_URL":   "https://api.test.com",
							},
						},
						Category:  "custom",
						CreatedAt: 1234567890,
					},
				},
				Current: "test-id",
			},
			"codex": {
				Providers: map[string]Provider{},
				Current:   "",
			},
		},
	}

	// 写入旧格式配置
	oldData, err := json.MarshalIndent(oldConfig, "", "  ")
	if err != nil {
		t.Fatalf("序列化旧配置失败: %v", err)
	}

	if err := os.WriteFile(configPath, oldData, 0600); err != nil {
		t.Fatalf("写入旧配置文件失败: %v", err)
	}

	// 2. 使用 Manager 加载（应该自动迁移）
	manager, err := NewManagerWithDir(tmpDir)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 3. 验证配置已迁移
	providers := manager.ListProvidersForApp("claude")
	if len(providers) != 1 {
		t.Errorf("迁移后 Claude 提供商数量 = %d, want 1", len(providers))
	}

	provider, err := manager.GetProviderForApp("claude", "Test Provider")
	if err != nil {
		t.Errorf("获取迁移后的提供商失败: %v", err)
	}

	if provider.ID != "test-id" {
		t.Errorf("提供商 ID = %s, want test-id", provider.ID)
	}

	// 4. 验证新格式文件结构（展平的）
	newData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("读取迁移后的配置文件失败: %v", err)
	}

	var rawConfig map[string]interface{}
	if err := json.Unmarshal(newData, &rawConfig); err != nil {
		t.Fatalf("解析迁移后的配置失败: %v", err)
	}

	// 验证是否是展平格式（直接在顶层有 "claude" 和 "codex"）
	if _, ok := rawConfig["claude"]; !ok {
		t.Error("新格式应该在顶层有 'claude' 键")
	}

	if _, ok := rawConfig["codex"]; !ok {
		t.Error("新格式应该在顶层有 'codex' 键")
	}

	// 验证不应该有嵌套的 "apps" 键
	if _, ok := rawConfig["apps"]; ok {
		t.Error("新格式不应该有嵌套的 'apps' 键")
	}

	// 5. 验证归档文件是否创建
	archiveDir := filepath.Join(tmpDir, "archive")
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		t.Errorf("读取归档目录失败: %v", err)
	} else if len(entries) == 0 {
		t.Error("应该创建归档备份文件")
	}
}

// TestNewFormatCompatibility 测试新格式的序列化和反序列化
func TestNewFormatCompatibility(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManagerWithDir(tmpDir)
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}

	// 添加测试数据
	err = manager.AddProviderForApp("claude", "Provider 1", "", "sk-test1", "https://api1.com", "custom", "", "", "", "")
	if err != nil {
		t.Fatalf("添加 Claude 提供商失败: %v", err)
	}

	err = manager.AddProviderForApp("codex", "Provider 2", "", "sk-test2", "https://api2.com", "custom", "", "", "", "")
	if err != nil {
		t.Fatalf("添加 Codex 提供商失败: %v", err)
	}

	// 读取配置文件
	configPath := filepath.Join(tmpDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("读取配置文件失败: %v", err)
	}

	// 验证JSON格式
	var rawConfig map[string]interface{}
	if err := json.Unmarshal(data, &rawConfig); err != nil {
		t.Fatalf("解析配置文件失败: %v", err)
	}

	// 验证展平格式
	if _, ok := rawConfig["version"]; !ok {
		t.Error("配置应该包含 'version' 字段")
	}

	if _, ok := rawConfig["claude"]; !ok {
		t.Error("配置应该包含 'claude' 字段（展平格式）")
	}

	if _, ok := rawConfig["codex"]; !ok {
		t.Error("配置应该包含 'codex' 字段（展平格式）")
	}

	// 重新加载并验证数据完整性
	manager2, err := NewManagerWithDir(tmpDir)
	if err != nil {
		t.Fatalf("重新加载配置失败: %v", err)
	}

	claudeProviders := manager2.ListProvidersForApp("claude")
	if len(claudeProviders) != 1 {
		t.Errorf("Claude 提供商数量 = %d, want 1", len(claudeProviders))
	}

	codexProviders := manager2.ListProvidersForApp("codex")
	if len(codexProviders) != 1 {
		t.Errorf("Codex 提供商数量 = %d, want 1", len(codexProviders))
	}
}

// TestBackupCreation 测试备份文件创建
func TestBackupCreation(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManagerWithDir(tmpDir)
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}

	// 添加第一个提供商（创建配置文件）
	err = manager.AddProviderForApp("claude", "Provider 1", "", "sk-test1", "https://api1.com", "custom", "", "", "", "")
	if err != nil {
		t.Fatalf("添加提供商失败: %v", err)
	}

	// 添加第二个提供商（确认不会创建额外备份文件）
	err = manager.AddProviderForApp("claude", "Provider 2", "", "sk-test2", "https://api2.com", "custom", "", "", "", "")
	if err != nil {
		t.Fatalf("添加第二个提供商失败: %v", err)
	}

	// 验证备份文件存在
	configPath := filepath.Join(tmpDir, "config.json")
	backupPath := configPath + ".bak.cli"

	if _, err := os.Stat(backupPath); err == nil {
		t.Error("不应自动创建 .bak.cli 备份文件")
	}
}

func TestMoveProviderForApp(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManagerWithDir(tmpDir)
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}

	add := func(name string) {
		t.Helper()
		if err := manager.AddProviderForApp("claude", name, "", "sk-"+name, "https://api.example.com", "custom", "", "", "", ""); err != nil {
			t.Fatalf("添加配置 %s 失败: %v", name, err)
		}
	}

	add("Alpha")
	add("Beta")
	add("Gamma")

	providers := manager.ListProvidersForApp("claude")
	if len(providers) != 3 {
		t.Fatalf("期望 3 个配置, 实际 %d", len(providers))
	}

	betaID := providers[1].ID

	if err := manager.MoveProviderForApp("claude", betaID, 1); err != nil {
		t.Fatalf("下移配置失败: %v", err)
	}

	ordered := manager.ListProvidersForApp("claude")
	if ordered[2].Name != "Beta" {
		t.Fatalf("下移后 Beta 应位于末尾, 实际在 %s", ordered[2].Name)
	}

	if err := manager.MoveProviderForApp("claude", betaID, -1); err != nil {
		t.Fatalf("上移配置失败: %v", err)
	}

	ordered = manager.ListProvidersForApp("claude")
	if ordered[1].Name != "Beta" {
		t.Fatalf("上移后 Beta 应回到中间位置, 实际在 %s", ordered[1].Name)
	}

	for i, p := range ordered {
		if got := p.SortOrder; got != i+1 {
			t.Fatalf("SortOrder 应为 %d, 实际 %d", i+1, got)
		}
	}
}
