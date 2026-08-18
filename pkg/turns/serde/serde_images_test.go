package serde

import (
	"testing"

	"github.com/go-go-golems/geppetto/pkg/turns"
)

// TestYAMLRoundTripPreservesImagesViaBlockImages guards against the regression
// where images attached to a user block were silently dropped by provider
// adapters after a persist/reload cycle: yaml decoding yields []any, not the
// []map[string]any that NewUserMultimodalBlock produces.
func TestYAMLRoundTripPreservesImagesViaBlockImages(t *testing.T) {
	src := &turns.Turn{ID: "t1"}
	turns.AppendBlock(src, turns.NewUserMultimodalBlock("what is this?", []map[string]any{
		{"media_type": "image/png", "url": "https://example.com/a.png", "detail": "high"},
		{"media_type": "image/jpeg", "content": []byte("JPG")},
	}))

	data, err := ToYAML(src, Options{})
	if err != nil {
		t.Fatalf("ToYAML: %v", err)
	}
	back, err := FromYAML(data)
	if err != nil {
		t.Fatalf("FromYAML: %v", err)
	}
	if len(back.Blocks) != 1 {
		t.Fatalf("blocks = %d", len(back.Blocks))
	}
	// The concrete type changes across the round trip; the accessor must not care.
	if _, ok := back.Blocks[0].Payload[turns.PayloadKeyImages].([]map[string]any); ok {
		t.Logf("note: yaml decoder returned []map[string]any directly; accessor still exercised")
	}
	imgs := turns.BlockImages(back.Blocks[0])
	if len(imgs) != 2 {
		t.Fatalf("BlockImages after round trip = %#v", imgs)
	}
	if imgs[0]["url"] != "https://example.com/a.png" || imgs[0]["detail"] != "high" || imgs[0]["media_type"] != "image/png" {
		t.Fatalf("first image = %#v", imgs[0])
	}
	if imgs[1]["media_type"] != "image/jpeg" {
		t.Fatalf("second image = %#v", imgs[1])
	}
	// []byte content comes back as a generic slice of numbers (yaml.v3 encodes
	// []byte held in an interface as a sequence); imageparts.NormalizeImageMap
	// must accept it. We only assert that the value is present here.
	if imgs[1]["content"] == nil {
		t.Fatalf("content missing after round trip: %#v", imgs[1])
	}
}
