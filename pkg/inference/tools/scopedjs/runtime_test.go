package scopedjs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dop251/goja"
	gojengine "github.com/go-go-golems/go-go-goja/engine"
	ggjmodules "github.com/go-go-golems/go-go-goja/modules"
)

type mathModule struct{}

func (mathModule) Name() string { return "mathx" }

func (mathModule) Doc() string { return "Simple math helpers." }

func (mathModule) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
	exports := moduleObj.Get("exports").(*goja.Object)
	ggjmodules.SetExport(exports, "mathx", "add", func(a, b int) int { return a + b })
}

func TestBuildRuntimeLoadsGlobalsAndBootstrap(t *testing.T) {
	bootstrapFile := filepath.Join(t.TempDir(), "bootstrap.js")
	if err := os.WriteFile(bootstrapFile, []byte(`function fromFile() { return "from-file"; }`), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	spec := EnvironmentSpec[struct{}, string]{
		RuntimeLabel: "demo",
		Tool:         ToolDefinitionSpec{Name: "eval_demo"},
		DefaultEval:  DefaultEvalOptions(),
		Configure: func(ctx context.Context, b *Builder, _ struct{}) (string, error) {
			if err := b.AddNativeModule(mathModule{}); err != nil {
				return "", err
			}
			if err := b.AddGlobal("db", func(ctx *gojengine.RuntimeContext) error {
				obj := ctx.VM.NewObject()
				if err := obj.Set("answer", 42); err != nil {
					return err
				}
				return ctx.VM.Set("db", obj)
			}, GlobalDoc{Type: "DatabaseClient"}); err != nil {
				return "", err
			}
			if err := b.AddBootstrapSource("helpers.js", `function greet(name) { return "hi " + name; }`); err != nil {
				return "", err
			}
			if err := b.AddBootstrapFile(bootstrapFile); err != nil {
				return "", err
			}
			return "ready", nil
		},
	}

	handle, err := BuildRuntime(context.Background(), spec, struct{}{})
	if err != nil {
		t.Fatalf("BuildRuntime failed: %v", err)
	}
	defer func() { _ = handle.Cleanup() }()

	if handle.Meta != "ready" {
		t.Fatalf("expected meta ready, got %q", handle.Meta)
	}
	if len(handle.Manifest.Modules) != 1 || handle.Manifest.Modules[0].Name != "mathx" {
		t.Fatalf("unexpected manifest modules: %#v", handle.Manifest.Modules)
	}

	out, err := RunEval(context.Background(), handle.Runtime, EvalInput{
		Code: `
const math = require("mathx");
return { sum: math.add(2, 3), greet: greet("bob"), file: fromFile(), answer: db.answer };
`,
	}, DefaultEvalOptions())
	if err != nil {
		t.Fatalf("RunEval failed: %v", err)
	}
	if out.Error != "" {
		t.Fatalf("unexpected eval error: %s", out.Error)
	}
	got, ok := out.Result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", out.Result)
	}
	if fmt.Sprint(got["sum"]) != "5" || got["greet"] != "hi bob" || got["file"] != "from-file" || fmt.Sprint(got["answer"]) != "42" {
		t.Fatalf("unexpected eval result: %#v", got)
	}
}

func TestRunEvalCapturesConsoleAndPromiseResolution(t *testing.T) {
	spec := EnvironmentSpec[struct{}, struct{}]{
		RuntimeLabel: "demo",
		Tool:         ToolDefinitionSpec{Name: "eval_demo"},
		DefaultEval:  DefaultEvalOptions(),
		Configure: func(ctx context.Context, b *Builder, _ struct{}) (struct{}, error) {
			if err := b.AddBootstrapSource("helpers.js", `async function delayed() { return 9; }`); err != nil {
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

	out, err := RunEval(context.Background(), handle.Runtime, EvalInput{
		Code: `
console.log("hello", 1);
return await delayed();
`,
	}, DefaultEvalOptions())
	if err != nil {
		t.Fatalf("RunEval failed: %v", err)
	}
	if out.Error != "" {
		t.Fatalf("unexpected eval error: %s", out.Error)
	}
	if fmt.Sprint(out.Result) != "9" {
		t.Fatalf("expected promise result 9, got %#v", out.Result)
	}
	if len(out.Console) != 1 || out.Console[0].Text != "hello 1" {
		t.Fatalf("unexpected console capture: %#v", out.Console)
	}
}

func TestRunEvalReturnsStructuredErrorForRejectionAndTimeout(t *testing.T) {
	spec := EnvironmentSpec[struct{}, struct{}]{
		RuntimeLabel: "demo",
		Tool:         ToolDefinitionSpec{Name: "eval_demo"},
		DefaultEval:  DefaultEvalOptions(),
		Configure: func(ctx context.Context, b *Builder, _ struct{}) (struct{}, error) {
			return struct{}{}, nil
		},
	}

	handle, err := BuildRuntime(context.Background(), spec, struct{}{})
	if err != nil {
		t.Fatalf("BuildRuntime failed: %v", err)
	}
	defer func() { _ = handle.Cleanup() }()

	rejected, err := RunEval(context.Background(), handle.Runtime, EvalInput{
		Code: `await Promise.reject("boom")`,
	}, DefaultEvalOptions())
	if err != nil {
		t.Fatalf("RunEval failed: %v", err)
	}
	if rejected.Error == "" || !strings.Contains(rejected.Error, "promise rejected: boom") {
		t.Fatalf("expected promise rejection in output, got %#v", rejected)
	}

	jsRejected, err := RunEval(context.Background(), handle.Runtime, EvalInput{
		Code: `await Promise.reject(new Error("boom"))`,
	}, DefaultEvalOptions())
	if err != nil {
		t.Fatalf("RunEval failed: %v", err)
	}
	if jsRejected.Error == "" || !strings.Contains(jsRejected.Error, "boom") || strings.Contains(jsRejected.Error, "map[]") {
		t.Fatalf("expected javascript error rejection text, got %#v", jsRejected)
	}

	thrown, err := RunEval(context.Background(), handle.Runtime, EvalInput{
		Code: `throw new Error("boom")`,
	}, DefaultEvalOptions())
	if err != nil {
		t.Fatalf("RunEval failed: %v", err)
	}
	if thrown.Error == "" || !strings.Contains(thrown.Error, "boom") || strings.Contains(thrown.Error, "map[]") {
		t.Fatalf("expected thrown javascript error text, got %#v", thrown)
	}

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	timedOut, err := RunEval(timeoutCtx, handle.Runtime, EvalInput{
		Code: `await new Promise(() => {})`,
	}, EvalOptions{
		Timeout:        100 * time.Millisecond,
		MaxOutputChars: 512,
		CaptureConsole: true,
	})
	if err != nil {
		t.Fatalf("RunEval failed: %v", err)
	}
	if timedOut.Error == "" || !strings.Contains(timedOut.Error, context.DeadlineExceeded.Error()) {
		t.Fatalf("expected timeout error, got %#v", timedOut)
	}
}

func TestRunEvalPreservesReturnedAndConsoleLoggedJavaScriptErrors(t *testing.T) {
	spec := EnvironmentSpec[struct{}, struct{}]{
		RuntimeLabel: "demo",
		Tool:         ToolDefinitionSpec{Name: "eval_demo"},
		DefaultEval:  DefaultEvalOptions(),
		Configure: func(ctx context.Context, b *Builder, _ struct{}) (struct{}, error) {
			return struct{}{}, nil
		},
	}

	handle, err := BuildRuntime(context.Background(), spec, struct{}{})
	if err != nil {
		t.Fatalf("BuildRuntime failed: %v", err)
	}
	defer func() { _ = handle.Cleanup() }()

	returned, err := RunEval(context.Background(), handle.Runtime, EvalInput{
		Code: `return new Error("boom")`,
	}, DefaultEvalOptions())
	if err != nil {
		t.Fatalf("RunEval failed: %v", err)
	}
	if returned.Error != "" {
		t.Fatalf("unexpected eval error: %#v", returned)
	}
	if got := fmt.Sprint(returned.Result); !strings.Contains(got, "boom") || strings.Contains(got, "map[]") {
		t.Fatalf("expected returned javascript error text, got %#v", returned)
	}

	logged, err := RunEval(context.Background(), handle.Runtime, EvalInput{
		Code: `
console.error(new Error("boom"));
return 1;
`,
	}, DefaultEvalOptions())
	if err != nil {
		t.Fatalf("RunEval failed: %v", err)
	}
	if logged.Error != "" {
		t.Fatalf("unexpected eval error: %#v", logged)
	}
	if len(logged.Console) != 1 || !strings.Contains(logged.Console[0].Text, "boom") || strings.Contains(logged.Console[0].Text, "map[]") {
		t.Fatalf("expected console-captured javascript error text, got %#v", logged.Console)
	}
}
