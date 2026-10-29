package pi

import (
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
)

// Thinking budgets for budget-based APIs (Anthropic, Gemini), Pi's defaults.
var thinkingBudgets = map[string]int{"minimal": 1024, "low": 2048, "medium": 8192, "high": 16384, "xhigh": 32768}

// thinkingConfig maps a Pi thinking level onto Geppetto's per-turn knobs,
// following pi-ai's request builders (api/openai-completions.js and friends).
// A level the model's thinkingLevelMap marks null is not sent.
func thinkingConfig(m Model, level string) engine.InferenceConfig {
	var c engine.InferenceConfig
	if !m.Reasoning {
		return c
	}
	off := level == "" || level == "off"
	mapped := func(l string) (string, bool) {
		if v, ok := m.ThinkingLevelMap[l]; ok {
			if v == nil {
				return "", false
			}
			return *v, true
		}
		return l, true
	}
	str := func(s string) *string { return &s }
	switch m.API {
	case APIOpenAICompletions:
		supportsEffort := m.Compat.SupportsReasoningEffort == nil || *m.Compat.SupportsReasoningEffort
		switch m.Compat.ThinkingFormat {
		case "zai", "deepseek":
			if off {
				if m.Compat.ThinkingFormat == "zai" || !explicitNull(m, "off") {
					c.ThinkingType = str("disabled")
				}
				return c
			}
			c.ThinkingType = str("enabled")
			if supportsEffort {
				if v, ok := mapped(level); ok {
					c.ReasoningEffort = str(v)
				}
			}
		default:
			if !supportsEffort {
				return c
			}
			if off {
				if v, ok := m.ThinkingLevelMap["off"]; ok && v != nil {
					c.ReasoningEffort = str(*v)
				}
				return c
			}
			if v, ok := mapped(level); ok {
				c.ReasoningEffort = str(v)
			}
		}
	case APIOpenAIResponses, APIOpenAICodexResponses:
		if off {
			return c
		}
		if v, ok := mapped(level); ok {
			c.ReasoningEffort = str(v)
		}
	case APIAnthropicMessages, APIGoogleGenerativeAI:
		if off {
			return c
		}
		if b, ok := thinkingBudgets[level]; ok {
			c.ThinkingBudget = &b
		}
	}
	return c
}

func explicitNull(m Model, level string) bool {
	v, ok := m.ThinkingLevelMap[level]
	return ok && v == nil
}
