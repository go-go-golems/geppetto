package tools

import (
	"context"
	"strings"
	"testing"
)

type testContextKey string

type testInput struct {
	Value int `json:"value"`
}

func TestToolFuncExecute_SupportsContextAndInputSignature(t *testing.T) {
	def, err := NewToolFromFunc(
		"ctx_input_tool",
		"test",
		func(ctx context.Context, in testInput) (int, error) {
			if ctx == nil {
				t.Fatalf("ctx should not be nil")
			}
			return in.Value + 1, nil
		},
	)
	if err != nil {
		t.Fatalf("NewToolFromFunc failed: %v", err)
	}

	out, err := def.Function.Execute([]byte(`{"value":41}`))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	v, ok := out.(int)
	if !ok {
		t.Fatalf("expected int result, got %T", out)
	}
	if v != 42 {
		t.Fatalf("expected 42, got %d", v)
	}
}

func TestToolFuncExecuteWithContext_PassesProvidedContext(t *testing.T) {
	key := testContextKey("tool-test-key")
	def, err := NewToolFromFunc(
		"ctx_passthrough_tool",
		"test",
		func(ctx context.Context, in testInput) (bool, error) {
			if ctx == nil {
				return false, nil
			}
			v, _ := ctx.Value(key).(string)
			return v == "ok" && in.Value == 7, nil
		},
	)
	if err != nil {
		t.Fatalf("NewToolFromFunc failed: %v", err)
	}

	ctx := context.WithValue(context.Background(), key, "ok")
	out, err := def.Function.ExecuteWithContext(ctx, []byte(`{"value":7}`))
	if err != nil {
		t.Fatalf("ExecuteWithContext failed: %v", err)
	}

	v, ok := out.(bool)
	if !ok {
		t.Fatalf("expected bool result, got %T", out)
	}
	if !v {
		t.Fatalf("expected true result")
	}
}

func TestNewToolFromFuncRejectsInterfaceTypedInputs(t *testing.T) {
	type badInput struct {
		Args []any `json:"args,omitempty"`
	}

	_, err := NewToolFromFunc(
		"bad_tool",
		"test",
		func(in badInput) (string, error) {
			return "ok", nil
		},
	)
	if err == nil {
		t.Fatalf("expected interface-typed tool input to be rejected")
	}
	if !strings.Contains(err.Error(), "unsupported interface{} array items") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "input.Args[]") {
		t.Fatalf("expected field path in error, got: %v", err)
	}
}

func TestNewToolFromFuncAllowsDynamicObjectFields(t *testing.T) {
	type scopedInput struct {
		Code  string         `json:"code"`
		Input map[string]any `json:"input,omitempty"`
	}

	def, err := NewToolFromFunc(
		"dynamic_object_tool",
		"test",
		func(in scopedInput) (string, error) {
			return in.Code, nil
		},
	)
	if err != nil {
		t.Fatalf("expected dynamic object field to remain supported, got: %v", err)
	}
	if def == nil || def.Parameters == nil {
		t.Fatalf("expected generated schema")
	}
}

func TestNewToolFromFuncSkipsJSONOmittedFields(t *testing.T) {
	type hiddenJSONFieldInput struct {
		Visible string `json:"visible"`
		Hidden  []any  `json:"-"`
	}

	def, err := NewToolFromFunc(
		"json_omitted_tool",
		"test",
		func(in hiddenJSONFieldInput) (string, error) {
			return in.Visible, nil
		},
	)
	if err != nil {
		t.Fatalf("expected json-omitted field to be ignored, got: %v", err)
	}
	if def == nil || def.Parameters == nil {
		t.Fatalf("expected generated schema")
	}
}

func TestNewToolFromFuncSkipsJSONSchemaOmittedFields(t *testing.T) {
	type hiddenSchemaFieldInput struct {
		Visible string `json:"visible"`
		Hidden  []any  `json:"hidden,omitempty" jsonschema:"-"`
	}

	def, err := NewToolFromFunc(
		"jsonschema_omitted_tool",
		"test",
		func(in hiddenSchemaFieldInput) (string, error) {
			return in.Visible, nil
		},
	)
	if err != nil {
		t.Fatalf("expected jsonschema-omitted field to be ignored, got: %v", err)
	}
	if def == nil || def.Parameters == nil {
		t.Fatalf("expected generated schema")
	}
}
