package geppetto

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/geppetto/pkg/events"
	inferenceengine "github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/turns"
	gojengine "github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/jsevents"
)

func newJSEventEmitterTestRuntime(t *testing.T) (*gojengine.Runtime, *moduleRuntime) {
	t.Helper()
	factory, err := gojengine.NewBuilder(
		gojengine.WithDataOnlyDefaultRegistryModules(true),
	).
		UseModuleMiddleware(gojengine.Pipeline()).
		WithRuntimeInitializers(jsevents.Install()).
		Build()
	if err != nil {
		t.Fatalf("failed creating runtime factory: %v", err)
	}
	rt, err := factory.NewRuntime(gojengine.WithStartupContext(context.Background()), gojengine.WithLifetimeContext(context.Background()))
	if err != nil {
		t.Fatalf("failed creating runtime: %v", err)
	}
	t.Cleanup(func() { _ = rt.Close(context.Background()) })
	manager, ok := jsevents.FromRuntime(rt)
	if !ok {
		t.Fatalf("jsevents manager not installed")
	}
	mod := &moduleRuntime{vm: rt.VM, eventEmitterManager: manager}
	return rt, mod
}

func TestEventEmitterSinkPublishesTextDelta(t *testing.T) {
	rt, mod := newJSEventEmitterTestRuntime(t)
	ret, err := rt.Owner.Call(context.Background(), "test.setupEmitter", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, runErr := vm.RunString(`
			const EventEmitter = require("events");
			globalThis.seen = [];
			globalThis.emitter = new EventEmitter();
			globalThis.emitter.on("event", ev => globalThis.seen.push(["event", ev.type]));
			globalThis.emitter.on("text-delta", ev => globalThis.seen.push(["text-delta", ev.delta, ev.text, ev.sequence]));
		`)
		if runErr != nil {
			return nil, runErr
		}
		sink, sinkErr := mod.newEventEmitterSinkFromValue(vm.Get("emitter"))
		if sinkErr != nil {
			return nil, sinkErr
		}
		return sink, nil
	})
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	sink, ok := ret.(*jsEventEmitterSink)
	if !ok || sink == nil {
		t.Fatalf("setup returned %T, want *jsEventEmitterSink", ret)
	}

	meta := events.EventMetadata{SessionID: "session-1", InferenceID: "inference-1", TurnID: "turn-1"}
	corr := events.Correlation{SessionID: "session-1", RunID: "run-1", TurnID: "turn-1"}
	if err := sink.PublishEvent(events.NewTextDeltaEvent(meta, corr, "he", "he", 1)); err != nil {
		t.Fatalf("PublishEvent failed: %v", err)
	}
	if err := rt.Owner.WaitIdle(context.Background()); err != nil {
		t.Fatalf("WaitIdle failed: %v", err)
	}
	got, err := rt.Owner.Call(context.Background(), "test.readSeen", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunString(`JSON.stringify(globalThis.seen)`)
	})
	if err != nil {
		t.Fatalf("read seen failed: %v", err)
	}
	if got.(goja.Value).String() != `[["event","text-delta"],["text-delta","he","he",1]]` {
		t.Fatalf("seen = %s", got.(goja.Value).String())
	}
}

func TestEventEmitterSinkPublishesToolResultAndMapsErrorName(t *testing.T) {
	rt, mod := newJSEventEmitterTestRuntime(t)
	var sink *jsEventEmitterSink
	_, err := rt.Owner.Call(context.Background(), "test.setupEmitter", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, runErr := vm.RunString(`
			const EventEmitter = require("events");
			globalThis.seen = [];
			globalThis.emitter = new EventEmitter();
			globalThis.emitter.on("tool-result-ready", ev => globalThis.seen.push(["tool", ev.toolResult.id, ev.toolResult.result]));
			globalThis.emitter.on("inference-error", ev => globalThis.seen.push(["error", ev.error, ev.message]));
			globalThis.emitter.on("error", () => globalThis.seen.push(["node-error"]));
		`)
		if runErr != nil {
			return nil, runErr
		}
		var sinkErr error
		sink, sinkErr = mod.newEventEmitterSinkFromValue(vm.Get("emitter"))
		return nil, sinkErr
	})
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	meta := events.EventMetadata{SessionID: "session-1"}
	corr := events.Correlation{SessionID: "session-1", ToolCallID: "tool-1"}
	if err := sink.PublishEvent(events.NewToolResultReadyEvent(meta, corr, "tool-1", "lookup", `{"ok":true}`, "success")); err != nil {
		t.Fatalf("tool PublishEvent failed: %v", err)
	}
	if err := sink.PublishEvent(events.NewErrorEvent(meta, errors.New("boom"))); err != nil {
		t.Fatalf("error PublishEvent failed: %v", err)
	}
	if err := rt.Owner.WaitIdle(context.Background()); err != nil {
		t.Fatalf("WaitIdle failed: %v", err)
	}
	got, err := rt.Owner.Call(context.Background(), "test.readSeen", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunString(`JSON.stringify(globalThis.seen)`)
	})
	if err != nil {
		t.Fatalf("read seen failed: %v", err)
	}
	want := `[["tool","tool-1","{\"ok\":true}"],["error","boom","boom"]]`
	if got.(goja.Value).String() != want {
		t.Fatalf("seen = %s, want %s", got.(goja.Value).String(), want)
	}
}

type geppettoEventTestModuleSpec struct {
	opts Options
}

func (s geppettoEventTestModuleSpec) ID() string { return "geppetto" }

func (s geppettoEventTestModuleSpec) RegisterRuntimeModule(ctx *gojengine.RuntimeModuleContext, reg *require.Registry) error {
	opts := s.opts
	opts.RuntimeOwner = ctx.Owner
	opts.EventEmitterManagerResolver = func() (*jsevents.Manager, bool) {
		value, ok := ctx.Value(jsevents.RuntimeValueKey)
		if !ok {
			return nil, false
		}
		manager, ok := value.(*jsevents.Manager)
		return manager, ok && manager != nil
	}
	Register(reg, opts)
	return nil
}

func newGeppettoEventRuntime(t *testing.T) *gojengine.Runtime {
	t.Helper()
	factory, err := gojengine.NewBuilder(
		gojengine.WithDataOnlyDefaultRegistryModules(true),
	).
		UseModuleMiddleware(gojengine.Pipeline()).
		WithRuntimeInitializers(jsevents.Install()).
		WithModules(geppettoEventTestModuleSpec{}).
		Build()
	if err != nil {
		t.Fatalf("failed creating runtime factory: %v", err)
	}
	rt, err := factory.NewRuntime(gojengine.WithStartupContext(context.Background()), gojengine.WithLifetimeContext(context.Background()))
	if err != nil {
		t.Fatalf("failed creating runtime: %v", err)
	}
	t.Cleanup(func() { _ = rt.Close(context.Background()) })
	return rt
}

type eventPublishingEngine struct {
	afterPublish time.Duration
}

var _ inferenceengine.Engine = (*eventPublishingEngine)(nil)

func (e *eventPublishingEngine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	meta := events.EventMetadata{SessionID: "session-async", InferenceID: "inference-async", TurnID: "turn-async"}
	corr := events.Correlation{SessionID: "session-async", RunID: "run-async", TurnID: "turn-async"}
	events.PublishEventToContext(ctx, events.NewTextDeltaEvent(meta, corr, "hello", "hello", 1))
	if e.afterPublish > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(e.afterPublish):
		}
	}
	out := t.Clone()
	out.Blocks = append(out.Blocks, turns.NewAssistantTextBlock("hello"))
	return out, nil
}

type eventThenBlockEngine struct {
	started chan struct{}
	release chan struct{}
}

var _ inferenceengine.Engine = (*eventThenBlockEngine)(nil)

func (e *eventThenBlockEngine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	meta := events.EventMetadata{SessionID: "session-lifecycle", InferenceID: "inference-lifecycle", TurnID: "turn-lifecycle"}
	corr := events.Correlation{SessionID: "session-lifecycle", RunID: "run-lifecycle", TurnID: "turn-lifecycle"}
	events.PublishEventToContext(ctx, events.NewTextDeltaEvent(meta, corr, "tick", "tick", 1))
	close(e.started)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-e.release:
	}
	out := t.Clone()
	out.Blocks = append(out.Blocks, turns.NewAssistantTextBlock("tick"))
	return out, nil
}

func managerRefCount(manager *jsevents.Manager) int {
	if manager == nil {
		return 0
	}
	return reflect.ValueOf(manager).Elem().FieldByName("refs").Len()
}

func TestAgentRunAsyncUsesRunScopedEventEmitterRefs(t *testing.T) {
	rt := newGeppettoEventRuntime(t)
	manager, ok := jsevents.FromRuntime(rt)
	if !ok {
		t.Fatalf("jsevents manager not installed")
	}
	if got := managerRefCount(manager); got != 0 {
		t.Fatalf("manager refs before run = %d, want 0", got)
	}
	eng := &eventThenBlockEngine{started: make(chan struct{}), release: make(chan struct{})}
	_, err := rt.Owner.Call(context.Background(), "test.startRunScopedEmitter", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if setErr := vm.Set("blockingEventEngine", eng); setErr != nil {
			return nil, setErr
		}
		_, runErr := vm.RunString(`
			const gp = require("geppetto");
			const EventEmitter = require("events");
			globalThis.seen = [];
			globalThis.resolved = false;
			const emitter = new EventEmitter();
			emitter.on("text-delta", ev => globalThis.seen.push(ev.delta));
			const agent = gp.agent().engine(globalThis.blockingEventEngine).events(emitter).build();
			const session = agent.session().id("run-scoped-emitter-test").build();
			globalThis.handle = session.next().user("block then return").runAsync();
			globalThis.handle.promise.then(result => {
				globalThis.resolved = true;
				globalThis.finalText = result.text();
			}, err => {
				globalThis.resolved = true;
				globalThis.finalError = String(err);
			});
		`)
		return nil, runErr
	})
	if err != nil {
		t.Fatalf("start script failed: %v", err)
	}
	select {
	case <-eng.started:
	case <-time.After(time.Second):
		t.Fatalf("engine did not publish/start")
	}
	if err := rt.Owner.WaitIdle(context.Background()); err != nil {
		t.Fatalf("WaitIdle failed: %v", err)
	}
	if got := managerRefCount(manager); got != 1 {
		t.Fatalf("manager refs during run = %d, want 1", got)
	}
	close(eng.release)
	if err := waitForOwnerCondition(rt, time.Second, `globalThis.resolved === true`); err != nil {
		t.Fatalf("promise did not resolve: %v", err)
	}
	if err := rt.Owner.WaitIdle(context.Background()); err != nil {
		t.Fatalf("WaitIdle after resolve failed: %v", err)
	}
	if got := managerRefCount(manager); got != 0 {
		t.Fatalf("manager refs after run = %d, want 0", got)
	}
	got, err := rt.Owner.Call(context.Background(), "test.readRunScopedEmitter", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunString(`JSON.stringify({seen: globalThis.seen, resolved: globalThis.resolved, finalText: globalThis.finalText, finalError: globalThis.finalError})`)
	})
	if err != nil {
		t.Fatalf("read result failed: %v", err)
	}
	want := `{"seen":["tick"],"resolved":true,"finalText":"tick"}`
	if got.(goja.Value).String() != want {
		t.Fatalf("state = %s, want %s", got.(goja.Value).String(), want)
	}
}

func TestAgentRunAsyncDeliversBuilderLevelEmitterBeforePromiseResolution(t *testing.T) {
	rt := newGeppettoEventRuntime(t)
	_, err := rt.Owner.Call(context.Background(), "test.runAsyncLiveEvents", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if setErr := vm.Set("fakeEngine", &eventPublishingEngine{afterPublish: 25 * time.Millisecond}); setErr != nil {
			return nil, setErr
		}
		_, runErr := vm.RunString(`
			const gp = require("geppetto");
			const EventEmitter = require("events");
			globalThis.seen = [];
			globalThis.resolved = false;
			const emitter = new EventEmitter();
			emitter.on("text-delta", ev => globalThis.seen.push(["delta", ev.delta, globalThis.resolved]));
			const agent = gp.agent().engine(globalThis.fakeEngine).events(emitter).build();
			const session = agent.session().id("builder-emitter-test").build();
			const handle = session.next().user("say hello").runAsync();
			if (typeof handle.cancel !== "function") throw new Error("missing cancel");
			if (typeof handle.close !== "function") throw new Error("missing close");
			if (typeof handle.on !== "undefined") throw new Error("handle.on should not be public");
			handle.promise.then(result => {
				globalThis.resolved = true;
				globalThis.finalText = result.text();
			}, err => {
				globalThis.resolved = true;
				globalThis.finalError = String(err);
			});
		`)
		return nil, runErr
	})
	if err != nil {
		t.Fatalf("runAsync script failed: %v", err)
	}
	if err := waitForOwnerCondition(rt, time.Second, `globalThis.resolved === true`); err != nil {
		t.Fatalf("promise did not resolve: %v", err)
	}
	got, err := rt.Owner.Call(context.Background(), "test.readRunAsyncLiveEvents", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunString(`JSON.stringify({seen: globalThis.seen, resolved: globalThis.resolved, finalText: globalThis.finalText, finalError: globalThis.finalError})`)
	})
	if err != nil {
		t.Fatalf("read result failed: %v", err)
	}
	want := `{"seen":[["delta","hello",false]],"resolved":true,"finalText":"hello"}`
	if got.(goja.Value).String() != want {
		t.Fatalf("state = %s, want %s", got.(goja.Value).String(), want)
	}
}

type blockingEngine struct {
	started  chan struct{}
	canceled chan struct{}
}

var _ inferenceengine.Engine = (*blockingEngine)(nil)

func (e *blockingEngine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	close(e.started)
	<-ctx.Done()
	close(e.canceled)
	return nil, ctx.Err()
}

func TestAgentRunAsyncToolLoopPreparesOnOwner(t *testing.T) {
	rt := newGeppettoEventRuntime(t)
	_, err := rt.Owner.Call(context.Background(), "test.runAsyncToolLoop", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if setErr := vm.Set("fakeEngine", &eventPublishingEngine{}); setErr != nil {
			return nil, setErr
		}
		_, runErr := vm.RunString(`
			const gp = require("geppetto");
			globalThis.resolved = false;
			const agent = gp.agent()
				.engine(globalThis.fakeEngine)
				.toolLoop({
					maxIterations: 1,
					toolChoice: "none",
					hooks: {
						beforeToolCall: () => {},
					},
				})
				.build();
			const session = agent.session().id("tool-loop-run-async-test").build();
			const handle = session.next().user("say hello").runAsync();
			handle.promise.then(result => {
				globalThis.resolved = true;
				globalThis.finalText = result.text();
			}, err => {
				globalThis.resolved = true;
				globalThis.finalError = String(err);
			});
		`)
		return nil, runErr
	})
	if err != nil {
		t.Fatalf("toolLoop runAsync script failed: %v", err)
	}
	if err := waitForOwnerCondition(rt, time.Second, `globalThis.resolved === true`); err != nil {
		t.Fatalf("promise did not resolve: %v", err)
	}
	got, err := rt.Owner.Call(context.Background(), "test.readToolLoopRunAsync", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunString(`JSON.stringify({finalText: globalThis.finalText, finalError: globalThis.finalError})`)
	})
	if err != nil {
		t.Fatalf("read result failed: %v", err)
	}
	want := `{"finalText":"hello"}`
	if got.(goja.Value).String() != want {
		t.Fatalf("state = %s, want %s", got.(goja.Value).String(), want)
	}
}

type failingEngine struct{}

var _ inferenceengine.Engine = (*failingEngine)(nil)

func (e *failingEngine) RunInference(context.Context, *turns.Turn) (*turns.Turn, error) {
	return nil, errors.New("engine boom")
}

func TestCloneRunOutputRejectsNilOutputTurn(t *testing.T) {
	out, err := cloneRunOutput(nil)
	if err == nil {
		t.Fatalf("expected nil output error, got output %#v", out)
	}
	if err.Error() != "agent run returned nil output turn" {
		t.Fatalf("nil output error = %q", err.Error())
	}
}

func TestAgentRunAsyncRejectsWithErrorObject(t *testing.T) {
	rt := newGeppettoEventRuntime(t)
	_, err := rt.Owner.Call(context.Background(), "test.errorObjectRunAsync", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if setErr := vm.Set("failingEngine", &failingEngine{}); setErr != nil {
			return nil, setErr
		}
		_, runErr := vm.RunString(`
			const gp = require("geppetto");
			const agent = gp.agent().engine(globalThis.failingEngine).build();
			const session = agent.session().id("error-object-test").build();
			globalThis.rejected = false;
			session.next().user("fail").runAsync().promise.then(
				() => { globalThis.rejected = true; globalThis.asyncResolved = true; },
				err => {
					globalThis.rejected = true;
					globalThis.asyncErrorString = String(err);
					globalThis.asyncErrorMessage = err && err.message;
					globalThis.asyncErrorIsError = err instanceof Error;
				}
			);
		`)
		return nil, runErr
	})
	if err != nil {
		t.Fatalf("error object async script failed: %v", err)
	}
	if err := waitForOwnerCondition(rt, time.Second, `globalThis.rejected === true`); err != nil {
		t.Fatalf("promise did not reject: %v", err)
	}
	got, err := rt.Owner.Call(context.Background(), "test.readErrorObjectRunAsync", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunString(`JSON.stringify({resolved: globalThis.asyncResolved, isError: globalThis.asyncErrorIsError, message: globalThis.asyncErrorMessage, string: globalThis.asyncErrorString})`)
	})
	if err != nil {
		t.Fatalf("read async error failed: %v", err)
	}
	want := `{"isError":true,"message":"engine boom","string":"GoError: engine boom"}`
	if got.(goja.Value).String() != want {
		t.Fatalf("async error state = %s, want %s", got.(goja.Value).String(), want)
	}
}

func TestRuntimeCloseCancelsRunAsyncExecutionContext(t *testing.T) {
	rt := newGeppettoEventRuntime(t)
	eng := &blockingEngine{started: make(chan struct{}), canceled: make(chan struct{})}
	_, err := rt.Owner.Call(context.Background(), "test.startRuntimeCloseRunAsync", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if setErr := vm.Set("blockingEngine", eng); setErr != nil {
			return nil, setErr
		}
		_, runErr := vm.RunString(`
			const gp = require("geppetto");
			const agent = gp.agent().engine(globalThis.blockingEngine).build();
			const session = agent.session().id("runtime-close-test").build();
			globalThis.handle = session.next().user("block until runtime close").runAsync();
		`)
		return nil, runErr
	})
	if err != nil {
		t.Fatalf("start script failed: %v", err)
	}
	select {
	case <-eng.started:
	case <-time.After(time.Second):
		t.Fatalf("engine did not start")
	}
	if err := rt.Close(context.Background()); err != nil {
		t.Fatalf("runtime close failed: %v", err)
	}
	select {
	case <-eng.canceled:
	case <-time.After(time.Second):
		t.Fatalf("engine context was not canceled by runtime close")
	}
}

func TestAgentRunAsyncCancelCancelsExecutionHandle(t *testing.T) {
	rt := newGeppettoEventRuntime(t)
	eng := &blockingEngine{started: make(chan struct{}), canceled: make(chan struct{})}
	_, err := rt.Owner.Call(context.Background(), "test.startCancelableRunAsync", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if setErr := vm.Set("blockingEngine", eng); setErr != nil {
			return nil, setErr
		}
		_, runErr := vm.RunString(`
			const gp = require("geppetto");
			const agent = gp.agent().engine(globalThis.blockingEngine).build();
			const session = agent.session().id("cancel-run-async-test").build();
			const handle = session.next().user("block").runAsync();
			globalThis.rejected = false;
			handle.promise.catch(err => { globalThis.rejected = String(err).length > 0; });
			globalThis.handle = handle;
		`)
		return nil, runErr
	})
	if err != nil {
		t.Fatalf("start script failed: %v", err)
	}
	select {
	case <-eng.started:
	case <-time.After(time.Second):
		t.Fatalf("engine did not start")
	}
	_, err = rt.Owner.Call(context.Background(), "test.cancelRunAsync", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunString(`globalThis.handle.cancel()`)
	})
	if err != nil {
		t.Fatalf("cancel script failed: %v", err)
	}
	select {
	case <-eng.canceled:
	case <-time.After(time.Second):
		t.Fatalf("engine context was not canceled")
	}
	if err := waitForOwnerCondition(rt, time.Second, `globalThis.rejected === true`); err != nil {
		t.Fatalf("promise was not rejected after cancel: %v", err)
	}
}

func waitForOwnerCondition(rt *gojengine.Runtime, timeout time.Duration, expr string) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ret, err := rt.Owner.Call(context.Background(), "test.waitForCondition", func(_ context.Context, vm *goja.Runtime) (any, error) {
			return vm.RunString(expr)
		})
		if err != nil {
			return err
		}
		if value, ok := ret.(goja.Value); ok && value.ToBoolean() {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return context.DeadlineExceeded
}
