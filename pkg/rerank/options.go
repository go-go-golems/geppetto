package rerank

// ProviderOption configures a provider factory during construction.
type ProviderOption func(*ProviderOptions)

// ProviderOptions holds shared, transport-neutral provider construction
// options. Concrete adapters (e.g. pkg/rerank/llamacpp) define their own
// Options structs that embed or mirror the subset they need.
type ProviderOptions struct {
	ProviderType string
	Engine       string
	BaseURL      string
	APIKey       string
}

// WithType sets the provider type (e.g. "llamacpp").
func WithType(t string) ProviderOption {
	return func(o *ProviderOptions) {
		o.ProviderType = t
	}
}

// WithEngine sets the model engine.
func WithEngine(e string) ProviderOption {
	return func(o *ProviderOptions) {
		o.Engine = e
	}
}

// WithBaseURL sets the provider base URL.
func WithBaseURL(url string) ProviderOption {
	return func(o *ProviderOptions) {
		o.BaseURL = url
	}
}

// WithAPIKey sets the provider API key.
func WithAPIKey(key string) ProviderOption {
	return func(o *ProviderOptions) {
		o.APIKey = key
	}
}
