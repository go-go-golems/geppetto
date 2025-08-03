package adapter

import (
	"context"

	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/inference"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/gemini"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	ai_types "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// EngineStepFactory creates Steps by wrapping inference engines in adapters.
// This provides backwards compatibility while using the new engine architecture.
type EngineStepFactory struct {
	Settings *settings.StepSettings
}

// NewStep creates a new Step by creating an inference engine and wrapping it in a StepAdapter.
// This preserves all existing step creation patterns while using the new engine system internally.
func (f *EngineStepFactory) NewStep(
	options ...chat.StepOption,
) (chat.Step, error) {
	settings_ := f.Settings.Clone()

	if settings_.Chat == nil || settings_.Chat.Engine == nil {
		return nil, errors.New("no chat engine specified")
	}

	// For now, we'll create the existing step implementations and wrap them
	// This allows us to preserve all the complex streaming logic while
	// gradually transitioning to the engine architecture

	var chatStep chat.SimpleChatStep
	var err error
	var stepType string

	if settings_.Chat.ApiType != nil {
		log.Debug().Interface("api_type", settings_.Chat.ApiType).Msg("creating chat based on api type")
		switch *settings_.Chat.ApiType {
		case ai_types.ApiTypeOpenAI, ai_types.ApiTypeAnyScale, ai_types.ApiTypeFireworks:
			chatStep, err = openai.NewStep(settings_)
			if err != nil {
				return nil, err
			}
			stepType = "openai-chat"

		case ai_types.ApiTypeClaude:
			chatStep, err = claude.NewChatStep(settings_, []api.Tool{})
			if err != nil {
				return nil, err
			}
			stepType = "claude-chat"

		case ai_types.ApiTypeGemini:
			chatStep, err = gemini.NewChatStep(settings_)
			if err != nil {
				return nil, err
			}
			stepType = "gemini-chat"

		case ai_types.ApiTypeOllama:
			return nil, errors.New("ollama is not supported")

		case ai_types.ApiTypeMistral:
			return nil, errors.New("mistral is not supported")

		case ai_types.ApiTypePerplexity:
			return nil, errors.New("perplexity is not supported")

		case ai_types.ApiTypeCohere:
			return nil, errors.New("cohere is not supported")
		}

	} else {
		switch {
		case openai.IsOpenAiEngine(*settings_.Chat.Engine):
			apiType := ai_types.ApiTypeOpenAI
			settings_.Chat.ApiType = &apiType
			chatStep, err = openai.NewStep(settings_)
			if err != nil {
				return nil, err
			}
			stepType = "openai-chat"

		case claude.IsClaudeEngine(*settings_.Chat.Engine):
			apiType := ai_types.ApiTypeClaude
			settings_.Chat.ApiType = &apiType
			chatStep, err = claude.NewChatStep(settings_, []api.Tool{})
			if err != nil {
				return nil, err
			}
			stepType = "claude-chat"

		case gemini.IsGeminiEngine(*settings_.Chat.Engine):
			apiType := ai_types.ApiTypeGemini
			settings_.Chat.ApiType = &apiType
			chatStep, err = gemini.NewChatStep(settings_)
			if err != nil {
				return nil, err
			}
			stepType = "gemini-chat"

		default:
			return nil, errors.New("unsupported engine type")
		}
	}

	if chatStep == nil {
		return nil, errors.New("failed to create chat step")
	}

	// Create metadata for the adapter
	metadata := &steps.StepMetadata{
		StepID:     uuid.New(),
		Type:       stepType,
		InputType:  "conversation.Conversation",
		OutputType: "*conversation.Message",
		Metadata: map[string]interface{}{
			steps.MetadataSettingsSlug: settings_.GetMetadata(),
		},
	}

	// Create the adapter
	adapter := NewStepAdapter(chatStep, metadata)

	// Apply step options to the adapter
	for _, option := range options {
		err := option(adapter)
		if err != nil {
			return nil, err
		}
	}

	// Wrap with caching if configured
	var ret chat.Step = adapter
	if settings_.Chat != nil {
		ret, err = settings_.Chat.WrapWithCache(ret, options...)
		if err != nil {
			return nil, errors.Wrap(err, "failed to wrap step with cache")
		}
	}

	return ret, nil
}

// CreateEngineFromStep creates an inference engine from an existing step implementation.
// This is used when we want to expose the engine interface from existing steps.
func CreateEngineFromStep(step chat.SimpleChatStep) inference.Engine {
	return &stepEngineAdapter{step: step}
}

// stepEngineAdapter adapts a SimpleChatStep to implement the inference.Engine interface
type stepEngineAdapter struct {
	step chat.SimpleChatStep
}

func (a *stepEngineAdapter) RunInference(ctx context.Context, messages conversation.Conversation) (*conversation.Message, error) {
	return a.step.RunInference(ctx, messages)
}
