package geppetto

import (
	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop/enginebuilder"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	aistepssettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
)

type engineRef struct {
	Name      string
	Engine    engine.Engine
	Metadata  map[string]any
	ModelInfo *aistepssettings.ModelInfo
}

type builderRef struct {
	api *moduleRuntime

	base        engine.Engine
	middlewares []middleware.Middleware

	registry         tools.ToolRegistry
	runtimeToolNames []string
	runtimeMetadata  map[string]any
	loopCfg          *toolloop.LoopConfig
	toolCfg          *tools.ToolConfig
	toolExecutor     tools.ToolExecutor
	toolHooks        *jsToolHooks
	eventSinks       []events.EventSink
	snapshotHook     toolloop.SnapshotHook
	persister        enginebuilder.TurnPersister
}

type sessionRef struct {
	api             *moduleRuntime
	session         *session.Session
	runtimeMetadata map[string]any
}

type runOptions struct {
	timeoutMs int
	tags      map[string]any
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
