package toolloop

import (
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
)

type EngineBuilderOption func(*EngineBuilder)

func NewEngineBuilder(opts ...EngineBuilderOption) *EngineBuilder {
	b := &EngineBuilder{}
	for _, opt := range opts {
		if opt != nil {
			opt(b)
		}
	}
	return b
}

func WithBase(base engine.Engine) EngineBuilderOption {
	return func(b *EngineBuilder) {
		b.Base = base
	}
}

func WithMiddlewares(mws ...middleware.Middleware) EngineBuilderOption {
	return func(b *EngineBuilder) {
		b.Middlewares = append(b.Middlewares, mws...)
	}
}

func WithToolRegistry(registry tools.ToolRegistry) EngineBuilderOption {
	return func(b *EngineBuilder) {
		b.Registry = registry
	}
}

func WithToolConfig(cfg ToolConfig) EngineBuilderOption {
	return func(b *EngineBuilder) {
		b.ToolConfig = &cfg
	}
}

func WithEventSinks(sinks ...events.EventSink) EngineBuilderOption {
	return func(b *EngineBuilder) {
		b.EventSinks = append(b.EventSinks, sinks...)
	}
}

func WithEngineBuilderSnapshotHook(hook SnapshotHook) EngineBuilderOption {
	return func(b *EngineBuilder) {
		b.SnapshotHook = hook
	}
}

func WithStepControllerService(sc *StepController) EngineBuilderOption {
	return func(b *EngineBuilder) {
		b.StepController = sc
	}
}

func WithStepPauseTimeout(d time.Duration) EngineBuilderOption {
	return func(b *EngineBuilder) {
		b.StepPauseTimeout = d
	}
}

func WithPersister(persister TurnPersister) EngineBuilderOption {
	return func(b *EngineBuilder) {
		b.Persister = persister
	}
}
