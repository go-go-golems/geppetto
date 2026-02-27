package runtime

import (
	"context"
	"fmt"

	"github.com/dop251/goja_nodejs/require"
	gp "github.com/go-go-golems/geppetto/pkg/js/modules/geppetto"
	gojengine "github.com/go-go-golems/go-go-goja/engine"
	ggmodules "github.com/go-go-golems/go-go-goja/modules"
)

// Options configure a geppetto JavaScript runtime bootstrapped on top of the
// go-go-goja owned runtime builder.
type Options struct {
	// ModuleOptions are forwarded to geppetto.Register. Runner is always bound
	// to the created runtime owner.
	ModuleOptions gp.Options

	// RequireOptions are applied to the runtime's module registry.
	RequireOptions []require.Option

	// IncludeDefaultModules enables go-go-goja default native modules in
	// addition to geppetto.
	IncludeDefaultModules bool

	// RuntimeInitializers are executed after require/geppetto registration.
	RuntimeInitializers []gojengine.RuntimeInitializer
}

// NewRuntime creates a new owned JS runtime that exposes require("geppetto").
func NewRuntime(ctx context.Context, opts Options) (*gojengine.Runtime, error) {
	builderOpts := make([]gojengine.Option, 0, 1)
	if len(opts.RequireOptions) > 0 {
		builderOpts = append(builderOpts, gojengine.WithRequireOptions(opts.RequireOptions...))
	}
	factory, err := gojengine.NewBuilder(builderOpts...).Build()
	if err != nil {
		return nil, fmt.Errorf("build runtime factory: %w", err)
	}

	rt, err := factory.NewRuntime(ctx)
	if err != nil {
		return nil, fmt.Errorf("create runtime: %w", err)
	}

	moduleOpts := opts.ModuleOptions
	moduleOpts.Runner = rt.Owner

	reg := require.NewRegistry(opts.RequireOptions...)
	if opts.IncludeDefaultModules {
		ggmodules.EnableAll(reg)
	}
	gp.Register(reg, moduleOpts)
	reqMod := reg.Enable(rt.VM)
	rt.Require = reqMod

	if len(opts.RuntimeInitializers) > 0 {
		runtimeCtx := &gojengine.RuntimeContext{
			VM:      rt.VM,
			Require: reqMod,
			Loop:    rt.Loop,
			Owner:   rt.Owner,
		}
		for _, init := range opts.RuntimeInitializers {
			if init == nil {
				continue
			}
			if err := init.InitRuntime(runtimeCtx); err != nil {
				_ = rt.Close(context.Background())
				return nil, fmt.Errorf("runtime initializer %q: %w", init.ID(), err)
			}
		}
	}

	return rt, nil
}
