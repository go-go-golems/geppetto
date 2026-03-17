package scopedjs

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dop251/goja"
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

func TestRegisterPrebuiltSerializesConcurrentSharedRuntimeCalls(t *testing.T) {
	spec := EnvironmentSpec[struct{}, struct{}]{
		RuntimeLabel: "serialized",
		Tool: ToolDefinitionSpec{
			Name: "eval_serialized",
			Description: ToolDescription{
				Summary: "Evaluate JavaScript inside a serialized shared runtime",
			},
		},
		DefaultEval: DefaultEvalOptions(),
		Describe: func() (EnvironmentManifest, error) {
			return EnvironmentManifest{
				Globals: []GlobalDoc{{Name: "phase", Type: "string"}},
				Helpers: []HelperDoc{{Signature: "waitForRelease()"}},
			}, nil
		},
		Configure: func(ctx context.Context, b *Builder, _ struct{}) (struct{}, error) {
			if err := b.AddBootstrapSource("serialization.js", `
globalThis.phase = "idle";
globalThis.waitForRelease = function() {
  globalThis.phase = "running";
  return new Promise((resolve) => {
    globalThis.__releaseWait = function() {
      globalThis.phase = "done";
      resolve("released");
    };
  });
};
`); err != nil {
				return struct{}{}, err
			}
			return struct{}{}, nil
		},
	}

	handle, err := BuildRuntime(context.Background(), spec, struct{}{})
	if err != nil {
		t.Fatalf("BuildRuntime failed: %v", err)
	}
	defer func() { _ = handle.Cleanup() }()

	reg := tools.NewInMemoryToolRegistry()
	if err := RegisterPrebuilt(reg, spec, handle, EvalOptionOverrides{}); err != nil {
		t.Fatalf("RegisterPrebuilt failed: %v", err)
	}
	def, err := reg.GetTool("eval_serialized")
	if err != nil {
		t.Fatalf("GetTool failed: %v", err)
	}

	type evalResult struct {
		out EvalOutput
		err error
	}

	firstDone := make(chan evalResult, 1)
	go func() {
		ret, err := def.Function.ExecuteWithContext(context.Background(), []byte(`{"code":"await waitForRelease(); console.log(\"first\"); return { phase }; "}`))
		if err != nil {
			firstDone <- evalResult{err: err}
			return
		}
		out, ok := ret.(EvalOutput)
		if !ok {
			firstDone <- evalResult{err: fmt.Errorf("expected EvalOutput, got %T", ret)}
			return
		}
		firstDone <- evalResult{out: out}
	}()

	waitForSharedPhase(t, handle.Runtime, "running")

	secondDone := make(chan evalResult, 1)
	go func() {
		ret, err := def.Function.ExecuteWithContext(context.Background(), []byte(`{"code":"console.log(\"second\"); return { phase }; "}`))
		if err != nil {
			secondDone <- evalResult{err: err}
			return
		}
		out, ok := ret.(EvalOutput)
		if !ok {
			secondDone <- evalResult{err: fmt.Errorf("expected EvalOutput, got %T", ret)}
			return
		}
		secondDone <- evalResult{out: out}
	}()

	select {
	case result := <-secondDone:
		t.Fatalf("expected second call to block until first finished, got early result: %#v / %v", result.out, result.err)
	case <-time.After(20 * time.Millisecond):
	}

	releaseSharedDeferred(t, handle.Runtime)

	first := <-firstDone
	if first.err != nil {
		t.Fatalf("first call failed: %v", first.err)
	}
	second := <-secondDone
	if second.err != nil {
		t.Fatalf("second call failed: %v", second.err)
	}

	firstResult, ok := first.out.Result.(map[string]any)
	if !ok {
		t.Fatalf("expected first result map, got %T", first.out.Result)
	}
	if got := fmt.Sprint(firstResult["phase"]); got != "done" {
		t.Fatalf("expected first phase done, got %#v", firstResult)
	}
	if len(first.out.Console) != 1 || first.out.Console[0].Text != "first" {
		t.Fatalf("unexpected first console: %#v", first.out.Console)
	}

	secondResult, ok := second.out.Result.(map[string]any)
	if !ok {
		t.Fatalf("expected second result map, got %T", second.out.Result)
	}
	if got := fmt.Sprint(secondResult["phase"]); got != "done" {
		t.Fatalf("expected second call to observe serialized done phase, got %#v", secondResult)
	}
	if len(second.out.Console) != 1 || second.out.Console[0].Text != "second" {
		t.Fatalf("unexpected second console: %#v", second.out.Console)
	}
}

func waitForSharedPhase(t *testing.T, rt *gojengine.Runtime, want string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if got := readSharedPhase(t, rt); got == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for phase %q, last=%q", want, readSharedPhase(t, rt))
}

func readSharedPhase(t *testing.T, rt *gojengine.Runtime) string {
	t.Helper()
	ret, err := rt.Owner.Call(context.Background(), "test.read-phase", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.Get("phase").String(), nil
	})
	if err != nil {
		t.Fatalf("read phase failed: %v", err)
	}
	phase, ok := ret.(string)
	if !ok {
		t.Fatalf("expected string phase, got %T", ret)
	}
	return phase
}

func releaseSharedDeferred(t *testing.T, rt *gojengine.Runtime) {
	t.Helper()
	_, err := rt.Owner.Call(context.Background(), "test.release-phase", func(_ context.Context, vm *goja.Runtime) (any, error) {
		fn, ok := goja.AssertFunction(vm.Get("__releaseWait"))
		if !ok {
			return nil, fmt.Errorf("__releaseWait is not callable")
		}
		_, err := fn(goja.Undefined())
		return nil, err
	})
	if err != nil {
		t.Fatalf("release deferred failed: %v", err)
	}
}
