package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
)

type currentToolCallKey struct{}

// WithCurrentToolCall annotates context with the current tool call for executor hook consumers.
func WithCurrentToolCall(ctx context.Context, call ToolCall) context.Context {
	return context.WithValue(ctx, currentToolCallKey{}, call)
}

// CurrentToolCallFromContext returns the current tool call if available.
func CurrentToolCallFromContext(ctx context.Context) (ToolCall, bool) {
	if ctx == nil {
		return ToolCall{}, false
	}
	v := ctx.Value(currentToolCallKey{})
	call, ok := v.(ToolCall)
	return call, ok
}

// ToolExecutorExt defines lifecycle hooks that can be overridden.
type ToolExecutorExt interface {
	// PreExecute may mutate the call (e.g., inject auth) or reject it.
	PreExecute(ctx context.Context, call ToolCall, registry ToolRegistry) (ToolCall, error)

	// IsAllowed adds authorization beyond AllowedTools/IsToolAllowed in config.
	IsAllowed(ctx context.Context, call ToolCall) bool

	// MaskArguments returns a compact and masked JSON string for event payloads.
	MaskArguments(ctx context.Context, call ToolCall) string

	// PublishStart/PublishResult control event publishing.
	PublishStart(ctx context.Context, call ToolCall, maskedArgs string)
	PublishResult(ctx context.Context, call ToolCall, result *ToolResult)

	// ShouldRetry decides retry and backoff after a failed attempt.
	ShouldRetry(ctx context.Context, attempt int, res *ToolResult, execErr error) (retry bool, backoff time.Duration)

	// MaxParallel decides concurrency for a batch.
	MaxParallel(ctx context.Context, calls []ToolCall) int
}

// BaseToolExecutor hosts orchestration and default hook implementations.
type BaseToolExecutor struct {
	ToolExecutorExt // self reference used for dynamic dispatch
	config          ToolConfig
}

func NewBaseToolExecutor(cfg ToolConfig) *BaseToolExecutor {
	b := &BaseToolExecutor{config: cfg}
	b.ToolExecutorExt = b // default to self; outer types overwrite this
	return b
}

// Ensure BaseToolExecutor satisfies ToolExecutorExt by default
var _ ToolExecutorExt = (*BaseToolExecutor)(nil)

// Default hooks
func (b *BaseToolExecutor) PreExecute(ctx context.Context, call ToolCall, _ ToolRegistry) (ToolCall, error) {
	return call, nil
}

func (b *BaseToolExecutor) IsAllowed(_ context.Context, call ToolCall) bool {
	return b.config.IsToolAllowed(call.Name)
}

func (b *BaseToolExecutor) MaskArguments(_ context.Context, call ToolCall) string {
	// Compact JSON for events by default
	if len(call.Arguments) == 0 {
		return ""
	}
	var tmp any
	if err := json.Unmarshal(call.Arguments, &tmp); err == nil {
		if bts, err2 := json.Marshal(tmp); err2 == nil {
			return string(bts)
		}
	}
	return string(call.Arguments)
}

func (b *BaseToolExecutor) PublishStart(ctx context.Context, call ToolCall, masked string) {
	events.PublishEventToContext(ctx, events.NewToolCallExecuteEvent(
		events.EventMetadata{},
		events.ToolCall{ID: call.ID, Name: call.Name, Input: masked},
	))
}

func (b *BaseToolExecutor) PublishResult(ctx context.Context, call ToolCall, res *ToolResult) {
	payload := ""
	if res != nil && res.Result != nil {
		if bts, err := json.Marshal(res.Result); err == nil {
			payload = string(bts)
		} else {
			payload = fmt.Sprintf("%v", res.Result)
		}
	}
	if res != nil && res.Error != "" {
		if payload == "" {
			payload = fmt.Sprintf("Error: %s", res.Error)
		} else {
			payload = fmt.Sprintf("%s | Error: %s", payload, res.Error)
		}
	}
	events.PublishEventToContext(ctx, events.NewToolCallExecutionResultEvent(
		events.EventMetadata{},
		events.ToolResult{ID: call.ID, Result: payload},
	))
}

func (b *BaseToolExecutor) ShouldRetry(_ context.Context, attempt int, _ *ToolResult, _ error) (bool, time.Duration) {
	if b.config.ToolErrorHandling != ToolErrorRetry {
		return false, 0
	}
	if attempt >= b.config.RetryConfig.MaxRetries {
		return false, 0
	}
	// exponential backoff
	backoff := time.Duration(float64(b.config.RetryConfig.BackoffBase) * pow(b.config.RetryConfig.BackoffFactor, float64(attempt)))
	return true, backoff
}

func (b *BaseToolExecutor) MaxParallel(_ context.Context, _ []ToolCall) int {
	if b.config.MaxParallelTools <= 1 {
		return 1
	}
	return b.config.MaxParallelTools
}

// Orchestration using dynamic dispatch to hooks
func (b *BaseToolExecutor) ExecuteToolCall(ctx context.Context, call ToolCall, registry ToolRegistry) (*ToolResult, error) {
	start := time.Now()

	// PreExecute (allow mutation)
	var err error
	call, err = b.ToolExecutorExt.PreExecute(ctx, call, registry)
	if err != nil {
		return &ToolResult{ID: call.ID, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	ctx = WithCurrentToolCall(ctx, call)

	// Lookup + allow checks
	def, err := registry.GetTool(call.Name)
	if err != nil {
		return &ToolResult{ID: call.ID, Error: fmt.Sprintf("tool not found: %s", call.Name), Duration: time.Since(start)}, nil
	}
	if !b.ToolExecutorExt.IsAllowed(ctx, call) {
		return &ToolResult{ID: call.ID, Error: fmt.Sprintf("tool not allowed: %s", call.Name), Duration: time.Since(start)}, nil
	}

	// Publish start
	b.ToolExecutorExt.PublishStart(ctx, call, b.ToolExecutorExt.MaskArguments(ctx, call))

	// Execute with retries
	var result *ToolResult
	var execErr error
	for attempt := 0; ; attempt++ {
		r, e := b.executeOnce(ctx, call, def)
		if r != nil {
			result = r
		}
		execErr = e
		if execErr == nil && (result == nil || result.Error == "") {
			break
		}
		retry, backoff := b.ToolExecutorExt.ShouldRetry(ctx, attempt, result, execErr)
		if !retry {
			break
		}
		select {
		case <-ctx.Done():
			return &ToolResult{ID: call.ID, Error: "context cancelled during retry backoff", Duration: time.Since(start)}, ctx.Err()
		case <-time.After(backoff):
		}
	}

	if result != nil {
		result.ID = call.ID
		result.Duration = time.Since(start)
	}

	// Publish result
	b.ToolExecutorExt.PublishResult(ctx, call, result)
	return result, execErr
}

func (b *BaseToolExecutor) ExecuteToolCalls(ctx context.Context, calls []ToolCall, registry ToolRegistry) ([]*ToolResult, error) {
	if len(calls) == 0 {
		return nil, nil
	}
	if len(calls) == 1 {
		r, err := b.ExecuteToolCall(ctx, calls[0], registry)
		return []*ToolResult{r}, err
	}

	maxPar := b.ToolExecutorExt.MaxParallel(ctx, calls)
	if maxPar <= 1 {
		return b.executeSequential(ctx, calls, registry)
	}
	return b.executeParallel(ctx, calls, registry, maxPar)
}

// Internal helpers
func (b *BaseToolExecutor) executeOnce(ctx context.Context, call ToolCall, def *ToolDefinition) (*ToolResult, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return &ToolResult{Error: "execution cancelled"}, ctx.Err()
	default:
	}

	// Execute the tool function
	out, err := def.Function.ExecuteWithContext(ctx, call.Arguments)
	if err != nil {
		return &ToolResult{Error: err.Error()}, nil
	}
	return &ToolResult{Result: out}, nil
}

func (b *BaseToolExecutor) executeSequential(ctx context.Context, calls []ToolCall, registry ToolRegistry) ([]*ToolResult, error) {
	results := make([]*ToolResult, len(calls))
	for i, c := range calls {
		r, err := b.ExecuteToolCall(ctx, c, registry)
		if err != nil {
			return results, err
		}
		results[i] = r
		if r != nil && r.Error != "" && b.config.ToolErrorHandling == ToolErrorAbort {
			return results, fmt.Errorf("tool execution aborted due to error in %s: %s", c.Name, r.Error)
		}
	}
	return results, nil
}

func (b *BaseToolExecutor) executeParallel(ctx context.Context, calls []ToolCall, registry ToolRegistry, maxParallel int) ([]*ToolResult, error) {
	results := make([]*ToolResult, len(calls))
	errs := make([]error, len(calls))
	sem := make(chan struct{}, maxParallel)
	var wg sync.WaitGroup
	for i, c := range calls {
		wg.Add(1)
		go func(idx int, call ToolCall) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			r, err := b.ExecuteToolCall(ctx, call, registry)
			results[idx] = r
			errs[idx] = err
		}(i, c)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			return results, err
		}
		if results[i] != nil && results[i].Error != "" && b.config.ToolErrorHandling == ToolErrorAbort {
			return results, fmt.Errorf("tool execution aborted due to error in %s: %s", calls[i].Name, results[i].Error)
		}
	}
	return results, nil
}

// pow is a simple integer power function to avoid importing math
func pow(base float64, exp float64) float64 {
	if exp == 0 {
		return 1
	}
	if exp == 1 {
		return base
	}
	result := base
	for i := 1; i < int(exp); i++ {
		result *= base
	}
	return result
}
