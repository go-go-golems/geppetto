package turns

type TurnDataKey string
type TurnMetadataKey string
type BlockMetadataKey string
type RunMetadataKey string

type Turn struct {
	Data     map[TurnDataKey]any
	Metadata map[TurnMetadataKey]any
	Blocks   []Block
}

type Block struct {
	Payload  map[string]any
	Metadata map[BlockMetadataKey]any
}

type Run struct {
	Metadata map[RunMetadataKey]any
}

const (
	DataKeyToolRegistry TurnDataKey = "tool_registry"
	DataKeyAgentMode    TurnDataKey = "agent_mode"
)

const (
	TurnMetaKeyModel TurnMetadataKey = "model"
)

const (
	BlockMetaKeyMiddleware BlockMetadataKey = "middleware"
)

const (
	RunMetaKeyTraceID RunMetadataKey = "trace_id"
)

const (
	PayloadKeyText = "text"
)
