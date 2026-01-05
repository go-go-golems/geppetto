package turns

// Standard keys used in Block.Payload maps
const (
	PayloadKeyText   = "text"
	PayloadKeyID     = "id"
	PayloadKeyName   = "name"
	PayloadKeyArgs   = "args"
	PayloadKeyResult = "result"
	// PayloadKeyImages carries a slice of image specs attached to a chat block
	PayloadKeyImages = "images"
	// PayloadKeyEncryptedContent stores provider encrypted reasoning content
	PayloadKeyEncryptedContent = "encrypted_content"
	// PayloadKeyItemID stores provider-native output item identifier (e.g., fc_...)
	PayloadKeyItemID = "item_id"
)

// Run metadata keys for Run.Metadata map
const (
	RunMetaKeyTraceID RunMetadataKey = "trace_id" // tracing id for correlation
)

// Canonical namespace for geppetto-owned turn data and metadata keys.
const GeppettoNamespaceKey = "geppetto"

// Canonical value keys (scoped to GeppettoNamespaceKey).
const (
	// Turn.Data
	ToolConfigValueKey            = "tool_config" // typed key lives in inference/engine to avoid import cycles
	AgentModeAllowedToolsValueKey = "agent_mode_allowed_tools"
	AgentModeValueKey             = "agent_mode"
	ResponsesServerToolsValueKey  = "responses_server_tools"

	// Turn.Metadata
	TurnMetaProviderValueKey   = "provider"
	TurnMetaRuntimeValueKey    = "runtime"
	TurnMetaTraceIDValueKey    = "trace_id"
	TurnMetaUsageValueKey      = "usage"
	TurnMetaStopReasonValueKey = "stop_reason"
	TurnMetaModelValueKey      = "model"

	// Block.Metadata
	BlockMetaClaudeOriginalContentValueKey = "claude_original_content"
	BlockMetaToolCallsValueKey             = "tool_calls"
	BlockMetaMiddlewareValueKey            = "middleware"
	BlockMetaAgentModeTagValueKey          = "agentmode_tag"
	BlockMetaAgentModeValueKey             = "agentmode"
)

// Typed keys for Turn.Data (geppetto-owned).
//
// Note: KeyToolConfig lives in `geppetto/pkg/inference/engine` to avoid the import cycle:
// turns -> engine (ToolConfig type) -> turns (Engine interface uses *turns.Turn)
var (
	KeyAgentModeAllowedTools = DataK[[]string](GeppettoNamespaceKey, AgentModeAllowedToolsValueKey, 1)
	KeyAgentMode             = DataK[string](GeppettoNamespaceKey, AgentModeValueKey, 1)
	KeyResponsesServerTools  = DataK[[]any](GeppettoNamespaceKey, ResponsesServerToolsValueKey, 1)
)

// Typed keys for Turn.Metadata (geppetto-owned).
var (
	KeyTurnMetaProvider   = TurnMetaK[string](GeppettoNamespaceKey, TurnMetaProviderValueKey, 1)
	KeyTurnMetaRuntime    = TurnMetaK[any](GeppettoNamespaceKey, TurnMetaRuntimeValueKey, 1)
	KeyTurnMetaTraceID    = TurnMetaK[string](GeppettoNamespaceKey, TurnMetaTraceIDValueKey, 1)
	KeyTurnMetaUsage      = TurnMetaK[any](GeppettoNamespaceKey, TurnMetaUsageValueKey, 1)
	KeyTurnMetaStopReason = TurnMetaK[string](GeppettoNamespaceKey, TurnMetaStopReasonValueKey, 1)
	KeyTurnMetaModel      = TurnMetaK[string](GeppettoNamespaceKey, TurnMetaModelValueKey, 1)
)

// Typed keys for Block.Metadata (geppetto-owned).
var (
	KeyBlockMetaClaudeOriginalContent = BlockMetaK[any](GeppettoNamespaceKey, BlockMetaClaudeOriginalContentValueKey, 1)
	KeyBlockMetaToolCalls             = BlockMetaK[any](GeppettoNamespaceKey, BlockMetaToolCallsValueKey, 1)
	KeyBlockMetaMiddleware            = BlockMetaK[string](GeppettoNamespaceKey, BlockMetaMiddlewareValueKey, 1)
	KeyBlockMetaAgentModeTag          = BlockMetaK[string](GeppettoNamespaceKey, BlockMetaAgentModeTagValueKey, 1)
	KeyBlockMetaAgentMode             = BlockMetaK[string](GeppettoNamespaceKey, BlockMetaAgentModeValueKey, 1)
)
