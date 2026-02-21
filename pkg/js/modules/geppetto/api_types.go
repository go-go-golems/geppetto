package geppetto

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop/enginebuilder"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

type engineRef struct {
	Name   string
	Engine engine.Engine
}

type builderRef struct {
	api *moduleRuntime

	base        engine.Engine
	middlewares []middleware.Middleware

	registry     tools.ToolRegistry
	loopCfg      *toolloop.LoopConfig
	toolCfg      *tools.ToolConfig
	toolExecutor tools.ToolExecutor
	toolHooks    *jsToolHooks
	eventSinks   []events.EventSink
	snapshotHook toolloop.SnapshotHook
	persister    enginebuilder.TurnPersister
}

type sessionRef struct {
	api     *moduleRuntime
	session *session.Session
}

type runOptions struct {
	timeoutMs int
	tags      map[string]any
}

type jsEventCollector struct {
	api *moduleRuntime

	mu        sync.RWMutex
	listeners map[string][]goja.Callable
	closed    bool
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

type jsToolHooks struct {
	Before     goja.Callable
	After      goja.Callable
	OnError    goja.Callable
	FailOpen   bool
	RetryLimit int
}

type jsToolHookExecutor struct {
	*tools.BaseToolExecutor
	api   *moduleRuntime
	hooks *jsToolHooks
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

func (e *jsCallableEngine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	ret, err := e.api.callOnOwner(ctx, "engine.fromFunction.runInference", func(context.Context) (any, error) {
		arg, err := e.api.encodeTurnValue(t)
		if err != nil {
			return nil, err
		}
		v, err := e.fn(goja.Undefined(), arg)
		if err != nil {
			return nil, fmt.Errorf("js engine callback: %w", err)
		}
		if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
			return nil, nil
		}
		decoded, err := e.api.decodeTurnValue(v)
		if err != nil {
			return nil, err
		}
		return decoded, nil
	})
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return t, nil
	}
	out, ok := ret.(*turns.Turn)
	if !ok {
		return nil, fmt.Errorf("js engine callback returned unexpected type %T", ret)
	}
	return out, nil
}
