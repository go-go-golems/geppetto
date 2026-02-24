package profiles

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

	profile := &Profile{Slug: MustProfileSlug("default")}
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

func TestNormalizeProfileExtensions_KnownCodecDecodeSuccess(t *testing.T) {
	codec := starterSuggestionsCodec{key: MustExtensionKey("webchat.starter_suggestions@v1")}
	registry, err := NewInMemoryExtensionCodecRegistry(codec)
	if err != nil {
		t.Fatalf("NewInMemoryExtensionCodecRegistry returned error: %v", err)
	}

	normalized, err := NormalizeProfileExtensions(map[string]any{
		" WebChat.Starter_Suggestions@V1 ": map[string]any{
			"items": []any{"one", "two"},
		},
	}, registry)
	if err != nil {
		t.Fatalf("NormalizeProfileExtensions returned error: %v", err)
	}

	value, ok := normalized["webchat.starter_suggestions@v1"]
	if !ok {
		t.Fatalf("expected normalized key present")
	}
	payload, ok := value.(starterSuggestionsPayload)
	if !ok {
		t.Fatalf("expected decoded payload type, got %T", value)
	}
	if got, want := len(payload.Items), 2; got != want {
		t.Fatalf("items length mismatch: got=%d want=%d", got, want)
	}
}

func TestNormalizeProfileExtensions_KnownCodecDecodeFailure(t *testing.T) {
	codec := starterSuggestionsCodec{
		key:      MustExtensionKey("webchat.starter_suggestions@v1"),
		forceErr: true,
	}
	registry, err := NewInMemoryExtensionCodecRegistry(codec)
	if err != nil {
		t.Fatalf("NewInMemoryExtensionCodecRegistry returned error: %v", err)
	}

	_, err = NormalizeProfileExtensions(map[string]any{
		"webchat.starter_suggestions@v1": map[string]any{
			"items": []any{"one"},
		},
	}, registry)
	if err == nil {
		t.Fatalf("expected decode failure")
	}
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestNormalizeProfileExtensions_UnknownPassThroughAndDeepCopy(t *testing.T) {
	input := map[string]any{
		"App.Custom@V1": map[string]any{
			"items": []any{
				map[string]any{"enabled": true},
			},
		},
	}
	normalized, err := NormalizeProfileExtensions(input, nil)
	if err != nil {
		t.Fatalf("NormalizeProfileExtensions returned error: %v", err)
	}

	value, ok := normalized["app.custom@v1"]
	if !ok {
		t.Fatalf("expected normalized unknown key")
	}
	items := value.(map[string]any)["items"].([]any)
	items[0].(map[string]any)["enabled"] = false

	originalItems := input["App.Custom@V1"].(map[string]any)["items"].([]any)
	enabled := originalItems[0].(map[string]any)["enabled"].(bool)
	if !enabled {
		t.Fatalf("expected original unknown payload to remain unchanged")
	}
}

func TestNewStoreRegistry_WithExtensionCodecRegistryOption(t *testing.T) {
	store := NewInMemoryProfileStore()
	codecRegistry, err := NewInMemoryExtensionCodecRegistry()
	if err != nil {
		t.Fatalf("NewInMemoryExtensionCodecRegistry returned error: %v", err)
	}

	svc, err := NewStoreRegistry(store, MustRegistrySlug("default"), WithExtensionCodecRegistry(codecRegistry))
	if err != nil {
		t.Fatalf("NewStoreRegistry returned error: %v", err)
	}
	if svc.extensionCodecs != codecRegistry {
		t.Fatalf("expected extension codec registry to be wired on service")
	}
}
