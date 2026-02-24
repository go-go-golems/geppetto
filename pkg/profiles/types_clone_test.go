package profiles

import "testing"

func TestProfileClone_DeepCopyMutableFields(t *testing.T) {
	original := &Profile{
		Slug: MustProfileSlug("agent"),
		Runtime: RuntimeSpec{
			StepSettingsPatch: map[string]any{
				"ai-chat": map[string]any{
					"providers": []any{
						"openai",
						map[string]any{"model": "gpt-4o-mini"},
					},
				},
			},
			Middlewares: []MiddlewareUse{
				{
					Name: "agentmode",
					Config: map[string]any{
						"settings": []any{
							map[string]any{"enabled": true},
							"trace",
						},
					},
				},
			},
			Tools: []string{"calculator"},
		},
		Policy: PolicySpec{
			AllowOverrides:      true,
			AllowedOverrideKeys: []string{"system_prompt"},
			DeniedOverrideKeys:  []string{"tools"},
		},
		Metadata: ProfileMetadata{
			Tags: []string{"stable"},
		},
	}

	cloned := original.Clone()
	if cloned == nil {
		t.Fatalf("expected clone")
	}

	chatPatch := cloned.Runtime.StepSettingsPatch["ai-chat"].(map[string]any)
	providers := chatPatch["providers"].([]any)
	providers[0] = "anthropic"
	providers[1].(map[string]any)["model"] = "claude-3-5-sonnet"
	chatPatch["new_flag"] = true

	cloned.Runtime.Middlewares[0].Name = "updated-mw"
	mwConfig := cloned.Runtime.Middlewares[0].Config.(map[string]any)
	mwSettings := mwConfig["settings"].([]any)
	mwSettings[0].(map[string]any)["enabled"] = false
	mwConfig["settings"] = append(mwSettings, "new-entry")

	cloned.Runtime.Tools[0] = "search"
	cloned.Policy.AllowedOverrideKeys[0] = "tools"
	cloned.Policy.DeniedOverrideKeys[0] = "middlewares"
	cloned.Metadata.Tags[0] = "mutated"

	originalChatPatch := original.Runtime.StepSettingsPatch["ai-chat"].(map[string]any)
	originalProviders := originalChatPatch["providers"].([]any)
	if got := originalProviders[0].(string); got != "openai" {
		t.Fatalf("expected original provider unchanged, got %q", got)
	}
	if got := originalProviders[1].(map[string]any)["model"].(string); got != "gpt-4o-mini" {
		t.Fatalf("expected original provider model unchanged, got %q", got)
	}
	if _, ok := originalChatPatch["new_flag"]; ok {
		t.Fatalf("expected original step settings patch unchanged")
	}
	if got := original.Runtime.Middlewares[0].Name; got != "agentmode" {
		t.Fatalf("expected original middleware name unchanged, got %q", got)
	}
	originalSettings := original.Runtime.Middlewares[0].Config.(map[string]any)["settings"].([]any)
	if got := originalSettings[0].(map[string]any)["enabled"].(bool); !got {
		t.Fatalf("expected original middleware config unchanged")
	}
	if got := len(originalSettings); got != 2 {
		t.Fatalf("expected original middleware settings length=2, got %d", got)
	}
	if got := original.Runtime.Tools[0]; got != "calculator" {
		t.Fatalf("expected original tools unchanged, got %q", got)
	}
	if got := original.Policy.AllowedOverrideKeys[0]; got != "system_prompt" {
		t.Fatalf("expected original allowed override keys unchanged, got %q", got)
	}
	if got := original.Policy.DeniedOverrideKeys[0]; got != "tools" {
		t.Fatalf("expected original denied override keys unchanged, got %q", got)
	}
	if got := original.Metadata.Tags[0]; got != "stable" {
		t.Fatalf("expected original metadata tags unchanged, got %q", got)
	}
}

func TestProfileRegistryClone_DeepCopyProfilesMapAndNestedPayloads(t *testing.T) {
	original := &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {
				Slug: MustProfileSlug("default"),
				Runtime: RuntimeSpec{
					StepSettingsPatch: map[string]any{
						"ai-chat": map[string]any{"ai-engine": "gpt-4o-mini"},
					},
				},
			},
			MustProfileSlug("agent"): {
				Slug: MustProfileSlug("agent"),
				Runtime: RuntimeSpec{
					Middlewares: []MiddlewareUse{{
						Name:   "agentmode",
						Config: map[string]any{"mode": "planner"},
					}},
				},
			},
		},
		Metadata: RegistryMetadata{Tags: []string{"prod"}},
	}

	cloned := original.Clone()
	if cloned == nil {
		t.Fatalf("expected clone")
	}
	if cloned == original {
		t.Fatalf("expected distinct registry pointer")
	}

	cloned.DefaultProfileSlug = MustProfileSlug("agent")
	delete(cloned.Profiles, MustProfileSlug("default"))
	cloned.Metadata.Tags[0] = "staging"

	agent := cloned.Profiles[MustProfileSlug("agent")]
	agent.DisplayName = "Agent v2"
	agent.Runtime.Middlewares[0].Config.(map[string]any)["mode"] = "executor"

	if got := original.DefaultProfileSlug; got != MustProfileSlug("default") {
		t.Fatalf("expected original default profile unchanged, got %q", got)
	}
	if got := len(original.Profiles); got != 2 {
		t.Fatalf("expected original profiles map size=2, got %d", got)
	}
	if got := original.Metadata.Tags[0]; got != "prod" {
		t.Fatalf("expected original registry metadata unchanged, got %q", got)
	}
	if got := original.Profiles[MustProfileSlug("agent")].DisplayName; got != "" {
		t.Fatalf("expected original nested profile unchanged, got %q", got)
	}
	mode := original.Profiles[MustProfileSlug("agent")].Runtime.Middlewares[0].Config.(map[string]any)["mode"]
	if got := mode.(string); got != "planner" {
		t.Fatalf("expected original middleware config unchanged, got %q", got)
	}
}
