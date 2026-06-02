package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/geppetto/pkg/events"
	inferenceengine "github.com/go-go-golems/geppetto/pkg/inference/engine"
	geppettomodule "github.com/go-go-golems/geppetto/pkg/js/modules/geppetto"
	"github.com/go-go-golems/geppetto/pkg/turns"
	gojengine "github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/jsevents"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

type fakeHost struct {
	seen        Config
	opts        geppettomodule.Options
	storage     geppettomodule.StorageOptions
	storageSeen bool
}

func (h *fakeHost) GeppettoOptions(_ context.Context, cfg Config) (geppettomodule.Options, error) {
	h.seen = cfg
	return h.opts, nil
}

func (h *fakeHost) GeppettoTurnStores(_ context.Context, cfg Config) (geppettomodule.StorageOptions, error) {
	h.seen = cfg
	h.storageSeen = true
	return h.storage, nil
}

func (h *fakeHost) AssetResolver() providerapi.AssetResolver {
	return nil
}

type fakeOptionsOnlyHost struct{}

func (fakeOptionsOnlyHost) GeppettoOptions(context.Context, Config) (geppettomodule.Options, error) {
	return geppettomodule.Options{}, nil
}

func (fakeOptionsOnlyHost) AssetResolver() providerapi.AssetResolver { return nil }

type providerRecordingStore struct{}

var _ geppettomodule.TurnStore = (*providerRecordingStore)(nil)

func (s *providerRecordingStore) PersistTurn(context.Context, *turns.Turn) error { return nil }
func (s *providerRecordingStore) ListTurns(context.Context, geppettomodule.TurnStoreQuery) ([]geppettomodule.TurnStoreSnapshot, error) {
	return nil, nil
}
func (s *providerRecordingStore) LoadLatestTurn(context.Context, geppettomodule.TurnStoreQuery) (*geppettomodule.TurnStoreSnapshot, error) {
	return nil, nil
}
func (s *providerRecordingStore) Close() error { return nil }

func TestRegisterProvider(t *testing.T) {
	registry := providerapi.NewRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	mod, ok := registry.ResolveModule(PackageID, geppettomodule.ModuleName)
	if !ok {
		t.Fatalf("missing module %s.%s", PackageID, geppettomodule.ModuleName)
	}
	if mod.DefaultAs != geppettomodule.ModuleName {
		t.Fatalf("default alias = %q, want %q", mod.DefaultAs, geppettomodule.ModuleName)
	}
}

func TestProviderRequiresHostServices(t *testing.T) {
	mod := resolveModule(t)
	if _, err := mod.New(providerapi.ModuleContext{}); err == nil {
		t.Fatalf("expected missing host services error")
	}
}

func TestProviderLoadsProfileRegistriesWhenAllowed(t *testing.T) {
	profilePath := filepath.Join(t.TempDir(), "profiles.yaml")
	if err := os.WriteFile(profilePath, []byte(`slug: xgoja
profiles:
  assistant:
    inference_settings:
      api:
        api_keys:
          openai-api-key: test-key
      chat:
        api_type: openai
        engine: gpt-4o-mini
`), 0o644); err != nil {
		t.Fatalf("WriteFile profiles.yaml: %v", err)
	}
	mod := resolveModule(t)
	host := &fakeHost{}
	loader, err := mod.New(providerapi.ModuleContext{
		Name: geppettomodule.ModuleName,
		As:   geppettomodule.ModuleName,
		Host: host,
		Config: json.RawMessage(`{
			"profileRegistries": [` + strconv.Quote(profilePath) + `],
			"defaultProfile": "assistant",
			"allowRegistryLoad": true
		}`),
	})
	if err != nil {
		t.Fatalf("create loader: %v", err)
	}
	if got := len(host.seen.ProfileRegistries); got != 1 {
		t.Fatalf("host saw %d profileRegistries", got)
	}

	vm := goja.New()
	moduleObj := vm.NewObject()
	exports := vm.NewObject()
	if err := moduleObj.Set("exports", exports); err != nil {
		t.Fatalf("set exports: %v", err)
	}
	loader(vm, moduleObj)
	resolve, ok := goja.AssertFunction(exports.Get("inferenceProfiles").ToObject(vm).Get("resolve"))
	if !ok {
		t.Fatalf("inferenceProfiles.resolve is not callable")
	}
	settings, err := resolve(goja.Undefined(), vm.ToValue("assistant"))
	if err != nil {
		t.Fatalf("resolve assistant: %v", err)
	}
	toJSON, ok := goja.AssertFunction(settings.ToObject(vm).Get("toJSON"))
	if !ok {
		t.Fatalf("settings.toJSON is not callable")
	}
	snapshot, err := toJSON(settings)
	if err != nil {
		t.Fatalf("settings.toJSON: %v", err)
	}
	m, ok := snapshot.Export().(map[string]any)
	if !ok || m["chat"] == nil {
		t.Fatalf("unexpected snapshot %#v", snapshot.Export())
	}
}

func TestProviderRejectsProfileRegistriesUnlessAllowed(t *testing.T) {
	mod := resolveModule(t)
	_, err := mod.New(providerapi.ModuleContext{
		Name:   geppettomodule.ModuleName,
		As:     geppettomodule.ModuleName,
		Host:   &fakeHost{},
		Config: json.RawMessage(`{"profileRegistries": ["profiles.yaml"]}`),
	})
	if err == nil {
		t.Fatalf("expected allowRegistryLoad error")
	}
}

func TestProviderTurnsConfigRequiresEnableStorage(t *testing.T) {
	mod := resolveModule(t)
	_, err := mod.New(providerapi.ModuleContext{
		Context: context.Background(),
		Name:    geppettomodule.ModuleName,
		As:      geppettomodule.ModuleName,
		Host:    &fakeHost{},
		Config:  json.RawMessage(`{"turns":{"dsn":"file:test.sqlite"}}`),
	})
	if err == nil {
		t.Fatalf("expected enableStorage error")
	}
}

func TestProviderEnableStorageRequiresStorageHost(t *testing.T) {
	mod := resolveModule(t)
	_, err := mod.New(providerapi.ModuleContext{
		Context: context.Background(),
		Name:    geppettomodule.ModuleName,
		As:      geppettomodule.ModuleName,
		Host:    fakeOptionsOnlyHost{},
		Config:  json.RawMessage(`{"enableStorage":true}`),
	})
	if err == nil {
		t.Fatalf("expected storage host capability error")
	}
}

func TestProviderEnableStorageInstallsDefaultTurnStore(t *testing.T) {
	mod := resolveModule(t)
	store := &providerRecordingStore{}
	host := &fakeHost{storage: geppettomodule.StorageOptions{Default: store}}
	loader, err := mod.New(providerapi.ModuleContext{
		Context: context.Background(),
		Name:    geppettomodule.ModuleName,
		As:      geppettomodule.ModuleName,
		Host:    host,
		Config:  json.RawMessage(`{"enableStorage":true,"turns":{"default":true,"phase":"final"}}`),
	})
	if err != nil {
		t.Fatalf("create loader with storage: %v", err)
	}
	if loader == nil {
		t.Fatalf("loader is nil")
	}
	if !host.storageSeen {
		t.Fatalf("storage host was not invoked")
	}
	if host.seen.Turns == nil || !host.seen.Turns.Default || host.seen.Turns.Phase != "final" {
		t.Fatalf("host saw storage config %+v", host.seen)
	}
}

type providerRuntimeModuleSpec struct{}

func (providerRuntimeModuleSpec) ID() string { return "geppetto-provider-test" }

func (providerRuntimeModuleSpec) RegisterRuntimeModule(ctx *gojengine.RuntimeModuleContext, reg *require.Registry) error {
	registry := providerapi.NewRegistry()
	if err := Register(registry); err != nil {
		return err
	}
	mod, ok := registry.ResolveModule(PackageID, geppettomodule.ModuleName)
	if !ok {
		return fmt.Errorf("missing provider module")
	}
	host := &fakeHost{opts: geppettomodule.Options{
		RuntimeOwner: ctx.Owner,
		EventEmitterManagerResolver: func() (*jsevents.Manager, bool) {
			value, ok := ctx.Value(jsevents.RuntimeValueKey)
			if !ok {
				return nil, false
			}
			manager, ok := value.(*jsevents.Manager)
			return manager, ok && manager != nil
		},
	}}
	loader, err := mod.New(providerapi.ModuleContext{
		Context: context.Background(),
		Name:    geppettomodule.ModuleName,
		As:      geppettomodule.ModuleName,
		Host:    host,
	})
	if err != nil {
		return err
	}
	reg.RegisterNativeModule(geppettomodule.ModuleName, loader)
	return nil
}

type providerEventEngine struct{}

var _ inferenceengine.Engine = (*providerEventEngine)(nil)

func (e *providerEventEngine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	meta := events.EventMetadata{SessionID: "provider-session", InferenceID: "provider-inference", TurnID: "provider-turn"}
	corr := events.Correlation{SessionID: "provider-session", RunID: "provider-run", TurnID: "provider-turn"}
	events.PublishEventToContext(ctx, events.NewTextDeltaEvent(meta, corr, "provider", "provider", 1))
	out := t.Clone()
	out.Blocks = append(out.Blocks, turns.NewAssistantTextBlock("provider"))
	return out, nil
}

func TestProviderRuntimeSupportsEventEmitterRunAsync(t *testing.T) {
	factory, err := gojengine.NewBuilder(
		gojengine.WithDataOnlyDefaultRegistryModules(true),
	).
		UseModuleMiddleware(gojengine.Pipeline()).
		WithRuntimeInitializers(jsevents.Install()).
		WithModules(providerRuntimeModuleSpec{}).
		Build()
	if err != nil {
		t.Fatalf("failed creating runtime factory: %v", err)
	}
	rt, err := factory.NewRuntime(gojengine.WithStartupContext(context.Background()), gojengine.WithLifetimeContext(context.Background()))
	if err != nil {
		t.Fatalf("failed creating runtime: %v", err)
	}
	t.Cleanup(func() { _ = rt.Close(context.Background()) })

	_, err = rt.Owner.Call(context.Background(), "test.providerRunAsyncEvents", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if setErr := vm.Set("providerEventEngine", &providerEventEngine{}); setErr != nil {
			return nil, setErr
		}
		_, runErr := vm.RunString(`
			const gp = require("geppetto");
			const EventEmitter = require("events");
			globalThis.seen = [];
			globalThis.resolved = false;
			const emitter = new EventEmitter();
			emitter.on("text-delta", ev => globalThis.seen.push(ev.delta));
			const agent = gp.agent().engine(globalThis.providerEventEngine).events(emitter).build();
			agent.runAsync(gp.turn().user("provider path").build()).promise.then(result => {
				globalThis.resolved = true;
				globalThis.finalText = result.text();
			}, err => {
				globalThis.resolved = true;
				globalThis.finalError = String(err);
			});
		`)
		return nil, runErr
	})
	if err != nil {
		t.Fatalf("provider script failed: %v", err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		ret, waitErr := rt.Owner.Call(context.Background(), "test.providerWait", func(_ context.Context, vm *goja.Runtime) (any, error) {
			return vm.RunString(`globalThis.resolved === true`)
		})
		if waitErr != nil {
			t.Fatalf("wait failed: %v", waitErr)
		}
		if value, ok := ret.(goja.Value); ok && value.ToBoolean() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	got, err := rt.Owner.Call(context.Background(), "test.providerRead", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunString(`JSON.stringify({seen: globalThis.seen, resolved: globalThis.resolved, finalText: globalThis.finalText, finalError: globalThis.finalError})`)
	})
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	want := `{"seen":["provider"],"resolved":true,"finalText":"provider"}`
	if got.(goja.Value).String() != want {
		t.Fatalf("state = %s, want %s", got.(goja.Value).String(), want)
	}
}

func TestModuleLoaderInstallsGeppettoExports(t *testing.T) {
	mod := resolveModule(t)
	host := &fakeHost{}
	loader, err := mod.New(providerapi.ModuleContext{
		Name:   geppettomodule.ModuleName,
		As:     geppettomodule.ModuleName,
		Host:   host,
		Config: json.RawMessage(`{"profile":"test-profile","allowNetwork":true}`),
	})
	if err != nil {
		t.Fatalf("create loader: %v", err)
	}
	if host.seen.Profile != "test-profile" || !host.seen.AllowNetwork {
		t.Fatalf("host saw config %+v", host.seen)
	}

	vm := goja.New()
	moduleObj := vm.NewObject()
	exports := vm.NewObject()
	if err := moduleObj.Set("exports", exports); err != nil {
		t.Fatalf("set exports: %v", err)
	}
	loader(vm, moduleObj)
	if got := exports.Get("version").String(); got != "0.1.0" {
		t.Fatalf("version = %q, want 0.1.0", got)
	}
	for _, name := range []string{"agent", "turn", "engine", "tool", "toolRegistry"} {
		if _, ok := goja.AssertFunction(exports.Get(name)); !ok {
			t.Fatalf("%s export is not a function", name)
		}
	}
	for _, name := range []string{"consts", "inferenceProfiles", "schema", "turnStores"} {
		if obj := exports.Get(name).ToObject(vm); obj == nil {
			t.Fatalf("%s export is not an object", name)
		}
	}
	for _, name := range []string{"createBuilder", "createSession", "runInference", "turns", "engines", "profiles", "runner", "schemas", "middlewares", "tools"} {
		if v := exports.Get(name); v != nil && !goja.IsUndefined(v) {
			t.Fatalf("legacy export %s should be absent", name)
		}
	}
}

func resolveModule(t *testing.T) providerapi.Module {
	t.Helper()
	registry := providerapi.NewRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	mod, ok := registry.ResolveModule(PackageID, geppettomodule.ModuleName)
	if !ok {
		t.Fatalf("missing module")
	}
	return mod
}
