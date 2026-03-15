package scopeddb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/go-go-golems/geppetto/pkg/inference/tools"
)

func RegisterPrebuilt[Scope any, Meta any](reg tools.ToolRegistry, spec DatasetSpec[Scope, Meta], db *sql.DB, opts QueryOptions) error {
	if reg == nil {
		return fmt.Errorf("tool registry is nil")
	}
	runner, err := NewQueryRunner(db, AllowedObjectMap(spec.AllowedObjects), opts)
	if err != nil {
		return err
	}
	def, err := tools.NewToolFromFunc(
		spec.Tool.Name,
		BuildDescription(spec.Tool.Description, spec.AllowedObjects, opts),
		func(ctx context.Context, in QueryInput) (QueryOutput, error) {
			return runner.Run(ctx, in)
		},
	)
	if err != nil {
		return fmt.Errorf("create %s tool: %w", spec.Tool.Name, err)
	}
	def.Tags = append([]string(nil), spec.Tool.Tags...)
	def.Version = spec.Tool.Version
	if err := reg.RegisterTool(def.Name, *def); err != nil {
		return fmt.Errorf("register %s tool: %w", spec.Tool.Name, err)
	}
	return nil
}

func NewLazyRegistrar[Scope any, Meta any](spec DatasetSpec[Scope, Meta], resolve ScopeResolver[Scope], opts QueryOptions) func(reg tools.ToolRegistry) error {
	return func(reg tools.ToolRegistry) error {
		if reg == nil {
			return fmt.Errorf("tool registry is nil")
		}
		if resolve == nil {
			return fmt.Errorf("scope resolver is nil")
		}
		def, err := tools.NewToolFromFunc(
			spec.Tool.Name,
			BuildDescription(spec.Tool.Description, spec.AllowedObjects, opts),
			func(ctx context.Context, in QueryInput) (QueryOutput, error) {
				scope, err := resolve(ctx)
				if err != nil {
					return QueryOutput{Error: err.Error()}, nil
				}
				handle, err := BuildInMemory(ctx, spec, scope)
				if err != nil {
					return QueryOutput{Error: err.Error()}, nil
				}
				defer func() {
					if handle.Cleanup != nil {
						_ = handle.Cleanup()
					}
				}()
				runner, err := NewQueryRunner(handle.DB, AllowedObjectMap(spec.AllowedObjects), opts)
				if err != nil {
					return QueryOutput{Error: err.Error()}, nil
				}
				return runner.Run(ctx, in)
			},
		)
		if err != nil {
			return fmt.Errorf("create %s tool: %w", spec.Tool.Name, err)
		}
		def.Tags = append([]string(nil), spec.Tool.Tags...)
		def.Version = spec.Tool.Version
		if err := reg.RegisterTool(def.Name, *def); err != nil {
			return fmt.Errorf("register %s tool: %w", spec.Tool.Name, err)
		}
		return nil
	}
}
