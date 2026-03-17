package scopedjs

import (
	"context"
	"fmt"
	"sync"

	gojengine "github.com/go-go-golems/go-go-goja/engine"
)

type RuntimeExecutor struct {
	Runtime *gojengine.Runtime
	mu      sync.Mutex
}

func NewRuntimeExecutor(rt *gojengine.Runtime) *RuntimeExecutor {
	if rt == nil {
		return nil
	}
	return &RuntimeExecutor{Runtime: rt}
}

func (r *RuntimeExecutor) RunEval(ctx context.Context, in EvalInput, opts EvalOptions) (EvalOutput, error) {
	if r == nil || r.Runtime == nil {
		return EvalOutput{}, fmt.Errorf("runtime is nil")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return RunEval(ctx, r.Runtime, in, opts)
}

func executorFromBuildResult[Meta any](handle *BuildResult[Meta]) *RuntimeExecutor {
	if handle == nil {
		return nil
	}
	if handle.Executor != nil {
		return handle.Executor
	}
	return NewRuntimeExecutor(handle.Runtime)
}
