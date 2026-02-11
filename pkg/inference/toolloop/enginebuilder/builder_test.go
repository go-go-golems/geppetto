package enginebuilder

import (
	"context"
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
