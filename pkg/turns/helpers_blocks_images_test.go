package turns

import "testing"

func TestBlockImagesAcceptsConcreteSlice(t *testing.T) {
	b := NewUserMultimodalBlock("hi", []map[string]any{{"url": "https://x/a.png"}})
	imgs := BlockImages(b)
	if len(imgs) != 1 || imgs[0]["url"] != "https://x/a.png" {
		t.Fatalf("images = %#v", imgs)
	}
	if !HasImages(b) {
		t.Fatalf("expected HasImages true")
	}
}

func TestBlockImagesAcceptsGenericSlices(t *testing.T) {
	cases := map[string]any{
		"[]any of map[string]any": []any{map[string]any{"url": "u1"}, "junk", map[string]any{"url": "u2"}},
		"[]any of map[any]any":    []any{map[any]any{"url": "u1", 42: "ignored"}, map[any]any{"url": "u2"}},
	}
	for name, raw := range cases {
		b := Block{Kind: BlockKindUser, Role: RoleUser, Payload: map[string]any{PayloadKeyText: "t", PayloadKeyImages: raw}}
		imgs := BlockImages(b)
		if len(imgs) != 2 || imgs[0]["url"] != "u1" || imgs[1]["url"] != "u2" {
			t.Fatalf("%s: images = %#v", name, imgs)
		}
	}
}

func TestBlockImagesEmptyCases(t *testing.T) {
	if got := BlockImages(Block{}); got != nil {
		t.Fatalf("nil payload: %#v", got)
	}
	if got := BlockImages(NewUserTextBlock("x")); got != nil {
		t.Fatalf("text block: %#v", got)
	}
	b := Block{Payload: map[string]any{PayloadKeyImages: "not a slice"}}
	if got := BlockImages(b); got != nil {
		t.Fatalf("string value: %#v", got)
	}
	b = Block{Payload: map[string]any{PayloadKeyImages: []any{"a", 1}}}
	if got := BlockImages(b); got != nil {
		t.Fatalf("no maps: %#v", got)
	}
	if HasImages(NewUserMultimodalBlock("only text", nil)) {
		t.Fatalf("expected HasImages false for nil images")
	}
}

func TestTurnBuilderWithUserMessage(t *testing.T) {
	tb := NewTurnBuilder().WithUserMessage("", nil).WithUserMessage("", []map[string]any{{"url": "u"}}).WithUserMessage("caption", nil)
	tn := tb.Build()
	if len(tn.Blocks) != 2 {
		t.Fatalf("blocks = %d, want 2 (empty message skipped)", len(tn.Blocks))
	}
	if !HasImages(tn.Blocks[0]) || tn.Blocks[0].Payload[PayloadKeyText] != "" {
		t.Fatalf("first block = %#v", tn.Blocks[0])
	}
	if HasImages(tn.Blocks[1]) || tn.Blocks[1].Payload[PayloadKeyText] != "caption" {
		t.Fatalf("second block = %#v", tn.Blocks[1])
	}
}
