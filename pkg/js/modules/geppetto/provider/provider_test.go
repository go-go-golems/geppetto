package provider

import (
	"context"
	"encoding/json"
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
	for _, name := range []string{"createBuilder", "createSession", "runInference"} {
		if _, ok := goja.AssertFunction(exports.Get(name)); !ok {
			t.Fatalf("%s export is not a function", name)
		}
	}
	for _, name := range []string{"turns", "engines", "profiles", "runner"} {
		if obj := exports.Get(name).ToObject(vm); obj == nil {
			t.Fatalf("%s export is not an object", name)
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
