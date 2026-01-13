package a

import "github.com/go-go-golems/geppetto/pkg/turns"

func okRunMetadataConst(r *turns.Run) {
	if r.Metadata == nil {
		r.Metadata = map[turns.RunMetadataKey]any{}
	}
	_ = r.Metadata[turns.RunMetaKeyTraceID]

	const Local turns.RunMetadataKey = "local"
	_ = r.Metadata[Local]

	const Indirect = turns.RunMetaKeyTraceID
	_ = r.Metadata[Indirect]
}

func badRunMetadataString(r *turns.Run) {
	_ = r.Metadata["trace_id"] // want `Metadata key must be of type`
}

func badRunMetadataUntypedConstIdent(r *turns.Run) {
	const k = "trace_id"
	_ = r.Metadata[k] // want `Metadata key must be of type`
}

func okPayloadConst(b *turns.Block) {
	if b.Payload == nil {
		b.Payload = map[string]any{}
	}
	_ = b.Payload[turns.PayloadKeyText]

	const Local = "local_payload_key"
	_ = b.Payload[Local]
}

func badPayloadStringLiteral(b *turns.Block) {
	_ = b.Payload["text"] // want `Payload key must be a const string`
}

func badPayloadVar(b *turns.Block) {
	k := turns.PayloadKeyText
	_ = b.Payload[k] // want `Payload key must be a const string`
}

func badAdHocKeyConstructor() {
	_ = turns.DataK[string]("ns", "val", 1) // want `do not call turns.DataK outside key-definition files`
}
