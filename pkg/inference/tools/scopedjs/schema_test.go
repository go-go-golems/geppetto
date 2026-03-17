package scopedjs

import (
	"context"
	"testing"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	gojengine "github.com/go-go-golems/go-go-goja/engine"
	ggjmodules "github.com/go-go-golems/go-go-goja/modules"
)

type testNativeModule struct{}

func (testNativeModule) Name() string { return "fs-lite" }

func (testNativeModule) Doc() string { return "Lightweight file helper module." }

func (testNativeModule) Loader(*goja.Runtime, *goja.Object) {}

type testRuntimeInitializer struct{}

func (testRuntimeInitializer) ID() string { return "test-init" }

func (testRuntimeInitializer) InitRuntime(*gojengine.RuntimeContext) error { return nil }

var _ ggjmodules.NativeModule = (*testNativeModule)(nil)

func TestDefaultEvalOptions(t *testing.T) {
	opts := DefaultEvalOptions()
	if !opts.CaptureConsole {
		t.Fatalf("expected console capture to be enabled")
	}
	if opts.Timeout <= 0 {
		t.Fatalf("expected positive timeout")
	}
}

func TestResolveEvalOptionsSupportsExplicitFalseOverrides(t *testing.T) {
	base := EvalOptions{
		Timeout:        5 * time.Second,
		MaxOutputChars: 100,
		CaptureConsole: true,
	}
	timeout := 2 * time.Second
	maxOutput := 42
	captureConsole := false

	got := resolveEvalOptions(base, EvalOptionOverrides{
		Timeout:        &timeout,
		MaxOutputChars: &maxOutput,
		CaptureConsole: &captureConsole,
	})

	if got.Timeout != timeout {
		t.Fatalf("expected timeout %v, got %v", timeout, got.Timeout)
	}
	if got.MaxOutputChars != maxOutput {
		t.Fatalf("expected max output %d, got %d", maxOutput, got.MaxOutputChars)
	}
	if got.CaptureConsole {
		t.Fatalf("expected captureConsole override to force false")
	}
}

func TestBuilderValidationAndManifest(t *testing.T) {
	b := &Builder{}
	if err := b.AddModule("", func(_ *require.Registry) error { return nil }, ModuleDoc{}); err == nil {
		t.Fatalf("expected empty module name error")
	}
	if err := b.AddModule("db", nil, ModuleDoc{}); err == nil {
		t.Fatalf("expected nil module registrar error")
	}
	if err := b.AddModule("db", func(_ *require.Registry) error { return nil }, ModuleDoc{
		Description: "Database helpers",
		Exports:     []string{"query", "exec", "query"},
	}); err != nil {
		t.Fatalf("AddModule failed: %v", err)
	}
	if err := b.AddGlobal("db", nil, GlobalDoc{}); err == nil {
		t.Fatalf("expected nil global binding error")
	}
	if err := b.AddBootstrapSource("helpers.js", "const x = 1;"); err != nil {
		t.Fatalf("AddBootstrapSource failed: %v", err)
	}
	if err := b.AddBootstrapFile("/tmp/router.js"); err != nil {
		t.Fatalf("AddBootstrapFile failed: %v", err)
	}
	if err := b.AddHelper("serve", "serve(port)", "Start a HTTP server"); err != nil {
		t.Fatalf("AddHelper failed: %v", err)
	}
	if err := b.AddInitializer(testRuntimeInitializer{}); err != nil {
		t.Fatalf("AddInitializer failed: %v", err)
	}
	if err := b.AddNativeModule(testNativeModule{}); err != nil {
		t.Fatalf("AddNativeModule failed: %v", err)
	}
	if err := b.AddGlobal("db", func(*gojengine.RuntimeContext) error { return nil }, GlobalDoc{
		Type:        "DatabaseClient",
		Description: "Scoped DB handle",
	}); err != nil {
		t.Fatalf("AddGlobal failed: %v", err)
	}

	got := b.Manifest()
	if len(got.Modules) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(got.Modules))
	}
	if got.Modules[0].Name != "db" || len(got.Modules[0].Exports) != 2 {
		t.Fatalf("expected normalized db module manifest, got %#v", got.Modules[0])
	}
	if got.Modules[1].Name != "fs-lite" {
		t.Fatalf("expected fs-lite module, got %#v", got.Modules[1])
	}
	if len(got.Globals) != 1 || got.Globals[0].Name != "db" {
		t.Fatalf("unexpected globals manifest: %#v", got.Globals)
	}
	if len(got.Helpers) != 1 || got.Helpers[0].Signature != "serve(port)" {
		t.Fatalf("unexpected helpers manifest: %#v", got.Helpers)
	}
	if len(got.BootstrapFiles) != 2 {
		t.Fatalf("expected 2 bootstrap entries, got %d", len(got.BootstrapFiles))
	}

	got.Modules[0].Name = "mutated"
	if b.Manifest().Modules[0].Name != "db" {
		t.Fatalf("manifest should be cloned on read")
	}
}

func TestEnvironmentSpecCarriesConfigure(t *testing.T) {
	type scope struct{ Name string }
	type meta struct{ Count int }

	spec := EnvironmentSpec[scope, meta]{
		RuntimeLabel: "dbserver",
		Tool: ToolDefinitionSpec{
			Name: "eval_dbserver",
		},
		DefaultEval: DefaultEvalOptions(),
		Describe: func() (EnvironmentManifest, error) {
			return EnvironmentManifest{
				Helpers: []HelperDoc{{Name: "hello", Signature: "hello()"}},
			}, nil
		},
		Configure: func(ctx context.Context, b *Builder, s scope) (meta, error) {
			if err := b.AddHelper("hello", "hello()", "demo"); err != nil {
				return meta{}, err
			}
			return meta{Count: len(s.Name)}, nil
		},
	}

	b := &Builder{}
	out, err := spec.Configure(context.Background(), b, scope{Name: "abc"})
	if err != nil {
		t.Fatalf("Configure failed: %v", err)
	}
	if out.Count != 3 {
		t.Fatalf("expected meta count 3, got %d", out.Count)
	}
	if len(b.Manifest().Helpers) != 1 {
		t.Fatalf("expected helper to be recorded")
	}
	manifest, err := spec.Describe()
	if err != nil {
		t.Fatalf("Describe failed: %v", err)
	}
	if len(manifest.Helpers) != 1 {
		t.Fatalf("expected describe helper to be recorded")
	}
}
