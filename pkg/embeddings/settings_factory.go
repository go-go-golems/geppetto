package embeddings

import (
	"context"
	"errors"
	"fmt"
	providerdefaults "github.com/go-go-golems/geppetto/pkg/steps/ai/providers/defaults"
	"strconv"
	"strings"
	"text/template"

	_ "embed"

	"github.com/go-go-golems/geppetto/pkg/embeddings/config"
	"github.com/go-go-golems/geppetto/pkg/security"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	geppetto_openai "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-emrichen/pkg/emrichen"
	"github.com/sashabaranov/go-openai"
	"gopkg.in/yaml.v3"
)

type ProviderFactory interface {
	NewProvider(opts ...ProviderOption) (Provider, error)
	GetEmbeddingFuncMap() template.FuncMap
	GetEmbeddingTagFunc() emrichen.TagFunc
}

// ProviderOption is a function that configures a provider
type ProviderOption func(*providerOptions)

type providerOptions struct {
	providerType string
	engine       string
	baseURL      string
	apiKey       string
	dimensions   int
	connection   *connection.Connection
}

// WithType sets the provider type
func WithType(t string) ProviderOption {
	return func(o *providerOptions) {
		o.providerType = t
	}
}

// WithEngine sets the model engine
func WithEngine(e string) ProviderOption {
	return func(o *providerOptions) {
		o.engine = e
	}
}

// WithBaseURL sets the base URL
func WithBaseURL(url string) ProviderOption {
	return func(o *providerOptions) {
		o.baseURL = url
	}
}

// WithAPIKey sets the API key
func WithAPIKey(key string) ProviderOption {
	return func(o *providerOptions) {
		o.apiKey = key
	}
}

// WithDimensions sets the embedding dimensions
func WithDimensions(d int) ProviderOption {
	return func(o *providerOptions) {
		o.dimensions = d
	}
}

// SettingsFactory creates embedding providers based on configuration
type SettingsFactory struct {
	credentialPairErrors map[string]error
	config               *config.EmbeddingsConfig
	// api carries the profile-level API settings used to derive the outbound
	// URL policy for hosted providers; nil means the fail-closed default.
	api *settings.APISettings
	// client carries host-owned HTTP client settings (timeouts, proxy, TLS);
	// nil means http.DefaultClient via settings.EnsureHTTPClient.
	client *settings.ClientSettings
}

var _ ProviderFactory = &SettingsFactory{}

// NewSettingsFactory creates a new factory that uses the provided configuration
// openAIDefaultEmbeddingsBaseURL is OpenAI's API root, used when neither an
// option nor openai-base-url names one.
const openAIDefaultEmbeddingsBaseURL = providerdefaults.OpenAIBaseURL

func NewSettingsFactory(config *config.EmbeddingsConfig) *SettingsFactory {
	if config != nil {
		config = config.Clone()
	}
	return &SettingsFactory{config: config}
}

// NewProvider creates a new embedding provider based on the configuration and options
func (f *SettingsFactory) NewProvider(opts ...ProviderOption) (Provider, error) {
	if f.config == nil {
		return nil, fmt.Errorf("no configuration provided")
	}

	// Create default options from config
	options := &providerOptions{
		providerType: f.config.Type,
		engine:       f.config.Engine,
		dimensions:   f.config.Dimensions,
	}

	// Apply provided options
	for _, opt := range opts {
		opt(options)
	}

	// Validate required fields
	if options.providerType == "" {
		return nil, fmt.Errorf("no embeddings type specified")
	}

	if options.engine == "" {
		return nil, fmt.Errorf("no embeddings model specified")
	}

	// Set default dimensions if not specified
	if options.dimensions == 0 {
		switch options.providerType {
		case "openai":
			options.dimensions = 1536 // Default for OpenAI
		case "cohere":
			// 0 means the provider omits output_dimension and the Cohere API
			// uses the model's native default dimension.
		default:
			return nil, fmt.Errorf("no dimensions specified for embeddings")
		}
	}

	var provider Provider
	var err error

	c := options.connection
	if c != nil && (options.apiKey != "" || options.baseURL != "") {
		return nil, fmt.Errorf("connection already owns credentials and endpoint")
	}
	if c == nil {
		if pairErr := f.credentialPairErrors[options.providerType]; pairErr != nil {
			return nil, pairErr
		}
		key, baseURL := "", ""
		if options.apiKey != "" || options.baseURL != "" {
			if options.providerType != "ollama" && (options.apiKey == "" || options.baseURL == "") {
				return nil, fmt.Errorf("embeddings credential overrides require both API key and base URL")
			}
			key, baseURL = options.apiKey, options.baseURL
		} else {
			key = f.config.APIKeys[options.providerType+"-api-key"]
			baseURL = f.config.BaseURLs[options.providerType+"-base-url"]
		}
		defaults := map[string]string{"openai": openAIDefaultEmbeddingsBaseURL, "cohere": defaultCohereBaseURL, "ollama": "http://localhost:11434"}
		defaultURL, ok := defaults[options.providerType]
		if !ok {
			return nil, fmt.Errorf("unsupported provider type for embeddings: %s", options.providerType)
		}
		if strings.TrimSpace(baseURL) == "" {
			baseURL = defaultURL
		}
		auth := connection.Authentication{}
		if options.providerType != "ollama" {
			if strings.TrimSpace(key) == "" {
				return nil, fmt.Errorf("no API key provided for %s", map[string]string{"cohere": "Cohere", "openai": "OpenAI"}[options.providerType])
			}
			auth = connection.StaticBearer(key)
		}
		routes := connection.Routes{"embed": connection.Relative("embeddings")}
		if options.providerType == "cohere" {
			routes = CohereRoutes()
		}
		if options.providerType == "ollama" {
			routes = OllamaRoutes()
		}
		policy := settings.OutboundURLOptions(f.api, "embeddings")
		// Local Ollama is an explicitly local provider, as before this migration.
		if options.providerType == "ollama" {
			policy = security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true}
		}
		c, err = settings.BindConnection(f.client, connection.Config{Provider: options.providerType, BaseURL: baseURL, Routes: routes, Auth: auth, OutboundURL: policy})
		if err != nil {
			return nil, err
		}
	}
	switch options.providerType {
	case "openai":
		provider, err = NewOpenAIProvider(c, openai.EmbeddingModel(options.engine), options.dimensions)
	case "cohere":
		provider, err = NewCohereProvider(c, options.engine, options.dimensions)
	case "ollama":
		provider, err = NewOllamaProvider(c, options.engine, options.dimensions)
	default:
		return nil, fmt.Errorf("unsupported provider type for embeddings: %s", options.providerType)
	}
	if err != nil {
		return nil, err
	}

	// Wrap with caching if enabled
	switch f.config.CacheType {
	case "none":
		return provider, nil
	case "memory":
		return NewCachedProvider(provider, f.config.CacheMaxEntries), nil
	case "file":
		cacheOptions := []Option{WithDirectory(f.config.CacheDirectory)}
		if f.config.CacheMaxSize > 0 {
			cacheOptions = append(cacheOptions, WithMaxSize(f.config.CacheMaxSize))
		}
		if f.config.CacheMaxEntries > 0 {
			cacheOptions = append(cacheOptions, WithMaxEntries(f.config.CacheMaxEntries))
		}
		return NewDiskCacheProvider(provider, cacheOptions...)
	default:
		return provider, nil
	}
}

// NewSettingsFactoryFromParsedValues keeps section provenance through pair selection.
func NewSettingsFactoryFromParsedValues(parsedValues *values.Values) (ProviderFactory, error) {
	api := settings.NewAPISettings()
	if _, ok := parsedValues.Get(geppetto_openai.OpenAiChatSlug); ok {
		if err := parsedValues.DecodeSectionInto(geppetto_openai.OpenAiChatSlug, api); err != nil {
			return nil, err
		}
	}
	local := &config.EmbeddingsConfig{}
	if err := parsedValues.DecodeSectionInto(config.EmbeddingsSlug, local); err != nil {
		return nil, err
	}
	if _, ok := parsedValues.Get(config.EmbeddingsApiKeySlug); ok {
		if err := parsedValues.DecodeSectionInto(config.EmbeddingsApiKeySlug, local); err != nil {
			return nil, err
		}
	}
	return NewSettingsFactoryFromInferenceSettings(&settings.InferenceSettings{API: api, Embeddings: local}), nil
}

// NewSettingsFactoryFromInferenceSettings creates a factory while retaining credential-layer provenance
func NewSettingsFactoryFromInferenceSettings(s *settings.InferenceSettings) *SettingsFactory {
	config := &config.EmbeddingsConfig{
		APIKeys:  make(map[string]string),
		BaseURLs: make(map[string]string),
	}

	if s.Embeddings != nil {
		if s.Embeddings.Type != "" {
			config.Type = s.Embeddings.Type
		}
		if s.Embeddings.Engine != "" {
			config.Engine = s.Embeddings.Engine
		}
		if s.Embeddings.Dimensions != 0 {
			config.Dimensions = s.Embeddings.Dimensions
		}

		if s.Embeddings.CacheType != "" {
			config.CacheType = s.Embeddings.CacheType
		}
		if s.Embeddings.CacheMaxSize != 0 {
			config.CacheMaxSize = s.Embeddings.CacheMaxSize
		}
		if s.Embeddings.CacheMaxEntries != 0 {
			config.CacheMaxEntries = s.Embeddings.CacheMaxEntries
		}
		if s.Embeddings.CacheDirectory != "" {
			config.CacheDirectory = s.Embeddings.CacheDirectory
		}
	}

	if s.API != nil {
		// Copy top-level API keys and base URLs as defaults.
		for apiType, key := range s.API.APIKeys {
			config.APIKeys[apiType] = key
		}
		for apiType, url := range s.API.BaseUrls {
			config.BaseURLs[apiType] = url
		}
	}

	pairErrors := map[string]error{}
	if s.Embeddings != nil {
		for _, provider := range []string{"openai", "cohere", "ollama"} {
			key := strings.TrimSpace(s.Embeddings.APIKeys[provider+"-api-key"])
			endpoint := strings.TrimSpace(s.Embeddings.BaseURLs[provider+"-base-url"])
			if key == "" && endpoint == "" {
				continue
			}
			// A local layer replaces the whole binding. It cannot inherit the other field.
			delete(config.APIKeys, provider+"-api-key")
			delete(config.BaseURLs, provider+"-base-url")
			if provider != "ollama" && (key == "" || endpoint == "") {
				pairErrors[provider] = fmt.Errorf("embedding-local %s credentials require both api_keys.%s-api-key and base_urls.%s-base-url", provider, provider, provider)
			}
			config.APIKeys[provider+"-api-key"] = key
			config.BaseURLs[provider+"-base-url"] = endpoint
		}
	}

	factory := NewSettingsFactory(config)
	factory.credentialPairErrors = pairErrors
	factory.api = s.API
	if s.API != nil {
		factory.api = s.API.Clone()
	}
	factory.client = s.Client
	return factory
}

func (f *SettingsFactory) GetEmbeddingFuncMap() template.FuncMap {
	return template.FuncMap{
		"embeddings": func(text string) ([]float32, error) {
			provider, err := f.NewProvider()
			if err != nil {
				return nil, err
			}
			return provider.GenerateEmbedding(context.Background(), text)
		},
	}
}

func (f *SettingsFactory) GetEmbeddingTagFunc() emrichen.TagFunc {
	return func(ei *emrichen.Interpreter, node *yaml.Node) (*yaml.Node, error) {
		if node.Kind != yaml.MappingNode {
			return nil, errors.New("!Embeddings requires a mapping node")
		}

		// Parse arguments with text being required and config being optional
		args, err := ei.ParseArgs(node, []emrichen.ParsedVariable{
			{Name: "text", Required: true},
			{Name: "config", Required: false, Expand: true},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to parse !Embeddings arguments: %w", err)
		}

		textNode, ok := args["text"]
		if !ok || textNode == nil {
			return nil, errors.New("!Embeddings requires a 'text' argument")
		}

		text, ok := textNode.Value, textNode.Kind == yaml.ScalarNode
		if !ok {
			return nil, errors.New("!Embeddings 'text' argument must be a string")
		}

		// Create provider with potential config overrides
		var provider Provider
		if configNode, ok := args["config"]; ok && configNode != nil {
			// Convert config to options
			var opts []ProviderOption

			// Process each config option
			if configNode.Kind != yaml.MappingNode {
				return nil, errors.New("config must be a mapping")
			}

			configArgs, err := ei.ParseArgs(configNode, []emrichen.ParsedVariable{
				{Name: "type", Required: false},
				{Name: "engine", Required: false},
				{Name: "dimensions", Required: false},
				{Name: "base_url", Required: false},
				{Name: "api_key", Required: false},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to parse config arguments: %w", err)
			}

			if typeNode, ok := configArgs["type"]; ok && typeNode.Kind == yaml.ScalarNode {
				opts = append(opts, WithType(typeNode.Value))
			}
			if engineNode, ok := configArgs["engine"]; ok && engineNode.Kind == yaml.ScalarNode {
				opts = append(opts, WithEngine(engineNode.Value))
			}
			if dimNode, ok := configArgs["dimensions"]; ok && dimNode.Kind == yaml.ScalarNode {
				if dim, err := strconv.Atoi(dimNode.Value); err == nil {
					opts = append(opts, WithDimensions(dim))
				}
			}
			if urlNode, ok := configArgs["base_url"]; ok && urlNode.Kind == yaml.ScalarNode {
				opts = append(opts, WithBaseURL(urlNode.Value))
			}
			if keyNode, ok := configArgs["api_key"]; ok && keyNode.Kind == yaml.ScalarNode {
				opts = append(opts, WithAPIKey(keyNode.Value))
			}
			provider, err = f.NewProvider(opts...)
			if err != nil {
				return nil, fmt.Errorf("failed to create embeddings provider: %w", err)
			}
		} else {
			provider, err = f.NewProvider()
			if err != nil {
				return nil, fmt.Errorf("failed to create embeddings provider: %w", err)
			}
		}

		embedding, err := provider.GenerateEmbedding(context.Background(), text)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding: %w", err)
		}

		return emrichen.ValueToNode(embedding)
	}
}

// WithConnection supplies a complete authority without separate key/URL overrides.
func WithConnection(c *connection.Connection) ProviderOption {
	return func(o *providerOptions) { o.connection = c }
}
