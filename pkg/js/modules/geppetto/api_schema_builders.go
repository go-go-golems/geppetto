package geppetto

import (
	"fmt"
	"strings"

	"github.com/dop251/goja"
)

type schemaRef struct {
	schema map[string]any
}

type schemaBuilderRef struct {
	api    *moduleRuntime
	schema map[string]any
}

func (m *moduleRuntime) installSchemaNamespace(exports *goja.Object) {
	ns := m.vm.NewObject()
	for _, typ := range []string{"string", "integer", "number", "boolean", "array", "object"} {
		typ := typ
		m.mustSet(ns, typ, func(goja.FunctionCall) goja.Value {
			s := map[string]any{"type": typ}
			if typ == "object" {
				s["properties"] = map[string]any{}
			}
			return m.newSchemaBuilderObject(&schemaBuilderRef{api: m, schema: s})
		})
	}
	m.mustSet(ns, "enum", func(call goja.FunctionCall) goja.Value {
		values := make([]any, 0, len(call.Arguments))
		for _, arg := range call.Arguments {
			values = append(values, cloneJSONValue(arg.Export()))
		}
		return m.newSchemaBuilderObject(&schemaBuilderRef{api: m, schema: map[string]any{"enum": values}})
	})
	m.mustSet(exports, "schema", ns)
}

func (m *moduleRuntime) newSchemaBuilderObject(ref *schemaBuilderRef) *goja.Object {
	if ref == nil {
		ref = &schemaBuilderRef{api: m, schema: map[string]any{}}
	}
	ref.api = m
	o := m.vm.NewObject()
	m.attachRef(o, ref.clone())
	m.mustSet(o, "description", func(call goja.FunctionCall) goja.Value {
		next := ref.clone()
		if len(call.Arguments) > 0 {
			next.schema["description"] = call.Arguments[0].String()
		}
		return m.newSchemaBuilderObject(next)
	})
	m.mustSet(o, "property", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(m.vm.NewTypeError("schema.object().property requires name and schema"))
		}
		name := strings.TrimSpace(call.Arguments[0].String())
		if name == "" {
			panic(m.vm.NewTypeError("schema property name must not be empty"))
		}
		child, err := m.requireSchemaMap(call.Arguments[1])
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		next := ref.clone()
		props, _ := next.schema["properties"].(map[string]any)
		if props == nil {
			props = map[string]any{}
			next.schema["properties"] = props
		}
		props[name] = child
		return m.newSchemaBuilderObject(next)
	})
	m.mustSet(o, "items", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(m.vm.NewTypeError("schema.array().items requires schema"))
		}
		child, err := m.requireSchemaMap(call.Arguments[0])
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		next := ref.clone()
		next.schema["items"] = child
		return m.newSchemaBuilderObject(next)
	})
	m.mustSet(o, "required", func(call goja.FunctionCall) goja.Value {
		next := ref.clone()
		req := []any{}
		for _, arg := range call.Arguments {
			if arr := decodeSlice(arg.Export()); arr != nil {
				for _, item := range arr {
					if s := strings.TrimSpace(toString(item, "")); s != "" {
						req = append(req, s)
					}
				}
				continue
			}
			if s := strings.TrimSpace(arg.String()); s != "" {
				req = append(req, s)
			}
		}
		next.schema["required"] = req
		return m.newSchemaBuilderObject(next)
	})
	m.mustSet(o, "build", func(goja.FunctionCall) goja.Value {
		return m.newSchemaObject(&schemaRef{schema: cloneJSONMap(ref.schema)})
	})
	m.mustSet(o, "toJSON", func(goja.FunctionCall) goja.Value {
		return m.toJSValue(cloneJSONMap(ref.schema))
	})
	return o
}

func (r *schemaBuilderRef) clone() *schemaBuilderRef {
	if r == nil {
		return &schemaBuilderRef{schema: map[string]any{}}
	}
	return &schemaBuilderRef{api: r.api, schema: cloneJSONMap(r.schema)}
}

func (m *moduleRuntime) newSchemaObject(ref *schemaRef) *goja.Object {
	o := m.vm.NewObject()
	if ref == nil {
		ref = &schemaRef{schema: map[string]any{}}
	}
	m.attachRef(o, &schemaRef{schema: cloneJSONMap(ref.schema)})
	m.mustSet(o, "toJSON", func(goja.FunctionCall) goja.Value { return m.toJSValue(cloneJSONMap(ref.schema)) })
	m.mustSet(o, "clone", func(goja.FunctionCall) goja.Value {
		return m.newSchemaObject(&schemaRef{schema: cloneJSONMap(ref.schema)})
	})
	return o
}

func (m *moduleRuntime) requireSchemaMap(v goja.Value) (map[string]any, error) {
	ref := m.getRef(v)
	switch x := ref.(type) {
	case *schemaRef:
		return cloneJSONMap(x.schema), nil
	case *schemaBuilderRef:
		return cloneJSONMap(x.schema), nil
	}
	if raw := decodeMap(v.Export()); raw != nil {
		return cloneJSONMap(raw), nil
	}
	return nil, fmt.Errorf("expected schema wrapper")
}
