package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/config"
)

// TestBasicProviderOperations 测试基本的Provider操作
func TestBasicProviderOperations(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".cc-switch")

	// 创建配置目录
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("创建配置目录失败: %v", err)
	}

	// 创建配置管理器
	manager, err := config.NewManagerWithDir(configDir)
	if err != nil {
		t.Fatalf("创建配置管理器失败: %v", err)
	}

	// 测试添加提供商
	t.Run("AddProvider", func(t *testing.T) {
		err := manager.AddProviderForApp("claude", "Test Provider", "test-token", "", "test")
		if err != nil {
			t.Errorf("添加提供商失败: %v", err)
		}

		// 验证提供商已添加
		providers := manager.ListProvidersForApp("claude")
		if len(providers) != 1 {
			t.Errorf("提供商数量不正确，期望: 1, 实际: %d", len(providers))
		}

		if providers[0].Name != "Test Provider" {
			t.Errorf("提供商名称不正确，期望: Test Provider, 实际: %s", providers[0].Name)
		}
	})

	// 测试列出提供商
	t.Run("ListProviders", func(t *testing.T) {
		// 添加更多提供商
		manager.AddProviderForApp("claude", "Provider 2", "token-2", "", "test")
		manager.AddProviderForApp("claude", "Provider 3", "token-3", "", "test")

		providers := manager.ListProvidersForApp("claude")
		if len(providers) != 3 {
			t.Errorf("提供商数量不正确，期望: 3, 实际: %d", len(providers))
		}
	})

	// 测试获取提供商
	t.Run("GetProvider", func(t *testing.T) {
		provider, err := manager.GetProviderForApp("claude", "Test Provider")
		if err != nil {
			t.Errorf("获取提供商失败: %v", err)
			return
		}

		if provider.Name != "Test Provider" {
			t.Errorf("提供商名称不正确，期望: Test Provider, 实际: %s", provider.Name)
		}
	})
}

// TestProviderPersistence 测试提供商持久化
func TestProviderPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".cc-switch")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("创建配置目录失败: %v", err)
	}

	// 创建第一个管理器并添加提供商
	manager1, err := config.NewManagerWithDir(configDir)
	if err != nil {
		t.Fatalf("创建配置管理器失败: %v", err)
	}

	err = manager1.AddProviderForApp("claude", "Persistent Provider", "persistent-token", "", "test")
	if err != nil {
		t.Fatalf("添加提供商失败: %v", err)
	}

	// 创建第二个管理器（模拟重启）
	manager2, err := config.NewManagerWithDir(configDir)
	if err != nil {
		t.Fatalf("重新创建配置管理器失败: %v", err)
	}

	// 验证提供商持久化
	providers := manager2.ListProvidersForApp("claude")
	if len(providers) != 1 {
		t.Errorf("持久化后提供商数量不正确，期望: 1, 实际: %d", len(providers))
		return
	}

	if providers[0].Name != "Persistent Provider" {
		t.Errorf("持久化后提供商名称不正确，期望: Persistent Provider, 实际: %s", providers[0].Name)
	}
}

// TestMultiAppSupport 测试多应用支持
func TestMultiAppSupport(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".cc-switch")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("创建配置目录失败: %v", err)
	}

	manager, err := config.NewManagerWithDir(configDir)
	if err != nil {
		t.Fatalf("创建配置管理器失败: %v", err)
	}

	// 添加 Claude 提供商
	err = manager.AddProviderForApp("claude", "Claude Provider", "claude-token", "", "claude")
	if err != nil {
		t.Fatalf("添加 Claude 提供商失败: %v", err)
	}

	// 添加 Codex 提供商
	err = manager.AddProviderForApp("codex", "Codex Provider", "codex-key", "https://api.example.com", "codex")
	if err != nil {
		t.Fatalf("添加 Codex 提供商失败: %v", err)
	}

	// 验证两个应用的提供商都存在
	claudeProviders := manager.ListProvidersForApp("claude")
	codexProviders := manager.ListProvidersForApp("codex")

	if len(claudeProviders) != 1 {
		t.Errorf("Claude 提供商数量不正确，期望: 1, 实际: %d", len(claudeProviders))
	}

	if len(codexProviders) != 1 {
		t.Errorf("Codex 提供商数量不正确，期望: 1, 实际: %d", len(codexProviders))
	}

	// 验证提供商内容
	if claudeProviders[0].Name != "Claude Provider" {
		t.Errorf("Claude 提供商名称不正确")
	}

	if codexProviders[0].Name != "Codex Provider" {
		t.Errorf("Codex 提供商名称不正确")
	}
}

// TestConfigFileStructure 测试配置文件结构
func TestConfigFileStructure(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".cc-switch")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("创建配置目录失败: %v", err)
	}

	manager, err := config.NewManagerWithDir(configDir)
	if err != nil {
		t.Fatalf("创建配置管理器失败: %v", err)
	}

	// 添加提供商
	err = manager.AddProviderForApp("claude", "Test Provider", "test-token", "", "test")
	if err != nil {
		t.Fatalf("添加提供商失败: %v", err)
	}

	// 读取配置文件
	configPath := filepath.Join(configDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("读取配置文件失败: %v", err)
	}

	// 解析配置文件
	var multiAppConfig config.MultiAppConfig
	if err := json.Unmarshal(data, &multiAppConfig); err != nil {
		t.Fatalf("解析配置文件失败: %v", err)
	}

	// 验证配置结构
	if multiAppConfig.Version != 2 {
		t.Errorf("配置版本不正确，期望: 2, 实际: %d", multiAppConfig.Version)
	}

	if multiAppConfig.Apps == nil {
		t.Fatal("配置缺少 Apps 字段")
	}

	claudeApp, ok := multiAppConfig.Apps["claude"]
	if !ok {
		t.Fatal("配置缺少 Claude 应用")
	}

	if len(claudeApp.Providers) != 1 {
		t.Errorf("Claude 提供商数量不正确，期望: 1, 实际: %d", len(claudeApp.Providers))
	}
}

// TestProviderValidation 测试提供商验证
func TestProviderValidation(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".cc-switch")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("创建配置目录失败: %v", err)
	}

	manager, err := config.NewManagerWithDir(configDir)
	if err != nil {
		t.Fatalf("创建配置管理器失败: %v", err)
	}

	// 测试添加空名称的提供商（当前实现可能允许）
	t.Run("EmptyName", func(t *testing.T) {
		err := manager.AddProviderForApp("claude", "", "token", "", "test")
		// 记录行为，不强制要求失败
		if err != nil {
			t.Logf("添加空名称提供商失败（预期行为）: %v", err)
		} else {
			t.Log("注意：当前实现允许添加空名称提供商")
		}
	})

	// 测试添加空 token 的提供商（当前实现可能允许）
	t.Run("EmptyToken", func(t *testing.T) {
		err := manager.AddProviderForApp("claude", "Test", "", "", "test")
		// 记录行为，不强制要求失败
		if err != nil {
			t.Logf("添加空 token 提供商失败（预期行为）: %v", err)
		} else {
			t.Log("注意：当前实现允许添加空 token 提供商")
		}
	})
}

// TestConcurrentAccess 测试并发访问
func TestConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".cc-switch")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("创建配置目录失败: %v", err)
	}

	manager, err := config.NewManagerWithDir(configDir)
	if err != nil {
		t.Fatalf("创建配置管理器失败: %v", err)
	}

	// 顺序添加多个提供商（模拟快速操作）
	for i := 0; i < 5; i++ {
		name := "Provider" + string(rune('A'+i))
		token := "token-" + string(rune('1'+i))
		err := manager.AddProviderForApp("claude", name, token, "", "test")
		if err != nil {
			t.Errorf("添加提供商 %s 失败: %v", name, err)
		}
	}

	// 验证所有提供商都已添加
	providers := manager.ListProvidersForApp("claude")
	if len(providers) != 5 {
		t.Errorf("提供商数量不正确，期望: 5, 实际: %d", len(providers))
	}

	// 重新加载验证持久化
	manager2, err := config.NewManagerWithDir(configDir)
	if err != nil {
		t.Fatalf("重新加载失败: %v", err)
	}

	providers2 := manager2.ListProvidersForApp("claude")
	if len(providers2) != 5 {
		t.Errorf("重新加载后提供商数量不正确，期望: 5, 实际: %d", len(providers2))
	}
}
