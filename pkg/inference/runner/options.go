package runner

import (
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
	"github.com/go-go-golems/geppetto/pkg/inference/middlewarecfg"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop/enginebuilder"
	geptools "github.com/go-go-golems/geppetto/pkg/inference/tools"
)

// Option mutates a runner during construction.
type Option func(*Runner)

// New constructs a Runner with sensible defaults.
func New(opts ...Option) *Runner {
	r := &Runner{
		loopConfig:    toolloop.DefaultLoopConfig(),
		toolConfig:    geptools.DefaultToolConfig(),
		engineFactory: factory.NewEngineFromSettings,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(r)
		}
	}
	return r
}

func WithMiddlewareDefinitions(defs middlewarecfg.DefinitionRegistry) Option {
	return func(r *Runner) {
		r.middlewareDefinitions = defs
	}
}

func WithMiddlewareBuildDeps(deps middlewarecfg.BuildDeps) Option {
	return func(r *Runner) {
		r.middlewareBuildDeps = deps.Clone()
	}
}

func WithDefaultLoopConfig(cfg toolloop.LoopConfig) Option {
	return func(r *Runner) {
		r.loopConfig = cfg
	}
}

func WithDefaultToolConfig(cfg geptools.ToolConfig) Option {
	return func(r *Runner) {
		r.toolConfig = cfg
	}
}

func WithToolExecutor(exec geptools.ToolExecutor) Option {
	return func(r *Runner) {
		r.toolExecutor = exec
	}
}

func WithToolRegistrars(registrars ...ToolRegistrar) Option {
	return func(r *Runner) {
		r.toolRegistrars = append(r.toolRegistrars, registrars...)
	}
}

func WithFuncTool(name, description string, fn any) Option {
	return WithToolRegistrars(FuncTool(name, description, fn))
}

func WithEventSinks(sinks ...events.EventSink) Option {
	return func(r *Runner) {
		r.eventSinks = append(r.eventSinks, sinks...)
	}
}

func WithSnapshotHook(hook toolloop.SnapshotHook) Option {
	return func(r *Runner) {
		r.snapshotHook = hook
	}
}

func WithPersister(persister enginebuilder.TurnPersister) Option {
	return func(r *Runner) {
		r.persister = persister
	}
}

func WithStepController(sc *toolloop.StepController) Option {
	return func(r *Runner) {
		r.stepController = sc
	}
}

func WithStepPauseTimeout(d time.Duration) Option {
	return func(r *Runner) {
		r.stepPauseTimeout = d
	}
}
