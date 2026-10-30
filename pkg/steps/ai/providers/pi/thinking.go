package pi

import (
	"slices"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
)

// Thinking budgets for budget-based APIs (Anthropic, Gemini), Pi's defaults.
// Pi clamps xhigh and max to high for budgets.
var thinkingBudgets = map[string]int{"minimal": 1024, "low": 2048, "medium": 8192, "high": 16384, "xhigh": 16384, "max": 16384}

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
	if !off && m.API != APIAnthropicMessages {
		// pi-ai's OpenAI and Google builders clamp to a supported level first;
		// its Anthropic builder maps any level to an effort or budget.
		level = clampLevel(m, level)
	}
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
	case APIAnthropicMessages:
		// pi-ai api/anthropic-messages.js: adaptive models take an effort,
		// the others a budget; "off" disables thinking unless the model's
		// level map has no "off".
		midConvo := isTrue(m.Compat.SupportsMidConvoEffort)
		if midConvo || (!off && isTrue(m.Compat.ForceAdaptiveThinking)) {
			c.ThinkingType = str("adaptive")
			if off {
				c.ReasoningEffort = str("high")
			} else {
				c.ReasoningEffort = str(anthropicEffort(m, level))
			}
			return c
		}
		if off {
			if !explicitNull(m, "off") {
				c.ThinkingType = str("disabled")
			}
			return c
		}
		if b, ok := thinkingBudgets[level]; ok {
			c.ThinkingBudget = &b
		}
	case APIGoogleGenerativeAI:
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

// anthropicEffort is pi-ai's mapThinkingLevelToEffort: a level map string
// wins, otherwise minimal and low are "low", medium "medium", and the rest
// "high".
func anthropicEffort(m Model, level string) string {
	if v, ok := m.ThinkingLevelMap[level]; ok && v != nil {
		return *v
	}
	switch level {
	case "minimal", "low":
		return "low"
	case "medium":
		return "medium"
	default:
		return "high"
	}
}

func isTrue(b *bool) bool { return b != nil && *b }

// thinkingLevels is pi-ai's EXTENDED_THINKING_LEVELS.
var thinkingLevels = []string{"off", "minimal", "low", "medium", "high", "xhigh", "max"}

// clampLevel is pi-ai's clampThinkingLevel: a level the model does not offer
// (null in thinkingLevelMap; xhigh and max only when mapped) becomes the
// nearest offered level above it, else below it.
func clampLevel(m Model, level string) string {
	offered := func(l string) bool {
		v, ok := m.ThinkingLevelMap[l]
		if ok && v == nil {
			return false
		}
		if l == "xhigh" || l == "max" {
			return ok
		}
		return true
	}
	i := slices.Index(thinkingLevels, level)
	if i < 0 {
		i = 0
	}
	if offered(level) {
		return level
	}
	for j := i + 1; j < len(thinkingLevels); j++ {
		if offered(thinkingLevels[j]) {
			return thinkingLevels[j]
		}
	}
	for j := i - 1; j >= 0; j-- {
		if offered(thinkingLevels[j]) {
			return thinkingLevels[j]
		}
	}
	return "off"
}
