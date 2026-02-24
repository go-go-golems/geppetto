package profiles

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var extensionKeyPattern = regexp.MustCompile(`^([a-z0-9](?:[a-z0-9_-]{0,62}[a-z0-9])?)\.([a-z0-9](?:[a-z0-9_-]{0,62}[a-z0-9])?)@v([1-9][0-9]{0,4})$`)

// ExtensionKey is a canonical profile-extension identifier in the form:
// namespace.feature@vN
type ExtensionKey string

func ParseExtensionKey(raw string) (ExtensionKey, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return "", fmt.Errorf("extension key cannot be empty")
	}
	if !extensionKeyPattern.MatchString(normalized) {
		return "", fmt.Errorf("extension key %q is invalid (expected namespace.feature@vN)", raw)
	}
	return ExtensionKey(normalized), nil
}

func NewExtensionKey(namespace, feature string, version uint16) (ExtensionKey, error) {
	if version == 0 {
		return "", fmt.Errorf("extension key version must be >= 1")
	}
	return ParseExtensionKey(fmt.Sprintf("%s.%s@v%d", namespace, feature, version))
}

func MustExtensionKey(raw string) ExtensionKey {
	key, err := ParseExtensionKey(raw)
	if err != nil {
		panic(err)
	}
	return key
}

func (k ExtensionKey) String() string {
	return string(k)
}

func (k ExtensionKey) IsZero() bool {
	return strings.TrimSpace(string(k)) == ""
}

type ProfileExtensionKey[T any] struct {
	id ExtensionKey
}

func NewProfileExtensionKey[T any](namespace, feature string, version uint16) (ProfileExtensionKey[T], error) {
	key, err := NewExtensionKey(namespace, feature, version)
	if err != nil {
		return ProfileExtensionKey[T]{}, err
	}
	return ProfileExtensionKey[T]{id: key}, nil
}

func MustProfileExtensionKey[T any](namespace, feature string, version uint16) ProfileExtensionKey[T] {
	key, err := NewProfileExtensionKey[T](namespace, feature, version)
	if err != nil {
		panic(err)
	}
	return key
}

func ProfileExtensionKeyFromID[T any](id ExtensionKey) ProfileExtensionKey[T] {
	return ProfileExtensionKey[T]{id: id}
}

func (k ProfileExtensionKey[T]) String() string {
	return k.id.String()
}

func (k ProfileExtensionKey[T]) ID() ExtensionKey {
	return k.id
}

func (k ProfileExtensionKey[T]) Decode(raw any) (T, error) {
	var zero T
	if raw == nil {
		return zero, fmt.Errorf("profile.extensions[%q]: cannot decode <nil>", k.id)
	}
	if typed, ok := raw.(T); ok {
		return typed, nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return zero, fmt.Errorf("profile.extensions[%q]: json marshal %T: %w", k.id, raw, err)
	}
	ptr := new(T)
	if err := json.Unmarshal(b, ptr); err != nil {
		return zero, fmt.Errorf("profile.extensions[%q]: json unmarshal into %T: %w", k.id, zero, err)
	}
	return *ptr, nil
}

func (k ProfileExtensionKey[T]) Get(profile *Profile) (T, bool, error) {
	var zero T
	if profile == nil || len(profile.Extensions) == 0 {
		return zero, false, nil
	}
	raw, ok := profile.Extensions[k.id.String()]
	if !ok {
		return zero, false, nil
	}
	typed, err := k.Decode(raw)
	if err != nil {
		return zero, true, err
	}
	return typed, true, nil
}

func (k ProfileExtensionKey[T]) Set(profile *Profile, value T) error {
	if profile == nil {
		return fmt.Errorf("profile is required")
	}
	if _, err := json.Marshal(value); err != nil {
		return fmt.Errorf("profile.extensions[%q]: value not serializable: %w", k.id, err)
	}
	if profile.Extensions == nil {
		profile.Extensions = map[string]any{}
	}
	profile.Extensions[k.id.String()] = value
	return nil
}

func (k ProfileExtensionKey[T]) Delete(profile *Profile) {
	if profile == nil || profile.Extensions == nil {
		return
	}
	delete(profile.Extensions, k.id.String())
	if len(profile.Extensions) == 0 {
		profile.Extensions = nil
	}
}

// ExtensionCodec decodes and normalizes extension payloads for one extension key.
type ExtensionCodec interface {
	Key() ExtensionKey
	Decode(raw any) (any, error)
}

// ExtensionCodecRegistry resolves codecs for extension keys.
type ExtensionCodecRegistry interface {
	Lookup(key ExtensionKey) (ExtensionCodec, bool)
}

// InMemoryExtensionCodecRegistry stores codecs in a map keyed by extension key.
type InMemoryExtensionCodecRegistry struct {
	codecs map[ExtensionKey]ExtensionCodec
}

func NewInMemoryExtensionCodecRegistry(codecs ...ExtensionCodec) (*InMemoryExtensionCodecRegistry, error) {
	ret := &InMemoryExtensionCodecRegistry{
		codecs: map[ExtensionKey]ExtensionCodec{},
	}
	for _, codec := range codecs {
		if err := ret.Register(codec); err != nil {
			return nil, err
		}
	}
	return ret, nil
}

func (r *InMemoryExtensionCodecRegistry) Lookup(key ExtensionKey) (ExtensionCodec, bool) {
	if r == nil || len(r.codecs) == 0 {
		return nil, false
	}
	codec, ok := r.codecs[key]
	return codec, ok
}

func (r *InMemoryExtensionCodecRegistry) Register(codec ExtensionCodec) error {
	if r == nil {
		return fmt.Errorf("codec registry is nil")
	}
	if codec == nil {
		return &ValidationError{Field: "extensions.codec", Reason: "codec must not be nil"}
	}
	key := codec.Key()
	if key.IsZero() {
		return &ValidationError{Field: "extensions.codec.key", Reason: "codec key must not be empty"}
	}
	if _, err := ParseExtensionKey(key.String()); err != nil {
		return &ValidationError{Field: "extensions.codec.key", Reason: err.Error()}
	}
	if _, exists := r.codecs[key]; exists {
		return &ValidationError{Field: fmt.Sprintf("extensions.codecs[%s]", key), Reason: "duplicate codec key"}
	}
	r.codecs[key] = codec
	return nil
}

// NormalizeProfileExtensions canonicalizes extension keys and applies registered
// codec decode/normalization where available. Unknown keys are preserved.
func NormalizeProfileExtensions(raw map[string]any, registry ExtensionCodecRegistry) (map[string]any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	out := make(map[string]any, len(raw))
	for rawKey, rawValue := range raw {
		key, err := ParseExtensionKey(rawKey)
		if err != nil {
			return nil, &ValidationError{
				Field:  fmt.Sprintf("profile.extensions[%s]", strings.TrimSpace(rawKey)),
				Reason: err.Error(),
			}
		}
		if registry != nil {
			if codec, ok := registry.Lookup(key); ok {
				decoded, err := codec.Decode(deepCopyAny(rawValue))
				if err != nil {
					return nil, &ValidationError{
						Field:  fmt.Sprintf("profile.extensions[%s]", key),
						Reason: fmt.Sprintf("decode failed: %v", err),
					}
				}
				out[key.String()] = deepCopyAny(decoded)
				continue
			}
		}
		out[key.String()] = deepCopyAny(rawValue)
	}
	return out, nil
}
