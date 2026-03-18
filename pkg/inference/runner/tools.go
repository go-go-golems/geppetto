package runner

import (
	"context"
	"fmt"

	geptools "github.com/go-go-golems/geppetto/pkg/inference/tools"
)

// FuncTool converts a Go function into a registrar that adds the resulting tool
// definition to a registry.
func FuncTool(name, description string, fn any) ToolRegistrar {
	return func(ctx context.Context, reg geptools.ToolRegistry) error {
		_ = ctx
		if reg == nil {
			return fmt.Errorf("tool registry is nil")
		}
		def, err := geptools.NewToolFromFunc(name, description, fn)
		if err != nil {
			return err
		}
		return reg.RegisterTool(def.Name, *def)
	}
}

// MustFuncTool is like FuncTool but panics immediately when the function shape
// cannot be converted to a Geppetto tool definition.
func MustFuncTool(name, description string, fn any) ToolRegistrar {
	def, err := geptools.NewToolFromFunc(name, description, fn)
	if err != nil {
		panic(err)
	}
	return func(ctx context.Context, reg geptools.ToolRegistry) error {
		_ = ctx
		if reg == nil {
			return fmt.Errorf("tool registry is nil")
		}
		return reg.RegisterTool(def.Name, *def)
	}
}

func buildRegistry(ctx context.Context, registrars []ToolRegistrar, names []string) (geptools.ToolRegistry, error) {
	if len(registrars) == 0 {
		if len(names) == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: %q", ErrRequestedToolMissing, names[0])
	}

	reg := geptools.NewInMemoryToolRegistry()
	for _, registrar := range registrars {
		if registrar == nil {
			continue
		}
		if err := registrar(ctx, reg); err != nil {
			return nil, err
		}
	}

	if len(reg.ListTools()) == 0 {
		if len(names) == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: %q", ErrRequestedToolMissing, names[0])
	}
	if len(names) == 0 {
		return reg, nil
	}
	return filterRegistry(reg, names)
}

func filterRegistry(reg geptools.ToolRegistry, names []string) (geptools.ToolRegistry, error) {
	if reg == nil {
		if len(names) == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: %q", ErrRequestedToolMissing, names[0])
	}
	if len(names) == 0 {
		return reg.Clone(), nil
	}

	filtered := geptools.NewInMemoryToolRegistry()
	for _, name := range names {
		def, err := reg.GetTool(name)
		if err != nil {
			return nil, fmt.Errorf("%w: %q", ErrRequestedToolMissing, name)
		}
		if err := filtered.RegisterTool(def.Name, *def); err != nil {
			return nil, err
		}
	}
	return filtered, nil
}
