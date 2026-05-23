package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-go-golems/geppetto/cmd/examples/internal/examplecmd"
	"github.com/go-go-golems/geppetto/pkg/embeddings"
	"github.com/go-go-golems/geppetto/pkg/embeddings/config"
	profiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	geppettosections "github.com/go-go-golems/geppetto/pkg/sections"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type embeddingCommand struct {
	*cmds.CommandDescription
}

var _ cmds.GlazeCommand = (*embeddingCommand)(nil)

type embeddingSettings struct {
	BaseProfile         string `glazed:"base-profile"`
	EmbeddingType       string `glazed:"embeddings-type"`
	EmbeddingEngine     string `glazed:"embeddings-engine"`
	EmbeddingDimensions int    `glazed:"embeddings-dimensions"`
	Text                string `glazed:"text"`
	Preview             int    `glazed:"preview"`
	TimeoutSeconds      int    `glazed:"timeout-seconds"`
}

func newEmbeddingCommand() (*embeddingCommand, error) {
	profileSettingsSection, err := geppettosections.NewProfileSettingsSection(
		geppettosections.WithProfileRegistriesDefault(defaultPinocchioProfilesPath()),
	)
	if err != nil {
		return nil, err
	}

	description := cmds.NewCommandDescription(
		"run",
		cmds.WithShort("Generate one embedding from a profile-backed embedding configuration"),
		cmds.WithLong(`Generate one embedding using profile registry settings.

If --profile is set, the selected profile must already contain embeddings settings.
If --profile is empty, the command resolves --base-profile and overlays the
embedding flags onto that base profile.

Use Glazed output flags for machine-readable output, for example:
  embedding-profile-smoke run --output json --text "hello"`),
		cmds.WithFlags(
			fields.New("base-profile", fields.TypeString,
				fields.WithDefault("openai-responses-base"),
				fields.WithHelp("Base profile to stack when --profile is empty"),
			),
			fields.New("embeddings-type", fields.TypeString,
				fields.WithDefault("openai"),
				fields.WithHelp("Embedding provider type: openai or ollama"),
			),
			fields.New("embeddings-engine", fields.TypeString,
				fields.WithDefault("text-embedding-3-small"),
				fields.WithHelp("Embedding model/engine"),
			),
			fields.New("embeddings-dimensions", fields.TypeInteger,
				fields.WithDefault(1536),
				fields.WithHelp("Expected embedding vector dimensions"),
			),
			fields.New("text", fields.TypeString,
				fields.WithDefault("hello profile-backed embeddings"),
				fields.WithHelp("Text to embed"),
			),
			fields.New("preview", fields.TypeInteger,
				fields.WithDefault(5),
				fields.WithHelp("Number of vector dimensions to include in preview"),
			),
			fields.New("timeout-seconds", fields.TypeInteger,
				fields.WithDefault(30),
				fields.WithHelp("Embedding request timeout in seconds"),
			),
		),
		cmds.WithSections(profileSettingsSection),
	)

	return &embeddingCommand{CommandDescription: description}, nil
}

func (c *embeddingCommand) RunIntoGlazeProcessor(ctx context.Context, parsedValues *values.Values, gp middlewares.Processor) error {
	s := &embeddingSettings{}
	if err := parsedValues.DecodeSectionInto(values.DefaultSlug, s); err != nil {
		return errors.Wrap(err, "decode embedding settings")
	}
	profileSettings := &geppettosections.ProfileSettings{}
	if err := parsedValues.DecodeSectionInto(geppettosections.ProfileSettingsSectionSlug, profileSettings); err != nil {
		return errors.Wrap(err, "decode profile settings")
	}

	timeout := time.Duration(s.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resolved, closeFn, effectiveProfile, err := resolveSettings(ctx, profileSettings.ProfileRegistries, profileSettings.Profile, s.BaseProfile, s.EmbeddingType, s.EmbeddingEngine, s.EmbeddingDimensions)
	if err != nil {
		return err
	}
	if closeFn != nil {
		defer func() { _ = closeFn() }()
	}

	if err := embeddings.ValidateInferenceSettingsForEmbeddings(resolved); err != nil {
		return err
	}

	provider, err := embeddings.NewSettingsFactoryFromInferenceSettings(resolved).NewProvider()
	if err != nil {
		return err
	}

	vector, err := provider.GenerateEmbedding(ctx, s.Text)
	if err != nil {
		return err
	}

	model := provider.GetModel()
	if len(vector) != model.Dimensions {
		return fmt.Errorf("dimension mismatch: configured=%d actual=%d", model.Dimensions, len(vector))
	}

	row := types.NewRow(
		types.MRP("profile", effectiveProfile),
		types.MRP("profile_registries", strings.Join(profileSettings.ProfileRegistries, ",")),
		types.MRP("provider_type", resolved.Embeddings.Type),
		types.MRP("model", model.Name),
		types.MRP("configured_dimensions", model.Dimensions),
		types.MRP("actual_dimensions", len(vector)),
		types.MRP("key_configured", openAIKeyConfigured(resolved)),
		types.MRP("base_url_configured", baseURLConfigured(resolved)),
		types.MRP("preview", vectorPreview(vector, s.Preview)),
	)
	return gp.AddRow(ctx, row)
}

func resolveSettings(ctx context.Context, registryEntries []string, profile string, baseProfile string, embeddingType string, embeddingEngine string, dimensions int) (*settings.InferenceSettings, func() error, string, error) {
	specs, err := profiles.ParseRegistrySourceSpecs(registryEntries)
	if err != nil {
		return nil, nil, "", err
	}
	chain, err := profiles.NewChainedRegistryFromSourceSpecs(ctx, specs)
	if err != nil {
		return nil, nil, "", err
	}

	if strings.TrimSpace(profile) != "" {
		profileSlug, err := profiles.ParseEngineProfileSlug(profile)
		if err != nil {
			_ = chain.Close()
			return nil, nil, "", err
		}
		resolved, err := chain.ResolveEngineProfile(ctx, profiles.ResolveInput{EngineProfileSlug: profileSlug})
		if err != nil {
			_ = chain.Close()
			return nil, nil, "", err
		}
		return resolved.InferenceSettings, chain.Close, profile, nil
	}

	baseSlug, err := profiles.ParseEngineProfileSlug(baseProfile)
	if err != nil {
		_ = chain.Close()
		return nil, nil, "", err
	}
	baseResolved, err := chain.ResolveEngineProfile(ctx, profiles.ResolveInput{EngineProfileSlug: baseSlug})
	if err != nil {
		_ = chain.Close()
		return nil, nil, "", err
	}

	overlay := &settings.InferenceSettings{
		Embeddings: &config.EmbeddingsConfig{
			Type:       embeddingType,
			Engine:     embeddingEngine,
			Dimensions: dimensions,
		},
	}
	merged, err := profiles.MergeInferenceSettings(baseResolved.InferenceSettings, overlay)
	if err != nil {
		_ = chain.Close()
		return nil, nil, "", err
	}
	return merged, chain.Close, fmt.Sprintf("%s + embeddings(%s/%s)", baseProfile, embeddingType, embeddingEngine), nil
}

func defaultPinocchioProfilesPath() string {
	home, err := homeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return filepath.Join(".config", "pinocchio", "profiles.yaml")
	}
	return filepath.Join(home, ".config", "pinocchio", "profiles.yaml")
}

func homeDir() (string, error) {
	return os.UserHomeDir()
}

func vectorPreview(vector []float32, n int) []float32 {
	if n <= 0 || len(vector) == 0 {
		return nil
	}
	if n > len(vector) {
		n = len(vector)
	}
	return vector[:n]
}

func openAIKeyConfigured(s *settings.InferenceSettings) bool {
	return s != nil && s.API != nil && strings.TrimSpace(s.API.APIKeys["openai-api-key"]) != ""
}

func baseURLConfigured(s *settings.InferenceSettings) bool {
	return s != nil && s.API != nil && (strings.TrimSpace(s.API.BaseUrls["ollama-base-url"]) != "" || strings.TrimSpace(s.API.BaseUrls["openai-base-url"]) != "")
}

func main() {
	root := examplecmd.NewRoot("embedding-profile-smoke", "Profile-backed embeddings smoke test")
	cmd, err := newEmbeddingCommand()
	cobra.CheckErr(err)
	cobra.CheckErr(examplecmd.ExecuteSingleCommand(root, "geppetto", cmd))
}
