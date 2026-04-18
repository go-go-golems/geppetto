package bootstrap

import (
	"context"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	glazedconfig "github.com/go-go-golems/glazed/pkg/config"
)

type ProfileSettings struct {
	Profile           string   `glazed:"profile"`
	ProfileRegistries []string `glazed:"profile-registries"`
}

type ResolvedCLIConfigFiles struct {
	Paths  []string
	Files  []glazedconfig.ResolvedConfigFile
	Report *glazedconfig.PlanReport
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

func PrepareProfileSettingsForRuntime(cfg AppBootstrapConfig, settings ProfileSettings) ProfileSettings {
	settings.Profile = strings.TrimSpace(settings.Profile)
	settings.ProfileRegistries = normalizeProfileRegistries(settings.ProfileRegistries)
	if len(settings.ProfileRegistries) == 0 {
		settings.ProfileRegistries = normalizeProfileRegistries(defaultProfileRegistrySources(cfg))
	}
	return settings
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
	resolved, err := ResolveCLIConfigFilesResolved(cfg, parsed)
	if err != nil {
		return nil, err
	}
	return append([]string(nil), resolved.Paths...), nil
}

func ResolveCLIConfigFilesForExplicit(cfg AppBootstrapConfig, explicit string) ([]string, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(explicit) == "" {
		return ResolveCLIConfigFiles(cfg, nil)
	}
	parsed, err := NewCLISelectionValues(cfg, CLISelectionInput{ConfigFile: explicit})
	if err != nil {
		return nil, err
	}
	return ResolveCLIConfigFiles(cfg, parsed)
}

func ResolveCLIConfigFilesResolved(cfg AppBootstrapConfig, parsed *values.Values) (*ResolvedCLIConfigFiles, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	plan, err := cfg.ConfigPlanBuilder(parsed)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return &ResolvedCLIConfigFiles{}, nil
	}

	files, report, err := plan.Resolve(context.Background())
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}
	return &ResolvedCLIConfigFiles{
		Paths:  paths,
		Files:  append([]glazedconfig.ResolvedConfigFile(nil), files...),
		Report: report,
	}, nil
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
