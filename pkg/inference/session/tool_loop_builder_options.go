package session

import (
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
)

type ToolLoopEngineBuilderOption func(*ToolLoopEngineBuilder)

func NewToolLoopEngineBuilder(opts ...ToolLoopEngineBuilderOption) *ToolLoopEngineBuilder {
	b := &ToolLoopEngineBuilder{}
	for _, opt := range opts {
		if opt != nil {
			opt(b)
		}
	}
	return b
}

func WithToolLoopBase(base engine.Engine) ToolLoopEngineBuilderOption {
	return func(b *ToolLoopEngineBuilder) {
		b.Base = base
	}
}

func WithToolLoopMiddlewares(mws ...middleware.Middleware) ToolLoopEngineBuilderOption {
	return func(b *ToolLoopEngineBuilder) {
		b.Middlewares = append(b.Middlewares, mws...)
	}
}

func WithToolLoopRegistry(registry tools.ToolRegistry) ToolLoopEngineBuilderOption {
	return func(b *ToolLoopEngineBuilder) {
		b.Registry = registry
	}
}

func WithToolLoopToolConfig(cfg toolloop.ToolConfig) ToolLoopEngineBuilderOption {
	return func(b *ToolLoopEngineBuilder) {
		b.ToolConfig = &cfg
	}
}

func WithToolLoopEventSinks(sinks ...events.EventSink) ToolLoopEngineBuilderOption {
	return func(b *ToolLoopEngineBuilder) {
		b.EventSinks = append(b.EventSinks, sinks...)
	}
}

func WithToolLoopSnapshotHook(hook toolloop.SnapshotHook) ToolLoopEngineBuilderOption {
	return func(b *ToolLoopEngineBuilder) {
		b.SnapshotHook = hook
	}
}

func WithToolLoopStepController(sc *toolloop.StepController) ToolLoopEngineBuilderOption {
	return func(b *ToolLoopEngineBuilder) {
		b.StepController = sc
	}
}

func WithToolLoopStepPauseTimeout(d time.Duration) ToolLoopEngineBuilderOption {
	return func(b *ToolLoopEngineBuilder) {
		b.StepPauseTimeout = d
	}
}

func WithToolLoopPersister(persister TurnPersister) ToolLoopEngineBuilderOption {
	return func(b *ToolLoopEngineBuilder) {
		b.Persister = persister
	}
}
