package runner

import (
	"context"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop/enginebuilder"
	geptools "github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

// Prepare assembles a session, engine, registry, and seed turn from resolved
// runtime input without starting inference.
func (r *Runner) Prepare(ctx context.Context, req StartRequest) (*PreparedRun, error) {
	if r == nil {
		return nil, ErrRunnerNil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(req.Prompt) == "" && (req.SeedTurn == nil || len(req.SeedTurn.Blocks) == 0) {
		return nil, ErrPromptAndSeedEmpty
	}

	eng, _, err := r.buildEngine(ctx, req.Runtime)
	if err != nil {
		return nil, err
	}

	registry, err := buildRegistry(ctx, appendToolRegistrars(r.toolRegistrars, req.Runtime.ToolRegistrars), req.Runtime.ToolNames)
	if err != nil {
		return nil, err
	}

	sess := session.NewSession()
	if sessionID := strings.TrimSpace(req.SessionID); sessionID != "" {
		sess.SessionID = sessionID
	}
	sess.Builder = &enginebuilder.Builder{
		Base:             eng,
		Registry:         registry,
		LoopConfig:       cloneLoopConfig(r.loopConfig),
		ToolConfig:       cloneToolConfig(r.toolConfig),
		ToolExecutor:     r.toolExecutor,
		EventSinks:       appendEventSinks(r.eventSinks, req.EventSinks),
		SnapshotHook:     chooseSnapshotHook(r.snapshotHook, req.SnapshotHook),
		StepController:   r.stepController,
		StepPauseTimeout: r.stepPauseTimeout,
		Persister:        choosePersister(r.persister, req.Persister),
	}

	turn, err := appendSeedTurn(sess, req)
	if err != nil {
		return nil, err
	}

	return &PreparedRun{
		Runtime:  req.Runtime,
		Engine:   eng,
		Registry: registry,
		Session:  sess,
		Turn:     turn,
	}, nil
}

func appendSeedTurn(sess *session.Session, req StartRequest) (*turns.Turn, error) {
	if req.SeedTurn == nil {
		return sess.AppendNewTurnFromUserPrompt(req.Prompt)
	}

	seed := req.SeedTurn.Clone()
	if seed == nil {
		seed = &turns.Turn{}
	}
	if strings.TrimSpace(req.Prompt) != "" {
		turns.AppendBlock(seed, turns.NewUserTextBlock(req.Prompt))
	}
	sess.Append(seed)
	return seed, nil
}

func appendToolRegistrars(a []ToolRegistrar, b []ToolRegistrar) []ToolRegistrar {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	out := make([]ToolRegistrar, 0, len(a)+len(b))
	out = append(out, a...)
	out = append(out, b...)
	return out
}

func appendEventSinks(a []events.EventSink, b []events.EventSink) []events.EventSink {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	out := make([]events.EventSink, 0, len(a)+len(b))
	out = append(out, a...)
	out = append(out, b...)
	return out
}

func chooseSnapshotHook(base, override toolloop.SnapshotHook) toolloop.SnapshotHook {
	if override != nil {
		return override
	}
	return base
}

func choosePersister(base, override enginebuilder.TurnPersister) enginebuilder.TurnPersister {
	if override != nil {
		return override
	}
	return base
}

func cloneLoopConfig(cfg toolloop.LoopConfig) *toolloop.LoopConfig {
	copied := cfg
	return &copied
}

func cloneToolConfig(cfg geptools.ToolConfig) *geptools.ToolConfig {
	copied := cfg
	return &copied
}
