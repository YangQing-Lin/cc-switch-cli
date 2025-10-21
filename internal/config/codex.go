package config

import (
	"fmt"
	"strings"
)

func generateCodexConfigTOML(providerName, baseURL, modelName string) string {
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

	return fmt.Sprintf(`model_provider = "%s"
model = "%s"
model_reasoning_effort = "high"
disable_response_storage = true

[model_providers.%s]
name = "%s"
base_url = "%s"
wire_api = "responses"`, cleanName, modelName, cleanName, cleanName, baseURL)
}
