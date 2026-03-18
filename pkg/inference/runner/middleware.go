package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	gepprofiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	gepmiddleware "github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/middlewarecfg"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop/enginebuilder"
)

func (r *Runner) buildEngine(ctx context.Context, runtime Runtime) (engine.Engine, []gepmiddleware.Middleware, error) {
	if runtime.InferenceSettings == nil {
		return nil, nil, ErrRuntimeInferenceSettingsNil
	}
	engineFactory := r.engineFactory
	if engineFactory == nil {
		return nil, nil, fmt.Errorf("build engine: engine factory is nil")
	}
	base, err := engineFactory(runtime.InferenceSettings)
	if err != nil {
		return nil, nil, err
	}
	return r.buildEngineFromBase(ctx, base, runtime)
}

func (r *Runner) buildEngineFromBase(ctx context.Context, base engine.Engine, runtime Runtime) (engine.Engine, []gepmiddleware.Middleware, error) {
	if ctx == nil {
		return nil, nil, fmt.Errorf("build engine: context is nil")
	}
	if base == nil {
		return nil, nil, fmt.Errorf("build engine: base engine is nil")
	}

	resolved, err := r.resolveMiddlewares(ctx, runtime)
	if err != nil {
		return nil, nil, err
	}

	mws := make([]gepmiddleware.Middleware, 0, 2+len(resolved))
	mws = append(mws, gepmiddleware.NewToolResultReorderMiddleware())
	mws = append(mws, resolved...)
	if strings.TrimSpace(runtime.SystemPrompt) != "" {
		mws = append(mws, gepmiddleware.NewSystemPromptMiddleware(runtime.SystemPrompt))
	}

	builder := &enginebuilder.Builder{
		Base:        base,
		Middlewares: mws,
	}
	eng, err := builder.Build(ctx, "")
	if err != nil {
		return nil, nil, err
	}
	return eng, mws, nil
}

func (r *Runner) resolveMiddlewares(ctx context.Context, runtime Runtime) ([]gepmiddleware.Middleware, error) {
	if len(runtime.Middlewares) > 0 {
		return append([]gepmiddleware.Middleware(nil), runtime.Middlewares...), nil
	}
	if len(runtime.MiddlewareUses) == 0 {
		return nil, nil
	}
	if r == nil || r.middlewareDefinitions == nil {
		return nil, fmt.Errorf("middleware definitions are not configured")
	}

	resolved := make([]middlewarecfg.ResolvedInstance, 0, len(runtime.MiddlewareUses))
	for i, use := range runtime.MiddlewareUses {
		def, ok := r.middlewareDefinitions.GetDefinition(use.Name)
		if !ok {
			return nil, fmt.Errorf("resolve middleware %s: unknown middleware %q", middlewarecfg.MiddlewareInstanceKey(use, i), use.Name)
		}

		cfg, err := normalizeMiddlewareConfig(use.Config)
		if err != nil {
			return nil, fmt.Errorf("resolve middleware %s: %w", middlewarecfg.MiddlewareInstanceKey(use, i), err)
		}

		sources := make([]middlewarecfg.Source, 0, 1)
		if len(cfg) > 0 {
			sources = append(sources, fixedPayloadSource{
				name:    "runtime",
				layer:   middlewarecfg.SourceLayerProfile,
				payload: cfg,
			})
		}

		resolver := middlewarecfg.NewResolver(sources...)
		resolvedCfg, err := resolver.Resolve(def, gepprofiles.MiddlewareUse{
			Name:    strings.TrimSpace(use.Name),
			ID:      strings.TrimSpace(use.ID),
			Enabled: cloneBoolPtr(use.Enabled),
		})
		if err != nil {
			return nil, fmt.Errorf("resolve middleware %s: %w", middlewarecfg.MiddlewareInstanceKey(use, i), err)
		}

		resolved = append(resolved, middlewarecfg.ResolvedInstance{
			Key: middlewarecfg.MiddlewareInstanceKey(use, i),
			Use: gepprofiles.MiddlewareUse{
				Name:    strings.TrimSpace(use.Name),
				ID:      strings.TrimSpace(use.ID),
				Enabled: cloneBoolPtr(use.Enabled),
			},
			Resolved: resolvedCfg,
			Def:      def,
		})
	}

	return middlewarecfg.BuildChain(ctx, r.middlewareBuildDeps, resolved)
}

type fixedPayloadSource struct {
	name    string
	layer   middlewarecfg.SourceLayer
	payload map[string]any
}

func (s fixedPayloadSource) Name() string {
	return s.name
}

func (s fixedPayloadSource) Layer() middlewarecfg.SourceLayer {
	return s.layer
}

func (s fixedPayloadSource) Payload(middlewarecfg.Definition, gepprofiles.MiddlewareUse) (map[string]any, bool, error) {
	if len(s.payload) == 0 {
		return nil, false, nil
	}
	return copyStringAnyMap(s.payload), true, nil
}

func normalizeMiddlewareConfig(raw any) (map[string]any, error) {
	if raw == nil {
		return nil, nil
	}
	if m, ok := raw.(map[string]any); ok {
		return copyStringAnyMap(m), nil
	}

	b, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal middleware config: %w", err)
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("decode middleware config object: %w", err)
	}
	if m == nil {
		return nil, fmt.Errorf("middleware config must decode to an object")
	}
	return m, nil
}

func copyStringAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneBoolPtr(in *bool) *bool {
	if in == nil {
		return nil
	}
	v := *in
	return &v
}
