package types

type ApiType string

const (
	ApiTypeOpenAI    ApiType = "openai"
	ApiTypeAnyScale  ApiType = "anyscale"
	ApiTypeFireworks ApiType = "fireworks"
	ApiTypeClaude    ApiType = "claude"
	ApiTypeGemini    ApiType = "gemini"
	// not implemented from here on down
	ApiTypeOllama     ApiType = "ollama"
	ApiTypeMistral    ApiType = "mistral"
	ApiTypePerplexity ApiType = "perplexity"
	// Cohere has connectors
	ApiTypeCohere ApiType = "cohere"
)
