# Streaming LLM Inference Results – JavaScript API Design

*Date: 2025-06-16*

---

## 1. Motivation
Large-Language-Model (LLM) inference can produce **partial** results (tokens, deltas, tool calls …) long before the final answer is known.  
On the Go side we already emit those over Watermill as `events.*`.  
JavaScript users, however, only see a *single* final value through the current bindings (`startAsync`, `startBlocking`, `startWithCallbacks`).  
To unlock real-time UX (chat UIs, progress bars, live tool calling) we need first-class streaming support in `@/js`.

## 2. Goals
1. Preserve current API (non-streaming) – no breaking changes.
2. Provide **ergonomic** streaming primitives that feel native in modern JS/TS (*AsyncIterator*, *EventEmitter*, *Observable*-like callbacks).
3. Map one-to-one to existing Go `events.EventType` values so that **all** information is surfaced.
4. Make cancellation explicit.
5. Work in both common JS runtimes:
   * Embedded *goja* (current CLI / scripting use-case)
   * Node.js (future-proofing)

## 3. High-Level Shape
The proposal introduces a new *streaming entry-point* on each step wrapper:
```js
/**
 * Start the step and receive a stream of events.
 * Returns an iterable + helper methods.
 */
const stream = chatStep.startStream(conversation, options);
```
`stream` implements **all three** access patterns so users can pick what they like:
1. **Async iteration** (modern, `for await`):
   ```js
   for await (const e of stream) {
     if (e.type === 'partial') appendToUI(e.delta);
     if (e.type === 'final') done(e.text);
   }
   ```
2. **EventEmitter** (Node-style):
   ```js
   stream
     .on('partial', delta => appendToUI(delta))
     .on('tool-call', tc => showToolCall(tc))
     .on('error', err => console.error(err))
     .on('final', res => done(res));
   ```
3. **Callback object** (superset of current API):
   ```js
   chatStep.startStream(
     conv,
     {
       onPartial: delta => append(delta),
       onToolCall: tc => handleTool(tc),
       onFinal: res => done(res),
       onError: err => show(err),
       onCancel: () => console.log('cancelled')
     }
   );
   ```

`startAsync`/`startBlocking`/`startWithCallbacks` stay unchanged and internally just ***collect*** the stream until `final` arrives.

### Cancellation
```js
const stream = chatStep.startStream(conv);
setTimeout(() => stream.cancel(), 10_000); // user stopped typing -> abort
```
Calling `.cancel()` triggers an `interrupt` event on the Go side and the JS stream finishes.

## 4. Event Surface
A thin TS declaration file (`geppetto.d.ts`) captures the shape so editors get autocompletion:
```ts
export type StreamEvent =
  | { type: 'start';    meta: Meta; step: StepMeta }
  | { type: 'partial';  delta: string; completion: string; meta: Meta; step: StepMeta }
  | { type: 'tool-call';   toolCall: { id: string; name: string; input: string };  meta: Meta; step: StepMeta }
  | { type: 'tool-result'; toolResult: { id: string; result: string };            meta: Meta; step: StepMeta }
  | { type: 'final';    text: string;   meta: Meta; step: StepMeta }
  | { type: 'interrupt'; text: string;  meta: Meta; step: StepMeta }
  | { type: 'error';    error: string;  meta: Meta; step: StepMeta };

export interface Stream<T extends StreamEvent = StreamEvent> extends AsyncIterable<T> {
  on<E extends T["type"]>(event: E, handler: (ev: Extract<T, { type: E }>) => void): this;
  cancel(): void;
}
```

*All* Go fields are preserved; additional sugar (e.g. `delta`, `text`) is added where convenient.

## 5. Usage Examples
### Chat completion with live UI
```js
const stream = chatStep.startStream(conv);
stream
  .on('partial', e => updateUI(e.delta))
  .on('tool-call', e => runTool(e.toolCall))
  .on('final',   e => showFinal(e.text))
  .on('error',   e => alert(e.error));
```

### Collect full text via async iteration
```js
let result = '';
for await (const e of chatStep.startStream(conv)) {
  if (e.type === 'partial') result += e.delta;
  if (e.type === 'final')  console.log('LLM:', result + e.text);
}
```

## 6. Mapping to Go Implementation
* **Emitter → channel**: In Go the step will push `events.Event` into a channel.  
* **JSEventStream**: A small wrapper converts the channel into callbacks & async iteration inside the goja event-loop.
* **Backpressure**: AsyncIterator naturally applies backpressure; for EventEmitter we keep an internal buffer with a max size (configurable).
* **Cancellation**: `.cancel()` closes the Go context, sends an `EventInterrupt`, then closes the channel.

A sketch (Go):
```go
func (w *JSStepWrapper[In, Out]) makeStartStream(...) func(call goja.FunctionCall) goja.Value {
  return func(call goja.FunctionCall) goja.Value {
    // 1. create ctx, start step → returns (<-chan events.Event, cancel)
    // 2. wrap chan into JS object implementing AsyncIterator & EventEmitter
    // 3. expose .cancel()
  }
}
```

## 7. Migration & Compatibility
* **Old code** keeps working.
* Internally, `startAsync` now delegates to `startStream` and gathers events until `final`/`error`.
* Version bump to `0.x+1` with changelog entry.

## 8. Open Questions
1. Should we surface token probabilities? (Not currently in `events.Event`.)
2. How to represent nested/parallel tool calls? (Future multi-call support.)
3. Provide an RxJS Observable helper for reactive enthusiasts?

Implementation Plan for JavaScript Streaming Support
====================================================

Below is a concrete, file-level roadmap with the structs, methods, and helper files that need to be created or adapted.  Pseudocode sketches are provided to illustrate the logic paths but **not** meant to compile as-is.

--------------------------------------------------------------------
1. `geppetto/pkg/js/steps-js.go`
--------------------------------------------------------------------
### 1.1  Add `makeStartStream` generator to `JSStepWrapper`
```go
func (w *JSStepWrapper[T,U]) makeStartStream(
    inputConverter func(goja.Value)T,
    outputConverter func(U)goja.Value,
) func(goja.FunctionCall) goja.Value {

    return func(call goja.FunctionCall) goja.Value {
        input := inputConverter(call.Arguments[0])

        ctx, cancel := context.WithCancel(context.Background())
        // ⇨ Step returns: (<-chan events.Event, func()) via new interface (see 2.1)
        evCh, stepCancel, err := w.step.Stream(ctx, input)
        if err != nil { … }

        stream := newJSEventStream(w.runtime, w.loop, evCh, func() {
            stepCancel(); cancel()
        })
        return w.runtime.ToValue(stream)
    }
}
```

### 1.2  Wire it into `CreateStepObject`
```go
stepObj.Set("startStream", wrapper.makeStartStream(inputConv, outputConv))
```

### 1.3  Update `startAsync`/`startBlocking`/`startWithCallbacks`
Each collects the new stream instead of duplicating logic:
```go
stream := wrapper.makeStartStream(…)(callWithInputOnly)
for ev := range stream.Iter() { 
    switch ev.Type() { case events.EventTypeFinal: … } 
}
```

--------------------------------------------------------------------
2. Core step interfaces
--------------------------------------------------------------------
### 2.1  New Go interface for streaming
`geppetto/pkg/steps/stream.go`
```go
type StreamableStep[I,O any] interface {
    // previous Start remains
    Step[I,O]
    // NEW
    Stream(ctx context.Context, in I) (<-chan events.Event, CancelFunc, error)
}
```
*All existing steps are gradually upgraded; wrapper will type-assert and fall back to `Start` + buffer when `Stream` is missing.*

--------------------------------------------------------------------
3. JS-side stream helper
--------------------------------------------------------------------
### 3.1  New file `geppetto/pkg/js/event_stream_js.go`
```go
type jsEventStream struct {
    runtime *goja.Runtime
    loop    *eventloop.EventLoop
    ch      <-chan events.Event
    cancel  func()
    emitter *event.Emitter   // small internal Node-style emitter
}

/* Methods exposed to JS:
   – [Symbol.asyncIterator]()    // for-await syntax
   – on(event, handler)          // EventEmitter chainable
   – cancel()                    // abort + propagate interrupt
*/
```

Pseudocode for async iterator:
```go
func (s *jsEventStream) asyncIterator(call goja.FunctionCall) goja.Value {
    return s.runtime.ToValue(map[string]any{
        "next": func(goja.FunctionCall) goja.Value {
             promise, resolve, reject := s.runtime.NewPromise()
             s.loop.RunOnLoop(func(*goja.Runtime) { go func() {
                 ev, ok := <-s.ch
                 if !ok { resolve(valueDone) ; return }
                 resolve(valueOf(ev))
             }() })
             return promise
        },
    })
}
```

### 3.2  Marshal `events.Event` → plain JS object
```go
func toJS(runtime *goja.Runtime, e events.Event) goja.Value {
    switch v := e.(type) {
    case *events.EventPartialCompletion:
        return runtime.ToValue(map[string]any{
            "type": "partial", "delta": v.Delta, "completion": v.Completion,
            "meta": v.Metadata_, "step": v.Step_,
        })
    // …handle all other variants…
    }
}
```

--------------------------------------------------------------------
4. Cancellation / Interrupt plumbing
--------------------------------------------------------------------
* When `.cancel()` is invoked in JS → call internal `cancel()` →  
  – Close context  
  – If step provides `Interrupt()` utility send `EventInterrupt` immediately.  

--------------------------------------------------------------------
5. TypeScript typings
--------------------------------------------------------------------
`geppetto/ts/geppetto.d.ts` (generated or handwritten)
```ts
export interface Stream { … }          // as in design doc
export type StreamEvent = …            // exhaustive union
```

--------------------------------------------------------------------
6. Tests
--------------------------------------------------------------------
### 6.1  Go unit tests (`pkg/js/steps-js_test.go`)
* spin up a fake `StreamableStep` producing synthetic events  
* assert that asyncIterator delivers them in order

### 6.2  JS tests (`/js/tests/stream.spec.js`)
* Use Mocha in goja env; verify EventEmitter and cancellation

--------------------------------------------------------------------
7. Refactor existing steps
--------------------------------------------------------------------
* `openai/chat_step.go` (and friends) implement `Stream` by forwarding OpenAI stream.
* Where not yet possible → wrap `Start` output into synthetic `{type:"final"}` event.
