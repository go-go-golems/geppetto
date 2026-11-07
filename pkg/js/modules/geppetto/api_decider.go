package geppetto

import (
	"context"
	"fmt"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/decision"
	decisionfactory "github.com/go-go-golems/geppetto/pkg/decision/factory"
)

// deciderRef holds a profile-resolved decision provider behind a hidden Go
// reference; JavaScript only sees the wrapper's methods.
type deciderRef struct {
	api      *moduleRuntime
	provider decision.Provider
}

// deciderBuilder is gp.decider(settings). It accepts only a registry-resolved
// InferenceSettings wrapper, so JavaScript cannot supply an endpoint or key.
func (m *moduleRuntime) deciderBuilder(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
		panic(m.vm.NewTypeError("decider(settings) requires a registry-resolved InferenceSettings wrapper"))
	}
	settingsRef, err := m.requireInferenceSettingsRef(call.Arguments[0])
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	if settingsRef.settings == nil {
		panic(m.vm.NewGoError(fmt.Errorf("decider(settings) requires non-empty inference settings")))
	}
	factory, err := decisionfactory.NewSettingsFactoryFromInferenceSettings(settingsRef.settings)
	if err != nil {
		panic(m.vm.NewGoError(fmt.Errorf("decider(settings): %w", err)))
	}
	provider, err := factory.NewProvider()
	if err != nil {
		panic(m.vm.NewGoError(fmt.Errorf("decider(settings): %w", err)))
	}
	return m.newDeciderObject(&deciderRef{api: m, provider: provider})
}

func (m *moduleRuntime) newDeciderObject(ref *deciderRef) *goja.Object {
	o := m.vm.NewObject()
	m.attachRef(o, ref)
	m.mustSet(o, "decide", func(call goja.FunctionCall) goja.Value {
		req, err := decodeDecisionRequest(m.vm, call.Argument(0))
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		resp, err := ref.provider.Decide(m.deciderContext(), req)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(decisionResponseToJS(resp))
	})
	m.mustSet(o, "decideAsync", func(call goja.FunctionCall) goja.Value {
		return m.deciderAsync(call, ref)
	})
	m.mustSet(o, "model", func() map[string]any {
		model := ref.provider.Model()
		return map[string]any{"provider": model.Provider, "name": model.Name}
	})
	return o
}

func (m *moduleRuntime) deciderContext() context.Context {
	if m != nil && m.runtimeLifetimeContext != nil {
		return m.runtimeLifetimeContext
	}
	return context.Background()
}

// deciderAsync returns {promise, cancel, close}; the provider call runs on a
// goroutine and the promise settles on the runtime owner thread.
func (m *moduleRuntime) deciderAsync(call goja.FunctionCall, ref *deciderRef) goja.Value {
	if _, err := m.requireBridge("decider.decideAsync"); err != nil {
		panic(m.vm.NewTypeError(err.Error()))
	}
	req, err := decodeDecisionRequest(m.vm, call.Argument(0))
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	promise, resolve, reject := m.vm.NewPromise()
	handle := m.vm.NewObject()
	ctx, cancel := context.WithCancel(m.deciderContext())
	m.mustSet(handle, "promise", promise)
	m.mustSet(handle, "cancel", func(goja.FunctionCall) goja.Value { cancel(); return goja.Undefined() })
	m.mustSet(handle, "close", func(goja.FunctionCall) goja.Value { cancel(); return goja.Undefined() })
	provider := ref.provider
	go func() {
		defer cancel()
		resp, runErr := provider.Decide(ctx, req)
		postErr := m.postOnOwner(m.deciderContext(), "decider.decideAsync.settle", func(context.Context) {
			if runErr != nil {
				_ = reject(m.vm.NewGoError(runErr))
				return
			}
			_ = resolve(m.toJSValue(decisionResponseToJS(resp)))
		})
		if postErr != nil {
			m.logger.Error().Err(postErr).Msg("decider.decideAsync: failed to settle promise on owner thread")
		}
	}()
	return handle
}

// decodeDecisionRequest reads {state, questions, images?, model?} strictly.
// Objects are walked with Keys() so a choice's options keep JavaScript
// insertion order (Export would yield an unordered Go map).
func decodeDecisionRequest(vm *goja.Runtime, v goja.Value) (decision.Request, error) {
	bad := func(format string, args ...any) error {
		return fmt.Errorf("decide: %s: %w", fmt.Sprintf(format, args...), decision.ErrInvalidRequest)
	}
	obj, ok := asObject(vm, v)
	if !ok {
		return decision.Request{}, bad("request must be an object {state, questions, images?, model?}")
	}
	req := decision.Request{Questions: map[string]decision.Question{}}
	for _, k := range obj.Keys() {
		val := obj.Get(k)
		switch k {
		case "state":
			req.State = val.Export()
		case "model":
			s, ok := val.Export().(string)
			if !ok {
				return req, bad("model must be a string")
			}
			req.Model = s
		case "images":
			if !isArray(val) && !goja.IsUndefined(val) && !goja.IsNull(val) {
				return req, bad("images must be an array of data URL strings")
			}
			for _, x := range exportSlice(val) {
				s, ok := x.(string)
				if !ok {
					return req, bad("images must be data URL strings")
				}
				req.Images = append(req.Images, s)
			}
		case "questions":
			qs, ok := asObject(vm, val)
			if !ok {
				return req, bad("questions must be an object keyed by question id")
			}
			for _, id := range qs.Keys() {
				q, err := decodeQuestion(vm, qs.Get(id))
				if err != nil {
					return req, bad("question %q: %s", safeID(id), err)
				}
				req.Questions[id] = q
			}
		default:
			return req, bad("unknown key %q (expected state, questions, images, model)", safeID(k))
		}
	}
	return req, nil
}

func decodeQuestion(vm *goja.Runtime, v goja.Value) (decision.Question, error) {
	obj, ok := asObject(vm, v)
	if !ok {
		return decision.Question{}, fmt.Errorf("must be an object")
	}
	var q decision.Question
	for _, k := range obj.Keys() {
		val := obj.Get(k)
		switch k {
		case "type":
			s, _ := val.Export().(string)
			q.Type = decision.Type(s)
		case "instructions":
			q.Instructions = val.Export()
		case "true", "false":
			s, ok := val.Export().(string)
			if !ok {
				return q, fmt.Errorf("%s must be a string", k)
			}
			if k == "true" {
				q.True = s
			} else {
				q.False = s
			}
		case "levels":
			if !isArray(val) {
				return q, fmt.Errorf("levels must be an array of strings")
			}
			for _, x := range exportSlice(val) {
				s, ok := x.(string)
				if !ok {
					return q, fmt.Errorf("levels must be strings")
				}
				q.Levels = append(q.Levels, s)
			}
		case "options":
			opts, err := decodeOptions(vm, val)
			if err != nil {
				return q, err
			}
			q.Options = opts
		default:
			return q, fmt.Errorf("unknown key %q (expected type, instructions, options, levels, true, false)", safeID(k))
		}
	}
	return q, nil
}

// decodeOptions accepts {name: description, ...} (insertion order), an array
// of names, or an array of {name, description}.
func decodeOptions(vm *goja.Runtime, v goja.Value) ([]decision.Option, error) {
	obj, ok := asObject(vm, v)
	if !ok {
		return nil, fmt.Errorf("options must be an object or an array")
	}
	if obj.ClassName() != "Array" {
		out := []decision.Option{}
		for _, name := range obj.Keys() {
			d, ok := obj.Get(name).Export().(string)
			if !ok {
				return nil, fmt.Errorf("option descriptions must be strings")
			}
			out = append(out, decision.Option{Name: name, Description: d})
		}
		return out, nil
	}
	out := []decision.Option{}
	for _, x := range exportSlice(v) {
		switch o := x.(type) {
		case string:
			out = append(out, decision.Option{Name: o})
		case map[string]any:
			name, _ := o["name"].(string)
			desc, _ := o["description"].(string)
			for k := range o {
				if k != "name" && k != "description" {
					return nil, fmt.Errorf("option objects take only name and description")
				}
			}
			out = append(out, decision.Option{Name: name, Description: desc})
		default:
			return nil, fmt.Errorf("options array items must be strings or {name, description}")
		}
	}
	return out, nil
}

// isArray reports whether v is a JavaScript array.
func isArray(v goja.Value) bool {
	o, ok := v.(*goja.Object)
	return ok && o.ClassName() == "Array"
}

func asObject(vm *goja.Runtime, v goja.Value) (*goja.Object, bool) {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil, false
	}
	o, ok := v.(*goja.Object)
	return o, ok
}

func exportSlice(v goja.Value) []any {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil
	}
	s, _ := v.Export().([]any)
	return s
}

// safeID keeps caller keys short in error messages.
func safeID(s string) string {
	if len(s) > 64 {
		return s[:64] + "…"
	}
	return s
}

// decisionResponseToJS converts a response to a camelCase JS object; nil
// optional fields are omitted.
func decisionResponseToJS(resp decision.Response) map[string]any {
	answers := make(map[string]any, len(resp.Answers))
	for id, a := range resp.Answers {
		x := map[string]any{"type": string(a.Type)}
		switch a.Type {
		case decision.Noul:
			x["probability"] = a.Probability
		case decision.Choice:
			x["choice"] = a.Choice
			x["confidence"] = a.Confidence
			p := make(map[string]any, len(a.Probabilities))
			for k, v := range a.Probabilities {
				p[k] = v
			}
			x["probabilities"] = p
		case decision.Score:
			x["score"] = a.Score
			x["confidence"] = a.Confidence
			lp := make([]any, len(a.LevelProbabilities))
			for i, v := range a.LevelProbabilities {
				lp[i] = v
			}
			x["levelProbabilities"] = lp
		}
		answers[id] = x
	}
	out := map[string]any{"provider": resp.Provider, "model": resp.Model, "answers": answers}
	if resp.Usage != nil {
		out["usage"] = map[string]any{"inputTokens": resp.Usage.InputTokens, "outputTokens": resp.Usage.OutputTokens}
	}
	if resp.Cost != nil {
		out["cost"] = *resp.Cost
	}
	if resp.RequestID != "" {
		out["requestId"] = resp.RequestID
	}
	if resp.DurationMs != nil {
		out["durationMs"] = *resp.DurationMs
	}
	return out
}
