package config

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	codexModelRegex     = regexp.MustCompile(`model\s*=\s*"([^"]+)"`)
	codexReasoningRegex = regexp.MustCompile(`model_reasoning_effort\s*=\s*"([^"]+)"`)
)

func ValidateProvider(name, apiToken, baseURL string) error {
	if name == "" {
		return fmt.Errorf("配置名称不能为空")
	}

	if apiToken == "" {
		return fmt.Errorf("API Token 不能为空")
	}

	if !strings.HasPrefix(apiToken, "sk-") && !strings.HasPrefix(apiToken, "88_") {
		return fmt.Errorf("API Token 格式错误，应以 'sk-' 或 '88_' 开头")
	}

	if baseURL == "" {
		return fmt.Errorf("Base URL 不能为空")
	}

	if _, err := url.Parse(baseURL); err != nil {
		return fmt.Errorf("无效的 Base URL: %w", err)
	}

	return nil
}

func ValidateMaxTokens(maxTokens string) error {
	if maxTokens == "" {
		return nil
	}
	if _, err := strconv.Atoi(maxTokens); err != nil {
		return fmt.Errorf("Max Tokens 必须是数字")
	}
	return nil
}

func MaskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

func ExtractTokenFromProvider(p *Provider) string {
	if p == nil {
		return ""
	}

	if envMap, ok := p.SettingsConfig["env"].(map[string]interface{}); ok {
		if token, ok := envMap["ANTHROPIC_AUTH_TOKEN"].(string); ok {
			return token
		}
	}

	if authMap, ok := p.SettingsConfig["auth"].(map[string]interface{}); ok {
		if token, ok := authMap["OPENAI_API_KEY"].(string); ok {
			return token
		}
	}

	return ""
}

func ExtractBaseURLFromProvider(p *Provider) string {
	if p == nil {
		return ""
	}

	if envMap, ok := p.SettingsConfig["env"].(map[string]interface{}); ok {
		if baseURL, ok := envMap["ANTHROPIC_BASE_URL"].(string); ok {
			return baseURL
		}
	}

	if configStr, ok := p.SettingsConfig["config"].(string); ok {
		re := regexp.MustCompile(`base_url\s*=\s*"([^"]+)"`)
		if matches := re.FindStringSubmatch(configStr); len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

func ExtractDefaultSonnetModelFromProvider(p *Provider) string {
	if p == nil {
		return ""
	}

	if envMap, ok := p.SettingsConfig["env"].(map[string]interface{}); ok {
		if model, ok := envMap["ANTHROPIC_DEFAULT_SONNET_MODEL"].(string); ok {
			return model
		}
	}

	return ""
}

func ExtractDefaultHaikuModelFromProvider(p *Provider) string {
	if p == nil {
		return ""
	}

	if envMap, ok := p.SettingsConfig["env"].(map[string]interface{}); ok {
		if model, ok := envMap["ANTHROPIC_DEFAULT_HAIKU_MODEL"].(string); ok {
			return model
		}
	}

	return ""
}

func ExtractDefaultOpusModelFromProvider(p *Provider) string {
	if p == nil {
		return ""
	}

	if envMap, ok := p.SettingsConfig["env"].(map[string]interface{}); ok {
		if model, ok := envMap["ANTHROPIC_DEFAULT_OPUS_MODEL"].(string); ok {
			return model
		}
	}

	return ""
}

func ExtractAnthropicModelFromProvider(p *Provider) string {
	if p == nil {
		return ""
	}

	if envMap, ok := p.SettingsConfig["env"].(map[string]interface{}); ok {
		if model, ok := envMap["ANTHROPIC_MODEL"].(string); ok && model != "" {
			return model
		}
	}

	if model, ok := p.SettingsConfig["model"].(string); ok && model != "" {
		return model
	}

	return ""
}

func ExtractModelFromProvider(p *Provider) string {
	if p == nil {
		return ""
	}

	if envMap, ok := p.SettingsConfig["env"].(map[string]interface{}); ok {
		if model, ok := envMap["ANTHROPIC_MODEL"].(string); ok && model != "" {
			return model
		}
	}

	if model, ok := p.SettingsConfig["model"].(string); ok && model != "" {
		return model
	}

	if configStr, ok := p.SettingsConfig["config"].(string); ok {
		if matches := codexModelRegex.FindStringSubmatch(configStr); len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

func ExtractCodexReasoningFromProvider(p *Provider) string {
	if p == nil {
		return ""
	}

	if reasoning, ok := p.SettingsConfig["model_reasoning_effort"].(string); ok && reasoning != "" {
		return reasoning
	}

	if configStr, ok := p.SettingsConfig["config"].(string); ok {
		if matches := codexReasoningRegex.FindStringSubmatch(configStr); len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}
