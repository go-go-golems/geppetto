package engineprofiles

import "testing"

func TestProfileClone_DeepCopyMutableFields(t *testing.T) {
	original := &EngineProfile{
		Slug: MustEngineProfileSlug("agent"),
		Stack: []EngineProfileRef{
			{EngineProfileSlug: MustEngineProfileSlug("provider-openai")},
			{RegistrySlug: MustRegistrySlug("team"), EngineProfileSlug: MustEngineProfileSlug("model-gpt4o")},
		},
		Runtime: RuntimeSpec{
			Middlewares: []MiddlewareUse{
				{
					Name:    "agentmode",
					ID:      "agent-primary",
					Enabled: boolPtr(true),
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
		Metadata: EngineProfileMetadata{
			Tags: []string{"stable"},
		},
		Extensions: map[string]any{
			"webchat.starter_suggestions@v1": map[string]any{
				"items": []any{
					"Show inventory alerts",
					map[string]any{"text": "Summarize yesterday's sales"},
				},
			},
		},
	}

	cloned := original.Clone()
	if cloned == nil {
		t.Fatalf("expected clone")
	}

	cloned.Runtime.Middlewares[0].Name = "updated-mw"
	cloned.Runtime.Middlewares[0].ID = "agent-secondary"
	*cloned.Runtime.Middlewares[0].Enabled = false
	mwConfig := cloned.Runtime.Middlewares[0].Config.(map[string]any)
	mwSettings := mwConfig["settings"].([]any)
	mwSettings[0].(map[string]any)["enabled"] = false
	mwConfig["settings"] = append(mwSettings, "new-entry")

	cloned.Runtime.Tools[0] = "search"
	cloned.Stack[0].EngineProfileSlug = MustEngineProfileSlug("provider-anthropic")
	cloned.Stack[1].RegistrySlug = MustRegistrySlug("ops")
	cloned.Metadata.Tags[0] = "mutated"
	clonedExtensions := cloned.Extensions["webchat.starter_suggestions@v1"].(map[string]any)
	clonedItems := clonedExtensions["items"].([]any)
	clonedItems[0] = "Mutated suggestion"
	clonedItems[1].(map[string]any)["text"] = "Mutated nested suggestion"
	clonedExtensions["items"] = append(clonedItems, "new suggestion")
	clonedExtensions["new_flag"] = true

	if got := original.Runtime.Middlewares[0].Name; got != "agentmode" {
		t.Fatalf("expected original middleware name unchanged, got %q", got)
	}
	if got := original.Runtime.Middlewares[0].ID; got != "agent-primary" {
		t.Fatalf("expected original middleware id unchanged, got %q", got)
	}
	if !*original.Runtime.Middlewares[0].Enabled {
		t.Fatalf("expected original middleware enabled pointer unchanged")
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
	if got := original.Stack[0].EngineProfileSlug; got != MustEngineProfileSlug("provider-openai") {
		t.Fatalf("expected original first stack ref unchanged, got %q", got)
	}
	if got := original.Stack[1].RegistrySlug; got != MustRegistrySlug("team") {
		t.Fatalf("expected original second stack registry unchanged, got %q", got)
	}
	if got := original.Metadata.Tags[0]; got != "stable" {
		t.Fatalf("expected original metadata tags unchanged, got %q", got)
	}
	originalExtensions := original.Extensions["webchat.starter_suggestions@v1"].(map[string]any)
	originalItems := originalExtensions["items"].([]any)
	if got := originalItems[0].(string); got != "Show inventory alerts" {
		t.Fatalf("expected original extension first item unchanged, got %q", got)
	}
	if got := originalItems[1].(map[string]any)["text"].(string); got != "Summarize yesterday's sales" {
		t.Fatalf("expected original extension nested item unchanged, got %q", got)
	}
	if got := len(originalItems); got != 2 {
		t.Fatalf("expected original extension items length=2, got %d", got)
	}
	if _, ok := originalExtensions["new_flag"]; ok {
		t.Fatalf("expected original extensions map unchanged")
	}
}

func TestEngineProfileRegistryClone_DeepCopyProfilesMapAndNestedPayloads(t *testing.T) {
	original := &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("default"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("default"): {
				Slug: MustEngineProfileSlug("default"),
				Runtime: RuntimeSpec{
					SystemPrompt: "hello",
				},
				Extensions: map[string]any{
					"app.note@v1": map[string]any{"enabled": true},
				},
			},
			MustEngineProfileSlug("agent"): {
				Slug: MustEngineProfileSlug("agent"),
				Runtime: RuntimeSpec{
					Middlewares: []MiddlewareUse{{
						Name:    "agentmode",
						ID:      "agent",
						Enabled: boolPtr(true),
						Config:  map[string]any{"mode": "planner"},
					}},
				},
				Extensions: map[string]any{
					"app.note@v1": map[string]any{
						"items": []any{"a", "b"},
					},
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

	cloned.DefaultEngineProfileSlug = MustEngineProfileSlug("agent")
	delete(cloned.Profiles, MustEngineProfileSlug("default"))
	cloned.Metadata.Tags[0] = "staging"

	agent := cloned.Profiles[MustEngineProfileSlug("agent")]
	agent.DisplayName = "Agent v2"
	agent.Runtime.Middlewares[0].ID = "agent-updated"
	*agent.Runtime.Middlewares[0].Enabled = false
	agent.Runtime.Middlewares[0].Config.(map[string]any)["mode"] = "executor"
	agentExt := agent.Extensions["app.note@v1"].(map[string]any)
	agentExtItems := agentExt["items"].([]any)
	agentExtItems[0] = "mutated"
	agentExt["items"] = append(agentExtItems, "c")
	agentExt["added"] = true

	if got := original.DefaultEngineProfileSlug; got != MustEngineProfileSlug("default") {
		t.Fatalf("expected original default profile unchanged, got %q", got)
	}
	if got := len(original.Profiles); got != 2 {
		t.Fatalf("expected original profiles map size=2, got %d", got)
	}
	if got := original.Metadata.Tags[0]; got != "prod" {
		t.Fatalf("expected original registry metadata unchanged, got %q", got)
	}
	if got := original.Profiles[MustEngineProfileSlug("agent")].DisplayName; got != "" {
		t.Fatalf("expected original nested profile unchanged, got %q", got)
	}
	mode := original.Profiles[MustEngineProfileSlug("agent")].Runtime.Middlewares[0].Config.(map[string]any)["mode"]
	if got := mode.(string); got != "planner" {
		t.Fatalf("expected original middleware config unchanged, got %q", got)
	}
	if got := original.Profiles[MustEngineProfileSlug("agent")].Runtime.Middlewares[0].ID; got != "agent" {
		t.Fatalf("expected original middleware id unchanged, got %q", got)
	}
	if !*original.Profiles[MustEngineProfileSlug("agent")].Runtime.Middlewares[0].Enabled {
		t.Fatalf("expected original middleware enabled pointer unchanged")
	}
	origAgentExt := original.Profiles[MustEngineProfileSlug("agent")].Extensions["app.note@v1"].(map[string]any)
	origItems := origAgentExt["items"].([]any)
	if got := origItems[0].(string); got != "a" {
		t.Fatalf("expected original extension item unchanged, got %q", got)
	}
	if got := len(origItems); got != 2 {
		t.Fatalf("expected original extension items length=2, got %d", got)
	}
	if _, ok := origAgentExt["added"]; ok {
		t.Fatalf("expected original extension map unchanged")
	}
}

func boolPtr(v bool) *bool {
	return &v
}
