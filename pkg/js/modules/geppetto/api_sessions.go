package geppetto

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop/enginebuilder"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

func (m *moduleRuntime) createBuilder(call goja.FunctionCall) goja.Value {
	b := m.newBuilderRef()
	if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
		if err := m.applyBuilderOptions(b, call.Arguments[0]); err != nil {
			panic(m.vm.NewGoError(err))
		}
	}
	return m.newBuilderObject(b)
}

func (m *moduleRuntime) createSession(call goja.FunctionCall) goja.Value {
	b := m.newBuilderRef()
	if len(call.Arguments) == 0 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
		panic(m.vm.NewTypeError("createSession requires options object with engine"))
	}
	if err := m.applyBuilderOptions(b, call.Arguments[0]); err != nil {
		panic(m.vm.NewGoError(err))
	}
	sr, err := b.buildSession()
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.newSessionObject(sr)
}

func (m *moduleRuntime) runInference(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(m.vm.NewTypeError("runInference requires (engine, turn[, options])"))
	}
	engineRef, err := m.requireEngineRef(call.Arguments[0])
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	turn, err := m.decodeTurnValue(call.Arguments[1])
	if err != nil {
		panic(m.vm.NewGoError(err))
	}

	b := &builderRef{
		api:          m,
		base:         engineRef.Engine,
		eventSinks:   append([]events.EventSink(nil), m.defaultEventSinks...),
		snapshotHook: m.defaultSnapshotHook,
		persister:    m.defaultPersister,
	}
	if len(call.Arguments) > 2 && !goja.IsUndefined(call.Arguments[2]) && !goja.IsNull(call.Arguments[2]) {
		if err := m.applyBuilderOptions(b, call.Arguments[2]); err != nil {
			panic(m.vm.NewGoError(err))
		}
	}
	sr, err := b.buildSession()
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	sr.session.Append(turn)
	out, err := sr.runSync(nil, runOptions{})
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	v, err := m.encodeTurnValue(out)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return v
}

func (m *moduleRuntime) newBuilderObject(b *builderRef) goja.Value {
	o := m.vm.NewObject()
	m.attachRef(o, b)
	m.mustSet(o, "withEngine", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(m.vm.NewTypeError("withEngine requires engine argument"))
		}
		ref, err := m.requireEngineRef(call.Arguments[0])
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		b.base = ref.Engine
		return o
	})
	m.mustSet(o, "useMiddleware", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(m.vm.NewTypeError("useMiddleware requires middleware argument"))
		}
		mw, err := m.resolveMiddleware(call.Arguments[0])
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		b.middlewares = append(b.middlewares, mw)
		return o
	})
	m.mustSet(o, "useGoMiddleware", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(m.vm.NewTypeError("useGoMiddleware requires middleware name"))
		}
		name := call.Arguments[0].String()
		var options map[string]any
		if len(call.Arguments) > 1 {
			options = decodeMap(call.Arguments[1].Export())
		}
		mw, err := m.resolveGoMiddleware(name, options)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		b.middlewares = append(b.middlewares, mw)
		return o
	})
	m.mustSet(o, "withTools", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(m.vm.NewTypeError("withTools requires registry argument"))
		}
		reg, err := m.requireToolRegistry(call.Arguments[0])
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		b.registry = reg
		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) && !goja.IsNull(call.Arguments[1]) {
			m.applyToolLoopSettings(b, decodeMap(call.Arguments[1].Export()), call.Arguments[1])
		}
		return o
	})
	m.mustSet(o, "withToolLoop", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
			m.applyToolLoopSettings(b, decodeMap(call.Arguments[0].Export()), call.Arguments[0])
		}
		return o
	})
	m.mustSet(o, "withToolHooks", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
			panic(m.vm.NewTypeError("withToolHooks requires hooks object"))
		}
		hooks, err := m.parseToolHooks(call.Arguments[0])
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		b.toolHooks = hooks
		if b.toolHooks == nil {
			b.toolExecutor = nil
			return o
		}
		cfg := tools.DefaultToolConfig()
		if b.toolCfg != nil {
			cfg = *b.toolCfg
		}
		b.toolExecutor = newJSToolHookExecutor(m, cfg, b.toolHooks)
		return o
	})
	m.mustSet(o, "withPersister", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(m.vm.NewTypeError("withPersister requires persister argument"))
		}
		persister, err := m.requireTurnPersister(call.Arguments[0])
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		b.persister = persister
		return o
	})
	m.mustSet(o, "withEventSink", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(m.vm.NewTypeError("withEventSink requires event sink argument"))
		}
		sink, err := m.requireEventSink(call.Arguments[0])
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		b.eventSinks = append(b.eventSinks, sink)
		return o
	})
	m.mustSet(o, "withSnapshotHook", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(m.vm.NewTypeError("withSnapshotHook requires snapshot hook argument"))
		}
		hook, err := m.requireSnapshotHook(call.Arguments[0])
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		b.snapshotHook = hook
		return o
	})
	m.mustSet(o, "buildSession", func(goja.FunctionCall) goja.Value {
		sr, err := b.buildSession()
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.newSessionObject(sr)
	})
	return o
}

func (b *builderRef) buildSession() (*sessionRef, error) {
	if b.base == nil {
		return nil, fmt.Errorf("builder has no engine configured")
	}
	opts := []enginebuilder.Option{
		enginebuilder.WithBase(b.base),
	}
	if len(b.middlewares) > 0 {
		opts = append(opts, enginebuilder.WithMiddlewares(b.middlewares...))
	}
	if b.registry != nil {
		opts = append(opts, enginebuilder.WithToolRegistry(b.registry))
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
	s := session.NewSession()
	s.Builder = enginebuilder.New(opts...)
	return &sessionRef{
		api:     b.api,
		session: s,
	}, nil
}

func (m *moduleRuntime) newBuilderRef() *builderRef {
	return &builderRef{
		api:          m,
		eventSinks:   append([]events.EventSink(nil), m.defaultEventSinks...),
		snapshotHook: m.defaultSnapshotHook,
		persister:    m.defaultPersister,
	}
}

func (m *moduleRuntime) requireTurnPersister(v goja.Value) (enginebuilder.TurnPersister, error) {
	ref := m.getRef(v)
	switch x := ref.(type) {
	case enginebuilder.TurnPersister:
		return x, nil
	default:
		return nil, fmt.Errorf("expected turn persister reference, got %T (value: %v)", ref, v)
	}
}

func (m *moduleRuntime) requireEventSink(v goja.Value) (events.EventSink, error) {
	ref := m.getRef(v)
	switch x := ref.(type) {
	case events.EventSink:
		return x, nil
	default:
		return nil, fmt.Errorf("expected event sink reference, got %T (value: %v)", ref, v)
	}
}

func (m *moduleRuntime) requireSnapshotHook(v goja.Value) (toolloop.SnapshotHook, error) {
	ref := m.getRef(v)
	switch x := ref.(type) {
	case toolloop.SnapshotHook:
		return x, nil
	default:
		return nil, fmt.Errorf("expected snapshot hook reference, got %T (value: %v)", ref, v)
	}
}

func (m *moduleRuntime) newSessionObject(sr *sessionRef) goja.Value {
	o := m.vm.NewObject()
	m.attachRef(o, sr)
	m.mustSet(o, "append", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(m.vm.NewTypeError("append requires a turn object"))
		}
		t, err := m.decodeTurnValue(call.Arguments[0])
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		sr.session.Append(t)
		v, err := m.encodeTurnValue(t)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return v
	})
	m.mustSet(o, "latest", func(goja.FunctionCall) goja.Value {
		latest := sr.session.Latest()
		v, err := m.encodeTurnValue(latest)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return v
	})
	m.mustSet(o, "turnCount", func(goja.FunctionCall) goja.Value {
		return m.vm.ToValue(sr.session.TurnCount())
	})
	m.mustSet(o, "turns", func(goja.FunctionCall) goja.Value {
		snapshots := sr.session.TurnsSnapshot()
		v, err := m.encodeTurnsValue(snapshots)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return v
	})
	m.mustSet(o, "getTurn", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(m.vm.NewTypeError("getTurn requires index"))
		}
		idx := toInt(call.Arguments[0].Export(), -1)
		t := sr.session.GetTurn(idx)
		if t == nil {
			return goja.Null()
		}
		v, err := m.encodeTurnValue(t)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return v
	})
	m.mustSet(o, "turnsRange", func(call goja.FunctionCall) goja.Value {
		start := 0
		end := sr.session.TurnCount()
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
			start = toInt(call.Arguments[0].Export(), 0)
		}
		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) && !goja.IsNull(call.Arguments[1]) {
			end = toInt(call.Arguments[1].Export(), end)
		}
		if start < 0 {
			start = 0
		}
		if end < start {
			end = start
		}
		total := sr.session.TurnCount()
		if start > total {
			start = total
		}
		if end > total {
			end = total
		}
		snapshots := sr.session.TurnsSnapshot()
		ranged := snapshots[start:end]
		v, err := m.encodeTurnsValue(ranged)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return v
	})
	m.mustSet(o, "isRunning", func(goja.FunctionCall) goja.Value {
		return m.vm.ToValue(sr.session.IsRunning())
	})
	m.mustSet(o, "cancelActive", func(goja.FunctionCall) goja.Value {
		if err := sr.session.CancelActive(); err != nil {
			panic(m.vm.NewGoError(err))
		}
		return goja.Undefined()
	})
	m.mustSet(o, "run", func(call goja.FunctionCall) goja.Value {
		var t *turns.Turn
		var err error
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
			t, err = m.decodeTurnValue(call.Arguments[0])
			if err != nil {
				panic(m.vm.NewGoError(err))
			}
		}
		opts, err := m.parseRunOptions(call.Arguments, 1)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		out, err := sr.runSync(t, opts)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		v, err := m.encodeTurnValue(out)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return v
	})
	m.mustSet(o, "start", func(call goja.FunctionCall) goja.Value {
		var t *turns.Turn
		var err error
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
			t, err = m.decodeTurnValue(call.Arguments[0])
			if err != nil {
				panic(m.vm.NewGoError(err))
			}
		}
		opts, err := m.parseRunOptions(call.Arguments, 1)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return sr.start(t, opts)
	})
	m.mustSet(o, "runAsync", func(call goja.FunctionCall) goja.Value {
		var t *turns.Turn
		var err error
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
			t, err = m.decodeTurnValue(call.Arguments[0])
			if err != nil {
				panic(m.vm.NewGoError(err))
			}
		}
		return sr.runAsync(t)
	})
	return o
}

func (sr *sessionRef) runSync(seed *turns.Turn, opts runOptions) (*turns.Turn, error) {
	if seed != nil {
		sr.session.Append(seed)
	}
	ctx, cancel, err := sr.buildRunContext(opts)
	if err != nil {
		return nil, err
	}
	defer cancel()
	handle, err := sr.session.StartInference(ctx)
	if err != nil {
		return nil, err
	}
	return handle.Wait()
}

func (sr *sessionRef) runAsync(seed *turns.Turn) goja.Value {
	if _, err := sr.api.requireBridge("runAsync"); err != nil {
		panic(sr.api.vm.NewTypeError(err.Error()))
	}
	promise, resolve, reject := sr.api.vm.NewPromise()

	go func() {
		out, err := sr.runSync(seed, runOptions{})
		postErr := sr.api.postOnOwner(context.Background(), "session.runAsync.settle", func(context.Context) {
			if err != nil {
				_ = reject(sr.api.vm.ToValue(err.Error()))
				return
			}
			v, encErr := sr.api.encodeTurnValue(out)
			if encErr != nil {
				_ = reject(sr.api.vm.ToValue(encErr.Error()))
				return
			}
			_ = resolve(v)
		})
		if postErr != nil {
			sr.api.logger.Error().Err(postErr).Msg("runAsync: failed to settle promise on owner thread")
		}
	}()

	return sr.api.vm.ToValue(promise)
}

func (sr *sessionRef) start(seed *turns.Turn, opts runOptions) goja.Value {
	if _, err := sr.api.requireBridge("start"); err != nil {
		panic(sr.api.vm.NewTypeError(err.Error()))
	}
	promise, resolve, reject := sr.api.vm.NewPromise()
	collector := newJSEventCollector(sr.api)
	handleObj := sr.api.vm.NewObject()

	var (
		cancelMu sync.RWMutex
		cancelFn context.CancelFunc
	)
	setCancel := func(fn context.CancelFunc) {
		cancelMu.Lock()
		cancelFn = fn
		cancelMu.Unlock()
	}
	getCancel := func() context.CancelFunc {
		cancelMu.RLock()
		defer cancelMu.RUnlock()
		return cancelFn
	}

	sr.api.mustSet(handleObj, "promise", promise)
	sr.api.mustSet(handleObj, "cancel", func(goja.FunctionCall) goja.Value {
		if fn := getCancel(); fn != nil {
			fn()
		}
		return goja.Undefined()
	})
	sr.api.mustSet(handleObj, "on", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(sr.api.vm.NewTypeError("on(eventType, callback) requires 2 arguments"))
		}
		eventType := call.Arguments[0].String()
		cb, ok := goja.AssertFunction(call.Arguments[1])
		if !ok {
			panic(sr.api.vm.NewTypeError("on() callback must be a function"))
		}
		collector.subscribe(eventType, cb)
		return handleObj
	})

	go func() {
		if seed != nil {
			sr.session.Append(seed)
		}
		ctx, cancel, err := sr.buildRunContext(opts)
		if err != nil {
			postErr := sr.api.postOnOwner(context.Background(), "session.start.reject.buildContext", func(context.Context) {
				_ = reject(sr.api.vm.ToValue(err.Error()))
			})
			if postErr != nil {
				sr.api.logger.Error().Err(postErr).Msg("start: failed to reject promise after context build error")
			}
			return
		}
		ctx = events.WithEventSinks(ctx, collector)
		setCancel(cancel)

		handle, err := sr.session.StartInference(ctx)
		if err != nil {
			cancel()
			postErr := sr.api.postOnOwner(context.Background(), "session.start.reject.startInference", func(context.Context) {
				collector.close()
				_ = reject(sr.api.vm.ToValue(err.Error()))
			})
			if postErr != nil {
				collector.close()
				sr.api.logger.Error().Err(postErr).Msg("start: failed to reject promise after start error")
			}
			return
		}
		out, err := handle.Wait()
		cancel()

		postErr := sr.api.postOnOwner(context.Background(), "session.start.settle", func(context.Context) {
			defer collector.close()
			if err != nil {
				_ = reject(sr.api.vm.ToValue(err.Error()))
				return
			}
			v, encErr := sr.api.encodeTurnValue(out)
			if encErr != nil {
				_ = reject(sr.api.vm.ToValue(encErr.Error()))
				return
			}
			_ = resolve(v)
		})
		if postErr != nil {
			collector.close()
			sr.api.logger.Error().Err(postErr).Msg("start: failed to settle promise on owner thread")
		}
	}()

	return handleObj
}

func (sr *sessionRef) buildRunContext(opts runOptions) (context.Context, context.CancelFunc, error) {
	ctx := context.Background()
	cancel := func() {}
	if opts.timeoutMs < 0 {
		return nil, nil, fmt.Errorf("timeoutMs must be >= 0")
	}
	if opts.timeoutMs > 0 {
		var timeoutCancel context.CancelFunc
		ctx, timeoutCancel = context.WithTimeout(ctx, time.Duration(opts.timeoutMs)*time.Millisecond)
		cancel = timeoutCancel
	}
	if len(opts.tags) > 0 {
		ctx = session.WithRunTags(ctx, opts.tags)
	}
	return ctx, cancel, nil
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
