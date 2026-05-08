package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
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
	applyChatProviderDataToRecord(&rec, metadata.Extra)
	if info, ok := event.(*events.EventInfo); ok {
		rec.InfoMessage = info.Message
		applyChatProviderDataToRecord(&rec, info.Data)
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
	provider := e.inferenceProvider()
	responseID := stringFromRawMap(ev.RawPayload, "id")
	streamKind := chatStreamKind(ev)
	tc := firstChatToolCall(ev.ToolCalls)
	var toolCallID string
	var toolCallIndex *int
	if tc != nil {
		toolCallID = tc.ID
		toolCallIndex = cloneIntPtr(tc.Index)
	}
	rec := geppettoobs.Record{
		Provider:       provider,
		Model:          model,
		SessionID:      metadata.SessionID,
		InferenceID:    metadata.InferenceID,
		TurnID:         metadata.TurnID,
		MessageID:      metadata.ID.String(),
		EventType:      chatProviderEventType(ev),
		ResponseID:     responseID,
		ChoiceIndex:    cloneIntPtr(ev.ChoiceIndex),
		StreamKind:     streamKind,
		CorrelationKey: chatCorrelationKey(provider, responseID, ev.ChoiceIndex, streamKind, toolCallID, toolCallIndex),
		ToolCallID:     toolCallID,
		ToolCallIndex:  toolCallIndex,
		DeltaLen:       len(ev.DeltaText) + len(ev.DeltaReasoning),
	}
	if rec.EventType == "" {
		rec.EventType = defaultChatCompletionEventType
	}
	return rec
}

type chatToolCallIDTracker map[string]string

func (t chatToolCallIDTracker) Enrich(ev chatStreamEvent) chatStreamEvent {
	if len(ev.ToolCalls) == 0 {
		return ev
	}
	if t == nil {
		return ev
	}
	enriched := ev
	enriched.ToolCalls = make([]ChatToolCall, len(ev.ToolCalls))
	copy(enriched.ToolCalls, ev.ToolCalls)
	for i := range enriched.ToolCalls {
		call := &enriched.ToolCalls[i]
		if call.Index == nil {
			continue
		}
		key := chatToolCallTrackerKey(ev.ChoiceIndex, call.Index)
		if call.ID != "" {
			t[key] = call.ID
			continue
		}
		if rememberedID := t[key]; rememberedID != "" {
			call.ID = rememberedID
		}
	}
	return enriched
}

func chatToolCallTrackerKey(choiceIndex, toolCallIndex *int) string {
	choice := 0
	if choiceIndex != nil {
		choice = *choiceIndex
	}
	return fmt.Sprintf("%d:%d", choice, *toolCallIndex)
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

func chatStreamKind(ev chatStreamEvent) string {
	switch {
	case ev.DeltaReasoning != "":
		return "reasoning"
	case ev.DeltaText != "":
		return "content"
	case len(ev.ToolCalls) > 0:
		return "tool_call"
	case ev.FinishReason != nil && *ev.FinishReason != "":
		return "finish"
	default:
		return "unknown"
	}
}

func firstChatToolCall(calls []ChatToolCall) *ChatToolCall {
	if len(calls) == 0 {
		return nil
	}
	return &calls[0]
}

func chatProviderData(provider, responseID string, choiceIndex *int, streamKind, correlationKey, toolCallID string, toolCallIndex *int) map[string]any {
	data := map[string]any{}
	if provider != "" {
		data["provider"] = provider
	}
	if responseID != "" {
		data["response_id"] = responseID
	}
	if choiceIndex != nil {
		data["choice_index"] = *choiceIndex
	}
	if streamKind != "" {
		data["stream_kind"] = streamKind
	}
	if correlationKey != "" {
		data["correlation_key"] = correlationKey
	}
	if toolCallID != "" {
		data["tool_call_id"] = toolCallID
	}
	if toolCallIndex != nil {
		data["tool_call_index"] = *toolCallIndex
	}
	if len(data) == 0 {
		return nil
	}
	return data
}

func chatProviderDataFromEvent(provider string, ev chatStreamEvent) map[string]any {
	responseID := stringFromRawMap(ev.RawPayload, "id")
	streamKind := chatStreamKind(ev)
	tc := firstChatToolCall(ev.ToolCalls)
	var toolCallID string
	var toolCallIndex *int
	if tc != nil {
		toolCallID = tc.ID
		toolCallIndex = cloneIntPtr(tc.Index)
	}
	return chatProviderData(provider, responseID, ev.ChoiceIndex, streamKind, chatCorrelationKey(provider, responseID, ev.ChoiceIndex, streamKind, toolCallID, toolCallIndex), toolCallID, toolCallIndex)
}

func chatCorrelationKey(provider, responseID string, choiceIndex *int, streamKind, toolCallID string, toolCallIndex *int) string {
	return events.BuildChatCompletionsCorrelation(provider, responseID, choiceIndex, streamKind, toolCallID, toolCallIndex).CorrelationKey
}

func metadataWithChatProviderData(metadata events.EventMetadata, data map[string]any) events.EventMetadata {
	if len(data) == 0 {
		return metadata
	}
	if metadata.Extra == nil {
		metadata.Extra = map[string]any{}
	} else {
		copyExtra := make(map[string]any, len(metadata.Extra)+len(data))
		for k, v := range metadata.Extra {
			copyExtra[k] = v
		}
		metadata.Extra = copyExtra
	}
	for k, v := range data {
		metadata.Extra[k] = v
	}
	return metadata
}

func applyChatProviderDataToRecord(rec *geppettoobs.Record, data map[string]interface{}) {
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
	if v, ok := intFromAny(data["choice_index"]); ok {
		rec.ChoiceIndex = &v
	}
	if v, ok := data["stream_kind"].(string); ok && v != "" {
		rec.StreamKind = v
	}
	if v, ok := data["correlation_key"].(string); ok && v != "" {
		rec.CorrelationKey = v
	}
	if v, ok := data["tool_call_id"].(string); ok && v != "" {
		rec.ToolCallID = v
	}
	if v, ok := intFromAny(data["tool_call_index"]); ok {
		rec.ToolCallIndex = &v
	}
}

func intFromAny(v any) (int, bool) {
	switch tv := v.(type) {
	case int:
		return tv, true
	case int32:
		return intFromSigned64(int64(tv))
	case int64:
		return intFromSigned64(tv)
	case uint:
		return intFromUnsigned64(uint64(tv))
	case uint32:
		return intFromUnsigned64(uint64(tv))
	case uint64:
		return intFromUnsigned64(tv)
	case float64:
		return intFromFloat64(tv)
	case string:
		return intFromString(tv)
	default:
		return 0, false
	}
}

func intFromString(v string) (int, bool) {
	i, err := strconv.Atoi(strings.TrimSpace(v))
	return i, err == nil
}

func intFromSigned64(v int64) (int, bool) {
	i, err := strconv.Atoi(strconv.FormatInt(v, 10))
	return i, err == nil
}

func intFromUnsigned64(v uint64) (int, bool) {
	i, err := strconv.Atoi(strconv.FormatUint(v, 10))
	return i, err == nil
}

func intFromFloat64(v float64) (int, bool) {
	if math.IsNaN(v) || math.IsInf(v, 0) || math.Trunc(v) != v {
		return 0, false
	}
	i, err := strconv.Atoi(strconv.FormatFloat(v, 'f', 0, 64))
	return i, err == nil
}

func cloneIntPtr(v *int) *int {
	if v == nil {
		return nil
	}
	vv := *v
	return &vv
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
