package sections

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	embeddingsconfig "github.com/go-go-golems/geppetto/pkg/embeddings/config"
	profiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/gemini"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	glazedConfig "github.com/go-go-golems/glazed/pkg/config"
	"github.com/spf13/cobra"
)

type ProfileSettings struct {
	Profile           string   `glazed:"profile"`
	ProfileRegistries []string `glazed:"profile-registries"`
}

type ProfileSettingsSectionOption func(*profileSettingsSectionOptions)

type profileSettingsSectionOptions struct {
	profileDefault           string
	profileRegistriesDefault []string
}

func WithProfileDefault(profile string) ProfileSettingsSectionOption {
	return func(o *profileSettingsSectionOptions) {
		o.profileDefault = strings.TrimSpace(profile)
	}
}

func WithProfileRegistriesDefault(entries ...string) ProfileSettingsSectionOption {
	return func(o *profileSettingsSectionOptions) {
		o.profileRegistriesDefault = o.profileRegistriesDefault[:0]
		for _, entry := range entries {
			if trimmed := strings.TrimSpace(entry); trimmed != "" {
				o.profileRegistriesDefault = append(o.profileRegistriesDefault, trimmed)
			}
		}
	}
}

const ProfileSettingsSectionSlug = "profile-settings"

func defaultPinocchioProfileRegistriesIfPresent() string {
	configDir, err := os.UserConfigDir()
	if err != nil || strings.TrimSpace(configDir) == "" {
		return ""
	}
	path := filepath.Join(configDir, "pinocchio", "profiles.yaml")
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return ""
	}
	return path
}

func NewProfileSettingsSection(opts ...ProfileSettingsSectionOption) (schema.Section, error) {
	var sectionOptions profileSettingsSectionOptions
	for _, opt := range opts {
		opt(&sectionOptions)
	}

	profileOptions := []fields.Option{
		fields.WithHelp("Load the profile"),
	}
	if sectionOptions.profileDefault != "" {
		profileOptions = append(profileOptions, fields.WithDefault(sectionOptions.profileDefault))
	}

	profileRegistriesOptions := []fields.Option{
		fields.WithHelp("Comma-separated profile registry sources (yaml/sqlite/sqlite-dsn)"),
	}
	if len(sectionOptions.profileRegistriesDefault) > 0 {
		profileRegistriesOptions = append(profileRegistriesOptions, fields.WithDefault(append([]string(nil), sectionOptions.profileRegistriesDefault...)))
	}

	return schema.NewSection(
		ProfileSettingsSectionSlug,
		"Profile settings",
		schema.WithFields(
			fields.New("profile", fields.TypeString, profileOptions...),
			fields.New("profile-registries", fields.TypeStringList, profileRegistriesOptions...),
		),
	)
}

// GetProfileSettingsMiddleware remains for legacy Cobra middleware wiring.
// New CLI bootstrap paths should prefer geppetto/pkg/cli/bootstrap with an
// explicit AppBootstrapConfig instead of extending this helper.
func GetProfileSettingsMiddleware(
	parsedCommandSections *values.Values,
	cmd *cobra.Command,
	args []string,
) ([]sources.Middleware, error) {
	// Mapper to filter out non-section keys like "repositories" which are handled separately.
	// We keep it here so it can be reused both for bootstrap parsing (profile selection)
	// and for the main config middleware.
	configMapper := func(rawConfig interface{}) (map[string]map[string]interface{}, error) {
		configMap, ok := rawConfig.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("expected map[string]interface{}, got %T", rawConfig)
		}

		result := make(map[string]map[string]interface{})

		// Keys to exclude from layer parsing (handled separately)
		excludedKeys := map[string]bool{
			"repositories": true,
		}

		for key, value := range configMap {
			if excludedKeys[key] {
				continue // Skip excluded keys
			}

			// If the value is a map, treat the key as a section slug
			if layerParams, ok := value.(map[string]interface{}); ok {
				result[key] = layerParams
			}
		}

		return result, nil
	}

	// ---------------------------------------------------------------------
	// Option A bootstrap parse:
	//
	// We must resolve profile selection (profile-settings.profile + profile-settings.profile-registries)
	// from defaults + config + env + flags BEFORE instantiating the profiles middleware.
	//
	// NOTE: parsedCommandSections (from cli.ParseCommandSettingsLayer) is Cobra-only. We keep it
	// around as a fallback source of command settings, but we do our own bootstrap parse so that
	// PINOCCHIO_PROFILE and config can influence selection.
	// ---------------------------------------------------------------------

	// 1) Bootstrap command settings from Cobra + env + defaults (no config).
	commandSettings := &cli.CommandSettings{}
	commandSettingsLayer, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}
	bootstrapCommandSchema := schema.NewSchema(schema.WithSections(commandSettingsLayer))
	bootstrapCommandParsed := values.New()
	err = sources.Execute(
		bootstrapCommandSchema,
		bootstrapCommandParsed,
		sources.FromCobra(cmd, fields.WithSource("cobra")),
		sources.FromEnv("PINOCCHIO", fields.WithSource("env")),
		sources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
	)
	if err != nil {
		return nil, err
	}
	if err := bootstrapCommandParsed.DecodeSectionInto(cli.CommandSettingsSlug, commandSettings); err != nil {
		return nil, err
	}
	// Backward-compatibility: if bootstrap didn't produce it, fall back to Cobra-only parsed command settings.
	if commandSettings.ConfigFile == "" && parsedCommandSections != nil {
		_ = parsedCommandSections.DecodeSectionInto(cli.CommandSettingsSlug, commandSettings)
	}

	// 2) Resolve config files once (low -> high precedence) so bootstrap + main chain are consistent.
	var configFiles []string
	configPath, err := glazedConfig.ResolveAppConfigPath("pinocchio", "")
	if err == nil && configPath != "" {
		configFiles = append(configFiles, configPath)
	}
	if commandSettings.ConfigFile != "" {
		configFiles = append(configFiles, commandSettings.ConfigFile)
	}

	// 3) Bootstrap profile settings from config + env + Cobra + defaults.
	profileSettings := &ProfileSettings{}
	profileSettingsSection, err := NewProfileSettingsSection()
	if err != nil {
		return nil, err
	}
	bootstrapProfileSchema := schema.NewSchema(schema.WithSections(profileSettingsSection))
	bootstrapProfileParsed := values.New()
	err = sources.Execute(
		bootstrapProfileSchema,
		bootstrapProfileParsed,
		sources.FromCobra(cmd, fields.WithSource("cobra")),
		sources.FromEnv("PINOCCHIO", fields.WithSource("env")),
		sources.FromFiles(
			configFiles,
			sources.WithConfigFileMapper(configMapper),
			sources.WithParseOptions(fields.WithSource("config")),
		),
		sources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
	)
	if err != nil {
		return nil, err
	}
	if err := bootstrapProfileParsed.DecodeSectionInto(ProfileSettingsSectionSlug, profileSettings); err != nil {
		return nil, err
	}
	// Backward-compatibility: if bootstrap didn't produce it, fall back to Cobra-only parsed command settings.
	if profileSettings.Profile == "" && parsedCommandSections != nil {
		_ = parsedCommandSections.DecodeSectionInto(ProfileSettingsSectionSlug, profileSettings)
	}
	if profileSettings.Profile == "" {
		profileSettings.Profile = "default"
	}
	if len(profileSettings.ProfileRegistries) == 0 {
		if defaultPath := defaultPinocchioProfileRegistriesIfPresent(); defaultPath != "" {
			profileSettings.ProfileRegistries = []string{defaultPath}
		}
	}
	if len(profileSettings.ProfileRegistries) == 0 {
		return nil, &profiles.ValidationError{Field: "profile-settings.profile-registries", Reason: "must be configured (hard cutover: no profile-file fallback)"}
	}

	// Build middleware chain in reverse precedence order (last applied has highest precedence)
	middlewares_ := []sources.Middleware{
		// Highest precedence: command-line flags
		sources.FromCobra(cmd,
			fields.WithSource("cobra"),
		),
		// Positional arguments
		sources.FromArgs(args,
			fields.WithSource("arguments"),
		),
	}

	// Environment variables (PINOCCHIO_*)
	// Whitelist the same layers that were previously whitelisted for Viper parsing
	middlewares_ = append(middlewares_,
		sources.WrapWithWhitelistedSections(
			[]string{
				settings.AiChatSlug,
				settings.AiClientSlug,
				settings.AiInferenceSlug,
				openai.OpenAiChatSlug,
				claude.ClaudeChatSlug,
				gemini.GeminiChatSlug,
				embeddingsconfig.EmbeddingsSlug,
				ProfileSettingsSectionSlug,
			},
			sources.FromEnv("PINOCCHIO",
				fields.WithSource("env"),
			),
		),
	)

	// Config files (low -> high precedence) - resolved once above to keep bootstrap + main chain consistent.
	middlewares_ = append(middlewares_, sources.FromFiles(
		configFiles,
		sources.WithConfigFileMapper(configMapper),
		sources.WithParseOptions(fields.WithSource("config")),
	))

	// Lowest precedence: defaults
	middlewares_ = append(middlewares_,
		sources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
	)

	return middlewares_, nil
}
