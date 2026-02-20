package geppetto

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/invopop/jsonschema"
)

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

	fn := func(goCtx context.Context, in map[string]any) (any, error) {
		toolCtx := map[string]any{
			"toolName":    name,
			"timestampMs": time.Now().UnixMilli(),
		}
		if sessionID := session.SessionIDFromContext(goCtx); sessionID != "" {
			toolCtx["sessionId"] = sessionID
		}
		if inferenceID := session.InferenceIDFromContext(goCtx); inferenceID != "" {
			toolCtx["inferenceId"] = inferenceID
		}
		if tags := session.RunTagsFromContext(goCtx); len(tags) > 0 {
			toolCtx["tags"] = cloneJSONMap(tags)
		}
		if call, ok := tools.CurrentToolCallFromContext(goCtx); ok {
			if call.ID != "" {
				toolCtx["callId"] = call.ID
			}
			if call.Name != "" {
				toolCtx["callName"] = call.Name
			}
		}
		if deadline, ok := goCtx.Deadline(); ok {
			toolCtx["deadlineMs"] = deadline.UnixMilli()
		}
		retAny, err := r.api.callOnOwner(goCtx, "tools.register.handler", func(context.Context) (any, error) {
			ret, invokeErr := handler(goja.Undefined(), r.api.vm.ToValue(in), r.api.toJSValue(toolCtx))
			if invokeErr != nil {
				return nil, invokeErr
			}
			if ret == nil || goja.IsUndefined(ret) || goja.IsNull(ret) {
				return nil, nil
			}
			return cloneJSONValue(ret.Export()), nil
		})
		if err != nil {
			return nil, fmt.Errorf("js tool %s: %w", name, err)
		}
		return retAny, nil
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
