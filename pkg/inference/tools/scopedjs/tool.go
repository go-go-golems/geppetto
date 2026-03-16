package scopedjs

import (
	"context"
	"fmt"

	"github.com/go-go-golems/geppetto/pkg/inference/tools"
)

func RegisterPrebuilt[Scope any, Meta any](reg tools.ToolRegistry, spec EnvironmentSpec[Scope, Meta], handle *BuildResult[Meta], opts EvalOptions) error {
	if reg == nil {
		return fmt.Errorf("tool registry is nil")
	}
	if handle == nil || handle.Runtime == nil {
		return fmt.Errorf("build result runtime is nil")
	}
	evalOpts := resolveEvalOptions(spec.DefaultEval, opts)
	def, err := tools.NewToolFromFunc(
		spec.Tool.Name,
		BuildDescription(spec.Tool.Description, handle.Manifest, evalOpts),
		func(ctx context.Context, in EvalInput) (EvalOutput, error) {
			return RunEval(ctx, handle.Runtime, in, evalOpts)
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

func NewLazyRegistrar[Scope any, Meta any](spec EnvironmentSpec[Scope, Meta], resolve ScopeResolver[Scope], opts EvalOptions) func(reg tools.ToolRegistry) error {
	return func(reg tools.ToolRegistry) error {
		if reg == nil {
			return fmt.Errorf("tool registry is nil")
		}
		if resolve == nil {
			return fmt.Errorf("scope resolver is nil")
		}
		evalOpts := resolveEvalOptions(spec.DefaultEval, opts)
		def, err := tools.NewToolFromFunc(
			spec.Tool.Name,
			BuildDescription(spec.Tool.Description, EnvironmentManifest{}, evalOpts),
			func(ctx context.Context, in EvalInput) (EvalOutput, error) {
				scope, err := resolve(ctx)
				if err != nil {
					return EvalOutput{Error: err.Error()}, nil
				}
				handle, err := BuildRuntime(ctx, spec, scope)
				if err != nil {
					return EvalOutput{Error: err.Error()}, nil
				}
				defer func() {
					if handle.Cleanup != nil {
						_ = handle.Cleanup()
					}
				}()
				return RunEval(ctx, handle.Runtime, in, evalOpts)
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
