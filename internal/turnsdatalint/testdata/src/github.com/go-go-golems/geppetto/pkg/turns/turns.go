package turns

type TurnDataKey string

type Turn struct {
	Data map[TurnDataKey]any
}

const (
	DataKeyToolRegistry TurnDataKey = "tool_registry"
	DataKeyAgentMode    TurnDataKey = "agent_mode"
)


