---
title: "Code evidence"
doc_type: reference
topics:
  - geppetto
  - goja
  - js-bindings
  - streaming
  - events
status: active
intent: evidence
owners:
  - manuel
created: 2026-06-01
updated: 2026-06-01
---

# Code Evidence

Generated: 2026-06-01T19:17:08-04:00
Repository: /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto
go-go-goja module: /home/manuel/go/pkg/mod/github.com/go-go-golems/go-go-goja@v0.7.0


## Geppetto JS module exports

```text
   140		rt := newRuntime(vm, m.opts)
   141		exports := moduleObj.Get("exports").(*goja.Object)
   142		rt.installExports(exports)
   143	}
   144	
   145	func (m *moduleRuntime) installExports(exports *goja.Object) {
   146		m.mustSet(exports, "version", "0.1.0")
   147		m.installConsts(exports)
   148	
   149		inferenceProfilesObj := m.vm.NewObject()
   150		m.mustSet(inferenceProfilesObj, "load", m.inferenceProfilesLoad)
   151		m.mustSet(inferenceProfilesObj, "resolve", m.inferenceProfilesResolve)
   152		m.mustSet(inferenceProfilesObj, "default", m.inferenceProfilesDefault)
   153		m.mustSet(exports, "inferenceProfiles", inferenceProfilesObj)
   154		m.mustSet(exports, "engine", m.engineBuilder)
   155		m.mustSet(exports, "agent", m.agentBuilder)
   156		m.mustSet(exports, "turn", m.turnBuilder)
   157		m.mustSet(exports, "tool", m.toolBuilder)
   158		m.mustSet(exports, "toolRegistry", m.toolRegistryBuilder)
   159		m.installSchemaNamespace(exports)
   160	
   161		eventsObj := m.vm.NewObject()
   162		m.mustSet(eventsObj, "collector", m.eventsCollector)
   163		m.mustSet(exports, "events", eventsObj)
   164	}
   165	
   166	func (m *moduleRuntime) mustSet(o *goja.Object, key string, v any) {
   167		if err := o.Set(key, v); err != nil {
   168			panic(m.vm.NewGoError(fmt.Errorf("set %s: %w", key, err)))
   169		}
   170	}
   171	
   172	func (m *moduleRuntime) attachRef(o *goja.Object, ref any) {
   173		// Set the value first so goja stores the Go pointer as-is (m.vm.ToValue
   174		// would wrap struct pointers in a proxy whose Export() returns a map).
   175		// Then redefine the property to make it non-enumerable/non-writable/non-configurable.
   176		_ = o.Set(hiddenRefKey, ref)
   177		_ = o.DefineDataProperty(hiddenRefKey, o.Get(hiddenRefKey),
   178			goja.FLAG_FALSE, // writable
   179			goja.FLAG_FALSE, // enumerable
   180			goja.FLAG_FALSE, // configurable
```

## Geppetto JS agent stream/run path

```text
   220			if err != nil {
   221				panic(m.vm.NewGoError(err))
   222			}
   223			opts, err := m.parseAgentRunOptions(ref.runDefaults, call.Arguments, 1)
   224			if err != nil {
   225				panic(m.vm.NewGoError(err))
   226			}
   227			result, err := ref.runSync(turn.turn, opts)
   228			if err != nil {
   229				panic(m.vm.NewGoError(err))
   230			}
   231			return m.newRunResultObject(result)
   232		})
   233		m.mustSet(o, "stream", func(call goja.FunctionCall) goja.Value {
   234			if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
   235				panic(m.vm.NewTypeError("agent.stream requires a Go-owned Turn wrapper"))
   236			}
   237			turn, err := m.requireTurnRef(call.Arguments[0])
   238			if err != nil {
   239				panic(m.vm.NewGoError(err))
   240			}
   241			opts, err := m.parseAgentRunOptions(ref.runDefaults, call.Arguments, 1)
   242			if err != nil {
   243				panic(m.vm.NewGoError(err))
   244			}
   245			return ref.start(turn.turn, opts)
   246		})
   247		return o
   248	}
   249	
   250	func (m *moduleRuntime) parseAgentRunOptions(defaults runOptions, args []goja.Value, idx int) (runOptions, error) {
   251		extra, err := m.parseRunOptions(args, idx)
   252		if err != nil {
   253			return runOptions{}, err
   254		}
   255		out := defaults
   256		if extra.timeoutMs != 0 {
   257			out.timeoutMs = extra.timeoutMs
   258		}
   259		if len(extra.tags) > 0 {
   260			out.tags = mergeRuntimeMetadata(out.tags, extra.tags)
   261		}
   262		return out, nil
   263	}
   264	
   265	func (a *agentRef) buildSession() (*sessionRef, error) {
   266		b := &builderRef{
   267			api:              a.api,
   268			base:             a.base.Engine,
   269			middlewares:      append([]middleware.Middleware(nil), a.middlewares...),
   270			registry:         a.registry,
   271			runtimeToolNames: append([]string(nil), a.runtimeToolNames...),
   272			runtimeMetadata:  cloneJSONMap(a.runtimeMetadata),
   273			eventSinks:       append([]events.EventSink(nil), a.eventSinks...),
   274			snapshotHook:     a.api.defaultSnapshotHook,
   275			persister:        a.api.defaultPersister,
   276		}
   277		if len(a.loopOptions) > 0 {
   278			a.api.applyToolLoopSettings(b, a.loopOptions, a.api.vm.ToValue(a.loopOptions))
   279		}
   280		return b.buildSession()
   281	}
   282	
   283	func (a *agentRef) runSync(input *turns.Turn, opts runOptions) (*runResultRef, error) {
   284		if input == nil {
   285			return nil, fmt.Errorf("agent.run requires turn")
   286		}
   287		sr, err := a.buildSession()
   288		if err != nil {
   289			return nil, err
   290		}
   291		inputSnapshot := input.Clone()
   292		seed := input.Clone()
   293		stampTurnRuntimeMetadata(seed, sr.runtimeMetadata)
   294		effective := seed.Clone()
   295		sr.session.Append(seed)
   296		ctx, cancel, err := sr.buildRunContext(opts)
   297		if err != nil {
   298			return nil, err
   299		}
   300		defer cancel()
   301		handle, err := sr.session.StartInference(ctx)
   302		if err != nil {
   303			return nil, err
   304		}
   305		out, err := handle.Wait()
   306		if err != nil {
   307			return nil, err
   308		}
   309		return &runResultRef{api: a.api, inputTurn: inputSnapshot, effectiveTurn: effective, outputTurn: out.Clone()}, nil
   310	}
   311	
   312	func (a *agentRef) start(input *turns.Turn, opts runOptions) goja.Value {
   313		if _, err := a.api.requireBridge("agent.stream"); err != nil {
   314			panic(a.api.vm.NewTypeError(err.Error()))
   315		}
   316		promise, resolve, reject := a.api.vm.NewPromise()
   317		collector := newJSEventCollector(a.api)
   318		handleObj := a.api.vm.NewObject()
   319	
   320		var cancelMu sync.RWMutex
   321		var cancelFn context.CancelFunc
   322		setCancel := func(fn context.CancelFunc) { cancelMu.Lock(); cancelFn = fn; cancelMu.Unlock() }
   323		getCancel := func() context.CancelFunc { cancelMu.RLock(); defer cancelMu.RUnlock(); return cancelFn }
   324	
   325		a.api.mustSet(handleObj, "promise", promise)
   326		a.api.mustSet(handleObj, "cancel", func(goja.FunctionCall) goja.Value {
   327			if fn := getCancel(); fn != nil {
   328				fn()
   329			}
   330			return goja.Undefined()
   331		})
   332		a.api.mustSet(handleObj, "on", func(call goja.FunctionCall) goja.Value {
   333			if len(call.Arguments) < 2 {
   334				panic(a.api.vm.NewTypeError("on(eventType, callback) requires 2 arguments"))
   335			}
   336			eventType := call.Arguments[0].String()
   337			cb, ok := goja.AssertFunction(call.Arguments[1])
   338			if !ok {
   339				panic(a.api.vm.NewTypeError("on() callback must be a function"))
   340			}
   341			collector.subscribe(eventType, cb)
   342			return handleObj
   343		})
   344	
   345		go func() {
   346			result, err := a.runSync(input, opts)
   347			postErr := a.api.postOnOwner(context.Background(), "agent.stream.settle", func(context.Context) {
   348				defer collector.close()
   349				if err != nil {
   350					_ = reject(a.api.vm.ToValue(err.Error()))
   351					return
   352				}
   353				_ = resolve(a.api.newRunResultObject(result))
   354			})
   355			if postErr != nil {
   356				collector.close()
   357				a.api.logger.Error().Err(postErr).Msg("agent.stream: failed to settle promise on owner thread")
   358			}
   359		}()
   360		setCancel(func() {})
   361		return handleObj
   362	}
   363	
   364	func (m *moduleRuntime) newRunResultObject(ref *runResultRef) *goja.Object {
   365		if ref == nil {
   366			ref = &runResultRef{api: m}
   367		}
   368		ref.api = m
   369		o := m.vm.NewObject()
   370		m.attachRef(o, ref)
   371		m.mustSet(o, "inputTurn", func(goja.FunctionCall) goja.Value {
   372			return m.newTurnObject(&turnRef{api: m, turn: ref.inputTurn.Clone()})
   373		})
   374		m.mustSet(o, "effectiveTurn", func(goja.FunctionCall) goja.Value {
   375			return m.newTurnObject(&turnRef{api: m, turn: ref.effectiveTurn.Clone()})
   376		})
   377		m.mustSet(o, "outputTurn", func(goja.FunctionCall) goja.Value {
   378			return m.newTurnObject(&turnRef{api: m, turn: ref.outputTurn.Clone()})
   379		})
   380		m.mustSet(o, "text", func(goja.FunctionCall) goja.Value { return m.vm.ToValue(turnText(ref.outputTurn)) })
```

## Existing JS event collector

```text
     1	package geppetto
     2	
     3	import (
     4		"context"
     5		"strings"
     6		"time"
     7	
     8		"github.com/dop251/goja"
     9		"github.com/go-go-golems/geppetto/pkg/events"
    10	)
    11	
    12	func newJSEventCollector(api *moduleRuntime) *jsEventCollector {
    13		return &jsEventCollector{
    14			api:       api,
    15			listeners: map[string][]goja.Callable{},
    16		}
    17	}
    18	
    19	func (m *moduleRuntime) eventsCollector(goja.FunctionCall) goja.Value {
    20		if _, err := m.requireBridge("events.collector"); err != nil {
    21			panic(m.vm.NewTypeError(err.Error()))
    22		}
    23		collector := newJSEventCollector(m)
    24		obj := m.vm.NewObject()
    25		m.attachRef(obj, collector)
    26	
    27		m.mustSet(obj, "on", func(call goja.FunctionCall) goja.Value {
    28			if len(call.Arguments) < 2 {
    29				panic(m.vm.NewTypeError("collector.on(eventType, callback) requires 2 arguments"))
    30			}
    31			eventType := call.Arguments[0].String()
    32			cb, ok := goja.AssertFunction(call.Arguments[1])
    33			if !ok {
    34				panic(m.vm.NewTypeError("collector.on() callback must be a function"))
    35			}
    36			collector.subscribe(eventType, cb)
    37			return obj
    38		})
    39	
    40		m.mustSet(obj, "clear", func(call goja.FunctionCall) goja.Value {
    41			if len(call.Arguments) > 0 && call.Arguments[0] != nil && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
    42				collector.clear(call.Arguments[0].String())
    43			} else {
    44				collector.clear("*")
    45			}
    46			return obj
    47		})
    48	
    49		m.mustSet(obj, "close", func(goja.FunctionCall) goja.Value {
    50			collector.close()
    51			return goja.Undefined()
    52		})
    53	
    54		return obj
    55	}
    56	
    57	var _ events.EventSink = (*jsEventCollector)(nil)
    58	
    59	func (c *jsEventCollector) subscribe(eventType string, fn goja.Callable) {
    60		if c == nil || fn == nil {
    61			return
    62		}
    63		eventType = strings.TrimSpace(eventType)
    64		if eventType == "" {
    65			eventType = "*"
    66		}
    67		c.mu.Lock()
    68		defer c.mu.Unlock()
    69		if c.closed {
    70			return
    71		}
    72		c.listeners[eventType] = append(c.listeners[eventType], fn)
    73	}
    74	
    75	func (c *jsEventCollector) close() {
    76		if c == nil {
    77			return
    78		}
    79		c.mu.Lock()
    80		c.closed = true
    81		c.listeners = nil
    82		c.mu.Unlock()
    83	}
    84	
    85	func (c *jsEventCollector) clear(eventType string) {
    86		if c == nil {
    87			return
    88		}
    89		eventType = strings.TrimSpace(eventType)
    90		if eventType == "" || eventType == "*" {
    91			c.mu.Lock()
    92			defer c.mu.Unlock()
    93			if c.closed {
    94				return
    95			}
    96			c.listeners = map[string][]goja.Callable{}
    97			return
    98		}
    99		c.mu.Lock()
   100		defer c.mu.Unlock()
   101		if c.closed {
   102			return
   103		}
   104		delete(c.listeners, eventType)
   105	}
   106	
   107	func (c *jsEventCollector) PublishEvent(ev events.Event) error {
   108		if c == nil || ev == nil {
   109			return nil
   110		}
   111		c.mu.RLock()
   112		if c.closed {
   113			c.mu.RUnlock()
   114			return nil
   115		}
   116		eventType := string(ev.Type())
   117		callbacks := make([]goja.Callable, 0, len(c.listeners[eventType])+len(c.listeners["*"]))
   118		callbacks = append(callbacks, c.listeners[eventType]...)
   119		callbacks = append(callbacks, c.listeners["*"]...)
   120		c.mu.RUnlock()
   121		if len(callbacks) == 0 || c.api == nil {
   122			return nil
   123		}
   124		if _, err := c.api.requireBridge("event collector publish"); err != nil {
   125			c.api.logger.Warn().Err(err).Msg("event collector publish skipped")
   126			return nil
   127		}
   128	
   129		payload := c.encodeEventPayload(ev)
   130		_, err := c.api.callOnOwner(context.Background(), "eventCollector.publish", func(context.Context) (any, error) {
   131			jsPayload := c.api.toJSValue(payload)
   132			for _, cb := range callbacks {
   133				_, _ = cb(goja.Undefined(), jsPayload)
   134			}
   135			return nil, nil
   136		})
   137		if err != nil {
   138			c.api.logger.Warn().Err(err).Msg("event collector publish failed")
   139		}
   140		return nil
   141	}
   142	
   143	func (c *jsEventCollector) encodeEventPayload(ev events.Event) map[string]any {
   144		meta := ev.Metadata()
   145		payload := map[string]any{
   146			"type":        string(ev.Type()),
   147			"timestampMs": time.Now().UnixMilli(),
   148		}
   149		if meta.SessionID != "" {
   150			payload["sessionId"] = meta.SessionID
   151		}
   152		if meta.InferenceID != "" {
   153			payload["inferenceId"] = meta.InferenceID
   154		}
   155		if meta.TurnID != "" {
   156			payload["turnId"] = meta.TurnID
   157		}
   158		if len(meta.Extra) > 0 {
   159			payload["metaExtra"] = cloneJSONValue(meta.Extra)
   160		}
   161	
   162		switch e := ev.(type) {
   163		case events.CorrelatedEvent:
   164			payload["correlation"] = cloneJSONValue(e.Correlation())
   165		}
   166	
   167		switch e := ev.(type) {
   168		case *events.EventTextDelta:
   169			payload["delta"] = e.Delta
   170			payload["text"] = e.Text
   171			payload["sequence"] = e.Sequence
   172		case *events.EventTextSegmentFinished:
   173			payload["text"] = e.Text
   174			payload["finishReason"] = e.FinishReason
   175		case *events.EventReasoningDelta:
   176			payload["delta"] = e.Delta
   177			payload["text"] = e.Text
   178			payload["sequence"] = e.Sequence
   179			if e.Source != "" {
   180				payload["source"] = e.Source
   181			}
   182		case *events.EventReasoningSegmentFinished:
   183			payload["text"] = e.Text
   184			payload["finishReason"] = e.FinishReason
   185			if e.Source != "" {
   186				payload["source"] = e.Source
   187			}
   188		case *events.EventToolCallStarted:
   189			payload["toolCall"] = map[string]any{
   190				"id":   e.ToolCallID,
   191				"name": e.ToolName,
   192			}
   193		case *events.EventToolCallArgumentsDelta:
   194			payload["toolCall"] = map[string]any{
   195				"id":        e.ToolCallID,
   196				"delta":     e.Delta,
   197				"arguments": e.Arguments,
   198				"sequence":  e.Sequence,
   199			}
   200		case *events.EventToolCallRequested:
   201			payload["toolCall"] = map[string]any{
   202				"id":    e.ToolCallID,
   203				"name":  e.ToolName,
   204				"input": e.Input,
   205			}
   206		case *events.EventToolExecutionStarted:
   207			payload["toolCall"] = map[string]any{
   208				"id":    e.ToolCallID,
   209				"name":  e.ToolName,
   210				"input": e.Input,
   211			}
   212		case *events.EventToolResultReady:
   213			payload["toolResult"] = map[string]any{
   214				"id":     e.ToolCallID,
   215				"name":   e.ToolName,
   216				"result": e.Result,
   217				"status": e.Status,
   218			}
   219		case *events.EventToolCallFinished:
   220			payload["toolCall"] = map[string]any{
   221				"id":     e.ToolCallID,
   222				"name":   e.ToolName,
   223				"status": e.Status,
   224			}
   225		case *events.EventError:
   226			payload["error"] = e.ErrorString
   227		case *events.EventInterrupt:
   228			payload["text"] = e.Text
   229		}
   230		if raw := ev.Payload(); len(raw) > 0 {
   231			payload["rawPayload"] = string(raw)
   232		}
   233		return payload
   234	}
```

## Owner bridge helpers

```text
     1	package geppetto
     2	
     3	import (
     4		"context"
     5		"fmt"
     6	
     7		"github.com/dop251/goja"
     8		"github.com/go-go-golems/geppetto/pkg/js/runtimebridge"
     9	)
    10	
    11	func (m *moduleRuntime) requireBridge(op string) (*runtimebridge.Bridge, error) {
    12		if m == nil || m.bridge == nil {
    13			return nil, fmt.Errorf("%s requires module options runner to be configured", op)
    14		}
    15		return m.bridge, nil
    16	}
    17	
    18	func (m *moduleRuntime) callOnOwner(ctx context.Context, op string, fn func(context.Context) (any, error)) (any, error) {
    19		if fn == nil {
    20			return nil, fmt.Errorf("%s: owner callback is nil", op)
    21		}
    22		bridge, err := m.requireBridge(op)
    23		if err != nil {
    24			return nil, err
    25		}
    26		return bridge.Call(ctx, op, func(callCtx context.Context, _ *goja.Runtime) (any, error) {
    27			return fn(callCtx)
    28		})
    29	}
    30	
    31	func (m *moduleRuntime) postOnOwner(ctx context.Context, op string, fn func(context.Context)) error {
    32		if fn == nil {
    33			return fmt.Errorf("%s: owner callback is nil", op)
    34		}
    35		bridge, err := m.requireBridge(op)
    36		if err != nil {
    37			return err
    38		}
    39		return bridge.Post(ctx, op, func(callCtx context.Context, _ *goja.Runtime) {
    40			fn(callCtx)
    41		})
    42	}
```

## Session StartInference lifecycle

```text
   185		s.Turns = append(s.Turns, t)
   186		s.mu.Unlock()
   187	}
   188	
   189	// StartInference starts an inference asynchronously and returns an ExecutionHandle.
   190	//
   191	// The builder is invoked to produce a blocking runner (RunInference). The runner is
   192	// executed in a goroutine against the latest appended Turn, which is intentionally
   193	// mutated in-place (middlewares may modify it).
   194	func (s *Session) StartInference(ctx context.Context) (*ExecutionHandle, error) {
   195		if s == nil {
   196			return nil, ErrSessionNil
   197		}
   198		if s.SessionID == "" {
   199			return nil, ErrSessionIDEmpty
   200		}
   201		if s.Builder == nil {
   202			return nil, ErrSessionBuilderNil
   203		}
   204		if ctx == nil {
   205			ctx = context.Background()
   206		}
   207	
   208		s.mu.Lock()
   209		if s.active != nil && s.active.IsRunning() {
   210			s.mu.Unlock()
   211			return nil, ErrSessionAlreadyActive
   212		}
   213		var input *turns.Turn
   214		if len(s.Turns) > 0 {
   215			input = s.Turns[len(s.Turns)-1]
   216		}
   217		if input == nil || len(input.Blocks) == 0 {
   218			s.mu.Unlock()
   219			return nil, ErrSessionEmptyTurn
   220		}
   221	
   222		// Inference runs against the latest appended turn in-place. This allows middlewares
   223		// to intentionally mutate the turn so the updated version becomes the next seed base.
   224		if input.ID == "" {
   225			input.ID = uuid.NewString()
   226		}
   227		_ = turns.KeyTurnMetaSessionID.Set(&input.Metadata, s.SessionID)
   228		inferenceID := uuid.NewString()
   229		_ = turns.KeyTurnMetaInferenceID.Set(&input.Metadata, inferenceID)
   230		s.mu.Unlock()
   231	
   232		runner, err := s.Builder.Build(ctx, s.SessionID)
   233		if err != nil {
   234			return nil, err
   235		}
   236	
   237		runCtx, cancel := context.WithCancel(ctx)
   238		runCtx = WithSessionMeta(runCtx, s.SessionID, inferenceID)
   239		handle := newExecutionHandle(s.SessionID, inferenceID, input, cancel)
   240	
   241		s.mu.Lock()
   242		// Re-check after build: another goroutine may have started a run while we were building.
   243		if s.active != nil && s.active.IsRunning() {
   244			s.mu.Unlock()
   245			cancel()
   246			return nil, ErrSessionAlreadyActive
   247		}
   248		s.active = handle
   249		s.mu.Unlock()
   250	
   251		go func() {
   252			defer func() {
   253				s.mu.Lock()
   254				s.active = nil
   255				s.mu.Unlock()
   256			}()
   257	
   258			out, err := runner.RunInference(runCtx, input)
   259			if err == nil {
   260				if out == nil {
   261					out = input
   262				}
   263				if out.ID == "" {
   264					out.ID = input.ID
   265				}
   266				_ = turns.KeyTurnMetaSessionID.Set(&out.Metadata, s.SessionID)
   267				_ = turns.KeyTurnMetaInferenceID.Set(&out.Metadata, inferenceID)
   268	
   269				// Keep the session's latest turn as the canonical result, even if the runner
   270				// returns a different pointer.
   271				if out != input {
   272					s.mu.Lock()
   273					*input = *out
   274					s.mu.Unlock()
   275					out = input
   276				}
   277			}
   278			handle.setResult(out, err)
   279		}()
   280	
```

## ExecutionHandle wait/cancel

```text
     1	package session
     2	
     3	import (
     4		"context"
     5		"errors"
     6		"sync"
     7	
     8		"github.com/go-go-golems/geppetto/pkg/turns"
     9	)
    10	
    11	var ErrExecutionHandleNil = errors.New("execution handle is nil")
    12	
    13	// ExecutionHandle represents a single in-flight inference execution.
    14	//
    15	// It is cancelable and waitable. The underlying inference is always driven by context cancellation.
    16	type ExecutionHandle struct {
    17		SessionID   string
    18		InferenceID string
    19	
    20		Input *turns.Turn
    21	
    22		done chan struct{}
    23	
    24		mu     sync.Mutex
    25		cancel context.CancelFunc
    26		out    *turns.Turn
    27		err    error
    28	}
    29	
    30	func newExecutionHandle(sessionID, inferenceID string, input *turns.Turn, cancel context.CancelFunc) *ExecutionHandle {
    31		return &ExecutionHandle{
    32			SessionID:   sessionID,
    33			InferenceID: inferenceID,
    34			Input:       input,
    35			done:        make(chan struct{}),
    36			cancel:      cancel,
    37		}
    38	}
    39	
    40	func (h *ExecutionHandle) setResult(out *turns.Turn, err error) {
    41		h.mu.Lock()
    42		h.out = out
    43		h.err = err
    44		close(h.done)
    45		h.cancel = nil
    46		h.mu.Unlock()
    47	}
    48	
    49	// Cancel cancels the in-flight inference. It is safe to call multiple times.
    50	func (h *ExecutionHandle) Cancel() {
    51		if h == nil {
    52			return
    53		}
    54		h.mu.Lock()
    55		cancel := h.cancel
    56		h.mu.Unlock()
    57		if cancel != nil {
    58			cancel()
    59		}
    60	}
    61	
    62	// Wait blocks until the inference completes and returns the output Turn and error.
    63	func (h *ExecutionHandle) Wait() (*turns.Turn, error) {
    64		if h == nil {
    65			return nil, ErrExecutionHandleNil
    66		}
    67		<-h.done
    68		h.mu.Lock()
    69		defer h.mu.Unlock()
    70		return h.out, h.err
    71	}
    72	
    73	// IsRunning reports whether the inference appears to still be running.
    74	func (h *ExecutionHandle) IsRunning() bool {
    75		if h == nil {
    76			return false
    77		}
    78		select {
    79		case <-h.done:
    80			return false
```

## Enginebuilder event sink injection

```text
   140	func newEngineWithMiddlewares(eng engine.Engine, mws []middleware.Middleware) engine.Engine {
   141		if len(mws) == 0 {
   142			return eng
   143		}
   144	
   145		handler := func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
   146			return eng.RunInference(ctx, t)
   147		}
   148	
   149		return &engineWithMiddlewares{
   150			handler: middleware.Chain(handler, mws...),
   151		}
   152	}
   153	
   154	func (e *engineWithMiddlewares) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
   155		return e.handler(ctx, t)
   156	}
   157	
   158	func (r *runner) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
   159		if ctx == nil {
   160			ctx = context.Background()
   161		}
   162	
   163		runCtx := ctx
   164		if len(r.eventSinks) > 0 {
   165			runCtx = events.WithEventSinks(runCtx, r.eventSinks...)
   166		}
   167		if r.snapshotHook != nil {
   168			runCtx = toolloop.WithTurnSnapshotHook(runCtx, r.snapshotHook)
   169		}
   170	
   171		if t == nil {
   172			t = &turns.Turn{}
   173		}
   174		if t.ID == "" {
   175			t.ID = uuid.NewString()
   176		}
   177		if r.sessionID != "" {
   178			if _, ok, err := turns.KeyTurnMetaSessionID.Get(t.Metadata); err != nil || !ok {
   179				_ = turns.KeyTurnMetaSessionID.Set(&t.Metadata, r.sessionID)
   180			}
   181		}
   182		if _, ok, err := turns.KeyTurnMetaInferenceID.Get(t.Metadata); err != nil || !ok {
   183			_ = turns.KeyTurnMetaInferenceID.Set(&t.Metadata, uuid.NewString())
   184		}
   185	
   186		var (
   187			updated *turns.Turn
   188			err     error
   189		)
   190		preInferenceBlockCount := len(t.Blocks)
   191		if r.registry == nil {
   192			updated, _, err = engine.RunInferenceWithResult(runCtx, r.eng, t)
   193		} else {
   194			opts := []toolloop.Option{
   195				toolloop.WithEngine(r.eng),
   196				toolloop.WithRegistry(r.registry),
   197				toolloop.WithLoopConfig(r.loopCfg),
   198				toolloop.WithToolConfig(r.toolCfg),
   199				toolloop.WithStepController(r.stepController),
   200			}
   201			if r.toolExecutor != nil {
   202				opts = append(opts, toolloop.WithExecutor(r.toolExecutor))
   203			}
   204			if r.stepPauseTimeout > 0 {
   205				opts = append(opts, toolloop.WithPauseTimeout(r.stepPauseTimeout))
   206			}
   207			loop := toolloop.New(opts...)
   208			updated, err = loop.RunLoop(runCtx, t)
   209			// The tool loop calls eng.RunInference per iteration but does not
   210			// stamp block-level inference metadata. Extract from turn metadata
   211			// and project onto generated blocks so downstream consumers (UI)
   212			// can render per-block inference badges.
   213			if err == nil && updated != nil {
   214				if result, ok, getErr := engine.ExtractInferenceResult(updated); getErr == nil && ok {
   215					_ = engine.StampInferenceResultOnGeneratedBlocksFromIndex(updated, result, preInferenceBlockCount)
   216				}
   217			}
   218		}
   219	
   220		if updated != nil && r.sessionID != "" {
   221			if _, ok, err := turns.KeyTurnMetaSessionID.Get(updated.Metadata); err != nil || !ok {
   222				_ = turns.KeyTurnMetaSessionID.Set(&updated.Metadata, r.sessionID)
   223			}
   224		}
   225		if updated != nil {
   226			if updated.ID == "" {
   227				updated.ID = t.ID
   228			}
   229			if iid, ok, err := turns.KeyTurnMetaInferenceID.Get(t.Metadata); err == nil && ok && iid != "" {
   230				if _, ok2, err2 := turns.KeyTurnMetaInferenceID.Get(updated.Metadata); err2 != nil || !ok2 {
   231					_ = turns.KeyTurnMetaInferenceID.Set(&updated.Metadata, iid)
   232				}
   233			}
   234		}
   235	
   236		if err == nil && r.persister != nil && updated != nil {
   237			_ = r.persister.PersistTurn(runCtx, updated)
   238		}
   239	
   240		return updated, err
```

## EventSink interface

```text
     1	package events
     2	
     3	// EventSink represents a destination for inference events emitted while an inference
     4	// is running (including streaming deltas and the final completion).
     5	//
     6	// Intended use:
     7	// - UX/telemetry: live timelines, WebSocket broadcasts, logging, tracing, metrics
     8	// - Debugging: capture raw payloads and intermediate diagnostics
     9	//
    10	// Non-goal / warning:
    11	//   - Avoid committing durable application state based on partial/streaming events.
    12	//     Streaming output can be malformed, incomplete, or later superseded.
    13	//     Prefer doing validation + persistence at a clear RunInference boundary
    14	//     (e.g. when the final completion is known, or after RunInference returns).
    15	type EventSink interface {
    16		// PublishEvent publishes an event to the sink.
    17		// Returns an error if the event could not be published.
    18		PublishEvent(event Event) error
    19	}
```

## Event context helpers

```text
     1	package events
     2	
     3	import (
     4		"context"
     5	)
     6	
     7	// ctxKey is an unexported type for keys defined in this package.
     8	// This prevents collisions with keys defined in other packages.
     9	type ctxKey int
    10	
    11	const (
    12		ctxKeyEventSinks ctxKey = iota
    13	)
    14	
    15	// WithEventSinks attaches one or more EventSink instances to the context.
    16	// Downstream code can retrieve the sinks and publish events without
    17	// requiring access to engine configuration.
    18	func WithEventSinks(ctx context.Context, sinks ...EventSink) context.Context {
    19		if len(sinks) == 0 {
    20			return ctx
    21		}
    22		existing := GetEventSinks(ctx)
    23		combined := append([]EventSink{}, existing...)
    24		combined = append(combined, sinks...)
    25		return context.WithValue(ctx, ctxKeyEventSinks, combined)
    26	}
    27	
    28	// GetEventSinks returns the list of EventSinks attached to the context.
    29	func GetEventSinks(ctx context.Context) []EventSink {
    30		if v := ctx.Value(ctxKeyEventSinks); v != nil {
    31			if sinks, ok := v.([]EventSink); ok {
    32				return sinks
    33			}
    34		}
    35		return nil
    36	}
    37	
    38	// PublishEventToContext publishes the provided event to all EventSinks stored in the context.
    39	// If no sinks are present, this is a no-op.
    40	func PublishEventToContext(ctx context.Context, event Event) {
    41		sinks := GetEventSinks(ctx)
    42		if len(sinks) == 0 {
    43			return
    44		}
    45		for _, sink := range sinks {
    46			// Best-effort: ignore individual sink errors to avoid disrupting the flow
    47			_ = sink.PublishEvent(event)
    48		}
    49	}
```

## Canonical event type constants

```text
     1	package events
     2	
     3	import (
     4		"encoding/json"
     5		"fmt"
     6	
     7		"github.com/google/uuid"
     8		"github.com/rs/zerolog"
     9	)
    10	
    11	type EventType string
    12	
    13	const (
    14		// Canonical run lifecycle events.
    15		EventTypeRunStarted  EventType = "run-started"
    16		EventTypeRunFinished EventType = "run-finished"
    17		EventTypeRunStopped  EventType = "run-stopped"
    18		EventTypeRunFailed   EventType = "run-failed"
    19	
    20		// Canonical provider-call lifecycle events. These are non-transcript
    21		// events: they must not create or finish visible assistant text segments.
    22		EventTypeProviderCallStarted         EventType = "provider-call-started"
    23		EventTypeProviderCallMetadataUpdated EventType = "provider-call-metadata-updated"
    24		EventTypeProviderCallFinished        EventType = "provider-call-finished"
    25	
    26		// Canonical transcript segment events.
    27		EventTypeTextSegmentStarted  EventType = "text-segment-started"
    28		EventTypeTextDelta           EventType = "text-delta"
    29		EventTypeTextSegmentFinished EventType = "text-segment-finished"
    30	
    31		// Canonical reasoning segment events.
    32		EventTypeReasoningSegmentStarted  EventType = "reasoning-segment-started"
    33		EventTypeReasoningDelta           EventType = "reasoning-delta"
    34		EventTypeReasoningSegmentFinished EventType = "reasoning-segment-finished"
    35	
    36		// Canonical tool lifecycle events.
    37		EventTypeToolCallStarted        EventType = "tool-call-started"
    38		EventTypeToolCallArgumentsDelta EventType = "tool-call-arguments-delta"
    39		EventTypeToolCallRequested      EventType = "tool-call-requested"
    40		EventTypeToolExecutionStarted   EventType = "tool-execution-started"
    41		EventTypeToolResultReady        EventType = "tool-result-ready"
    42		EventTypeToolCallFinished       EventType = "tool-call-finished"
    43	
    44		EventTypeError     EventType = "error"
    45		EventTypeInterrupt EventType = "interrupt"
    46	
    47		// Informational/logging events (emitted by engines, middlewares or tools)
    48		EventTypeLog  EventType = "log"
    49		EventTypeInfo EventType = "info"
    50	
    51		// Debugger pause event (step-mode)
    52		EventTypeDebuggerPause EventType = "debugger.pause"
    53	
    54		// Agent-mode custom event (exported so UIs can act upon it)
    55		EventTypeAgentModeSwitch EventType = "agent-mode-switch"
    56	
    57		// Web search progress events (built-in/server tools)
    58		EventTypeWebSearchStarted   EventType = "web-search-started"
    59		EventTypeWebSearchSearching EventType = "web-search-searching"
    60		EventTypeWebSearchOpenPage  EventType = "web-search-open-page"
    61		EventTypeWebSearchDone      EventType = "web-search-done"
    62	
    63		// Citation annotations attached to output text
    64		EventTypeCitation EventType = "citation"
    65	
    66		// File search progress events
    67		EventTypeFileSearchStarted   EventType = "file-search-started"
    68		EventTypeFileSearchSearching EventType = "file-search-searching"
    69		EventTypeFileSearchDone      EventType = "file-search-done"
    70	
    71		// Code interpreter events
    72		EventTypeCodeInterpreterStarted      EventType = "code-interpreter-started"
    73		EventTypeCodeInterpreterInterpreting EventType = "code-interpreter-interpreting"
    74		EventTypeCodeInterpreterDone         EventType = "code-interpreter-done"
    75		EventTypeCodeInterpreterCodeDelta    EventType = "code-interpreter-code-delta"
    76		EventTypeCodeInterpreterCodeDone     EventType = "code-interpreter-code-done"
    77	
    78		// MCP tools
    79		EventTypeMCPArgsDelta      EventType = "mcp-args-delta"
    80		EventTypeMCPArgsDone       EventType = "mcp-args-done"
    81		EventTypeMCPInProgress     EventType = "mcp-in-progress"
    82		EventTypeMCPCompleted      EventType = "mcp-completed"
    83		EventTypeMCPFailed         EventType = "mcp-failed"
    84		EventTypeMCPListInProgress EventType = "mcp-list-tools-in-progress"
    85		EventTypeMCPListCompleted  EventType = "mcp-list-tools-completed"
    86		EventTypeMCPListFailed     EventType = "mcp-list-tools-failed"
    87	
    88		// Image generation built-in
    89		EventTypeImageGenInProgress   EventType = "image-generation-in-progress"
    90		EventTypeImageGenGenerating   EventType = "image-generation-generating"
    91		EventTypeImageGenPartialImage EventType = "image-generation-partial-image"
    92		EventTypeImageGenCompleted    EventType = "image-generation-completed"
    93	
    94		// Normalized tool results
    95		EventTypeToolSearchResults EventType = "tool-search-results"
    96	)
    97	
    98	type Event interface {
    99		Type() EventType
   100		Metadata() EventMetadata
   101		Payload() []byte
   102	}
   103	
   104	// MetadataSettingsSlug is the debug metadata key used to attach resolved
   105	// inference settings to EventMetadata.Extra. Extra is debug-only and must not be
   106	// used for routing or joining canonical events.
   107	const MetadataSettingsSlug = "settings"
   108	
   109	type EventImpl struct {
   110		Type_     EventType     `json:"type"`
   111		Error_    error         `json:"error,omitempty"`
   112		Metadata_ EventMetadata `json:"meta,omitempty"`
   113	
   114		// store payload if the event was deserialized from JSON (see NewEventFromJson), not further used
   115		payload []byte
   116	}
   117	
   118	func (e *EventImpl) MarshalZerologObject(ev *zerolog.Event) {
   119		ev.Str("type", string(e.Type_))
   120	
```

## Canonical text/provider events

```text
     1	package events
     2	
     3	import "github.com/rs/zerolog"
     4	
     5	type EventRunStarted struct {
     6		EventImpl
     7		Correlation_ Correlation `json:"correlation"`
     8		Prompt       string      `json:"prompt,omitempty"`
     9	}
    10	
    11	func NewRunStartedEvent(metadata EventMetadata, corr Correlation, prompt string) *EventRunStarted {
    12		return &EventRunStarted{EventImpl: EventImpl{Type_: EventTypeRunStarted, Metadata_: metadata}, Correlation_: corr, Prompt: prompt}
    13	}
    14	
    15	func (e *EventRunStarted) Correlation() Correlation { return e.Correlation_ }
    16	
    17	var _ CorrelatedEvent = &EventRunStarted{}
    18	
    19	type EventRunFinished struct {
    20		EventImpl
    21		Correlation_ Correlation `json:"correlation"`
    22		Status       string      `json:"status,omitempty"`
    23	}
    24	
    25	func NewRunFinishedEvent(metadata EventMetadata, corr Correlation, status string) *EventRunFinished {
    26		return &EventRunFinished{EventImpl: EventImpl{Type_: EventTypeRunFinished, Metadata_: metadata}, Correlation_: corr, Status: status}
    27	}
    28	
    29	func (e *EventRunFinished) Correlation() Correlation { return e.Correlation_ }
    30	
    31	var _ CorrelatedEvent = &EventRunFinished{}
    32	
    33	type EventRunStopped struct {
    34		EventImpl
    35		Correlation_ Correlation `json:"correlation"`
    36		Reason       string      `json:"reason,omitempty"`
    37	}
    38	
    39	func NewRunStoppedEvent(metadata EventMetadata, corr Correlation, reason string) *EventRunStopped {
    40		return &EventRunStopped{EventImpl: EventImpl{Type_: EventTypeRunStopped, Metadata_: metadata}, Correlation_: corr, Reason: reason}
    41	}
    42	
    43	func (e *EventRunStopped) Correlation() Correlation { return e.Correlation_ }
    44	
    45	var _ CorrelatedEvent = &EventRunStopped{}
    46	
    47	type EventRunFailed struct {
    48		EventImpl
    49		Correlation_ Correlation `json:"correlation"`
    50		ErrorString  string      `json:"error_string,omitempty"`
    51	}
    52	
    53	func NewRunFailedEvent(metadata EventMetadata, corr Correlation, err error) *EventRunFailed {
    54		errorString := ""
    55		if err != nil {
    56			errorString = err.Error()
    57		}
    58		return &EventRunFailed{EventImpl: EventImpl{Type_: EventTypeRunFailed, Metadata_: metadata, Error_: err}, Correlation_: corr, ErrorString: errorString}
    59	}
    60	
    61	func (e *EventRunFailed) Correlation() Correlation { return e.Correlation_ }
    62	
    63	var _ CorrelatedEvent = &EventRunFailed{}
    64	
    65	type EventProviderCallStarted struct {
    66		EventImpl
    67		Correlation_ Correlation `json:"correlation"`
    68	}
    69	
    70	func NewProviderCallStartedEvent(metadata EventMetadata, corr Correlation) *EventProviderCallStarted {
    71		return &EventProviderCallStarted{EventImpl: EventImpl{Type_: EventTypeProviderCallStarted, Metadata_: metadata}, Correlation_: corr}
    72	}
    73	
    74	func (e *EventProviderCallStarted) Correlation() Correlation { return e.Correlation_ }
    75	
    76	var _ CorrelatedEvent = &EventProviderCallStarted{}
    77	
    78	type EventProviderCallMetadataUpdated struct {
    79		EventImpl
    80		Correlation_ Correlation `json:"correlation"`
    81		StopReason   string      `json:"stop_reason,omitempty"`
    82		StopSequence string      `json:"stop_sequence,omitempty"`
    83		Usage        *Usage      `json:"usage,omitempty"`
    84	}
    85	
    86	func NewProviderCallMetadataUpdatedEvent(metadata EventMetadata, corr Correlation, stopReason, stopSequence string, usage *Usage) *EventProviderCallMetadataUpdated {
    87		return &EventProviderCallMetadataUpdated{EventImpl: EventImpl{Type_: EventTypeProviderCallMetadataUpdated, Metadata_: metadata}, Correlation_: corr, StopReason: stopReason, StopSequence: stopSequence, Usage: usage}
    88	}
    89	
    90	func (e *EventProviderCallMetadataUpdated) Correlation() Correlation { return e.Correlation_ }
    91	
    92	var _ CorrelatedEvent = &EventProviderCallMetadataUpdated{}
    93	
    94	type EventProviderCallFinished struct {
    95		EventImpl
    96		Correlation_ Correlation `json:"correlation"`
    97		StopReason   string      `json:"stop_reason,omitempty"`
    98		FinishClass  string      `json:"finish_class,omitempty"`
    99		Usage        *Usage      `json:"usage,omitempty"`
   100		DurationMs   *int64      `json:"duration_ms,omitempty"`
   101		HasToolCalls bool        `json:"has_tool_calls,omitempty"`
   102	}
   103	
   104	func NewProviderCallFinishedEvent(metadata EventMetadata, corr Correlation, stopReason, finishClass string, usage *Usage, durationMs *int64, hasToolCalls bool) *EventProviderCallFinished {
   105		return &EventProviderCallFinished{EventImpl: EventImpl{Type_: EventTypeProviderCallFinished, Metadata_: metadata}, Correlation_: corr, StopReason: stopReason, FinishClass: finishClass, Usage: usage, DurationMs: durationMs, HasToolCalls: hasToolCalls}
   106	}
   107	
   108	func (e *EventProviderCallFinished) Correlation() Correlation { return e.Correlation_ }
   109	
   110	var _ CorrelatedEvent = &EventProviderCallFinished{}
   111	
   112	type EventTextSegmentStarted struct {
   113		EventImpl
   114		Correlation_ Correlation `json:"correlation"`
   115		Role         string      `json:"role,omitempty"`
   116	}
   117	
   118	func NewTextSegmentStartedEvent(metadata EventMetadata, corr Correlation, role string) *EventTextSegmentStarted {
   119		return &EventTextSegmentStarted{EventImpl: EventImpl{Type_: EventTypeTextSegmentStarted, Metadata_: metadata}, Correlation_: corr, Role: role}
   120	}
   121	
   122	func (e *EventTextSegmentStarted) Correlation() Correlation { return e.Correlation_ }
   123	
   124	var _ CorrelatedEvent = &EventTextSegmentStarted{}
   125	
   126	type EventTextDelta struct {
   127		EventImpl
   128		Correlation_ Correlation `json:"correlation"`
   129		Delta        string      `json:"delta"`
   130		Text         string      `json:"text"`
   131		Sequence     int64       `json:"sequence,omitempty"`
   132	}
   133	
   134	func NewTextDeltaEvent(metadata EventMetadata, corr Correlation, delta, text string, sequence int64) *EventTextDelta {
   135		return &EventTextDelta{EventImpl: EventImpl{Type_: EventTypeTextDelta, Metadata_: metadata}, Correlation_: corr, Delta: delta, Text: text, Sequence: sequence}
   136	}
   137	
   138	func (e *EventTextDelta) Correlation() Correlation { return e.Correlation_ }
   139	
   140	var _ CorrelatedEvent = &EventTextDelta{}
   141	
   142	type EventTextSegmentFinished struct {
   143		EventImpl
   144		Correlation_ Correlation `json:"correlation"`
   145		Text         string      `json:"text"`
   146		FinishReason string      `json:"finish_reason,omitempty"`
   147	}
   148	
   149	func NewTextSegmentFinishedEvent(metadata EventMetadata, corr Correlation, text, finishReason string) *EventTextSegmentFinished {
   150		return &EventTextSegmentFinished{EventImpl: EventImpl{Type_: EventTypeTextSegmentFinished, Metadata_: metadata}, Correlation_: corr, Text: text, FinishReason: finishReason}
   151	}
   152	
   153	func (e *EventTextSegmentFinished) Correlation() Correlation { return e.Correlation_ }
   154	
   155	var _ CorrelatedEvent = &EventTextSegmentFinished{}
   156	
   157	type EventReasoningSegmentStarted struct {
   158		EventImpl
   159		Correlation_ Correlation `json:"correlation"`
   160		Source       string      `json:"source,omitempty"`
   161	}
   162	
   163	func NewReasoningSegmentStartedEvent(metadata EventMetadata, corr Correlation, source string) *EventReasoningSegmentStarted {
   164		return &EventReasoningSegmentStarted{EventImpl: EventImpl{Type_: EventTypeReasoningSegmentStarted, Metadata_: metadata}, Correlation_: corr, Source: source}
   165	}
   166	
   167	func (e *EventReasoningSegmentStarted) Correlation() Correlation { return e.Correlation_ }
   168	
   169	var _ CorrelatedEvent = &EventReasoningSegmentStarted{}
   170	
   171	type EventReasoningDelta struct {
   172		EventImpl
   173		Correlation_ Correlation `json:"correlation"`
   174		Delta        string      `json:"delta"`
   175		Text         string      `json:"text"`
   176		Sequence     int64       `json:"sequence,omitempty"`
   177		Source       string      `json:"source,omitempty"`
   178	}
   179	
   180	func NewReasoningDeltaEvent(metadata EventMetadata, corr Correlation, delta, text string, sequence int64) *EventReasoningDelta {
   181		return NewReasoningDeltaEventWithSource(metadata, corr, "", delta, text, sequence)
   182	}
   183	
   184	func NewReasoningDeltaEventWithSource(metadata EventMetadata, corr Correlation, source, delta, text string, sequence int64) *EventReasoningDelta {
   185		return &EventReasoningDelta{EventImpl: EventImpl{Type_: EventTypeReasoningDelta, Metadata_: metadata}, Correlation_: corr, Delta: delta, Text: text, Sequence: sequence, Source: source}
   186	}
   187	
   188	func (e *EventReasoningDelta) Correlation() Correlation { return e.Correlation_ }
   189	
   190	var _ CorrelatedEvent = &EventReasoningDelta{}
   191	
   192	type EventReasoningSegmentFinished struct {
   193		EventImpl
   194		Correlation_ Correlation `json:"correlation"`
   195		Text         string      `json:"text,omitempty"`
   196		FinishReason string      `json:"finish_reason,omitempty"`
   197		Source       string      `json:"source,omitempty"`
   198	}
   199	
   200	func NewReasoningSegmentFinishedEvent(metadata EventMetadata, corr Correlation, text, finishReason string) *EventReasoningSegmentFinished {
   201		return NewReasoningSegmentFinishedEventWithSource(metadata, corr, "", text, finishReason)
   202	}
   203	
   204	func NewReasoningSegmentFinishedEventWithSource(metadata EventMetadata, corr Correlation, source, text, finishReason string) *EventReasoningSegmentFinished {
   205		return &EventReasoningSegmentFinished{EventImpl: EventImpl{Type_: EventTypeReasoningSegmentFinished, Metadata_: metadata}, Correlation_: corr, Text: text, FinishReason: finishReason, Source: source}
   206	}
   207	
   208	func (e *EventReasoningSegmentFinished) Correlation() Correlation { return e.Correlation_ }
   209	
   210	var _ CorrelatedEvent = &EventReasoningSegmentFinished{}
   211	
   212	func (e EventTextDelta) MarshalZerologObject(ev *zerolog.Event) {
   213		e.EventImpl.MarshalZerologObject(ev)
   214		ev.Str("delta", e.Delta).Str("text", e.Text)
   215	}
   216	
   217	func (e EventTextSegmentFinished) MarshalZerologObject(ev *zerolog.Event) {
   218		e.EventImpl.MarshalZerologObject(ev)
   219		ev.Str("text", e.Text).Str("finish_reason", e.FinishReason)
   220	}
```

## Canonical tool events

```text
     1	package events
     2	
     3	type EventToolCallStarted struct {
     4		EventImpl
     5		Correlation_ Correlation `json:"correlation"`
     6		ToolCallID   string      `json:"tool_call_id"`
     7		ToolName     string      `json:"tool_name,omitempty"`
     8	}
     9	
    10	func NewToolCallStartedEvent(metadata EventMetadata, corr Correlation, toolCallID, toolName string) *EventToolCallStarted {
    11		return &EventToolCallStarted{EventImpl: EventImpl{Type_: EventTypeToolCallStarted, Metadata_: metadata}, Correlation_: corr, ToolCallID: toolCallID, ToolName: toolName}
    12	}
    13	
    14	func (e *EventToolCallStarted) Correlation() Correlation { return e.Correlation_ }
    15	
    16	var _ CorrelatedEvent = &EventToolCallStarted{}
    17	
    18	type EventToolCallArgumentsDelta struct {
    19		EventImpl
    20		Correlation_ Correlation `json:"correlation"`
    21		ToolCallID   string      `json:"tool_call_id"`
    22		Delta        string      `json:"delta"`
    23		Arguments    string      `json:"arguments"`
    24		Sequence     int64       `json:"sequence,omitempty"`
    25	}
    26	
    27	func NewToolCallArgumentsDeltaEvent(metadata EventMetadata, corr Correlation, toolCallID, delta, arguments string, sequence int64) *EventToolCallArgumentsDelta {
    28		return &EventToolCallArgumentsDelta{EventImpl: EventImpl{Type_: EventTypeToolCallArgumentsDelta, Metadata_: metadata}, Correlation_: corr, ToolCallID: toolCallID, Delta: delta, Arguments: arguments, Sequence: sequence}
    29	}
    30	
    31	func (e *EventToolCallArgumentsDelta) Correlation() Correlation { return e.Correlation_ }
    32	
    33	var _ CorrelatedEvent = &EventToolCallArgumentsDelta{}
    34	
    35	type EventToolCallRequested struct {
    36		EventImpl
    37		Correlation_ Correlation `json:"correlation"`
    38		ToolCallID   string      `json:"tool_call_id"`
    39		ToolName     string      `json:"tool_name"`
    40		Input        string      `json:"input"`
    41	}
    42	
    43	func NewToolCallRequestedEvent(metadata EventMetadata, corr Correlation, toolCallID, toolName, input string) *EventToolCallRequested {
    44		return &EventToolCallRequested{EventImpl: EventImpl{Type_: EventTypeToolCallRequested, Metadata_: metadata}, Correlation_: corr, ToolCallID: toolCallID, ToolName: toolName, Input: input}
    45	}
    46	
    47	func (e *EventToolCallRequested) Correlation() Correlation { return e.Correlation_ }
    48	
    49	var _ CorrelatedEvent = &EventToolCallRequested{}
    50	
    51	type EventToolExecutionStarted struct {
    52		EventImpl
    53		Correlation_ Correlation `json:"correlation"`
    54		ToolCallID   string      `json:"tool_call_id"`
    55		ToolName     string      `json:"tool_name,omitempty"`
    56		Input        string      `json:"input,omitempty"`
    57	}
    58	
    59	func NewToolExecutionStartedEvent(metadata EventMetadata, corr Correlation, toolCallID, toolName, input string) *EventToolExecutionStarted {
    60		return &EventToolExecutionStarted{EventImpl: EventImpl{Type_: EventTypeToolExecutionStarted, Metadata_: metadata}, Correlation_: corr, ToolCallID: toolCallID, ToolName: toolName, Input: input}
    61	}
    62	
    63	func (e *EventToolExecutionStarted) Correlation() Correlation { return e.Correlation_ }
    64	
    65	var _ CorrelatedEvent = &EventToolExecutionStarted{}
    66	
    67	type EventToolResultReady struct {
    68		EventImpl
    69		Correlation_ Correlation `json:"correlation"`
    70		ToolCallID   string      `json:"tool_call_id"`
    71		ToolName     string      `json:"tool_name,omitempty"`
    72		Result       string      `json:"result"`
    73		Status       string      `json:"status,omitempty"`
    74	}
    75	
    76	func NewToolResultReadyEvent(metadata EventMetadata, corr Correlation, toolCallID, toolName, result, status string) *EventToolResultReady {
    77		return &EventToolResultReady{EventImpl: EventImpl{Type_: EventTypeToolResultReady, Metadata_: metadata}, Correlation_: corr, ToolCallID: toolCallID, ToolName: toolName, Result: result, Status: status}
    78	}
    79	
    80	func (e *EventToolResultReady) Correlation() Correlation { return e.Correlation_ }
    81	
    82	var _ CorrelatedEvent = &EventToolResultReady{}
    83	
    84	type EventToolCallFinished struct {
    85		EventImpl
    86		Correlation_ Correlation `json:"correlation"`
    87		ToolCallID   string      `json:"tool_call_id"`
    88		ToolName     string      `json:"tool_name,omitempty"`
    89		Status       string      `json:"status,omitempty"`
    90	}
    91	
    92	func NewToolCallFinishedEvent(metadata EventMetadata, corr Correlation, toolCallID, toolName, status string) *EventToolCallFinished {
    93		return &EventToolCallFinished{EventImpl: EventImpl{Type_: EventTypeToolCallFinished, Metadata_: metadata}, Correlation_: corr, ToolCallID: toolCallID, ToolName: toolName, Status: status}
    94	}
    95	
    96	func (e *EventToolCallFinished) Correlation() Correlation { return e.Correlation_ }
    97	
    98	var _ CorrelatedEvent = &EventToolCallFinished{}
```

## go-go-goja EventEmitter module

```text
     1	package events
     2	
     3	import (
     4		"fmt"
     5		"reflect"
     6		"sort"
     7	
     8		"github.com/dop251/goja"
     9		"github.com/go-go-golems/go-go-goja/modules"
    10		"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
    11	)
    12	
    13	type module struct {
    14		name string
    15	}
    16	
    17	var _ modules.NativeModule = (*module)(nil)
    18	var _ modules.TypeScriptDeclarer = (*module)(nil)
    19	
    20	// EventEmitter is the Go-native backing object for JavaScript EventEmitter
    21	// instances returned by require("events").
    22	//
    23	// It is not goroutine-safe. All methods that touch listeners or goja values must
    24	// be called on the owning goja runtime goroutine.
    25	type EventEmitter struct {
    26		vm        *goja.Runtime
    27		object    *goja.Object
    28		listeners map[eventName][]listenerEntry
    29	}
    30	
    31	type listenerEntry struct {
    32		value    goja.Value
    33		callable goja.Callable
    34		once     bool
    35		original goja.Value
    36	}
    37	
    38	var eventEmitterType = reflect.TypeOf((*EventEmitter)(nil))
    39	
    40	type eventName struct {
    41		text   string
    42		symbol *goja.Symbol
    43	}
    44	
    45	func eventNameFromString(name string) eventName {
    46		return eventName{text: name}
    47	}
    48	
    49	func eventNameFromValue(value goja.Value) eventName {
    50		if sym, ok := value.(*goja.Symbol); ok {
    51			return eventName{symbol: sym}
    52		}
    53		if value == nil || goja.IsUndefined(value) {
    54			return eventName{text: "undefined"}
    55		}
    56		return eventName{text: value.String()}
    57	}
    58	
    59	func (n eventName) isString(name string) bool {
    60		return n.symbol == nil && n.text == name
    61	}
    62	
    63	func (n eventName) value(vm *goja.Runtime) goja.Value {
    64		if n.symbol != nil {
    65			return n.symbol
    66		}
    67		return vm.ToValue(n.text)
    68	}
    69	
    70	func (n eventName) sortKey() string {
    71		if n.symbol != nil {
    72			return fmt.Sprintf("symbol:%s:%p", n.symbol.String(), n.symbol)
    73		}
    74		return "string:" + n.text
    75	}
    76	
    77	func (m *module) Name() string { return m.name }
    78	
    79	func (m *module) Doc() string {
    80		return `
    81	The events module provides a Go-native subset of Node.js EventEmitter.
    82	
    83	Exports:
    84	  EventEmitter / module.exports: constructor for Go-backed EventEmitter objects.
    85	
    86	Supported methods:
    87	  on/addListener, once, off/removeListener, removeAllListeners, emit,
    88	  listeners, rawListeners, listenerCount, eventNames.
    89	`
    90	}
    91	
    92	func (m *module) TypeScriptModule() *spec.Module {
    93		return &spec.Module{
    94			Name: m.name,
    95			RawDTS: []string{
    96				"type EventName = string | symbol;",
    97				"type Listener = (...args: any[]) => void;",
    98				"class EventEmitter {",
    99				"  constructor();",
   100				"  on(name: EventName, listener: Listener): this;",
   101				"  addListener(name: EventName, listener: Listener): this;",
   102				"  once(name: EventName, listener: Listener): this;",
   103				"  off(name: EventName, listener: Listener): this;",
   104				"  removeListener(name: EventName, listener: Listener): this;",
   105				"  removeAllListeners(name?: EventName): this;",
   106				"  emit(name: EventName, ...args: any[]): boolean;",
   107				"  listeners(name: EventName): Listener[];",
   108				"  rawListeners(name: EventName): Listener[];",
   109				"  listenerCount(name: EventName): number;",
   110				"  eventNames(): EventName[];",
   111				"}",
   112				"export = EventEmitter;",
   113				"export { EventEmitter };",
   114			},
   115		}
   116	}
   117	
   118	func (m *module) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
   119		constructor := vm.ToValue(func(call goja.ConstructorCall) *goja.Object {
   120			emitter := New(vm)
   121			obj := vm.ToValue(emitter).(*goja.Object)
   122			if err := obj.SetPrototype(call.This.Prototype()); err != nil {
   123				panic(vm.NewGoError(fmt.Errorf("events: set emitter prototype: %w", err)))
   124			}
   125			emitter.object = obj
   126			return obj
   127		}).(*goja.Object)
   128	
   129		proto := vm.NewObject()
   130		mustSet(vm, proto, "on", func(call goja.FunctionCall) goja.Value {
   131			return methodOn(vm, call, false)
   132		})
   133		mustSet(vm, proto, "addListener", func(call goja.FunctionCall) goja.Value {
   134			return methodOn(vm, call, false)
   135		})
   136		mustSet(vm, proto, "once", func(call goja.FunctionCall) goja.Value {
   137			return methodOn(vm, call, true)
   138		})
   139		mustSet(vm, proto, "off", func(call goja.FunctionCall) goja.Value {
   140			return methodRemoveListener(vm, call)
   141		})
   142		mustSet(vm, proto, "removeListener", func(call goja.FunctionCall) goja.Value {
   143			return methodRemoveListener(vm, call)
   144		})
   145		mustSet(vm, proto, "removeAllListeners", func(call goja.FunctionCall) goja.Value {
   146			emitter := mustEmitter(vm, call.This)
   147			if len(call.Arguments) == 0 || goja.IsUndefined(call.Argument(0)) {
   148				emitter.RemoveAllListeners()
   149			} else {
   150				emitter.removeAllListeners(eventNameFromValue(call.Argument(0)))
   151			}
   152			return call.This
   153		})
   154		mustSet(vm, proto, "emit", func(call goja.FunctionCall) goja.Value {
   155			if len(call.Arguments) == 0 {
   156				panic(vm.NewTypeError("event name is required"))
   157			}
   158			emitter := mustEmitter(vm, call.This)
   159			name := eventNameFromValue(call.Argument(0))
   160			var args []goja.Value
   161			if len(call.Arguments) > 1 {
   162				args = call.Arguments[1:]
   163			}
   164			ok, err := emitter.emit(name, args)
   165			if err != nil {
   166				panic(vm.NewGoError(err))
   167			}
   168			return vm.ToValue(ok)
   169		})
   170		mustSet(vm, proto, "listeners", func(call goja.FunctionCall) goja.Value {
   171			emitter := mustEmitter(vm, call.This)
   172			return vm.ToValue(emitter.listenersForName(eventNameFromValue(call.Argument(0)), true))
   173		})
   174		mustSet(vm, proto, "rawListeners", func(call goja.FunctionCall) goja.Value {
   175			emitter := mustEmitter(vm, call.This)
   176			return vm.ToValue(emitter.listenersForName(eventNameFromValue(call.Argument(0)), false))
   177		})
   178		mustSet(vm, proto, "listenerCount", func(call goja.FunctionCall) goja.Value {
   179			emitter := mustEmitter(vm, call.This)
   180			return vm.ToValue(emitter.listenerCount(eventNameFromValue(call.Argument(0))))
   181		})
   182		mustSet(vm, proto, "eventNames", func(call goja.FunctionCall) goja.Value {
   183			emitter := mustEmitter(vm, call.This)
   184			return vm.ToValue(emitter.eventNameValues())
   185		})
   186	
   187		mustSet(vm, constructor, "prototype", proto)
   188		if err := proto.DefineDataProperty("constructor", constructor, goja.FLAG_FALSE, goja.FLAG_FALSE, goja.FLAG_FALSE); err != nil {
   189			panic(vm.NewGoError(fmt.Errorf("events: define constructor property: %w", err)))
   190		}
   191		mustSet(vm, constructor, "EventEmitter", constructor)
   192		mustSet(vm, constructor, "default", constructor)
   193	
   194		if err := moduleObj.Set("exports", constructor); err != nil {
   195			panic(vm.NewGoError(fmt.Errorf("events: set exports: %w", err)))
   196		}
   197	}
   198	
   199	// New creates a Go-native EventEmitter backing value for vm. The caller is
   200	// responsible for wrapping it in a goja object when exposing it to JavaScript.
   201	func New(vm *goja.Runtime) *EventEmitter {
   202		return &EventEmitter{
   203			vm:        vm,
   204			listeners: map[eventName][]listenerEntry{},
   205		}
   206	}
   207	
   208	// FromValue unwraps a JavaScript value created by the Go-native EventEmitter
   209	// constructor.
   210	func FromValue(value goja.Value) (*EventEmitter, *goja.Object, bool) {
   211		if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
   212			return nil, nil, false
   213		}
   214		if value.ExportType() != eventEmitterType {
   215			return nil, nil, false
   216		}
   217		emitter, ok := value.Export().(*EventEmitter)
   218		if !ok || emitter == nil || emitter.vm == nil {
   219			return nil, nil, false
   220		}
   221		obj := value.ToObject(emitter.vm)
   222		if emitter.object == nil {
   223			emitter.object = obj
   224		}
   225		return emitter, obj, true
   226	}
   227	
   228	// AddListenerValue registers a JavaScript callable value as a listener.
   229	func (e *EventEmitter) AddListenerValue(name string, value goja.Value) error {
   230		callable, ok := goja.AssertFunction(value)
   231		if !ok {
   232			return fmt.Errorf("listener must be a function")
   233		}
   234		e.addListener(eventNameFromString(name), listenerEntry{value: value, callable: callable})
   235		return nil
   236	}
   237	
   238	// AddGoListener registers a Go function as a listener. It must be called on the
   239	// owner goroutine for e's runtime.
   240	func (e *EventEmitter) AddGoListener(name string, fn func(goja.FunctionCall) goja.Value) error {
   241		if e == nil || e.vm == nil {
   242			return fmt.Errorf("events: nil emitter")
   243		}
   244		return e.AddListenerValue(name, e.vm.ToValue(fn))
   245	}
   246	
   247	// Emit invokes all listeners for name synchronously on the owner goroutine.
   248	func (e *EventEmitter) Emit(name string, args ...goja.Value) (bool, error) {
   249		if e == nil {
   250			return false, fmt.Errorf("events: nil emitter")
   251		}
   252		return e.emit(eventNameFromString(name), args)
   253	}
   254	
   255	func (e *EventEmitter) addListener(name eventName, entry listenerEntry) {
   256		if e.listeners == nil {
   257			e.listeners = map[eventName][]listenerEntry{}
   258		}
   259		e.listeners[name] = append(e.listeners[name], entry)
   260	}
```

## go-go-goja EventEmitter Go adoption test

```text
   126	func TestGoCanAdoptJSCreatedEmitterAndEmitToIt(t *testing.T) {
   127		rt := newRuntime(t)
   128		var adopted *eventsmodule.EventEmitter
   129	
   130		_, err := rt.Owner.Call(context.Background(), "events.install-adopt", func(_ context.Context, vm *goja.Runtime) (any, error) {
   131			if err := vm.Set("adoptEmitter", func(value goja.Value) bool {
   132				emitter, _, ok := eventsmodule.FromValue(value)
   133				if ok {
   134					adopted = emitter
   135				}
   136				return ok
   137			}); err != nil {
   138				return nil, err
   139			}
   140			_, err := vm.RunString(`
   141				const EventEmitter = require("events");
   142				globalThis.seen = [];
   143				globalThis.ee = new EventEmitter();
   144				globalThis.ee.on("go", value => globalThis.seen.push("js:" + value));
   145				adoptEmitter(globalThis.ee);
   146			`)
   147			return nil, err
   148		})
   149		require.NoError(t, err)
   150		require.NotNil(t, adopted)
   151	
   152		_, err = rt.Owner.Call(context.Background(), "events.emit-from-go", func(_ context.Context, vm *goja.Runtime) (any, error) {
   153			if err := adopted.AddGoListener("fromJS", func(call goja.FunctionCall) goja.Value {
   154				arg := call.Argument(0).String()
   155				seen := vm.Get("seen").ToObject(vm)
   156				push, ok := goja.AssertFunction(seen.Get("push"))
   157				require.True(t, ok)
   158				_, callErr := push(seen, vm.ToValue("go:"+arg))
   159				require.NoError(t, callErr)
   160				return goja.Undefined()
   161			}); err != nil {
   162				return nil, err
   163			}
   164			_, err := adopted.Emit("go", vm.ToValue("payload"))
   165			if err != nil {
   166				return nil, err
   167			}
   168			_, err = vm.RunString(`globalThis.ee.emit("fromJS", "callback")`)
   169			return nil, err
   170		})
   171		require.NoError(t, err)
   172	
   173		got := runJS(t, rt, `JSON.stringify(globalThis.seen)`)
   174		require.Equal(t, `["js:payload","go:callback"]`, got)
   175	}
   176	
   177	func TestEventsModuleIsEnabledByDefault(t *testing.T) {
   178		rt := newRuntime(t)
   179		got := runJS(t, rt, `
   180			function canRequire(name) {
   181				try { return typeof require(name).EventEmitter === "function"; }
   182				catch (e) { return false; }
   183			}
   184			JSON.stringify({ events: canRequire("events"), nodeEvents: canRequire("node:events") });
   185		`)
   186		require.JSONEq(t, `{"events":true,"nodeEvents":true}`, got)
   187	}
   188	
   189	func newRuntime(t *testing.T) *gggengine.Runtime {
   190		t.Helper()
```

## Runner streaming example sink

```text
     1	package main
     2	
     3	import (
     4		"context"
     5		"fmt"
     6		"io"
     7	
     8		"github.com/go-go-golems/geppetto/cmd/examples/internal/examplecmd"
     9		"github.com/go-go-golems/geppetto/cmd/examples/internal/runnerexample"
    10		"github.com/go-go-golems/geppetto/pkg/events"
    11		"github.com/go-go-golems/geppetto/pkg/inference/runner"
    12		geppettosections "github.com/go-go-golems/geppetto/pkg/sections"
    13		"github.com/go-go-golems/geppetto/pkg/turns"
    14		"github.com/go-go-golems/glazed/pkg/cmds"
    15		"github.com/go-go-golems/glazed/pkg/cmds/fields"
    16		"github.com/go-go-golems/glazed/pkg/cmds/values"
    17		"github.com/pkg/errors"
    18		"github.com/spf13/cobra"
    19	)
    20	
    21	type writerSink struct {
    22		w io.Writer
    23	}
    24	
    25	func (s *writerSink) PublishEvent(event events.Event) error {
    26		_, err := fmt.Fprintf(s.w, "event: %s\n", event.Type())
    27		return err
    28	}
    29	
    30	type runCommand struct {
    31		*cmds.CommandDescription
    32	}
    33	
    34	var _ cmds.WriterCommand = (*runCommand)(nil)
    35	
    36	type runSettings struct {
    37		Prompt string `glazed:"prompt"`
    38	}
    39	
    40	func newRunCommand() (*runCommand, error) {
    41		profileSettingsSection, err := geppettosections.NewProfileSettingsSection(
    42			geppettosections.WithProfileDefault("gpt-5-nano-low"),
    43			geppettosections.WithProfileRegistriesDefault(runnerexample.PinocchioProfileRegistryPath()),
    44		)
    45		if err != nil {
    46			return nil, err
    47		}
    48	
    49		description := cmds.NewCommandDescription(
    50			"run",
    51			cmds.WithShort("Run a profile-backed streaming inference request"),
    52			cmds.WithArguments(
    53				fields.New(
    54					"prompt",
    55					fields.TypeString,
    56					fields.WithHelp("Prompt to run"),
    57					fields.WithDefault("Explain, in a few sentences, how event sinks help streaming applications."),
    58				),
    59			),
    60			cmds.WithSections(profileSettingsSection),
    61		)
    62	
    63		return &runCommand{CommandDescription: description}, nil
    64	}
    65	
    66	func (c *runCommand) RunIntoWriter(ctx context.Context, parsedValues *values.Values, w io.Writer) error {
    67		s := &runSettings{}
    68		if err := parsedValues.DecodeSectionInto(values.DefaultSlug, s); err != nil {
    69			return errors.Wrap(err, "decode run settings")
    70		}
    71		profileSettings := &geppettosections.ProfileSettings{}
    72		if err := parsedValues.DecodeSectionInto(geppettosections.ProfileSettingsSectionSlug, profileSettings); err != nil {
    73			return errors.Wrap(err, "decode profile settings")
    74		}
    75	
    76		stepSettings, closeProfiles, err := runnerexample.ResolveInferenceSettingsFromRegistry(ctx, profileSettings.ProfileRegistries, profileSettings.Profile)
    77		if err != nil {
    78			return err
    79		}
    80		defer func() {
    81			if closeProfiles != nil {
    82				_ = closeProfiles()
    83			}
    84		}()
    85	
    86		r := runner.New()
    87		prepared, handle, err := r.Start(ctx, runner.StartRequest{
    88			Prompt: s.Prompt,
    89			Runtime: runner.Runtime{
    90				InferenceSettings: stepSettings,
    91				SystemPrompt:      "You are a concise assistant.",
    92			},
    93			EventSinks: []events.EventSink{&writerSink{w: w}},
    94		})
    95		if err != nil {
    96			return err
    97		}
    98	
    99		fmt.Fprintf(w, "session: %s\n", prepared.Session.SessionID)
   100		out, err := handle.Wait()
   101		if err != nil {
   102			return err
   103		}
   104	
   105		fmt.Fprintln(w, "\nfinal turn:")
   106		turns.FprintTurn(w, out)
   107		return nil
   108	}
   109	
   110	func main() {
   111		root := examplecmd.NewRoot("runner-streaming", "Profile-backed streaming runner example")
   112		cmd, err := newRunCommand()
   113		cobra.CheckErr(err)
   114		cobra.CheckErr(examplecmd.ExecuteSingleCommand(root, "geppetto", cmd))
   115	}
```
