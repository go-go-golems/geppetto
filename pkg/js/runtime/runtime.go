package runtime

import (
	"context"
	"fmt"

	"github.com/dop251/goja_nodejs/require"
	gp "github.com/go-go-golems/geppetto/pkg/js/modules/geppetto"
	gojengine "github.com/go-go-golems/go-go-goja/engine"
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

type geppettoModuleSpec struct {
	opts gp.Options
}

func (s geppettoModuleSpec) ID() string { return "geppetto" }

func (s geppettoModuleSpec) RegisterRuntimeModule(ctx *gojengine.RuntimeModuleContext, reg *require.Registry) error {
	if ctx == nil {
		return fmt.Errorf("runtime module context is nil")
	}
	if reg == nil {
		return fmt.Errorf("require registry is nil")
	}
	opts := s.opts
	opts.Runner = ctx.Owner
	gp.Register(reg, opts)
	return nil
}

// NewRuntime creates a new owned JS runtime that exposes require("geppetto").
func NewRuntime(ctx context.Context, opts Options) (*gojengine.Runtime, error) {
	builderOpts := make([]gojengine.Option, 0, 1)
	if len(opts.RequireOptions) > 0 {
		builderOpts = append(builderOpts, gojengine.WithRequireOptions(opts.RequireOptions...))
	}
	builder := gojengine.NewBuilder(builderOpts...).WithModules(geppettoModuleSpec{opts: opts.ModuleOptions})
	if opts.IncludeDefaultModules {
		builder = builder.UseModuleMiddleware(gojengine.Pipeline())
	}
	if len(opts.RuntimeInitializers) > 0 {
		builder = builder.WithRuntimeInitializers(opts.RuntimeInitializers...)
	}
	factory, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("build runtime factory: %w", err)
	}

	rt, err := factory.NewRuntime(ctx)
	if err != nil {
		return nil, fmt.Errorf("create runtime: %w", err)
	}
	return rt, nil
}
