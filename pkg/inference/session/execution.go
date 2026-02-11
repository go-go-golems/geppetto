package session

import (
	"context"
	"errors"
	"sync"

	"github.com/go-go-golems/geppetto/pkg/turns"
)

var ErrExecutionHandleNil = errors.New("execution handle is nil")

// ExecutionHandle represents a single in-flight inference execution.
//
// It is cancelable and waitable. The underlying inference is always driven by context cancellation.
type ExecutionHandle struct {
	SessionID   string
	InferenceID string

	Input *turns.Turn

	done chan struct{}

	mu     sync.Mutex
	cancel context.CancelFunc
	out    *turns.Turn
	err    error
}

func newExecutionHandle(sessionID, inferenceID string, input *turns.Turn, cancel context.CancelFunc) *ExecutionHandle {
	return &ExecutionHandle{
		SessionID:   sessionID,
		InferenceID: inferenceID,
		Input:       input,
		done:        make(chan struct{}),
		cancel:      cancel,
	}
}

func (h *ExecutionHandle) setResult(out *turns.Turn, err error) {
	h.mu.Lock()
	h.out = out
	h.err = err
	close(h.done)
	h.cancel = nil
	h.mu.Unlock()
}

// Cancel cancels the in-flight inference. It is safe to call multiple times.
func (h *ExecutionHandle) Cancel() {
	if h == nil {
		return
	}
	h.mu.Lock()
	cancel := h.cancel
	h.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// Wait blocks until the inference completes and returns the output Turn and error.
func (h *ExecutionHandle) Wait() (*turns.Turn, error) {
	if h == nil {
		return nil, ErrExecutionHandleNil
	}
	<-h.done
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.out, h.err
}

// IsRunning reports whether the inference appears to still be running.
func (h *ExecutionHandle) IsRunning() bool {
	if h == nil {
		return false
	}
	select {
	case <-h.done:
		return false
	default:
		return true
	}
}
