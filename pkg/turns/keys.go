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

// Recommended keys for Turn/Block/Run Metadata maps
// Note: These are conventions; callers may use additional keys as needed.
const (
	MetaKeyProvider   = "provider"    // e.g., provider name or payload snippets
	MetaKeyRuntime    = "runtime"     // runtime annotations
	MetaKeyTraceID    = "trace_id"    // tracing id for correlation
	MetaKeyUsage      = "usage"       // token usage summary
	MetaKeyStopReason = "stop_reason" // provider stop reason
	MetaKeyModel      = "model"       // model identifier
	// MetaKeyClaudeOriginalContent stores provider-native content blocks for Claude
	MetaKeyClaudeOriginalContent = "claude_original_content"
)

// Standard keys for Turn.Data map
const (
	DataKeyToolRegistry = "tool_registry"
	DataKeyToolConfig   = "tool_config"
)
