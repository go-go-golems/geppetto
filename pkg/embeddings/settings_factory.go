package embeddings

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"text/template"

	_ "embed"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/go-emrichen/pkg/emrichen"
	"github.com/sashabaranov/go-openai"
	"gopkg.in/yaml.v3"
)

// ProviderOption is a function that configures a provider
type ProviderOption func(*providerOptions)

type providerOptions struct {
	providerType string
	engine       string
	baseURL      string
	apiKey       string
	dimensions   int
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

// EmbeddingsConfig contains the minimal configuration needed for embeddings
type EmbeddingsConfig struct {
	// Type specifies the provider type (e.g. "openai", "ollama")
	Type string `glazed.parameter:"embeddings-type"`
	// Engine specifies the model to use (e.g. "text-embedding-ada-002" for OpenAI)
	Engine string `glazed.parameter:"embeddings-engine"`
	// Dimensions specifies the embedding dimensions (defaults to 1536 for OpenAI)
	Dimensions int `glazed.parameter:"embeddings-dimensions"`
	// APIKeys maps provider types to their API keys
	APIKeys map[settings.ApiType]string `yaml:"api_keys,omitempty" glazed.parameter:"*-api-key"`
	// BaseURLs maps provider types to their base URLs
	BaseURLs map[settings.ApiType]string `yaml:"base_urls,omitempty" glazed.parameter:"*-base-url"`
}

func NewEmbeddingsConfig() (*EmbeddingsConfig, error) {
	s := &EmbeddingsConfig{
		APIKeys:  make(map[settings.ApiType]string),
		BaseURLs: make(map[settings.ApiType]string),
	}

	p, err := NewEmbeddingsFlagsLayer()
	if err != nil {
		return nil, err
	}
	err = p.InitializeStructFromParameterDefaults(s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

//go:embed "flags/embeddings.yaml"
var embeddingsFlagsYAML []byte

type EmbeddingsFlagsLayer struct {
	*layers.ParameterLayerImpl `yaml:",inline"`
}

const EmbeddingsSlug = "embeddings"

func NewEmbeddingsFlagsLayer(options ...layers.ParameterLayerOptions) (*EmbeddingsFlagsLayer, error) {
	ret, err := layers.NewParameterLayerFromYAML(embeddingsFlagsYAML, options...)
	if err != nil {
		return nil, err
	}
	return &EmbeddingsFlagsLayer{ParameterLayerImpl: ret}, nil
}

// SettingsFactory creates embedding providers based on configuration
type SettingsFactory struct {
	config *EmbeddingsConfig
}

// NewSettingsFactory creates a new factory that uses the provided configuration
func NewSettingsFactory(config *EmbeddingsConfig) *SettingsFactory {
	return &SettingsFactory{
		config: config,
	}
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
		if options.providerType == "openai" {
			options.dimensions = 1536 // Default for OpenAI
		} else {
			return nil, fmt.Errorf("no dimensions specified for embeddings")
		}
	}

	switch options.providerType {
	case "ollama":
		baseURL := "http://localhost:11434"
		if options.baseURL != "" {
			baseURL = options.baseURL
		} else if f.config.BaseURLs != nil {
			if url, ok := f.config.BaseURLs["ollama-base-url"]; ok {
				baseURL = url
			}
		}
		return NewOllamaProvider(baseURL, options.engine, options.dimensions), nil

	case "openai":
		apiKey := options.apiKey
		if apiKey == "" && f.config.APIKeys != nil {
			if key, ok := f.config.APIKeys["openai-api-key"]; ok {
				apiKey = key
			}
		}
		if apiKey == "" {
			return nil, fmt.Errorf("no API key provided for OpenAI")
		}

		return NewOpenAIProvider(apiKey, openai.EmbeddingModel(options.engine), options.dimensions), nil

	default:
		return nil, fmt.Errorf("unsupported provider type for embeddings: %s", options.providerType)
	}
}

// NewCachedProvider creates a new cached embedding provider based on the configuration
// maxSize determines how many embeddings to keep in cache (default 1000)
func (f *SettingsFactory) NewCachedProvider(maxSize int) (Provider, error) {
	provider, err := f.NewProvider()
	if err != nil {
		return nil, err
	}
	return NewCachedProvider(provider, maxSize), nil
}

// NewSettingsFactoryFromStepSettings creates a new factory from StepSettings for backwards compatibility
func NewSettingsFactoryFromStepSettings(s *settings.StepSettings) *SettingsFactory {
	config := &EmbeddingsConfig{
		APIKeys:  make(map[settings.ApiType]string),
		BaseURLs: make(map[settings.ApiType]string),
	}

	if s.Embeddings != nil {
		if s.Embeddings.Type != nil {
			config.Type = *s.Embeddings.Type
		}
		if s.Embeddings.Engine != nil {
			config.Engine = *s.Embeddings.Engine
		}
		if s.Embeddings.Dimensions != nil {
			config.Dimensions = *s.Embeddings.Dimensions
		}
	}

	if s.API != nil {
		// Copy relevant API keys and base URLs
		for apiType, key := range s.API.APIKeys {
			config.APIKeys[apiType] = key
		}
		for apiType, url := range s.API.BaseUrls {
			config.BaseURLs[apiType] = url
		}
	}

	return NewSettingsFactory(config)
}

func (f *SettingsFactory) GetEmbeddingFuncMap() template.FuncMap {
	return template.FuncMap{
		"foobar": func() []float32 {
			return []float32{1, 2, 3}
		},
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
