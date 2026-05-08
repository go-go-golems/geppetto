package gemini

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
	geppettoobs "github.com/go-go-golems/geppetto/pkg/observability"
)

// WithObserver attaches a best-effort observability observer to the engine.
func WithObserver(obs geppettoobs.Observer) EngineOption {
	return func(e *GeminiEngine) {
		e.observer = obs
	}
}

// WithObservabilityConfig controls how much Geppetto/Gemini evidence is emitted.
func WithObservabilityConfig(cfg geppettoobs.Config) EngineOption {
	return func(e *GeminiEngine) {
		e.observabilityConfig = cfg.Normalized()
	}
}

func (e *GeminiEngine) notifyObserver(ctx context.Context, rec geppettoobs.Record) {
	if e == nil || !e.observabilityConfig.Enabled() {
		return
	}
	geppettoobs.Notify(ctx, e.observer, rec)
}

func (e *GeminiEngine) publishEventRecord(ctx context.Context, event events.Event) {
	if e == nil || !e.observabilityConfig.RecordsEvents() || event == nil {
		return
	}
	metadata := event.Metadata()
	rec := geppettoobs.Record{
		Timestamp:   time.Now().UTC(),
		Stage:       geppettoobs.StageGeppettoPublishDone,
		Kind:        geppettoobs.RecordKindCanonicalEvent,
		Provider:    "gemini",
		Model:       metadata.Model,
		SessionID:   metadata.SessionID,
		InferenceID: metadata.InferenceID,
		TurnID:      metadata.TurnID,
		MessageID:   metadata.ID.String(),
		EventType:   string(event.Type()),
	}
	if body, err := json.Marshal(event); err == nil {
		rec.EventJSON = body
	}
	if body, err := json.Marshal(metadata); err == nil {
		rec.MetadataJSON = body
	}
	geppettoobs.EnrichRecordFromEvent(&rec, event)
	e.notifyObserver(ctx, rec)
	for _, derived := range geppettoobs.DerivedRecordsFromEvent(rec, event) {
		e.notifyObserver(ctx, derived)
	}
}

func (e *GeminiEngine) publishProviderRecord(ctx context.Context, metadata events.EventMetadata, corr events.Correlation, eventType string, object any) {
	if e == nil || !e.observabilityConfig.RecordsProvider() {
		return
	}
	rec := geppettoobs.Record{
		Timestamp:            time.Now().UTC(),
		Stage:                geppettoobs.StageProviderRoutedEvent,
		Kind:                 geppettoobs.RecordKindProviderEvent,
		Provider:             "gemini",
		Model:                corr.Model,
		SessionID:            firstNonEmptyString(corr.SessionID, metadata.SessionID),
		RunID:                corr.RunID,
		InferenceID:          firstNonEmptyString(corr.InferenceID, metadata.InferenceID),
		TurnID:               firstNonEmptyString(corr.TurnID, metadata.TurnID),
		MessageID:            metadata.ID.String(),
		EventType:            eventType,
		ProviderCallID:       corr.ProviderCallID,
		ProviderCallIndex:    intPtrFromInt32NonZero(corr.ProviderCallIndex),
		ResponseID:           corr.ResponseID,
		StreamKind:           corr.StreamKind,
		CorrelationKey:       corr.CorrelationKey,
		ParentCorrelationKey: corr.ParentCorrelationKey,
	}
	if body, err := json.Marshal(object); err == nil {
		rec.ObjectJSON = body
	}
	e.notifyObserver(ctx, rec)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func intPtrFromInt32NonZero(v int32) *int {
	if v == 0 {
		return nil
	}
	out := int(v)
	return &out
}
