package enginebuilder

import (
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
)

type Option func(*Builder)

func New(opts ...Option) *Builder {
	b := &Builder{}
	for _, opt := range opts {
		if opt != nil {
			opt(b)
		}
	}
	return b
}

func WithBase(base engine.Engine) Option {
	return func(b *Builder) {
		b.Base = base
	}
}

func WithMiddlewares(mws ...middleware.Middleware) Option {
	return func(b *Builder) {
		b.Middlewares = append(b.Middlewares, mws...)
	}
}

func WithToolRegistry(registry tools.ToolRegistry) Option {
	return func(b *Builder) {
		b.Registry = registry
	}
}

func WithToolConfig(cfg toolloop.ToolConfig) Option {
	return func(b *Builder) {
		b.ToolConfig = &cfg
	}
}

func WithEventSinks(sinks ...events.EventSink) Option {
	return func(b *Builder) {
		b.EventSinks = append(b.EventSinks, sinks...)
	}
}

func WithSnapshotHook(hook toolloop.SnapshotHook) Option {
	return func(b *Builder) {
		b.SnapshotHook = hook
	}
}

func WithStepController(sc *toolloop.StepController) Option {
	return func(b *Builder) {
		b.StepController = sc
	}
}

func WithStepPauseTimeout(d time.Duration) Option {
	return func(b *Builder) {
		b.StepPauseTimeout = d
	}
}

func WithPersister(persister TurnPersister) Option {
	return func(b *Builder) {
		b.Persister = persister
	}
}
