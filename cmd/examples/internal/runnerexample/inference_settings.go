package runnerexample

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	profiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	"github.com/go-go-golems/geppetto/pkg/inference/runner"
	geppettosections "github.com/go-go-golems/geppetto/pkg/sections"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

// OpenAIInferenceSettingsFromEnv returns basic OpenAI-backed inference settings suitable
// for example programs.
func OpenAIInferenceSettingsFromEnv(model string, stream bool) (*settings.InferenceSettings, error) {
	ss, err := settings.NewInferenceSettings()
	if err != nil {
		return nil, err
	}

	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is not set")
	}

	if strings.TrimSpace(model) == "" {
		model = "gpt-4o-mini"
	}

	apiType := types.ApiTypeOpenAI
	ss.Chat.ApiType = &apiType
	ss.Chat.Engine = &model
	ss.Chat.Stream = stream
	ss.API.APIKeys["openai-api-key"] = apiKey

	return ss, nil
}

// BaseInferenceSettingsFromDefaults mirrors the Pinocchio pattern of resolving hidden
// base settings from Geppetto sections, but only from section defaults.
//
// Small apps can replace this with their own hidden bootstrap from config files,
// secrets, or deployment defaults without exposing the full Geppetto flag surface.
func BaseInferenceSettingsFromDefaults() (*settings.InferenceSettings, error) {
	sections_, err := geppettosections.CreateGeppettoSections()
	if err != nil {
		return nil, err
	}
	schema_ := schema.NewSchema(schema.WithSections(sections_...))
	parsedValues := values.New()
	if err := sources.Execute(
		schema_,
		parsedValues,
		sources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
	); err != nil {
		return nil, err
	}
	return settings.NewInferenceSettingsFromParsedValues(parsedValues)
}

// ExampleEngineProfileRegistryPath returns the bundled sample profile registry path.
func ExampleEngineProfileRegistryPath() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "cmd/examples/internal/runnerexample/profiles/basic.yaml"
	}
	return filepath.Join(filepath.Dir(file), "profiles", "basic.yaml")
}

// ResolveRuntimeFromRegistry loads a chained registry, resolves one profile, and
// returns runner.Runtime plus the registry closer.
func ResolveRuntimeFromRegistry(
	ctx context.Context,
	stepSettings *settings.InferenceSettings,
	rawSources string,
	profileSlug string,
) (runner.Runtime, func() error, error) {
	if stepSettings == nil {
		return runner.Runtime{}, nil, fmt.Errorf("inference settings are required")
	}
	entries, err := profiles.ParseEngineProfileRegistrySourceEntries(rawSources)
	if err != nil {
		return runner.Runtime{}, nil, err
	}
	specs, err := profiles.ParseRegistrySourceSpecs(entries)
	if err != nil {
		return runner.Runtime{}, nil, err
	}
	chain, err := profiles.NewChainedRegistryFromSourceSpecs(ctx, specs)
	if err != nil {
		return runner.Runtime{}, nil, err
	}

	in := profiles.ResolveInput{}
	if strings.TrimSpace(profileSlug) != "" {
		parsed, err := profiles.ParseEngineProfileSlug(profileSlug)
		if err != nil {
			_ = chain.Close()
			return runner.Runtime{}, nil, err
		}
		in.EngineProfileSlug = parsed
	}

	resolved, err := chain.ResolveEngineProfile(ctx, in)
	if err != nil {
		_ = chain.Close()
		return runner.Runtime{}, nil, err
	}

	profileVersion := uint64(0)
	if resolved.Metadata != nil {
		if v, ok := resolved.Metadata["profile.version"].(uint64); ok {
			profileVersion = v
		}
	}

	return runner.Runtime{
		InferenceSettings:  stepSettings,
		SystemPrompt:       resolved.EffectiveRuntime.SystemPrompt,
		MiddlewareUses:     resolved.EffectiveRuntime.Middlewares,
		ToolNames:          append([]string(nil), resolved.EffectiveRuntime.Tools...),
		RuntimeKey:         resolved.RuntimeKey.String(),
		RuntimeFingerprint: resolved.RuntimeFingerprint,
		ProfileVersion:     profileVersion,
	}, chain.Close, nil
}
