package engineprofiles

import (
	"context"
	"testing"

	rerankconfig "github.com/go-go-golems/geppetto/pkg/rerank/config"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
)

func TestResolveEngineProfile_RerankProfileStacksBaseAPI(t *testing.T) {
	registry := mustNewStackTestRegistry(t, &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("bge-reranker-local"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("llamacpp-base"): {
				Slug: MustEngineProfileSlug("llamacpp-base"),
				InferenceSettings: &settings.InferenceSettings{
					API: &settings.APISettings{
						BaseUrls: map[string]string{
							"rerank-base-url": "http://127.0.0.1:18012",
						},
						AllowHTTP:          map[string]bool{"rerank": true},
						AllowLocalNetworks: map[string]bool{"rerank": true},
					},
				},
			},
			MustEngineProfileSlug("bge-reranker-local"): {
				Slug: MustEngineProfileSlug("bge-reranker-local"),
				Stack: []EngineProfileRef{
					{EngineProfileSlug: MustEngineProfileSlug("llamacpp-base")},
				},
				InferenceSettings: &settings.InferenceSettings{
					Rerank: &rerankconfig.RerankConfig{
						Type:             "llamacpp",
						Engine:           "qllama/bge-reranker-v2-m3:q4_k_m",
						MaxRequestBytes:  2097152,
						MaxResponseBytes: 1048576,
					},
				},
			},
		},
	})

	resolved, err := registry.ResolveEngineProfile(context.Background(), ResolveInput{EngineProfileSlug: MustEngineProfileSlug("bge-reranker-local")})
	if err != nil {
		t.Fatalf("ResolveEngineProfile failed: %v", err)
	}

	// The base profile's API endpoint/opt-in settings must stack onto the
	// rerank-only profile.
	if resolved.InferenceSettings == nil || resolved.InferenceSettings.API == nil {
		t.Fatalf("resolved inference settings did not include API settings: %#v", resolved.InferenceSettings)
	}
	if got := resolved.InferenceSettings.API.BaseUrls["rerank-base-url"]; got != "http://127.0.0.1:18012" {
		t.Fatalf("rerank base URL mismatch: got=%q", got)
	}
	if !resolved.InferenceSettings.API.AllowHTTP["rerank"] {
		t.Fatalf("allow_http[rerank] did not stack: %#v", resolved.InferenceSettings.API.AllowHTTP)
	}
	if !resolved.InferenceSettings.API.AllowLocalNetworks["rerank"] {
		t.Fatalf("allow_local_networks[rerank] did not stack: %#v", resolved.InferenceSettings.API.AllowLocalNetworks)
	}
	if resolved.InferenceSettings.Rerank == nil {
		t.Fatalf("resolved inference settings did not include rerank settings")
	}
	if got := resolved.InferenceSettings.Rerank.Type; got != "llamacpp" {
		t.Fatalf("rerank type mismatch: got=%q", got)
	}
	if got := resolved.InferenceSettings.Rerank.Engine; got != "qllama/bge-reranker-v2-m3:q4_k_m" {
		t.Fatalf("rerank engine mismatch: got=%q", got)
	}
	if got := resolved.InferenceSettings.Rerank.MaxRequestBytes; got != 2097152 {
		t.Fatalf("rerank max_request_bytes mismatch: got=%d", got)
	}
}

func TestRerankConfig_YAMLRoundTripThroughMerge(t *testing.T) {
	base := &settings.InferenceSettings{
		API: &settings.APISettings{
			BaseUrls: map[string]string{"rerank-base-url": "http://base.example:18012"},
		},
	}
	overlay := &settings.InferenceSettings{
		Rerank: &rerankconfig.RerankConfig{
			Type:             "llamacpp",
			Engine:           "qllama/bge-reranker-v2-m3:q4_k_m",
			MaxRequestBytes:  1024,
			MaxResponseBytes: 512,
		},
	}

	merged, err := MergeInferenceSettings(base, overlay)
	if err != nil {
		t.Fatalf("MergeInferenceSettings failed: %v", err)
	}
	if merged.Rerank == nil {
		t.Fatalf("merged settings did not include rerank")
	}
	if merged.Rerank.Engine != "qllama/bge-reranker-v2-m3:q4_k_m" {
		t.Fatalf("rerank engine did not survive merge: got=%q", merged.Rerank.Engine)
	}
	// Base API URL must survive the overlay (overlay did not specify API).
	if got := merged.API.BaseUrls["rerank-base-url"]; got != "http://base.example:18012" {
		t.Fatalf("base rerank-base-url did not survive merge: got=%q", got)
	}
}
