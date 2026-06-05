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
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	gojengine "github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/jsevents"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

type fakeHost struct {
	seen Config
	opts geppettomodule.Options
}

func (h *fakeHost) GeppettoOptions(_ context.Context, cfg Config) (geppettomodule.Options, error) {
	h.seen = cfg
	return h.opts, nil
}

func (h *fakeHost) AssetResolver() providerapi.AssetResolver {
	return nil
}

func TestRegisterProvider(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
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

func TestProviderWorksWithoutHostServices(t *testing.T) {
	mod := resolveModule(t)
	loader, err := mod.NewModuleFactory(providerapi.ModuleSetupContext{})
	if err != nil {
		t.Fatalf("expected provider to work without host services: %v", err)
	}
	if loader == nil {
		t.Fatalf("loader is nil")
	}
}

func TestProviderLoadsDefaultProfileRegistries(t *testing.T) {
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
	loader, err := mod.NewModuleFactory(providerapi.ModuleSetupContext{
		Name: geppettomodule.ModuleName,
		As:   geppettomodule.ModuleName,
		Host: host,
		Config: json.RawMessage(`{
			"defaultProfileRegistries": [` + strconv.Quote(profilePath) + `],
			"defaultProfile": "assistant"
		}`),
	})
	if err != nil {
		t.Fatalf("create loader: %v", err)
	}
	if got := len(host.seen.DefaultProfileRegistries); got != 1 {
		t.Fatalf("host saw %d defaultProfileRegistries", got)
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

func TestProviderIgnoresRemovedLegacyRegistryField(t *testing.T) {
	mod := resolveModule(t)
	host := &fakeHost{}
	_, err := mod.NewModuleFactory(providerapi.ModuleSetupContext{
		Name:   geppettomodule.ModuleName,
		As:     geppettomodule.ModuleName,
		Host:   host,
		Config: json.RawMessage(`{"registry": "legacy-host-selector"}`),
	})
	if err != nil {
		t.Fatalf("legacy registry field should be ignored by provider decode: %v", err)
	}
	if len(host.seen.DefaultProfileRegistries) != 0 {
		t.Fatalf("legacy registry populated defaultProfileRegistries: %#v", host.seen.DefaultProfileRegistries)
	}
}

func TestProviderIgnoresRemovedLegacyStorageFields(t *testing.T) {
	mod := resolveModule(t)
	host := &fakeHost{}
	loader, err := mod.NewModuleFactory(providerapi.ModuleSetupContext{
		Context: context.Background(),
		Name:    geppettomodule.ModuleName,
		As:      geppettomodule.ModuleName,
		Host:    host,
		Config:  json.RawMessage(`{"enableStorage":true,"turns":{"default":true,"phase":"final"}}`),
	})
	if err != nil {
		t.Fatalf("legacy storage fields should be ignored by provider decode: %v", err)
	}
	if loader == nil {
		t.Fatalf("loader is nil")
	}
}

func TestProviderMapsGlazedFlagsToXGojaConfig(t *testing.T) {
	providerCapability := capability{}
	sections, err := providerCapability.GlazedConfigSections(providerapi.SectionRequest{})
	if err != nil {
		t.Fatalf("GlazedConfigSections failed: %v", err)
	}
	if len(sections) != 1 {
		t.Fatalf("sections = %d, want 1", len(sections))
	}
	if sections[0].GetPrefix() != "" {
		t.Fatalf("geppetto public flags should be unprefixed, got prefix %q", sections[0].GetPrefix())
	}
	glazedSection, err := values.NewSectionValues(sections[0],
		values.WithFieldValue("profile-registries", []string{"profiles.yaml"}),
		values.WithFieldValue("profile", "assistant"),
		values.WithFieldValue("turns-db", "/tmp/turns.db"),
	)
	if err != nil {
		t.Fatalf("NewSectionValues failed: %v", err)
	}
	configSection, err := xgojaConfigSection()
	if err != nil {
		t.Fatalf("xgojaConfigSection failed: %v", err)
	}
	out, err := providerCapability.XGojaConfigFromGlazed(context.Background(), providerapi.XGojaConfigRequest{
		ConfigSection: configSection,
		GlazedValues:  values.New(values.WithSectionValues(configSectionSlug, glazedSection)),
	})
	if err != nil {
		t.Fatalf("XGojaConfigFromGlazed failed: %v", err)
	}
	assertSectionField(t, out, "defaultProfile", "assistant")
	assertSectionField(t, out, "turnsDB", "/tmp/turns.db")
	registries, ok := out.GetField("defaultProfileRegistries")
	if !ok {
		t.Fatalf("defaultProfileRegistries missing")
	}
	entries, ok := registries.([]string)
	if !ok || len(entries) != 1 || entries[0] != "profiles.yaml" {
		t.Fatalf("defaultProfileRegistries = %#v", registries)
	}
}

func TestProviderRegistersSQLiteTurnStoreCloser(t *testing.T) {
	mod := resolveModule(t)
	closers := []func(context.Context) error{}
	_, err := mod.NewModuleFactory(providerapi.ModuleSetupContext{
		Context: context.Background(),
		Name:    geppettomodule.ModuleName,
		As:      geppettomodule.ModuleName,
		Config: json.RawMessage(`{
			"turnsDB": ` + strconv.Quote(filepath.Join(t.TempDir(), "turns.db")) + `
		}`),
		AddCloser: func(fn func(context.Context) error) error {
			closers = append(closers, fn)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("NewModuleFactory: %v", err)
	}
	if len(closers) != 1 {
		t.Fatalf("closers = %d, want 1", len(closers))
	}
	if err := closers[0](context.Background()); err != nil {
		t.Fatalf("closer: %v", err)
	}
}

func TestSQLiteTurnStorePersistsAndReadsTurns(t *testing.T) {
	store, err := openSQLiteTurnStore("", filepath.Join(t.TempDir(), "turns.db"))
	if err != nil {
		t.Fatalf("openSQLiteTurnStore failed: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	turn := &turns.Turn{ID: "turn-a"}
	if err := turns.KeyTurnMetaSessionID.Set(&turn.Metadata, "session-a"); err != nil {
		t.Fatalf("set session id: %v", err)
	}
	turns.AppendBlock(turn, turns.NewUserTextBlock("hello"))
	turns.AppendBlock(turn, turns.NewAssistantTextBlock("stored"))
	if err := store.PersistTurn(context.Background(), turn); err != nil {
		t.Fatalf("PersistTurn failed: %v", err)
	}
	listed, err := store.ListTurns(context.Background(), geppettomodule.TurnStoreQuery{SessionID: "session-a", Phase: "final"})
	if err != nil {
		t.Fatalf("ListTurns failed: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("listed = %d, want 1", len(listed))
	}
	latest, err := store.LoadLatestTurn(context.Background(), geppettomodule.TurnStoreQuery{SessionID: "session-a", Phase: "final"})
	if err != nil {
		t.Fatalf("LoadLatestTurn failed: %v", err)
	}
	if latest == nil || latest.Turn == nil || latest.SessionID != "session-a" || latest.TurnID != "turn-a" {
		t.Fatalf("unexpected latest: %#v", latest)
	}
}

func assertSectionField(t *testing.T, sectionValues *values.SectionValues, key string, want any) {
	t.Helper()
	got, ok := sectionValues.GetField(key)
	if !ok {
		t.Fatalf("%s missing", key)
	}
	if got != want {
		t.Fatalf("%s = %#v, want %#v", key, got, want)
	}
}

type providerRuntimeModuleRegistrar struct{}

func (providerRuntimeModuleRegistrar) ID() string { return "geppetto-provider-test" }

func (providerRuntimeModuleRegistrar) RegisterRuntimeModule(ctx *gojengine.RuntimeModuleRegistrationContext, reg *require.Registry) error {
	registry := providerapi.NewProviderRegistry()
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
	loader, err := mod.NewModuleFactory(providerapi.ModuleSetupContext{
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
	factory, err := gojengine.NewRuntimeFactoryBuilder(
		gojengine.WithDataOnlyDefaultRegistryModules(true),
	).
		UseModuleMiddleware(gojengine.Pipeline()).
		WithRuntimeInitializers(jsevents.Install()).
		WithModules(providerRuntimeModuleRegistrar{}).
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
			const session = agent.session().id("provider-path-test").build();
			session.next().user("provider path").runAsync().promise.then(result => {
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
	loader, err := mod.NewModuleFactory(providerapi.ModuleSetupContext{
		Name:   geppettomodule.ModuleName,
		As:     geppettomodule.ModuleName,
		Host:   host,
		Config: json.RawMessage(`{"profile":"test-profile","allowNetwork":true}`),
	})
	if err != nil {
		t.Fatalf("create loader: %v", err)
	}
	if host.seen.DefaultProfile != "" || len(host.seen.DefaultProfileRegistries) != 0 {
		t.Fatalf("removed legacy config fields should be ignored, host saw %+v", host.seen)
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
	for _, name := range []string{"agent", "engine", "tool", "toolRegistry"} {
		if _, ok := goja.AssertFunction(exports.Get(name)); !ok {
			t.Fatalf("%s export is not a function", name)
		}
	}
	for _, name := range []string{"consts", "inferenceProfiles", "schema", "turnStores"} {
		if obj := exports.Get(name).ToObject(vm); obj == nil {
			t.Fatalf("%s export is not an object", name)
		}
	}
	for _, name := range []string{"turn", "createBuilder", "createSession", "runInference", "turns", "engines", "profiles", "runner", "schemas", "middlewares", "tools"} {
		if v := exports.Get(name); v != nil && !goja.IsUndefined(v) {
			t.Fatalf("legacy export %s should be absent", name)
		}
	}
}

func resolveModule(t *testing.T) providerapi.Module {
	t.Helper()
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	mod, ok := registry.ResolveModule(PackageID, geppettomodule.ModuleName)
	if !ok {
		t.Fatalf("missing module")
	}
	return mod
}
