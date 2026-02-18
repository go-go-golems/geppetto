package tools

import (
	"context"
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
