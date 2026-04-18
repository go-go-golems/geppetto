package bootstrap

import (
	"context"

	gepprofiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
	aisettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/pkg/errors"
)

type ResolvedCLIEngineSettings struct {
	BaseInferenceSettings  *aisettings.InferenceSettings
	FinalInferenceSettings *aisettings.InferenceSettings
	ProfileRuntime         *ResolvedCLIProfileRuntime
	ResolvedEngineProfile  *gepprofiles.ResolvedEngineProfile
	ConfigFiles            []string
	Close                  func()
}

func ResolveBaseInferenceSettings(cfg AppBootstrapConfig, parsed *values.Values) (*aisettings.InferenceSettings, []string, error) {
	if err := cfg.Validate(); err != nil {
		return nil, nil, err
	}

	sections_, err := cfg.BuildBaseSections()
	if err != nil {
		return nil, nil, errors.Wrap(err, "create hidden base sections")
	}
	schema_ := schema.NewSchema(schema.WithSections(sections_...))
	parsedValues := values.New()
	configMiddleware, configFiles, err := resolveConfigMiddleware(cfg, parsed)
	if err != nil {
		return nil, nil, err
	}
	if err := sources.Execute(
		schema_,
		parsedValues,
		sources.FromEnv(cfg.normalizedEnvPrefix(), fields.WithSource("env")),
		configMiddleware,
		sources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
	); err != nil {
		return nil, configFiles.Paths, errors.Wrap(err, "resolve hidden base inference settings")
	}
	stepSettings, err := aisettings.NewInferenceSettingsFromParsedValues(parsedValues)
	if err != nil {
		return nil, configFiles.Paths, errors.Wrap(err, "build inference settings from hidden parsed values")
	}
	return stepSettings, configFiles.Paths, nil
}

func ResolveCLIEngineSettings(
	ctx context.Context,
	cfg AppBootstrapConfig,
	parsed *values.Values,
) (*ResolvedCLIEngineSettings, error) {
	base, baseConfigFiles, err := ResolveBaseInferenceSettings(cfg, parsed)
	if err != nil {
		return nil, err
	}
	return ResolveCLIEngineSettingsFromBase(ctx, cfg, base, parsed, baseConfigFiles)
}

func ResolveCLIEngineSettingsFromBase(
	ctx context.Context,
	cfg AppBootstrapConfig,
	base *aisettings.InferenceSettings,
	parsed *values.Values,
	baseConfigFiles []string,
) (*ResolvedCLIEngineSettings, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if base == nil {
		return nil, errors.New("base inference settings cannot be nil")
	}

	profileRuntime, err := ResolveCLIProfileRuntime(ctx, cfg, parsed)
	if err != nil {
		return nil, err
	}

	// Start with base config files, but if the profile runtime resolved
	// its own config files, those replace the base list entirely. This is
	// because profile runtime runs the same config plan as base, so its
	// config files are a superset (or equal) to the base files.
	configFiles := append([]string(nil), baseConfigFiles...)
	if len(profileRuntime.ConfigFiles) > 0 {
		configFiles = append([]string(nil), profileRuntime.ConfigFiles...)
	}

	registryChain := profileRuntime.ProfileRegistryChain
	if registryChain == nil || registryChain.Registry == nil {
		return &ResolvedCLIEngineSettings{
			BaseInferenceSettings:  base,
			FinalInferenceSettings: base,
			ProfileRuntime:         profileRuntime,
			ConfigFiles:            configFiles,
			Close:                  profileRuntime.Close,
		}, nil
	}

	resolved, err := registryChain.Registry.ResolveEngineProfile(ctx, registryChain.DefaultProfileResolve)
	if err != nil {
		if profileRuntime.Close != nil {
			profileRuntime.Close()
		}
		return nil, err
	}
	finalSettings, err := gepprofiles.MergeInferenceSettings(base, resolved.InferenceSettings)
	if err != nil {
		if profileRuntime.Close != nil {
			profileRuntime.Close()
		}
		return nil, errors.Wrap(err, "merge base inference settings with engine profile")
	}

	return &ResolvedCLIEngineSettings{
		BaseInferenceSettings:  base,
		FinalInferenceSettings: finalSettings,
		ProfileRuntime:         profileRuntime,
		ResolvedEngineProfile:  resolved,
		ConfigFiles:            configFiles,
		Close:                  profileRuntime.Close,
	}, nil
}

func NewEngineFromResolvedCLIEngineSettings(
	resolved *ResolvedCLIEngineSettings,
) (engine.Engine, error) {
	return NewEngineFromResolvedCLIEngineSettingsWithFactory(nil, resolved)
}

func NewEngineFromResolvedCLIEngineSettingsWithFactory(
	engineFactory factory.EngineFactory,
	resolved *ResolvedCLIEngineSettings,
) (engine.Engine, error) {
	if resolved == nil {
		return nil, errors.New("resolved engine settings cannot be nil")
	}
	if resolved.FinalInferenceSettings == nil {
		return nil, errors.New("resolved final inference settings cannot be nil")
	}
	if engineFactory == nil {
		engineFactory = factory.NewStandardEngineFactory()
	}
	return engineFactory.CreateEngine(resolved.FinalInferenceSettings)
}
