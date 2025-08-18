package gemini

import (
	"context"
	"io"
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
	genai "github.com/google/generative-ai-go/genai"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// GeminiEngine implements the Engine interface for Google's Gemini API
type GeminiEngine struct {
	settings *settings.StepSettings
	config   *engine.Config
}

// NewGeminiEngine creates a new Gemini inference engine with the given settings and options.
func NewGeminiEngine(settings *settings.StepSettings, options ...engine.Option) (*GeminiEngine, error) {
	cfg := engine.NewConfig()
	if err := engine.ApplyOptions(cfg, options...); err != nil {
		return nil, err
	}
	return &GeminiEngine{settings: settings, config: cfg}, nil
}

// RunInference processes a Turn using the Gemini API and appends result blocks.
func (e *GeminiEngine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	if e.settings == nil || e.settings.Chat == nil || e.settings.Chat.Engine == nil {
		return nil, errors.New("no engine specified")
	}

	apiType_ := e.settings.Chat.ApiType
	if apiType_ == nil {
		return nil, errors.New("no chat api type specified")
	}

	// Build client
	apiKey, ok := e.settings.API.APIKeys[string(*apiType_)+"-api-key"]
	if !ok || apiKey == "" {
		return nil, errors.Errorf("missing API key %s", string(*apiType_)+"-api-key")
	}
	baseURL := e.settings.API.BaseUrls[string(*apiType_)+"-base-url"]

	var client *genai.Client
	var err error
	if baseURL != "" {
		client, err = genai.NewClient(ctx, option.WithAPIKey(apiKey), option.WithEndpoint(baseURL))
	} else {
		client, err = genai.NewClient(ctx, option.WithAPIKey(apiKey))
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to create gemini client")
	}
	defer client.Close()

	modelName := *e.settings.Chat.Engine
	model := client.GenerativeModel(modelName)

	// Configure generation if present
	if e.settings.Chat.Temperature != nil || e.settings.Chat.TopP != nil || e.settings.Chat.MaxResponseTokens != nil {
		cfg := genai.GenerationConfig{}
		if e.settings.Chat.Temperature != nil {
			v := float32(*e.settings.Chat.Temperature)
			cfg.Temperature = &v
		}
		if e.settings.Chat.TopP != nil {
			v := float32(*e.settings.Chat.TopP)
			cfg.TopP = &v
		}
		if e.settings.Chat.MaxResponseTokens != nil {
			v := int32(*e.settings.Chat.MaxResponseTokens)
			cfg.MaxOutputTokens = &v
		}
		model.GenerationConfig = cfg
	}

	// Build parts from Turn blocks
	parts := e.buildPartsFromTurn(t)

	// Prepare metadata for events
	startTime := time.Now()
	metadata := events.EventMetadata{
		ID: uuid.New(),
		LLMInferenceData: events.LLMInferenceData{
			Model:       modelName,
			Usage:       nil,
			StopReason:  nil,
			Temperature: e.settings.Chat.Temperature,
			TopP:        e.settings.Chat.TopP,
			MaxTokens:   e.settings.Chat.MaxResponseTokens,
		},
	}
	if t != nil {
		metadata.RunID = t.RunID
		metadata.TurnID = t.ID
	}
	if metadata.Extra == nil {
		metadata.Extra = map[string]interface{}{}
	}
	metadata.Extra[events.MetadataSettingsSlug] = e.settings.GetMetadata()

	// Publish start event
	e.publishEvent(ctx, events.NewStartEvent(metadata))

	// Streaming mode always on for engines in this architecture
	log.Debug().Int("num_blocks", len(t.Blocks)).Str("model", modelName).Msg("Gemini RunInference started (streaming)")
	iter := model.GenerateContentStream(ctx, parts...)

	message := ""
	chunkCount := 0
	for {
		resp, err := iter.Next()
		if err == iterator.Done || errors.Is(err, io.EOF) {
			log.Debug().Int("chunks_received", chunkCount).Msg("Gemini stream completed")
			break
		}
		if err != nil {
			log.Error().Err(err).Int("chunks_received", chunkCount).Msg("Gemini stream receive failed")
			d := time.Since(startTime).Milliseconds()
			dm := int64(d)
			metadata.DurationMs = &dm
			e.publishEvent(ctx, events.NewErrorEvent(metadata, err))
			return nil, err
		}
		chunkCount++
		// Extract text from response parts
		delta := extractText(resp)
		if delta != "" {
			message += delta
			e.publishEvent(ctx, events.NewPartialCompletionEvent(metadata, delta, message))
		}
	}

	// Set duration and publish final
	d := time.Since(startTime).Milliseconds()
	dm := int64(d)
	metadata.DurationMs = &dm
	e.publishEvent(ctx, events.NewFinalEvent(metadata, message))

	if message != "" {
		turns.AppendBlock(t, turns.NewAssistantTextBlock(message))
	}
	log.Debug().Int("final_text_len", len(message)).Msg("Gemini RunInference completed (streaming)")
	return t, nil
}

// buildPartsFromTurn converts Turn blocks into a flat slice of genai.Part.
func (e *GeminiEngine) buildPartsFromTurn(t *turns.Turn) []genai.Part {
	if t == nil || len(t.Blocks) == 0 {
		return []genai.Part{}
	}
	var parts []genai.Part
	for _, b := range t.Blocks {
		if txt, ok := b.Payload[turns.PayloadKeyText]; ok && txt != nil {
			switch sv := txt.(type) {
			case string:
				parts = append(parts, genai.Text(sv))
			case []byte:
				parts = append(parts, genai.Text(string(sv)))
			default:
				// ignore non-text payloads for now
			}
		}
	}
	return parts
}

// extractText concatenates all text parts from a streaming response.
func extractText(resp *genai.GenerateContentResponse) string {
	if resp == nil || len(resp.Candidates) == 0 {
		return ""
	}
	acc := ""
	for _, cand := range resp.Candidates {
		if cand.Content == nil {
			continue
		}
		for _, p := range cand.Content.Parts {
			if t, ok := p.(genai.Text); ok {
				acc += string(t)
			}
		}
	}
	return acc
}

// publishEvent publishes an event to all configured sinks and any sinks carried in context.
func (e *GeminiEngine) publishEvent(ctx context.Context, event events.Event) {
	for _, sink := range e.config.EventSinks {
		if err := sink.PublishEvent(event); err != nil {
			log.Warn().Err(err).Str("event_type", string(event.Type())).Msg("Failed to publish event to sink")
		}
	}
	events.PublishEventToContext(ctx, event)
}

var _ engine.Engine = (*GeminiEngine)(nil)
