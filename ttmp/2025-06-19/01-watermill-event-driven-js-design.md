### Updated design — handler auto-unregister on step completion

Below I’ve folded one small but crucial change into the previous spec: **each per-step handler is stopped (and therefore unsubscribed) the moment the step’s `finished` channel closes.**
Everything else is unchanged, so the RuntimeEngine still owns a single Goja VM, a single in-process GoChannel bus, and one long-lived Watermill router.

---

## 1 RuntimeEngine (recap)

```go
type RuntimeEngine struct {
    ctx    context.Context
    cancel context.CancelFunc

    VM      *goja.Runtime
    Loop    *eventloop.EventLoop
    Bus     *gochannel.GoChannel       // 1 shared pub/sub
    Router  *message.Router           // 1 shared router

    mu      sync.RWMutex
    running map[string]*runner        // stepID → runner
}

type runner struct {
    stepID   string
    onEvent  goja.Callable            // JS callback
    finished chan struct{}            // closed when step ends
}
```

---

## 2 Creating and starting the engine

```go
func NewRuntimeEngine() *RuntimeEngine {
    ctx, cancel := context.WithCancel(context.Background())

    eng := &RuntimeEngine{
        ctx:     ctx,
        cancel:  cancel,
        VM:      goja.New(),
        Loop:    eventloop.NewEventLoop(),
        Bus:     gochannel.NewGoChannel(
                     gochannel.Config{BlockPublishUntilSubscriberAck: true},
                     watermill.NopLogger{}),
        running: map[string]*runner{},
    }

    eng.Router, _ = message.NewRouter(message.RouterConfig{
        CloseTimeout: 15 * time.Second,
    }, watermill.NopLogger{})

    go eng.Router.Run(ctx)  // start once
    <-eng.Router.Running()  // wait until handlers live
    go eng.Loop.Start()     // JS event loop
    return eng
}
```

---

## 3 Running a step **and** auto-registering / unregistering its handler

```go
func (e *RuntimeEngine) RunStep[T, U any](
    step steps.Step[T, U],
    in T,
    onEvent goja.Callable,
) string {
    stepID := uuid.NewString()

    // record JS callback
    e.mu.Lock()
    r := &runner{stepID: stepID, onEvent: onEvent, finished: make(chan struct{})}
    e.running[stepID] = r
    e.mu.Unlock()

    // 1️⃣  Add a per-step handler on the shared router
    h := e.Router.AddNoPublisherHandler(
        "into-vm-"+stepID,          // unique name
        "step."+stepID,             // topic filter
        e.Bus,                      // subscriber
        func(msg *message.Message) error {
            var ev steps.Event
            _ = json.Unmarshal(msg.Payload, &ev)
            e.Loop.RunOnLoop(func(vm *goja.Runtime) {
                r.onEvent(goja.Undefined(), vm.ToValue(ev))
            })
            return nil
        })

    _ = e.Router.RunHandlers(e.ctx) // hot-plug immediately

    // 2️⃣  Launch the step
    go func() {
        _ = steps.ExecuteWithEvents(e.ctx, stepID, step, in, e.Bus)
        close(r.finished)           // signal completion
    }()

    // 3️⃣  Auto-detach the handler when done
    go func() {
        <-r.finished
        h.Stop()                    // unsubscribes the topic
        e.mu.Lock()
        delete(e.running, stepID)   // tidy map
        e.mu.Unlock()
    }()

    return stepID
}
```

**What changed?**

* We capture the `handler` returned by `AddNoPublisherHandler`.
* A background goroutine waits on `r.finished`; when it fires, we call
  `h.Stop()`, which **unregisters** the handler and closes its subscriber.
* No extra routers and no accumulation of unused subscriptions.

---

## 4 Watching events from Go (unchanged API, now leak-safe)

```go
func (e *RuntimeEngine) WatchStepEvents(stepID string) (*StepWatcher, error) {
    e.mu.RLock()
    r, ok := e.running[stepID]
    e.mu.RUnlock()
    if !ok {
        return nil, fmt.Errorf("unknown step %s", stepID)
    }

    ctx, cancel := context.WithCancel(e.ctx)
    sub, err := e.Bus.Subscribe(ctx, "step."+stepID)
    if err != nil { cancel(); return nil, err }

    out := make(chan steps.Event, 16)
    go pump(sub, out, ctx)

    // close watcher when step ends
    go func() {
        <-r.finished
        cancel()
    }()

    return &StepWatcher{Events: out, close: cancel}, nil
}
```

Because the per-step router handler is now removed automatically, this watcher’s
subscription is the only remaining open listener once the step is finished; it
closes immediately when the caller drains `out` or invokes `close()`.

---

### TL;DR

* **One** Watermill router for the process.
* **One handler per running step**, added at launch.
* Handler is **stopped and unsubscribed** in the background as soon as the step
  finishes, keeping the router lean and brokers free of orphan consumers.
