package turns

// Standard keys used in Block.Payload maps
const (
	PayloadKeyText   = "text"
	PayloadKeyID     = "id"
	PayloadKeyName   = "name"
	PayloadKeyArgs   = "args"
	PayloadKeyResult = "result"
	PayloadKeyError  = "error"
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

