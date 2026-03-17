package scopedjs

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	gojengine "github.com/go-go-golems/go-go-goja/engine"
)

type scopeKey struct{}

func TestRegisterPrebuiltAndLazyRegistrar(t *testing.T) {
	type scope struct {
		Prefix string
	}

	var buildCount int
	spec := EnvironmentSpec[scope, struct{}]{
		RuntimeLabel: "dbserver",
		Tool: ToolDefinitionSpec{
			Name: "eval_dbserver",
			Description: ToolDescription{
				Summary: "Evaluate JavaScript inside the scoped dbserver runtime",
			},
			Tags:    []string{"javascript", "scopedjs"},
			Version: "v1",
		},
		DefaultEval: DefaultEvalOptions(),
		Describe: func() (EnvironmentManifest, error) {
			return EnvironmentManifest{
				Modules:        []ModuleDoc{{Name: "mathx"}},
				Globals:        []GlobalDoc{{Name: "prefix", Type: "string"}},
				BootstrapFiles: []string{"state.js"},
			}, nil
		},
		Configure: func(ctx context.Context, b *Builder, s scope) (struct{}, error) {
			buildCount++
			if err := b.AddNativeModule(mathModule{}); err != nil {
				return struct{}{}, err
			}
			if err := b.AddGlobal("prefix", func(ctx *gojengine.RuntimeContext) error {
				return ctx.VM.Set("prefix", s.Prefix)
			}, GlobalDoc{Type: "string"}); err != nil {
				return struct{}{}, err
			}
			if err := b.AddBootstrapSource("state.js", `globalThis.counter = 0;`); err != nil {
				return struct{}{}, err
			}
			return struct{}{}, nil
		},
	}

	handle, err := BuildRuntime(context.Background(), spec, scope{Prefix: "prebuilt"})
	if err != nil {
		t.Fatalf("BuildRuntime failed: %v", err)
	}
	defer func() { _ = handle.Cleanup() }()

	reg := tools.NewInMemoryToolRegistry()
	if err := RegisterPrebuilt(reg, spec, handle, EvalOptionOverrides{}); err != nil {
		t.Fatalf("RegisterPrebuilt failed: %v", err)
	}
	def, err := reg.GetTool("eval_dbserver")
	if err != nil {
		t.Fatalf("GetTool failed: %v", err)
	}
	if def.Parameters == nil {
		t.Fatalf("expected parameters schema")
	}
	codeSchema, ok := def.Parameters.Properties.Get("code")
	if !ok || codeSchema == nil || codeSchema.Type != "string" {
		t.Fatalf("expected code schema, got %#v", codeSchema)
	}
	if !reg.HasTool("eval_dbserver") {
		t.Fatalf("expected eval_dbserver to be registered")
	}
	if !strings.Contains(def.Description, "runtime state can persist across calls") {
		t.Fatalf("expected prebuilt description to describe shared runtime reuse, got %q", def.Description)
	}
	if !strings.Contains(def.Description, "Available modules: mathx.") {
		t.Fatalf("expected prebuilt description to include manifest modules, got %q", def.Description)
	}

	result, err := def.Function.ExecuteWithContext(context.Background(), []byte(`{"code":"const math = require(\"mathx\"); globalThis.counter += 1; console.log(prefix); return { sum: math.add(1, 2), prefix, counter: globalThis.counter }; "}`))
	if err != nil {
		t.Fatalf("ExecuteWithContext failed: %v", err)
	}
	out, ok := result.(EvalOutput)
	if !ok {
		t.Fatalf("expected EvalOutput, got %T", result)
	}
	if out.Error != "" {
		t.Fatalf("unexpected prebuilt output error: %#v", out)
	}
	got, ok := out.Result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", out.Result)
	}
	if fmt.Sprint(got["sum"]) != "3" || got["prefix"] != "prebuilt" || fmt.Sprint(got["counter"]) != "1" {
		t.Fatalf("unexpected prebuilt result: %#v", got)
	}
	if len(out.Console) != 1 || out.Console[0].Text != "prebuilt" {
		t.Fatalf("unexpected console capture: %#v", out.Console)
	}

	lazyReg := tools.NewInMemoryToolRegistry()
	registrar := NewLazyRegistrar(spec, func(ctx context.Context) (scope, error) {
		v, _ := ctx.Value(scopeKey{}).(scope)
		return v, nil
	}, EvalOptionOverrides{})
	if err := registrar(lazyReg); err != nil {
		t.Fatalf("NewLazyRegistrar failed: %v", err)
	}
	lazyDef, err := lazyReg.GetTool("eval_dbserver")
	if err != nil {
		t.Fatalf("lazy GetTool failed: %v", err)
	}
	if !strings.Contains(lazyDef.Description, "Each call builds a fresh runtime from the resolved scope.") {
		t.Fatalf("expected lazy description to describe per-call runtime builds, got %q", lazyDef.Description)
	}
	for _, fragment := range []string{
		"Available modules: mathx.",
		"Available globals: prefix (string).",
		"Preloaded bootstrap files: state.js.",
	} {
		if !strings.Contains(lazyDef.Description, fragment) {
			t.Fatalf("expected lazy description to contain %q, got %q", fragment, lazyDef.Description)
		}
	}

	for _, tc := range []scope{{Prefix: "lazy-a"}, {Prefix: "lazy-b"}} {
		ret, err := lazyDef.Function.ExecuteWithContext(context.WithValue(context.Background(), scopeKey{}, tc), []byte(`{"code":"globalThis.counter += 1; return { prefix, counter: globalThis.counter }; "}`))
		if err != nil {
			t.Fatalf("lazy ExecuteWithContext failed: %v", err)
		}
		out, ok := ret.(EvalOutput)
		if !ok {
			t.Fatalf("expected EvalOutput, got %T", ret)
		}
		got, ok := out.Result.(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %T", out.Result)
		}
		if got["prefix"] != tc.Prefix || fmt.Sprint(got["counter"]) != "1" {
			t.Fatalf("unexpected lazy result for %+v: %#v", tc, got)
		}
	}
	if buildCount < 3 {
		t.Fatalf("expected runtime builds for prebuilt and lazy calls, got %d", buildCount)
	}
}
