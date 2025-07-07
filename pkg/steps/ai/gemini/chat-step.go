package gemini

import (
	"context"
	"io"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	ai_types "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	genai "google.golang.org/genai"
)

var _ chat.Step = &ChatStep{}
var _ chat.SimpleChatStep = &ChatStep{}

type ChatStep struct {
	Settings         *settings.StepSettings
	publisherManager *events.PublisherManager
}

type StepOption func(*ChatStep) error

func WithSubscriptionManager(pm *events.PublisherManager) StepOption {
	return func(step *ChatStep) error {
		step.publisherManager = pm
		return nil
	}
}

func NewChatStep(settings *settings.StepSettings, options ...StepOption) (*ChatStep, error) {
	ret := &ChatStep{
		Settings:         settings,
		publisherManager: events.NewPublisherManager(),
	}
	for _, o := range options {
		if err := o(ret); err != nil {
			return nil, err
		}
	}
	return ret, nil
}

func (cs *ChatStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	cs.publisherManager.RegisterPublisher(topic, publisher)
	return nil
}

func roleToGeminiRole(r conversation.Role) genai.Role {
	if r == conversation.RoleAssistant {
		return genai.RoleModel
	}
	return genai.RoleUser
}

func messageToGeminiContent(msg *conversation.Message) *genai.Content {
	switch c := msg.Content.(type) {
	case *conversation.ChatMessageContent:
		parts := []*genai.Part{genai.NewPartFromText(c.Text)}
		for _, img := range c.Images {
			if img.ImageURL != "" {
				parts = append(parts, genai.NewPartFromURI(img.ImageURL, img.MediaType))
			} else if len(img.ImageContent) > 0 {
				parts = append(parts, genai.NewPartFromBytes(img.ImageContent, img.MediaType))
			}
		}
		return genai.NewContentFromParts(parts, roleToGeminiRole(c.Role))
	default:
		return genai.NewContentFromText(msg.Content.String(), genai.RoleUser)
	}
}

func makeContents(msgs conversation.Conversation) []*genai.Content {
	res := make([]*genai.Content, 0, len(msgs))
	for _, m := range msgs {
		res = append(res, messageToGeminiContent(m))
	}
	return res
}

func makeClient(ctx context.Context, api *settings.APISettings) (*genai.Client, error) {
	apiKey, ok := api.APIKeys[string(ai_types.ApiTypeGemini)+"-api-key"]
	if !ok {
		return nil, errors.Errorf("no API key for %s", ai_types.ApiTypeGemini)
	}
	baseURL := api.BaseUrls[string(ai_types.ApiTypeGemini)+"-base-url"]
	return genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:      apiKey,
		Backend:     genai.BackendGeminiAPI,
		HTTPOptions: genai.HTTPOptions{BaseURL: baseURL},
	})
}

func (cs *ChatStep) RunInference(ctx context.Context, messages conversation.Conversation) (*conversation.Message, error) {
	if cs.Settings.Chat.Engine == nil {
		return nil, errors.New("no engine specified")
	}
	client, err := makeClient(ctx, cs.Settings.API)
	if err != nil {
		return nil, err
	}
	contents := makeContents(messages)
	model := *cs.Settings.Chat.Engine

	parentID := conversation.NullNode
	if len(messages) > 0 {
		parentID = messages[len(messages)-1].ID
	}

	metadata := events.EventMetadata{ID: conversation.NewNodeID(), ParentID: parentID}
	stepMeta := &steps.StepMetadata{
		StepID:     uuid.New(),
		Type:       "gemini-chat",
		InputType:  "conversation.Conversation",
		OutputType: "string",
		Metadata: map[string]interface{}{
			steps.MetadataSettingsSlug: cs.Settings.GetMetadata(),
		},
	}

	cs.publisherManager.PublishBlind(events.NewStartEvent(metadata, stepMeta))

	if cs.Settings.Chat.Stream {
		var text string
		for chunk, err := range client.Models.GenerateContentStream(ctx, model, contents, nil) {
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				cs.publisherManager.PublishBlind(events.NewErrorEvent(metadata, stepMeta, err))
				return nil, err
			}
			delta := chunk.Text()
			text += delta
			cs.publisherManager.PublishBlind(events.NewPartialCompletionEvent(metadata, stepMeta, delta, text))
		}
		cs.publisherManager.PublishBlind(events.NewFinalEvent(metadata, stepMeta, text))
		msg := conversation.NewMessage(
			conversation.NewChatMessageContent(conversation.RoleAssistant, text, nil),
			conversation.WithLLMMessageMetadata(&conversation.LLMMessageMetadata{Engine: model}),
		)
		return msg, nil
	}

	resp, err := client.Models.GenerateContent(ctx, model, contents, nil)
	if err != nil {
		cs.publisherManager.PublishBlind(events.NewErrorEvent(metadata, stepMeta, err))
		return nil, err
	}
	text := resp.Text()
	cs.publisherManager.PublishBlind(events.NewFinalEvent(metadata, stepMeta, text))
	meta := &conversation.LLMMessageMetadata{Engine: model}
	if resp.UsageMetadata != nil {
		meta.Usage = &conversation.Usage{
			InputTokens:  int(resp.UsageMetadata.PromptTokenCount),
			OutputTokens: int(resp.UsageMetadata.CandidatesTokenCount),
		}
	}
	msg := conversation.NewMessage(
		conversation.NewChatMessageContent(conversation.RoleAssistant, text, nil),
		conversation.WithLLMMessageMetadata(meta),
	)
	return msg, nil
}

func (cs *ChatStep) Start(ctx context.Context, messages conversation.Conversation) (steps.StepResult[*conversation.Message], error) {
	if !cs.Settings.Chat.Stream {
		msg, err := cs.RunInference(ctx, messages)
		if err != nil {
			return steps.Reject[*conversation.Message](err), nil
		}
		return steps.Resolve(msg), nil
	}

	cancellableCtx, cancel := context.WithCancel(ctx)
	c := make(chan helpers.Result[*conversation.Message])
	ret := steps.NewStepResult[*conversation.Message](c,
		steps.WithCancel[*conversation.Message](cancel),
		steps.WithMetadata[*conversation.Message](&steps.StepMetadata{
			StepID:     uuid.New(),
			Type:       "gemini-chat",
			InputType:  "conversation.Conversation",
			OutputType: "*conversation.Message",
			Metadata: map[string]interface{}{
				steps.MetadataSettingsSlug: cs.Settings.GetMetadata(),
			},
		}))

	go func() {
		defer close(c)
		defer cancel()
		msg, err := cs.RunInference(cancellableCtx, messages)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				c <- helpers.NewErrorResult[*conversation.Message](context.Canceled)
			} else {
				c <- helpers.NewErrorResult[*conversation.Message](err)
			}
			return
		}
		c <- helpers.NewValueResult[*conversation.Message](msg)
	}()

	return ret, nil
}
