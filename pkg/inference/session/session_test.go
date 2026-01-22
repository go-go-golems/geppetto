package session

import (
	"context"
	"testing"
	"time"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/stretchr/testify/require"
)

type fakeRunner struct {
	run func(ctx context.Context, t *turns.Turn) (*turns.Turn, error)
}

func (r fakeRunner) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	return r.run(ctx, t)
}

type fakeBuilder struct {
	build func(ctx context.Context, sessionID string) (InferenceRunner, error)
}

func (b fakeBuilder) Build(ctx context.Context, sessionID string) (InferenceRunner, error) {
	return b.build(ctx, sessionID)
}

func TestSession_StartInference_AppendsTurnOnSuccess(t *testing.T) {
	s := &Session{
		SessionID: "sess-1",
		Builder: fakeBuilder{build: func(ctx context.Context, sessionID string) (InferenceRunner, error) {
			return fakeRunner{run: func(ctx context.Context, in *turns.Turn) (*turns.Turn, error) {
				out := *in
				out.ID = "turn-2"
				if err := turns.KeyTurnMetaSessionID.Set(&out.Metadata, sessionID); err != nil {
					return nil, err
				}
				return &out, nil
			}}, nil
		}},
	}

	seed := &turns.Turn{}
	turns.AppendBlock(seed, turns.NewUserTextBlock("hi"))
	s.Append(seed)

	h, err := s.StartInference(context.Background())
	require.NoError(t, err)
	out, err := h.Wait()
	require.NoError(t, err)
	require.NotNil(t, out)
	sid, ok, err := turns.KeyTurnMetaSessionID.Get(out.Metadata)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "sess-1", sid)
	iid, ok, err := turns.KeyTurnMetaInferenceID.Get(out.Metadata)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, h.InferenceID, iid)
	require.NotEqual(t, "", iid)
	require.Len(t, s.Turns, 2) // seed + appended output
	require.Equal(t, out, s.Turns[len(s.Turns)-1])
}

func TestSession_StartInference_Cancel(t *testing.T) {
	s := &Session{
		SessionID: "sess-2",
		Builder: fakeBuilder{build: func(ctx context.Context, sessionID string) (InferenceRunner, error) {
			return fakeRunner{run: func(ctx context.Context, in *turns.Turn) (*turns.Turn, error) {
				<-ctx.Done()
				return nil, ctx.Err()
			}}, nil
		}},
	}

	seed := &turns.Turn{}
	turns.AppendBlock(seed, turns.NewUserTextBlock("hi"))
	s.Append(seed)

	h, err := s.StartInference(context.Background())
	require.NoError(t, err)
	require.True(t, h.IsRunning())
	h.Cancel()
	_, err = h.Wait()
	require.ErrorIs(t, err, context.Canceled)
}

func TestSession_StartInference_EmptyTurnFails(t *testing.T) {
	s := &Session{
		SessionID: "sess-empty",
		Builder: fakeBuilder{build: func(ctx context.Context, sessionID string) (InferenceRunner, error) {
			return fakeRunner{run: func(ctx context.Context, in *turns.Turn) (*turns.Turn, error) {
				return in, nil
			}}, nil
		}},
	}

	_, err := s.StartInference(context.Background())
	require.ErrorIs(t, err, ErrSessionEmptyTurn)

	s.Append(&turns.Turn{})
	_, err = s.StartInference(context.Background())
	require.ErrorIs(t, err, ErrSessionEmptyTurn)
}

func TestSession_StartInference_EmptySessionIDFails(t *testing.T) {
	s := &Session{
		SessionID: "",
		Builder: fakeBuilder{build: func(ctx context.Context, sessionID string) (InferenceRunner, error) {
			t.Fatal("builder should not be called when SessionID is empty")
			return nil, nil
		}},
	}

	seed := &turns.Turn{}
	turns.AppendBlock(seed, turns.NewUserTextBlock("hi"))
	s.Append(seed)

	_, err := s.StartInference(context.Background())
	require.ErrorIs(t, err, ErrSessionIDEmpty)
}

func TestSession_StartInference_OnlyOneActive(t *testing.T) {
	block := make(chan struct{})
	s := &Session{
		SessionID: "sess-3",
		Builder: fakeBuilder{build: func(ctx context.Context, sessionID string) (InferenceRunner, error) {
			return fakeRunner{run: func(ctx context.Context, in *turns.Turn) (*turns.Turn, error) {
				<-block
				out := *in
				out.ID = "turn-out"
				return &out, nil
			}}, nil
		}},
	}

	seed := &turns.Turn{}
	turns.AppendBlock(seed, turns.NewUserTextBlock("hi"))
	s.Append(seed)

	h1, err := s.StartInference(context.Background())
	require.NoError(t, err)

	_, err = s.StartInference(context.Background())
	require.ErrorIs(t, err, ErrSessionAlreadyActive)

	close(block)
	_, _ = h1.Wait()
}

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

type ctxCheckingEngine struct {
	t *testing.T
}

func (e ctxCheckingEngine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	// Ensure we can call through without mutating.
	out := *t
	out.ID = "turn-out"
	return &out, nil
}

func TestToolLoopEngineBuilder_RunsToolLoopAndPersists(t *testing.T) {
	p := &recordingPersister{}
	snapshotCalls := 0
	hook := func(ctx context.Context, t *turns.Turn, phase string) {
		snapshotCalls++
	}

	b := &ToolLoopEngineBuilder{
		Base:         engine.Engine(ctxCheckingEngine{t: t}),
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

func TestExecutionHandle_WaitNil(t *testing.T) {
	_, err := (*ExecutionHandle)(nil).Wait()
	require.ErrorIs(t, err, ErrExecutionHandleNil)
}

func TestExecutionHandle_CancelIsIdempotent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	h := newExecutionHandle("sess-x", "inf-x", &turns.Turn{}, cancel)
	h.Cancel()
	h.Cancel()

	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("expected ctx cancellation")
	}
}
