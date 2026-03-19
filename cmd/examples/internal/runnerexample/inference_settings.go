package runnerexample

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	profiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/rs/zerolog/log"
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

// ExampleEngineProfileRegistryPath returns the bundled sample profile registry path.
func ExampleEngineProfileRegistryPath() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "cmd/examples/internal/runnerexample/profiles/basic.yaml"
	}
	return filepath.Join(filepath.Dir(file), "profiles", "basic.yaml")
}

// ResolveInferenceSettingsFromRegistry loads a chained registry, resolves one engine
// profile, and returns the final merged inference settings plus the registry closer.
func ResolveInferenceSettingsFromRegistry(
	ctx context.Context,
	entries []string,
	profileSlug string,
) (*settings.InferenceSettings, func() error, error) {
	specs, err := profiles.ParseRegistrySourceSpecs(entries)
	if err != nil {
		return nil, nil, err
	}
	log.Debug().Msg("parsing profile specs")
	chain, err := profiles.NewChainedRegistryFromSourceSpecs(ctx, specs)
	if err != nil {
		return nil, nil, err
	}

	in := profiles.ResolveInput{}
	if strings.TrimSpace(profileSlug) != "" {
		parsed, err := profiles.ParseEngineProfileSlug(profileSlug)
		if err != nil {
			_ = chain.Close()
			return nil, nil, err
		}
		in.EngineProfileSlug = parsed
	}

	resolved, err := chain.ResolveEngineProfile(ctx, in)
	if err != nil {
		_ = chain.Close()
		return nil, nil, err
	}

	return resolved.InferenceSettings, chain.Close, nil
}
