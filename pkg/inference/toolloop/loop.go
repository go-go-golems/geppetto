package toolloop

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/toolblocks"
	"github.com/go-go-golems/geppetto/pkg/inference/toolcontext"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type Loop struct {
	eng      engine.Engine
	registry tools.ToolRegistry
	loopCfg  LoopConfig
	toolCfg  tools.ToolConfig

	executor tools.ToolExecutor

	stepCtrl     *StepController
	pauseTimeout time.Duration

	snapshotHook SnapshotHook
}

type Option func(*Loop)

func New(opts ...Option) *Loop {
	l := &Loop{
		loopCfg:      DefaultLoopConfig(),
		toolCfg:      tools.DefaultToolConfig(),
		pauseTimeout: 30 * time.Second,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(l)
		}
	}
	return l
}

func WithEngine(eng engine.Engine) Option {
	return func(l *Loop) { l.eng = eng }
}

func WithRegistry(reg tools.ToolRegistry) Option {
	return func(l *Loop) { l.registry = reg }
}

func WithLoopConfig(cfg LoopConfig) Option {
	return func(l *Loop) { l.loopCfg = cfg }
}

func WithToolConfig(cfg tools.ToolConfig) Option {
	return func(l *Loop) { l.toolCfg = cfg }
}

func WithExecutor(exec tools.ToolExecutor) Option {
	return func(l *Loop) { l.executor = exec }
}

func WithStepController(sc *StepController) Option {
	return func(l *Loop) { l.stepCtrl = sc }
}

func WithPauseTimeout(d time.Duration) Option {
	return func(l *Loop) { l.pauseTimeout = d }
}

func WithSnapshotHook(h SnapshotHook) Option {
	return func(l *Loop) { l.snapshotHook = h }
}

func (l *Loop) snapshot(ctx context.Context, t *turns.Turn, phase string) {
	if l.snapshotHook != nil {
		l.snapshotHook(ctx, t, phase)
		return
	}
	if h, ok := TurnSnapshotHookFromContext(ctx); ok {
		h(ctx, t, phase)
	}
}

// RunLoop runs the tool calling workflow with iteration until no pending tools remain
// or the max iteration safety cap is hit.
func (l *Loop) RunLoop(ctx context.Context, initialTurn *turns.Turn) (*turns.Turn, error) {
	if l == nil {
		return nil, errors.New("tool loop is nil")
	}
	if l.eng == nil {
		return nil, errors.New("tool loop engine is nil")
	}
	if l.registry == nil {
		return nil, errors.New("tool loop registry is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	t := initialTurn
	if t == nil {
		t = &turns.Turn{}
	}

	ctx = toolcontext.WithRegistry(ctx, l.registry)

	maxIterations := l.loopCfg.MaxIterations
	if maxIterations <= 0 {
		maxIterations = DefaultLoopConfig().MaxIterations
	}

	if err := engine.KeyToolConfig.Set(&t.Data, engineToolConfig(maxIterations, l.toolCfg)); err != nil {
		return nil, errors.Wrap(err, "set tool config")
	}

	for i := 0; i < maxIterations; i++ {
		log.Debug().Int("iteration", i+1).Msg("toolloop: engine inference step")

		l.snapshot(ctx, t, "pre_inference")
		updated, err := l.eng.RunInference(ctx, t)
		if err != nil {
			return nil, err
		}
		l.snapshot(ctx, updated, "post_inference")

		calls := toolblocks.ExtractPendingToolCalls(updated)
		if len(calls) > 0 {
			l.maybePause(ctx, updated, StepPhaseAfterInference, "Review next action", map[string]any{
				"pending_tools": len(calls),
			})
		}
		if len(calls) == 0 {
			return updated, nil
		}

		results := l.executeTools(ctx, calls)

		var appended []toolblocks.ToolResult
		for _, r := range results {
			if r.Error != nil {
				appended = append(appended, toolblocks.ToolResult{ID: r.ToolCallID, Error: r.Error.Error()})
				continue
			}
			var content string
			if b, err := json.Marshal(r.Result); err == nil {
				content = string(b)
			} else {
				content = fmt.Sprintf("%v", r.Result)
			}
			appended = append(appended, toolblocks.ToolResult{ID: r.ToolCallID, Content: content})
		}
		toolblocks.AppendToolResultsBlocks(updated, appended)
		l.snapshot(ctx, updated, "post_tools")

		l.maybePause(ctx, updated, StepPhaseAfterTools, "Review tool results", nil)

		t = updated
	}

	log.Warn().Int("max_iterations", maxIterations).Msg("toolloop: maximum iterations reached")
	return t, fmt.Errorf("max iterations (%d) reached", maxIterations)
}

func engineToolConfig(maxIterations int, cfg tools.ToolConfig) engine.ToolConfig {
	if maxIterations <= 0 {
		maxIterations = DefaultLoopConfig().MaxIterations
	}
	return engine.ToolConfig{
		Enabled:           cfg.Enabled,
		ToolChoice:        engine.ToolChoice(cfg.ToolChoice),
		MaxIterations:     maxIterations,
		ExecutionTimeout:  cfg.ExecutionTimeout,
		MaxParallelTools:  cfg.MaxParallelTools,
		AllowedTools:      cfg.AllowedTools,
		ToolErrorHandling: engine.ToolErrorHandling(cfg.ToolErrorHandling),
		RetryConfig: engine.RetryConfig{
			MaxRetries:    cfg.RetryConfig.MaxRetries,
			BackoffBase:   cfg.RetryConfig.BackoffBase,
			BackoffFactor: cfg.RetryConfig.BackoffFactor,
		},
	}
}

type toolResult struct {
	ToolCallID string
	Result     any
	Error      error
}

func (l *Loop) executeTools(ctx context.Context, toolCalls []toolblocks.ToolCall) []toolResult {
	if len(toolCalls) == 0 {
		return nil
	}
	registry, ok := toolcontext.RegistryFrom(ctx)
	if !ok || registry == nil {
		out := make([]toolResult, len(toolCalls))
		for i, c := range toolCalls {
			out[i] = toolResult{ToolCallID: c.ID, Result: nil, Error: fmt.Errorf("no tool registry in context")}
		}
		return out
	}
	executor := l.executor
	if executor == nil {
		executor = tools.NewDefaultToolExecutor(l.toolCfg)
	}

	execCalls := make([]tools.ToolCall, 0, len(toolCalls))
	for _, call := range toolCalls {
		argBytes, _ := json.Marshal(call.Arguments)
		execCalls = append(execCalls, tools.ToolCall{ID: call.ID, Name: call.Name, Arguments: json.RawMessage(argBytes)})
	}

	execResults, err := executor.ExecuteToolCalls(ctx, execCalls, registry)
	out := make([]toolResult, len(toolCalls))
	for i, c := range toolCalls {
		if err != nil || i >= len(execResults) || execResults[i] == nil {
			out[i] = toolResult{ToolCallID: c.ID, Result: nil, Error: fmt.Errorf("no result returned")}
			continue
		}
		var resultErr error
		if execResults[i].Error != "" {
			resultErr = fmt.Errorf("%s", execResults[i].Error)
		}
		out[i] = toolResult{ToolCallID: c.ID, Result: execResults[i].Result, Error: resultErr}
	}
	return out
}

func (l *Loop) maybePause(ctx context.Context, t *turns.Turn, phase StepPhase, summary string, extra map[string]any) {
	if l.stepCtrl == nil || t == nil {
		return
	}

	sessionID, ok, err := turns.KeyTurnMetaSessionID.Get(t.Metadata)
	if err != nil || !ok || sessionID == "" {
		return
	}
	scope, ok := l.stepCtrl.IsEnabled(sessionID)
	if !ok {
		return
	}
	inferenceID, _, _ := turns.KeyTurnMetaInferenceID.Get(t.Metadata)

	meta := PauseMeta{
		Phase:       phase,
		Summary:     summary,
		DeadlineMs:  time.Now().Add(l.pauseTimeout).UnixMilli(),
		SessionID:   sessionID,
		InferenceID: inferenceID,
		TurnID:      t.ID,
		Scope:       scope,
		Extra:       extra,
	}
	meta, registered := l.stepCtrl.Pause(meta)
	if !registered {
		return
	}

	evMeta := events.EventMetadata{
		ID:          uuid.New(),
		SessionID:   sessionID,
		InferenceID: inferenceID,
		TurnID:      t.ID,
		Extra:       map[string]any{},
	}
	if scope.ConversationID != "" {
		evMeta.Extra["conversation_id"] = scope.ConversationID
	}
	events.PublishEventToContext(ctx, events.NewDebuggerPauseEvent(evMeta, meta.PauseID, string(meta.Phase), meta.Summary, meta.DeadlineMs, meta.Extra))

	_ = l.stepCtrl.Wait(ctx, meta.PauseID, l.pauseTimeout)
}
