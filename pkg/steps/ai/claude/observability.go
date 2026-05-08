package claude

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/events"
	geppettoobs "github.com/go-go-golems/geppetto/pkg/observability"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
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
	applyClaudeCorrelationToRecord(&rec, event)
	e.observe(ctx, rec)
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

func applyClaudeCorrelationToRecord(rec *geppettoobs.Record, event events.Event) {
	if rec == nil || event == nil {
		return
	}
	correlated, ok := event.(events.CorrelatedEvent)
	if !ok {
		return
	}
	corr := correlated.Correlation()
	if corr.Provider != "" {
		rec.Provider = corr.Provider
	}
	if corr.Model != "" {
		rec.Model = corr.Model
	}
	if corr.ResponseID != "" {
		rec.ResponseID = corr.ResponseID
	}
	if corr.ItemID != "" {
		rec.ItemID = corr.ItemID
	}
	if corr.OutputIndex != nil {
		v := int(*corr.OutputIndex)
		rec.OutputIndex = &v
	}
	if corr.SummaryIndex != nil {
		v := int(*corr.SummaryIndex)
		rec.SummaryIndex = &v
	}
	if corr.ChoiceIndex != nil {
		v := int(*corr.ChoiceIndex)
		rec.ChoiceIndex = &v
	}
	if corr.StreamKind != "" {
		rec.StreamKind = corr.StreamKind
	}
	if corr.CorrelationKey != "" {
		rec.CorrelationKey = corr.CorrelationKey
	}
	if corr.ToolCallID != "" {
		rec.ToolCallID = corr.ToolCallID
	}
	if corr.ToolCallIndex != nil {
		v := int(*corr.ToolCallIndex)
		rec.ToolCallIndex = &v
	}
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
