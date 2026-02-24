package middlewarecfg

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"

	gepprofiles "github.com/go-go-golems/geppetto/pkg/profiles"
	"github.com/rs/zerolog/log"
)

// Resolver resolves middleware config using canonical source precedence and JSON schema validation.
type Resolver struct {
	sources []Source
}

// NewResolver constructs a middleware config resolver.
func NewResolver(sources ...Source) *Resolver {
	return &Resolver{
		sources: append([]Source(nil), sources...),
	}
}

// ResolvedConfig is a schema-validated middleware config result.
type ResolvedConfig struct {
	Config       map[string]any       `json:"config"`
	PathValues   map[string]any       `json:"path_values,omitempty"`
	OrderedPaths []string             `json:"ordered_paths,omitempty"`
	Sources      []ResolvedSourceRef  `json:"sources,omitempty"`
	Trace        map[string]PathTrace `json:"trace,omitempty"`
}

// ResolvedSourceRef identifies a source that participated in resolution.
type ResolvedSourceRef struct {
	Name  string      `json:"name"`
	Layer SourceLayer `json:"layer"`
}

// ParseStep captures one path-level write applied by the resolver.
type ParseStep struct {
	Source   string         `json:"source"`
	Layer    SourceLayer    `json:"layer"`
	Path     string         `json:"path"`
	Raw      any            `json:"raw"`
	Value    any            `json:"value"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// PathTrace stores full write history for one JSON pointer path.
type PathTrace struct {
	Path  string      `json:"path"`
	Value any         `json:"value"`
	Steps []ParseStep `json:"steps"`
}

// LatestValue returns the latest winning value for a path.
func (r *ResolvedConfig) LatestValue(path string) (any, bool) {
	if r == nil {
		return nil, false
	}
	if r.Trace != nil {
		if trace, ok := r.Trace[path]; ok {
			return copyAny(trace.Value), true
		}
	}
	v, ok := r.PathValues[path]
	if !ok {
		return nil, false
	}
	return copyAny(v), true
}

// History returns the full parse-step history for a path.
func (r *ResolvedConfig) History(path string) []ParseStep {
	if r == nil || r.Trace == nil {
		return nil
	}
	trace, ok := r.Trace[path]
	if !ok || len(trace.Steps) == 0 {
		return nil
	}
	steps := make([]ParseStep, 0, len(trace.Steps))
	for _, step := range trace.Steps {
		steps = append(steps, ParseStep{
			Source:   step.Source,
			Layer:    step.Layer,
			Path:     step.Path,
			Raw:      copyAny(step.Raw),
			Value:    copyAny(step.Value),
			Metadata: copyStringAnyMap(step.Metadata),
		})
	}
	return steps
}

// Resolve resolves one middleware use config against a middleware definition schema.
func (r *Resolver) Resolve(def Definition, use gepprofiles.MiddlewareUse) (*ResolvedConfig, error) {
	if def == nil {
		return nil, fmt.Errorf("middleware definition is nil")
	}
	defName := strings.TrimSpace(def.Name())
	if defName == "" {
		return nil, fmt.Errorf("middleware definition name is empty")
	}

	useName := strings.TrimSpace(use.Name)
	if useName == "" {
		return nil, fmt.Errorf("middleware use name is empty")
	}
	if useName != defName {
		return nil, fmt.Errorf("middleware use name %q does not match definition %q", useName, defName)
	}
	useKey := middlewareUseDiagnosticKey(use)

	schema := copyStringAnyMap(def.ConfigJSONSchema())
	if len(schema) == 0 {
		schema = map[string]any{"type": "object"}
	}

	ordered, err := canonicalOrderedSources(r.sources)
	if err != nil {
		return nil, err
	}

	state := map[string]any{}
	pathValues := map[string]any{}
	trace := map[string]PathTrace{}
	sourceRefs := make([]ResolvedSourceRef, 0, len(ordered)+1)

	defaultPayload, ok := schemaDefaultsObject(schema)
	if ok {
		if err := applyPayloadWithProjection(
			state,
			pathValues,
			trace,
			schema,
			defaultPayload,
			SourceLayerSchemaDefaults,
			SourceLayerSchemaDefaults.String(),
			useKey,
		); err != nil {
			return nil, err
		}
		sourceRefs = append(sourceRefs, ResolvedSourceRef{
			Name:  SourceLayerSchemaDefaults.String(),
			Layer: SourceLayerSchemaDefaults,
		})
	}

	for _, source := range ordered {
		payload, hasPayload, err := source.source.Payload(def, use)
		if err != nil {
			sourceName := strings.TrimSpace(source.source.Name())
			layer := source.source.Layer()
			logResolverSourceReject(useKey, sourceName, layer, "", err)
			return nil, fmt.Errorf("resolve middleware %s source %s[%s]: %w", useKey, sourceName, layer, err)
		}
		if !hasPayload || len(payload) == 0 {
			continue
		}
		if err := applyPayloadWithProjection(
			state,
			pathValues,
			trace,
			schema,
			payload,
			source.source.Layer(),
			source.source.Name(),
			useKey,
		); err != nil {
			return nil, err
		}
		sourceRefs = append(sourceRefs, ResolvedSourceRef{
			Name:  strings.TrimSpace(source.source.Name()),
			Layer: source.source.Layer(),
		})
	}

	validated, err := coerceAndValidate(schema, state)
	if err != nil {
		return nil, fmt.Errorf("final middleware %s config validation failed: %w", useKey, err)
	}
	validatedMap, ok := toStringAnyMap(validated)
	if !ok {
		return nil, fmt.Errorf("final middleware %s config must resolve to object", useKey)
	}

	orderedPaths := make([]string, 0, len(pathValues))
	for path := range pathValues {
		orderedPaths = append(orderedPaths, path)
	}
	sort.Strings(orderedPaths)

	return &ResolvedConfig{
		Config:       validatedMap,
		PathValues:   pathValues,
		OrderedPaths: orderedPaths,
		Sources:      sourceRefs,
		Trace:        trace,
	}, nil
}

func applyPayloadWithProjection(
	state map[string]any,
	pathValues map[string]any,
	trace map[string]PathTrace,
	schema map[string]any,
	payload map[string]any,
	layer SourceLayer,
	sourceName string,
	useKey string,
) error {
	sourceName = strings.TrimSpace(sourceName)
	writes, err := projectPayloadWrites(schema, payload)
	if err != nil {
		logResolverSourceReject(useKey, sourceName, layer, "", err)
		return fmt.Errorf("project middleware %s payload for source %s[%s]: %w", useKey, sourceName, layer, err)
	}

	for _, write := range writes {
		value, err := coerceAndValidate(write.Schema, write.RawValue)
		if err != nil {
			logResolverSourceReject(useKey, sourceName, layer, write.Path, err)
			return fmt.Errorf(
				"coerce middleware %s payload at %s from source %s[%s]: %w",
				useKey,
				write.Path,
				sourceName,
				layer,
				err,
			)
		}
		if err := setJSONPointer(state, write.Path, value); err != nil {
			logResolverSourceReject(useKey, sourceName, layer, write.Path, err)
			return fmt.Errorf("apply middleware %s payload at %s from source %s[%s]: %w", useKey, write.Path, sourceName, layer, err)
		}
		pathValues[write.Path] = copyAny(value)
		pathTrace := trace[write.Path]
		pathTrace.Path = write.Path
		pathTrace.Value = copyAny(value)
		pathTrace.Steps = append(pathTrace.Steps, ParseStep{
			Source: sourceName,
			Layer:  layer,
			Path:   write.Path,
			Raw:    copyAny(write.RawValue),
			Value:  copyAny(value),
			Metadata: map[string]any{
				"schema_type": schemaType(write.Schema),
			},
		})
		trace[write.Path] = pathTrace
	}
	return nil
}

func middlewareUseDiagnosticKey(use gepprofiles.MiddlewareUse) string {
	name := strings.TrimSpace(use.Name)
	if name == "" {
		name = "middleware"
	}
	if id := strings.TrimSpace(use.ID); id != "" {
		return fmt.Sprintf("%s#%s", name, id)
	}
	return name
}

func logResolverSourceReject(useKey string, sourceName string, layer SourceLayer, path string, err error) {
	logger := log.Error().
		Str("component", "middlewarecfg.resolver").
		Str("middleware_use", strings.TrimSpace(useKey)).
		Str("source", strings.TrimSpace(sourceName)).
		Str("layer", string(layer)).
		Err(err)
	if strings.TrimSpace(path) != "" {
		logger = logger.Str("path", strings.TrimSpace(path))
	}
	logger.Msg("middleware source payload rejected")
}

type projectedWrite struct {
	Path     string
	Schema   map[string]any
	RawValue any
}

func projectPayloadWrites(schema map[string]any, payload map[string]any) ([]projectedWrite, error) {
	if len(payload) == 0 {
		return nil, nil
	}
	writes := make([]projectedWrite, 0, len(payload))
	if err := projectWritesRecursive(schema, payload, "", &writes); err != nil {
		return nil, err
	}
	sort.Slice(writes, func(i, j int) bool {
		return writes[i].Path < writes[j].Path
	})
	return writes, nil
}

func projectWritesRecursive(schema map[string]any, payload map[string]any, basePath string, out *[]projectedWrite) error {
	keys := make([]string, 0, len(payload))
	for key := range payload {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := payload[key]
		childPath := basePath + "/" + escapeJSONPointerToken(key)
		childSchema, ok, err := childSchemaForProperty(schema, key)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("path %s is not allowed by schema", childPath)
		}

		childType := schemaType(childSchema)
		if childType == "object" {
			childObject, objectOK := toStringAnyMap(value)
			if !objectOK {
				*out = append(*out, projectedWrite{
					Path:     childPath,
					Schema:   childSchema,
					RawValue: copyAny(value),
				})
				continue
			}
			if len(childObject) == 0 {
				*out = append(*out, projectedWrite{
					Path:     childPath,
					Schema:   childSchema,
					RawValue: map[string]any{},
				})
				continue
			}
			if err := projectWritesRecursive(childSchema, childObject, childPath, out); err != nil {
				return err
			}
			continue
		}

		*out = append(*out, projectedWrite{
			Path:     childPath,
			Schema:   childSchema,
			RawValue: copyAny(value),
		})
	}
	return nil
}

func childSchemaForProperty(schema map[string]any, property string) (map[string]any, bool, error) {
	if props, ok := toStringAnyMap(schema["properties"]); ok {
		if propSchema, ok := toStringAnyMap(props[property]); ok {
			return propSchema, true, nil
		}
	}

	additional := schema["additionalProperties"]
	if additional == nil {
		return nil, false, nil
	}
	if allowed, ok := additional.(bool); ok {
		if allowed {
			return map[string]any{}, true, nil
		}
		return nil, false, nil
	}
	additionalSchema, ok := toStringAnyMap(additional)
	if !ok {
		return nil, false, fmt.Errorf("additionalProperties schema must be an object or boolean")
	}
	return additionalSchema, true, nil
}

func schemaDefaultsObject(schema map[string]any) (map[string]any, bool) {
	value, ok := schemaDefaultValue(schema)
	if !ok {
		return nil, false
	}
	object, ok := toStringAnyMap(value)
	if !ok {
		return nil, false
	}
	return object, true
}

func schemaDefaultValue(schema map[string]any) (any, bool) {
	if def, ok := schema["default"]; ok {
		return copyAny(def), true
	}

	if schemaType(schema) != "object" {
		return nil, false
	}

	props, ok := toStringAnyMap(schema["properties"])
	if !ok {
		return nil, false
	}
	if len(props) == 0 {
		return nil, false
	}

	keys := make([]string, 0, len(props))
	for key := range props {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := map[string]any{}
	for _, key := range keys {
		childSchema, ok := toStringAnyMap(props[key])
		if !ok {
			continue
		}
		if childDefault, childHasDefault := schemaDefaultValue(childSchema); childHasDefault {
			out[key] = childDefault
		}
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

func schemaType(schema map[string]any) string {
	if raw, ok := schema["type"]; ok {
		if s, ok := raw.(string); ok {
			return strings.TrimSpace(strings.ToLower(s))
		}
		if list, ok := toAnySlice(raw); ok {
			for _, item := range list {
				if s, ok := item.(string); ok && strings.TrimSpace(s) != "" && strings.TrimSpace(strings.ToLower(s)) != "null" {
					return strings.TrimSpace(strings.ToLower(s))
				}
			}
		}
	}

	if _, ok := toStringAnyMap(schema["properties"]); ok {
		return "object"
	}
	if _, ok := toStringAnyMap(schema["items"]); ok {
		return "array"
	}
	return ""
}

func coerceAndValidate(schema map[string]any, raw any) (any, error) {
	coerced, err := coerceBySchemaType(schema, raw)
	if err != nil {
		return nil, err
	}
	if err := validateEnum(schema, coerced); err != nil {
		return nil, err
	}
	return coerced, nil
}

func coerceBySchemaType(schema map[string]any, raw any) (any, error) {
	typ := schemaType(schema)
	if typ == "" {
		return copyAny(raw), nil
	}

	if typ == "object" {
		obj, ok := toStringAnyMap(raw)
		if !ok {
			return nil, fmt.Errorf("must be an object")
		}
		return coerceObject(schema, obj)
	}
	if typ == "array" {
		itemsSchema, _ := toStringAnyMap(schema["items"])
		values, ok := toAnySlice(raw)
		if !ok {
			return nil, fmt.Errorf("must be an array")
		}
		out := make([]any, 0, len(values))
		for index, item := range values {
			if itemsSchema == nil {
				out = append(out, copyAny(item))
				continue
			}
			coercedItem, err := coerceAndValidate(itemsSchema, item)
			if err != nil {
				return nil, fmt.Errorf("items[%d]: %w", index, err)
			}
			out = append(out, coercedItem)
		}
		return out, nil
	}
	if typ == "string" {
		if value, ok := raw.(string); ok {
			return value, nil
		}
		return fmt.Sprintf("%v", raw), nil
	}
	if typ == "boolean" {
		switch value := raw.(type) {
		case bool:
			return value, nil
		case string:
			parsed, err := strconv.ParseBool(strings.TrimSpace(value))
			if err != nil {
				return nil, fmt.Errorf("must be a boolean")
			}
			return parsed, nil
		default:
			return nil, fmt.Errorf("must be a boolean")
		}
	}
	if typ == "integer" {
		value, err := toInt64(raw)
		if err != nil {
			return nil, fmt.Errorf("must be an integer")
		}
		return value, nil
	}
	if typ == "number" {
		value, err := toFloat64(raw)
		if err != nil {
			return nil, fmt.Errorf("must be a number")
		}
		return value, nil
	}
	return copyAny(raw), nil
}

func coerceObject(schema map[string]any, raw map[string]any) (map[string]any, error) {
	out := map[string]any{}
	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := raw[key]
		childSchema, allowed, err := childSchemaForProperty(schema, key)
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, fmt.Errorf("field %q is not allowed", key)
		}
		coerced, err := coerceAndValidate(childSchema, value)
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", key, err)
		}
		out[key] = coerced
	}

	required := schemaRequiredFields(schema)
	for _, key := range required {
		if _, ok := out[key]; ok {
			continue
		}
		childSchema, ok, err := childSchemaForProperty(schema, key)
		if err != nil {
			return nil, err
		}
		if ok {
			if defValue, hasDefault := schemaDefaultValue(childSchema); hasDefault {
				out[key] = defValue
				continue
			}
		}
		return nil, fmt.Errorf("missing required field %q", key)
	}

	return out, nil
}

func validateEnum(schema map[string]any, value any) error {
	enumValues, ok := toAnySlice(schema["enum"])
	if !ok || len(enumValues) == 0 {
		return nil
	}
	for _, candidate := range enumValues {
		if reflect.DeepEqual(value, candidate) {
			return nil
		}
	}
	return fmt.Errorf("must be one of %v", enumValues)
}

func schemaRequiredFields(schema map[string]any) []string {
	required, ok := toAnySlice(schema["required"])
	if !ok || len(required) == 0 {
		return nil
	}
	out := make([]string, 0, len(required))
	for _, item := range required {
		s, ok := item.(string)
		if !ok {
			continue
		}
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func toInt64(v any) (int64, error) {
	switch value := v.(type) {
	case int:
		return int64(value), nil
	case int8:
		return int64(value), nil
	case int16:
		return int64(value), nil
	case int32:
		return int64(value), nil
	case int64:
		return value, nil
	case uint:
		if uint64(value) > math.MaxInt64 {
			return 0, fmt.Errorf("out of range")
		}
		return int64(value), nil
	case uint8:
		return int64(value), nil
	case uint16:
		return int64(value), nil
	case uint32:
		return int64(value), nil
	case uint64:
		if value > math.MaxInt64 {
			return 0, fmt.Errorf("out of range")
		}
		return int64(value), nil
	case float32:
		if math.Trunc(float64(value)) != float64(value) {
			return 0, fmt.Errorf("not an integer")
		}
		return int64(value), nil
	case float64:
		if math.Trunc(value) != value {
			return 0, fmt.Errorf("not an integer")
		}
		return int64(value), nil
	case string:
		i, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil {
			return 0, err
		}
		return i, nil
	default:
		return 0, fmt.Errorf("unsupported type %T", v)
	}
}

func toFloat64(v any) (float64, error) {
	switch value := v.(type) {
	case int:
		return float64(value), nil
	case int8:
		return float64(value), nil
	case int16:
		return float64(value), nil
	case int32:
		return float64(value), nil
	case int64:
		return float64(value), nil
	case uint:
		return float64(value), nil
	case uint8:
		return float64(value), nil
	case uint16:
		return float64(value), nil
	case uint32:
		return float64(value), nil
	case uint64:
		return float64(value), nil
	case float32:
		return float64(value), nil
	case float64:
		return value, nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			return 0, err
		}
		return f, nil
	default:
		return 0, fmt.Errorf("unsupported type %T", v)
	}
}

func setJSONPointer(root map[string]any, pointer string, value any) error {
	if strings.TrimSpace(pointer) == "" || pointer == "/" {
		obj, ok := toStringAnyMap(value)
		if !ok {
			return fmt.Errorf("root value must be an object")
		}
		for key := range root {
			delete(root, key)
		}
		for key, item := range obj {
			root[key] = copyAny(item)
		}
		return nil
	}
	if !strings.HasPrefix(pointer, "/") {
		return fmt.Errorf("invalid JSON pointer %q", pointer)
	}
	tokens := strings.Split(pointer[1:], "/")
	current := root
	for i, token := range tokens {
		key := unescapeJSONPointerToken(token)
		if i == len(tokens)-1 {
			current[key] = copyAny(value)
			return nil
		}
		next, ok := toStringAnyMap(current[key])
		if !ok || next == nil {
			next = map[string]any{}
			current[key] = next
		}
		current = next
	}
	return nil
}

func escapeJSONPointerToken(in string) string {
	return strings.ReplaceAll(strings.ReplaceAll(in, "~", "~0"), "/", "~1")
}

func unescapeJSONPointerToken(in string) string {
	return strings.ReplaceAll(strings.ReplaceAll(in, "~1", "/"), "~0", "~")
}

func toStringAnyMap(v any) (map[string]any, bool) {
	switch typed := v.(type) {
	case map[string]any:
		return typed, true
	default:
		return nil, false
	}
}

func toAnySlice(v any) ([]any, bool) {
	switch typed := v.(type) {
	case []any:
		return typed, true
	default:
		rv := reflect.ValueOf(v)
		if !rv.IsValid() || rv.Kind() != reflect.Slice {
			return nil, false
		}
		ret := make([]any, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			ret = append(ret, rv.Index(i).Interface())
		}
		return ret, true
	}
}

func copyStringAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	ret := make(map[string]any, len(in))
	for key, value := range in {
		ret[key] = copyAny(value)
	}
	return ret
}

func copyAny(in any) any {
	switch typed := in.(type) {
	case map[string]any:
		return copyStringAnyMap(typed)
	case []any:
		ret := make([]any, 0, len(typed))
		for _, value := range typed {
			ret = append(ret, copyAny(value))
		}
		return ret
	default:
		return typed
	}
}

func (l SourceLayer) String() string {
	return string(l)
}
