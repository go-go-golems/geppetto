package provider

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/dop251/goja"
	geppettomodule "github.com/go-go-golems/geppetto/pkg/js/modules/geppetto"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

type fakeHost struct {
	seen Config
}

func (h *fakeHost) GeppettoOptions(_ context.Context, cfg Config) (geppettomodule.Options, error) {
	h.seen = cfg
	return geppettomodule.Options{}, nil
}

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
	for _, name := range []string{"consts", "inferenceProfiles", "schema", "events"} {
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
