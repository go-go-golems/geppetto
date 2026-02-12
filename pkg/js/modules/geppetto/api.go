package geppetto

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop/enginebuilder"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
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

	registry tools.ToolRegistry
	loopCfg  *toolloop.LoopConfig
	toolCfg  *tools.ToolConfig
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
			m.applyToolLoopSettings(b, decodeMap(call.Arguments[1].Export()))
		}
		return o
	})
	m.mustSet(o, "withToolLoop", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
			m.applyToolLoopSettings(b, decodeMap(call.Arguments[0].Export()))
		}
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
	if tlRaw, ok := opts["toolLoop"]; ok && tlRaw != nil {
		m.applyToolLoopSettings(b, decodeMap(tlRaw))
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

func (m *moduleRuntime) applyToolLoopSettings(b *builderRef, cfg map[string]any) {
	if cfg == nil {
		return
	}
	enabled := toBool(cfg["enabled"], true)
	if !enabled {
		b.registry = nil
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
	o := m.vm.NewObject()
	m.attachRef(o, ref)
	m.mustSet(o, "name", ref.Name)
	return o
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
	o := m.vm.NewObject()
	m.attachRef(o, ref)
	m.mustSet(o, "name", ref.Name)
	return o
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
