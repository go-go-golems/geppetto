package layers

import (
	embeddingsconfig "github.com/go-go-golems/geppetto/pkg/embeddings/config"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/gemini"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	cmdlayers "github.com/go-go-golems/glazed/pkg/cmds/layers"
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
