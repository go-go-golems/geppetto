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
	glazedConfig "github.com/go-go-golems/glazed/pkg/config"
	"github.com/go-go-golems/glazed/pkg/cmds/middlewares"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
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
	commandSettings := &cli.CommandSettings{}
	err := parsedCommandLayers.InitializeStruct(cli.CommandSettingsSlug, commandSettings)
	if err != nil {
		return nil, err
	}

	profileSettings := &cli.ProfileSettings{}
	err = parsedCommandLayers.InitializeStruct(cli.ProfileSettingsSlug, profileSettings)
	if err != nil {
		return nil, err
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

	// Config files (low -> high precedence)
	// Note: We use parsedCommandLayers (captured from function parameter) instead of parsed
	// (from middleware) because flags haven't been parsed into parsedLayers yet when the
	// resolver runs. parsedCommandLayers already has command settings parsed from
	// ParseCommandSettingsLayer which runs before the middleware chain.
	configFilesResolver := func(_ *cmdlayers.ParsedLayers, _ *cobra.Command, _ []string) ([]string, error) {
		var files []string
		
		// Resolve app config path (XDG, ~/.pinocchio, /etc/pinocchio)
		// Load default config first (low precedence)
		configPath, err := glazedConfig.ResolveAppConfigPath("pinocchio", "")
		if err == nil && configPath != "" {
			files = append(files, configPath)
		}
		
		// Check for explicit config file from command settings (already parsed in parsedCommandLayers)
		// Use the commandSettings we read earlier from parsedCommandLayers
		// Explicit config is loaded last (high precedence)
		if commandSettings.ConfigFile != "" {
			files = append(files, commandSettings.ConfigFile)
		}
		
		// Also check LoadParametersFromFile for backward compatibility
		if commandSettings.LoadParametersFromFile != "" {
			files = append(files, commandSettings.LoadParametersFromFile)
		}
		
		return files, nil
	}
	
	// Mapper to filter out non-layer keys like "repositories" which are handled separately
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
	
	// Load config files with mapper to exclude repositories
	middlewares_ = append(middlewares_,
		func(next middlewares.HandlerFunc) middlewares.HandlerFunc {
			return func(layers_ *cmdlayers.ParameterLayers, parsedLayers *cmdlayers.ParsedLayers) error {
				if err := next(layers_, parsedLayers); err != nil {
					return err
				}
				files, err := configFilesResolver(parsedLayers, cmd, args)
				if err != nil {
					return err
				}
				return middlewares.LoadParametersFromFiles(files,
					middlewares.WithConfigFileMapper(configMapper),
					middlewares.WithParseOptions(
						parameters.WithParseStepSource("config"),
					),
				)(func(_ *cmdlayers.ParameterLayers, _ *cmdlayers.ParsedLayers) error { return nil })(layers_, parsedLayers)
			}
		},
	)

	xdgConfigPath, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	// Profile loading (NOTE: profile name is still read from defaults at construction time;
	// this will be fixed in plan point 3 to read after env/config are applied)
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

	// Lowest precedence: defaults
	middlewares_ = append(middlewares_,
		middlewares.SetFromDefaults(parameters.WithParseStepSource("defaults")),
	)

	return middlewares_, nil
}
