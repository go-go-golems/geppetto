package events

// Correlation carries the canonical identity needed to join provider objects,
// Geppetto events, Pinocchio backend events, frontend frames, timeline entities,
// and debug SQLite rows. It intentionally contains only explicit IDs.
//
// Provider-native fields (response IDs, item IDs, output indexes, choice indexes,
// model names, provider names, etc.) belong in observability/debug records or in
// provider-specific adapter state. Canonical routing must use these IDs only.
type Correlation struct {
	// Application/runtime scope.
	SessionID string `json:"session_id,omitempty" yaml:"session_id,omitempty" mapstructure:"session_id,omitempty"`
	RunID     string `json:"run_id,omitempty" yaml:"run_id,omitempty" mapstructure:"run_id,omitempty"`
	TurnID    string `json:"turn_id,omitempty" yaml:"turn_id,omitempty" mapstructure:"turn_id,omitempty"`

	// Canonical lifecycle identities.
	ProviderCallID string `json:"provider_call_id,omitempty" yaml:"provider_call_id,omitempty" mapstructure:"provider_call_id,omitempty"`
	SegmentID      string `json:"segment_id,omitempty" yaml:"segment_id,omitempty" mapstructure:"segment_id,omitempty"`
	ToolCallID     string `json:"tool_call_id,omitempty" yaml:"tool_call_id,omitempty" mapstructure:"tool_call_id,omitempty"`
}

// CorrelatedEvent is implemented by canonical events that carry typed
// correlation. Legacy events only expose EventMetadata.
type CorrelatedEvent interface {
	Event
	Correlation() Correlation
}
