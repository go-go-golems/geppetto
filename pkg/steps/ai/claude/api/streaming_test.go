package api

import (
	"context"
	"net/http"
	"testing"
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
