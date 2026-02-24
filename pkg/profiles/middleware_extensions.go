package profiles

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	middlewareConfigExtensionNamespace = "middleware"
	middlewareConfigExtensionVersion   = uint16(1)
)

var middlewareExtensionNamePattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9_-]{0,62}[a-z0-9])?$`)

// MiddlewareConfigExtensionPayload stores per-middleware-instance config payloads.
//
// Each instance is addressed by slot:
//   - id:<middleware-id> when MiddlewareUse.ID is present
//   - index:<runtime-index> when MiddlewareUse.ID is empty
type MiddlewareConfigExtensionPayload struct {
	Instances map[string]map[string]any `json:"instances,omitempty"`
}

// MiddlewareConfigExtensionKey builds the typed extension key for a middleware name.
// Key format: middleware.<name>_config@v1
func MiddlewareConfigExtensionKey(middlewareName string) (ProfileExtensionKey[MiddlewareConfigExtensionPayload], error) {
	normalizedName, err := normalizeMiddlewareNameForExtension(middlewareName)
	if err != nil {
		return ProfileExtensionKey[MiddlewareConfigExtensionPayload]{}, err
	}
	return NewProfileExtensionKey[MiddlewareConfigExtensionPayload](
		middlewareConfigExtensionNamespace,
		normalizedName+"_config",
		middlewareConfigExtensionVersion,
	)
}

// MiddlewareConfigInstanceSlot computes the slot key used inside
// MiddlewareConfigExtensionPayload.Instances.
func MiddlewareConfigInstanceSlot(use MiddlewareUse, index int) string {
	if id := strings.TrimSpace(use.ID); id != "" {
		return "id:" + id
	}
	if index < 0 {
		index = 0
	}
	return "index:" + strconv.Itoa(index)
}

// ProjectRuntimeMiddlewareConfigsToExtensions moves any inline
// runtime.middlewares[*].config payloads into typed-key extensions and clears
// the inline config fields on the runtime object.
func ProjectRuntimeMiddlewareConfigsToExtensions(runtime *RuntimeSpec, extensions map[string]any) (map[string]any, error) {
	normalizedExtensions := deepCopyStringAnyMap(extensions)
	if runtime == nil || len(runtime.Middlewares) == 0 {
		return normalizedExtensions, nil
	}

	for i := range runtime.Middlewares {
		use := runtime.Middlewares[i]
		config, err := middlewareConfigMapFromAny(use.Config)
		if err != nil {
			return nil, fmt.Errorf("runtime.middlewares[%d].config: %w", i, err)
		}
		runtime.Middlewares[i].Config = nil
		if config == nil {
			continue
		}
		nextExtensions, err := SetMiddlewareConfigInExtensions(normalizedExtensions, use, i, config)
		if err != nil {
			return nil, fmt.Errorf("runtime.middlewares[%d].config: %w", i, err)
		}
		normalizedExtensions = nextExtensions
	}

	return normalizedExtensions, nil
}

// MiddlewareConfigFromExtensions loads a middleware config payload for one
// middleware instance from typed-key extensions.
func MiddlewareConfigFromExtensions(extensions map[string]any, use MiddlewareUse, index int) (map[string]any, bool, error) {
	key, err := MiddlewareConfigExtensionKey(use.Name)
	if err != nil {
		return nil, false, err
	}
	payload, ok, err := middlewareConfigPayloadFromExtensions(extensions, key)
	if err != nil || !ok {
		return nil, ok, err
	}
	slot := MiddlewareConfigInstanceSlot(use, index)
	config, ok := payload.Instances[slot]
	if !ok && strings.TrimSpace(use.ID) == "" {
		config, ok = payload.Instances["default"]
	}
	if !ok {
		return nil, false, nil
	}
	return deepCopyStringAnyMap(config), true, nil
}

// SetMiddlewareConfigInExtensions writes one middleware instance config payload
// into typed-key extensions and returns a normalized copy.
func SetMiddlewareConfigInExtensions(
	extensions map[string]any,
	use MiddlewareUse,
	index int,
	config map[string]any,
) (map[string]any, error) {
	key, err := MiddlewareConfigExtensionKey(use.Name)
	if err != nil {
		return nil, err
	}

	normalizedExtensions := deepCopyStringAnyMap(extensions)
	payload, _, err := middlewareConfigPayloadFromExtensions(normalizedExtensions, key)
	if err != nil {
		return nil, err
	}
	if payload.Instances == nil {
		payload.Instances = map[string]map[string]any{}
	}

	slot := MiddlewareConfigInstanceSlot(use, index)
	if config == nil {
		delete(payload.Instances, slot)
	} else {
		payload.Instances[slot] = deepCopyStringAnyMap(config)
	}

	if len(payload.Instances) == 0 {
		if len(normalizedExtensions) == 0 {
			return nil, nil
		}
		delete(normalizedExtensions, key.String())
		if len(normalizedExtensions) == 0 {
			return nil, nil
		}
		return normalizedExtensions, nil
	}

	if normalizedExtensions == nil {
		normalizedExtensions = map[string]any{}
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal middleware payload: %w", err)
	}
	var normalizedPayload map[string]any
	if err := json.Unmarshal(b, &normalizedPayload); err != nil {
		return nil, fmt.Errorf("normalize middleware payload: %w", err)
	}
	normalizedExtensions[key.String()] = normalizedPayload
	return normalizedExtensions, nil
}

func normalizeMiddlewareNameForExtension(raw string) (string, error) {
	name := strings.ToLower(strings.TrimSpace(raw))
	if name == "" {
		return "", fmt.Errorf("middleware name must not be empty")
	}
	if !middlewareExtensionNamePattern.MatchString(name) {
		return "", fmt.Errorf("middleware name %q is invalid for typed-key extension mapping", raw)
	}
	return name, nil
}

func middlewareConfigMapFromAny(raw any) (map[string]any, error) {
	if raw == nil {
		return nil, nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("must be JSON-serializable: %w", err)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("must be an object: %w", err)
	}
	return out, nil
}

func middlewareConfigPayloadFromExtensions(
	extensions map[string]any,
	key ProfileExtensionKey[MiddlewareConfigExtensionPayload],
) (MiddlewareConfigExtensionPayload, bool, error) {
	profile := &Profile{Extensions: deepCopyStringAnyMap(extensions)}
	payload, ok, err := key.Get(profile)
	if err != nil {
		return MiddlewareConfigExtensionPayload{}, false, err
	}
	if !ok {
		return MiddlewareConfigExtensionPayload{}, false, nil
	}
	normalized := MiddlewareConfigExtensionPayload{
		Instances: map[string]map[string]any{},
	}
	for slot, cfg := range payload.Instances {
		normalized.Instances[slot] = deepCopyStringAnyMap(cfg)
	}
	return normalized, true, nil
}
