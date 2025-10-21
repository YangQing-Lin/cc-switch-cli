package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/YangQing-Lin/cc-switch-cli/internal/utils"
)

func (m *Manager) isEmptyConfig(data []byte) bool {
	return len(data) == 0 || string(data) == "" || string(data) == "{}"
}

func (m *Manager) handleEmptyConfig() error {
	fmt.Println("配置文件为空，创建默认配置...")
	m.createDefaultConfig()
	return m.Save()
}

func (m *Manager) loadAndMigrate(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return m.handleCorruptedConfig(data, err)
	}

	hasVersion, hasAppsKey := m.detectConfigVersion(raw)

	if !hasVersion && !hasAppsKey {
		if err := m.migrateV1Config(data); err == nil {
			return nil
		}
	}

	if hasAppsKey {
		if err := m.migrateV2OldConfig(data); err == nil {
			return nil
		}
	}

	return m.parseV2Config(data)
}

func (m *Manager) detectConfigVersion(raw map[string]json.RawMessage) (hasVersion, hasAppsKey bool) {
	if versionRaw, exists := raw["version"]; exists {
		var version int
		if json.Unmarshal(versionRaw, &version) == nil {
			hasVersion = true
		}
	}
	_, hasAppsKey = raw["apps"]
	return
}

func (m *Manager) handleCorruptedConfig(data []byte, err error) error {
	fmt.Printf("警告: 配置文件损坏 (%v)，将创建默认配置\n", err)

	fmt.Println()
	fmt.Println("💡 提示: 您可以使用以下命令从备份恢复配置:")
	fmt.Println("   cc-switch backup list      # 查看可用备份")
	fmt.Println("   cc-switch backup restore <backup-id>  # 恢复备份")
	fmt.Println()

	m.createDefaultConfig()
	return m.Save()
}

func (m *Manager) migrateV1Config(data []byte) error {
	var v1Config ProviderManager
	if err := json.Unmarshal(data, &v1Config); err != nil || v1Config.Providers == nil {
		return err
	}

	fmt.Println("检测到 v1 配置格式，自动迁移到 v2...")
	backupPath := m.configPath + ".v1.backup." + fmt.Sprintf("%d", time.Now().Unix())
	if os.WriteFile(backupPath, data, 0600) == nil {
		fmt.Printf("已备份 v1 配置到: %s\n", backupPath)
	}

	m.config = &MultiAppConfig{
		Version: 2,
		Apps: map[string]ProviderManager{
			"claude": v1Config,
			"codex":  {Providers: make(map[string]Provider), Current: ""},
		},
	}

	if err := m.Save(); err != nil {
		return fmt.Errorf("保存迁移后的配置失败: %w", err)
	}
	fmt.Println("v1 配置迁移完成")
	return nil
}

func (m *Manager) migrateV2OldConfig(data []byte) error {
	var oldConfig OldMultiAppConfig
	if err := json.Unmarshal(data, &oldConfig); err != nil || oldConfig.Apps == nil || len(oldConfig.Apps) == 0 {
		return err
	}

	fmt.Println("检测到旧版配置格式，自动迁移到新格式...")
	if err := m.archiveOldConfig(); err != nil {
		fmt.Printf("警告: 归档旧配置失败: %v\n", err)
	}

	m.config = &MultiAppConfig{Version: 2, Apps: oldConfig.Apps}
	m.ensureProvidersInitialized()

	if err := m.Save(); err != nil {
		return fmt.Errorf("保存迁移后的配置失败: %w", err)
	}
	fmt.Println("配置迁移完成")
	return nil
}

func (m *Manager) parseV2Config(data []byte) error {
	m.config = &MultiAppConfig{}
	if err := json.Unmarshal(data, m.config); err == nil {
		if m.config.Apps != nil && len(m.config.Apps) > 0 {
			m.ensureProvidersInitialized()
			return nil
		}

		fmt.Println("配置文件数据为空，创建默认配置...")
		m.createDefaultConfig()
		return m.Save()
	}

	fmt.Println("警告: 配置格式不支持，将创建默认配置")
	backupPath := m.configPath + ".unsupported." + fmt.Sprintf("%d", time.Now().Unix())
	if os.WriteFile(backupPath, data, 0600) == nil {
		fmt.Printf("已备份不支持的配置到: %s\n", backupPath)
	}
	m.createDefaultConfig()
	return m.Save()
}

func (m *Manager) ensureProvidersInitialized() {
	for appName, app := range m.config.Apps {
		if app.Providers == nil {
			app.Providers = make(map[string]Provider)
			m.config.Apps[appName] = app
		}
	}
}

func (m *Manager) archiveOldConfig() error {
	if !utils.FileExists(m.configPath) {
		return nil
	}

	dir := filepath.Dir(m.configPath)
	archiveDir := filepath.Join(dir, "archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return fmt.Errorf("创建归档目录失败: %w", err)
	}

	timestamp := time.Now().Unix()
	archivePath := filepath.Join(archiveDir, fmt.Sprintf("config.v2-old.backup.%d.json", timestamp))

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	if err := os.WriteFile(archivePath, data, 0600); err != nil {
		return fmt.Errorf("写入归档文件失败: %w", err)
	}

	fmt.Printf("已归档旧配置: %s\n", archivePath)
	return nil
}
