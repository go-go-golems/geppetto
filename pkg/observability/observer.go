package observability

import (
	"context"
	"encoding/json"
	"time"
)

// Stage identifies the boundary at which a record was captured.
type Stage string

const (
	StageProviderRoutedEvent    Stage = "provider_routed_event"
	StageProviderNormalizeDelta Stage = "provider_normalize_delta"

	StageGeppettoPublishStarted Stage = "geppetto_publish_started"
	StageGeppettoPublishDone    Stage = "geppetto_publish_done"
	StageGeppettoPublishError   Stage = "geppetto_publish_error"
)

// Record is the neutral evidence object emitted by Geppetto provider engines.
// Stable scalar fields make records queryable; JSON payloads make them
// inspectable when lower-level provider shape or Geppetto enrichment is buggy.
type Record struct {
	Timestamp time.Time `json:"timestamp"`

	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`

	SessionID   string `json:"sessionId,omitempty"`
	InferenceID string `json:"inferenceId,omitempty"`
	TurnID      string `json:"turnId,omitempty"`
	MessageID   string `json:"messageId,omitempty"`

	Stage       Stage  `json:"stage"`
	EventType   string `json:"eventType,omitempty"`
	InfoMessage string `json:"infoMessage,omitempty"`

	ResponseID   string `json:"responseId,omitempty"`
	ItemID       string `json:"itemId,omitempty"`
	OutputIndex  *int   `json:"outputIndex,omitempty"`
	SummaryIndex *int   `json:"summaryIndex,omitempty"`

	ChoiceIndex    *int   `json:"choiceIndex,omitempty"`
	StreamKind     string `json:"streamKind,omitempty"`
	CorrelationKey string `json:"correlationKey,omitempty"`
	ToolCallID     string `json:"toolCallId,omitempty"`
	ToolCallIndex  *int   `json:"toolCallIndex,omitempty"`

	ObjectJSON   json.RawMessage `json:"objectJson,omitempty"`
	EventJSON    json.RawMessage `json:"eventJson,omitempty"`
	MetadataJSON json.RawMessage `json:"metadataJson,omitempty"`

	DeltaLen           int `json:"deltaLen,omitempty"`
	NormalizedDeltaLen int `json:"normalizedDeltaLen,omitempty"`
	BufferLen          int `json:"bufferLen,omitempty"`

	Error string `json:"error,omitempty"`
}

// Observer receives best-effort Geppetto observability records.
type Observer interface {
	OnGeppettoRecord(ctx context.Context, rec Record)
}

// Notify delivers a record to obs without letting observer failures affect
// inference. Panics are intentionally swallowed: observability is evidence, not
// behavior required for inference correctness.
func Notify(ctx context.Context, obs Observer, rec Record) {
	if obs == nil {
		return
	}
	if rec.Timestamp.IsZero() {
		rec.Timestamp = time.Now().UTC()
	}
	defer func() {
		_ = recover()
	}()
	obs.OnGeppettoRecord(ctx, rec)
}
