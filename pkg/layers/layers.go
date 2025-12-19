package layers

import (
	"fmt"
	embeddingsconfig "github.com/go-go-golems/geppetto/pkg/embeddings/config"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/gemini"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	"github.com/go-go-golems/glazed/pkg/cli"
	cmdlayers "github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/middlewares"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	glazedConfig "github.com/go-go-golems/glazed/pkg/config"
	"github.com/spf13/cobra"
	"os"
)

// CreateOption configures behavior of CreateGeppettoLayers.
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

// CreateGeppettoLayers returns parameter layers for Geppetto AI settings.
// If no StepSettings are provided via WithStepSettings, default settings.NewStepSettings() is used.
func CreateGeppettoLayers(opts ...CreateOption) ([]cmdlayers.ParameterLayer, error) {
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

	chatParameterLayer, err := settings.NewChatParameterLayer(cmdlayers.WithDefaults(ss.Chat))
	if err != nil {
		return nil, err
	}

	clientParameterLayer, err := settings.NewClientParameterLayer(cmdlayers.WithDefaults(ss.Client))
	if err != nil {
		return nil, err
	}

	claudeParameterLayer, err := claude.NewParameterLayer(cmdlayers.WithDefaults(ss.Claude))
	if err != nil {
		return nil, err
	}

	geminiParameterLayer, err := gemini.NewParameterLayer(cmdlayers.WithDefaults(ss.Gemini))
	if err != nil {
		return nil, err
	}

	openaiParameterLayer, err := openai.NewParameterLayer(cmdlayers.WithDefaults(ss.OpenAI))
	if err != nil {
		return nil, err
	}

	embeddingsParameterLayer, err := embeddingsconfig.NewEmbeddingsParameterLayer(cmdlayers.WithDefaults(ss.Embeddings))
	if err != nil {
		return nil, err
	}

	// Assemble layers
	result := []cmdlayers.ParameterLayer{
		chatParameterLayer,
		clientParameterLayer,
		claudeParameterLayer,
		geminiParameterLayer,
		openaiParameterLayer,
		embeddingsParameterLayer,
	}
	return result, nil
}

func GetCobraCommandGeppettoMiddlewares(
	parsedCommandLayers *cmdlayers.ParsedLayers,
	cmd *cobra.Command,
	args []string,
) ([]middlewares.Middleware, error) {
	// Mapper to filter out non-layer keys like "repositories" which are handled separately.
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

			// If the value is a map, treat the key as a layer slug
			if layerParams, ok := value.(map[string]interface{}); ok {
				result[key] = layerParams
			}
		}

		return result, nil
	}

	// ---------------------------------------------------------------------
	// Option A bootstrap parse:
	//
	// We must resolve profile selection (profile-settings.profile + profile-settings.profile-file)
	// from defaults + config + env + flags BEFORE instantiating the profiles middleware.
	//
	// NOTE: parsedCommandLayers (from cli.ParseCommandSettingsLayer) is Cobra-only. We keep it
	// around as a fallback source of command settings, but we do our own bootstrap parse so that
	// PINOCCHIO_PROFILE and config can influence selection.
	// ---------------------------------------------------------------------

	// 1) Bootstrap command settings from Cobra + env + defaults (no config).
	commandSettings := &cli.CommandSettings{}
	commandSettingsLayer, err := cli.NewCommandSettingsLayer()
	if err != nil {
		return nil, err
	}
	bootstrapCommandLayers := cmdlayers.NewParameterLayers(
		cmdlayers.WithLayers(commandSettingsLayer),
	)
	bootstrapCommandParsed := cmdlayers.NewParsedLayers()
	err = middlewares.ExecuteMiddlewares(
		bootstrapCommandLayers,
		bootstrapCommandParsed,
		middlewares.ParseFromCobraCommand(cmd, parameters.WithParseStepSource("cobra")),
		middlewares.UpdateFromEnv("PINOCCHIO", parameters.WithParseStepSource("env")),
		middlewares.SetFromDefaults(parameters.WithParseStepSource("defaults")),
	)
	if err != nil {
		return nil, err
	}
	if err := bootstrapCommandParsed.InitializeStruct(cli.CommandSettingsSlug, commandSettings); err != nil {
		return nil, err
	}
	// Backward-compatibility: if bootstrap didn't produce it, fall back to Cobra-only parsedCommandLayers.
	if commandSettings.ConfigFile == "" && commandSettings.LoadParametersFromFile == "" && parsedCommandLayers != nil {
		_ = parsedCommandLayers.InitializeStruct(cli.CommandSettingsSlug, commandSettings)
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
	if commandSettings.LoadParametersFromFile != "" {
		configFiles = append(configFiles, commandSettings.LoadParametersFromFile)
	}

	// 3) Bootstrap profile settings from config + env + Cobra + defaults.
	profileSettings := &cli.ProfileSettings{}
	profileSettingsLayer, err := cli.NewProfileSettingsLayer()
	if err != nil {
		return nil, err
	}
	bootstrapProfileLayers := cmdlayers.NewParameterLayers(
		cmdlayers.WithLayers(profileSettingsLayer),
	)
	bootstrapProfileParsed := cmdlayers.NewParsedLayers()
	err = middlewares.ExecuteMiddlewares(
		bootstrapProfileLayers,
		bootstrapProfileParsed,
		middlewares.ParseFromCobraCommand(cmd, parameters.WithParseStepSource("cobra")),
		middlewares.UpdateFromEnv("PINOCCHIO", parameters.WithParseStepSource("env")),
		middlewares.LoadParametersFromFiles(
			configFiles,
			middlewares.WithConfigFileMapper(configMapper),
			middlewares.WithParseOptions(parameters.WithParseStepSource("config")),
		),
		middlewares.SetFromDefaults(parameters.WithParseStepSource("defaults")),
	)
	if err != nil {
		return nil, err
	}
	if err := bootstrapProfileParsed.InitializeStruct(cli.ProfileSettingsSlug, profileSettings); err != nil {
		return nil, err
	}
	// Backward-compatibility: if bootstrap didn't produce it, fall back to Cobra-only parsedCommandLayers.
	if profileSettings.Profile == "" && profileSettings.ProfileFile == "" && parsedCommandLayers != nil {
		_ = parsedCommandLayers.InitializeStruct(cli.ProfileSettingsSlug, profileSettings)
	}

	// Build middleware chain in reverse precedence order (last applied has highest precedence)
	middlewares_ := []middlewares.Middleware{
		// Highest precedence: command-line flags
		middlewares.ParseFromCobraCommand(cmd,
			parameters.WithParseStepSource("cobra"),
		),
		// Positional arguments
		middlewares.GatherArguments(args,
			parameters.WithParseStepSource("arguments"),
		),
	}

	// Environment variables (PINOCCHIO_*)
	// Whitelist the same layers that were previously whitelisted for Viper parsing
	middlewares_ = append(middlewares_,
		middlewares.WrapWithWhitelistedLayers(
			[]string{
				settings.AiChatSlug,
				settings.AiClientSlug,
				openai.OpenAiChatSlug,
				claude.ClaudeChatSlug,
				gemini.GeminiChatSlug,
				embeddingsconfig.EmbeddingsSlug,
				cli.ProfileSettingsSlug,
			},
			middlewares.UpdateFromEnv("PINOCCHIO",
				parameters.WithParseStepSource("env"),
			),
		),
	)

	xdgConfigPath, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	// Profile loading:
	// - profile-settings are resolved via a bootstrap parse above (defaults + config + env + flags)
	// - profile values are then loaded at the correct precedence level (above defaults, below config/env/flags)
	defaultProfileFile := fmt.Sprintf("%s/pinocchio/profiles.yaml", xdgConfigPath)
	if profileSettings.ProfileFile == "" {
		profileSettings.ProfileFile = defaultProfileFile
	}
	if profileSettings.Profile == "" {
		profileSettings.Profile = "default"
	}
	middlewares_ = append(middlewares_,
		middlewares.GatherFlagsFromProfiles(
			defaultProfileFile,
			profileSettings.ProfileFile,
			profileSettings.Profile,
			parameters.WithParseStepSource("profiles"),
			parameters.WithParseStepMetadata(map[string]interface{}{
				"profileFile": profileSettings.ProfileFile,
				"profile":     profileSettings.Profile,
			}),
		),
	)

	// Config files (low -> high precedence) - resolved once above to keep bootstrap + main chain consistent.
	//
	// NOTE: This is intentionally placed AFTER the profiles middleware in the slice ordering.
	// Most Glazed "value-setting" middlewares call next(...) first and then update parsedLayers,
	// so later middlewares in the slice apply earlier. By placing config after profiles here,
	// config is applied BEFORE profiles, ensuring profiles override config (while env/flags still override both).
	middlewares_ = append(middlewares_,
		middlewares.LoadParametersFromFiles(
			configFiles,
			middlewares.WithConfigFileMapper(configMapper),
			middlewares.WithParseOptions(parameters.WithParseStepSource("config")),
		),
	)

	// Lowest precedence: defaults
	middlewares_ = append(middlewares_,
		middlewares.SetFromDefaults(parameters.WithParseStepSource("defaults")),
	)

	return middlewares_, nil
}
