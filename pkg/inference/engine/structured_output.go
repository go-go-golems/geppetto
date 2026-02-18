package engine

import (
	"fmt"
	"strings"
)

type StructuredOutputMode string

const (
	StructuredOutputModeOff        StructuredOutputMode = "off"
	StructuredOutputModeJSONSchema StructuredOutputMode = "json_schema"
)

type StructuredOutputConfig struct {
	Mode         StructuredOutputMode `json:"mode,omitempty"`
	Name         string               `json:"name,omitempty"`
	Description  string               `json:"description,omitempty"`
	Schema       map[string]any       `json:"schema,omitempty"`
	Strict       *bool                `json:"strict,omitempty"`
	RequireValid bool                 `json:"require_valid,omitempty"`
}

func (c StructuredOutputConfig) IsEnabled() bool {
	return strings.EqualFold(string(c.Mode), string(StructuredOutputModeJSONSchema))
}

func (c StructuredOutputConfig) StrictOrDefault() bool {
	if c.Strict == nil {
		return true
	}
	return *c.Strict
}

func (c StructuredOutputConfig) Validate() error {
	if !c.IsEnabled() {
		return nil
	}
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("structured output mode %q requires a non-empty schema name", c.Mode)
	}
	if len(c.Schema) == 0 {
		return fmt.Errorf("structured output mode %q requires a non-empty JSON schema", c.Mode)
	}
	return nil
}

// ResolveStructuredOutputConfig applies per-turn overrides over chat-level defaults.
// Turn override has precedence when present.
func ResolveStructuredOutputConfig(defaultCfg *StructuredOutputConfig, turnOverride *StructuredOutputConfig) *StructuredOutputConfig {
	if turnOverride != nil {
		return turnOverride
	}
	return defaultCfg
}
