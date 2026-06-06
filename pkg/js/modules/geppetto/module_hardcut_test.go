package geppetto

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	gojengine "github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
)

type jsRuntime struct {
	vm           *goja.Runtime
	runtimeOwner runtimeowner.RuntimeOwner
}

func newJSRuntime(t *testing.T, opts Options) *jsRuntime {
	t.Helper()
	factory, err := gojengine.NewRuntimeFactoryBuilder().Build()
	if err != nil {
		t.Fatalf("failed creating go-go-goja factory: %v", err)
	}
	rt, err := factory.NewRuntime(gojengine.WithStartupContext(context.Background()), gojengine.WithLifetimeContext(context.Background()))
	if err != nil {
		t.Fatalf("failed creating go-go-goja runtime: %v", err)
	}
	opts.RuntimeOwner = rt.Owner
	reg := require.NewRegistry()
	Register(reg, opts)
	reg.Enable(rt.VM)
	t.Cleanup(func() { _ = rt.Close(context.Background()) })
	return &jsRuntime{vm: rt.VM, runtimeOwner: rt.Owner}
}

func mustRunJS(t *testing.T, rt *jsRuntime, src string) goja.Value {
	t.Helper()
	v, err := rt.vm.RunString(src)
	if err != nil {
		t.Fatalf("js execution failed: %v\nscript:\n%s", err, src)
	}
	return v
}

func mustEvalExprExport(t *testing.T, rt *jsRuntime, expr string) any {
	t.Helper()
	ret, err := rt.runtimeOwner.Call(context.Background(), "module_test.EvalExpr", func(_ context.Context, vm *goja.Runtime) (any, error) {
		v, runErr := vm.RunString(expr)
		if runErr != nil {
			return nil, runErr
		}
		if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
			return nil, nil
		}
		return v.Export(), nil
	})
	if err != nil {
		t.Fatalf("failed evaluating %q: %v", expr, err)
	}
	return ret
}

func TestHardCutPublicSurface(t *testing.T) {
	rt := newJSRuntime(t, Options{})
	mustRunJS(t, rt, `
		const gp = require("geppetto");
		function assert(cond, msg) { if (!cond) throw new Error(msg); }
		function has(obj, key) { return Object.prototype.hasOwnProperty.call(obj, key); }
		const required = ["version", "consts", "agent", "inferenceProfiles", "turnStores", "engine", "embeddings", "tool", "toolRegistry", "schema"];
		for (const key of required) assert(has(gp, key), "missing export: " + key);
		const removed = ["chat", "inferenceSettings", "turn", "createBuilder", "createSession", "runInference", "profiles", "engines", "turns", "runner", "schemas", "middlewares", "tools", "events"];
		for (const key of removed) assert(!has(gp, key), "legacy export should be absent: " + key);
	`)
}

func TestHardCutExamples(t *testing.T) {
	sourcePath, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "examples", "js", "geppetto", "profiles", "50-hardcut-phase123.yaml"))
	if err != nil {
		t.Fatalf("Abs profile fixture failed: %v", err)
	}
	for _, scriptRel := range []string{
		filepath.Join("..", "..", "..", "..", "examples", "js", "geppetto", "25_inference_profiles_load_resolve_settings.js"),
		filepath.Join("..", "..", "..", "..", "examples", "js", "geppetto", "26_engine_builder_from_registry_profile.js"),
		filepath.Join("..", "..", "..", "..", "examples", "js", "geppetto", "28_agent_from_registry_profile.js"),
		filepath.Join("..", "..", "..", "..", "examples", "js", "geppetto", "29_tools_schema_multimodal_turn.js"),
		filepath.Join("..", "..", "..", "..", "examples", "js", "geppetto", "hardcut", "01_load_registry_resolve_profile.js"),
		filepath.Join("..", "..", "..", "..", "examples", "js", "geppetto", "hardcut", "02_engine_from_registry_profile.js"),
		filepath.Join("..", "..", "..", "..", "examples", "js", "geppetto", "hardcut", "03_agent_from_registry_profile.js"),
		filepath.Join("..", "..", "..", "..", "examples", "js", "geppetto", "hardcut", "04_tools_and_schema.js"),
		filepath.Join("..", "..", "..", "..", "examples", "js", "geppetto", "hardcut", "05_multimodal_turn.js"),
		filepath.Join("..", "..", "..", "..", "examples", "js", "geppetto", "hardcut", "06_embeddings_with_registry_profile.js"),
	} {
		scriptRel := scriptRel
		t.Run(filepath.Base(scriptRel), func(t *testing.T) {
			script, err := os.ReadFile(scriptRel)
			if err != nil {
				t.Fatalf("ReadFile(%s) failed: %v", scriptRel, err)
			}
			rt := newJSRuntime(t, Options{})
			mustRunJS(t, rt, fmt.Sprintf("globalThis.GEPPETTO_PHASE123_PROFILE = %q;\n%s", sourcePath, string(script)))
		})
	}
}
