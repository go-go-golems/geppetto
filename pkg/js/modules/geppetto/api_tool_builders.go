package geppetto

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/invopop/jsonschema"
)

type toolSpecRef struct {
	api         *moduleRuntime
	name        string
	description string
	parameters  map[string]any
	handler     goja.Callable
}

type toolBuilderRef struct {
	api         *moduleRuntime
	name        string
	description string
	parameters  map[string]any
	handler     goja.Callable
}

func (m *moduleRuntime) toolBuilder(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(m.vm.NewTypeError("tool(name) requires name"))
	}
	name := strings.TrimSpace(call.Arguments[0].String())
	if name == "" {
		panic(m.vm.NewTypeError("tool name must not be empty"))
	}
	return m.newToolBuilderObject(&toolBuilderRef{api: m, name: name})
}

func (m *moduleRuntime) newToolBuilderObject(ref *toolBuilderRef) *goja.Object {
	if ref == nil {
		ref = &toolBuilderRef{api: m}
	}
	ref.api = m
	o := m.vm.NewObject()
	m.attachRef(o, ref.clone())
	m.mustSet(o, "description", func(call goja.FunctionCall) goja.Value {
		next := ref.clone()
		if len(call.Arguments) > 0 {
			next.description = call.Arguments[0].String()
		}
		return m.newToolBuilderObject(next)
	})
	m.mustSet(o, "input", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(m.vm.NewTypeError("tool.input requires schema"))
		}
		schema, err := m.requireSchemaMap(call.Arguments[0])
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		next := ref.clone()
		next.parameters = schema
		return m.newToolBuilderObject(next)
	})
	m.mustSet(o, "handler", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(m.vm.NewTypeError("tool.handler requires function"))
		}
		fn, ok := goja.AssertFunction(call.Arguments[0])
		if !ok {
			panic(m.vm.NewTypeError("tool.handler argument must be callable"))
		}
		next := ref.clone()
		next.handler = fn
		return m.newToolBuilderObject(next)
	})
	m.mustSet(o, "build", func(goja.FunctionCall) goja.Value {
		if ref.name == "" {
			panic(m.vm.NewTypeError("tool name is required"))
		}
		if ref.handler == nil {
			panic(m.vm.NewTypeError("tool handler is required"))
		}
		return m.newToolSpecObject(&toolSpecRef{api: m, name: ref.name, description: ref.description, parameters: cloneJSONMap(ref.parameters), handler: ref.handler})
	})
	return o
}

func (r *toolBuilderRef) clone() *toolBuilderRef {
	if r == nil {
		return &toolBuilderRef{}
	}
	return &toolBuilderRef{api: r.api, name: r.name, description: r.description, parameters: cloneJSONMap(r.parameters), handler: r.handler}
}

func (m *moduleRuntime) newToolSpecObject(ref *toolSpecRef) *goja.Object {
	o := m.vm.NewObject()
	m.attachRef(o, ref)
	m.mustSet(o, "name", ref.name)
	m.mustSet(o, "description", ref.description)
	m.mustSet(o, "toJSON", func(goja.FunctionCall) goja.Value {
		return m.toJSValue(map[string]any{"name": ref.name, "description": ref.description, "parameters": cloneJSONMap(ref.parameters)})
	})
	return o
}

func (m *moduleRuntime) toolRegistryBuilder(call goja.FunctionCall) goja.Value {
	ref := &toolRegistryRef{api: m, registry: tools.NewInMemoryToolRegistry(), goRegistry: m.goToolRegistry}
	return m.newHardcutToolRegistryObject(ref)
}

func (m *moduleRuntime) newHardcutToolRegistryObject(ref *toolRegistryRef) *goja.Object {
	o := m.vm.NewObject()
	m.attachRef(o, ref)
	m.mustSet(o, "add", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(m.vm.NewTypeError("toolRegistry.add requires tool spec"))
		}
		if err := ref.addToolSpec(call.Arguments[0]); err != nil {
			panic(m.vm.NewGoError(err))
		}
		return o
	})
	m.mustSet(o, "addGo", func(call goja.FunctionCall) goja.Value {
		var names []string
		for _, arg := range call.Arguments {
			if s := strings.TrimSpace(arg.String()); s != "" {
				names = append(names, s)
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
			out = append(out, map[string]any{"name": t.Name, "description": t.Description, "version": t.Version, "tags": t.Tags})
		}
		return m.toJSValue(out)
	})
	m.mustSet(o, "call", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(m.vm.NewTypeError("toolRegistry.call requires tool name"))
		}
		name := call.Arguments[0].String()
		args := map[string]any{}
		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) && !goja.IsNull(call.Arguments[1]) {
			args = cloneJSONMap(decodeMap(call.Arguments[1].Export()))
		}
		b, err := json.Marshal(args)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		exec := tools.NewDefaultToolExecutor(tools.DefaultToolConfig())
		res, err := exec.ExecuteToolCall(context.Background(), tools.ToolCall{ID: "js-call", Name: name, Arguments: b}, ref.registry)
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

func (r *toolRegistryRef) addToolSpec(v goja.Value) error {
	ref := r.api.getRef(v)
	if spec, ok := ref.(*toolSpecRef); ok {
		return r.registerToolSpec(spec)
	}
	return r.register(v, nil)
}

func (r *toolRegistryRef) registerToolSpec(spec *toolSpecRef) error {
	if spec == nil || spec.name == "" {
		return fmt.Errorf("tool spec name is required")
	}
	if spec.handler == nil {
		return fmt.Errorf("tool %s handler must be a function", spec.name)
	}
	name := spec.name
	fn := func(goCtx context.Context, in map[string]any) (any, error) {
		toolCtx := map[string]any{"toolName": name, "timestampMs": time.Now().UnixMilli()}
		if sessionID := session.SessionIDFromContext(goCtx); sessionID != "" {
			toolCtx["sessionId"] = sessionID
		}
		if inferenceID := session.InferenceIDFromContext(goCtx); inferenceID != "" {
			toolCtx["inferenceId"] = inferenceID
		}
		if tags := session.RunTagsFromContext(goCtx); len(tags) > 0 {
			toolCtx["tags"] = cloneJSONMap(tags)
		}
		retAny, err := r.api.callOnOwner(goCtx, "tool.handler", func(context.Context) (any, error) {
			ret, invokeErr := spec.handler(goja.Undefined(), r.api.vm.ToValue(in), r.api.toJSValue(toolCtx))
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
	def, err := tools.NewToolFromFunc(name, spec.description, fn)
	if err != nil {
		return err
	}
	if len(spec.parameters) > 0 {
		b, err := json.Marshal(spec.parameters)
		if err != nil {
			return fmt.Errorf("marshal tool parameters: %w", err)
		}
		var schema jsonschema.Schema
		if err := json.Unmarshal(b, &schema); err != nil {
			return fmt.Errorf("decode tool parameters: %w", err)
		}
		def.Parameters = &schema
	}
	return r.registry.RegisterTool(name, *def)
}
