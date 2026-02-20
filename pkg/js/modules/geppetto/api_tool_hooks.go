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
)

func newJSToolHookExecutor(api *moduleRuntime, cfg tools.ToolConfig, hooks *jsToolHooks) tools.ToolExecutor {
	base := tools.NewBaseToolExecutor(cfg)
	exec := &jsToolHookExecutor{
		BaseToolExecutor: base,
		api:              api,
		hooks:            hooks,
	}
	exec.ToolExecutorExt = exec
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

func cloneJSONMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	cloned := cloneJSONValue(in)
	if out, ok := cloned.(map[string]any); ok {
		return out
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func addSessionMetaFromContext(ctx context.Context, payload map[string]any) {
	if payload == nil {
		return
	}
	if sessionID := session.SessionIDFromContext(ctx); sessionID != "" {
		payload["sessionId"] = sessionID
	}
	if inferenceID := session.InferenceIDFromContext(ctx); inferenceID != "" {
		payload["inferenceId"] = inferenceID
	}
	if tags := session.RunTagsFromContext(ctx); len(tags) > 0 {
		payload["tags"] = cloneJSONMap(tags)
	}
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
	addSessionMetaFromContext(ctx, payload)
	retAny, err := e.api.callOnOwner(ctx, "toolHooks.beforeToolCall", func(context.Context) (any, error) {
		ret, invokeErr := e.hooks.Before(goja.Undefined(), e.api.toJSValue(payload))
		if invokeErr != nil {
			return nil, invokeErr
		}
		if ret == nil || goja.IsUndefined(ret) || goja.IsNull(ret) {
			return nil, nil
		}
		return decodeMap(ret.Export()), nil
	})
	if hookErr := e.hookError("beforeToolCall", err); hookErr != nil {
		return call, hookErr
	}
	if retAny == nil {
		return call, nil
	}

	resp, ok := retAny.(map[string]any)
	if !ok || resp == nil {
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
	addSessionMetaFromContext(ctx, payload)
	retAny, err := e.api.callOnOwner(ctx, "toolHooks.afterToolCall", func(context.Context) (any, error) {
		ret, invokeErr := e.hooks.After(goja.Undefined(), e.api.toJSValue(payload))
		if invokeErr != nil {
			return nil, invokeErr
		}
		if ret == nil || goja.IsUndefined(ret) || goja.IsNull(ret) {
			return nil, nil
		}
		return decodeMap(ret.Export()), nil
	})
	if hookErr := e.hookError("afterToolCall", err); hookErr != nil {
		res.Error = hookErr.Error()
		e.BaseToolExecutor.PublishResult(ctx, call, res)
		return
	}
	if resp, ok := retAny.(map[string]any); ok && resp != nil {
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
		"error":        errMsg,
		"defaultRetry": defaultRetry,
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
	addSessionMetaFromContext(ctx, payload)
	retAny, err := e.api.callOnOwner(ctx, "toolHooks.onToolError", func(context.Context) (any, error) {
		ret, invokeErr := e.hooks.OnError(goja.Undefined(), e.api.toJSValue(payload))
		if invokeErr != nil {
			return nil, invokeErr
		}
		if ret == nil || goja.IsUndefined(ret) || goja.IsNull(ret) {
			return nil, nil
		}
		return decodeMap(ret.Export()), nil
	})
	if hookErr := e.hookError("onToolError", err); hookErr != nil {
		return false, 0
	}
	if retAny == nil {
		return defaultRetry, defaultBackoff
	}
	resp, ok := retAny.(map[string]any)
	if !ok || resp == nil {
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
