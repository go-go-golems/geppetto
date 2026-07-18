package settings

import (
	_ "embed"
	"fmt"
	"io"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/embeddings/config"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	rerankconfig "github.com/go-go-golems/geppetto/pkg/rerank/config"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/gemini"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/ollama"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/huandu/go-clone"
	"gopkg.in/yaml.v3"
)

//go:embed "flags/inference.yaml"
var inferenceFlagsYAML []byte

type InferenceValueSection struct {
	*schema.SectionImpl `yaml:",inline"`
}

const AiInferenceSlug = "ai-inference"

func NewInferenceValueSection(options ...schema.SectionOption) (*InferenceValueSection, error) {
	ret, err := schema.NewSectionFromYAML(inferenceFlagsYAML, options...)
	if err != nil {
		return nil, err
	}

	return &InferenceValueSection{SectionImpl: ret}, nil
}

type factoryConfigFileWrapper struct {
	Factories *InferenceSettings
}

type APISettings struct {
	APIKeys  map[string]string `yaml:"api_keys,omitempty" glazed:"*-api-key"`
	BaseUrls map[string]string `yaml:"base_urls,omitempty" glazed:"*-base-url"`

	// AllowHTTP permits plain HTTP provider URLs for explicitly opted-in API
	// types. It is intentionally false by default; use only for local test
	// providers or trusted local gateways.
	AllowHTTP map[string]bool `yaml:"allow_http,omitempty"`
	// AllowLocalNetworks permits loopback/private/link-local provider targets for
	// explicitly opted-in API types. It is intentionally false by default; use
	// only for local test providers or trusted local gateways.
	AllowLocalNetworks map[string]bool `yaml:"allow_local_networks,omitempty"`
}

func NewAPISettings() *APISettings {
	return &APISettings{
		APIKeys:            map[string]string{},
		BaseUrls:           map[string]string{},
		AllowHTTP:          map[string]bool{},
		AllowLocalNetworks: map[string]bool{},
	}
}

func (s *APISettings) Clone() *APISettings {
	return clone.Clone(s).(*APISettings)
}

type InferenceSettings struct {
	API    *APISettings     `yaml:"api,omitempty"`
	Chat   *ChatSettings    `yaml:"chat,omitempty" glazed:"ai-chat"`
	OpenAI *openai.Settings `yaml:"openai,omitempty" glazed:"openai-chat"`
	Client *ClientSettings  `yaml:"client,omitempty" glazed:"ai-client"`
	Claude *claude.Settings `yaml:"claude,omitempty" glazed:"claude-chat"`
	Gemini *gemini.Settings `yaml:"gemini,omitempty" glazed:"gemini-chat"`
	Ollama *ollama.Settings `yaml:"ollama,omitempty" glazed:"ollama-chat"`
	// NOTE: Maybe we should separate the InferenceSettings struct into:
	// - Provider settings (API, OpenAI, Claude, Ollama)
	// - Chat settings (Chat, OpenAI, Claude, Ollama)
	// - Embeddings settings (Embeddings)
	Embeddings *config.EmbeddingsConfig `yaml:"embeddings,omitempty" glazed:"embeddings"`

	// Rerank provides cross-encoder reranker provider configuration. It is
	// optional; chat and embedding-only profiles remain valid without it.
	Rerank *rerankconfig.RerankConfig `yaml:"rerank,omitempty" glazed:"rerank"`

	// Inference provides engine-level defaults for per-turn inference parameters
	// (thinking budget, reasoning effort, temperature overrides, etc.).
	// These can be further overridden per-turn via Turn.Data KeyInferenceConfig.
	Inference *engine.InferenceConfig `yaml:"inference,omitempty" glazed:"ai-inference"`

	// ModelInfo describes static model-level capabilities, limits, and pricing.
	// It is loaded from profile YAML as inference_settings.model_info.
	ModelInfo *ModelInfo `yaml:"model_info,omitempty" glazed:"ai-model-info"`
}

func NewInferenceSettings() (*InferenceSettings, error) {
	chatSettings, err := NewChatSettings()
	if err != nil {
		return nil, err
	}
	openaiSettings, err := openai.NewSettings()
	if err != nil {
		return nil, err
	}
	claudeSettings, err := claude.NewSettings()
	if err != nil {
		return nil, err
	}
	geminiSettings, err := gemini.NewSettings()
	if err != nil {
		return nil, err
	}
	ollamaSettings, err := ollama.NewSettings()
	if err != nil {
		return nil, err
	}
	embeddingsSettings, err := config.NewEmbeddingsConfig()
	if err != nil {
		return nil, err
	}
	rerankSettings, err := rerankconfig.NewRerankConfig()
	if err != nil {
		return nil, err
	}
	return &InferenceSettings{
		Chat:       chatSettings,
		OpenAI:     openaiSettings,
		Client:     NewClientSettings(),
		Claude:     claudeSettings,
		Gemini:     geminiSettings,
		Ollama:     ollamaSettings,
		API:        NewAPISettings(),
		Embeddings: embeddingsSettings,
		Rerank:     rerankSettings,
	}, nil
}

func (ss *InferenceSettings) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(value.Content); i += 2 {
			key := strings.TrimSpace(value.Content[i].Value)
			if key == "api_keys" {
				return fmt.Errorf("legacy inference_settings.api_keys wrapper is no longer supported; rename it to inference_settings.api")
			}
		}
	}

	type inferenceSettingsAlias InferenceSettings
	return value.Decode((*inferenceSettingsAlias)(ss))
}

func NewInferenceSettingsFromYAML(s io.Reader) (*InferenceSettings, error) {
	stepSettings, err := NewInferenceSettings()
	if err != nil {
		return nil, err
	}

	settings_ := factoryConfigFileWrapper{
		Factories: stepSettings,
	}
	if err := yaml.NewDecoder(s).Decode(&settings_); err != nil {
		return nil, err
	}

	return settings_.Factories, nil
}

func NewInferenceSettingsFromParsedValues(parsedValues *values.Values) (*InferenceSettings, error) {
	stepSettings, err := NewInferenceSettings()
	if err != nil {
		return nil, err
	}

	err = stepSettings.UpdateFromParsedValues(parsedValues)
	if err != nil {
		return nil, err
	}

	return stepSettings, nil
}

func (ss *InferenceSettings) GetMetadata() map[string]interface{} {
	metadata := make(map[string]interface{})

	if ss.Chat != nil {
		if ss.Chat.Engine != nil {
			metadata["ai-engine"] = *ss.Chat.Engine
		}
		if ss.Chat.ApiType != nil {
			metadata["ai-api-type"] = *ss.Chat.ApiType

			baseUrl, ok := ss.API.BaseUrls[string(*ss.Chat.ApiType)]
			if ok {
				metadata["ai-base-url"] = baseUrl
			}
		}
		if ss.Chat.MaxResponseTokens != nil {
			metadata["ai-max-response-tokens"] = *ss.Chat.MaxResponseTokens
		}
		if ss.Chat.TopP != nil && *ss.Chat.TopP != 1 {
			metadata["ai-top-p"] = *ss.Chat.TopP
		}
		if ss.Chat.Temperature != nil {
			metadata["ai-temperature"] = *ss.Chat.Temperature
		}
		if len(ss.Chat.Stop) > 0 {
			metadata["ai-stop"] = ss.Chat.Stop
		}
		if ss.Chat.IsStructuredOutputEnabled() {
			metadata["ai-structured-output-mode"] = ss.Chat.StructuredOutputMode
			if ss.Chat.StructuredOutputName != "" {
				metadata["ai-structured-output-name"] = ss.Chat.StructuredOutputName
			}
			if ss.Chat.StructuredOutputDescription != "" {
				metadata["ai-structured-output-description"] = ss.Chat.StructuredOutputDescription
			}
			metadata["ai-structured-output-strict"] = ss.Chat.StructuredOutputStrictOrDefault()
			metadata["ai-structured-output-require-valid"] = ss.Chat.StructuredOutputRequireValid
		}
	}

	if ss.OpenAI != nil {
		if ss.OpenAI.N != nil && *ss.OpenAI.N != 1 {
			metadata["openai-n"] = *ss.OpenAI.N
		}
		if ss.OpenAI.PresencePenalty != nil && *ss.OpenAI.PresencePenalty != 0 {
			metadata["openai-presence-penalty"] = *ss.OpenAI.PresencePenalty
		}
		if ss.OpenAI.FrequencyPenalty != nil && *ss.OpenAI.FrequencyPenalty != 0 {
			metadata["openai-frequency-penalty"] = *ss.OpenAI.FrequencyPenalty
		}
		if len(ss.OpenAI.LogitBias) > 0 {
			metadata["openai-logit-bias"] = ss.OpenAI.LogitBias
		}
	}

	if ss.Client != nil {
		if ss.Client.Timeout != nil {
			metadata["timeout"] = ss.Client.Timeout.String()
		}
		if ss.Client.TimeoutSeconds != nil {
			metadata["timeout_second"] = *ss.Client.TimeoutSeconds
		}
		if ss.Client.Organization != nil && *ss.Client.Organization != "" {
			metadata["organization"] = *ss.Client.Organization
		}
		if ss.Client.UserAgent != nil {
			metadata["user-agent"] = *ss.Client.UserAgent
		}
		if ss.Client.ProxyURL != nil && strings.TrimSpace(*ss.Client.ProxyURL) != "" {
			metadata["proxy-url"] = RedactedProxyURL(*ss.Client.ProxyURL)
		}
		if ss.Client.ProxyFromEnvironment != nil {
			metadata["proxy-from-environment"] = *ss.Client.ProxyFromEnvironment
		}
	}

	if ss.Claude != nil {
		if ss.Claude.TopK != nil && *ss.Claude.TopK != 1 {
			metadata["claude-top-k"] = *ss.Claude.TopK
		}
		if ss.Claude.UserID != nil && *ss.Claude.UserID != "" {
			metadata["claude-user-id"] = *ss.Claude.UserID
		}
	}

	if ss.Ollama != nil {
		if ss.Ollama.Temperature != nil && *ss.Ollama.Temperature != 0 {
			metadata["ollama-temperature"] = *ss.Ollama.Temperature
		}
		if ss.Ollama.Seed != nil && *ss.Ollama.Seed != 0 {
			metadata["ollama-seed"] = *ss.Ollama.Seed
		}
		if len(ss.Ollama.Stop) > 0 {
			metadata["ollama-stop"] = ss.Ollama.Stop
		}
		if ss.Ollama.TopK != nil && *ss.Ollama.TopK != 40 {
			metadata["ollama-top-k"] = *ss.Ollama.TopK
		}
		if ss.Ollama.TopP != nil && *ss.Ollama.TopP != 0.9 {
			metadata["ollama-top-p"] = *ss.Ollama.TopP
		}
	}

	if ss.Inference != nil {
		if ss.Inference.ThinkingBudget != nil {
			metadata["inference-thinking-budget"] = *ss.Inference.ThinkingBudget
		}
		if ss.Inference.ReasoningEffort != nil {
			metadata["inference-reasoning-effort"] = *ss.Inference.ReasoningEffort
		}
		if ss.Inference.ThinkingType != nil {
			metadata["inference-thinking-type"] = *ss.Inference.ThinkingType
		}
		if ss.Inference.ReasoningSummary != nil {
			metadata["inference-reasoning-summary"] = *ss.Inference.ReasoningSummary
		}
		if ss.Inference.Seed != nil {
			metadata["inference-seed"] = *ss.Inference.Seed
		}
	}

	if ss.ModelInfo != nil {
		if ss.ModelInfo.ID != nil {
			metadata["ai-model-id"] = *ss.ModelInfo.ID
		}
		if ss.ModelInfo.Name != nil {
			metadata["ai-model-name"] = *ss.ModelInfo.Name
		}
		if ss.ModelInfo.Reasoning != nil {
			metadata["ai-model-reasoning"] = *ss.ModelInfo.Reasoning
		}
		if len(ss.ModelInfo.Input) > 0 {
			input := make([]string, 0, len(ss.ModelInfo.Input))
			for _, modality := range ss.ModelInfo.Input {
				input = append(input, string(modality))
			}
			metadata["ai-model-input"] = input
		}
		if ss.ModelInfo.ContextWindow != nil {
			metadata["ai-model-context-window"] = *ss.ModelInfo.ContextWindow
		}
		if ss.ModelInfo.QualityHighWatermark != nil {
			metadata["ai-model-quality-high-watermark"] = *ss.ModelInfo.QualityHighWatermark
		}
		if ss.ModelInfo.MaxOutputTokens != nil {
			metadata["ai-model-max-output-tokens"] = *ss.ModelInfo.MaxOutputTokens
		}
		if ss.ModelInfo.Cost != nil {
			metadata["ai-model-cost-input"] = ss.ModelInfo.Cost.Input
			metadata["ai-model-cost-output"] = ss.ModelInfo.Cost.Output
			metadata["ai-model-cost-cache-read"] = ss.ModelInfo.Cost.CacheRead
			metadata["ai-model-cost-cache-write"] = ss.ModelInfo.Cost.CacheWrite
		}
	}

	if ss.Embeddings != nil {
		if ss.Embeddings.Engine != "" {
			metadata["embeddings-engine"] = ss.Embeddings.Engine
		}
		if ss.Embeddings.Type != "" {
			metadata["embeddings-type"] = ss.Embeddings.Type
		}
		if ss.Embeddings.Dimensions != 0 {
			metadata["embeddings-dimensions"] = ss.Embeddings.Dimensions
		}
	}

	if ss.Rerank != nil {
		if ss.Rerank.Engine != "" {
			metadata["rerank-engine"] = ss.Rerank.Engine
		}
		if ss.Rerank.Type != "" {
			metadata["rerank-type"] = ss.Rerank.Type
		}
		if ss.Rerank.MaxRequestBytes != 0 {
			metadata["rerank-max-request-bytes"] = ss.Rerank.MaxRequestBytes
		}
		if ss.Rerank.MaxResponseBytes != 0 {
			metadata["rerank-max-response-bytes"] = ss.Rerank.MaxResponseBytes
		}
	}

	return metadata
}

func (s *InferenceSettings) Clone() *InferenceSettings {
	if s == nil {
		return nil
	}
	ret := &InferenceSettings{}
	if s.API != nil {
		ret.API = s.API.Clone()
	}
	if s.Chat != nil {
		ret.Chat = s.Chat.Clone()
	}
	if s.OpenAI != nil {
		ret.OpenAI = s.OpenAI.Clone()
	}
	if s.Client != nil {
		ret.Client = s.Client.Clone()
	}
	if s.Claude != nil {
		ret.Claude = s.Claude.Clone()
	}
	if s.Gemini != nil {
		ret.Gemini = s.Gemini.Clone()
	}
	if s.Ollama != nil {
		ret.Ollama = s.Ollama.Clone()
	}
	if s.Embeddings != nil {
		ret.Embeddings = s.Embeddings.Clone()
	}
	if s.Rerank != nil {
		ret.Rerank = s.Rerank.Clone()
	}
	if s.Inference != nil {
		ret.Inference = clone.Clone(s.Inference).(*engine.InferenceConfig)
	}
	if s.ModelInfo != nil {
		ret.ModelInfo = s.ModelInfo.Clone()
	}
	return ret
}

func (ss *InferenceSettings) UpdateFromParsedValues(parsedValues *values.Values) error {
	err := parsedValues.DecodeSectionInto(AiClientSlug, ss.Client)
	if err != nil {
		return err
	}

	err = parsedValues.DecodeSectionInto(AiChatSlug, ss.Chat)
	if err != nil {
		return err
	}

	err = parsedValues.DecodeSectionInto(openai.OpenAiChatSlug, ss.OpenAI)
	if err != nil {
		return err
	}

	err = parsedValues.DecodeSectionInto(claude.ClaudeChatSlug, ss.Claude)
	if err != nil {
		return err
	}

	err = parsedValues.DecodeSectionInto(gemini.GeminiChatSlug, ss.Gemini)
	if err != nil {
		return err
	}

	err = parsedValues.DecodeSectionInto(config.EmbeddingsSlug, ss.Embeddings)
	if err != nil {
		return err
	}

	err = parsedValues.DecodeSectionInto(rerankconfig.RerankSlug, ss.Rerank)
	if err != nil {
		return err
	}

	apiSlugs := []string{
		openai.OpenAiChatSlug,
		claude.ClaudeChatSlug,
		gemini.GeminiChatSlug,
		config.EmbeddingsSlug,
		rerankconfig.RerankSlug,
	}
	for _, slug := range apiSlugs {
		err = parsedValues.DecodeSectionInto(slug, ss.API)
		if err != nil {
			return err
		}
	}

	// Decode inference overrides directly into InferenceConfig.
	// Fields without defaults in the YAML stay nil (= don't override).
	if ss.Inference == nil {
		ss.Inference = &engine.InferenceConfig{}
	}
	err = parsedValues.DecodeSectionInto(AiInferenceSlug, ss.Inference)
	if err != nil {
		return err
	}

	return nil
}

func (ss *InferenceSettings) GetSummary(verbose bool) string {
	var summary strings.Builder

	// API Settings
	if ss.API != nil {
		summary.WriteString("API Settings:\n")
		for apiType, key := range ss.API.APIKeys {
			if key != "" {
				// Show only first 4 and last 4 characters of API key
				maskedKey := key
				if len(key) > 8 {
					maskedKey = key[:4] + "..." + key[len(key)-4:]
				}
				fmt.Fprintf(&summary, "  - %s API Key: %s\n", apiType, maskedKey)
			}
		}
		for apiType, url := range ss.API.BaseUrls {
			if url != "" {
				fmt.Fprintf(&summary, "  - %s Base URL: %s\n", apiType, url)
			}
		}
	}

	// Chat Settings
	if ss.Chat != nil {
		summary.WriteString("\nChat Settings:\n")
		if ss.Chat.Engine != nil {
			fmt.Fprintf(&summary, "  - Engine: %s\n", *ss.Chat.Engine)
		}
		if ss.Chat.ApiType != nil {
			fmt.Fprintf(&summary, "  - API Type: %s\n", *ss.Chat.ApiType)
		}
		if ss.Chat.MaxResponseTokens != nil {
			fmt.Fprintf(&summary, "  - Max Response Tokens: %d\n", *ss.Chat.MaxResponseTokens)
		}
		if ss.Chat.Temperature != nil {
			fmt.Fprintf(&summary, "  - Temperature: %.2f\n", *ss.Chat.Temperature)
		}
		if verbose {
			if ss.Chat.TopP != nil {
				fmt.Fprintf(&summary, "  - Top P: %.2f\n", *ss.Chat.TopP)
			}
			if len(ss.Chat.Stop) > 0 {
				fmt.Fprintf(&summary, "  - Stop Sequences: %v\n", ss.Chat.Stop)
			}
			if ss.Chat.IsStructuredOutputEnabled() {
				fmt.Fprintf(&summary, "  - Structured Output Mode: %s\n", ss.Chat.StructuredOutputMode)
				if ss.Chat.StructuredOutputName != "" {
					fmt.Fprintf(&summary, "  - Structured Output Name: %s\n", ss.Chat.StructuredOutputName)
				}
				if ss.Chat.StructuredOutputDescription != "" {
					fmt.Fprintf(&summary, "  - Structured Output Description: %s\n", ss.Chat.StructuredOutputDescription)
				}
				fmt.Fprintf(&summary, "  - Structured Output Strict: %t\n", ss.Chat.StructuredOutputStrictOrDefault())
				fmt.Fprintf(&summary, "  - Structured Output Require Valid: %t\n", ss.Chat.StructuredOutputRequireValid)
			}
		}
	}

	// OpenAI Settings
	if ss.OpenAI != nil && verbose {
		summary.WriteString("\nOpenAI Settings:\n")
		if ss.OpenAI.N != nil {
			fmt.Fprintf(&summary, "  - N: %d\n", *ss.OpenAI.N)
		}
		if ss.OpenAI.PresencePenalty != nil {
			fmt.Fprintf(&summary, "  - Presence Penalty: %.2f\n", *ss.OpenAI.PresencePenalty)
		}
		if ss.OpenAI.FrequencyPenalty != nil {
			fmt.Fprintf(&summary, "  - Frequency Penalty: %.2f\n", *ss.OpenAI.FrequencyPenalty)
		}
		if len(ss.OpenAI.LogitBias) > 0 {
			summary.WriteString("  - Logit Bias:\n")
			for token, bias := range ss.OpenAI.LogitBias {
				fmt.Fprintf(&summary, "    %s: %s\n", token, bias)
			}
		}
	}

	// Client Settings
	if ss.Client != nil && verbose {
		summary.WriteString("\nClient Settings:\n")
		if ss.Client.Timeout != nil {
			fmt.Fprintf(&summary, "  - Timeout: %s\n", ss.Client.Timeout)
		}
		if ss.Client.TimeoutSeconds != nil {
			fmt.Fprintf(&summary, "  - Timeout Seconds: %d\n", *ss.Client.TimeoutSeconds)
		}
		if ss.Client.Organization != nil && *ss.Client.Organization != "" {
			fmt.Fprintf(&summary, "  - Organization: %s\n", *ss.Client.Organization)
		}
		if ss.Client.UserAgent != nil {
			fmt.Fprintf(&summary, "  - User Agent: %s\n", *ss.Client.UserAgent)
		}
	}

	// Claude Settings
	if ss.Claude != nil && verbose {
		summary.WriteString("\nClaude Settings:\n")
		if ss.Claude.TopK != nil {
			fmt.Fprintf(&summary, "  - Top K: %d\n", *ss.Claude.TopK)
		}
		if ss.Claude.UserID != nil && *ss.Claude.UserID != "" {
			fmt.Fprintf(&summary, "  - User ID: %s\n", *ss.Claude.UserID)
		}
	}

	if ss.Gemini != nil && verbose {
		summary.WriteString("\nGemini Settings:\n")
	}

	if ss.ModelInfo != nil {
		summary.WriteString("\nModel Info:\n")
		if ss.ModelInfo.ID != nil {
			fmt.Fprintf(&summary, "  - ID: %s\n", *ss.ModelInfo.ID)
		}
		if ss.ModelInfo.Name != nil {
			fmt.Fprintf(&summary, "  - Name: %s\n", *ss.ModelInfo.Name)
		}
		if ss.ModelInfo.Reasoning != nil {
			fmt.Fprintf(&summary, "  - Reasoning: %t\n", *ss.ModelInfo.Reasoning)
		}
		if len(ss.ModelInfo.Input) > 0 {
			fmt.Fprintf(&summary, "  - Input: %v\n", ss.ModelInfo.Input)
		}
		if ss.ModelInfo.ContextWindow != nil {
			fmt.Fprintf(&summary, "  - Context Window: %d\n", *ss.ModelInfo.ContextWindow)
		}
		if ss.ModelInfo.QualityHighWatermark != nil {
			fmt.Fprintf(&summary, "  - Quality High Watermark: %d\n", *ss.ModelInfo.QualityHighWatermark)
		}
		if ss.ModelInfo.MaxOutputTokens != nil {
			fmt.Fprintf(&summary, "  - Max Output Tokens: %d\n", *ss.ModelInfo.MaxOutputTokens)
		}
		if ss.ModelInfo.Cost != nil && verbose {
			fmt.Fprintf(&summary, "  - Cost (USD/1M): input=%.6f output=%.6f cache_read=%.6f cache_write=%.6f\n",
				ss.ModelInfo.Cost.Input,
				ss.ModelInfo.Cost.Output,
				ss.ModelInfo.Cost.CacheRead,
				ss.ModelInfo.Cost.CacheWrite)
		}
	}

	// Ollama Settings
	if ss.Ollama != nil && verbose {
		summary.WriteString("\nOllama Settings:\n")
		if ss.Ollama.Temperature != nil {
			fmt.Fprintf(&summary, "  - Temperature: %.2f\n", *ss.Ollama.Temperature)
		}
		if ss.Ollama.Seed != nil {
			fmt.Fprintf(&summary, "  - Seed: %d\n", *ss.Ollama.Seed)
		}
		if len(ss.Ollama.Stop) > 0 {
			fmt.Fprintf(&summary, "  - Stop Sequences: %v\n", ss.Ollama.Stop)
		}
		if ss.Ollama.TopK != nil {
			fmt.Fprintf(&summary, "  - Top K: %d\n", *ss.Ollama.TopK)
		}
		if ss.Ollama.TopP != nil {
			fmt.Fprintf(&summary, "  - Top P: %.2f\n", *ss.Ollama.TopP)
		}
	}

	// Embeddings Settings
	if ss.Embeddings != nil {
		summary.WriteString("\nEmbeddings Settings:\n")
		if ss.Embeddings.Engine != "" {
			fmt.Fprintf(&summary, "  - Engine: %s\n", ss.Embeddings.Engine)
		}
		if ss.Embeddings.Type != "" {
			fmt.Fprintf(&summary, "  - Type: %s\n", ss.Embeddings.Type)
		}
		if ss.Embeddings.Dimensions != 0 {
			fmt.Fprintf(&summary, "  - Dimensions: %d\n", ss.Embeddings.Dimensions)
		}

		// Embeddings Cache Settings
		if ss.Embeddings.CacheType != "" {
			fmt.Fprintf(&summary, "  - Cache Type: %s\n", ss.Embeddings.CacheType)
		}
		if ss.Embeddings.CacheMaxSize != 0 {
			fmt.Fprintf(&summary, "  - Max Size: %d\n", ss.Embeddings.CacheMaxSize)
		}
		if ss.Embeddings.CacheMaxEntries != 0 {
			fmt.Fprintf(&summary, "  - Max Entries: %d\n", ss.Embeddings.CacheMaxEntries)
		}
		if ss.Embeddings.CacheDirectory != "" {
			fmt.Fprintf(&summary, "  - Cache Directory: %s\n", ss.Embeddings.CacheDirectory)
		}
	}

	// Rerank Settings
	if ss.Rerank != nil {
		summary.WriteString("\nRerank Settings:\n")
		if ss.Rerank.Engine != "" {
			fmt.Fprintf(&summary, "  - Engine: %s\n", ss.Rerank.Engine)
		}
		if ss.Rerank.Type != "" {
			fmt.Fprintf(&summary, "  - Type: %s\n", ss.Rerank.Type)
		}
		if ss.Rerank.MaxRequestBytes != 0 {
			fmt.Fprintf(&summary, "  - Max Request Bytes: %d\n", ss.Rerank.MaxRequestBytes)
		}
		if ss.Rerank.MaxResponseBytes != 0 {
			fmt.Fprintf(&summary, "  - Max Response Bytes: %d\n", ss.Rerank.MaxResponseBytes)
		}
	}

	return summary.String()
}
