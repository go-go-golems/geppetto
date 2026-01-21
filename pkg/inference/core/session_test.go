package core

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/state"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

type sinkAssertingEngine struct {
	t            *testing.T
	wantSink     events.EventSink
	calls        atomic.Int64
	returnedTurn *turns.Turn
}

func (e *sinkAssertingEngine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	e.calls.Add(1)
	sinks := events.GetEventSinks(ctx)
	found := false
	for _, s := range sinks {
		if s == e.wantSink {
			found = true
			break
		}
	}
	if !found {
		e.t.Fatalf("expected sink to be present on context; got %d sinks", len(sinks))
	}
	if e.returnedTurn != nil {
		return e.returnedTurn, nil
	}
	out := &turns.Turn{}
	if t != nil {
		out.RunID = t.RunID
		out.Blocks = append([]turns.Block(nil), t.Blocks...)
		out.Metadata = t.Metadata.Clone()
		out.Data = t.Data.Clone()
	}
	return out, nil
}

type blockingEngine struct{}

func (e *blockingEngine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

type captureSink struct{ count atomic.Int64 }

func (s *captureSink) PublishEvent(event events.Event) error {
	s.count.Add(1)
	return nil
}

func TestSession_AttachesEventSinksToContext(t *testing.T) {
	t.Parallel()

	sink := &captureSink{}
	eng := &sinkAssertingEngine{t: t, wantSink: sink}
	inf := state.NewInferenceState("run-1", &turns.Turn{RunID: "run-1"}, eng)

	sess := &Session{
		State:      inf,
		EventSinks: []events.EventSink{sink},
	}

	_, err := sess.RunInference(context.Background(), &turns.Turn{RunID: "run-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inf.IsRunning() {
		t.Fatalf("expected inference to be finished")
	}
}

func TestSession_CancelRunCancelsInference(t *testing.T) {
	t.Parallel()

	inf := state.NewInferenceState("run-1", &turns.Turn{RunID: "run-1"}, &blockingEngine{})
	sess := &Session{State: inf}

	errCh := make(chan error, 1)
	go func() {
		_, err := sess.RunInference(context.Background(), inf.Turn)
		errCh <- err
	}()

	deadline := time.Now().Add(2 * time.Second)
	for !inf.IsRunning() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if !inf.IsRunning() {
		t.Fatalf("expected inference to be running before cancellation")
	}

	if err := inf.CancelRun(); err != nil {
		t.Fatalf("CancelRun returned error: %v", err)
	}

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for inference to return after cancellation")
	}

	if inf.IsRunning() {
		t.Fatalf("expected inference to be finished after cancellation")
	}
}
