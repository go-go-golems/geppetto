package main

import (
	"os"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

func applyProfileEnvironment(profile string, parsedValues *values.Values) error {
	if profile != "" {
		if err := os.Setenv("PINOCCHIO_PROFILE", profile); err != nil {
			return err
		}
	}

	var profileSettings struct {
		ProfileFile string `glazed:"profile-file"`
	}
	if err := parsedValues.DecodeSectionInto(cli.ProfileSettingsSlug, &profileSettings); err == nil {
		if strings.TrimSpace(profileSettings.ProfileFile) != "" {
			if err := os.Setenv("PINOCCHIO_PROFILE_FILE", profileSettings.ProfileFile); err != nil {
				return err
			}
		}
	}
	return nil
}

func resolvePinocchioProfile(parsedValues *values.Values) (string, error) {
	var profileSettings struct {
		Profile string `glazed:"profile"`
	}
	if err := parsedValues.DecodeSectionInto(cli.ProfileSettingsSlug, &profileSettings); err != nil {
		return "", err
	}
	return strings.TrimSpace(profileSettings.Profile), nil
}

func resolveEngineOptions(parsedValues *values.Values) (map[string]any, error) {
	opts := map[string]any{}

	var chatSettings struct {
		APIType           string `glazed:"ai-api-type"`
		Engine            string `glazed:"ai-engine"`
		MaxResponseTokens int    `glazed:"ai-max-response-tokens"`
	}
	if err := parsedValues.DecodeSectionInto("ai-chat", &chatSettings); err != nil {
		return nil, err
	}

	apiType := strings.TrimSpace(chatSettings.APIType)
	engine := strings.TrimSpace(chatSettings.Engine)
	if apiType != "" {
		opts["apiType"] = apiType
	}
	if engine != "" {
		opts["model"] = engine
	}
	if chatSettings.MaxResponseTokens > 0 {
		opts["maxTokens"] = chatSettings.MaxResponseTokens
	}

	switch strings.ToLower(apiType) {
	case "openai", "openai-responses", "anyscale", "fireworks":
		var providerSettings struct {
			APIKey  string `glazed:"openai-api-key"`
			BaseURL string `glazed:"openai-base-url"`
		}
		if err := parsedValues.DecodeSectionInto("openai-chat", &providerSettings); err == nil {
			if key := strings.TrimSpace(providerSettings.APIKey); key != "" {
				opts["apiKey"] = key
			}
			if baseURL := strings.TrimSpace(providerSettings.BaseURL); baseURL != "" {
				opts["baseURL"] = baseURL
			}
		}
	case "claude":
		var providerSettings struct {
			APIKey  string `glazed:"claude-api-key"`
			BaseURL string `glazed:"claude-base-url"`
		}
		if err := parsedValues.DecodeSectionInto("claude-chat", &providerSettings); err == nil {
			if key := strings.TrimSpace(providerSettings.APIKey); key != "" {
				opts["apiKey"] = key
			}
			if baseURL := strings.TrimSpace(providerSettings.BaseURL); baseURL != "" {
				opts["baseURL"] = baseURL
			}
		}
	case "gemini":
		var providerSettings struct {
			APIKey  string `glazed:"gemini-api-key"`
			BaseURL string `glazed:"gemini-base-url"`
		}
		if err := parsedValues.DecodeSectionInto("gemini-chat", &providerSettings); err == nil {
			if key := strings.TrimSpace(providerSettings.APIKey); key != "" {
				opts["apiKey"] = key
			}
			if baseURL := strings.TrimSpace(providerSettings.BaseURL); baseURL != "" {
				opts["baseURL"] = baseURL
			}
		}
	}

	return opts, nil
}
