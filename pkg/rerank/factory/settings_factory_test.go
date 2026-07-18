package factory

import (
	"testing"

	"github.com/go-go-golems/geppetto/pkg/rerank"
	rerankconfig "github.com/go-go-golems/geppetto/pkg/rerank/config"
	"github.com/go-go-golems/geppetto/pkg/rerank/llamacpp"
	"github.com/go-go-golems/geppetto/pkg/security"
	aistepssettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestSupportedProviders(t *testing.T) {
	f := NewSettingsFactory(&rerankconfig.RerankConfig{}, nil, nil, nil)
	assert.Equal(t, []string{"llamacpp"}, f.SupportedProviders())
}

func TestNewProvider_RejectsMissingConfig(t *testing.T) {
	f := NewSettingsFactory(nil, nil, nil, nil)
	_, err := f.NewProvider()
	require.ErrorIs(t, err, rerank.ErrInvalidRequest)
	assert.Contains(t, err.Error(), "no rerank configuration provided")
}

func TestNewProvider_RejectsMissingType(t *testing.T) {
	f := NewSettingsFactory(&rerankconfig.RerankConfig{Engine: "m"}, nil, nil, nil)
	_, err := f.NewProvider()
	require.ErrorIs(t, err, rerank.ErrInvalidRequest)
	assert.Contains(t, err.Error(), "no rerank type specified")
}

func TestNewProvider_RejectsMissingEngine(t *testing.T) {
	f := NewSettingsFactory(&rerankconfig.RerankConfig{Type: "llamacpp"}, nil, nil, nil)
	_, err := f.NewProvider()
	require.ErrorIs(t, err, rerank.ErrInvalidRequest)
	assert.Contains(t, err.Error(), "no rerank model specified")
}

func TestNewProvider_RejectsUnsupportedType(t *testing.T) {
	f := NewSettingsFactory(&rerankconfig.RerankConfig{Type: "cohere", Engine: "m"}, nil, nil, nil)
	_, err := f.NewProvider()
	require.ErrorIs(t, err, rerank.ErrInvalidRequest)
	assert.Contains(t, err.Error(), "unsupported rerank provider type")
}

func TestNewProvider_RejectsMissingBaseURL(t *testing.T) {
	f := NewSettingsFactory(&rerankconfig.RerankConfig{Type: "llamacpp", Engine: "m"}, aistepssettings.NewAPISettings(), nil, nil)
	_, err := f.NewProvider()
	require.ErrorIs(t, err, rerank.ErrInvalidRequest)
	assert.Contains(t, err.Error(), "rerank-base-url")
}

func TestNewProvider_ConstructsLlamaCppFromDirectConfig(t *testing.T) {
	api := aistepssettings.NewAPISettings()
	api.BaseUrls["rerank-base-url"] = "http://127.0.0.1:18012"
	api.AllowHTTP["rerank"] = true
	api.AllowLocalNetworks["rerank"] = true

	cfg := &rerankconfig.RerankConfig{
		Type:             "llamacpp",
		Engine:           "qllama/bge-reranker-v2-m3:q4_k_m",
		MaxRequestBytes:  1024,
		MaxResponseBytes: 1024,
	}
	f := NewSettingsFactory(cfg, api, aistepssettings.NewClientSettings(), nil)
	provider, err := f.NewProvider()
	require.NoError(t, err)

	llamaProvider, ok := provider.(*llamacpp.Provider)
	require.True(t, ok)
	m := llamaProvider.Model()
	assert.Equal(t, "llama.cpp", m.Provider)
	assert.Equal(t, "qllama/bge-reranker-v2-m3:q4_k_m", m.Name)
}

func TestNewProvider_LocalHTTPDeniedByDefault(t *testing.T) {
	api := aistepssettings.NewAPISettings()
	api.BaseUrls["rerank-base-url"] = "http://127.0.0.1:18012"
	// AllowHTTP and AllowLocalNetworks not set for "rerank" key.

	cfg := &rerankconfig.RerankConfig{Type: "llamacpp", Engine: "m"}
	f := NewSettingsFactory(cfg, api, aistepssettings.NewClientSettings(), nil)
	_, err := f.NewProvider()
	// Constructor validates the endpoint under the outbound policy; local HTTP
	// is denied by default, so construction fails with ErrInvalidRequest.
	require.ErrorIs(t, err, rerank.ErrInvalidRequest)
}

func TestNewProvider_ResolvesInputCostFromModelInfo(t *testing.T) {
	api := aistepssettings.NewAPISettings()
	api.BaseUrls["rerank-base-url"] = "http://127.0.0.1:18012"
	api.AllowHTTP["rerank"] = true
	api.AllowLocalNetworks["rerank"] = true

	inputCost := 0.15
	modelInfo := &aistepssettings.ModelInfo{
		Cost: &aistepssettings.ModelCost{Input: inputCost, Output: 0.0},
	}
	cfg := &rerankconfig.RerankConfig{Type: "llamacpp", Engine: "m"}
	f := NewSettingsFactory(cfg, api, aistepssettings.NewClientSettings(), modelInfo)
	provider, err := f.NewProvider()
	require.NoError(t, err)
	// The provider is constructed; cost resolution is internal. We assert no
	// error and that it is a llama.cpp provider.
	_, ok := provider.(*llamacpp.Provider)
	require.True(t, ok)
}

func TestValidateInferenceSettingsForRerank_Errors(t *testing.T) {
	require.ErrorIs(t, ValidateInferenceSettingsForRerank(nil), rerank.ErrInvalidRequest)
	require.ErrorIs(t, ValidateInferenceSettingsForRerank(&aistepssettings.InferenceSettings{}), rerank.ErrInvalidRequest)

	require.ErrorIs(t, ValidateInferenceSettingsForRerank(&aistepssettings.InferenceSettings{
		Rerank: &rerankconfig.RerankConfig{Engine: "m"},
	}), rerank.ErrInvalidRequest)

	require.ErrorIs(t, ValidateInferenceSettingsForRerank(&aistepssettings.InferenceSettings{
		Rerank: &rerankconfig.RerankConfig{Type: "llamacpp"},
	}), rerank.ErrInvalidRequest)

	require.ErrorIs(t, ValidateInferenceSettingsForRerank(&aistepssettings.InferenceSettings{
		Rerank: &rerankconfig.RerankConfig{Type: "cohere", Engine: "m"},
	}), rerank.ErrInvalidRequest)

	api := aistepssettings.NewAPISettings()
	// No rerank-base-url set.
	require.ErrorIs(t, ValidateInferenceSettingsForRerank(&aistepssettings.InferenceSettings{
		Rerank: &rerankconfig.RerankConfig{Type: "llamacpp", Engine: "m"},
		API:    api,
	}), rerank.ErrInvalidRequest)
}

func TestNewSettingsFactoryFromInferenceSettings_ConstructsProvider(t *testing.T) {
	api := aistepssettings.NewAPISettings()
	api.BaseUrls["rerank-base-url"] = "http://127.0.0.1:18012"
	api.AllowHTTP["rerank"] = true
	api.AllowLocalNetworks["rerank"] = true

	in := &aistepssettings.InferenceSettings{
		API: api,
		Rerank: &rerankconfig.RerankConfig{
			Type:             "llamacpp",
			Engine:           "qllama/bge-reranker-v2-m3:q4_k_m",
			MaxRequestBytes:  2048,
			MaxResponseBytes: 1024,
		},
	}
	f, err := NewSettingsFactoryFromInferenceSettings(in)
	require.NoError(t, err)
	provider, err := f.NewProvider()
	require.NoError(t, err)
	_, ok := provider.(*llamacpp.Provider)
	require.True(t, ok)
}

func TestNewSettingsFactoryFromInferenceSettings_RejectsMissingRerank(t *testing.T) {
	in := &aistepssettings.InferenceSettings{}
	_, err := NewSettingsFactoryFromInferenceSettings(in)
	require.ErrorIs(t, err, rerank.ErrInvalidRequest)
	assert.Contains(t, err.Error(), "missing inference_settings.rerank")
}

func TestRerankConfig_YAMLRoundTrip(t *testing.T) {
	in := &aistepssettings.InferenceSettings{
		API: &aistepssettings.APISettings{
			BaseUrls:           map[string]string{"rerank-base-url": "http://127.0.0.1:18012"},
			AllowHTTP:          map[string]bool{"rerank": true},
			AllowLocalNetworks: map[string]bool{"rerank": true},
		},
		Rerank: &rerankconfig.RerankConfig{
			Type:             "llamacpp",
			Engine:           "qllama/bge-reranker-v2-m3:q4_k_m",
			MaxRequestBytes:  2097152,
			MaxResponseBytes: 1048576,
		},
	}
	b, err := yaml.Marshal(in)
	require.NoError(t, err)
	var out aistepssettings.InferenceSettings
	require.NoError(t, yaml.Unmarshal(b, &out))
	require.NotNil(t, out.Rerank)
	assert.Equal(t, "llamacpp", out.Rerank.Type)
	assert.Equal(t, "qllama/bge-reranker-v2-m3:q4_k_m", out.Rerank.Engine)
	assert.Equal(t, int64(2097152), out.Rerank.MaxRequestBytes)
	assert.Equal(t, int64(1048576), out.Rerank.MaxResponseBytes)
}

func TestRerankConfig_ClonesDeeply(t *testing.T) {
	cfg := &rerankconfig.RerankConfig{Type: "llamacpp", Engine: "m", MaxRequestBytes: 10}
	clone := cfg.Clone()
	clone.Engine = "other"
	assert.Equal(t, "m", cfg.Engine, "original must be unaffected by clone mutation")
}

func TestInferenceSettings_CloneIncludesExplicitRerank(t *testing.T) {
	in, err := aistepssettings.NewInferenceSettings()
	require.NoError(t, err)
	in.Rerank = &rerankconfig.RerankConfig{Type: "llamacpp", Engine: "original-engine"}
	clone := in.Clone()
	require.NotNil(t, clone.Rerank)
	assert.Equal(t, "original-engine", clone.Rerank.Engine)
	clone.Rerank.Engine = "cloned-engine"
	assert.Equal(t, "original-engine", in.Rerank.Engine, "clone must not share Rerank with original")
}

func TestInferenceSettings_DefaultRerankIsNilAndOmittedFromYAML(t *testing.T) {
	in, err := aistepssettings.NewInferenceSettings()
	require.NoError(t, err)
	assert.Nil(t, in.Rerank)
	encoded, err := yaml.Marshal(in)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "rerank:")
}

func TestOutboundURLOptionsResolution(t *testing.T) {
	api := aistepssettings.NewAPISettings()
	api.AllowHTTP["rerank"] = true
	api.AllowLocalNetworks["rerank"] = true
	f := NewSettingsFactory(&rerankconfig.RerankConfig{Type: "llamacpp", Engine: "m"}, api, nil, nil)
	opts := f.resolveOutboundURLOptions()
	assert.True(t, opts.AllowHTTP)
	assert.True(t, opts.AllowLocalNetworks)
}

func TestOutboundURLOptionsDefaultsDeny(t *testing.T) {
	f := NewSettingsFactory(&rerankconfig.RerankConfig{Type: "llamacpp", Engine: "m"}, aistepssettings.NewAPISettings(), nil, nil)
	opts := f.resolveOutboundURLOptions()
	assert.False(t, opts.AllowHTTP)
	assert.False(t, opts.AllowLocalNetworks)
}

// Compile-time assertion that security.OutboundURLOptions is the resolved
// type, guarding against accidental drift in the helper signature.
var _ security.OutboundURLOptions = security.OutboundURLOptions{}
