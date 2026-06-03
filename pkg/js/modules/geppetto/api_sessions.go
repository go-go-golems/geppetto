package geppetto

import (
	"context"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop/enginebuilder"
)

func (b *builderRef) buildEngineBuilder() (session.EngineBuilder, error) {
	if b.base == nil {
		return nil, fmt.Errorf("builder has no engine configured")
	}
	baseRegistry := b.registry
	if baseRegistry == nil && len(b.runtimeToolNames) > 0 && b.api != nil {
		baseRegistry = b.api.goToolRegistry
	}
	registry, err := materializeToolRegistry(baseRegistry, b.runtimeToolNames)
	if err != nil {
		return nil, err
	}
	opts := []enginebuilder.Option{enginebuilder.WithBase(b.base)}
	if len(b.middlewares) > 0 {
		opts = append(opts, enginebuilder.WithMiddlewares(b.middlewares...))
	}
	if registry != nil {
		opts = append(opts, enginebuilder.WithToolRegistry(registry))
	}
	if b.loopCfg != nil {
		opts = append(opts, enginebuilder.WithLoopConfig(*b.loopCfg))
	}
	if b.toolCfg != nil {
		opts = append(opts, enginebuilder.WithToolConfig(*b.toolCfg))
	}
	if b.toolExecutor != nil {
		opts = append(opts, enginebuilder.WithToolExecutor(b.toolExecutor))
	}
	if len(b.eventSinks) > 0 {
		opts = append(opts, enginebuilder.WithEventSinks(b.eventSinks...))
	}
	if b.snapshotHook != nil {
		opts = append(opts, enginebuilder.WithSnapshotHook(b.snapshotHook))
	}
	if b.persister != nil {
		opts = append(opts, enginebuilder.WithPersister(b.persister))
	}
	return enginebuilder.New(opts...), nil
}

func (m *moduleRuntime) requireEventSink(v goja.Value) (events.EventSink, error) {
	ref := m.getRef(v)
	if sink, ok := ref.(events.EventSink); ok {
		return sink, nil
	}
	return nil, fmt.Errorf("expected event sink reference or go-go-goja EventEmitter, got %T (value: %v)", ref, v)
}

func (sr *sessionRef) buildRunContext(opts runOptions) (context.Context, context.CancelFunc, error) {
	ctx := context.Background()
	if sr != nil && sr.api != nil {
		ctx = sr.api.runtimeContext()
	}
	if opts.timeoutMs < 0 {
		return nil, nil, fmt.Errorf("timeoutMs must be >= 0")
	}
	if opts.timeoutMs > 0 {
		ctxWithTimeout, timeoutCancel := context.WithTimeout(ctx, time.Duration(opts.timeoutMs)*time.Millisecond)
		if len(opts.tags) > 0 {
			ctxWithTimeout = session.WithRunTags(ctxWithTimeout, opts.tags)
		}
		return ctxWithTimeout, timeoutCancel, nil
	}
	if len(opts.tags) > 0 {
		ctx = session.WithRunTags(ctx, opts.tags)
	}
	return ctx, func() {}, nil
}

func (m *moduleRuntime) parseRunOptions(args []goja.Value, idx int) (runOptions, error) {
	out := runOptions{}
	if len(args) <= idx || args[idx] == nil || goja.IsUndefined(args[idx]) || goja.IsNull(args[idx]) {
		return out, nil
	}
	raw := decodeMap(args[idx].Export())
	if raw == nil {
		return out, fmt.Errorf("run options must be an object")
	}
	out.timeoutMs = toInt(raw["timeoutMs"], 0)
	if out.timeoutMs < 0 {
		return out, fmt.Errorf("timeoutMs must be >= 0")
	}
	if tags := decodeMap(raw["tags"]); len(tags) > 0 {
		out.tags = cloneJSONMap(tags)
	}
	return out, nil
}
