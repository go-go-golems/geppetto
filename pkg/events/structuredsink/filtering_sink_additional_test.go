package structuredsink

import (
	"context"
	"testing"
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilteringSink_PassThrough_NoStructured(t *testing.T) {
	col := &eventCollector{}
	sink := NewFilteringSink(col, Options{})

	id := uuid.New()
	meta := events.EventMetadata{ID: id, RunID: "run-1", TurnID: "turn-1"}

	// Two partials and a final without any structured tags
	in := []events.Event{
		events.NewPartialCompletionEvent(meta, "Hello, ", "Hello, "),
		events.NewPartialCompletionEvent(meta, "world!", "Hello, world!"),
		events.NewFinalEvent(meta, "Hello, world!"),
	}

	for _, ev := range in {
		require.NoError(t, sink.PublishEvent(ev))
	}

	// Expect same number of forwarded events
	require.Equal(t, len(in), len(col.list))

	// Check partials preserved
	p1, ok := col.list[0].(*events.EventPartialCompletion)
	require.True(t, ok)
	assert.Equal(t, "Hello, ", p1.Delta)
	assert.Equal(t, "Hello, ", p1.Completion)

	p2, ok := col.list[1].(*events.EventPartialCompletion)
	require.True(t, ok)
	assert.Equal(t, "world!", p2.Delta)
	assert.Equal(t, "Hello, world!", p2.Completion)

	f, ok := col.list[2].(*events.EventFinal)
	require.True(t, ok)
	assert.Equal(t, "Hello, world!", f.Text)
}

func TestFilteringSink_ContextLifecycle(t *testing.T) {
	col := &eventCollector{}
	baseCtx, baseCancel := context.WithCancel(context.Background())
	defer baseCancel()
	sink := NewFilteringSinkWithContext(baseCtx, col, Options{})

	id := uuid.New()
	meta := events.EventMetadata{ID: id}

	// First partial creates stream state and stream context
	require.NoError(t, sink.PublishEvent(events.NewPartialCompletionEvent(meta, "x", "x")))

	// Access internal state for this stream
	st := sink.getState(meta)
	require.NotNil(t, st)
	require.NotNil(t, st.ctx)

	// Context should not be cancelled yet
	select {
	case <-st.ctx.Done():
		t.Fatal("stream context cancelled too early")
	default:
	}

	// Final should cancel and remove state
	require.NoError(t, sink.PublishEvent(events.NewFinalEvent(meta, "x")))

	// st.ctx must be cancelled shortly
	select {
	case <-st.ctx.Done():
		// ok
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected stream context to be cancelled after final")
	}

	// State removed from map
	sink.mu.Lock()
	_, ok := sink.byStreamID[id]
	sink.mu.Unlock()
	require.False(t, ok, "stream state should be deleted after final")
}
