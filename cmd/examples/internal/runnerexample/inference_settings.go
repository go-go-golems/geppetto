package runnerexample

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	profiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/rs/zerolog/log"
)

// PinocchioProfileRegistryPath returns the user's Pinocchio profile registry path.
func PinocchioProfileRegistryPath() string {
	return filepath.Join(homeDir(), ".config", "pinocchio", "profiles.yaml")
}

// OpenAIInferenceSettingsFromProfiles resolves OpenAI-backed inference settings from
// Geppetto/Pinocchio profiles. Credentials must come from the profile stack; examples
// should not read provider API keys directly from process environment variables.
func OpenAIInferenceSettingsFromProfiles(ctx context.Context, registryEntries string, profileSlug string, stream bool) (*settings.InferenceSettings, func() error, error) {
	if strings.TrimSpace(registryEntries) == "" {
		registryEntries = PinocchioProfileRegistryPath()
	}
	if strings.TrimSpace(profileSlug) == "" {
		profileSlug = "gpt-5-nano-low"
	}

	ss, closeFn, err := ResolveInferenceSettingsFromRegistry(ctx, splitRegistryEntries(registryEntries), profileSlug)
	if err != nil {
		return nil, nil, err
	}
	if ss != nil && ss.Chat != nil {
		ss.Chat.Stream = stream
	}
	return ss, closeFn, nil
}

// ExampleEngineProfileRegistryPath returns the bundled sample profile registry path.
func ExampleEngineProfileRegistryPath() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "cmd/examples/internal/runnerexample/profiles/basic.yaml"
	}
	return filepath.Join(filepath.Dir(file), "profiles", "basic.yaml")
}

func homeDir() string {
	if home := strings.TrimSpace(SystemHomeDir()); home != "" {
		return home
	}
	return "."
}

// SystemHomeDir exists so tests can replace home-dir lookup without changing the
// public example helper API.
var SystemHomeDir = func() string {
	home, _ := os.UserHomeDir()
	return home
}

func splitRegistryEntries(raw string) []string {
	parts := strings.Split(raw, ",")
	entries := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			entries = append(entries, part)
		}
	}
	return entries
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
