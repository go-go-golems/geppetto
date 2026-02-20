package geppetto

import (
	"fmt"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
)

func (m *moduleRuntime) applyBuilderOptions(b *builderRef, v goja.Value) error {
	obj := v.ToObject(m.vm)
	if obj == nil {
		return fmt.Errorf("options must be an object")
	}
	// Read ref-carrying properties (engine, middlewares, tools) directly from
	// the goja object to preserve non-enumerable __geppetto_ref. Using
	// v.Export() would serialize the object to a Go map, losing non-enumerable
	// properties.
	if engVal := obj.Get("engine"); engVal != nil && !goja.IsUndefined(engVal) && !goja.IsNull(engVal) {
		ref, err := m.requireEngineRef(engVal)
		if err != nil {
			return err
		}
		b.base = ref.Engine
	}
	if mwsVal := obj.Get("middlewares"); mwsVal != nil && !goja.IsUndefined(mwsVal) && !goja.IsNull(mwsVal) {
		mwsObj := mwsVal.ToObject(m.vm)
		if mwsObj != nil {
			lengthVal := mwsObj.Get("length")
			if lengthVal != nil && !goja.IsUndefined(lengthVal) {
				n := int(lengthVal.ToInteger())
				for i := 0; i < n; i++ {
					item := mwsObj.Get(fmt.Sprintf("%d", i))
					if item == nil || goja.IsUndefined(item) || goja.IsNull(item) {
						continue
					}
					mw, err := m.resolveMiddleware(item)
					if err != nil {
						return err
					}
					b.middlewares = append(b.middlewares, mw)
				}
			}
		}
	}
	if regVal := obj.Get("tools"); regVal != nil && !goja.IsUndefined(regVal) && !goja.IsNull(regVal) {
		reg, err := m.requireToolRegistry(regVal)
		if err != nil {
			return err
		}
		b.registry = reg
	}
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
	toolChoice := tools.ToolChoice(strings.TrimSpace(toString(cfg["toolChoice"], string(toolCfg.ToolChoice))))
	switch toolChoice {
	case tools.ToolChoiceAuto, tools.ToolChoiceNone, tools.ToolChoiceRequired:
		toolCfg.ToolChoice = toolChoice
	default:
		panic(m.vm.NewTypeError("invalid toolChoice %q, expected one of: auto, none, required", string(toolChoice)))
	}
	toolErrorHandling := tools.ToolErrorHandling(strings.TrimSpace(toString(cfg["toolErrorHandling"], string(toolCfg.ToolErrorHandling))))
	switch toolErrorHandling {
	case tools.ToolErrorContinue, tools.ToolErrorAbort, tools.ToolErrorRetry:
		toolCfg.ToolErrorHandling = toolErrorHandling
	default:
		panic(m.vm.NewTypeError("invalid toolErrorHandling %q, expected one of: continue, abort, retry", string(toolErrorHandling)))
	}
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
