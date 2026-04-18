package bootstrap

import (
	"context"

	gepprofiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/pkg/errors"
)

type ResolvedCLIProfileRuntime struct {
	ProfileSettings      ProfileSettings
	ConfigFiles          []string
	ProfileRegistryChain *ResolvedProfileRegistryChain
	Close                func()
}

func ResolveCLIProfileRuntime(
	ctx context.Context,
	cfg AppBootstrapConfig,
	parsed *values.Values,
) (*ResolvedCLIProfileRuntime, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	profileSection, err := cfg.NewProfileSection()
	if err != nil {
		return nil, errors.Wrap(err, "create profile settings section")
	}

	schema_ := schema.NewSchema(schema.WithSections(profileSection))
	resolvedValues := values.New()
	configMiddleware, configFiles, err := resolveConfigMiddleware(cfg, parsed)
	if err != nil {
		return nil, err
	}
	if err := sources.Execute(
		schema_,
		resolvedValues,
		sources.FromEnv(cfg.normalizedEnvPrefix(), fields.WithSource("env")),
		configMiddleware,
		sources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
	); err != nil {
		return nil, errors.Wrap(err, "resolve profile settings from config/env/defaults")
	}
	if parsed != nil {
		if err := resolvedValues.Merge(parsed); err != nil {
			return nil, errors.Wrap(err, "merge explicit profile settings")
		}
	}

	return ResolveCLIProfileRuntimeFromSettings(ctx, cfg, ResolveProfileSettings(resolvedValues), configFiles.Paths)
}

func ResolveCLIProfileRuntimeFromSettings(
	ctx context.Context,
	cfg AppBootstrapConfig,
	settings ProfileSettings,
	configFiles []string,
) (*ResolvedCLIProfileRuntime, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	settings = PrepareProfileSettingsForRuntime(cfg, settings)
	registryChain, err := ResolveProfileRegistryChain(ctx, settings)
	if err != nil {
		return nil, err
	}

	ret := &ResolvedCLIProfileRuntime{
		ProfileSettings:      settings,
		ConfigFiles:          append([]string(nil), configFiles...),
		ProfileRegistryChain: registryChain,
	}
	if registryChain != nil {
		ret.Close = registryChain.Close
	}
	return ret, nil
}

func (r *ResolvedCLIProfileRuntime) Registry() gepprofiles.Registry {
	if r == nil || r.ProfileRegistryChain == nil {
		return nil
	}
	return r.ProfileRegistryChain.Registry
}

func (r *ResolvedCLIProfileRuntime) Reader() gepprofiles.RegistryReader {
	if r == nil || r.ProfileRegistryChain == nil {
		return nil
	}
	return r.ProfileRegistryChain.Reader
}
