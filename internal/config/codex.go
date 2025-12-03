package config

import (
	"fmt"
	"strings"
)

func generateCodexConfigTOML(providerName, baseURL, modelName, reasoning string) string {
	cleanName := strings.ToLower(providerName)
	cleanName = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return '_'
	}, cleanName)
	cleanName = strings.Trim(cleanName, "_")
	if cleanName == "" {
		cleanName = "custom"
	}

	if modelName == "" {
		modelName = "gpt-5-codex"
	}
	if reasoning == "" {
		reasoning = "high"
	}

	displayName := strings.TrimSpace(providerName)
	if displayName == "" {
		displayName = cleanName
	}

	escapedDisplayName := strings.ReplaceAll(displayName, `"`, `\"`)
	escapedBaseURL := strings.ReplaceAll(baseURL, `"`, `\"`)

	return fmt.Sprintf(`model_provider = "%s"
model = "%s"
model_reasoning_effort = "%s"
disable_response_storage = true

[model_providers.%s]
name = "%s"
base_url = "%s"
wire_api = "responses"
requires_openai_auth = true`, cleanName, modelName, reasoning, cleanName, escapedDisplayName, escapedBaseURL)
}
