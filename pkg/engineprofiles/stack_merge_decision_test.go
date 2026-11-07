package engineprofiles

import (
	"context"
	"testing"

	decisionconfig "github.com/go-go-golems/geppetto/pkg/decision/config"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
)

// A decision-only profile stacked on a base profile that holds the key gets both.
func TestResolveEngineProfile_DecisionProfileStacksBaseAPI(t *testing.T) {
	registry := mustNewStackTestRegistry(t, &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("djev"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("lunaroute-base"): {
				Slug: MustEngineProfileSlug("lunaroute-base"),
				InferenceSettings: &settings.InferenceSettings{
					API: &settings.APISettings{APIKeys: map[string]string{"systemone-api-key": "base-key"}},
				},
			},
			MustEngineProfileSlug("djev"): {
				Slug:  MustEngineProfileSlug("djev"),
				Stack: []EngineProfileRef{{EngineProfileSlug: MustEngineProfileSlug("lunaroute-base")}},
				InferenceSettings: &settings.InferenceSettings{
					Decision: &decisionconfig.DecisionConfig{Type: "systemone", Engine: "djev"},
				},
			},
		},
	})
	resolved, err := registry.ResolveEngineProfile(context.Background(), ResolveInput{EngineProfileSlug: MustEngineProfileSlug("djev")})
	if err != nil {
		t.Fatal(err)
	}
	s := resolved.InferenceSettings
	if s == nil || s.API == nil || s.API.APIKeys["systemone-api-key"] != "base-key" {
		t.Fatalf("api key did not stack: %#v", s)
	}
	if s.Decision == nil || s.Decision.Type != "systemone" || s.Decision.Engine != "djev" {
		t.Fatalf("decision: %#v", s.Decision)
	}
}

// An overlay that sets one field keeps the inherited others (omitempty tags).
func TestDecisionConfig_PartialOverlayPreservesInheritedFields(t *testing.T) {
	base := &settings.InferenceSettings{Decision: &decisionconfig.DecisionConfig{Type: "systemone", Engine: "djev", MaxRequestBytes: 1024}}
	for name, c := range map[string]struct {
		overlay     *decisionconfig.DecisionConfig
		typ, engine string
		maxRequest  int64
	}{
		"engine only": {&decisionconfig.DecisionConfig{Engine: "clef-flash"}, "systemone", "clef-flash", 1024},
		"limit only":  {&decisionconfig.DecisionConfig{MaxRequestBytes: 2048}, "systemone", "djev", 2048},
	} {
		merged, err := MergeInferenceSettings(base, &settings.InferenceSettings{Decision: c.overlay})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		d := merged.Decision
		if d == nil || d.Type != c.typ || d.Engine != c.engine || d.MaxRequestBytes != c.maxRequest {
			t.Fatalf("%s: %#v", name, d)
		}
	}
}
