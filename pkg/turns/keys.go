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

// Turn metadata keys for Turn.Metadata map
const (
	TurnMetaKeyProvider   TurnMetadataKey = "provider"    // e.g., provider name or payload snippets
	TurnMetaKeyRuntime    TurnMetadataKey = "runtime"     // runtime annotations
	TurnMetaKeyTraceID    TurnMetadataKey = "trace_id"    // tracing id for correlation
	TurnMetaKeyUsage      TurnMetadataKey = "usage"       // token usage summary
	TurnMetaKeyStopReason TurnMetadataKey = "stop_reason" // provider stop reason
	TurnMetaKeyModel      TurnMetadataKey = "model"       // model identifier
)

// Run metadata keys for Run.Metadata map
const (
	RunMetaKeyTraceID RunMetadataKey = "trace_id" // tracing id for correlation
)

// Block metadata keys for Block.Metadata map
const (
	// BlockMetaKeyClaudeOriginalContent stores provider-native content blocks for Claude
	BlockMetaKeyClaudeOriginalContent BlockMetadataKey = "claude_original_content"
	BlockMetaKeyToolCalls             BlockMetadataKey = "tool_calls"
	BlockMetaKeyMiddleware            BlockMetadataKey = "middleware"
	// BlockMetaKeyAgentModeTag marks blocks inserted by agentmode middleware
	BlockMetaKeyAgentModeTag BlockMetadataKey = "agentmode_tag"
	// BlockMetaKeyAgentMode stores the agent mode name in block metadata
	BlockMetaKeyAgentMode BlockMetadataKey = "agentmode"
)

// Standard keys for Turn.Data map
const (
	DataKeyToolConfig            TurnDataKey = "tool_config"
	DataKeyAgentModeAllowedTools TurnDataKey = "agent_mode_allowed_tools"
	DataKeyAgentMode             TurnDataKey = "agent_mode"
	DataKeyResponsesServerTools  TurnDataKey = "responses_server_tools"
)
