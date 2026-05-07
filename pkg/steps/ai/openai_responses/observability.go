package openai_responses

import (
	"context"
	"strconv"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/events"
	geppettoobs "github.com/go-go-golems/geppetto/pkg/observability"
)

type EngineOption func(*Engine)

func WithObserver(obs geppettoobs.Observer) EngineOption {
	return func(e *Engine) {
		e.observer = obs
	}
}

func WithObservabilityConfig(cfg geppettoobs.Config) EngineOption {
	return func(e *Engine) {
		e.observabilityConfig = cfg.Normalized()
	}
}

func (e *Engine) observe(ctx context.Context, rec geppettoobs.Record) {
	if e == nil || !e.observabilityConfig.Enabled() {
		return
	}
	geppettoobs.Notify(ctx, e.observer, rec)
}

func (e *Engine) observeProviderEvent(ctx context.Context, metadata events.EventMetadata, model string, currentResponseID string, eventType string, m map[string]any) {
	if e == nil || !e.observabilityConfig.RecordsProvider() {
		return
	}
	rec := providerRecordBase(metadata, model, currentResponseID, eventType, m)
	rec.Stage = geppettoobs.StageProviderRoutedEvent
	rec.ObjectJSON = mustMarshalJSON(m)
	e.observe(ctx, rec)
}

func (e *Engine) observeProviderNormalizeDelta(ctx context.Context, metadata events.EventMetadata, model string, currentResponseID string, eventType string, m map[string]any, deltaLen, normalizedDeltaLen, bufferLen int) {
	if e == nil || !e.observabilityConfig.RecordsProvider() {
		return
	}
	rec := providerRecordBase(metadata, model, currentResponseID, eventType, m)
	rec.Stage = geppettoobs.StageProviderNormalizeDelta
	rec.DeltaLen = deltaLen
	rec.NormalizedDeltaLen = normalizedDeltaLen
	rec.BufferLen = bufferLen
	rec.ObjectJSON = mustMarshalJSON(m)
	e.observe(ctx, rec)
}

func (e *Engine) observePublish(ctx context.Context, event events.Event, stage geppettoobs.Stage, err error) {
	if e == nil || !e.observabilityConfig.RecordsEvents() || event == nil {
		return
	}
	metadata := event.Metadata()
	rec := geppettoobs.Record{
		Provider:    "openai_responses",
		Model:       metadata.Model,
		SessionID:   metadata.SessionID,
		InferenceID: metadata.InferenceID,
		TurnID:      metadata.TurnID,
		MessageID:   metadata.ID.String(),
		Stage:       stage,
		EventType:   string(event.Type()),
	}
	if stage == geppettoobs.StageGeppettoPublishDone || stage == geppettoobs.StageGeppettoPublishError {
		rec.EventJSON = mustMarshalJSON(event)
		rec.MetadataJSON = mustMarshalJSON(metadata)
	}
	if err != nil {
		rec.Error = err.Error()
	}
	if info, ok := event.(*events.EventInfo); ok {
		rec.InfoMessage = info.Message
		applyProviderDataToRecord(&rec, info.Data)
	}
	e.observe(ctx, rec)
}

func providerRecordBase(metadata events.EventMetadata, model string, currentResponseID string, eventType string, m map[string]any) geppettoobs.Record {
	rec := geppettoobs.Record{
		Provider:    "openai_responses",
		Model:       model,
		SessionID:   metadata.SessionID,
		InferenceID: metadata.InferenceID,
		TurnID:      metadata.TurnID,
		MessageID:   metadata.ID.String(),
		EventType:   eventType,
		ResponseID:  currentResponseID,
	}
	if v := stringFromProviderMap(m, "response_id"); v != "" {
		rec.ResponseID = v
	} else if response, ok := m["response"].(map[string]any); ok {
		if v := stringFromProviderMap(response, "id"); v != "" {
			rec.ResponseID = v
		}
	}
	if v := itemIDFromProviderObject(m); v != "" {
		rec.ItemID = v
	}
	if idx, ok := intFromProviderNumber(m["output_index"]); ok {
		rec.OutputIndex = &idx
	}
	if idx, ok := intFromProviderNumber(m["summary_index"]); ok {
		rec.SummaryIndex = &idx
	}
	return rec
}

func stringFromProviderMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func itemIDFromProviderObject(m map[string]any) string {
	if v := stringFromProviderMap(m, "item_id"); v != "" {
		return v
	}
	if item, ok := m["item"].(map[string]any); ok {
		if v := stringFromProviderMap(item, "id"); v != "" {
			return v
		}
	}
	return ""
}

func providerData(provider, responseID, itemID string, outputIndex, summaryIndex *int) map[string]any {
	data := map[string]any{}
	if provider != "" {
		data["provider"] = provider
	}
	if responseID != "" {
		data["response_id"] = responseID
	}
	if itemID != "" {
		data["item_id"] = itemID
	}
	if outputIndex != nil {
		data["output_index"] = *outputIndex
	}
	if summaryIndex != nil {
		data["summary_index"] = *summaryIndex
	}
	if len(data) == 0 {
		return nil
	}
	return data
}

func applyProviderDataToRecord(rec *geppettoobs.Record, data map[string]interface{}) {
	if rec == nil || data == nil {
		return
	}
	if v, ok := data["provider"].(string); ok && v != "" {
		rec.Provider = v
	}
	if v, ok := data["response_id"].(string); ok && v != "" {
		rec.ResponseID = v
	}
	if v, ok := data["item_id"].(string); ok && v != "" {
		rec.ItemID = v
	}
	if v, ok := intFromAny(data["output_index"]); ok {
		rec.OutputIndex = &v
	}
	if v, ok := intFromAny(data["summary_index"]); ok {
		rec.SummaryIndex = &v
	}
}

func intFromAny(v any) (int, bool) {
	if i, ok := intFromProviderNumber(v); ok {
		return i, true
	}
	switch tv := v.(type) {
	case int32:
		return int(tv), true
	case uint:
		return int(tv), true
	case uint32:
		return int(tv), true
	case uint64:
		return int(tv), true
	case string:
		i64, err := strconv.ParseInt(strings.TrimSpace(tv), 10, 0)
		if err == nil {
			return int(i64), true
		}
	}
	return 0, false
}
