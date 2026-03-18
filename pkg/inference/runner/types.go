package runner

import (
	"context"
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	gepmiddleware "github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/middlewarecfg"
	"github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop/enginebuilder"
	geptools "github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

// ToolRegistrar registers zero or more tools into a target registry.
type ToolRegistrar func(ctx context.Context, reg geptools.ToolRegistry) error

// Runtime is fully resolved application-owned runtime input consumed by the runner.
type Runtime struct {
	InferenceSettings *settings.InferenceSettings
	SystemPrompt      string

	MiddlewareUses []middlewarecfg.Use
	Middlewares    []gepmiddleware.Middleware

	ToolNames      []string
	ToolRegistrars []ToolRegistrar

	RuntimeKey         string
	RuntimeFingerprint string
	ProfileVersion     uint64
}

// StartRequest describes one prepared or executed inference run.
type StartRequest struct {
	SessionID string
	Prompt    string
	SeedTurn  *turns.Turn
	Runtime   Runtime

	EventSinks   []events.EventSink
	SnapshotHook toolloop.SnapshotHook
	Persister    enginebuilder.TurnPersister
}

// PreparedRun holds the assembled state required to start or inspect one run.
type PreparedRun struct {
	Runtime Runtime

	Engine   engine.Engine
	Registry geptools.ToolRegistry
	Session  *session.Session
	Turn     *turns.Turn
}

// Runner assembles engines, middleware, tool registries, and sessions from
// resolved runtime input.
type Runner struct {
	middlewareDefinitions middlewarecfg.DefinitionRegistry
	middlewareBuildDeps   middlewarecfg.BuildDeps

	loopConfig toolloop.LoopConfig
	toolConfig geptools.ToolConfig

	engineFactory  func(*settings.InferenceSettings) (engine.Engine, error)
	toolExecutor   geptools.ToolExecutor
	toolRegistrars []ToolRegistrar

	eventSinks       []events.EventSink
	snapshotHook     toolloop.SnapshotHook
	persister        enginebuilder.TurnPersister
	stepController   *toolloop.StepController
	stepPauseTimeout time.Duration
}
