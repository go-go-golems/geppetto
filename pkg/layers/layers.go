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
	appconfig "github.com/go-go-golems/glazed/pkg/config"
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

	// if we want profile support here, we would have to check for a --profile and --profile-file flag,
	// then load the file (or the default file), check for the profile values, then apply them before load-parameters-from-file

	middlewares_ := []middlewares.Middleware{
		middlewares.ParseFromCobraCommand(cmd,
			parameters.WithParseStepSource("cobra"),
		),
		middlewares.GatherArguments(args,
			parameters.WithParseStepSource("arguments"),
		),
	}

	if commandSettings.LoadParametersFromFile != "" {
		middlewares_ = append(middlewares_,
			middlewares.LoadParametersFromFile(commandSettings.LoadParametersFromFile))
	}

	xdgConfigPath, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	// TODO(manuel, 2024-03-20) I wonder if we should just use a custom layer for the profiles, as we want to load
	// the profile from the environment as well. So the sequence would be defaults -> viper -> command line
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

	// Discover config file using ResolveAppConfigPath
	configPath, err := appconfig.ResolveAppConfigPath("pinocchio", "")
	if err != nil {
		return nil, err
	}

	// Build config file and env middlewares wrapped with whitelisted layers
	configMiddlewares := []middlewares.Middleware{}
	if configPath != "" {
		configMiddlewares = append(configMiddlewares,
			middlewares.LoadParametersFromFile(configPath,
				middlewares.WithParseOptions(parameters.WithParseStepSource("config")),
			),
		)
	}
	configMiddlewares = append(configMiddlewares,
		middlewares.UpdateFromEnv("PINOCCHIO",
			parameters.WithParseStepSource("env"),
		),
	)

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
			middlewares.Chain(configMiddlewares...),
		),
		middlewares.SetFromDefaults(parameters.WithParseStepSource("defaults")),
	)

	return middlewares_, nil
}
