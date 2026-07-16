package claude

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/events"
	geppettoobs "github.com/go-go-golems/geppetto/pkg/observability"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
)

// EngineOption configures optional Claude engine behavior.
type EngineOption func(*ClaudeEngine)

// WithObserver attaches a best-effort observability observer to the engine.
func WithObserver(obs geppettoobs.Observer) EngineOption {
	return func(e *ClaudeEngine) {
		e.observer = obs
	}
}

// WithObservabilityConfig controls which Claude records are emitted.
func WithObservabilityConfig(cfg geppettoobs.Config) EngineOption {
	return func(e *ClaudeEngine) {
		e.observabilityConfig = cfg.Normalized()
	}
}

// WithBearerAuthorization enables the verified bearer header form for an
// Anthropic-compatible gateway. It is a Go-only runtime injection point.
func WithBearerAuthorization(value string) EngineOption {
	return func(e *ClaudeEngine) {
		e.bearerAuthorization = value
	}
}

// WithBearerTokenSource resolves an Anthropic-compatible gateway credential at
// request time. The resolved value is used for the verified dual-auth gateway
// form and remains in Go-only runtime state.
func WithBearerTokenSource(source credentials.BearerTokenSource) EngineOption {
	return func(e *ClaudeEngine) {
		e.bearerTokenSource = source
	}
}

func (e *ClaudeEngine) observe(ctx context.Context, rec geppettoobs.Record) {
	if e == nil || !e.observabilityConfig.Enabled() {
		return
	}
	geppettoobs.Notify(ctx, e.observer, rec)
}

func (e *ClaudeEngine) observePublishStarted(ctx context.Context, event events.Event) {
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
	geppettoobs.EnrichRecordFromEvent(&rec, event)
	e.observe(ctx, rec)
	for _, derived := range geppettoobs.DerivedRecordsFromEvent(rec, event) {
		e.observe(ctx, derived)
	}
}

func (e *ClaudeEngine) observeProviderEvent(ctx context.Context, metadata events.EventMetadata, model string, ev api.StreamingEvent) {
	if e == nil || !e.observabilityConfig.RecordsProvider() {
		return
	}
	rec := e.claudeProviderRecordBase(metadata, model, ev)
	rec.Stage = geppettoobs.StageProviderRoutedEvent
	rec.ObjectJSON = mustMarshalJSON(ev)
	e.observe(ctx, rec)
}

func (e *ClaudeEngine) claudeProviderRecordBase(metadata events.EventMetadata, model string, ev api.StreamingEvent) geppettoobs.Record {
	if model == "" && ev.Message != nil {
		model = ev.Message.Model
	}
	rec := geppettoobs.Record{
		Kind:        geppettoobs.RecordKindProviderEvent,
		Provider:    e.inferenceProvider(),
		Model:       model,
		SessionID:   metadata.SessionID,
		InferenceID: metadata.InferenceID,
		TurnID:      metadata.TurnID,
		MessageID:   metadata.ID.String(),
		EventType:   string(ev.Type),
		DeltaLen:    claudeDeltaLen(ev),
	}
	if ev.Message != nil && ev.Message.ID != "" {
		rec.ResponseID = ev.Message.ID
	}
	if ev.ContentBlock != nil && ev.ContentBlock.ID != "" {
		rec.ItemID = ev.ContentBlock.ID
	}
	if ev.Type == api.ContentBlockStartType || ev.Type == api.ContentBlockDeltaType || ev.Type == api.ContentBlockStopType {
		idx := ev.Index
		rec.OutputIndex = &idx
	}
	if ev.Error != nil {
		rec.Error = ev.Error.Message
	}
	return rec
}

func claudeDeltaLen(ev api.StreamingEvent) int {
	if ev.Delta == nil {
		return 0
	}
	return len(ev.Delta.Text) + len(ev.Delta.PartialJSON)
}

func (e *ClaudeEngine) inferenceProvider() string {
	if e != nil && e.settings != nil && e.settings.Chat != nil && e.settings.Chat.ApiType != nil {
		if provider := strings.ToLower(strings.TrimSpace(string(*e.settings.Chat.ApiType))); provider != "" {
			return provider
		}
	}
	return "claude"
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
