package geppetto

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/dop251/goja"
	enginefactory "github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
	openaiengine "github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
	aistepssettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	aitypes "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

func TestEnsureInferenceSettingsProviderDefaults(t *testing.T) {
	tests := []struct {
		name          string
		apiType       aitypes.ApiType
		wantOpenAI    bool
		wantClaude    bool
		wantGemini    bool
		wantClaudeURL bool
	}{
		{name: "openai", apiType: aitypes.ApiTypeOpenAI, wantOpenAI: true},
		{name: "anyscale", apiType: aitypes.ApiTypeAnyScale, wantOpenAI: true},
		{name: "fireworks", apiType: aitypes.ApiTypeFireworks, wantOpenAI: true},
		{name: "responses", apiType: aitypes.ApiTypeOpenAIResponses},
		{name: "claude", apiType: aitypes.ApiTypeClaude, wantClaude: true, wantClaudeURL: true},
		{name: "anthropic-alias", apiType: aitypes.ApiType("anthropic"), wantClaude: true, wantClaudeURL: true},
		{name: "gemini", apiType: aitypes.ApiTypeGemini, wantGemini: true},
		{name: "unsupported-ollama", apiType: aitypes.ApiTypeOllama},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := &aistepssettings.InferenceSettings{Chat: &aistepssettings.ChatSettings{ApiType: &tt.apiType}}
			if err := ensureInferenceSettingsProviderDefaults(settings); err != nil {
				t.Fatalf("ensure defaults: %v", err)
			}
			if settings.API == nil || settings.API.APIKeys == nil || settings.API.BaseUrls == nil || settings.API.AllowHTTP == nil || settings.API.AllowLocalNetworks == nil {
				t.Fatalf("API defaults were not fully initialized: %#v", settings.API)
			}
			if settings.Client == nil {
				t.Fatal("client defaults were not initialized")
			}
			if (settings.OpenAI != nil) != tt.wantOpenAI {
				t.Fatalf("OpenAI defaults present=%v, want %v", settings.OpenAI != nil, tt.wantOpenAI)
			}
			if (settings.Claude != nil) != tt.wantClaude {
				t.Fatalf("Claude defaults present=%v, want %v", settings.Claude != nil, tt.wantClaude)
			}
			if (settings.Gemini != nil) != tt.wantGemini {
				t.Fatalf("Gemini defaults present=%v, want %v", settings.Gemini != nil, tt.wantGemini)
			}
			_, hasClaudeURL := settings.API.BaseUrls["claude-base-url"]
			if hasClaudeURL != tt.wantClaudeURL {
				t.Fatalf("Claude default URL present=%v, want %v", hasClaudeURL, tt.wantClaudeURL)
			}
		})
	}
}

func TestEnsureInferenceSettingsProviderDefaultsPreservesExplicitValues(t *testing.T) {
	apiType := aitypes.ApiTypeClaude
	settings, err := aistepssettings.NewInferenceSettings()
	if err != nil {
		t.Fatalf("new inference settings: %v", err)
	}
	settings.Chat.ApiType = &apiType
	settings.API.BaseUrls["claude-base-url"] = "https://claude.example.test"
	explicitClient := settings.Client
	explicitClaude := settings.Claude
	if err := ensureInferenceSettingsProviderDefaults(settings); err != nil {
		t.Fatalf("ensure defaults: %v", err)
	}
	if settings.Client != explicitClient {
		t.Fatal("explicit client settings were replaced")
	}
	if settings.Claude != explicitClaude {
		t.Fatal("explicit Claude settings were replaced")
	}
	if got := settings.API.BaseUrls["claude-base-url"]; got != "https://claude.example.test" {
		t.Fatalf("explicit Claude base URL = %q", got)
	}
}

func TestEnsureInferenceSettingsProviderDefaultsPopulatesAnthropicAliasKeys(t *testing.T) {
	apiType := aitypes.ApiType("anthropic")
	settings := &aistepssettings.InferenceSettings{
		Chat: &aistepssettings.ChatSettings{ApiType: &apiType},
		API: &aistepssettings.APISettings{
			APIKeys:            map[string]string{"claude-api-key": "canonical-key"},
			BaseUrls:           map[string]string{"claude-base-url": "https://claude.example.test"},
			AllowHTTP:          map[string]bool{"claude": true},
			AllowLocalNetworks: map[string]bool{"claude-allow-local-networks": true},
		},
	}
	if err := ensureInferenceSettingsProviderDefaults(settings); err != nil {
		t.Fatalf("ensure defaults: %v", err)
	}

	if got := settings.API.APIKeys["anthropic-api-key"]; got != "canonical-key" {
		t.Fatalf("anthropic API key = %q, want canonical-key", got)
	}
	if got := settings.API.BaseUrls["anthropic-base-url"]; got != "https://claude.example.test" {
		t.Fatalf("anthropic base URL = %q, want canonical Claude URL", got)
	}
	if !settings.API.AllowHTTP["anthropic"] {
		t.Fatal("anthropic allow_http was not populated from claude")
	}
	if !settings.API.AllowLocalNetworks["anthropic-allow-local-networks"] {
		t.Fatal("anthropic allow_local_networks was not populated from claude")
	}

	settings.API.APIKeys["anthropic-api-key"] = "explicit-alias-key"
	if err := ensureInferenceSettingsProviderDefaults(settings); err != nil {
		t.Fatalf("ensure defaults with explicit alias: %v", err)
	}
	if got := settings.API.APIKeys["anthropic-api-key"]; got != "explicit-alias-key" {
		t.Fatalf("explicit anthropic API key = %q, want explicit-alias-key", got)
	}
}

func TestNormalizedSparseSettingsReachProviderFactories(t *testing.T) {
	tests := []struct {
		name    string
		apiType aitypes.ApiType
		apiKey  string
	}{
		{name: "openai", apiType: aitypes.ApiTypeOpenAI, apiKey: "openai-api-key"},
		{name: "responses", apiType: aitypes.ApiTypeOpenAIResponses, apiKey: "openai-api-key"},
		{name: "claude", apiType: aitypes.ApiTypeClaude, apiKey: "claude-api-key"},
		{name: "gemini", apiType: aitypes.ApiTypeGemini, apiKey: "gemini-api-key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := &aistepssettings.InferenceSettings{
				Chat: &aistepssettings.ChatSettings{ApiType: &tt.apiType, Engine: stringPtr("test-model")},
				API:  &aistepssettings.APISettings{APIKeys: map[string]string{tt.apiKey: "test-key"}},
			}
			if err := ensureInferenceSettingsProviderDefaults(settings); err != nil {
				t.Fatalf("ensure defaults: %v", err)
			}
			if _, err := enginefactory.NewEngineFromSettings(settings); err != nil {
				t.Fatalf("new engine from normalized sparse settings: %v", err)
			}
		})
	}

	ollamaType := aitypes.ApiTypeOllama
	unsupported := &aistepssettings.InferenceSettings{
		Chat: &aistepssettings.ChatSettings{ApiType: &ollamaType, Engine: stringPtr("llama3")},
		API:  &aistepssettings.APISettings{APIKeys: map[string]string{"ollama-api-key": "test-key"}},
	}
	if err := ensureInferenceSettingsProviderDefaults(unsupported); err != nil {
		t.Fatalf("ensure unsupported defaults: %v", err)
	}
	if _, err := enginefactory.NewEngineFromSettings(unsupported); err == nil {
		t.Fatal("unsupported ollama chat provider unexpectedly constructed an engine")
	}
}

func TestNormalizedSparseOpenAISettingsBuildRequestWithoutNetwork(t *testing.T) {
	apiType := aitypes.ApiTypeOpenAI
	settings := &aistepssettings.InferenceSettings{
		Chat: &aistepssettings.ChatSettings{ApiType: &apiType, Engine: stringPtr("gpt-4o-mini")},
		API:  &aistepssettings.APISettings{APIKeys: map[string]string{"openai-api-key": "test-key"}},
	}
	if err := ensureInferenceSettingsProviderDefaults(settings); err != nil {
		t.Fatalf("ensure defaults: %v", err)
	}
	eng, err := enginefactory.NewEngineFromSettings(settings)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	openAI, ok := eng.(*openaiengine.OpenAIEngine)
	if !ok {
		t.Fatalf("engine type = %T, want *openai.OpenAIEngine", eng)
	}
	request, err := openAI.MakeCompletionRequestFromTurn(&turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("construct a request without sending it"),
	}})
	if err != nil {
		t.Fatalf("build OpenAI request: %v", err)
	}
	if request.Model != "gpt-4o-mini" {
		t.Fatalf("request model = %q, want gpt-4o-mini", request.Model)
	}
}

func stringPtr(value string) *string { return &value }

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
			globalThis.profileAgent = agent;
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
	model, err := rt.runtimeOwner.Call(context.Background(), "test.profileAgentRequestConstruction", func(_ context.Context, vm *goja.Runtime) (any, error) {
		agentObject := vm.Get("profileAgent").ToObject(vm)
		ref, ok := agentObject.Get(hiddenRefKey).Export().(*agentRef)
		if !ok || ref == nil || ref.base == nil || ref.base.Engine == nil {
			return nil, fmt.Errorf("profile agent ref is unavailable: %#v", ref)
		}
		openAI, ok := ref.base.Engine.(*openaiengine.OpenAIEngine)
		if !ok {
			return nil, fmt.Errorf("profile agent engine type = %T, want *openai.OpenAIEngine", ref.base.Engine)
		}
		request, requestErr := openAI.MakeCompletionRequestFromTurn(&turns.Turn{Blocks: []turns.Block{
			turns.NewUserTextBlock("construct a request without sending it"),
		}})
		if requestErr != nil {
			return nil, requestErr
		}
		return request.Model, nil
	})
	if err != nil {
		t.Fatalf("profile request construction failed: %v", err)
	}
	if model.(string) != "gpt-5-mini" {
		t.Fatalf("profile request model = %q, want gpt-5-mini", model.(string))
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
