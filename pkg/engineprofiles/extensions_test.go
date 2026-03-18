package engineprofiles

import (
	"errors"
	"fmt"
	"testing"
)

type starterSuggestionsPayload struct {
	Items []string `json:"items"`
}

type starterSuggestionsCodec struct {
	key      ExtensionKey
	forceErr bool
}

func (c starterSuggestionsCodec) Key() ExtensionKey {
	return c.key
}

func (c starterSuggestionsCodec) Decode(raw any) (any, error) {
	if c.forceErr {
		return nil, fmt.Errorf("forced decode failure")
	}
	key := ProfileExtensionKeyFromID[starterSuggestionsPayload](c.key)
	return key.Decode(raw)
}

func (c starterSuggestionsCodec) JSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"items": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "string",
				},
			},
		},
	}
}

func TestParseExtensionKey_NormalizesAndValidates(t *testing.T) {
	key, err := ParseExtensionKey("  WebChat.Starter_Suggestions@V1 ")
	if err != nil {
		t.Fatalf("ParseExtensionKey returned error: %v", err)
	}
	if got, want := key.String(), "webchat.starter_suggestions@v1"; got != want {
		t.Fatalf("extension key mismatch: got=%q want=%q", got, want)
	}

	invalidInputs := []string{
		"",
		"missingversion",
		"namespace.feature@v0",
		"namespace.@v1",
		"namespace.feature@v-1",
		"name space.feature@v1",
	}
	for _, raw := range invalidInputs {
		if _, err := ParseExtensionKey(raw); err == nil {
			t.Fatalf("expected parse failure for %q", raw)
		}
	}
}

func TestNewExtensionKey_PanicFreeConstructor(t *testing.T) {
	key, err := NewExtensionKey("webchat", "starter_suggestions", 1)
	if err != nil {
		t.Fatalf("NewExtensionKey returned error: %v", err)
	}
	if got, want := key.String(), "webchat.starter_suggestions@v1"; got != want {
		t.Fatalf("extension key mismatch: got=%q want=%q", got, want)
	}

	if _, err := NewExtensionKey("webchat", "starter_suggestions", 0); err == nil {
		t.Fatalf("expected version validation failure")
	}
}

func TestProfileExtensionKey_GetSetDecode(t *testing.T) {
	key, err := NewProfileExtensionKey[starterSuggestionsPayload]("webchat", "starter_suggestions", 1)
	if err != nil {
		t.Fatalf("NewProfileExtensionKey returned error: %v", err)
	}

	profile := &EngineProfile{Slug: MustEngineProfileSlug("default")}
	if _, ok, err := key.Get(profile); err != nil || ok {
		t.Fatalf("expected missing extension without error, got ok=%v err=%v", ok, err)
	}

	in := starterSuggestionsPayload{Items: []string{"one", "two"}}
	if err := key.Set(profile, in); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	out, ok, err := key.Get(profile)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if !ok {
		t.Fatalf("expected extension value present")
	}
	if got, want := len(out.Items), 2; got != want {
		t.Fatalf("items length mismatch: got=%d want=%d", got, want)
	}
	if got, want := out.Items[0], "one"; got != want {
		t.Fatalf("first item mismatch: got=%q want=%q", got, want)
	}

	if err := key.Set(nil, in); err == nil {
		t.Fatalf("expected nil profile error")
	}
}

func TestInMemoryExtensionCodecRegistry_DuplicateGuard(t *testing.T) {
	codec := starterSuggestionsCodec{key: MustExtensionKey("webchat.starter_suggestions@v1")}
	registry, err := NewInMemoryExtensionCodecRegistry(codec)
	if err != nil {
		t.Fatalf("NewInMemoryExtensionCodecRegistry returned error: %v", err)
	}

	err = registry.Register(codec)
	if err == nil {
		t.Fatalf("expected duplicate codec registration to fail")
	}
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestInMemoryExtensionCodecRegistry_ListCodecsSorted(t *testing.T) {
	registry, err := NewInMemoryExtensionCodecRegistry(
		starterSuggestionsCodec{key: MustExtensionKey("zeta.feature@v1")},
		starterSuggestionsCodec{key: MustExtensionKey("alpha.feature@v1")},
	)
	if err != nil {
		t.Fatalf("NewInMemoryExtensionCodecRegistry returned error: %v", err)
	}

	codecs := registry.ListCodecs()
	if got, want := len(codecs), 2; got != want {
		t.Fatalf("codec count mismatch: got=%d want=%d", got, want)
	}
	if got, want := codecs[0].Key().String(), "alpha.feature@v1"; got != want {
		t.Fatalf("codec order mismatch at index 0: got=%q want=%q", got, want)
	}
	if got, want := codecs[1].Key().String(), "zeta.feature@v1"; got != want {
		t.Fatalf("codec order mismatch at index 1: got=%q want=%q", got, want)
	}
}
