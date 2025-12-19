package a

import "github.com/go-go-golems/geppetto/pkg/turns"

func okConst(t *turns.Turn) {
	_ = t.Data[turns.DataKeyAgentMode]

	const Local turns.TurnDataKey = "local"
	_ = t.Data[Local]

	const Indirect = turns.DataKeyAgentMode
	_ = t.Data[Indirect]
}

func okConversion(t *turns.Turn) {
	_ = t.Data[turns.TurnDataKey("raw")]
}

func okVar(t *turns.Turn) {
	k := turns.DataKeyAgentMode
	_ = t.Data[k]
}

func okParam(t *turns.Turn, k turns.TurnDataKey) {
	_ = t.Data[k]
}

func badStringLiteral(t *turns.Turn) {
	_ = t.Data["raw"] // want `Data key must be of type`
}

func badUntypedConstIdent(t *turns.Turn) {
	const k = "raw"
	_ = t.Data[k] // want `Data key must be of type`
}

func okTurnMetadataConst(t *turns.Turn) {
	if t.Metadata == nil {
		t.Metadata = map[turns.TurnMetadataKey]any{}
	}
	_ = t.Metadata[turns.TurnMetaKeyModel]
}

func badTurnMetadataString(t *turns.Turn) {
	_ = t.Metadata["model"] // want `Metadata key must be of type`
}

func okTurnMetadataVar(t *turns.Turn) {
	k := turns.TurnMetaKeyModel
	_ = t.Metadata[k]
}

func okBlockMetadataConst(b *turns.Block) {
	if b.Metadata == nil {
		b.Metadata = map[turns.BlockMetadataKey]any{}
	}
	_ = b.Metadata[turns.BlockMetaKeyMiddleware]
}

func badBlockMetadataString(b *turns.Block) {
	_ = b.Metadata["middleware"] // want `Metadata key must be of type`
}

func okBlockMetadataVar(b *turns.Block) {
	k := turns.BlockMetaKeyMiddleware
	_ = b.Metadata[k]
}

func okRunMetadataConst(r *turns.Run) {
	if r.Metadata == nil {
		r.Metadata = map[turns.RunMetadataKey]any{}
	}
	_ = r.Metadata[turns.RunMetaKeyTraceID]
}

func badRunMetadataString(r *turns.Run) {
	_ = r.Metadata["trace_id"] // want `Metadata key must be of type`
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
