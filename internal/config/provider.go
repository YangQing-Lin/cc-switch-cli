package config

import (
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
)

// sortProviders returns providers ordered by SortOrder (if present) and falls back to CreatedAt.
func sortProviders(providerMap map[string]Provider) []Provider {
	providers := make([]Provider, 0, len(providerMap))
	for _, p := range providerMap {
		providers = append(providers, p)
	}

	sort.Slice(providers, func(i, j int) bool {
		a := providers[i]
		b := providers[j]

		switch {
		case a.SortOrder > 0 && b.SortOrder > 0:
			if a.SortOrder == b.SortOrder {
				return a.CreatedAt < b.CreatedAt
			}
			return a.SortOrder < b.SortOrder
		case a.SortOrder > 0:
			return true
		case b.SortOrder > 0:
			return false
		default:
			if a.CreatedAt == b.CreatedAt {
				return a.Name < b.Name
			}
			if a.CreatedAt == 0 || b.CreatedAt == 0 {
				return a.Name < b.Name
			}
			return a.CreatedAt < b.CreatedAt
		}
	})

	return providers
}

// normalizeSortOrder ensures providers have sequential SortOrder values starting at 1.
func normalizeSortOrder(providerMap map[string]Provider) ([]Provider, bool) {
	providers := sortProviders(providerMap)
	changed := false
	for idx := range providers {
		expected := idx + 1
		if providers[idx].SortOrder != expected {
			providers[idx].SortOrder = expected
			changed = true
		}
	}
	return providers, changed
}

// nextSortOrder calculates the next SortOrder value for insertion.
func nextSortOrder(providerMap map[string]Provider) int {
	maxOrder := 0
	count := 0
	for _, p := range providerMap {
		count++
		if p.SortOrder > maxOrder {
			maxOrder = p.SortOrder
		}
	}
	if maxOrder == 0 {
		return count + 1
	}
	return maxOrder + 1
}

func (m *Manager) AddProvider(name, apiToken, baseURL, category string) error {
	return m.AddProviderWithWebsite("claude", name, "", apiToken, baseURL, category)
}

func (m *Manager) AddProviderWithWebsite(appName, name, websiteURL, apiToken, baseURL, category string) error {
	return m.AddProviderForApp(appName, name, websiteURL, apiToken, baseURL, category, "", "")
}

func (m *Manager) AddProviderForApp(appName, name, websiteURL, apiToken, baseURL, category, claudeModel, defaultSonnetModel string) error {
	if _, exists := m.config.Apps[appName]; !exists {
		m.config.Apps[appName] = ProviderManager{
			Providers: make(map[string]Provider),
			Current:   "",
		}
	}

	app := m.config.Apps[appName]

	for _, p := range app.Providers {
		if p.Name == name {
			return fmt.Errorf("配置 '%s' 已存在", name)
		}
	}

	if providers, changed := normalizeSortOrder(app.Providers); changed {
		for _, p := range providers {
			app.Providers[p.ID] = p
		}
	}

	id := uuid.New().String()

	var settingsConfig map[string]interface{}

	switch appName {
	case "claude":
		envMap := map[string]interface{}{
			"ANTHROPIC_AUTH_TOKEN": apiToken,
			"ANTHROPIC_BASE_URL":   baseURL,
		}
		if defaultSonnetModel != "" {
			envMap["ANTHROPIC_DEFAULT_SONNET_MODEL"] = defaultSonnetModel
		}
		settingsConfig = map[string]interface{}{
			"env": envMap,
		}
		if claudeModel != "" {
			settingsConfig["model"] = claudeModel
		}
	case "codex":
		codexModel := claudeModel
		if codexModel == "" {
			codexModel = "gpt-5-codex"
		}
		reasoningEffort := defaultSonnetModel
		if reasoningEffort == "" {
			reasoningEffort = "high"
		}
		settingsConfig = map[string]interface{}{
			"auth": map[string]interface{}{
				"OPENAI_API_KEY": apiToken,
			},
			"config":                 generateCodexConfigTOML(name, baseURL, codexModel, reasoningEffort),
			"model":                  codexModel,
			"model_reasoning_effort": reasoningEffort,
		}
	default:
		return fmt.Errorf("不支持的应用: %s", appName)
	}

	provider := Provider{
		ID:             id,
		Name:           name,
		SettingsConfig: settingsConfig,
		WebsiteURL:     websiteURL,
		Category:       category,
		CreatedAt:      time.Now().UnixMilli(),
		SortOrder:      nextSortOrder(app.Providers),
	}

	app.Providers[id] = provider

	isFirstProvider := len(app.Providers) == 1
	if isFirstProvider {
		app.Current = id
	}

	m.config.Apps[appName] = app

	if err := m.Save(); err != nil {
		return err
	}

	if isFirstProvider {
		if err := m.writeProviderConfig(appName, &provider); err != nil {
			return fmt.Errorf("保存配置成功，但写入 live 配置失败: %w", err)
		}
	}

	return nil
}

func (m *Manager) AddProviderDirect(appName string, provider Provider) error {
	if _, exists := m.config.Apps[appName]; !exists {
		m.config.Apps[appName] = ProviderManager{
			Providers: make(map[string]Provider),
			Current:   "",
		}
	}

	app := m.config.Apps[appName]

	if _, exists := app.Providers[provider.ID]; exists {
		return fmt.Errorf("Provider ID '%s' 已存在", provider.ID)
	}

	if providers, changed := normalizeSortOrder(app.Providers); changed {
		for _, p := range providers {
			app.Providers[p.ID] = p
		}
	}

	for _, p := range app.Providers {
		if p.Name == provider.Name {
			return fmt.Errorf("配置名称 '%s' 已存在", provider.Name)
		}
	}

	if provider.SortOrder == 0 {
		provider.SortOrder = nextSortOrder(app.Providers)
	}

	app.Providers[provider.ID] = provider

	if len(app.Providers) == 1 {
		app.Current = provider.ID
	}

	m.config.Apps[appName] = app
	return m.Save()
}

func (m *Manager) DeleteProvider(name string) error {
	return m.DeleteProviderForApp("claude", name)
}

func (m *Manager) DeleteProviderForApp(appName, name string) error {
	app, exists := m.config.Apps[appName]
	if !exists {
		return fmt.Errorf("应用 '%s' 不存在", appName)
	}

	var targetID string
	for id, p := range app.Providers {
		if p.Name == name {
			targetID = id
			break
		}
	}

	if targetID == "" {
		return fmt.Errorf("配置 '%s' 不存在", name)
	}

	if app.Current == targetID {
		return fmt.Errorf("不能删除当前激活的配置，请先切换到其他配置")
	}

	delete(app.Providers, targetID)

	if len(app.Providers) > 0 {
		if providers, changed := normalizeSortOrder(app.Providers); changed {
			for _, p := range providers {
				app.Providers[p.ID] = p
			}
		}
	}

	m.config.Apps[appName] = app

	return m.Save()
}

func (m *Manager) GetProvider(name string) (*Provider, error) {
	return m.GetProviderForApp("claude", name)
}

func (m *Manager) GetProviderForApp(appName, name string) (*Provider, error) {
	app, exists := m.config.Apps[appName]
	if !exists {
		return nil, fmt.Errorf("应用 '%s' 不存在", appName)
	}

	for _, p := range app.Providers {
		if p.Name == name {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("配置 '%s' 不存在", name)
}

func (m *Manager) ListProviders() []Provider {
	return m.ListProvidersForApp("claude")
}

func (m *Manager) ListProvidersForApp(appName string) []Provider {
	app, exists := m.config.Apps[appName]
	if !exists {
		return []Provider{}
	}

	return sortProviders(app.Providers)
}

// MoveProviderForApp 调整指定应用下配置的排序位置，direction 为 -1 表示上移，1 表示下移
func (m *Manager) MoveProviderForApp(appName, providerID string, direction int) error {
	if direction == 0 {
		return nil
	}

	app, exists := m.config.Apps[appName]
	if !exists {
		return fmt.Errorf("应用 '%s' 不存在", appName)
	}

	providers, changed := normalizeSortOrder(app.Providers)
	if changed {
		for _, p := range providers {
			app.Providers[p.ID] = p
		}
	}

	var index int = -1
	for i, p := range providers {
		if p.ID == providerID {
			index = i
			break
		}
	}
	if index == -1 {
		return fmt.Errorf("配置不存在")
	}

	target := index + direction
	if target < 0 || target >= len(providers) {
		return nil
	}

	providers[index], providers[target] = providers[target], providers[index]

	for i := range providers {
		providers[i].SortOrder = i + 1
		app.Providers[providers[i].ID] = providers[i]
	}

	m.config.Apps[appName] = app
	return m.Save()
}

func (m *Manager) GetCurrentProvider() *Provider {
	return m.GetCurrentProviderForApp("claude")
}

func (m *Manager) GetCurrentProviderForApp(appName string) *Provider {
	app, exists := m.config.Apps[appName]
	if !exists || app.Current == "" {
		return nil
	}

	if p, ok := app.Providers[app.Current]; ok {
		return &p
	}

	return nil
}

func (m *Manager) GetConfig() (*MultiAppConfig, error) {
	configCopy := &MultiAppConfig{
		Version: m.config.Version,
		Apps:    make(map[string]ProviderManager),
	}

	for appName, appConfig := range m.config.Apps {
		providersCopy := make(map[string]Provider)
		for id, provider := range appConfig.Providers {
			providersCopy[id] = provider
		}
		configCopy.Apps[appName] = ProviderManager{
			Providers: providersCopy,
			Current:   appConfig.Current,
		}
	}

	return configCopy, nil
}

func (m *Manager) UpdateProvider(oldName, newName, apiToken, baseURL, category string) error {
	return m.UpdateProviderWithWebsite("claude", oldName, newName, "", apiToken, baseURL, category)
}

func (m *Manager) UpdateProviderWithWebsite(appName, oldName, newName, websiteURL, apiToken, baseURL, category string) error {
	return m.UpdateProviderForApp(appName, oldName, newName, websiteURL, apiToken, baseURL, category, "", "")
}

func (m *Manager) UpdateProviderForApp(appName, oldName, newName, websiteURL, apiToken, baseURL, category, claudeModel, defaultSonnetModel string) error {
	app, exists := m.config.Apps[appName]
	if !exists {
		return fmt.Errorf("应用 '%s' 不存在", appName)
	}

	var targetID string
	var targetProvider Provider
	for id, p := range app.Providers {
		if p.Name == oldName {
			targetID = id
			targetProvider = p
			break
		}
	}

	if targetID == "" {
		return fmt.Errorf("配置 '%s' 不存在", oldName)
	}

	if newName != oldName {
		for _, p := range app.Providers {
			if p.Name == newName {
				return fmt.Errorf("配置名称 '%s' 已存在", newName)
			}
		}
	}

	targetProvider.Name = newName
	if websiteURL != "" {
		targetProvider.WebsiteURL = websiteURL
	}
	if category != "" {
		targetProvider.Category = category
	}

	switch appName {
	case "claude":
		if targetProvider.SettingsConfig == nil {
			targetProvider.SettingsConfig = make(map[string]interface{})
		}
		if _, ok := targetProvider.SettingsConfig["env"]; !ok {
			targetProvider.SettingsConfig["env"] = make(map[string]interface{})
		}
		if envMap, ok := targetProvider.SettingsConfig["env"].(map[string]interface{}); ok {
			envMap["ANTHROPIC_AUTH_TOKEN"] = apiToken
			envMap["ANTHROPIC_BASE_URL"] = baseURL
			if defaultSonnetModel != "" {
				envMap["ANTHROPIC_DEFAULT_SONNET_MODEL"] = defaultSonnetModel
			} else {
				delete(envMap, "ANTHROPIC_DEFAULT_SONNET_MODEL")
			}
		}
		if claudeModel != "" {
			targetProvider.SettingsConfig["model"] = claudeModel
		} else {
			delete(targetProvider.SettingsConfig, "model")
		}
	case "codex":
		if targetProvider.SettingsConfig == nil {
			targetProvider.SettingsConfig = make(map[string]interface{})
		}
		if _, ok := targetProvider.SettingsConfig["auth"]; !ok {
			targetProvider.SettingsConfig["auth"] = make(map[string]interface{})
		}
		if authMap, ok := targetProvider.SettingsConfig["auth"].(map[string]interface{}); ok {
			authMap["OPENAI_API_KEY"] = apiToken
		}
		codexModel := claudeModel
		if codexModel == "" {
			if existingModel, ok := targetProvider.SettingsConfig["model"].(string); ok && existingModel != "" {
				codexModel = existingModel
			} else {
				codexModel = "gpt-5-codex"
			}
		}
		reasoningEffort := defaultSonnetModel
		if reasoningEffort == "" {
			if existingReasoning, ok := targetProvider.SettingsConfig["model_reasoning_effort"].(string); ok && existingReasoning != "" {
				reasoningEffort = existingReasoning
			} else {
				reasoningEffort = "high"
			}
		}
		targetProvider.SettingsConfig["model"] = codexModel
		targetProvider.SettingsConfig["model_reasoning_effort"] = reasoningEffort
		targetProvider.SettingsConfig["config"] = generateCodexConfigTOML(newName, baseURL, codexModel, reasoningEffort)
	}

	app.Providers[targetID] = targetProvider
	m.config.Apps[appName] = app

	if app.Current == targetID {
		if err := m.writeProviderConfig(appName, &targetProvider); err != nil {
			return fmt.Errorf("更新 live 配置失败: %w", err)
		}
	}

	return m.Save()
}
