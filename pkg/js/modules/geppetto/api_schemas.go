package geppetto

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/profiles"
)

func cloneNestedStringAnyMap(in map[string]map[string]any) map[string]map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]map[string]any, len(in))
	for key, value := range in {
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" {
			continue
		}
		clonedValue, ok := cloneJSONValue(value).(map[string]any)
		if !ok {
			continue
		}
		out[normalizedKey] = clonedValue
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (m *moduleRuntime) schemasListMiddlewares(call goja.FunctionCall) goja.Value {
	if m.middlewareSchemas == nil {
		panic(m.vm.NewGoError(fmt.Errorf("schemas.listMiddlewares requires a configured middleware definition registry")))
	}

	definitions := m.middlewareSchemas.ListDefinitions()
	rows := make([]any, 0, len(definitions))
	for _, def := range definitions {
		if def == nil {
			continue
		}
		row := map[string]any{
			"key":    def.Name(),
			"name":   def.Name(),
			"schema": cloneJSONValue(def.ConfigJSONSchema()),
		}
		rows = append(rows, row)
	}

	return m.toJSValue(rows)
}

func (m *moduleRuntime) schemasListExtensions(call goja.FunctionCall) goja.Value {
	if m.extensionCodecs == nil && len(m.extensionSchemas) == 0 {
		panic(m.vm.NewGoError(fmt.Errorf("schemas.listExtensions requires a configured extension schema provider")))
	}

	type extensionRow struct {
		Key         string
		DisplayName string
		Description string
		Schema      map[string]any
	}
	byKey := map[string]extensionRow{}

	if m.extensionCodecs != nil {
		lister, ok := m.extensionCodecs.(profiles.ExtensionCodecLister)
		if !ok {
			panic(m.vm.NewGoError(fmt.Errorf("configured extension codec registry does not support listing codecs")))
		}
		for _, codec := range lister.ListCodecs() {
			if codec == nil {
				continue
			}
			key := strings.TrimSpace(codec.Key().String())
			if key == "" {
				continue
			}
			row := extensionRow{Key: key}
			if withMeta, ok := codec.(profiles.ExtensionCodecMetadataProvider); ok {
				row.DisplayName = strings.TrimSpace(withMeta.ExtensionDisplayName())
				row.Description = strings.TrimSpace(withMeta.ExtensionDescription())
			}
			if withSchema, ok := codec.(profiles.ExtensionSchemaCodec); ok {
				if schema, ok := cloneJSONValue(withSchema.JSONSchema()).(map[string]any); ok {
					row.Schema = schema
				}
			}
			byKey[key] = row
		}
	}

	for key, schema := range m.extensionSchemas {
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" {
			continue
		}
		row := byKey[normalizedKey]
		row.Key = normalizedKey
		if clonedSchema, ok := cloneJSONValue(schema).(map[string]any); ok {
			row.Schema = clonedSchema
		}
		byKey[normalizedKey] = row
	}

	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	rows := make([]any, 0, len(keys))
	for _, key := range keys {
		row := byKey[key]
		payload := map[string]any{
			"key": key,
		}
		if row.DisplayName != "" {
			payload["displayName"] = row.DisplayName
		}
		if row.Description != "" {
			payload["description"] = row.Description
		}
		if row.Schema != nil {
			payload["schema"] = row.Schema
		}
		rows = append(rows, payload)
	}

	return m.toJSValue(rows)
}
