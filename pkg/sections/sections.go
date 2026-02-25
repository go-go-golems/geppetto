package sections

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	embeddingsconfig "github.com/go-go-golems/geppetto/pkg/embeddings/config"
	"github.com/go-go-golems/geppetto/pkg/profiles"
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

// CreateOption configures behavior of CreateGeppettoSections.
type CreateOption func(*createOptions)
type createOptions struct {
	stepSettings *settings.StepSettings
}

// WithDefaultsFromStepSettings uses the given StepSettings for layer defaults.
func WithDefaultsFromStepSettings(s *settings.StepSettings) CreateOption {
	return func(o *createOptions) {
		o.stepSettings = s
	}
}

// CreateGeppettoSections returns settings sections for Geppetto AI settings.
// If no StepSettings are provided via WithStepSettings, default settings.NewStepSettings() is used.
func CreateGeppettoSections(opts ...CreateOption) ([]schema.Section, error) {
	// Apply options
	var co createOptions
	for _, opt := range opts {
		opt(&co)
	}
	// Determine StepSettings
	var ss *settings.StepSettings
	if co.stepSettings == nil {
		var err error
		ss, err = settings.NewStepSettings()
		if err != nil {
			return nil, err
		}
	} else {
		ss = co.stepSettings
	}

	chatSection, err := settings.NewChatValueSection()
	if err != nil {
		return nil, err
	}
	if err := chatSection.InitializeDefaultsFromStruct(ss.Chat); err != nil {
		return nil, err
	}

	clientSection, err := settings.NewClientValueSection()
	if err != nil {
		return nil, err
	}
	if err := clientSection.InitializeDefaultsFromStruct(ss.Client); err != nil {
		return nil, err
	}

	claudeSection, err := claude.NewValueSection()
	if err != nil {
		return nil, err
	}
	if err := claudeSection.InitializeDefaultsFromStruct(ss.Claude); err != nil {
		return nil, err
	}

	geminiSection, err := gemini.NewValueSection()
	if err != nil {
		return nil, err
	}
	if err := geminiSection.InitializeDefaultsFromStruct(ss.Gemini); err != nil {
		return nil, err
	}

	openaiSection, err := openai.NewValueSection()
	if err != nil {
		return nil, err
	}
	if err := openaiSection.InitializeDefaultsFromStruct(ss.OpenAI); err != nil {
		return nil, err
	}

	embeddingsSection, err := embeddingsconfig.NewEmbeddingsValueSection()
	if err != nil {
		return nil, err
	}
	if err := embeddingsSection.InitializeDefaultsFromStruct(ss.Embeddings); err != nil {
		return nil, err
	}

	inferenceSection, err := settings.NewInferenceValueSection()
	if err != nil {
		return nil, err
	}
	if err := inferenceSection.InitializeDefaultsFromStruct(ss.Inference); err != nil {
		return nil, err
	}

	// Assemble sections
	result := []schema.Section{
		chatSection,
		clientSection,
		claudeSection,
		geminiSection,
		openaiSection,
		embeddingsSection,
		inferenceSection,
	}
	return result, nil
}

type profileRegistrySettings struct {
	Profile           string `glazed:"profile"`
	ProfileRegistries string `glazed:"profile-registries"`
}

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

func newProfileRegistrySettingsSection() (schema.Section, error) {
	return schema.NewSection(
		cli.ProfileSettingsSlug,
		"Profile settings",
		schema.WithFields(
			fields.New(
				"profile",
				fields.TypeString,
				fields.WithHelp("Load the profile"),
			),
			fields.New(
				"profile-registries",
				fields.TypeString,
				fields.WithHelp("Comma-separated profile registry sources (yaml/sqlite/sqlite-dsn)"),
			),
		),
	)
}

func GetCobraCommandGeppettoMiddlewares(
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
	profileSettings := &profileRegistrySettings{}
	profileSettingsLayer, err := newProfileRegistrySettingsSection()
	if err != nil {
		return nil, err
	}
	bootstrapProfileSchema := schema.NewSchema(schema.WithSections(profileSettingsLayer))
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
	if err := bootstrapProfileParsed.DecodeSectionInto(cli.ProfileSettingsSlug, profileSettings); err != nil {
		return nil, err
	}
	// Backward-compatibility: if bootstrap didn't produce it, fall back to Cobra-only parsed command settings.
	if profileSettings.Profile == "" && parsedCommandSections != nil {
		_ = parsedCommandSections.DecodeSectionInto(cli.ProfileSettingsSlug, profileSettings)
	}
	if profileSettings.Profile == "" {
		profileSettings.Profile = "default"
	}
	if strings.TrimSpace(profileSettings.ProfileRegistries) == "" {
		if defaultPath := defaultPinocchioProfileRegistriesIfPresent(); defaultPath != "" {
			profileSettings.ProfileRegistries = defaultPath
		}
	}
	profileRegistrySources, err := profiles.ParseProfileRegistrySourceEntries(profileSettings.ProfileRegistries)
	if err != nil {
		return nil, err
	}
	if len(profileRegistrySources) == 0 {
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
				cli.ProfileSettingsSlug,
			},
			sources.FromEnv("PINOCCHIO",
				fields.WithSource("env"),
			),
		),
	)

	// Profile loading:
	// - profile-settings are resolved via a bootstrap parse above (defaults + config + env + flags)
	// - profile values are then loaded at the correct precedence level (above defaults, below config/env/flags)
	profileMiddleware := GatherFlagsFromProfileRegistry(
		profileRegistrySources,
		profileSettings.Profile,
		fields.WithSource("profiles"),
		fields.WithMetadata(map[string]interface{}{
			"profileRegistries": profileRegistrySources,
			"profile":           profileSettings.Profile,
			"mode":              "profile-registry-stack",
		}),
	)
	middlewares_ = append(middlewares_,
		profileMiddleware,
	)

	// Config files (low -> high precedence) - resolved once above to keep bootstrap + main chain consistent.
	//
	// NOTE: This is intentionally placed AFTER the profiles middleware in the slice ordering.
	// Most Glazed "value-setting" middlewares call next(...) first and then update parsedValues,
	// so later middlewares in the slice apply earlier. By placing config after profiles here,
	// config is applied BEFORE profiles, ensuring profiles override config (while env/flags still override both).
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
