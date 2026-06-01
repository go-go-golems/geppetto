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
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

type agentBuilderRef struct {
	api *moduleRuntime

	name        string
	base        *engineRef
	settings    *inferenceSettingsRef
	middlewares []middleware.Middleware

	registry         tools.ToolRegistry
	runtimeToolNames []string
	loopOptions      map[string]any
	eventSinks       []events.EventSink
	runDefaults      runOptions
}

type agentRef struct {
	api *moduleRuntime

	name             string
	base             *engineRef
	middlewares      []middleware.Middleware
	registry         tools.ToolRegistry
	runtimeToolNames []string
	loopOptions      map[string]any
	eventSinks       []events.EventSink
	runDefaults      runOptions
	runtimeMetadata  map[string]any
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
		api:              b.api,
		name:             b.name,
		base:             base,
		middlewares:      append([]middleware.Middleware(nil), b.middlewares...),
		registry:         b.registry,
		runtimeToolNames: append([]string(nil), b.runtimeToolNames...),
		loopOptions:      cloneJSONMap(b.loopOptions),
		eventSinks:       append([]events.EventSink(nil), b.eventSinks...),
		runDefaults:      b.runDefaults,
		runtimeMetadata:  map[string]any{"agentName": b.name},
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
	m.mustSet(o, "stream", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
			panic(m.vm.NewTypeError("agent.stream requires a Go-owned Turn wrapper"))
		}
		turn, err := m.requireTurnRef(call.Arguments[0])
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		opts, err := m.parseAgentRunOptions(ref.runDefaults, call.Arguments, 1)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return ref.start(turn.turn, opts)
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

func (a *agentRef) buildSession() (*sessionRef, error) {
	b := &builderRef{
		api:              a.api,
		base:             a.base.Engine,
		middlewares:      append([]middleware.Middleware(nil), a.middlewares...),
		registry:         a.registry,
		runtimeToolNames: append([]string(nil), a.runtimeToolNames...),
		runtimeMetadata:  cloneJSONMap(a.runtimeMetadata),
		eventSinks:       append([]events.EventSink(nil), a.eventSinks...),
		snapshotHook:     a.api.defaultSnapshotHook,
		persister:        a.api.defaultPersister,
	}
	if len(a.loopOptions) > 0 {
		a.api.applyToolLoopSettings(b, a.loopOptions, a.api.vm.ToValue(a.loopOptions))
	}
	return b.buildSession()
}

func (a *agentRef) runSync(input *turns.Turn, opts runOptions) (*runResultRef, error) {
	if input == nil {
		return nil, fmt.Errorf("agent.run requires turn")
	}
	sr, err := a.buildSession()
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
	defer cancel()
	handle, err := sr.session.StartInference(ctx)
	if err != nil {
		return nil, err
	}
	out, err := handle.Wait()
	if err != nil {
		return nil, err
	}
	return &runResultRef{api: a.api, inputTurn: inputSnapshot, effectiveTurn: effective, outputTurn: out.Clone()}, nil
}

func (a *agentRef) start(input *turns.Turn, opts runOptions) goja.Value {
	if _, err := a.api.requireBridge("agent.stream"); err != nil {
		panic(a.api.vm.NewTypeError(err.Error()))
	}
	promise, resolve, reject := a.api.vm.NewPromise()
	collector := newJSEventCollector(a.api)
	handleObj := a.api.vm.NewObject()

	var cancelMu sync.RWMutex
	var cancelFn context.CancelFunc
	setCancel := func(fn context.CancelFunc) { cancelMu.Lock(); cancelFn = fn; cancelMu.Unlock() }
	getCancel := func() context.CancelFunc { cancelMu.RLock(); defer cancelMu.RUnlock(); return cancelFn }

	a.api.mustSet(handleObj, "promise", promise)
	a.api.mustSet(handleObj, "cancel", func(goja.FunctionCall) goja.Value {
		if fn := getCancel(); fn != nil {
			fn()
		}
		return goja.Undefined()
	})
	a.api.mustSet(handleObj, "on", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(a.api.vm.NewTypeError("on(eventType, callback) requires 2 arguments"))
		}
		eventType := call.Arguments[0].String()
		cb, ok := goja.AssertFunction(call.Arguments[1])
		if !ok {
			panic(a.api.vm.NewTypeError("on() callback must be a function"))
		}
		collector.subscribe(eventType, cb)
		return handleObj
	})

	go func() {
		result, err := a.runSync(input, opts)
		postErr := a.api.postOnOwner(context.Background(), "agent.stream.settle", func(context.Context) {
			defer collector.close()
			if err != nil {
				_ = reject(a.api.vm.ToValue(err.Error()))
				return
			}
			_ = resolve(a.api.newRunResultObject(result))
		})
		if postErr != nil {
			collector.close()
			a.api.logger.Error().Err(postErr).Msg("agent.stream: failed to settle promise on owner thread")
		}
	}()
	setCancel(func() {})
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
