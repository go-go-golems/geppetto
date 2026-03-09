package api

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type errReadCloser struct {
	err error
}

func (r *errReadCloser) Read(_ []byte) (int, error) {
	return 0, r.err
}

func (r *errReadCloser) Close() error {
	return nil
}

func TestStreamEventsDoesNotPanicOnDeadlineExceeded(t *testing.T) {
	resp := &http.Response{
		Body: &errReadCloser{err: context.DeadlineExceeded},
	}
	events := make(chan StreamingEvent, 1)

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("streamEvents panicked on context deadline exceeded: %v", recovered)
		}
	}()

	streamEvents(context.Background(), resp, events)
}

func TestStreamEventsDoesNotWarnOnCleanEOF(t *testing.T) {
	var logBuf bytes.Buffer
	previousLogger := log.Logger
	log.Logger = zerolog.New(&logBuf).Level(zerolog.TraceLevel)
	defer func() {
		log.Logger = previousLogger
	}()

	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader("data: {\"type\":\"message_stop\"}\n\n")),
	}
	events := make(chan StreamingEvent, 1)

	streamEvents(context.Background(), resp, events)

	select {
	case event := <-events:
		if event.Type != MessageStopType {
			t.Fatalf("expected %q event, got %q", MessageStopType, event.Type)
		}
	default:
		t.Fatal("expected a parsed streaming event")
	}

	if strings.Contains(logBuf.String(), "Streaming response ended before completion") {
		t.Fatalf("expected clean EOF to avoid premature warning, got logs: %s", logBuf.String())
	}
}
