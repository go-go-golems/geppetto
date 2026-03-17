package scopedjs

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	gojengine "github.com/go-go-golems/go-go-goja/engine"
	ggjmodules "github.com/go-go-golems/go-go-goja/modules"
)

type registeredModuleSpec struct {
	id       string
	register ModuleRegistrar
}

func (s registeredModuleSpec) ID() string {
	return s.id
}

func (s registeredModuleSpec) Register(reg *require.Registry) error {
	if reg == nil {
		return fmt.Errorf("require registry is nil")
	}
	if s.register == nil {
		return fmt.Errorf("module %q register function is nil", s.id)
	}
	return s.register(reg)
}

type runtimeInitFunc struct {
	id string
	fn func(ctx *gojengine.RuntimeContext) error
}

func (f runtimeInitFunc) ID() string {
	return f.id
}

func (f runtimeInitFunc) InitRuntime(ctx *gojengine.RuntimeContext) error {
	if f.fn == nil {
		return nil
	}
	return f.fn(ctx)
}

func BuildRuntime[Scope any, Meta any](ctx context.Context, spec EnvironmentSpec[Scope, Meta], scope Scope) (*BuildResult[Meta], error) {
	if spec.Configure == nil {
		return nil, fmt.Errorf("configure callback is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	builder := &Builder{}
	meta, err := spec.Configure(ctx, builder, scope)
	if err != nil {
		return nil, fmt.Errorf("configure runtime %q: %w", spec.RuntimeLabel, err)
	}

	engineBuilder := gojengine.NewBuilder()
	if len(builder.modules) > 0 || len(builder.nativeModules) > 0 {
		engineBuilder = engineBuilder.WithModules(builder.moduleSpecs()...)
	}
	if len(builder.globals) > 0 || len(builder.initializers) > 0 {
		engineBuilder = engineBuilder.WithRuntimeInitializers(builder.runtimeInitializers()...)
	}

	factory, err := engineBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("build runtime factory %q: %w", spec.RuntimeLabel, err)
	}
	rt, err := factory.NewRuntime(ctx)
	if err != nil {
		return nil, fmt.Errorf("create runtime %q: %w", spec.RuntimeLabel, err)
	}
	if err := builder.loadBootstrap(ctx, rt); err != nil {
		_ = rt.Close(context.Background())
		return nil, fmt.Errorf("load bootstrap %q: %w", spec.RuntimeLabel, err)
	}

	return &BuildResult[Meta]{
		Runtime:  rt,
		Executor: NewRuntimeExecutor(rt),
		Meta:     meta,
		Manifest: builder.Manifest(),
		Cleanup: func() error {
			return rt.Close(context.Background())
		},
	}, nil
}

func (b *Builder) moduleSpecs() []gojengine.ModuleSpec {
	specs := make([]gojengine.ModuleSpec, 0, len(b.modules)+len(b.nativeModules))
	for _, mod := range b.modules {
		specs = append(specs, registeredModuleSpec{
			id:       "module:" + mod.name,
			register: mod.register,
		})
	}
	for _, mod := range b.nativeModules {
		if mod == nil {
			continue
		}
		specs = append(specs, gojengine.NativeModuleSpec{
			ModuleID:   "native:" + strings.TrimSpace(mod.Name()),
			ModuleName: strings.TrimSpace(mod.Name()),
			Loader:     mod.Loader,
		})
	}
	return specs
}

func (b *Builder) runtimeInitializers() []gojengine.RuntimeInitializer {
	ret := make([]gojengine.RuntimeInitializer, 0, len(b.globals)+len(b.initializers))
	for _, global := range b.globals {
		global_ := global
		ret = append(ret, runtimeInitFunc{
			id: "global:" + global_.name,
			fn: func(ctx *gojengine.RuntimeContext) error {
				return global_.bind(ctx)
			},
		})
	}
	ret = append(ret, b.initializers...)
	return ret
}

func (b *Builder) loadBootstrap(ctx context.Context, rt *gojengine.Runtime) error {
	for _, entry := range b.bootstrapEntries {
		entry := entry
		name, source, err := bootstrapSource(entry)
		if err != nil {
			return err
		}
		if _, err := rt.Owner.Call(ctx, "scopedjs.bootstrap."+name, func(_ context.Context, vm *goja.Runtime) (any, error) {
			return vm.RunScript(name, source)
		}); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

func bootstrapSource(entry bootstrapEntry) (string, string, error) {
	if strings.TrimSpace(entry.source) != "" {
		return firstNonEmpty(entry.name, "bootstrap.js"), entry.source, nil
	}
	path := strings.TrimSpace(entry.filePath)
	if path == "" {
		return "", "", fmt.Errorf("bootstrap entry is empty")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	return firstNonEmpty(entry.name, path), string(b), nil
}

func ModuleDocFromNativeModule(mod ggjmodules.NativeModule) ModuleDoc {
	if mod == nil {
		return ModuleDoc{}
	}
	return ModuleDoc{
		Name:        strings.TrimSpace(mod.Name()),
		Description: strings.TrimSpace(mod.Doc()),
	}
}
