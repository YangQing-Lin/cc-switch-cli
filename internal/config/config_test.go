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
			err := manager.AddProviderForApp(tt.appName, tt.provName, "", tt.apiToken, tt.baseURL, tt.category)
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
	manager.AddProviderForApp("claude", "Provider 1", "", "token1", "", "test")
	manager.AddProviderForApp("claude", "Provider 2", "", "token2", "", "test")
	manager.AddProviderForApp("claude", "Provider 3", "", "token3", "", "test")

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
	err = manager.AddProviderForApp("claude", "Test Provider", "", "test-token", "", "test")
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
	manager.AddProviderForApp("claude", "Provider 1", "", "token1", "", "test")
	manager.AddProviderForApp("claude", "To Delete", "", "token2", "", "test")

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
	manager.AddProviderForApp("claude", "Original", "", "old-token", "", "test")

	// 更新提供商
	err = manager.UpdateProviderForApp("claude", "Original", "New Name", "", "new-token", "", "updated")
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

	manager1.AddProviderForApp("claude", "Persistent", "", "token123", "", "test")

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
	manager.AddProviderForApp("claude", "Test", "", "token", "", "test")

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
	manager.AddProviderForApp("claude", "Provider 1", "", "token1", "", "test")
	manager.AddProviderForApp("claude", "Provider 2", "", "token2", "", "test")

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
	manager.AddProviderForApp("claude", "Provider 1", "", "claude-token-1", "", "test")
	manager.AddProviderForApp("claude", "Same Name", "", "claude-token", "", "test")
	manager.AddProviderForApp("codex", "Same Name", "", "codex-token", "", "test")

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
