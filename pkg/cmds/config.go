package cmds

import (
	"fmt"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/middlewares"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"os"
)

// LoadConfigFromSettings loads the geppetto step settings from the given profile and config file.
func LoadConfigFromSettings(settings_ cli.GlazedCommandSettings) (*settings.StepSettings, error) {
	middlewares_ := []middlewares.Middleware{}

	xdgConfigPath, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	// TODO(manuel, 2024-03-20) I wonder if we should just use a custom layer for the profiles, as we want to load
	// the profile from the environment as well. So the sequence would be defaults -> viper -> command line
	defaultProfileFile := fmt.Sprintf("%s/pinocchio/profiles.yaml", xdgConfigPath)
	if settings_.ProfileFile == "" {
		settings_.ProfileFile = defaultProfileFile
	}
	if settings_.Profile == "" {
		settings_.Profile = "default"
	}
	middlewares_ = append(middlewares_,
		middlewares.GatherFlagsFromProfiles(
			defaultProfileFile,
			settings_.ProfileFile,
			settings_.Profile,
			parameters.WithParseStepSource("profiles"),
			parameters.WithParseStepMetadata(map[string]interface{}{
				"profileFile": settings_.ProfileFile,
				"profile":     settings_.Profile,
			}),
		),
	)

	middlewares_ = append(middlewares_,
		middlewares.WrapWithWhitelistedLayers(
			[]string{
				settings.AiChatSlug,
				settings.AiClientSlug,
				openai.OpenAiChatSlug,
				claude.ClaudeChatSlug,
			},
			middlewares.GatherFlagsFromViper(parameters.WithParseStepSource("viper")),
		),
		middlewares.SetFromDefaults(parameters.WithParseStepSource("defaults")),
	)

	stepSettings, err := settings.NewStepSettings()
	if err != nil {
		return nil, err
	}

	geppettoLayers, err := CreateGeppettoLayers(stepSettings)
	if err != nil {
		return nil, err
	}

	layers_ := layers.NewParameterLayers(layers.WithLayers(geppettoLayers...))

	parsedLayers := layers.NewParsedLayers()

	err = middlewares.ExecuteMiddlewares(layers_, parsedLayers, middlewares_...)
	if err != nil {
		return nil, err
	}

	err = stepSettings.UpdateFromParsedLayers(parsedLayers)
	if err != nil {
		return nil, err
	}

	return stepSettings, nil
}

func LoadConfig() (*settings.StepSettings, error) {
	settings_, err := cli.ParseGlazedCommandLayer(nil)
	if err != nil {
		return nil, err
	}

	return LoadConfigFromSettings(*settings_)
}
