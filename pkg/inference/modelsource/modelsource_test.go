package modelsource

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/engineprofiles"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/providers/pi"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeJSON(t *testing.T, dir, name string, v any) {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), b, 0o600))
}

// piDir has one provider with a literal key (two models) and one provider without a credential.
func piDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeJSON(t, dir, "models-store.json", map[string]any{
		"lunaroute": map[string]any{"models": []any{
			map[string]any{"id": "deepseek-4.1-flash", "name": "DeepSeek 4.1 Flash", "api": "openai-completions", "provider": "lunaroute",
				"baseUrl": "https://gw.example/v1", "reasoning": true, "input": []string{"text"}, "contextWindow": 131072, "maxTokens": 32768},
			map[string]any{"id": "glm-5.3-vision", "api": "openai-completions", "provider": "lunaroute",
				"baseUrl": "https://gw.example/v1", "input": []string{"text", "image"}, "contextWindow": 65536, "maxTokens": 8192},
		}},
		"nokey": map[string]any{"models": []any{
			map[string]any{"id": "m", "api": "openai-completions", "provider": "nokey", "baseUrl": "https://nokey.example/v1"},
		}},
	})
	writeJSON(t, dir, "models.json", map[string]any{"providers": map[string]any{
		"lunaroute": map[string]any{"apiKey": "literal-test-key"},
	}})
	return dir
}

const profilesYAML = `slug: team
profiles:
  claude:
    slug: claude
    display_name: Claude (team key)
    inference_settings:
      chat:
        api_type: claude
        engine: claude-sonnet-5-5
      api:
        api_keys:
          claude-api-key: test-key
      model_info:
        context_window: 200000
        max_output_tokens: 64000
        input: [text, image]
        cost: {input: 3, output: 15}
  capped:
    slug: capped
    inference_settings:
      chat:
        api_type: claude
        engine: claude-haiku-5-5
      api:
        api_keys:
          claude-api-key: test-key
      inference:
        max_response_tokens: 8192
  base:
    slug: base
    inference_settings:
      chat:
        temperature: 0.2
`

func profilesFile(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "profiles.yaml")
	require.NoError(t, os.WriteFile(p, []byte(profilesYAML), 0o600))
	return p
}

func TestPiModelsListOnlyCredentialed(t *testing.T) {
	a, err := pi.LoadAgentDir(piDir(t))
	require.NoError(t, err)
	ms, err := FromPi(a).Models(context.Background())
	require.NoError(t, err)
	refs := []string{}
	for _, m := range ms {
		refs = append(refs, m.Ref)
	}
	assert.ElementsMatch(t, []string{"pi:lunaroute/deepseek-4.1-flash", "pi:lunaroute/glm-5.3-vision"}, refs, "nokey has no credential")
	for _, m := range ms {
		if m.Ref == "pi:lunaroute/glm-5.3-vision" {
			assert.True(t, m.Images)
			assert.Equal(t, 65536, m.ContextWindow)
		}
		if m.Ref == "pi:lunaroute/deepseek-4.1-flash" {
			assert.Equal(t, "DeepSeek 4.1 Flash", m.Label)
			assert.True(t, m.Reasoning)
		}
	}
}

func TestPiResolveWithAndWithoutPrefix(t *testing.T) {
	a, err := pi.LoadAgentDir(piDir(t))
	require.NoError(t, err)
	src := FromPi(a)
	for _, ref := range []string{"pi:lunaroute/deepseek-4.1-flash", "lunaroute/deepseek-4.1-flash"} {
		r, err := src.Resolve(context.Background(), ref, Options{MaxTokens: 1000})
		require.NoError(t, err, ref)
		assert.Equal(t, "pi:lunaroute/deepseek-4.1-flash", r.Model.Ref)
		e, err := r.Engine()
		require.NoError(t, err)
		assert.NotNil(t, e)
		require.NotNil(t, r.Settings.Inference)
		require.NotNil(t, r.Settings.Inference.MaxResponseTokens)
		assert.Equal(t, 1000, *r.Settings.Inference.MaxResponseTokens)
	}
	_, err = src.Resolve(context.Background(), "pi:lunaroute/nope", Options{})
	assert.True(t, errors.Is(err, ErrUnknownModel), "%v", err)
	_, err = src.Resolve(context.Background(), "profile:claude", Options{})
	assert.True(t, errors.Is(err, ErrUnknownModel), "another source's prefix is unknown here: %v", err)
}

func TestProfilesListAndResolve(t *testing.T) {
	set, notes, err := Load(context.Background(), LoadOptions{SkipPi: true, ProfileSources: []string{profilesFile(t)}})
	require.NoError(t, err)
	assert.Empty(t, notes)
	t.Cleanup(func() { require.NoError(t, set.Close()) })
	ms, err := set.Models(context.Background())
	require.NoError(t, err)
	require.Len(t, ms, 2, "the base profile names no engine and is not listed")
	var m Model
	for _, x := range ms {
		if x.Ref == "profile:claude" {
			m = x
		}
	}
	assert.Equal(t, "profile:claude", m.Ref)
	assert.Equal(t, "Claude (team key)", m.Label)
	assert.Equal(t, "claude", m.API)
	assert.Equal(t, 200000, m.ContextWindow)
	assert.True(t, m.Images)
	require.NotNil(t, m.Cost)
	assert.Equal(t, 15.0, m.Cost.Output)

	for _, ref := range []string{"profile:claude", "profile:team/claude", "claude"} {
		r, err := set.Resolve(context.Background(), ref, Options{MaxTokens: 512})
		require.NoError(t, err, ref)
		e, err := r.Engine()
		require.NoError(t, err, ref)
		assert.NotNil(t, e)
		assert.Equal(t, 512, *r.Settings.Chat.MaxResponseTokens)
		assert.Empty(t, r.Settings.API.APIKeys, "resolved settings carry no credentials")
	}
	capped, err := set.Resolve(context.Background(), "profile:capped", Options{MaxTokens: 512})
	require.NoError(t, err)
	require.NotNil(t, capped.Settings.Inference)
	assert.Equal(t, 512, *capped.Settings.Inference.MaxResponseTokens, "the option overrides the profile's inference-level limit")
	_, err = set.Resolve(context.Background(), "profile:missing", Options{})
	assert.True(t, errors.Is(err, ErrUnknownModel), "%v", err)
	_, err = set.Resolve(context.Background(), "profile:base", Options{})
	assert.ErrorContains(t, err, "names no engine")
}

func TestSetRoutesByPrefixAndFallsThrough(t *testing.T) {
	ctx := context.Background()
	set, notes, err := Load(ctx, LoadOptions{PiDir: piDir(t), ProfileSources: []string{profilesFile(t)}})
	require.NoError(t, err)
	assert.Empty(t, notes)
	t.Cleanup(func() { require.NoError(t, set.Close()) })
	assert.Equal(t, []string{"profile", "pi"}, set.Sources())
	ms, err := set.Models(ctx)
	require.NoError(t, err)
	assert.Len(t, ms, 4)

	r, err := set.Resolve(ctx, "lunaroute/glm-5.3-vision", Options{})
	require.NoError(t, err, "an unprefixed Pi ref falls through the profile source")
	assert.Equal(t, "pi:lunaroute/glm-5.3-vision", r.Model.Ref)
	_, err = set.Resolve(ctx, "nosuch:thing", Options{})
	assert.True(t, errors.Is(err, ErrUnknownModel))
	_, err = set.Resolve(ctx, "nowhere/model", Options{})
	assert.True(t, errors.Is(err, ErrUnknownModel))
}

func TestLoadSkipsMissingSources(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(pi.EnvAgentDir, "")
	set, notes, err := Load(context.Background(), LoadOptions{PiDir: filepath.Join(t.TempDir(), "absent"), AppName: "nothing-here"})
	require.NoError(t, err)
	assert.Empty(t, set.Sources())
	assert.Len(t, notes, 2)
}

func TestStoreRegistryRefsStayResolvable(t *testing.T) {
	ctx := context.Background()
	store := engineprofiles.NewInMemoryEngineProfileStore()
	profile := func(slug, model string) *engineprofiles.EngineProfile {
		api := types.ApiTypeClaude
		return &engineprofiles.EngineProfile{Slug: engineprofiles.MustEngineProfileSlug(slug), InferenceSettings: &settings.InferenceSettings{
			Chat: &settings.ChatSettings{ApiType: &api, Engine: &model},
			API:  &settings.APISettings{APIKeys: map[string]string{"claude-api-key": "k"}},
		}}
	}
	for _, r := range []struct{ slug, prof string }{{"main", "fast"}, {"team", "claude"}} {
		require.NoError(t, store.UpsertRegistry(ctx, &engineprofiles.EngineProfileRegistry{
			Slug:                     engineprofiles.MustRegistrySlug(r.slug),
			DefaultEngineProfileSlug: engineprofiles.MustEngineProfileSlug(r.prof),
			Profiles:                 map[engineprofiles.EngineProfileSlug]*engineprofiles.EngineProfile{engineprofiles.MustEngineProfileSlug(r.prof): profile(r.prof, "m-"+r.prof)},
		}, engineprofiles.SaveOptions{}))
	}
	reg, err := engineprofiles.NewStoreRegistry(store, engineprofiles.MustRegistrySlug("main"))
	require.NoError(t, err)
	src := FromProfiles(reg)
	ms, err := src.Models(ctx)
	require.NoError(t, err)
	refs := []string{}
	for _, m := range ms {
		refs = append(refs, m.Ref)
		_, err := src.Resolve(ctx, m.Ref, Options{})
		require.NoError(t, err, "every listed ref resolves: %s", m.Ref)
	}
	assert.ElementsMatch(t, []string{"profile:fast", "profile:team/claude"}, refs, "a profile outside the default registry keeps its registry")
}
