package middlewarecfg

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	gepmiddleware "github.com/go-go-golems/geppetto/pkg/inference/middleware"
	gepprofiles "github.com/go-go-golems/geppetto/pkg/profiles"
)

type resolverTestDefinition struct {
	name   string
	schema map[string]any
}

func (d *resolverTestDefinition) Name() string {
	return d.name
}

func (d *resolverTestDefinition) ConfigJSONSchema() map[string]any {
	return d.schema
}

func (d *resolverTestDefinition) Build(context.Context, BuildDeps, any) (gepmiddleware.Middleware, error) {
	return nil, nil
}

type staticSource struct {
	name    string
	layer   SourceLayer
	payload map[string]any
}

func (s staticSource) Name() string {
	return s.name
}

func (s staticSource) Layer() SourceLayer {
	return s.layer
}

func (s staticSource) Payload(Definition, gepprofiles.MiddlewareUse) (map[string]any, bool, error) {
	if len(s.payload) == 0 {
		return nil, false, nil
	}
	return copyStringAnyMap(s.payload), true, nil
}

type failingSource struct {
	name  string
	layer SourceLayer
	err   error
}

func (s failingSource) Name() string {
	return s.name
}

func (s failingSource) Layer() SourceLayer {
	return s.layer
}

func (s failingSource) Payload(Definition, gepprofiles.MiddlewareUse) (map[string]any, bool, error) {
	if s.err == nil {
		return nil, false, nil
	}
	return nil, false, s.err
}

func TestResolver_AppliesCanonicalSourcePrecedence(t *testing.T) {
	resolver := NewResolver(
		staticSource{
			name:  "request",
			layer: SourceLayerRequest,
			payload: map[string]any{
				"threshold": 7,
			},
		},
		staticSource{
			name:  "profile",
			layer: SourceLayerProfile,
			payload: map[string]any{
				"threshold": 2,
			},
		},
		staticSource{
			name:  "env",
			layer: SourceLayerEnvironment,
			payload: map[string]any{
				"threshold": "5",
			},
		},
	)
	definition := &resolverTestDefinition{
		name: "agentmode",
		schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"threshold": map[string]any{
					"type": "integer",
				},
			},
		},
	}

	resolved, err := resolver.Resolve(definition, gepprofiles.MiddlewareUse{Name: "agentmode"})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	value, ok := resolved.Config["threshold"].(int64)
	if !ok {
		t.Fatalf("expected threshold as int64, got %T", resolved.Config["threshold"])
	}
	if value != 7 {
		t.Fatalf("expected request layer to win with value 7, got %d", value)
	}
}

func TestResolver_AppliesSchemaDefaultsAsFirstLayer(t *testing.T) {
	resolver := NewResolver()
	definition := &resolverTestDefinition{
		name: "agentmode",
		schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"mode": map[string]any{
					"type":    "string",
					"default": "safe",
				},
				"retries": map[string]any{
					"type":    "integer",
					"default": 2,
				},
			},
		},
	}

	resolved, err := resolver.Resolve(definition, gepprofiles.MiddlewareUse{Name: "agentmode"})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if got := resolved.Config["mode"]; got != "safe" {
		t.Fatalf("expected mode default 'safe', got %#v", got)
	}
	if got := resolved.Config["retries"]; got != int64(2) {
		t.Fatalf("expected retries default 2, got %#v", got)
	}
}

func TestResolver_RejectsMissingRequiredFieldAfterResolution(t *testing.T) {
	resolver := NewResolver()
	definition := &resolverTestDefinition{
		name: "agentmode",
		schema: map[string]any{
			"type": "object",
			"required": []any{
				"mode",
			},
			"properties": map[string]any{
				"mode": map[string]any{
					"type": "string",
				},
			},
		},
	}

	_, err := resolver.Resolve(definition, gepprofiles.MiddlewareUse{Name: "agentmode"})
	if err == nil {
		t.Fatalf("expected required-field validation error")
	}
	if !strings.Contains(err.Error(), "missing required field") {
		t.Fatalf("expected required field error, got %v", err)
	}
}

func TestResolver_CoercesPerWriteAndValidatesType(t *testing.T) {
	resolver := NewResolver(staticSource{
		name:  "env",
		layer: SourceLayerEnvironment,
		payload: map[string]any{
			"threshold": "12",
		},
	})
	definition := &resolverTestDefinition{
		name: "agentmode",
		schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"threshold": map[string]any{
					"type": "integer",
				},
			},
		},
	}
	resolved, err := resolver.Resolve(definition, gepprofiles.MiddlewareUse{Name: "agentmode"})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if got := resolved.Config["threshold"]; got != int64(12) {
		t.Fatalf("expected coerced threshold 12, got %#v", got)
	}

	badResolver := NewResolver(staticSource{
		name:  "env",
		layer: SourceLayerEnvironment,
		payload: map[string]any{
			"threshold": "not-a-number",
		},
	})
	_, err = badResolver.Resolve(definition, gepprofiles.MiddlewareUse{Name: "agentmode"})
	if err == nil {
		t.Fatalf("expected coercion error")
	}
	if !strings.Contains(err.Error(), "/threshold") {
		t.Fatalf("expected path context in coercion error, got %v", err)
	}
}

func TestResolver_ProducesDeterministicPathOrdering(t *testing.T) {
	resolver := NewResolver(staticSource{
		name:  "profile",
		layer: SourceLayerProfile,
		payload: map[string]any{
			"nested": map[string]any{
				"z": "last",
				"a": "first",
			},
			"enabled": true,
		},
	})
	definition := &resolverTestDefinition{
		name: "agentmode",
		schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"enabled": map[string]any{"type": "boolean"},
				"nested": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"a": map[string]any{"type": "string"},
						"z": map[string]any{"type": "string"},
					},
				},
			},
		},
	}

	resolved, err := resolver.Resolve(definition, gepprofiles.MiddlewareUse{Name: "agentmode"})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	want := []string{"/enabled", "/nested/a", "/nested/z"}
	if len(resolved.OrderedPaths) != len(want) {
		t.Fatalf("ordered path length mismatch: got=%d want=%d", len(resolved.OrderedPaths), len(want))
	}
	for i := range want {
		if resolved.OrderedPaths[i] != want[i] {
			t.Fatalf("ordered path mismatch at %d: got=%q want=%q", i, resolved.OrderedPaths[i], want[i])
		}
	}
}

func TestResolver_TraceHistoryFollowsSourcePrecedenceOrder(t *testing.T) {
	resolver := NewResolver(
		staticSource{
			name:  "profile",
			layer: SourceLayerProfile,
			payload: map[string]any{
				"threshold": 1,
			},
		},
		staticSource{
			name:  "flags",
			layer: SourceLayerFlags,
			payload: map[string]any{
				"threshold": 3,
			},
		},
		staticSource{
			name:  "request",
			layer: SourceLayerRequest,
			payload: map[string]any{
				"threshold": 5,
			},
		},
	)
	definition := &resolverTestDefinition{
		name: "agentmode",
		schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"threshold": map[string]any{"type": "integer"},
			},
		},
	}

	resolved, err := resolver.Resolve(definition, gepprofiles.MiddlewareUse{Name: "agentmode"})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	history := resolved.History("/threshold")
	if len(history) != 3 {
		t.Fatalf("expected 3 parse steps for /threshold, got %d", len(history))
	}
	if got, want := history[0].Source, "profile"; got != want {
		t.Fatalf("history source mismatch at 0: got=%q want=%q", got, want)
	}
	if got, want := history[1].Source, "flags"; got != want {
		t.Fatalf("history source mismatch at 1: got=%q want=%q", got, want)
	}
	if got, want := history[2].Source, "request"; got != want {
		t.Fatalf("history source mismatch at 2: got=%q want=%q", got, want)
	}

	latest, ok := resolved.LatestValue("/threshold")
	if !ok {
		t.Fatalf("expected latest value for /threshold")
	}
	if got, want := latest.(int64), int64(5); got != want {
		t.Fatalf("latest value mismatch: got=%d want=%d", got, want)
	}
}

func TestResolver_TraceIncludesRawAndCoercedValues(t *testing.T) {
	resolver := NewResolver(staticSource{
		name:  "environment",
		layer: SourceLayerEnvironment,
		payload: map[string]any{
			"threshold": "42",
		},
	})
	definition := &resolverTestDefinition{
		name: "agentmode",
		schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"threshold": map[string]any{"type": "integer"},
			},
		},
	}

	resolved, err := resolver.Resolve(definition, gepprofiles.MiddlewareUse{Name: "agentmode"})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	history := resolved.History("/threshold")
	if len(history) != 1 {
		t.Fatalf("expected one parse step, got %d", len(history))
	}
	step := history[0]
	if got, want := step.Raw, "42"; got != want {
		t.Fatalf("raw value mismatch: got=%#v want=%#v", got, want)
	}
	if got, want := step.Value, int64(42); got != want {
		t.Fatalf("coerced value mismatch: got=%#v want=%#v", got, want)
	}
	schemaTypeRaw, ok := step.Metadata["schema_type"]
	if !ok {
		t.Fatalf("expected schema_type metadata")
	}
	if got, want := schemaTypeRaw, "integer"; got != want {
		t.Fatalf("schema_type metadata mismatch: got=%#v want=%#v", got, want)
	}
}

func TestResolvedConfig_MarshalDebugPayloadIsDeterministic(t *testing.T) {
	resolver := NewResolver(
		staticSource{
			name:  "profile",
			layer: SourceLayerProfile,
			payload: map[string]any{
				"threshold": "2",
				"mode":      "safe",
			},
		},
		staticSource{
			name:  "request",
			layer: SourceLayerRequest,
			payload: map[string]any{
				"threshold": 7,
			},
		},
	)
	definition := &resolverTestDefinition{
		name: "agentmode",
		schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"threshold": map[string]any{"type": "integer"},
				"mode":      map[string]any{"type": "string"},
			},
		},
	}

	resolved, err := resolver.Resolve(definition, gepprofiles.MiddlewareUse{Name: "agentmode", ID: "primary"})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	payloadA, err := resolved.MarshalDebugPayload()
	if err != nil {
		t.Fatalf("MarshalDebugPayload returned error: %v", err)
	}
	payloadB, err := resolved.MarshalDebugPayload()
	if err != nil {
		t.Fatalf("MarshalDebugPayload returned error on second call: %v", err)
	}
	if !bytes.Equal(payloadA, payloadB) {
		t.Fatalf("expected deterministic debug payload JSON")
	}

	var decoded DebugPayload
	if err := json.Unmarshal(payloadA, &decoded); err != nil {
		t.Fatalf("unmarshal debug payload failed: %v", err)
	}
	if len(decoded.Paths) == 0 {
		t.Fatalf("expected debug payload paths")
	}
	if got, want := decoded.Paths[0].Path, "/mode"; got != want {
		t.Fatalf("expected first path %q, got %q", want, got)
	}
	if got, want := decoded.Paths[1].Path, "/threshold"; got != want {
		t.Fatalf("expected second path %q, got %q", want, got)
	}
}

func TestResolver_ErrorIncludesMiddlewareSourceAndPathContext(t *testing.T) {
	resolver := NewResolver(staticSource{
		name:  "env",
		layer: SourceLayerEnvironment,
		payload: map[string]any{
			"threshold": "oops",
		},
	})
	definition := &resolverTestDefinition{
		name: "agentmode",
		schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"threshold": map[string]any{"type": "integer"},
			},
		},
	}

	_, err := resolver.Resolve(definition, gepprofiles.MiddlewareUse{Name: "agentmode", ID: "primary"})
	if err == nil {
		t.Fatalf("expected resolver error")
	}
	errText := err.Error()
	for _, expected := range []string{
		"agentmode#primary",
		"/threshold",
		"env[environment]",
	} {
		if !strings.Contains(errText, expected) {
			t.Fatalf("expected error to contain %q, got: %v", expected, err)
		}
	}
}

func TestResolver_SourcePayloadErrorIncludesMiddlewareAndSourceContext(t *testing.T) {
	resolver := NewResolver(failingSource{
		name:  "request",
		layer: SourceLayerRequest,
		err:   errors.New("boom"),
	})
	definition := &resolverTestDefinition{
		name: "agentmode",
		schema: map[string]any{
			"type": "object",
		},
	}

	_, err := resolver.Resolve(definition, gepprofiles.MiddlewareUse{Name: "agentmode", ID: "primary"})
	if err == nil {
		t.Fatalf("expected resolver error")
	}
	errText := err.Error()
	for _, expected := range []string{
		"agentmode#primary",
		"request[request]",
		"boom",
	} {
		if !strings.Contains(errText, expected) {
			t.Fatalf("expected error to contain %q, got: %v", expected, err)
		}
	}
}

func TestCanonicalOrderedSources_DeterministicByLayerThenName(t *testing.T) {
	sources := []Source{
		staticSource{name: "z-profile", layer: SourceLayerProfile},
		staticSource{name: "a-profile", layer: SourceLayerProfile},
		staticSource{name: "b-request", layer: SourceLayerRequest},
		staticSource{name: "a-request", layer: SourceLayerRequest},
	}
	ordered, err := canonicalOrderedSources(sources)
	if err != nil {
		t.Fatalf("canonicalOrderedSources returned error: %v", err)
	}

	got := make([]string, 0, len(ordered))
	for _, source := range ordered {
		got = append(got, string(source.source.Layer())+":"+source.source.Name())
	}
	want := []string{
		"profile:a-profile",
		"profile:z-profile",
		"request:a-request",
		"request:b-request",
	}
	if len(got) != len(want) {
		t.Fatalf("ordered source length mismatch: got=%d want=%d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ordered source mismatch at %d: got=%q want=%q", i, got[i], want[i])
		}
	}
}
