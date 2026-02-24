package profiles

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

func TestStoreRegistryResolve_DefaultProfileFallbackAndMetadata(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	mustUpsertRegistry(t, store, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("agent"),
		Metadata:           RegistryMetadata{Source: "file"},
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("agent"): {
				Slug: MustProfileSlug("agent"),
				Runtime: RuntimeSpec{StepSettingsPatch: map[string]any{
					"ai-chat": map[string]any{"ai-engine": "gpt-4.1-mini"},
				}},
				Metadata: ProfileMetadata{Version: 3, Source: "db"},
			},
		},
	})

	registry := mustNewStoreRegistry(t, store)
	resolved, err := registry.ResolveEffectiveProfile(ctx, ResolveInput{})
	if err != nil {
		t.Fatalf("ResolveEffectiveProfile returned error: %v", err)
	}

	if resolved.ProfileSlug != MustProfileSlug("agent") {
		t.Fatalf("expected fallback profile=agent, got %q", resolved.ProfileSlug)
	}
	if resolved.RuntimeKey != MustRuntimeKey("agent") {
		t.Fatalf("expected runtime key=agent, got %q", resolved.RuntimeKey)
	}
	if resolved.Metadata["profile.registry"] != "default" {
		t.Fatalf("metadata profile.registry mismatch: %#v", resolved.Metadata)
	}
	if resolved.Metadata["profile.slug"] != "agent" {
		t.Fatalf("metadata profile.slug mismatch: %#v", resolved.Metadata)
	}
	if resolved.Metadata["profile.version"] != uint64(3) {
		t.Fatalf("metadata profile.version mismatch: %#v", resolved.Metadata)
	}
	if resolved.Metadata["profile.source"] != "db" {
		t.Fatalf("metadata profile.source mismatch: %#v", resolved.Metadata)
	}
	if resolved.RuntimeFingerprint == "" {
		t.Fatalf("runtime fingerprint must be non-empty")
	}
	if resolved.EffectiveStepSettings == nil || resolved.EffectiveStepSettings.Chat == nil || resolved.EffectiveStepSettings.Chat.Engine == nil {
		t.Fatalf("expected resolved step settings with chat engine")
	}
	if *resolved.EffectiveStepSettings.Chat.Engine != "gpt-4.1-mini" {
		t.Fatalf("expected engine from profile patch, got %q", *resolved.EffectiveStepSettings.Chat.Engine)
	}
}

func TestStoreRegistryResolve_UnknownMapping(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	mustUpsertRegistry(t, store, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
		},
	})

	registry := mustNewStoreRegistry(t, store)

	_, err := registry.ResolveEffectiveProfile(ctx, ResolveInput{RegistrySlug: MustRegistrySlug("missing")})
	if !errors.Is(err, ErrRegistryNotFound) {
		t.Fatalf("expected ErrRegistryNotFound, got %v", err)
	}

	_, err = registry.ResolveEffectiveProfile(ctx, ResolveInput{ProfileSlug: MustProfileSlug("missing")})
	if !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("expected ErrProfileNotFound, got %v", err)
	}
}

func TestStoreRegistryResolve_EmptyProfileFallbackToDefaultSlug(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	// Build an intentionally invalid in-memory registry state (no default_profile_slug)
	// to assert fallback behavior in resolveProfileSlugForRegistry.
	store.registries[MustRegistrySlug("default")] = (&ProfileRegistry{
		Slug: MustRegistrySlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {
				Slug: MustProfileSlug("default"),
				Runtime: RuntimeSpec{
					SystemPrompt: "fallback profile",
				},
			},
		},
	}).Clone()

	registry := mustNewStoreRegistry(t, store)
	resolved, err := registry.ResolveEffectiveProfile(ctx, ResolveInput{})
	if err != nil {
		t.Fatalf("ResolveEffectiveProfile returned error: %v", err)
	}
	if got := resolved.ProfileSlug; got != MustProfileSlug("default") {
		t.Fatalf("expected fallback profile slug=default, got %q", got)
	}
}

func TestStoreRegistryResolve_EmptyProfileWithoutDefaultReturnsValidation(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	// Build an intentionally invalid in-memory registry state (no default_profile_slug)
	// to assert validation behavior when neither default_profile_slug nor "default" exists.
	store.registries[MustRegistrySlug("default")] = (&ProfileRegistry{
		Slug: MustRegistrySlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("agent"): {Slug: MustProfileSlug("agent")},
		},
	}).Clone()

	registry := mustNewStoreRegistry(t, store)
	_, err := registry.ResolveEffectiveProfile(ctx, ResolveInput{})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected ValidationError type, got %T", err)
	}
	if got, want := verr.Field, "profile.slug"; got != want {
		t.Fatalf("validation field mismatch: got=%q want=%q", got, want)
	}
}

func TestStoreRegistryResolve_PrecendenceAndPolicy(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	mustUpsertRegistry(t, store, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {
				Slug: MustProfileSlug("default"),
				Runtime: RuntimeSpec{
					StepSettingsPatch: map[string]any{
						"ai-chat": map[string]any{
							"ai-engine": "profile-engine",
						},
					},
					SystemPrompt: "profile prompt",
					Middlewares:  []MiddlewareUse{{Name: "profile-mw"}},
					Tools:        []string{"profile-tool"},
				},
				Policy: PolicySpec{
					AllowOverrides:      true,
					AllowedOverrideKeys: []string{"system_prompt", "middlewares", "tools", "step_settings_patch"},
				},
			},
		},
	})

	registry := mustNewStoreRegistry(t, store)
	base, err := settings.NewStepSettings()
	if err != nil {
		t.Fatalf("NewStepSettings returned error: %v", err)
	}
	base.Chat.Engine = stringPtr("base-engine")

	resolved, err := registry.ResolveEffectiveProfile(ctx, ResolveInput{
		BaseStepSettings: base,
		RequestOverrides: map[string]any{
			"system_prompt": "request prompt",
			"middlewares": []any{
				map[string]any{"name": "request-mw"},
			},
			"tools": []any{"request-tool"},
			"step_settings_patch": map[string]any{
				"ai-chat": map[string]any{"ai-engine": "request-engine"},
			},
		},
	})
	if err != nil {
		t.Fatalf("ResolveEffectiveProfile returned error: %v", err)
	}

	if got := resolved.EffectiveRuntime.SystemPrompt; got != "request prompt" {
		t.Fatalf("expected system prompt override, got %q", got)
	}
	if got := resolved.EffectiveRuntime.Middlewares[0].Name; got != "request-mw" {
		t.Fatalf("expected middleware override, got %q", got)
	}
	if got := resolved.EffectiveRuntime.Tools[0]; got != "request-tool" {
		t.Fatalf("expected tool override, got %q", got)
	}
	if resolved.EffectiveStepSettings == nil || resolved.EffectiveStepSettings.Chat == nil || resolved.EffectiveStepSettings.Chat.Engine == nil {
		t.Fatalf("expected effective step settings with chat engine")
	}
	if got := *resolved.EffectiveStepSettings.Chat.Engine; got != "request-engine" {
		t.Fatalf("expected request engine to win precedence, got %q", got)
	}

	_, err = registry.ResolveEffectiveProfile(ctx, ResolveInput{
		RequestOverrides: map[string]any{"tools": []any{"blocked"}},
		ProfileSlug:      MustProfileSlug("locked"),
	})
	if !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("expected missing profile for locked check setup, got %v", err)
	}

	mustUpsertProfile(t, store, MustRegistrySlug("default"), &Profile{
		Slug: MustProfileSlug("locked"),
		Policy: PolicySpec{
			AllowOverrides: false,
		},
	})
	_, err = registry.ResolveEffectiveProfile(ctx, ResolveInput{
		ProfileSlug:      MustProfileSlug("locked"),
		RequestOverrides: map[string]any{"tools": []any{"blocked"}},
	})
	if !errors.Is(err, ErrPolicyViolation) {
		t.Fatalf("expected ErrPolicyViolation for disallowed overrides, got %v", err)
	}
}

func TestStoreRegistryResolve_AllowedAndDeniedOverrideKeys(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	mustUpsertRegistry(t, store, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("strict"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("strict"): {
				Slug: MustProfileSlug("strict"),
				Policy: PolicySpec{
					AllowOverrides:      true,
					AllowedOverrideKeys: []string{"system_prompt"},
					DeniedOverrideKeys:  []string{"tools"},
				},
			},
		},
	})

	registry := mustNewStoreRegistry(t, store)
	_, err := registry.ResolveEffectiveProfile(ctx, ResolveInput{
		RequestOverrides: map[string]any{"tools": []any{"search"}},
	})
	if !errors.Is(err, ErrPolicyViolation) {
		t.Fatalf("expected deny-list policy violation, got %v", err)
	}

	_, err = registry.ResolveEffectiveProfile(ctx, ResolveInput{
		RequestOverrides: map[string]any{"middlewares": []any{}},
	})
	if !errors.Is(err, ErrPolicyViolation) {
		t.Fatalf("expected allow-list policy violation, got %v", err)
	}

	_, err = registry.ResolveEffectiveProfile(ctx, ResolveInput{
		RequestOverrides: map[string]any{"system_prompt": "ok"},
	})
	if err != nil {
		t.Fatalf("expected allowed override to pass, got %v", err)
	}
}

func TestStoreRegistryResolve_RejectsDuplicateMiddlewareOverrideIDs(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	mustUpsertRegistry(t, store, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {
				Slug: MustProfileSlug("default"),
				Policy: PolicySpec{
					AllowOverrides:      true,
					AllowedOverrideKeys: []string{"middlewares"},
				},
			},
		},
	})
	registry := mustNewStoreRegistry(t, store)

	_, err := registry.ResolveEffectiveProfile(ctx, ResolveInput{
		RequestOverrides: map[string]any{
			"middlewares": []any{
				map[string]any{"name": "agentmode", "id": "agent"},
				map[string]any{"name": "sqlite", "id": "agent"},
			},
		},
	})
	if err == nil {
		t.Fatalf("expected duplicate middleware id validation error")
	}
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected ValidationError type, got %T", err)
	}
	if got, want := verr.Field, "request_overrides.middlewares[1].id"; got != want {
		t.Fatalf("validation field mismatch: got=%q want=%q", got, want)
	}
}

func TestStoreRegistryUpdateProfile_ReadOnlyPolicyViolation(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	mustUpsertRegistry(t, store, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("locked"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("locked"): {
				Slug: MustProfileSlug("locked"),
				Policy: PolicySpec{
					ReadOnly: true,
				},
			},
		},
	})

	registry := mustNewStoreRegistry(t, store)
	name := "updated"
	_, err := registry.UpdateProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("locked"), ProfilePatch{
		DisplayName: &name,
	}, WriteOptions{Actor: "test", Source: "test"})
	if !errors.Is(err, ErrPolicyViolation) {
		t.Fatalf("expected ErrPolicyViolation, got %v", err)
	}
}

func TestStoreRegistryDeleteProfile_ReadOnlyPolicyViolation(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	mustUpsertRegistry(t, store, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("locked"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("locked"): {
				Slug: MustProfileSlug("locked"),
				Policy: PolicySpec{
					ReadOnly: true,
				},
			},
		},
	})

	registry := mustNewStoreRegistry(t, store)
	err := registry.DeleteProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("locked"), WriteOptions{
		Actor:  "test",
		Source: "test",
	})
	if !errors.Is(err, ErrPolicyViolation) {
		t.Fatalf("expected ErrPolicyViolation, got %v", err)
	}
}

func TestStoreRegistryUpdateProfile_VersionConflict(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	mustUpsertRegistry(t, store, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("agent"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("agent"): {Slug: MustProfileSlug("agent")},
		},
	})

	registry := mustNewStoreRegistry(t, store)
	description := "new description"
	_, err := registry.UpdateProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("agent"), ProfilePatch{
		Description: &description,
	}, WriteOptions{
		ExpectedVersion: 999,
		Actor:           "test",
		Source:          "test",
	})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}
}

func TestStoreRegistryListRegistries_DeterministicOrdering(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	mustUpsertRegistry(t, store, &ProfileRegistry{
		Slug:               MustRegistrySlug("zeta"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
		},
	})
	mustUpsertRegistry(t, store, &ProfileRegistry{
		Slug:               MustRegistrySlug("alpha"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
		},
	})

	registry := mustNewStoreRegistry(t, store)
	summaries, err := registry.ListRegistries(ctx)
	if err != nil {
		t.Fatalf("ListRegistries returned error: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}
	if got, want := summaries[0].Slug, MustRegistrySlug("alpha"); got != want {
		t.Fatalf("summary ordering mismatch at index 0: got=%q want=%q", got, want)
	}
	if got, want := summaries[1].Slug, MustRegistrySlug("zeta"); got != want {
		t.Fatalf("summary ordering mismatch at index 1: got=%q want=%q", got, want)
	}
}

func TestStoreRegistryCreateProfile_NormalizesKnownExtensionsWithCodec(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	mustUpsertRegistry(t, store, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
		},
	})

	codecRegistry, err := NewInMemoryExtensionCodecRegistry(starterSuggestionsCodec{
		key: MustExtensionKey("webchat.starter_suggestions@v1"),
	})
	if err != nil {
		t.Fatalf("NewInMemoryExtensionCodecRegistry returned error: %v", err)
	}
	registry, err := NewStoreRegistry(store, MustRegistrySlug("default"), WithExtensionCodecRegistry(codecRegistry))
	if err != nil {
		t.Fatalf("NewStoreRegistry returned error: %v", err)
	}

	created, err := registry.CreateProfile(ctx, MustRegistrySlug("default"), &Profile{
		Slug: MustProfileSlug("agent"),
		Extensions: map[string]any{
			"WebChat.Starter_Suggestions@V1": map[string]any{
				"items": []any{"one", "two"},
			},
		},
	}, WriteOptions{Actor: "test", Source: "test"})
	if err != nil {
		t.Fatalf("CreateProfile returned error: %v", err)
	}

	value, ok := created.Extensions["webchat.starter_suggestions@v1"]
	if !ok {
		t.Fatalf("expected canonical extension key in created profile")
	}
	decoded, ok := value.(starterSuggestionsPayload)
	if !ok {
		t.Fatalf("expected decoded payload type, got %T", value)
	}
	if got, want := len(decoded.Items), 2; got != want {
		t.Fatalf("decoded payload items mismatch: got=%d want=%d", got, want)
	}
}

func TestStoreRegistryCreateProfile_ExtensionDecodeFailureReturnsValidation(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	mustUpsertRegistry(t, store, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
		},
	})

	codecRegistry, err := NewInMemoryExtensionCodecRegistry(starterSuggestionsCodec{
		key:      MustExtensionKey("webchat.starter_suggestions@v1"),
		forceErr: true,
	})
	if err != nil {
		t.Fatalf("NewInMemoryExtensionCodecRegistry returned error: %v", err)
	}
	registry, err := NewStoreRegistry(store, MustRegistrySlug("default"), WithExtensionCodecRegistry(codecRegistry))
	if err != nil {
		t.Fatalf("NewStoreRegistry returned error: %v", err)
	}

	_, err = registry.CreateProfile(ctx, MustRegistrySlug("default"), &Profile{
		Slug: MustProfileSlug("agent"),
		Extensions: map[string]any{
			"webchat.starter_suggestions@v1": map[string]any{"items": []any{"one"}},
		},
	}, WriteOptions{Actor: "test", Source: "test"})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	requireValidationField(t, err, "profile.extensions[webchat.starter_suggestions@v1]")
}

func TestStoreRegistryCreateProfile_UnknownExtensionPassThrough(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	mustUpsertRegistry(t, store, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
		},
	})

	registry := mustNewStoreRegistry(t, store)
	created, err := registry.CreateProfile(ctx, MustRegistrySlug("default"), &Profile{
		Slug: MustProfileSlug("agent"),
		Extensions: map[string]any{
			"App.Custom@V1": map[string]any{
				"flags": []any{map[string]any{"enabled": true}},
			},
		},
	}, WriteOptions{Actor: "test", Source: "test"})
	if err != nil {
		t.Fatalf("CreateProfile returned error: %v", err)
	}

	if _, ok := created.Extensions["App.Custom@V1"]; ok {
		t.Fatalf("expected canonicalized extension key")
	}
	raw, ok := created.Extensions["app.custom@v1"]
	if !ok {
		t.Fatalf("expected canonicalized unknown extension key")
	}
	if got := raw.(map[string]any)["flags"].([]any)[0].(map[string]any)["enabled"].(bool); !got {
		t.Fatalf("expected unknown extension payload pass-through")
	}
}

func TestStoreRegistryUpdateProfile_NormalizesExtensionPatch(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	mustUpsertRegistry(t, store, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("agent"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("agent"): {Slug: MustProfileSlug("agent")},
		},
	})

	codecRegistry, err := NewInMemoryExtensionCodecRegistry(starterSuggestionsCodec{
		key: MustExtensionKey("webchat.starter_suggestions@v1"),
	})
	if err != nil {
		t.Fatalf("NewInMemoryExtensionCodecRegistry returned error: %v", err)
	}
	registry, err := NewStoreRegistry(store, MustRegistrySlug("default"), WithExtensionCodecRegistry(codecRegistry))
	if err != nil {
		t.Fatalf("NewStoreRegistry returned error: %v", err)
	}

	extensionsPatch := map[string]any{
		"WebChat.Starter_Suggestions@V1": map[string]any{
			"items": []any{"a", "b"},
		},
	}
	updated, err := registry.UpdateProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("agent"), ProfilePatch{
		Extensions: &extensionsPatch,
	}, WriteOptions{Actor: "test", Source: "test"})
	if err != nil {
		t.Fatalf("UpdateProfile returned error: %v", err)
	}

	value, ok := updated.Extensions["webchat.starter_suggestions@v1"]
	if !ok {
		t.Fatalf("expected canonical extension key after update")
	}
	if _, ok := value.(starterSuggestionsPayload); !ok {
		t.Fatalf("expected decoded payload type, got %T", value)
	}
}

func TestStoreRegistryUpdateProfile_InvalidExtensionKeyFieldPath(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	mustUpsertRegistry(t, store, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("agent"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("agent"): {Slug: MustProfileSlug("agent")},
		},
	})

	registry := mustNewStoreRegistry(t, store)
	extensionsPatch := map[string]any{
		"bad key": map[string]any{"value": true},
	}
	_, err := registry.UpdateProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("agent"), ProfilePatch{
		Extensions: &extensionsPatch,
	}, WriteOptions{Actor: "test", Source: "test"})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	requireValidationField(t, err, "profile.extensions[bad key]")
}

func TestResolveEffectiveProfile_GoldenAgainstGatherFlagsFromProfiles(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "profiles.yaml")

	legacyYAML := `default:
  ai-chat:
    ai-engine: default-engine
agent:
  ai-chat:
    ai-engine: profile-engine
    ai-api-type: openai
  ai-client:
    timeout: 17
`
	if err := os.WriteFile(profilePath, []byte(legacyYAML), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	legacyStepSettings := runGatherFlagsStepSettings(t, profilePath, "agent")

	store := NewInMemoryProfileStore()
	mustUpsertRegistry(t, store, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("agent"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("agent"): {
				Slug: MustProfileSlug("agent"),
				Runtime: RuntimeSpec{StepSettingsPatch: map[string]any{
					"ai-chat": map[string]any{
						"ai-engine":   "profile-engine",
						"ai-api-type": "openai",
					},
					"ai-client": map[string]any{
						"timeout": 17,
					},
				}},
			},
		},
	})
	registry := mustNewStoreRegistry(t, store)

	resolved, err := registry.ResolveEffectiveProfile(ctx, ResolveInput{ProfileSlug: MustProfileSlug("agent")})
	if err != nil {
		t.Fatalf("ResolveEffectiveProfile returned error: %v", err)
	}
	if resolved.EffectiveStepSettings == nil {
		t.Fatalf("expected effective step settings")
	}

	if got, want := mustStringPtrValue(resolved.EffectiveStepSettings.Chat.Engine), mustStringPtrValue(legacyStepSettings.Chat.Engine); got != want {
		t.Fatalf("chat engine mismatch: got=%q want=%q", got, want)
	}
	if got, want := mustStringPtrValue((*string)(resolved.EffectiveStepSettings.Chat.ApiType)), mustStringPtrValue((*string)(legacyStepSettings.Chat.ApiType)); got != want {
		t.Fatalf("api type mismatch: got=%q want=%q", got, want)
	}
	if got, want := resolved.EffectiveStepSettings.Client.Timeout.String(), legacyStepSettings.Client.Timeout.String(); got != want {
		t.Fatalf("client timeout mismatch: got=%q want=%q", got, want)
	}
	if got, want := resolved.EffectiveStepSettings.GetMetadata()["ai-engine"], legacyStepSettings.GetMetadata()["ai-engine"]; got != want {
		t.Fatalf("metadata ai-engine mismatch: got=%v want=%v", got, want)
	}
}

func runGatherFlagsStepSettings(t *testing.T, profilePath, profileSlug string) *settings.StepSettings {
	t.Helper()

	schema_, err := newStepSettingsSchema()
	if err != nil {
		t.Fatalf("newStepSettingsSchema returned error: %v", err)
	}
	parsed := values.New()
	if err := sources.Execute(
		schema_,
		parsed,
		sources.GatherFlagsFromProfiles(profilePath, profilePath, profileSlug, "default"),
		sources.FromDefaults(),
	); err != nil {
		t.Fatalf("sources.Execute returned error: %v", err)
	}
	stepSettings, err := settings.NewStepSettingsFromParsedValues(parsed)
	if err != nil {
		t.Fatalf("NewStepSettingsFromParsedValues returned error: %v", err)
	}
	return stepSettings
}

func mustNewStoreRegistry(t *testing.T, store ProfileStore) *StoreRegistry {
	t.Helper()
	registry, err := NewStoreRegistry(store, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewStoreRegistry returned error: %v", err)
	}
	return registry
}

func mustUpsertRegistry(t *testing.T, store *InMemoryProfileStore, registry *ProfileRegistry) {
	t.Helper()
	if err := store.UpsertRegistry(context.Background(), registry, SaveOptions{Actor: "test", Source: "test"}); err != nil {
		t.Fatalf("UpsertRegistry returned error: %v", err)
	}
}

func mustUpsertProfile(t *testing.T, store *InMemoryProfileStore, registrySlug RegistrySlug, profile *Profile) {
	t.Helper()
	if err := store.UpsertProfile(context.Background(), registrySlug, profile, SaveOptions{Actor: "test", Source: "test"}); err != nil {
		t.Fatalf("UpsertProfile returned error: %v", err)
	}
}

func stringPtr(s string) *string { return &s }

func mustStringPtrValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
