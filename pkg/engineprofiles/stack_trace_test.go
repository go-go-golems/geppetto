package engineprofiles

import (
	"bytes"
	"testing"
)

func TestMergeEngineProfileStackLayersWithTrace_PathHistoryAndFinalWinner(t *testing.T) {
	layers := []EngineProfileStackLayer{
		{
			RegistrySlug:      MustRegistrySlug("default"),
			EngineProfileSlug: MustEngineProfileSlug("provider"),
			EngineProfile: &EngineProfile{
				Slug: MustEngineProfileSlug("provider"),
				Runtime: RuntimeSpec{
					SystemPrompt: "provider prompt",
				},
				Metadata: EngineProfileMetadata{
					Source:  "provider-source",
					Version: 1,
				},
			},
		},
		{
			RegistrySlug:      MustRegistrySlug("default"),
			EngineProfileSlug: MustEngineProfileSlug("agent"),
			EngineProfile: &EngineProfile{
				Slug: MustEngineProfileSlug("agent"),
				Runtime: RuntimeSpec{
					SystemPrompt: "agent prompt",
				},
				Metadata: EngineProfileMetadata{
					Source:  "agent-source",
					Version: 2,
				},
			},
		},
	}

	_, trace, err := MergeEngineProfileStackLayersWithTrace(layers)
	if err != nil {
		t.Fatalf("MergeEngineProfileStackLayersWithTrace failed: %v", err)
	}

	systemPromptHistory := trace.History("/runtime/system_prompt")
	if got, want := len(systemPromptHistory), 2; got != want {
		t.Fatalf("system_prompt history length mismatch: got=%d want=%d", got, want)
	}
	if got, want := systemPromptHistory[0].EngineProfileSlug, MustEngineProfileSlug("provider"); got != want {
		t.Fatalf("system_prompt first step profile mismatch: got=%q want=%q", got, want)
	}
	if got, want := systemPromptHistory[1].EngineProfileSlug, MustEngineProfileSlug("agent"); got != want {
		t.Fatalf("system_prompt second step profile mismatch: got=%q want=%q", got, want)
	}

	if got, ok := trace.LatestValue("/runtime/system_prompt"); !ok || got.(string) != "agent prompt" {
		t.Fatalf("system_prompt latest value mismatch: ok=%v got=%#v", ok, got)
	}
}

func TestMergeEngineProfileStackLayersWithTrace_StableOrderingAndDeterministicPayload(t *testing.T) {
	layers := []EngineProfileStackLayer{
		{
			RegistrySlug:      MustRegistrySlug("default"),
			EngineProfileSlug: MustEngineProfileSlug("base"),
			EngineProfile: &EngineProfile{
				Slug: MustEngineProfileSlug("base"),
				Runtime: RuntimeSpec{
					Middlewares: []MiddlewareUse{
						{Name: "zeta"},
						{Name: "alpha"},
					},
				},
				Extensions: map[string]any{
					"custom.trace@v1": map[string]any{
						"z": "last",
						"a": "first",
					},
				},
			},
		},
	}

	_, trace, err := MergeEngineProfileStackLayersWithTrace(layers)
	if err != nil {
		t.Fatalf("MergeEngineProfileStackLayersWithTrace failed: %v", err)
	}
	if len(trace.OrderedPaths) == 0 {
		t.Fatalf("expected ordered paths")
	}

	for i := 1; i < len(trace.OrderedPaths); i++ {
		if trace.OrderedPaths[i-1] > trace.OrderedPaths[i] {
			t.Fatalf("ordered paths must be sorted: %q > %q", trace.OrderedPaths[i-1], trace.OrderedPaths[i])
		}
	}

	payloadA, err := trace.MarshalDebugPayload()
	if err != nil {
		t.Fatalf("MarshalDebugPayload failed: %v", err)
	}
	payloadB, err := trace.MarshalDebugPayload()
	if err != nil {
		t.Fatalf("MarshalDebugPayload second call failed: %v", err)
	}
	if !bytes.Equal(payloadA, payloadB) {
		t.Fatalf("debug payload must be deterministic across serializations")
	}
}
