package session

import (
	"context"
	"testing"
	"time"

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

func TestSession_StartInference_MutatesLatestTurnOnSuccess(t *testing.T) {
	s := &Session{
		SessionID: "sess-1",
		Builder: fakeBuilder{build: func(ctx context.Context, sessionID string) (InferenceRunner, error) {
			return fakeRunner{run: func(ctx context.Context, in *turns.Turn) (*turns.Turn, error) {
				turns.AppendBlock(in, turns.NewAssistantTextBlock("ok"))
				return in, nil
			}}, nil
		}},
	}

	seed, err := s.AppendNewTurnFromUserPrompt("hi")
	require.NoError(t, err)

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
	require.Len(t, s.Turns, 1) // in-place mutation of latest turn
	require.Equal(t, seed, s.Turns[len(s.Turns)-1])
	require.Equal(t, seed, out)
	require.GreaterOrEqual(t, len(seed.Blocks), 2)
}

func TestSession_StartInference_InjectsSessionMetaIntoContext(t *testing.T) {
	var (
		seenSessionID   string
		seenInferenceID string
	)
	s := &Session{
		SessionID: "sess-meta",
		Builder: fakeBuilder{build: func(ctx context.Context, sessionID string) (InferenceRunner, error) {
			return fakeRunner{run: func(ctx context.Context, in *turns.Turn) (*turns.Turn, error) {
				seenSessionID = SessionIDFromContext(ctx)
				seenInferenceID = InferenceIDFromContext(ctx)
				turns.AppendBlock(in, turns.NewAssistantTextBlock("ok"))
				return in, nil
			}}, nil
		}},
	}

	_, err := s.AppendNewTurnFromUserPrompt("hi")
	require.NoError(t, err)

	h, err := s.StartInference(context.Background())
	require.NoError(t, err)
	_, err = h.Wait()
	require.NoError(t, err)
	require.Equal(t, "sess-meta", seenSessionID)
	require.Equal(t, h.InferenceID, seenInferenceID)
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

	_, err := s.AppendNewTurnFromUserPrompt("hi")
	require.NoError(t, err)

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
				turns.AppendBlock(in, turns.NewAssistantTextBlock("ok"))
				return in, nil
			}}, nil
		}},
	}

	_, err := s.AppendNewTurnFromUserPrompt("hi")
	require.NoError(t, err)

	h1, err := s.StartInference(context.Background())
	require.NoError(t, err)

	_, err = s.StartInference(context.Background())
	require.ErrorIs(t, err, ErrSessionAlreadyActive)

	close(block)
	_, _ = h1.Wait()
}

func TestSession_AppendNewTurnFromUserPrompt_AssignsNewTurnID(t *testing.T) {
	s := &Session{SessionID: "sess-turnid"}

	t1, err := s.AppendNewTurnFromUserPrompt("hi")
	require.NoError(t, err)
	require.NotEmpty(t, t1.ID)

	t2, err := s.AppendNewTurnFromUserPrompt("again")
	require.NoError(t, err)
	require.NotEmpty(t, t2.ID)
	require.NotEqual(t, t1.ID, t2.ID)
	require.Len(t, s.Turns, 2)
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
