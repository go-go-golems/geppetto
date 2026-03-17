package scopedjs

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/inference/tools"
)

func RegisterPrebuilt[Scope any, Meta any](reg tools.ToolRegistry, spec EnvironmentSpec[Scope, Meta], handle *BuildResult[Meta], opts EvalOptionOverrides) error {
	if reg == nil {
		return fmt.Errorf("tool registry is nil")
	}
	if handle == nil || handle.Runtime == nil {
		return fmt.Errorf("build result runtime is nil")
	}
	evalOpts := resolveEvalOptions(spec.DefaultEval, opts)
	executor := executorFromBuildResult(handle)
	if executor == nil {
		return fmt.Errorf("build result executor is nil")
	}
	def, err := tools.NewToolFromFunc(
		spec.Tool.Name,
		BuildDescription(spec.Tool.Description, handle.Manifest, "Calls reuse one prebuilt runtime instance, so runtime state can persist across calls."),
		func(ctx context.Context, in EvalInput) (EvalOutput, error) {
			return executor.RunEval(ctx, in, evalOpts)
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

func NewLazyRegistrar[Scope any, Meta any](spec EnvironmentSpec[Scope, Meta], resolve ScopeResolver[Scope], opts EvalOptionOverrides) func(reg tools.ToolRegistry) error {
	return func(reg tools.ToolRegistry) error {
		if reg == nil {
			return fmt.Errorf("tool registry is nil")
		}
		if resolve == nil {
			return fmt.Errorf("scope resolver is nil")
		}
		evalOpts := resolveEvalOptions(spec.DefaultEval, opts)
		manifest, err := describeManifest(spec)
		if err != nil {
			return fmt.Errorf("describe %s tool: %w", spec.Tool.Name, err)
		}
		def, err := tools.NewToolFromFunc(
			spec.Tool.Name,
			BuildDescription(spec.Tool.Description, manifest, "Each call builds a fresh runtime from the resolved scope."),
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
				executor := executorFromBuildResult(handle)
				if executor == nil {
					return EvalOutput{Error: "build result executor is nil"}, nil
				}
				return executor.RunEval(ctx, in, evalOpts)
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

func describeManifest[Scope any, Meta any](spec EnvironmentSpec[Scope, Meta]) (EnvironmentManifest, error) {
	if spec.Describe == nil {
		return EnvironmentManifest{}, nil
	}
	manifest, err := spec.Describe()
	if err != nil {
		return EnvironmentManifest{}, err
	}
	return normalizeManifest(manifest), nil
}

func normalizeManifest(in EnvironmentManifest) EnvironmentManifest {
	out := cloneManifest(in)
	for i := range out.Modules {
		out.Modules[i].Name = strings.TrimSpace(out.Modules[i].Name)
		out.Modules[i].Description = strings.TrimSpace(out.Modules[i].Description)
		out.Modules[i].Exports = NormalizeNonEmptyStrings(out.Modules[i].Exports)
	}
	for i := range out.Globals {
		out.Globals[i].Name = strings.TrimSpace(out.Globals[i].Name)
		out.Globals[i].Type = strings.TrimSpace(out.Globals[i].Type)
		out.Globals[i].Description = strings.TrimSpace(out.Globals[i].Description)
	}
	for i := range out.Helpers {
		out.Helpers[i].Name = strings.TrimSpace(out.Helpers[i].Name)
		out.Helpers[i].Signature = strings.TrimSpace(out.Helpers[i].Signature)
		out.Helpers[i].Description = strings.TrimSpace(out.Helpers[i].Description)
	}
	out.BootstrapFiles = NormalizeNonEmptyStrings(out.BootstrapFiles)
	return out
}
