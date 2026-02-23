package main

import (
	"path/filepath"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	gp "github.com/go-go-golems/geppetto/pkg/js/modules/geppetto"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
	"github.com/rs/zerolog"
)

// jsRuntime bundles the goja VM, node-style eventloop, and module registry.
type jsRuntime struct {
	scriptRoot string

	loop   *eventloop.EventLoop
	vm     *goja.Runtime
	runner runtimeowner.Runner

	reg    *require.Registry
	reqMod *require.RequireModule
}

func newJSRuntime(scriptRoot string) (*jsRuntime, error) {
	if scriptRoot == "" {
		scriptRoot = "."
	}

	loop := eventloop.NewEventLoop()
	go loop.Start()

	vm := goja.New()
	runner := runtimeowner.NewRunner(vm, loop, runtimeowner.Options{
		Name:          "gepa-runner",
		RecoverPanics: true,
	})

	reg := require.NewRegistry(
		require.WithGlobalFolders(scriptRoot, filepath.Join(scriptRoot, "node_modules")),
	)

	// Install minimal console + helpers.
	if err := installConsoleAndHelpers(vm); err != nil {
		loop.Stop()
		return nil, err
	}

	// Register geppetto native module.
	gp.Register(reg, gp.Options{
		Runner: runner,
		Logger: zerolog.New(zerolog.NewConsoleWriter()),
	})

	reqMod := reg.Enable(vm)

	return &jsRuntime{
		scriptRoot: scriptRoot,
		loop:       loop,
		vm:         vm,
		runner:     runner,
		reg:        reg,
		reqMod:     reqMod,
	}, nil
}

func (r *jsRuntime) Close() {
	if r == nil {
		return
	}
	if r.loop != nil {
		r.loop.Stop()
	}
}
