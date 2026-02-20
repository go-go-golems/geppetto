package geppetto

import (
	"context"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/rs/zerolog"
)

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
			ctxPayload := map[string]any{
				"middlewareName": name,
				"timestampMs":    time.Now().UnixMilli(),
			}
			if t != nil {
				if t.ID != "" {
					ctxPayload["turnId"] = t.ID
				}
				if sessionID, ok, err := turns.KeyTurnMetaSessionID.Get(t.Metadata); err == nil && ok && sessionID != "" {
					ctxPayload["sessionId"] = sessionID
				}
				if inferenceID, ok, err := turns.KeyTurnMetaInferenceID.Get(t.Metadata); err == nil && ok && inferenceID != "" {
					ctxPayload["inferenceId"] = inferenceID
				}
				if traceID, ok, err := turns.KeyTurnMetaTraceID.Get(t.Metadata); err == nil && ok && traceID != "" {
					ctxPayload["traceId"] = traceID
				}
			}
			if _, ok := ctxPayload["sessionId"]; !ok {
				if sessionID := session.SessionIDFromContext(ctx); sessionID != "" {
					ctxPayload["sessionId"] = sessionID
				}
			}
			if _, ok := ctxPayload["inferenceId"]; !ok {
				if inferenceID := session.InferenceIDFromContext(ctx); inferenceID != "" {
					ctxPayload["inferenceId"] = inferenceID
				}
			}
			if tags := session.RunTagsFromContext(ctx); len(tags) > 0 {
				ctxPayload["tags"] = cloneJSONMap(tags)
			}
			if deadline, ok := ctx.Deadline(); ok {
				ctxPayload["deadlineMs"] = deadline.UnixMilli()
			}

			retAny, err := m.callOnOwner(ctx, "middleware.fromJS", func(ownerCtx context.Context) (any, error) {
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
					out, err := next(ownerCtx, inTurn)
					if err != nil {
						panic(m.vm.NewGoError(err))
					}
					v, err := m.encodeTurnValue(out)
					if err != nil {
						panic(m.vm.NewGoError(err))
					}
					return v
				}
				ret, err := fn(goja.Undefined(), jsTurn, m.vm.ToValue(nextFn), m.toJSValue(ctxPayload))
				if err != nil {
					return nil, err
				}
				if ret == nil || goja.IsUndefined(ret) || goja.IsNull(ret) {
					return nil, nil
				}
				return m.decodeTurnValue(ret)
			})
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			if retAny == nil {
				return t, nil
			}
			decoded, ok := retAny.(*turns.Turn)
			if !ok {
				return nil, fmt.Errorf("%s: expected middleware return *turns.Turn, got %T", name, retAny)
			}
			return decoded, nil
		}
	}
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
