package openai

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/events"
	geppettoobs "github.com/go-go-golems/geppetto/pkg/observability"
)

const defaultChatCompletionEventType = "chat.completion.chunk"

// EngineOption configures optional OpenAI Chat Completions engine behavior.
type EngineOption func(*OpenAIEngine)

// WithObserver attaches a best-effort observability observer to the engine.
func WithObserver(obs geppettoobs.Observer) EngineOption {
	return func(e *OpenAIEngine) {
		e.observer = obs
	}
}

// WithObservabilityConfig controls which OpenAI Chat Completions records are emitted.
func WithObservabilityConfig(cfg geppettoobs.Config) EngineOption {
	return func(e *OpenAIEngine) {
		e.observabilityConfig = cfg.Normalized()
	}
}

func (e *OpenAIEngine) observe(ctx context.Context, rec geppettoobs.Record) {
	if e == nil || !e.observabilityConfig.Enabled() {
		return
	}
	geppettoobs.Notify(ctx, e.observer, rec)
}

func (e *OpenAIEngine) observePublishStarted(ctx context.Context, event events.Event) {
	if e == nil || !e.observabilityConfig.RecordsEvents() || event == nil {
		return
	}
	metadata := event.Metadata()
	rec := geppettoobs.Record{
		Provider:    e.inferenceProvider(),
		Model:       metadata.Model,
		SessionID:   metadata.SessionID,
		InferenceID: metadata.InferenceID,
		TurnID:      metadata.TurnID,
		MessageID:   metadata.ID.String(),
		Stage:       geppettoobs.StageGeppettoPublishStarted,
		EventType:   string(event.Type()),
	}
	if info, ok := event.(*events.EventInfo); ok {
		rec.InfoMessage = info.Message
	}
	e.observe(ctx, rec)
}

func (e *OpenAIEngine) observeProviderEvent(ctx context.Context, metadata events.EventMetadata, model string, ev chatStreamEvent) {
	if e == nil || !e.observabilityConfig.RecordsProvider() {
		return
	}
	rec := e.chatProviderRecordBase(metadata, model, ev)
	rec.Stage = geppettoobs.StageProviderRoutedEvent
	rec.ObjectJSON = mustMarshalJSON(ev.RawPayload)
	e.observe(ctx, rec)
}

func (e *OpenAIEngine) observeProviderNormalizeDelta(ctx context.Context, metadata events.EventMetadata, model string, ev chatStreamEvent, deltaLen, normalizedDeltaLen, bufferLen int) {
	if e == nil || !e.observabilityConfig.RecordsProvider() {
		return
	}
	rec := e.chatProviderRecordBase(metadata, model, ev)
	rec.Stage = geppettoobs.StageProviderNormalizeDelta
	rec.DeltaLen = deltaLen
	rec.NormalizedDeltaLen = normalizedDeltaLen
	rec.BufferLen = bufferLen
	rec.ObjectJSON = mustMarshalJSON(ev.RawPayload)
	e.observe(ctx, rec)
}

func (e *OpenAIEngine) chatProviderRecordBase(metadata events.EventMetadata, model string, ev chatStreamEvent) geppettoobs.Record {
	if model == "" {
		model = stringFromRawMap(ev.RawPayload, "model")
	}
	rec := geppettoobs.Record{
		Provider:    e.inferenceProvider(),
		Model:       model,
		SessionID:   metadata.SessionID,
		InferenceID: metadata.InferenceID,
		TurnID:      metadata.TurnID,
		MessageID:   metadata.ID.String(),
		EventType:   chatProviderEventType(ev),
		ResponseID:  stringFromRawMap(ev.RawPayload, "id"),
		DeltaLen:    len(ev.DeltaText) + len(ev.DeltaReasoning),
	}
	if rec.EventType == "" {
		rec.EventType = defaultChatCompletionEventType
	}
	return rec
}

func chatProviderEventType(ev chatStreamEvent) string {
	if object := stringFromRawMap(ev.RawPayload, "object"); object != "" {
		return object
	}
	return defaultChatCompletionEventType
}

func (e *OpenAIEngine) inferenceProvider() string {
	if e != nil && e.settings != nil && e.settings.Chat != nil && e.settings.Chat.ApiType != nil {
		if provider := strings.ToLower(strings.TrimSpace(string(*e.settings.Chat.ApiType))); provider != "" {
			return provider
		}
	}
	return "openai"
}

func stringFromRawMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func mustMarshalJSON(v any) json.RawMessage {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}
