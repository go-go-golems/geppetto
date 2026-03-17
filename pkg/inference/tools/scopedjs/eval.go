package scopedjs

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dop251/goja"
	gojengine "github.com/go-go-golems/go-go-goja/engine"
)

var evalCounter atomic.Uint64

type evalSnapshot struct {
	result  any
	promise *goja.Promise
}

type promiseStateSnapshot struct {
	State  goja.PromiseState
	Result any
}

func RunEval(ctx context.Context, rt *gojengine.Runtime, in EvalInput, opts EvalOptions) (EvalOutput, error) {
	if rt == nil || rt.VM == nil || rt.Owner == nil {
		return EvalOutput{}, fmt.Errorf("runtime is nil")
	}

	opts = applyEvalDefaults(opts)
	if ctx == nil {
		ctx = context.Background()
	}
	var cancel context.CancelFunc
	if opts.Timeout > 0 {
		if _, ok := ctx.Deadline(); !ok {
			ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
			defer cancel()
		}
	}

	start := time.Now()
	consoleCapture := newConsoleCapture(opts.MaxOutputChars)
	inputVar := fmt.Sprintf("__scopedjs_input_%d__", evalCounter.Add(1))

	if err := prepareEval(ctx, rt, inputVar, in.Input, opts.CaptureConsole, consoleCapture); err != nil {
		return EvalOutput{}, err
	}
	defer func() {
		_ = cleanupEval(context.Background(), rt, inputVar, opts.CaptureConsole, consoleCapture)
	}()

	result, execErr := executeEval(ctx, rt, inputVar, in.Code)
	if execErr != nil {
		return EvalOutput{
			Console:    consoleCapture.snapshot(),
			Error:      truncateString(execErr.Error(), opts.MaxOutputChars),
			DurationMs: time.Since(start).Milliseconds(),
		}, nil
	}

	return EvalOutput{
		Result:     truncateResult(result, opts.MaxOutputChars),
		Console:    consoleCapture.snapshot(),
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

func prepareEval(ctx context.Context, rt *gojengine.Runtime, inputVar string, input map[string]any, captureConsole bool, consoleCapture *consoleCapture) error {
	_, err := rt.Owner.Call(ctx, "scopedjs.prepare", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if err := vm.Set(inputVar, input); err != nil {
			return nil, err
		}
		if captureConsole {
			original := vm.Get("console")
			consoleCapture.original = original
			consoleObj := vm.NewObject()
			for _, level := range []string{"log", "error", "warn", "info", "debug", "table"} {
				level := level
				if err := consoleObj.Set(level, func(call goja.FunctionCall) goja.Value {
					consoleCapture.append(level, stringifyConsoleArgs(call.Arguments))
					return goja.Undefined()
				}); err != nil {
					return nil, err
				}
			}
			if err := vm.GlobalObject().Set("console", consoleObj); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	return err
}

func cleanupEval(ctx context.Context, rt *gojengine.Runtime, inputVar string, captureConsole bool, consoleCapture *consoleCapture) error {
	if rt == nil || rt.Owner == nil {
		return nil
	}
	_, err := rt.Owner.Call(ctx, "scopedjs.cleanup", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if err := vm.GlobalObject().Delete(inputVar); err != nil {
			return nil, err
		}
		if captureConsole {
			if err := vm.GlobalObject().Set("console", consoleCapture.original); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	return err
}

func executeEval(ctx context.Context, rt *gojengine.Runtime, inputVar string, code string) (any, error) {
	ret, err := rt.Owner.Call(ctx, "scopedjs.eval", func(_ context.Context, vm *goja.Runtime) (any, error) {
		v, err := vm.RunString(wrapEvalBody(inputVar, code))
		if err != nil {
			return nil, err
		}
		if promise, ok := v.Export().(*goja.Promise); ok {
			return evalSnapshot{promise: promise}, nil
		}
		return evalSnapshot{result: exportValue(v)}, nil
	})
	if err != nil {
		return nil, err
	}
	snap, ok := ret.(evalSnapshot)
	if !ok {
		return nil, fmt.Errorf("unexpected eval snapshot type %T", ret)
	}
	if snap.promise == nil {
		return snap.result, nil
	}
	return waitForPromise(ctx, rt, snap.promise)
}

func waitForPromise(ctx context.Context, rt *gojengine.Runtime, promise *goja.Promise) (any, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		ret, err := rt.Owner.Call(ctx, "scopedjs.promise-state", func(_ context.Context, vm *goja.Runtime) (any, error) {
			return promiseStateSnapshot{
				State:  promise.State(),
				Result: exportValue(promise.Result()),
			}, nil
		})
		if err != nil {
			return nil, err
		}
		snap, ok := ret.(promiseStateSnapshot)
		if !ok {
			return nil, fmt.Errorf("unexpected promise snapshot type %T", ret)
		}
		switch snap.State {
		case goja.PromiseStatePending:
			time.Sleep(5 * time.Millisecond)
		case goja.PromiseStateRejected:
			return nil, fmt.Errorf("promise rejected: %v", snap.Result)
		case goja.PromiseStateFulfilled:
			return snap.Result, nil
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func wrapEvalBody(inputVar string, code string) string {
	return "(async function(input) {\n" + code + "\n})(" + inputVar + ")"
}

func exportValue(v goja.Value) any {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil
	}
	return v.Export()
}

func stringifyConsoleArgs(args []goja.Value) string {
	if len(args) == 0 {
		return ""
	}
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == nil || goja.IsUndefined(arg) || goja.IsNull(arg) {
			parts = append(parts, "undefined")
			continue
		}
		parts = append(parts, fmt.Sprint(arg.Export()))
	}
	return strings.Join(parts, " ")
}

func truncateResult(v any, maxLen int) any {
	if maxLen <= 0 {
		return v
	}
	s, ok := v.(string)
	if !ok {
		return v
	}
	return truncateString(s, maxLen)
}

func truncateString(v string, maxLen int) string {
	if maxLen <= 0 || len(v) <= maxLen {
		return v
	}
	if maxLen <= 3 {
		return v[:maxLen]
	}
	return v[:maxLen-3] + "..."
}

func applyEvalDefaults(opts EvalOptions) EvalOptions {
	def := DefaultEvalOptions()
	if opts.Timeout <= 0 {
		opts.Timeout = def.Timeout
	}
	if opts.MaxOutputChars <= 0 {
		opts.MaxOutputChars = def.MaxOutputChars
	}
	if opts.StateMode == "" {
		opts.StateMode = def.StateMode
	}
	return opts
}

func resolveEvalOptions(base EvalOptions, override EvalOptions) EvalOptions {
	if base == (EvalOptions{}) {
		base = DefaultEvalOptions()
	} else {
		base = applyEvalDefaults(base)
	}
	if override == (EvalOptions{}) {
		return base
	}
	if override.Timeout > 0 {
		base.Timeout = override.Timeout
	}
	if override.MaxOutputChars > 0 {
		base.MaxOutputChars = override.MaxOutputChars
	}
	if override.StateMode != "" {
		base.StateMode = override.StateMode
	}
	if override.CaptureConsole {
		base.CaptureConsole = true
	}
	return base
}

type consoleCapture struct {
	mu       sync.Mutex
	lines    []ConsoleLine
	maxChars int
	original goja.Value
}

func newConsoleCapture(maxChars int) *consoleCapture {
	return &consoleCapture{maxChars: maxChars}
}

func (c *consoleCapture) append(level string, text string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lines = append(c.lines, ConsoleLine{
		Level: level,
		Text:  truncateString(text, c.maxChars),
	})
}

func (c *consoleCapture) snapshot() []ConsoleLine {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]ConsoleLine(nil), c.lines...)
}
