package geppetto

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	enginefactory "github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop/enginebuilder"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	aistepssettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	aitypes "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/invopop/jsonschema"
	"github.com/rs/zerolog"
)

type engineRef struct {
	Name   string
	Engine engine.Engine
}

type builderRef struct {
	api *moduleRuntime

	base        engine.Engine
	middlewares []middleware.Middleware

	registry     tools.ToolRegistry
	loopCfg      *toolloop.LoopConfig
	toolCfg      *tools.ToolConfig
	toolExecutor tools.ToolExecutor
	toolHooks    *jsToolHooks
}

type sessionRef struct {
	api     *moduleRuntime
	session *session.Session
}

type jsMiddlewareRef struct {
	Name string
	Fn   goja.Callable
}

type goMiddlewareRef struct {
	Name    string
	Options map[string]any
}

type toolRegistryRef struct {
	api *moduleRuntime

	registry   *tools.InMemoryToolRegistry
	goRegistry tools.ToolRegistry
}

type jsToolHooks struct {
	Before     goja.Callable
	After      goja.Callable
	OnError    goja.Callable
	FailOpen   bool
	RetryLimit int
}

type jsToolHookExecutor struct {
	*tools.BaseToolExecutor
	api   *moduleRuntime
	hooks *jsToolHooks
}

type echoEngine struct {
	reply string
}

func (e *echoEngine) RunInference(_ context.Context, t *turns.Turn) (*turns.Turn, error) {
	if t == nil {
		t = &turns.Turn{}
	}
	reply := strings.TrimSpace(e.reply)
	if reply == "" {
		reply = "READY"
	}
	turns.AppendBlock(t, turns.NewAssistantTextBlock(reply))
	return t, nil
}

type jsCallableEngine struct {
	api *moduleRuntime
	fn  goja.Callable
}

func (e *jsCallableEngine) RunInference(_ context.Context, t *turns.Turn) (*turns.Turn, error) {
	arg, err := e.api.encodeTurnValue(t)
	if err != nil {
		return nil, err
	}
	v, err := e.fn(goja.Undefined(), arg)
	if err != nil {
		return nil, fmt.Errorf("js engine callback: %w", err)
	}
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return t, nil
	}
	decoded, err := e.api.decodeTurnValue(v)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

func (m *moduleRuntime) createBuilder(call goja.FunctionCall) goja.Value {
	b := &builderRef{
		api: m,
	}
	if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
		if err := m.applyBuilderOptions(b, call.Arguments[0]); err != nil {
			panic(m.vm.NewGoError(err))
		}
	}
	return m.newBuilderObject(b)
}

func (m *moduleRuntime) createSession(call goja.FunctionCall) goja.Value {
	b := &builderRef{api: m}
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
		api:  m,
		base: engineRef.Engine,
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
	out, err := sr.runSync(nil)
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
	s := session.NewSession()
	s.Builder = enginebuilder.New(opts...)
	return &sessionRef{
		api:     b.api,
		session: s,
	}, nil
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
		out, err := sr.runSync(t)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		v, err := m.encodeTurnValue(out)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return v
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

func (sr *sessionRef) runSync(seed *turns.Turn) (*turns.Turn, error) {
	if seed != nil {
		sr.session.Append(seed)
	}
	handle, err := sr.session.StartInference(context.Background())
	if err != nil {
		return nil, err
	}
	return handle.Wait()
}

func (sr *sessionRef) runAsync(seed *turns.Turn) goja.Value {
	if sr.api.loop == nil {
		panic(sr.api.vm.NewTypeError("runAsync requires module options Loop to be configured"))
	}
	promise, resolve, reject := sr.api.vm.NewPromise()

	go func() {
		out, err := sr.runSync(seed)
		sr.api.loop.RunOnLoop(func(*goja.Runtime) {
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
	}()

	return sr.api.vm.ToValue(promise)
}

func (m *moduleRuntime) applyBuilderOptions(b *builderRef, v goja.Value) error {
	opts := decodeMap(v.Export())
	if opts == nil {
		return fmt.Errorf("options must be an object")
	}
	if engRaw, ok := opts["engine"]; ok && engRaw != nil {
		ref, err := m.requireEngineRef(m.vm.ToValue(engRaw))
		if err != nil {
			return err
		}
		b.base = ref.Engine
	}
	if mwsRaw, ok := opts["middlewares"]; ok && mwsRaw != nil {
		for _, item := range decodeSlice(mwsRaw) {
			mw, err := m.resolveMiddleware(m.vm.ToValue(item))
			if err != nil {
				return err
			}
			b.middlewares = append(b.middlewares, mw)
		}
	}
	if regRaw, ok := opts["tools"]; ok && regRaw != nil {
		reg, err := m.requireToolRegistry(m.vm.ToValue(regRaw))
		if err != nil {
			return err
		}
		b.registry = reg
	}
	if obj := v.ToObject(m.vm); obj != nil {
		if tlRaw := obj.Get("toolLoop"); tlRaw != nil && !goja.IsUndefined(tlRaw) && !goja.IsNull(tlRaw) {
			m.applyToolLoopSettings(b, decodeMap(tlRaw.Export()), tlRaw)
		}
		if thRaw := obj.Get("toolHooks"); thRaw != nil && !goja.IsUndefined(thRaw) && !goja.IsNull(thRaw) {
			hooks, err := m.parseToolHooks(thRaw)
			if err != nil {
				return err
			}
			b.toolHooks = hooks
		}
	} else if tlRaw, ok := opts["toolLoop"]; ok && tlRaw != nil {
		m.applyToolLoopSettings(b, decodeMap(tlRaw), m.vm.ToValue(tlRaw))
	}
	if b.toolHooks != nil && b.toolExecutor == nil {
		cfg := tools.DefaultToolConfig()
		if b.toolCfg != nil {
			cfg = *b.toolCfg
		}
		b.toolExecutor = newJSToolHookExecutor(m, cfg, b.toolHooks)
	}
	return nil
}

func toBool(v any, def bool) bool {
	switch x := v.(type) {
	case bool:
		return x
	default:
		return def
	}
}

func toInt(v any, def int) int {
	switch x := v.(type) {
	case int:
		return x
	case int32:
		return int(x)
	case int64:
		return int(x)
	case float64:
		return int(x)
	default:
		return def
	}
}

func toString(v any, def string) string {
	switch x := v.(type) {
	case string:
		return x
	default:
		return def
	}
}

func toFloat64(v any, def float64) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int32:
		return float64(x)
	case int64:
		return float64(x)
	default:
		return def
	}
}

func (m *moduleRuntime) applyToolLoopSettings(b *builderRef, cfg map[string]any, raw goja.Value) {
	if cfg == nil {
		return
	}
	enabled := toBool(cfg["enabled"], true)
	if !enabled {
		b.registry = nil
		b.toolExecutor = nil
		return
	}
	loopCfg := toolloop.NewLoopConfig()
	loopCfg.MaxIterations = toInt(cfg["maxIterations"], loopCfg.MaxIterations)
	b.loopCfg = &loopCfg

	toolCfg := tools.DefaultToolConfig()
	toolCfg.Enabled = enabled
	toolCfg.MaxParallelTools = toInt(cfg["maxParallelTools"], 1)
	if toolCfg.MaxParallelTools < 1 {
		toolCfg.MaxParallelTools = 1
	}
	toolCfg.ExecutionTimeout = time.Duration(toInt(cfg["executionTimeoutMs"], int(toolCfg.ExecutionTimeout.Milliseconds()))) * time.Millisecond
	toolCfg.ToolChoice = tools.ToolChoice(toString(cfg["toolChoice"], string(toolCfg.ToolChoice)))
	toolCfg.ToolErrorHandling = tools.ToolErrorHandling(toString(cfg["toolErrorHandling"], string(toolCfg.ToolErrorHandling)))
	toolCfg.RetryConfig.MaxRetries = toInt(cfg["retryMaxRetries"], toolCfg.RetryConfig.MaxRetries)
	if backoffMS := toInt(cfg["retryBackoffMs"], int(toolCfg.RetryConfig.BackoffBase.Milliseconds())); backoffMS > 0 {
		toolCfg.RetryConfig.BackoffBase = time.Duration(backoffMS) * time.Millisecond
	}
	if backoffFactor := toFloat64(cfg["retryBackoffFactor"], toolCfg.RetryConfig.BackoffFactor); backoffFactor > 0 {
		toolCfg.RetryConfig.BackoffFactor = backoffFactor
	}
	if allowed := decodeSlice(cfg["allowedTools"]); len(allowed) > 0 {
		names := make([]string, 0, len(allowed))
		for _, n := range allowed {
			if s, ok := n.(string); ok && s != "" {
				names = append(names, s)
			}
		}
		toolCfg.AllowedTools = names
	}
	b.toolCfg = &toolCfg

	var hooksRaw goja.Value
	if raw != nil && !goja.IsUndefined(raw) && !goja.IsNull(raw) {
		if obj := raw.ToObject(m.vm); obj != nil {
			if hv := obj.Get("hooks"); hv != nil && !goja.IsUndefined(hv) && !goja.IsNull(hv) {
				hooksRaw = hv
			}
		}
	}
	if hooksRaw != nil && !goja.IsUndefined(hooksRaw) && !goja.IsNull(hooksRaw) {
		hooks, err := m.parseToolHooks(hooksRaw)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		b.toolHooks = hooks
	}
	if b.toolHooks != nil {
		b.toolExecutor = newJSToolHookExecutor(m, toolCfg, b.toolHooks)
	}
}

func (m *moduleRuntime) parseToolHooks(v goja.Value) (*jsToolHooks, error) {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil, nil
	}
	obj := v.ToObject(m.vm)
	if obj == nil {
		return nil, fmt.Errorf("tool hooks must be an object")
	}
	h := &jsToolHooks{
		RetryLimit: 10,
	}
	if fn, ok := goja.AssertFunction(obj.Get("beforeToolCall")); ok {
		h.Before = fn
	}
	if fn, ok := goja.AssertFunction(obj.Get("afterToolCall")); ok {
		h.After = fn
	}
	if fn, ok := goja.AssertFunction(obj.Get("onToolError")); ok {
		h.OnError = fn
	}
	failOpen := false
	if hv := obj.Get("hookErrorPolicy"); hv != nil && !goja.IsUndefined(hv) && !goja.IsNull(hv) {
		if mode := strings.ToLower(strings.TrimSpace(toString(hv.Export(), ""))); mode != "" {
			failOpen = mode == "fail-open" || mode == "open"
		}
	}
	if hv := obj.Get("onHookError"); hv != nil && !goja.IsUndefined(hv) && !goja.IsNull(hv) {
		if mode := strings.ToLower(strings.TrimSpace(toString(hv.Export(), ""))); mode != "" {
			failOpen = mode == "fail-open" || mode == "open"
		}
	}
	if b := obj.Get("failOpen"); b != nil && !goja.IsUndefined(b) && !goja.IsNull(b) {
		failOpen = toBool(b.Export(), failOpen)
	}
	h.FailOpen = failOpen
	if rv := obj.Get("maxHookRetries"); rv != nil && !goja.IsUndefined(rv) && !goja.IsNull(rv) {
		if lim := toInt(rv.Export(), h.RetryLimit); lim > 0 {
			h.RetryLimit = lim
		}
	}
	if h.Before == nil && h.After == nil && h.OnError == nil {
		return nil, nil
	}
	return h, nil
}

func newJSToolHookExecutor(api *moduleRuntime, cfg tools.ToolConfig, hooks *jsToolHooks) tools.ToolExecutor {
	base := tools.NewBaseToolExecutor(cfg)
	exec := &jsToolHookExecutor{
		BaseToolExecutor: base,
		api:              api,
		hooks:            hooks,
	}
	exec.BaseToolExecutor.ToolExecutorExt = exec
	return exec
}

func (e *jsToolHookExecutor) hookError(where string, err error) error {
	if err == nil {
		return nil
	}
	if e.hooks != nil && e.hooks.FailOpen {
		e.api.logger.Warn().Err(err).Str("hook", where).Msg("js tool hook error ignored (fail-open)")
		return nil
	}
	return fmt.Errorf("%s hook: %w", where, err)
}

func decodeToolCallArgs(call tools.ToolCall) any {
	if len(call.Arguments) == 0 {
		return map[string]any{}
	}
	var out any
	if err := json.Unmarshal(call.Arguments, &out); err != nil {
		return map[string]any{}
	}
	if out == nil {
		return map[string]any{}
	}
	return out
}

func applyCallMutation(call *tools.ToolCall, mutation map[string]any) error {
	if call == nil || mutation == nil {
		return nil
	}
	if v, ok := mutation["id"]; ok {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			call.ID = s
		}
	}
	if v, ok := mutation["name"]; ok {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			call.Name = s
		}
	}
	var args any
	if v, ok := mutation["args"]; ok {
		args = v
	}
	if v, ok := mutation["arguments"]; ok {
		args = v
	}
	if args != nil {
		b, err := json.Marshal(args)
		if err != nil {
			return err
		}
		call.Arguments = b
	}
	return nil
}

func (e *jsToolHookExecutor) PreExecute(ctx context.Context, call tools.ToolCall, registry tools.ToolRegistry) (tools.ToolCall, error) {
	call, err := e.BaseToolExecutor.PreExecute(ctx, call, registry)
	if err != nil {
		return call, err
	}
	if e.hooks == nil || e.hooks.Before == nil {
		return call, nil
	}

	payload := map[string]any{
		"phase": "beforeToolCall",
		"call": map[string]any{
			"id":   call.ID,
			"name": call.Name,
			"args": decodeToolCallArgs(call),
		},
		"timestampMs": time.Now().UnixMilli(),
	}
	ret, err := e.hooks.Before(goja.Undefined(), e.api.toJSValue(payload))
	if hookErr := e.hookError("beforeToolCall", err); hookErr != nil {
		return call, hookErr
	}
	if err != nil || ret == nil || goja.IsUndefined(ret) || goja.IsNull(ret) {
		return call, nil
	}

	resp := decodeMap(ret.Export())
	if resp == nil {
		return call, nil
	}
	if action := strings.ToLower(strings.TrimSpace(toString(resp["action"], ""))); action == "abort" {
		return call, fmt.Errorf("%s", toString(resp["error"], "aborted by beforeToolCall"))
	}
	if abort, ok := resp["abort"].(bool); ok && abort {
		return call, fmt.Errorf("%s", toString(resp["error"], "aborted by beforeToolCall"))
	}
	if callMap := decodeMap(resp["call"]); callMap != nil {
		if err := applyCallMutation(&call, callMap); err != nil {
			return call, err
		}
	}
	if err := applyCallMutation(&call, resp); err != nil {
		return call, err
	}
	return call, nil
}

func (e *jsToolHookExecutor) PublishResult(ctx context.Context, call tools.ToolCall, res *tools.ToolResult) {
	if e.hooks == nil || e.hooks.After == nil {
		e.BaseToolExecutor.PublishResult(ctx, call, res)
		return
	}
	if res == nil {
		res = &tools.ToolResult{ID: call.ID}
	}
	payload := map[string]any{
		"phase": "afterToolCall",
		"call": map[string]any{
			"id":   call.ID,
			"name": call.Name,
			"args": decodeToolCallArgs(call),
		},
		"result": map[string]any{
			"value":      cloneJSONValue(res.Result),
			"error":      res.Error,
			"durationMs": res.Duration.Milliseconds(),
		},
		"timestampMs": time.Now().UnixMilli(),
	}
	ret, err := e.hooks.After(goja.Undefined(), e.api.toJSValue(payload))
	if hookErr := e.hookError("afterToolCall", err); hookErr != nil {
		res.Error = hookErr.Error()
		e.BaseToolExecutor.PublishResult(ctx, call, res)
		return
	}
	if err == nil && ret != nil && !goja.IsUndefined(ret) && !goja.IsNull(ret) {
		resp := decodeMap(ret.Export())
		if resp != nil {
			if action := strings.ToLower(strings.TrimSpace(toString(resp["action"], ""))); action == "abort" {
				res.Error = toString(resp["error"], "aborted by afterToolCall")
			}
			if abort, ok := resp["abort"].(bool); ok && abort {
				res.Error = toString(resp["error"], "aborted by afterToolCall")
			}
			if v, ok := resp["result"]; ok {
				res.Result = cloneJSONValue(v)
			}
			if v, ok := resp["error"]; ok {
				if s, ok := v.(string); ok {
					res.Error = s
				}
			}
		}
	}
	e.BaseToolExecutor.PublishResult(ctx, call, res)
}

func (e *jsToolHookExecutor) ShouldRetry(ctx context.Context, attempt int, res *tools.ToolResult, execErr error) (bool, time.Duration) {
	defaultRetry, defaultBackoff := e.BaseToolExecutor.ShouldRetry(ctx, attempt, res, execErr)
	if e.hooks == nil || e.hooks.OnError == nil {
		return defaultRetry, defaultBackoff
	}
	if e.hooks.RetryLimit > 0 && attempt >= e.hooks.RetryLimit {
		return false, 0
	}
	call, _ := tools.CurrentToolCallFromContext(ctx)
	var errMsg string
	if execErr != nil {
		errMsg = execErr.Error()
	} else if res != nil {
		errMsg = res.Error
	}
	payload := map[string]any{
		"phase":   "onToolError",
		"attempt": attempt,
		"call": map[string]any{
			"id":   call.ID,
			"name": call.Name,
			"args": decodeToolCallArgs(call),
		},
		"error":           errMsg,
		"defaultRetry":    defaultRetry,
		"defaultBackoffMs": func() int64 {
			return defaultBackoff.Milliseconds()
		}(),
		"timestampMs": time.Now().UnixMilli(),
	}
	if res != nil {
		payload["result"] = map[string]any{
			"value": cloneJSONValue(res.Result),
			"error": res.Error,
		}
	}
	ret, err := e.hooks.OnError(goja.Undefined(), e.api.toJSValue(payload))
	if hookErr := e.hookError("onToolError", err); hookErr != nil {
		return false, 0
	}
	if err != nil || ret == nil || goja.IsUndefined(ret) || goja.IsNull(ret) {
		return defaultRetry, defaultBackoff
	}
	resp := decodeMap(ret.Export())
	if resp == nil {
		return defaultRetry, defaultBackoff
	}
	if action := strings.ToLower(strings.TrimSpace(toString(resp["action"], ""))); action == "abort" || action == "continue" {
		return false, 0
	} else if action == "retry" {
		backoffMS := toInt(resp["backoffMs"], int(defaultBackoff.Milliseconds()))
		if backoffMS < 0 {
			backoffMS = 0
		}
		return true, time.Duration(backoffMS) * time.Millisecond
	}
	if abort, ok := resp["abort"].(bool); ok && abort {
		return false, 0
	}
	if retry, ok := resp["retry"].(bool); ok {
		backoffMS := toInt(resp["backoffMs"], int(defaultBackoff.Milliseconds()))
		if backoffMS < 0 {
			backoffMS = 0
		}
		if retry {
			return true, time.Duration(backoffMS) * time.Millisecond
		}
		return false, 0
	}
	return defaultRetry, defaultBackoff
}

func (m *moduleRuntime) requireEngineRef(v goja.Value) (*engineRef, error) {
	ref := m.getRef(v)
	switch x := ref.(type) {
	case *engineRef:
		return x, nil
	case engine.Engine:
		return &engineRef{Name: "engine", Engine: x}, nil
	default:
		return nil, fmt.Errorf("expected engine reference")
	}
}

func (m *moduleRuntime) requireToolRegistry(v goja.Value) (tools.ToolRegistry, error) {
	ref := m.getRef(v)
	switch x := ref.(type) {
	case *toolRegistryRef:
		return x.registry, nil
	case tools.ToolRegistry:
		return x, nil
	default:
		return nil, fmt.Errorf("expected tool registry reference")
	}
}

func inferAPIType(model string) aitypes.ApiType {
	m := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.Contains(m, "gemini"):
		return aitypes.ApiTypeGemini
	case strings.Contains(m, "claude"):
		return aitypes.ApiTypeClaude
	case strings.HasPrefix(m, "o1"), strings.HasPrefix(m, "o3"), strings.HasPrefix(m, "o4"), strings.HasPrefix(m, "gpt-5"):
		return aitypes.ApiTypeOpenAIResponses
	default:
		return aitypes.ApiTypeOpenAI
	}
}

func inferAPIKeyFromEnv(apiType aitypes.ApiType) string {
	switch apiType {
	case aitypes.ApiTypeGemini:
		if v := strings.TrimSpace(os.Getenv("GEMINI_API_KEY")); v != "" {
			return v
		}
		if v := strings.TrimSpace(os.Getenv("GOOGLE_API_KEY")); v != "" {
			return v
		}
	case aitypes.ApiTypeClaude:
		return strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	default:
		return strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	}
	return ""
}

func profileFromPrecedence(explicitProfile string, opts map[string]any) string {
	if p := strings.TrimSpace(explicitProfile); p != "" {
		return p
	}
	if opts != nil {
		if p := strings.TrimSpace(toString(opts["profile"], "")); p != "" {
			return p
		}
	}
	if p := strings.TrimSpace(os.Getenv("PINOCCHIO_PROFILE")); p != "" {
		return p
	}
	return "4o-mini"
}

func (m *moduleRuntime) stepSettingsFromEngineOptions(explicitProfile string, opts map[string]any) (*aistepssettings.StepSettings, string, error) {
	ss, err := aistepssettings.NewStepSettings()
	if err != nil {
		return nil, "", err
	}

	resolvedProfile := profileFromPrecedence(explicitProfile, opts)
	model := resolvedProfile
	if opts != nil && strings.TrimSpace(explicitProfile) == "" {
		if override := strings.TrimSpace(toString(opts["model"], "")); override != "" {
			model = override
		}
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = "4o-mini"
	}

	apiTypeRaw := ""
	if opts != nil {
		apiTypeRaw = strings.TrimSpace(toString(opts["apiType"], ""))
		if apiTypeRaw == "" {
			apiTypeRaw = strings.TrimSpace(toString(opts["provider"], ""))
		}
	}
	apiType := inferAPIType(model)
	if apiTypeRaw != "" {
		apiType = aitypes.ApiType(strings.ToLower(apiTypeRaw))
	}

	ss.Chat.Engine = &model
	ss.Chat.ApiType = &apiType

	if opts != nil {
		if tRaw, ok := opts["temperature"]; ok {
			t := float64(toInt(tRaw, -1))
			switch v := tRaw.(type) {
			case float64:
				t = v
			case float32:
				t = float64(v)
			}
			if t >= 0 {
				ss.Chat.Temperature = &t
			}
		}
		if topPRaw, ok := opts["topP"]; ok {
			topP := float64(toInt(topPRaw, -1))
			switch v := topPRaw.(type) {
			case float64:
				topP = v
			case float32:
				topP = float64(v)
			}
			if topP >= 0 {
				ss.Chat.TopP = &topP
			}
		}
		if maxTok := toInt(opts["maxTokens"], -1); maxTok > 0 {
			ss.Chat.MaxResponseTokens = &maxTok
		}
		if timeoutSec := toInt(opts["timeoutSeconds"], 0); timeoutSec > 0 {
			d := time.Duration(timeoutSec) * time.Second
			ss.Client.Timeout = &d
			ss.Client.TimeoutSeconds = &timeoutSec
		}
		if timeoutMS := toInt(opts["timeoutMs"], 0); timeoutMS > 0 {
			d := time.Duration(timeoutMS) * time.Millisecond
			sec := int(d.Seconds())
			ss.Client.Timeout = &d
			ss.Client.TimeoutSeconds = &sec
		}
	}

	key := ""
	if opts != nil {
		key = strings.TrimSpace(toString(opts["apiKey"], ""))
	}
	if key == "" {
		key = inferAPIKeyFromEnv(apiType)
	}

	// Keep OpenAI key alias populated for responses engine and OpenAI-compatible providers.
	switch apiType {
	case aitypes.ApiTypeOpenAIResponses:
		if key != "" {
			ss.API.APIKeys["openai-api-key"] = key
			ss.API.APIKeys["openai-responses-api-key"] = key
		}
	case aitypes.ApiTypeOpenAI, aitypes.ApiTypeAnyScale, aitypes.ApiTypeFireworks:
		if key != "" {
			ss.API.APIKeys[string(apiType)+"-api-key"] = key
			ss.API.APIKeys["openai-api-key"] = key
		}
	case aitypes.ApiTypeGemini, aitypes.ApiTypeClaude:
		if key != "" {
			ss.API.APIKeys[string(apiType)+"-api-key"] = key
		}
	}

	if opts != nil {
		if baseURL := strings.TrimSpace(toString(opts["baseURL"], "")); baseURL != "" {
			ss.API.BaseUrls[string(apiType)+"-base-url"] = baseURL
			if apiType == aitypes.ApiTypeOpenAIResponses {
				ss.API.BaseUrls["openai-base-url"] = baseURL
			}
		}
	}
	if apiType == aitypes.ApiTypeClaude {
		if _, ok := ss.API.BaseUrls["claude-base-url"]; !ok {
			ss.API.BaseUrls["claude-base-url"] = "https://api.anthropic.com"
		}
	}

	return ss, resolvedProfile, nil
}

func (m *moduleRuntime) engineFromStepSettings(explicitProfile string, opts map[string]any, fromProfile bool) (*engineRef, error) {
	ss, resolvedProfile, err := m.stepSettingsFromEngineOptions(explicitProfile, opts)
	if err != nil {
		return nil, err
	}
	eng, err := enginefactory.NewEngineFromStepSettings(ss)
	if err != nil {
		return nil, err
	}
	name := "config"
	if fromProfile {
		name = "profile:" + resolvedProfile
	}
	return &engineRef{Name: name, Engine: eng}, nil
}

func (m *moduleRuntime) newEngineObject(ref *engineRef) goja.Value {
	o := m.vm.NewObject()
	m.attachRef(o, ref)
	m.mustSet(o, "name", ref.Name)
	return o
}

func (m *moduleRuntime) engineEcho(call goja.FunctionCall) goja.Value {
	reply := "READY"
	if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
		opts := decodeMap(call.Arguments[0].Export())
		if opts != nil {
			reply = toString(opts["reply"], reply)
		}
	}
	ref := &engineRef{
		Name:   "echo",
		Engine: &echoEngine{reply: reply},
	}
	return m.newEngineObject(ref)
}

func (m *moduleRuntime) engineFromProfile(call goja.FunctionCall) goja.Value {
	profile := ""
	if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
		profile = call.Arguments[0].String()
	}
	var opts map[string]any
	if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) && !goja.IsNull(call.Arguments[1]) {
		opts = decodeMap(call.Arguments[1].Export())
	}
	ref, err := m.engineFromStepSettings(profile, opts, true)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.newEngineObject(ref)
}

func (m *moduleRuntime) engineFromConfig(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
		panic(m.vm.NewTypeError("fromConfig requires options object"))
	}
	opts := decodeMap(call.Arguments[0].Export())
	if opts == nil {
		panic(m.vm.NewTypeError("fromConfig requires options object"))
	}
	ref, err := m.engineFromStepSettings("", opts, false)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.newEngineObject(ref)
}

func (m *moduleRuntime) engineFromFunction(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(m.vm.NewTypeError("fromFunction requires a JS function"))
	}
	fn, ok := goja.AssertFunction(call.Arguments[0])
	if !ok {
		panic(m.vm.NewTypeError("fromFunction expects callable argument"))
	}
	ref := &engineRef{
		Name: "jsFunction",
		Engine: &jsCallableEngine{
			api: m,
			fn:  fn,
		},
	}
	return m.newEngineObject(ref)
}

func (m *moduleRuntime) middlewareFromJS(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(m.vm.NewTypeError("fromJS requires callback argument"))
	}
	fn, ok := goja.AssertFunction(call.Arguments[0])
	if !ok {
		panic(m.vm.NewTypeError("fromJS expects callable callback"))
	}
	name := "js-middleware"
	if len(call.Arguments) > 1 {
		name = call.Arguments[1].String()
	}
	ref := &jsMiddlewareRef{Name: name, Fn: fn}
	o := m.vm.NewObject()
	m.attachRef(o, ref)
	m.mustSet(o, "type", "js")
	m.mustSet(o, "name", name)
	return o
}

func (m *moduleRuntime) middlewareFromGo(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(m.vm.NewTypeError("go middleware requires name argument"))
	}
	name := call.Arguments[0].String()
	var options map[string]any
	if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) && !goja.IsNull(call.Arguments[1]) {
		options = decodeMap(call.Arguments[1].Export())
	}
	ref := &goMiddlewareRef{Name: name, Options: options}
	o := m.vm.NewObject()
	m.attachRef(o, ref)
	m.mustSet(o, "type", "go")
	m.mustSet(o, "name", name)
	if options != nil {
		m.mustSet(o, "options", options)
	}
	return o
}

func (m *moduleRuntime) resolveMiddleware(v goja.Value) (middleware.Middleware, error) {
	if fn, ok := goja.AssertFunction(v); ok {
		return m.jsMiddleware("js-middleware", fn), nil
	}

	ref := m.getRef(v)
	switch x := ref.(type) {
	case *jsMiddlewareRef:
		return m.jsMiddleware(x.Name, x.Fn), nil
	case *goMiddlewareRef:
		return m.resolveGoMiddleware(x.Name, x.Options)
	}
	return nil, fmt.Errorf("unsupported middleware specification")
}

func (m *moduleRuntime) resolveGoMiddleware(name string, options map[string]any) (middleware.Middleware, error) {
	factory := m.goMiddlewareFactories[name]
	if factory == nil {
		return nil, fmt.Errorf("unknown go middleware: %s", name)
	}
	return factory(options)
}

func (m *moduleRuntime) jsMiddleware(name string, fn goja.Callable) middleware.Middleware {
	return func(next middleware.HandlerFunc) middleware.HandlerFunc {
		return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
			jsTurn, err := m.encodeTurnValue(t)
			if err != nil {
				return nil, err
			}

			nextFn := func(call goja.FunctionCall) goja.Value {
				inTurn := t
				if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
					decoded, err := m.decodeTurnValue(call.Arguments[0])
					if err != nil {
						panic(m.vm.NewGoError(err))
					}
					inTurn = decoded
				}
				out, err := next(ctx, inTurn)
				if err != nil {
					panic(m.vm.NewGoError(err))
				}
				v, err := m.encodeTurnValue(out)
				if err != nil {
					panic(m.vm.NewGoError(err))
				}
				return v
			}

			ret, err := fn(goja.Undefined(), jsTurn, m.vm.ToValue(nextFn))
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			if ret == nil || goja.IsUndefined(ret) || goja.IsNull(ret) {
				return t, nil
			}
			decoded, err := m.decodeTurnValue(ret)
			if err != nil {
				return nil, err
			}
			return decoded, nil
		}
	}
}

func (m *moduleRuntime) toolsCreateRegistry(call goja.FunctionCall) goja.Value {
	ref := &toolRegistryRef{
		api:        m,
		registry:   tools.NewInMemoryToolRegistry(),
		goRegistry: m.goToolRegistry,
	}
	o := m.vm.NewObject()
	m.attachRef(o, ref)
	m.mustSet(o, "register", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(m.vm.NewTypeError("register requires tool spec"))
		}
		if err := ref.register(call.Arguments[0], nil); err != nil {
			panic(m.vm.NewGoError(err))
		}
		return o
	})
	m.mustSet(o, "useGoTools", func(call goja.FunctionCall) goja.Value {
		var names []string
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
			for _, n := range decodeSlice(call.Arguments[0].Export()) {
				if s, ok := n.(string); ok && s != "" {
					names = append(names, s)
				}
			}
		}
		if err := ref.useGoTools(names); err != nil {
			panic(m.vm.NewGoError(err))
		}
		return o
	})
	m.mustSet(o, "list", func(goja.FunctionCall) goja.Value {
		list := ref.registry.ListTools()
		out := make([]any, 0, len(list))
		for _, t := range list {
			out = append(out, map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"version":     t.Version,
				"tags":        t.Tags,
			})
		}
		return m.toJSValue(out)
	})
	m.mustSet(o, "call", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(m.vm.NewTypeError("call requires tool name"))
		}
		name := call.Arguments[0].String()
		args := map[string]any{}
		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) && !goja.IsNull(call.Arguments[1]) {
			if parsed := decodeMap(call.Arguments[1].Export()); parsed != nil {
				args = parsed
			}
		}
		b, err := json.Marshal(args)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		exec := tools.NewDefaultToolExecutor(tools.DefaultToolConfig())
		res, err := exec.ExecuteToolCall(context.Background(), tools.ToolCall{
			ID:        "js-call",
			Name:      name,
			Arguments: b,
		}, ref.registry)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		if res == nil {
			panic(m.vm.NewGoError(fmt.Errorf("tool call returned nil result")))
		}
		if res.Error != "" {
			panic(m.vm.NewGoError(fmt.Errorf("%s", res.Error)))
		}
		return m.vm.ToValue(res.Result)
	})
	return o
}

func (r *toolRegistryRef) register(v goja.Value, _ map[string]any) error {
	obj := v.ToObject(r.api.vm)
	name := obj.Get("name").String()
	if name == "" {
		return fmt.Errorf("tool name is required")
	}
	description := ""
	if d := obj.Get("description"); d != nil && !goja.IsUndefined(d) && !goja.IsNull(d) {
		description = d.String()
	}
	handlerValue := obj.Get("handler")
	handler, ok := goja.AssertFunction(handlerValue)
	if !ok {
		return fmt.Errorf("tool %s handler must be a function", name)
	}

	fn := func(_ context.Context, in map[string]any) (any, error) {
		ret, err := handler(goja.Undefined(), r.api.vm.ToValue(in))
		if err != nil {
			return nil, fmt.Errorf("js tool %s: %w", name, err)
		}
		return cloneJSONValue(ret.Export()), nil
	}
	def, err := tools.NewToolFromFunc(name, description, fn)
	if err != nil {
		return err
	}

	parameters := obj.Get("parameters")
	if parameters != nil && !goja.IsUndefined(parameters) && !goja.IsNull(parameters) {
		b, err := json.Marshal(parameters.Export())
		if err != nil {
			return fmt.Errorf("marshal tool parameters: %w", err)
		}
		var schema jsonschema.Schema
		if err := json.Unmarshal(b, &schema); err != nil {
			return fmt.Errorf("decode tool parameters: %w", err)
		}
		def.Parameters = &schema
	}

	if err := r.registry.RegisterTool(name, *def); err != nil {
		return err
	}
	return nil
}

func (r *toolRegistryRef) useGoTools(names []string) error {
	if r.goRegistry == nil {
		return fmt.Errorf("no go tool registry configured")
	}
	if len(names) == 0 {
		for _, t := range r.goRegistry.ListTools() {
			if err := r.registry.RegisterTool(t.Name, t); err != nil {
				return err
			}
		}
		return nil
	}
	for _, name := range names {
		def, err := r.goRegistry.GetTool(name)
		if err != nil {
			return err
		}
		if err := r.registry.RegisterTool(name, *def); err != nil {
			return err
		}
	}
	return nil
}

func defaultGoMiddlewareFactories(logger zerolog.Logger) map[string]MiddlewareFactory {
	return map[string]MiddlewareFactory{
		"systemPrompt": func(options map[string]any) (middleware.Middleware, error) {
			prompt := ""
			if options != nil {
				prompt = toString(options["prompt"], "")
			}
			return middleware.NewSystemPromptMiddleware(prompt), nil
		},
		"reorderToolResults": func(map[string]any) (middleware.Middleware, error) {
			return middleware.NewToolResultReorderMiddleware(), nil
		},
		"turnLogging": func(map[string]any) (middleware.Middleware, error) {
			return middleware.NewTurnLoggingMiddleware(logger), nil
		},
	}
}

func (m *moduleRuntime) turnsNormalize(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(m.vm.NewTypeError("turns.normalize requires turn"))
	}
	t, err := m.decodeTurnValue(call.Arguments[0])
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	normalized := t.Clone()
	if normalized == nil {
		normalized = &turns.Turn{}
	}
	v, err := m.encodeTurnValue(normalized)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return v
}

func (m *moduleRuntime) turnsNewTurn(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) == 0 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
		v, _ := m.encodeTurnValue(&turns.Turn{})
		return v
	}
	t, err := m.decodeTurnValue(call.Arguments[0])
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	v, err := m.encodeTurnValue(t)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return v
}

func (m *moduleRuntime) turnsAppendBlock(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(m.vm.NewTypeError("turns.appendBlock requires turn and block"))
	}
	t, err := m.decodeTurnValue(call.Arguments[0])
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	b, err := m.decodeBlock(call.Arguments[1].Export())
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	turns.AppendBlock(t, b)
	v, err := m.encodeTurnValue(t)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return v
}

func (m *moduleRuntime) turnsNewUserBlock(call goja.FunctionCall) goja.Value {
	text := ""
	if len(call.Arguments) > 0 {
		text = call.Arguments[0].String()
	}
	return m.toJSValue(m.encodeBlock(turns.NewUserTextBlock(text)))
}

func (m *moduleRuntime) turnsNewSystemBlock(call goja.FunctionCall) goja.Value {
	text := ""
	if len(call.Arguments) > 0 {
		text = call.Arguments[0].String()
	}
	return m.toJSValue(m.encodeBlock(turns.NewSystemTextBlock(text)))
}

func (m *moduleRuntime) turnsNewAssistantBlock(call goja.FunctionCall) goja.Value {
	text := ""
	if len(call.Arguments) > 0 {
		text = call.Arguments[0].String()
	}
	return m.toJSValue(m.encodeBlock(turns.NewAssistantTextBlock(text)))
}

func (m *moduleRuntime) turnsNewToolCallBlock(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 3 {
		panic(m.vm.NewTypeError("turns.newToolCallBlock requires id, name, args"))
	}
	id := call.Arguments[0].String()
	name := call.Arguments[1].String()
	args := call.Arguments[2].Export()
	return m.toJSValue(m.encodeBlock(turns.NewToolCallBlock(id, name, args)))
}

func (m *moduleRuntime) turnsNewToolUseBlock(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(m.vm.NewTypeError("turns.newToolUseBlock requires id, result[, error]"))
	}
	id := call.Arguments[0].String()
	result := call.Arguments[1].Export()
	errText := ""
	if len(call.Arguments) > 2 {
		errText = call.Arguments[2].String()
	}
	return m.toJSValue(m.encodeBlock(turns.NewToolUseBlockWithError(id, result, errText)))
}
