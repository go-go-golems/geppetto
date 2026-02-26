package profiles

import (
	"bytes"
	"testing"
)

func TestMergeProfileStackLayersWithTrace_PathHistoryAndFinalWinner(t *testing.T) {
	layers := []ProfileStackLayer{
		{
			RegistrySlug: MustRegistrySlug("default"),
			ProfileSlug:  MustProfileSlug("provider"),
			Profile: &Profile{
				Slug: MustProfileSlug("provider"),
				Runtime: RuntimeSpec{
					SystemPrompt: "provider prompt",
					StepSettingsPatch: map[string]any{
						"ai-chat": map[string]any{
							"ai-engine": "provider-engine",
						},
					},
				},
				Policy: PolicySpec{
					AllowOverrides: false,
				},
				Metadata: ProfileMetadata{
					Source:  "provider-source",
					Version: 1,
				},
			},
		},
		{
			RegistrySlug: MustRegistrySlug("default"),
			ProfileSlug:  MustProfileSlug("agent"),
			Profile: &Profile{
				Slug: MustProfileSlug("agent"),
				Runtime: RuntimeSpec{
					SystemPrompt: "agent prompt",
				},
				Policy: PolicySpec{
					AllowOverrides: true,
				},
				Metadata: ProfileMetadata{
					Source:  "agent-source",
					Version: 2,
				},
			},
		},
	}

	merged, trace, err := MergeProfileStackLayersWithTrace(layers)
	if err != nil {
		t.Fatalf("MergeProfileStackLayersWithTrace failed: %v", err)
	}

	systemPromptHistory := trace.History("/runtime/system_prompt")
	if got, want := len(systemPromptHistory), 2; got != want {
		t.Fatalf("system_prompt history length mismatch: got=%d want=%d", got, want)
	}
	if got, want := systemPromptHistory[0].ProfileSlug, MustProfileSlug("provider"); got != want {
		t.Fatalf("system_prompt first step profile mismatch: got=%q want=%q", got, want)
	}
	if got, want := systemPromptHistory[1].ProfileSlug, MustProfileSlug("agent"); got != want {
		t.Fatalf("system_prompt second step profile mismatch: got=%q want=%q", got, want)
	}

	if got, ok := trace.LatestValue("/runtime/system_prompt"); !ok || got.(string) != "agent prompt" {
		t.Fatalf("system_prompt latest value mismatch: ok=%v got=%#v", ok, got)
	}
	if got, ok := trace.LatestValue("/policy/allow_overrides"); !ok || got.(bool) != false {
		t.Fatalf("policy allow_overrides latest value mismatch: ok=%v got=%#v", ok, got)
	}
	if merged.Policy.AllowOverrides {
		t.Fatalf("expected restrictive merged policy to keep allow_overrides=false")
	}
}

func TestMergeProfileStackLayersWithTrace_StableOrderingAndDeterministicPayload(t *testing.T) {
	layers := []ProfileStackLayer{
		{
			RegistrySlug: MustRegistrySlug("default"),
			ProfileSlug:  MustProfileSlug("base"),
			Profile: &Profile{
				Slug: MustProfileSlug("base"),
				Runtime: RuntimeSpec{
					StepSettingsPatch: map[string]any{
						"nested": map[string]any{
							"z": "last",
							"a": "first",
						},
					},
				},
			},
		},
	}

	_, trace, err := MergeProfileStackLayersWithTrace(layers)
	if err != nil {
		t.Fatalf("MergeProfileStackLayersWithTrace failed: %v", err)
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
