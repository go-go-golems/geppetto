package geppetto

import (
	"fmt"
	"strings"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	aistepssettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"gopkg.in/yaml.v3"
)

func encodeInferenceSettingsValue(in *aistepssettings.InferenceSettings) any {
	if in == nil {
		return nil
	}
	b, err := yaml.Marshal(in)
	if err != nil {
		return cloneJSONValue(in)
	}
	var out any
	if err := yaml.Unmarshal(b, &out); err != nil {
		return cloneJSONValue(in)
	}
	return out
}

func (m *moduleRuntime) newModelInfoObject(info *aistepssettings.ModelInfo) *goja.Object {
	out := m.vm.NewObject()
	if info == nil {
		return out
	}
	if info.ID != nil {
		m.mustSet(out, "id", *info.ID)
	}
	if info.Name != nil {
		m.mustSet(out, "name", *info.Name)
	}
	if info.Reasoning != nil {
		m.mustSet(out, "reasoning", *info.Reasoning)
	}
	if len(info.Input) > 0 {
		input := make([]string, 0, len(info.Input))
		for _, modality := range info.Input {
			input = append(input, string(modality))
		}
		m.mustSet(out, "input", input)
	}
	if info.ContextWindow != nil {
		m.mustSet(out, "contextWindow", *info.ContextWindow)
	}
	if info.QualityHighWatermark != nil {
		m.mustSet(out, "qualityHighWatermark", *info.QualityHighWatermark)
	}
	if info.MaxOutputTokens != nil {
		m.mustSet(out, "maxOutputTokens", *info.MaxOutputTokens)
	}
	if info.Cost != nil {
		cost := map[string]any{"input": info.Cost.Input, "output": info.Cost.Output, "cacheRead": info.Cost.CacheRead, "cacheWrite": info.Cost.CacheWrite}
		m.mustSet(out, "cost", cost)
	}
	if len(info.Metadata) > 0 {
		m.mustSet(out, "metadata", cloneJSONMap(info.Metadata))
	}
	return out
}

func mergeRuntimeMetadata(base map[string]any, extra map[string]any) map[string]any {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	out := cloneJSONMap(base)
	if out == nil {
		out = map[string]any{}
	}
	for k, v := range extra {
		out[k] = cloneJSONValue(v)
	}
	return out
}

func stampTurnRuntimeMetadata(t *turns.Turn, runtimeMetadata map[string]any) {
	if t == nil || len(runtimeMetadata) == 0 {
		return
	}
	attrib := map[string]any{}
	if existing, ok, err := turns.KeyTurnMetaRuntime.Get(t.Metadata); err == nil && ok {
		if m, ok := existing.(map[string]any); ok {
			attrib = mergeRuntimeMetadata(attrib, m)
		}
	}
	attrib = mergeRuntimeMetadata(attrib, runtimeMetadata)
	if len(attrib) == 0 {
		return
	}
	_ = turns.KeyTurnMetaRuntime.Set(&t.Metadata, attrib)
}

func materializeToolRegistry(base tools.ToolRegistry, toolNames []string) (tools.ToolRegistry, error) {
	if len(toolNames) == 0 {
		return base, nil
	}
	if base == nil {
		return nil, fmt.Errorf("resolved runtime requested tools but no tool registry is configured")
	}
	filtered := tools.NewInMemoryToolRegistry()
	for _, rawName := range toolNames {
		name := strings.TrimSpace(rawName)
		if name == "" {
			continue
		}
		def, err := base.GetTool(name)
		if err != nil {
			return nil, fmt.Errorf("resolved runtime tool %q not present in registry: %w", name, err)
		}
		if err := filtered.RegisterTool(name, *def); err != nil {
			return nil, err
		}
	}
	return filtered, nil
}
