package events

type EventToolCallStarted struct {
	EventImpl
	Correlation_ Correlation `json:"correlation"`
	ToolCallID   string      `json:"tool_call_id"`
	ToolName     string      `json:"tool_name,omitempty"`
}

func NewToolCallStartedEvent(metadata EventMetadata, corr Correlation, toolCallID, toolName string) *EventToolCallStarted {
	return &EventToolCallStarted{EventImpl: EventImpl{Type_: EventTypeToolCallStarted, Metadata_: metadata}, Correlation_: corr, ToolCallID: toolCallID, ToolName: toolName}
}

func (e *EventToolCallStarted) Correlation() Correlation { return e.Correlation_ }

var _ CorrelatedEvent = &EventToolCallStarted{}

type EventToolCallArgumentsDelta struct {
	EventImpl
	Correlation_ Correlation `json:"correlation"`
	ToolCallID   string      `json:"tool_call_id"`
	Delta        string      `json:"delta"`
	Arguments    string      `json:"arguments"`
	Sequence     int64       `json:"sequence,omitempty"`
}

func NewToolCallArgumentsDeltaEvent(metadata EventMetadata, corr Correlation, toolCallID, delta, arguments string, sequence int64) *EventToolCallArgumentsDelta {
	return &EventToolCallArgumentsDelta{EventImpl: EventImpl{Type_: EventTypeToolCallArgumentsDelta, Metadata_: metadata}, Correlation_: corr, ToolCallID: toolCallID, Delta: delta, Arguments: arguments, Sequence: sequence}
}

func (e *EventToolCallArgumentsDelta) Correlation() Correlation { return e.Correlation_ }

var _ CorrelatedEvent = &EventToolCallArgumentsDelta{}

type EventToolCallRequested struct {
	EventImpl
	Correlation_ Correlation `json:"correlation"`
	ToolCallID   string      `json:"tool_call_id"`
	ToolName     string      `json:"tool_name"`
	Input        string      `json:"input"`
}

func NewToolCallRequestedEvent(metadata EventMetadata, corr Correlation, toolCallID, toolName, input string) *EventToolCallRequested {
	return &EventToolCallRequested{EventImpl: EventImpl{Type_: EventTypeToolCallRequested, Metadata_: metadata}, Correlation_: corr, ToolCallID: toolCallID, ToolName: toolName, Input: input}
}

func (e *EventToolCallRequested) Correlation() Correlation { return e.Correlation_ }

var _ CorrelatedEvent = &EventToolCallRequested{}

type EventToolExecutionStarted struct {
	EventImpl
	Correlation_ Correlation `json:"correlation"`
	ToolCallID   string      `json:"tool_call_id"`
	ToolName     string      `json:"tool_name,omitempty"`
	Input        string      `json:"input,omitempty"`
}

func NewToolExecutionStartedEvent(metadata EventMetadata, corr Correlation, toolCallID, toolName, input string) *EventToolExecutionStarted {
	return &EventToolExecutionStarted{EventImpl: EventImpl{Type_: EventTypeToolExecutionStarted, Metadata_: metadata}, Correlation_: corr, ToolCallID: toolCallID, ToolName: toolName, Input: input}
}

func (e *EventToolExecutionStarted) Correlation() Correlation { return e.Correlation_ }

var _ CorrelatedEvent = &EventToolExecutionStarted{}

type EventToolResultReady struct {
	EventImpl
	Correlation_ Correlation `json:"correlation"`
	ToolCallID   string      `json:"tool_call_id"`
	ToolName     string      `json:"tool_name,omitempty"`
	Result       string      `json:"result"`
	Status       string      `json:"status,omitempty"`
}

func NewToolResultReadyEvent(metadata EventMetadata, corr Correlation, toolCallID, toolName, result, status string) *EventToolResultReady {
	return &EventToolResultReady{EventImpl: EventImpl{Type_: EventTypeToolResultReady, Metadata_: metadata}, Correlation_: corr, ToolCallID: toolCallID, ToolName: toolName, Result: result, Status: status}
}

func (e *EventToolResultReady) Correlation() Correlation { return e.Correlation_ }

var _ CorrelatedEvent = &EventToolResultReady{}

type EventToolCallFinished struct {
	EventImpl
	Correlation_ Correlation `json:"correlation"`
	ToolCallID   string      `json:"tool_call_id"`
	ToolName     string      `json:"tool_name,omitempty"`
	Status       string      `json:"status,omitempty"`
}

func NewToolCallFinishedEvent(metadata EventMetadata, corr Correlation, toolCallID, toolName, status string) *EventToolCallFinished {
	return &EventToolCallFinished{EventImpl: EventImpl{Type_: EventTypeToolCallFinished, Metadata_: metadata}, Correlation_: corr, ToolCallID: toolCallID, ToolName: toolName, Status: status}
}

func (e *EventToolCallFinished) Correlation() Correlation { return e.Correlation_ }

var _ CorrelatedEvent = &EventToolCallFinished{}
