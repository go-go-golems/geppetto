package gemini

import "github.com/go-go-golems/geppetto/pkg/turns"

const geminiMetadataNamespace = "gemini"

var (
	keyBlockMetaGeminiThoughtSignature = turns.BlockMetaK[string](geminiMetadataNamespace, "thought_signature", 1)
	keyBlockMetaGeminiThought          = turns.BlockMetaK[bool](geminiMetadataNamespace, "thought", 1)
)
