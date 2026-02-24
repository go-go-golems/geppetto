package middlewarecfg

import (
	"context"
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
