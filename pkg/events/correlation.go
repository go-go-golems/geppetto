package events

// Correlation carries stable identity for joining provider objects, Geppetto
// events, Pinocchio backend events, frontend frames, timeline entities, and
// debug SQLite rows. New canonical events should expose this struct directly
// instead of requiring consumers to inspect EventMetadata.Extra.
type Correlation struct {
	// Application/runtime scope.
	SessionID   string `json:"session_id,omitempty" yaml:"session_id,omitempty" mapstructure:"session_id,omitempty"`
	RunID       string `json:"run_id,omitempty" yaml:"run_id,omitempty" mapstructure:"run_id,omitempty"`
	InferenceID string `json:"inference_id,omitempty" yaml:"inference_id,omitempty" mapstructure:"inference_id,omitempty"`
	TurnID      string `json:"turn_id,omitempty" yaml:"turn_id,omitempty" mapstructure:"turn_id,omitempty"`

	// Provider-call scope.
	ProviderCallID    string `json:"provider_call_id,omitempty" yaml:"provider_call_id,omitempty" mapstructure:"provider_call_id,omitempty"`
	ProviderCallIndex int32  `json:"provider_call_index,omitempty" yaml:"provider_call_index,omitempty" mapstructure:"provider_call_index,omitempty"`
	Provider          string `json:"provider,omitempty" yaml:"provider,omitempty" mapstructure:"provider,omitempty"`
	Model             string `json:"model,omitempty" yaml:"model,omitempty" mapstructure:"model,omitempty"`
	ResponseID        string `json:"response_id,omitempty" yaml:"response_id,omitempty" mapstructure:"response_id,omitempty"`

	// Provider item/block scope.
	ItemID            string `json:"item_id,omitempty" yaml:"item_id,omitempty" mapstructure:"item_id,omitempty"`
	OutputIndex       *int32 `json:"output_index,omitempty" yaml:"output_index,omitempty" mapstructure:"output_index,omitempty"`
	SummaryIndex      *int32 `json:"summary_index,omitempty" yaml:"summary_index,omitempty" mapstructure:"summary_index,omitempty"`
	ChoiceIndex       *int32 `json:"choice_index,omitempty" yaml:"choice_index,omitempty" mapstructure:"choice_index,omitempty"`
	ContentBlockIndex *int32 `json:"content_block_index,omitempty" yaml:"content_block_index,omitempty" mapstructure:"content_block_index,omitempty"`

	// Transcript segment scope.
	SegmentID    string `json:"segment_id,omitempty" yaml:"segment_id,omitempty" mapstructure:"segment_id,omitempty"`
	SegmentIndex int32  `json:"segment_index,omitempty" yaml:"segment_index,omitempty" mapstructure:"segment_index,omitempty"`
	SegmentType  string `json:"segment_type,omitempty" yaml:"segment_type,omitempty" mapstructure:"segment_type,omitempty"`
	StreamKind   string `json:"stream_kind,omitempty" yaml:"stream_kind,omitempty" mapstructure:"stream_kind,omitempty"`

	// Tool scope.
	ToolCallID    string `json:"tool_call_id,omitempty" yaml:"tool_call_id,omitempty" mapstructure:"tool_call_id,omitempty"`
	ToolCallIndex *int32 `json:"tool_call_index,omitempty" yaml:"tool_call_index,omitempty" mapstructure:"tool_call_index,omitempty"`

	// Normalized join keys.
	CorrelationKey       string `json:"correlation_key,omitempty" yaml:"correlation_key,omitempty" mapstructure:"correlation_key,omitempty"`
	ParentCorrelationKey string `json:"parent_correlation_key,omitempty" yaml:"parent_correlation_key,omitempty" mapstructure:"parent_correlation_key,omitempty"`
}

// CorrelatedEvent is implemented by canonical events that carry typed
// correlation. Legacy events only expose EventMetadata.
type CorrelatedEvent interface {
	Event
	Correlation() Correlation
}
