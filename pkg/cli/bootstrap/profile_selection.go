package bootstrap

import (
	"strings"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	appconfig "github.com/go-go-golems/glazed/pkg/config"
	"github.com/pkg/errors"
)

type ProfileSettings struct {
	Profile           string   `glazed:"profile"`
	ProfileRegistries []string `glazed:"profile-registries"`
}

type ResolvedCLIProfileSelection struct {
	ProfileSettings
	ConfigFiles []string
}

type CLISelectionInput struct {
	ConfigFile        string
	Profile           string
	ProfileRegistries []string
}

func NewProfileSettingsSection(cfg AppBootstrapConfig) (schema.Section, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg.NewProfileSection()
}

func ResolveProfileSettings(parsed *values.Values) ProfileSettings {
	ret := ProfileSettings{}
	if parsed != nil {
		_ = parsed.DecodeSectionInto(ProfileSettingsSectionSlug, &ret)
	}
	ret.Profile = strings.TrimSpace(ret.Profile)
	ret.ProfileRegistries = normalizeProfileRegistries(ret.ProfileRegistries)
	return ret
}

func ResolveCLIProfileSelection(cfg AppBootstrapConfig, parsed *values.Values) (*ResolvedCLIProfileSelection, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	profileSection, err := cfg.NewProfileSection()
	if err != nil {
		return nil, errors.Wrap(err, "create profile settings section")
	}

	schema_ := schema.NewSchema(schema.WithSections(profileSection))
	resolvedValues := values.New()
	configFiles, err := ResolveCLIConfigFiles(cfg, parsed)
	if err != nil {
		return nil, err
	}
	if err := sources.Execute(
		schema_,
		resolvedValues,
		sources.FromEnv(cfg.normalizedEnvPrefix(), fields.WithSource("env")),
		sources.FromFiles(
			configFiles,
			sources.WithConfigFileMapper(cfg.ConfigFileMapper),
			sources.WithParseOptions(fields.WithSource("config")),
		),
		sources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
	); err != nil {
		return nil, errors.Wrap(err, "resolve profile settings from config/env/defaults")
	}
	if parsed != nil {
		if err := resolvedValues.Merge(parsed); err != nil {
			return nil, errors.Wrap(err, "merge explicit profile settings")
		}
	}

	profileSettings := ResolveProfileSettings(resolvedValues)
	return &ResolvedCLIProfileSelection{
		ProfileSettings: profileSettings,
		ConfigFiles:     append([]string(nil), configFiles...),
	}, nil
}

func ResolveEngineProfileSettings(cfg AppBootstrapConfig, parsed *values.Values) (ProfileSettings, []string, error) {
	resolved, err := ResolveCLIProfileSelection(cfg, parsed)
	if err != nil {
		return ProfileSettings{}, nil, err
	}
	return resolved.ProfileSettings, resolved.ConfigFiles, nil
}

func NewCLISelectionValues(cfg AppBootstrapConfig, input CLISelectionInput) (*values.Values, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	ret := values.New()

	commandSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}
	commandValues, err := values.NewSectionValues(commandSection)
	if err != nil {
		return nil, err
	}
	if configFile := strings.TrimSpace(input.ConfigFile); configFile != "" {
		if err := values.WithFieldValue("config-file", configFile, fields.WithSource("cli"))(commandValues); err != nil {
			return nil, err
		}
	}
	ret.Set(cli.CommandSettingsSlug, commandValues)

	profileSection, err := cfg.NewProfileSection()
	if err != nil {
		return nil, err
	}
	profileValues, err := values.NewSectionValues(profileSection)
	if err != nil {
		return nil, err
	}
	if profile := strings.TrimSpace(input.Profile); profile != "" {
		if err := values.WithFieldValue("profile", profile, fields.WithSource("cli"))(profileValues); err != nil {
			return nil, err
		}
	}
	if registries := normalizeProfileRegistries(input.ProfileRegistries); len(registries) > 0 {
		if err := values.WithFieldValue("profile-registries", registries, fields.WithSource("cli"))(profileValues); err != nil {
			return nil, err
		}
	}
	ret.Set(ProfileSettingsSectionSlug, profileValues)

	return ret, nil
}

func ResolveCLIConfigFiles(cfg AppBootstrapConfig, parsed *values.Values) ([]string, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	files := make([]string, 0, 2)
	defaultFile, err := appconfig.ResolveAppConfigPath(cfg.normalizedAppName(), "")
	if err != nil {
		return nil, errors.Wrapf(err, "resolve %s default config path", cfg.normalizedAppName())
	}
	if defaultFile != "" {
		files = append(files, defaultFile)
	}
	if parsed != nil {
		commandSettings := &cli.CommandSettings{}
		if err := parsed.DecodeSectionInto(cli.CommandSettingsSlug, commandSettings); err == nil {
			explicit := strings.TrimSpace(commandSettings.ConfigFile)
			if explicit != "" {
				explicitPath, err := appconfig.ResolveAppConfigPath(cfg.normalizedAppName(), explicit)
				if err != nil {
					return nil, err
				}
				if explicitPath != "" {
					duplicate := false
					for _, f := range files {
						if f == explicitPath {
							duplicate = true
							break
						}
					}
					if !duplicate {
						files = append(files, explicitPath)
					}
				}
			}
		}
	}
	return files, nil
}

func ResolveCLIConfigFilesForExplicit(cfg AppBootstrapConfig, explicit string) ([]string, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	files, err := ResolveCLIConfigFiles(cfg, nil)
	if err != nil {
		return nil, err
	}
	explicitPath, err := appconfig.ResolveAppConfigPath(cfg.normalizedAppName(), explicit)
	if err != nil {
		return nil, err
	}
	if explicitPath == "" {
		return files, nil
	}
	for _, f := range files {
		if f == explicitPath {
			return files, nil
		}
	}
	return append(files, explicitPath), nil
}

func normalizeProfileRegistries(entries []string) []string {
	ret := make([]string, 0, len(entries))
	for _, entry := range entries {
		if trimmed := strings.TrimSpace(entry); trimmed != "" {
			ret = append(ret, trimmed)
		}
	}
	return ret
}
