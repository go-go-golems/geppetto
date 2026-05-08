package observability

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
)

// Stage identifies the boundary at which a record was captured.
type Stage string

type RecordKind string

const (
	StageProviderRoutedEvent    Stage = "provider_routed_event"
	StageProviderNormalizeDelta Stage = "provider_normalize_delta"

	StageGeppettoPublishStarted Stage = "geppetto_publish_started"
	StageGeppettoPublishDone    Stage = "geppetto_publish_done"
	StageGeppettoPublishError   Stage = "geppetto_publish_error"

	StageProviderCallResultFinalized Stage = "provider_call_result_finalized"
	StageSegmentStarted              Stage = "segment_started"
	StageSegmentUpdated              Stage = "segment_updated"
	StageSegmentFinished             Stage = "segment_finished"
)

const (
	RecordKindProviderEvent      RecordKind = "provider_event"
	RecordKindCanonicalEvent     RecordKind = "canonical_event"
	RecordKindProviderCallResult RecordKind = "provider_call_result"
	RecordKindSegment            RecordKind = "segment"
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

	Stage Stage      `json:"stage"`
	Kind  RecordKind `json:"kind,omitempty"`

	EventType   string `json:"eventType,omitempty"`
	InfoMessage string `json:"infoMessage,omitempty"`

	ResponseID   string `json:"responseId,omitempty"`
	ItemID       string `json:"itemId,omitempty"`
	OutputIndex  *int   `json:"outputIndex,omitempty"`
	SummaryIndex *int   `json:"summaryIndex,omitempty"`

	RunID             string `json:"runId,omitempty"`
	ProviderCallID    string `json:"providerCallId,omitempty"`
	ProviderCallIndex *int   `json:"providerCallIndex,omitempty"`

	ChoiceIndex          *int   `json:"choiceIndex,omitempty"`
	ContentBlockIndex    *int   `json:"contentBlockIndex,omitempty"`
	StreamKind           string `json:"streamKind,omitempty"`
	CorrelationKey       string `json:"correlationKey,omitempty"`
	ParentCorrelationKey string `json:"parentCorrelationKey,omitempty"`
	ToolCallID           string `json:"toolCallId,omitempty"`
	ToolCallIndex        *int   `json:"toolCallIndex,omitempty"`

	SegmentID     string `json:"segmentId,omitempty"`
	SegmentIndex  *int   `json:"segmentIndex,omitempty"`
	SegmentType   string `json:"segmentType,omitempty"`
	SegmentStatus string `json:"segmentStatus,omitempty"`
	TextLen       int    `json:"textLen,omitempty"`

	StopReason   string        `json:"stopReason,omitempty"`
	FinishClass  string        `json:"finishClass,omitempty"`
	Usage        *events.Usage `json:"usage,omitempty"`
	DurationMs   *int64        `json:"durationMs,omitempty"`
	HasToolCalls bool          `json:"hasToolCalls,omitempty"`

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
