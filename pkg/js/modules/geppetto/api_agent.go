package geppetto

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/events"
	enginefactory "github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

type agentBuilderRef struct {
	api *moduleRuntime

	name        string
	base        *engineRef
	settings    *inferenceSettingsRef
	middlewares []middleware.Middleware

	registry           tools.ToolRegistry
	runtimeToolNames   []string
	loopOptions        map[string]any
	eventSinks         []events.EventSink
	eventEmitterValues []goja.Value
	runDefaults        runOptions
}

type agentRef struct {
	api *moduleRuntime

	name               string
	base               *engineRef
	middlewares        []middleware.Middleware
	registry           tools.ToolRegistry
	runtimeToolNames   []string
	loopOptions        map[string]any
	eventSinks         []events.EventSink
	eventEmitterValues []goja.Value
	runDefaults        runOptions
	runtimeMetadata    map[string]any
}

type runResultRef struct {
	api           *moduleRuntime
	inputTurn     *turns.Turn
	effectiveTurn *turns.Turn
	outputTurn    *turns.Turn
	events        []any
}

func (m *moduleRuntime) agentBuilder(call goja.FunctionCall) goja.Value {
	return m.newAgentBuilderObject(&agentBuilderRef{
		api:        m,
		eventSinks: append([]events.EventSink(nil), m.defaultEventSinks...),
	})
}

func (m *moduleRuntime) newAgentBuilderObject(ref *agentBuilderRef) *goja.Object {
	if ref == nil {
		ref = &agentBuilderRef{api: m}
	}
	ref.api = m
	o := m.vm.NewObject()
	m.attachRef(o, ref)
	m.mustSet(o, "name", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			ref.name = strings.TrimSpace(call.Arguments[0].String())
		}
		return o
	})
	m.mustSet(o, "inference", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
			panic(m.vm.NewTypeError("agent().inference requires a registry-resolved InferenceSettings wrapper"))
		}
		settings, err := m.requireInferenceSettingsRef(call.Arguments[0])
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		ref.settings = settings
		return o
	})
	m.mustSet(o, "engine", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(m.vm.NewTypeError("agent().engine requires engine argument"))
		}
		eng, err := m.requireEngineRef(call.Arguments[0])
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		ref.base = eng
		return o
	})
	m.mustSet(o, "middleware", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(m.vm.NewTypeError("agent().middleware requires middleware argument"))
		}
		mw, err := m.resolveMiddleware(call.Arguments[0])
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		ref.middlewares = append(ref.middlewares, mw)
		return o
	})
	m.mustSet(o, "goMiddleware", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(m.vm.NewTypeError("agent().goMiddleware requires middleware name"))
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
		ref.middlewares = append(ref.middlewares, mw)
		return o
	})
	m.mustSet(o, "tool", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(m.vm.NewTypeError("agent().tool requires a tool registry until gp.tool() lands"))
		}
		reg, err := m.requireToolRegistry(call.Arguments[0])
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		ref.registry = reg
		return o
	})
	m.mustSet(o, "goTool", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(m.vm.NewTypeError("agent().goTool requires tool name"))
		}
		name := strings.TrimSpace(call.Arguments[0].String())
		if name == "" {
			panic(m.vm.NewTypeError("agent().goTool name must not be empty"))
		}
		ref.runtimeToolNames = append(ref.runtimeToolNames, name)
		return o
	})
	m.mustSet(o, "toolLoop", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
			ref.loopOptions = cloneJSONMap(decodeMap(call.Arguments[0].Export()))
		}
		return o
	})
	m.mustSet(o, "events", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(m.vm.NewTypeError("agent().events requires event sink argument"))
		}
		if m.isEventEmitterValue(call.Arguments[0]) {
			ref.eventEmitterValues = append(ref.eventEmitterValues, call.Arguments[0])
			return o
		}
		sink, err := m.requireEventSink(call.Arguments[0])
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		ref.eventSinks = append(ref.eventSinks, sink)
		return o
	})
	m.mustSet(o, "runDefaults", func(call goja.FunctionCall) goja.Value {
		opts, err := m.parseRunOptions(call.Arguments, 0)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		ref.runDefaults = opts
		return o
	})
	m.mustSet(o, "build", func(goja.FunctionCall) goja.Value {
		agent, err := ref.build()
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.newAgentObject(agent)
	})
	return o
}

func (b *agentBuilderRef) build() (*agentRef, error) {
	if b == nil || b.api == nil {
		return nil, fmt.Errorf("agent builder is not initialized")
	}
	base := b.base
	if base == nil && b.settings != nil {
		settings := cloneInferenceSettings(b.settings.settings)
		ensureInferenceSettingsProviderDefaults(settings)
		eng, err := enginefactory.NewEngineFromSettings(settings)
		if err != nil {
			return nil, err
		}
		base = &engineRef{Name: "inferenceSettings", Engine: eng, ModelInfo: settings.ModelInfo.Clone(), Metadata: b.settings.provenance.toMap()}
	}
	if base == nil || base.Engine == nil {
		return nil, fmt.Errorf("agent build requires engine(...) or inference(settings)")
	}
	return &agentRef{
		api:                b.api,
		name:               b.name,
		base:               base,
		middlewares:        append([]middleware.Middleware(nil), b.middlewares...),
		registry:           b.registry,
		runtimeToolNames:   append([]string(nil), b.runtimeToolNames...),
		loopOptions:        cloneJSONMap(b.loopOptions),
		eventSinks:         append([]events.EventSink(nil), b.eventSinks...),
		eventEmitterValues: append([]goja.Value(nil), b.eventEmitterValues...),
		runDefaults:        b.runDefaults,
		runtimeMetadata:    map[string]any{"agentName": b.name},
	}, nil
}

func (m *moduleRuntime) newAgentObject(ref *agentRef) *goja.Object {
	o := m.vm.NewObject()
	m.attachRef(o, ref)
	m.mustSet(o, "name", ref.name)
	m.mustSet(o, "run", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
			panic(m.vm.NewTypeError("agent.run requires a Go-owned Turn wrapper"))
		}
		turn, err := m.requireTurnRef(call.Arguments[0])
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		opts, err := m.parseAgentRunOptions(ref.runDefaults, call.Arguments, 1)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		result, err := ref.runSync(turn.turn, opts)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.newRunResultObject(result)
	})
	m.mustSet(o, "runAsync", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
			panic(m.vm.NewTypeError("agent.runAsync requires a Go-owned Turn wrapper"))
		}
		turn, err := m.requireTurnRef(call.Arguments[0])
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		opts, err := m.parseAgentRunOptions(ref.runDefaults, call.Arguments, 1)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return ref.startAsync(turn.turn, opts)
	})
	return o
}

func (m *moduleRuntime) parseAgentRunOptions(defaults runOptions, args []goja.Value, idx int) (runOptions, error) {
	extra, err := m.parseRunOptions(args, idx)
	if err != nil {
		return runOptions{}, err
	}
	out := defaults
	if extra.timeoutMs != 0 {
		out.timeoutMs = extra.timeoutMs
	}
	if len(extra.tags) > 0 {
		out.tags = mergeRuntimeMetadata(out.tags, extra.tags)
	}
	return out, nil
}

func (a *agentRef) buildSession(runScopedEventSinks []events.EventSink) (*sessionRef, error) {
	eventSinks := append([]events.EventSink(nil), a.eventSinks...)
	eventSinks = append(eventSinks, runScopedEventSinks...)
	b := &builderRef{
		api:              a.api,
		base:             a.base.Engine,
		middlewares:      append([]middleware.Middleware(nil), a.middlewares...),
		registry:         a.registry,
		runtimeToolNames: append([]string(nil), a.runtimeToolNames...),
		runtimeMetadata:  cloneJSONMap(a.runtimeMetadata),
		eventSinks:       eventSinks,
		snapshotHook:     a.api.defaultSnapshotHook,
		persister:        a.api.defaultPersister,
	}
	if len(a.loopOptions) > 0 {
		a.api.applyToolLoopSettings(b, a.loopOptions, a.api.vm.ToValue(a.loopOptions))
	}
	return b.buildSession()
}

type startedAgentRun struct {
	handle        *session.ExecutionHandle
	inputSnapshot *turns.Turn
	effectiveTurn *turns.Turn
	cancel        context.CancelFunc
}

func (a *agentRef) startRun(input *turns.Turn, opts runOptions, runScopedEventSinks []events.EventSink) (*startedAgentRun, error) {
	if input == nil {
		return nil, fmt.Errorf("agent run requires turn")
	}
	sr, err := a.buildSession(runScopedEventSinks)
	if err != nil {
		return nil, err
	}
	inputSnapshot := input.Clone()
	seed := input.Clone()
	stampTurnRuntimeMetadata(seed, sr.runtimeMetadata)
	effective := seed.Clone()
	sr.session.Append(seed)
	ctx, cancel, err := sr.buildRunContext(opts)
	if err != nil {
		return nil, err
	}
	handle, err := sr.session.StartInference(ctx)
	if err != nil {
		cancel()
		return nil, err
	}
	return &startedAgentRun{handle: handle, inputSnapshot: inputSnapshot, effectiveTurn: effective, cancel: cancel}, nil
}

func (a *agentRef) runSync(input *turns.Turn, opts runOptions) (*runResultRef, error) {
	runScopedEventSinks, closers, err := a.newRunScopedEventEmitterSinks()
	if err != nil {
		return nil, err
	}
	started, err := a.startRun(input, opts, runScopedEventSinks)
	if err != nil {
		closeRunScopedEventEmitterSinks(context.Background(), closers)
		return nil, err
	}
	defer started.cancel()
	out, err := started.handle.Wait()
	a.closeRunScopedEventEmitterSinksAfterOwnerQueue(closers)
	if err != nil {
		return nil, err
	}
	return &runResultRef{api: a.api, inputTurn: started.inputSnapshot, effectiveTurn: started.effectiveTurn, outputTurn: out.Clone()}, nil
}

func (a *agentRef) startAsync(input *turns.Turn, opts runOptions) goja.Value {
	if _, err := a.api.requireBridge("agent.runAsync"); err != nil {
		panic(a.api.vm.NewTypeError(err.Error()))
	}
	promise, resolve, reject := a.api.vm.NewPromise()
	handleObj := a.api.vm.NewObject()

	var stateMu sync.Mutex
	var activeHandle *session.ExecutionHandle
	var activeCancel context.CancelFunc
	canceled := false
	cancelActive := func() {
		stateMu.Lock()
		canceled = true
		h := activeHandle
		cancel := activeCancel
		stateMu.Unlock()
		if h != nil {
			h.Cancel()
		}
		if cancel != nil {
			cancel()
		}
	}
	setActive := func(started *startedAgentRun) {
		if started == nil {
			return
		}
		stateMu.Lock()
		activeHandle = started.handle
		activeCancel = started.cancel
		shouldCancel := canceled
		stateMu.Unlock()
		if shouldCancel {
			started.handle.Cancel()
			started.cancel()
		}
	}
	clearActive := func() {
		stateMu.Lock()
		activeHandle = nil
		activeCancel = nil
		stateMu.Unlock()
	}

	a.api.mustSet(handleObj, "promise", promise)
	a.api.mustSet(handleObj, "cancel", func(goja.FunctionCall) goja.Value {
		cancelActive()
		return goja.Undefined()
	})
	a.api.mustSet(handleObj, "close", func(goja.FunctionCall) goja.Value {
		cancelActive()
		return goja.Undefined()
	})

	runScopedEventSinks, closers, err := a.newRunScopedEventEmitterSinks()
	if err != nil {
		_ = reject(a.api.vm.ToValue(err.Error()))
		return handleObj
	}
	started, err := a.startRun(input, opts, runScopedEventSinks)
	if err != nil {
		closeRunScopedEventEmitterSinks(context.Background(), closers)
		_ = reject(a.api.vm.ToValue(err.Error()))
		return handleObj
	}
	setActive(started)

	go func() {
		defer clearActive()
		defer started.cancel()
		out, waitErr := started.handle.Wait()
		postErr := a.api.postOnOwner(context.Background(), "agent.runAsync.settle", func(context.Context) {
			defer closeRunScopedEventEmitterSinks(context.Background(), closers)
			if waitErr != nil {
				_ = reject(a.api.vm.ToValue(waitErr.Error()))
				return
			}
			_ = resolve(a.api.newRunResultObject(&runResultRef{
				api:           a.api,
				inputTurn:     started.inputSnapshot,
				effectiveTurn: started.effectiveTurn,
				outputTurn:    out.Clone(),
			}))
		})
		if postErr != nil {
			closeRunScopedEventEmitterSinks(context.Background(), closers)
			a.api.logger.Error().Err(postErr).Msg("agent.runAsync: failed to settle promise on owner thread")
		}
	}()
	return handleObj
}

func (m *moduleRuntime) newRunResultObject(ref *runResultRef) *goja.Object {
	if ref == nil {
		ref = &runResultRef{api: m}
	}
	ref.api = m
	o := m.vm.NewObject()
	m.attachRef(o, ref)
	m.mustSet(o, "inputTurn", func(goja.FunctionCall) goja.Value {
		return m.newTurnObject(&turnRef{api: m, turn: ref.inputTurn.Clone()})
	})
	m.mustSet(o, "effectiveTurn", func(goja.FunctionCall) goja.Value {
		return m.newTurnObject(&turnRef{api: m, turn: ref.effectiveTurn.Clone()})
	})
	m.mustSet(o, "outputTurn", func(goja.FunctionCall) goja.Value {
		return m.newTurnObject(&turnRef{api: m, turn: ref.outputTurn.Clone()})
	})
	m.mustSet(o, "text", func(goja.FunctionCall) goja.Value { return m.vm.ToValue(turnText(ref.outputTurn)) })
	m.mustSet(o, "usage", func(goja.FunctionCall) goja.Value { return goja.Null() })
	m.mustSet(o, "stopReason", func(goja.FunctionCall) goja.Value { return goja.Null() })
	m.mustSet(o, "events", func(goja.FunctionCall) goja.Value { return m.toJSValue(ref.events) })
	m.mustSet(o, "toJSON", func(goja.FunctionCall) goja.Value {
		return m.toJSValue(map[string]any{
			"inputTurn":     m.encodeTurn(ref.inputTurn),
			"effectiveTurn": m.encodeTurn(ref.effectiveTurn),
			"outputTurn":    m.encodeTurn(ref.outputTurn),
			"text":          turnText(ref.outputTurn),
		})
	})
	return o
}

func turnText(t *turns.Turn) string {
	if t == nil {
		return ""
	}
	parts := []string{}
	for _, block := range t.Blocks {
		if block.Kind != turns.BlockKindLLMText && block.Role != turns.RoleAssistant {
			continue
		}
		if text, ok := block.Payload[turns.PayloadKeyText].(string); ok && text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}
