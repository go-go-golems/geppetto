package runner

import (
	"context"

	"github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

// Start prepares a run and starts inference asynchronously.
func (r *Runner) Start(ctx context.Context, req StartRequest) (*PreparedRun, *session.ExecutionHandle, error) {
	prepared, err := r.Prepare(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	handle, err := prepared.Session.StartInference(ctx)
	if err != nil {
		return nil, nil, err
	}
	return prepared, handle, nil
}

// Run prepares a run, starts inference, and waits for completion.
func (r *Runner) Run(ctx context.Context, req StartRequest) (*PreparedRun, *turns.Turn, error) {
	prepared, handle, err := r.Start(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	out, err := handle.Wait()
	if err != nil {
		return prepared, nil, err
	}
	return prepared, out, nil
}
