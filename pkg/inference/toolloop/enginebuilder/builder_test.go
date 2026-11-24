package enginebuilder

import (
	"context"
	"errors"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/stretchr/testify/require"
)

type recordingPersister struct {
	sessionID string
	t         *turns.Turn
}

func (p *recordingPersister) PersistTurn(ctx context.Context, t *turns.Turn) error {
	if sid, ok, err := turns.KeyTurnMetaSessionID.Get(t.Metadata); err == nil && ok {
		p.sessionID = sid
	}
	p.t = t
	return nil
}

type passthroughEngine struct{}

func (e passthroughEngine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	out := *t
	return &out, nil
}

func TestBuilder_RunsToolLoopAndPersists(t *testing.T) {
	p := &recordingPersister{}
	snapshotCalls := 0
	hook := func(ctx context.Context, t *turns.Turn, phase string) {
		snapshotCalls++
	}

	b := &Builder{
		Base:         engine.Engine(passthroughEngine{}),
		Registry:     tools.NewInMemoryToolRegistry(),
		SnapshotHook: hook,
		Persister:    p,
	}

	runner, err := b.Build(context.Background(), "sess-4")
	require.NoError(t, err)

	out, err := runner.RunInference(context.Background(), &turns.Turn{ID: "turn-in"})
	require.NoError(t, err)
	require.NotNil(t, out)

	sid, ok, err := turns.KeyTurnMetaSessionID.Get(out.Metadata)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "sess-4", sid)
	require.Equal(t, "sess-4", p.sessionID)
	require.NotNil(t, p.t)

	// Tool loop calls snapshot hook at least pre/post inference on the first iteration.
	require.GreaterOrEqual(t, snapshotCalls, 2)
}

type countingEngine struct{ calls *int }

func (e countingEngine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	*e.calls++
	out := *t
	return &out, nil
}

// Without a registry the runner makes one plain model call; the guard runs
// before it too (round 0), and a stop means the engine is not called.
func TestBuilder_RoundGuardRunsWithoutARegistry(t *testing.T) {
	stop := errors.New("budget used up")
	calls := 0
	var rounds []int
	allow := true
	b := New(
		WithBase(countingEngine{calls: &calls}),
		WithRoundGuard(func(_ context.Context, _ *turns.Turn, round int) error {
			rounds = append(rounds, round)
			if !allow {
				return stop
			}
			return nil
		}),
	)
	runner, err := b.Build(context.Background(), "sess-guard-plain")
	require.NoError(t, err)

	_, err = runner.RunInference(context.Background(), &turns.Turn{ID: "turn-1"})
	require.NoError(t, err)
	require.Equal(t, 1, calls)

	allow = false
	in := &turns.Turn{ID: "turn-2"}
	out, err := runner.RunInference(context.Background(), in)
	require.ErrorIs(t, err, stop)
	require.Equal(t, 1, calls, "a stopped call never reaches the engine")
	require.NotNil(t, out, "the turn so far is returned with the guard's error, as the tool loop does")
	require.Equal(t, []int{0, 0}, rounds)
}

func TestBuilder_RoundGuardStopsTheToolLoop(t *testing.T) {
	stop := errors.New("budget used up")
	rounds := 0
	b := New(
		WithBase(passthroughEngine{}),
		WithToolRegistry(tools.NewInMemoryToolRegistry()),
		WithRoundGuard(func(context.Context, *turns.Turn, int) error { rounds++; return stop }),
	)
	runner, err := b.Build(context.Background(), "sess-guard")
	require.NoError(t, err)
	_, err = runner.RunInference(context.Background(), &turns.Turn{ID: "turn-in"})
	require.ErrorIs(t, err, stop)
	require.Equal(t, 1, rounds)
}
