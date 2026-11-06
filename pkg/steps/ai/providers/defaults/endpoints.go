// Package defaults owns canonical provider endpoints. Defaults identify a
// credential issuer, never a wire protocol. Connections receive the resolved URL.
package defaults

const (
	OpenAIBaseURL    = "https://api.openai.com/v1"
	AnthropicBaseURL = "https://api.anthropic.com"
	GeminiBaseURL    = "https://generativelanguage.googleapis.com"
	CodexBaseURL     = "https://chatgpt.com/backend-api"
	LunaRouteBaseURL = "https://gw.lunaroute.com/v1"
	CohereBaseURL    = "https://api.cohere.com"
)

func BaseURL(provider string) (string, bool) {
	switch provider {
	case "openai":
		return OpenAIBaseURL, true
	case "anthropic":
		return AnthropicBaseURL, true
	case "google":
		return GeminiBaseURL, true
	case "openai-codex":
		return CodexBaseURL, true
	case "lunaroute":
		return LunaRouteBaseURL, true
	case "cohere":
		return CohereBaseURL, true
	default:
		return "", false
	}
}
