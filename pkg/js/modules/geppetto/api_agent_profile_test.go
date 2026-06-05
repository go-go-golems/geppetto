package geppetto

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dop251/goja"
	aistepssettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	aitypes "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
)

func TestEnsureInferenceSettingsProviderDefaultsHandlesMissingAPISettings(t *testing.T) {
	apiType := aitypes.ApiTypeOpenAI
	settings := &aistepssettings.InferenceSettings{Chat: &aistepssettings.ChatSettings{ApiType: &apiType}}
	ensureInferenceSettingsProviderDefaults(settings)
	if settings.API == nil {
		t.Fatalf("API settings were not initialized")
	}
	if settings.API.BaseUrls == nil {
		t.Fatalf("API base URLs were not initialized")
	}
}

func TestAgentSessionBuildFromProfileWithAPISettings(t *testing.T) {
	profilePath := filepath.Join(t.TempDir(), "profiles.yaml")
	if err := os.WriteFile(profilePath, []byte(`slug: workspace
profiles:
  default:
    slug: default
    display_name: Workspace Default
    inference_settings:
      api:
        api_keys:
          openai-api-key: test-key
      chat:
        api_type: openai
        engine: gpt-4o-mini
  assistant:
    slug: assistant
    display_name: Assistant
    stack:
      - profile_slug: default
    inference_settings:
      chat:
        engine: gpt-5-mini
`), 0o644); err != nil {
		t.Fatalf("write profile fixture: %v", err)
	}

	rt := newJSRuntime(t, Options{})
	_, err := rt.runtimeOwner.Call(context.Background(), "test.profileAgentSessionBuild", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if err := vm.Set("profilePath", profilePath); err != nil {
			return nil, err
		}
		_, runErr := vm.RunString(`
			const gp = require("geppetto");
			const settings = gp.inferenceProfiles.load(globalThis.profilePath).resolve("assistant");
			const snapshot = settings.toJSON();
			const agent = gp.agent()
				.name("missing-api-settings-smoke")
				.inference(settings)
				.build();
			const session = agent.session().id("missing-api-settings-session").build();
			globalThis.profileSmoke = {
				profile: snapshot.provenance && snapshot.provenance.profileSlug || "",
				registry: snapshot.provenance && snapshot.provenance.registrySlug || "",
				model: snapshot.chat && snapshot.chat.engine || "",
				apiType: snapshot.chat && snapshot.chat.api_type || "",
				session: session.id(),
				hasSessionNext: typeof session.next === "function",
			};
		`)
		return nil, runErr
	})
	if err != nil {
		t.Fatalf("profile agent/session build failed: %v", err)
	}
	got, err := rt.runtimeOwner.Call(context.Background(), "test.readProfileAgentSessionBuild", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunString(`JSON.stringify(globalThis.profileSmoke)`)
	})
	if err != nil {
		t.Fatalf("read profile smoke failed: %v", err)
	}
	want := `{"profile":"assistant","registry":"workspace","model":"gpt-5-mini","apiType":"openai","session":"missing-api-settings-session","hasSessionNext":true}`
	if got.(goja.Value).String() != want {
		t.Fatalf("profile smoke = %s, want %s", got.(goja.Value).String(), want)
	}
}
