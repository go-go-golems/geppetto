package sections

import (
	embeddingsconfig "github.com/go-go-golems/geppetto/pkg/embeddings/config"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/gemini"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
)

// CreateOption configures behavior of CreateGeppettoSections.
type CreateOption func(*createOptions)

type createOptions struct {
	stepSettings *settings.InferenceSettings
}

// WithDefaultsFromInferenceSettings uses the given InferenceSettings for layer defaults.
func WithDefaultsFromInferenceSettings(s *settings.InferenceSettings) CreateOption {
	return func(o *createOptions) {
		o.stepSettings = s
	}
}

// CreateGeppettoSections returns settings sections for Geppetto AI settings.
// If no InferenceSettings are provided via WithInferenceSettings, default settings.NewInferenceSettings() is used.
func CreateGeppettoSections(opts ...CreateOption) ([]schema.Section, error) {
	// Apply options
	var co createOptions
	for _, opt := range opts {
		opt(&co)
	}
	// Determine InferenceSettings
	var ss *settings.InferenceSettings
	if co.stepSettings == nil {
		var err error
		ss, err = settings.NewInferenceSettings()
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

	profileSettingsSection, err := NewProfileSettingsSection()
	if err != nil {
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
		profileSettingsSection,
	}
	return result, nil
}
