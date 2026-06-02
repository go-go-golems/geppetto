package geppetto

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/events"
	gosession "github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop/enginebuilder"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
	"github.com/google/uuid"
)

type resumeMode int

const (
	resumeNone resumeMode = iota
	resumeLatest
)

type sessionBuilderRef struct {
	api   *moduleRuntime
	agent *agentRef

	id   string
	name string

	base       *turns.Turn
	baseSource string

	persistMode persistMode
	persister   enginebuilder.TurnPersister
	store       *turnStoreRef

	resumeMode     resumeMode
	resumeQuery    TurnStoreQuery
	resumeRequired bool

	metadata    map[string]any
	runDefaults runOptions
}

type agentSessionRef struct {
	api   *moduleRuntime
	agent *agentRef
	sess  *gosession.Session

	name string

	persistMode persistMode
	persister   enginebuilder.TurnPersister
	store       *turnStoreRef

	metadata    map[string]any
	runDefaults runOptions
	runtimeMeta map[string]any

	closed bool
}

type sessionTurnBuilderRef struct {
	api     *moduleRuntime
	session *agentSessionRef
	turn    *turns.Turn
}

type startedSessionRun struct {
	handle         *gosession.ExecutionHandle
	inputSnapshot  *turns.Turn
	effectiveTurn  *turns.Turn
	cancel         context.CancelFunc
	runScopedSinks []*jsEventEmitterSink
}

func (m *moduleRuntime) newSessionBuilderObject(ref *sessionBuilderRef) *goja.Object {
	if ref == nil {
		ref = &sessionBuilderRef{api: m}
	}
	ref.api = m
	o := m.vm.NewObject()
	m.attachRef(o, ref)
	m.mustSet(o, "id", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			ref.id = strings.TrimSpace(call.Arguments[0].String())
		}
		return o
	})
	m.mustSet(o, "name", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			ref.name = strings.TrimSpace(call.Arguments[0].String())
		}
		return o
	})
	m.mustSet(o, "base", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
			panic(m.vm.NewTypeError("session().base requires a Go-owned Turn wrapper"))
		}
		turn, err := m.requireTurnRef(call.Arguments[0])
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		ref.base = turn.turn.Clone()
		ref.baseSource = "base"
		return o
	})
	m.mustSet(o, "store", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
			panic(m.vm.NewTypeError("session().store requires a TurnStore wrapper"))
		}
		store, err := m.requireTurnStoreRef(call.Arguments[0])
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		ref.persistMode = persistExplicit
		ref.store = store
		ref.persister = store
		return o
	})
	m.mustSet(o, "defaultStore", func(goja.FunctionCall) goja.Value {
		store := m.defaultReadableTurnStoreRef()
		if store != nil {
			ref.persistMode = persistUseDefault
			ref.store = store
			ref.persister = store
			return o
		}
		persister, err := m.defaultTurnPersister()
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		ref.persistMode = persistUseDefault
		ref.persister = persister
		ref.store = nil
		return o
	})
	m.mustSet(o, "persist", func(call goja.FunctionCall) goja.Value {
		enabled := true
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
			enabled = call.Arguments[0].ToBoolean()
		}
		if !enabled {
			ref.persistMode = persistDisabled
			ref.persister = nil
			return o
		}
		if ref.persister != nil {
			if ref.persistMode == persistDisabled {
				ref.persistMode = persistExplicit
			}
			return o
		}
		persister, err := m.defaultTurnPersister()
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		ref.persistMode = persistUseDefault
		ref.persister = persister
		ref.store = m.defaultReadableTurnStoreRef()
		return o
	})
	m.mustSet(o, "resumeLatest", func(call goja.FunctionCall) goja.Value {
		q, required, err := m.parseResumeLatestOptions(call.Arguments, 0)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		ref.resumeMode = resumeLatest
		ref.resumeQuery = q
		ref.resumeRequired = required
		return o
	})
	m.mustSet(o, "resumeNone", func(goja.FunctionCall) goja.Value {
		ref.resumeMode = resumeNone
		ref.resumeQuery = TurnStoreQuery{}
		ref.resumeRequired = false
		return o
	})
	m.mustSet(o, "metadata", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(m.vm.NewTypeError("session().metadata requires key and value"))
		}
		key := strings.TrimSpace(call.Arguments[0].String())
		if key == "" {
			panic(m.vm.NewTypeError("session metadata key must not be empty"))
		}
		if ref.metadata == nil {
			ref.metadata = map[string]any{}
		}
		ref.metadata[key] = cloneJSONValue(call.Arguments[1].Export())
		return o
	})
	m.mustSet(o, "runDefaults", func(call goja.FunctionCall) goja.Value {
		opts, err := m.parseAgentRunOptions(ref.runDefaults, call.Arguments, 0)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		ref.runDefaults = opts
		return o
	})
	m.mustSet(o, "build", func(goja.FunctionCall) goja.Value {
		s, err := ref.build()
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.newAgentSessionObject(s)
	})
	return o
}

func newSessionBuilderFromAgent(agent *agentRef) *sessionBuilderRef {
	ref := &sessionBuilderRef{agent: agent}
	if agent != nil {
		ref.api = agent.api
		ref.persistMode = agent.persistMode
		ref.persister = agent.persister
		ref.runDefaults = agent.runDefaults
		if ts, ok := agent.persister.(*turnStoreRef); ok {
			ref.store = ts
		}
	}
	return ref
}

func (b *sessionBuilderRef) build() (*agentSessionRef, error) {
	if b == nil || b.api == nil || b.agent == nil {
		return nil, fmt.Errorf("session builder is not initialized")
	}
	if b.base != nil && b.resumeMode == resumeLatest {
		return nil, fmt.Errorf("session builder cannot use both base(turn) and resumeLatest()")
	}
	sessionID := strings.TrimSpace(b.id)
	if sessionID == "" {
		sessionID = uuid.NewString()
	}
	s := &agentSessionRef{
		api:         b.api,
		agent:       b.agent,
		sess:        gosession.NewSessionWithID(sessionID),
		name:        b.name,
		persistMode: b.persistMode,
		persister:   b.selectedPersister(),
		store:       b.selectedStore(),
		metadata:    cloneJSONMap(b.metadata),
		runDefaults: b.runDefaults,
		runtimeMeta: map[string]any{"agentName": b.agent.name, "sessionName": b.name},
	}
	if s.persister == nil && s.persistMode == persistInherit {
		s.persister = b.agent.selectedPersister()
	}
	if err := s.resumeIfRequested(b); err != nil {
		return nil, err
	}
	if b.base != nil {
		s.importBaseTurn(b.base, b.baseSource)
	}
	return s, nil
}

func (b *sessionBuilderRef) selectedPersister() enginebuilder.TurnPersister {
	if b == nil || b.agent == nil {
		return nil
	}
	switch b.persistMode {
	case persistInherit:
		return b.agent.selectedPersister()
	case persistDisabled:
		return nil
	case persistExplicit, persistUseDefault:
		return b.persister
	default:
		return b.agent.selectedPersister()
	}
}

func (b *sessionBuilderRef) selectedStore() *turnStoreRef {
	if b == nil {
		return nil
	}
	if b.persistMode == persistDisabled {
		return nil
	}
	if b.store != nil {
		return b.store
	}
	if ts, ok := b.persister.(*turnStoreRef); ok {
		return ts
	}
	if b.persistMode == persistInherit && b.agent != nil {
		if ts, ok := b.agent.persister.(*turnStoreRef); ok {
			return ts
		}
	}
	return nil
}

func (s *agentSessionRef) resumeIfRequested(b *sessionBuilderRef) error {
	if b == nil || b.resumeMode != resumeLatest {
		return nil
	}
	store := s.store
	if store == nil {
		store = s.api.defaultReadableTurnStoreRef()
	}
	if store == nil {
		return fmt.Errorf("resumeLatest requires a readable TurnStore")
	}
	q := b.resumeQuery
	if q.SessionID == "" && q.ConvID == "" {
		q.SessionID = s.sess.SessionID
	}
	if q.Phase == "" {
		q.Phase = "final"
	}
	snap, err := store.store.LoadLatestTurn(s.api.runtimeContext(), q)
	if err != nil {
		return err
	}
	if snap == nil || snap.Turn == nil {
		if b.resumeRequired {
			return fmt.Errorf("no stored turn found for session %q", s.sess.SessionID)
		}
		return nil
	}
	s.importBaseTurn(snap.Turn, "resume")
	return nil
}

func (s *agentSessionRef) importBaseTurn(base *turns.Turn, source string) {
	if s == nil || s.sess == nil || base == nil {
		return
	}
	clone := base.Clone()
	originSessionID, _, _ := turns.KeyTurnMetaSessionID.Get(clone.Metadata)
	originTurnID := clone.ID
	_ = turns.KeyTurnMetaSessionID.Set(&clone.Metadata, s.sess.SessionID)
	if source != "" && source != "resume" {
		_ = turns.TurnMetaKeyFromID[any](canonicalTurnMetaKey("forkedFromSource")).Set(&clone.Metadata, source)
		if originSessionID != "" {
			_ = turns.TurnMetaKeyFromID[any](canonicalTurnMetaKey("forkedFromSessionID")).Set(&clone.Metadata, originSessionID)
		}
		if originTurnID != "" {
			_ = turns.TurnMetaKeyFromID[any](canonicalTurnMetaKey("forkedFromTurnID")).Set(&clone.Metadata, originTurnID)
		}
		_ = turns.TurnMetaKeyFromID[any](canonicalTurnMetaKey("forkedAtMs")).Set(&clone.Metadata, time.Now().UnixMilli())
	}
	s.sess.Append(clone)
}

func (m *moduleRuntime) newAgentSessionObject(ref *agentSessionRef) *goja.Object {
	o := m.vm.NewObject()
	m.attachRef(o, ref)
	m.mustSet(o, "id", func(goja.FunctionCall) goja.Value { return m.vm.ToValue(ref.sess.SessionID) })
	m.mustSet(o, "name", func(goja.FunctionCall) goja.Value { return m.vm.ToValue(ref.name) })
	m.mustSet(o, "next", func(goja.FunctionCall) goja.Value {
		builder, err := ref.nextBuilder()
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.newSessionTurnBuilderObject(builder)
	})
	m.mustSet(o, "fork", func(call goja.FunctionCall) goja.Value {
		builder, err := ref.forkBuilder(call.Arguments)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.newSessionBuilderObject(builder)
	})
	m.mustSet(o, "latestTurn", func(goja.FunctionCall) goja.Value {
		latest := ref.sess.Latest()
		if latest == nil {
			return goja.Null()
		}
		return m.newTurnObject(&turnRef{api: m, turn: latest.Clone()})
	})
	m.mustSet(o, "turn", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(m.vm.NewTypeError("session.turn requires an index"))
		}
		idx := int(call.Arguments[0].ToInteger())
		turn := ref.sess.GetTurn(idx)
		if turn == nil {
			return goja.Null()
		}
		return m.newTurnObject(&turnRef{api: m, turn: turn.Clone()})
	})
	m.mustSet(o, "turns", func(goja.FunctionCall) goja.Value {
		snapshot := ref.sess.TurnsSnapshot()
		items := make([]any, 0, len(snapshot))
		for _, t := range snapshot {
			if t == nil {
				items = append(items, goja.Null())
				continue
			}
			items = append(items, m.newTurnObject(&turnRef{api: m, turn: t.Clone()}))
		}
		return m.toJSValue(items)
	})
	m.mustSet(o, "turnCount", func(goja.FunctionCall) goja.Value { return m.vm.ToValue(ref.sess.TurnCount()) })
	m.mustSet(o, "isRunning", func(goja.FunctionCall) goja.Value { return m.vm.ToValue(ref.sess.IsRunning()) })
	m.mustSet(o, "cancel", func(goja.FunctionCall) goja.Value {
		if err := ref.sess.CancelActive(); err != nil && err != gosession.ErrSessionNoActive {
			panic(m.vm.NewGoError(err))
		}
		return goja.Undefined()
	})
	m.mustSet(o, "close", func(goja.FunctionCall) goja.Value {
		ref.closed = true
		if err := ref.sess.CancelActive(); err != nil && err != gosession.ErrSessionNoActive {
			panic(m.vm.NewGoError(err))
		}
		return goja.Undefined()
	})
	return o
}

func (s *agentSessionRef) nextBuilder() (*sessionTurnBuilderRef, error) {
	if s == nil || s.sess == nil {
		return nil, fmt.Errorf("session is not initialized")
	}
	if s.closed {
		return nil, fmt.Errorf("session is closed")
	}
	if s.sess.IsRunning() {
		return nil, gosession.ErrSessionAlreadyActive
	}
	seed := &turns.Turn{}
	if latest := s.sess.Latest(); latest != nil {
		seed = latest.Clone()
		seed.ID = ""
	}
	_ = turns.KeyTurnMetaSessionID.Set(&seed.Metadata, s.sess.SessionID)
	for k, v := range s.metadata {
		_ = turns.TurnMetaKeyFromID[any](canonicalTurnMetaKey(k)).Set(&seed.Metadata, cloneJSONValue(v))
	}
	return &sessionTurnBuilderRef{api: s.api, session: s, turn: seed}, nil
}

func (s *agentSessionRef) forkBuilder(args []goja.Value) (*sessionBuilderRef, error) {
	if s == nil || s.sess == nil || s.agent == nil {
		return nil, fmt.Errorf("session is not initialized")
	}
	builder := newSessionBuilderFromAgent(s.agent)
	builder.name = s.name
	builder.persistMode = s.persistMode
	builder.persister = s.persister
	builder.store = s.store
	builder.metadata = cloneJSONMap(s.metadata)
	builder.runDefaults = s.runDefaults
	base, err := s.forkBase(args)
	if err != nil {
		return nil, err
	}
	builder.base = base
	builder.baseSource = "fork"
	return builder, nil
}

func (s *agentSessionRef) forkBase(args []goja.Value) (*turns.Turn, error) {
	if len(args) == 0 || args[0] == nil || goja.IsUndefined(args[0]) || goja.IsNull(args[0]) {
		latest := s.sess.Latest()
		if latest == nil {
			return nil, fmt.Errorf("cannot fork an empty session")
		}
		return latest.Clone(), nil
	}
	if turn, err := s.api.requireTurnRef(args[0]); err == nil {
		return turn.turn.Clone(), nil
	}
	raw := decodeMap(args[0].Export())
	if raw == nil {
		return nil, fmt.Errorf("session.fork options must be an object, TurnWrapper, null, or undefined")
	}
	at, ok := raw["at"]
	if !ok || at == nil {
		latest := s.sess.Latest()
		if latest == nil {
			return nil, fmt.Errorf("cannot fork an empty session")
		}
		return latest.Clone(), nil
	}
	if obj := args[0].ToObject(s.api.vm); obj != nil {
		atValue := obj.Get("at")
		if turn, err := s.api.requireTurnRef(atValue); err == nil {
			return turn.turn.Clone(), nil
		}
	}
	if idx, ok := numberAsInt(at); ok {
		turn := s.sess.GetTurn(idx)
		if turn == nil {
			return nil, fmt.Errorf("session.fork at index %d is out of range", idx)
		}
		return turn.Clone(), nil
	}
	return nil, fmt.Errorf("session.fork at must be a turn index or TurnWrapper")
}

func (m *moduleRuntime) newSessionTurnBuilderObject(ref *sessionTurnBuilderRef) *goja.Object {
	o := m.vm.NewObject()
	m.attachRef(o, ref)
	m.mustSet(o, "system", func(call goja.FunctionCall) goja.Value {
		text := ""
		if len(call.Arguments) > 0 {
			text = call.Arguments[0].String()
		}
		next := ref.cloneFor(m)
		turns.AppendBlock(next.turn, turns.NewSystemTextBlock(text))
		return m.newSessionTurnBuilderObject(next)
	})
	m.mustSet(o, "user", func(call goja.FunctionCall) goja.Value {
		next := ref.cloneFor(m)
		if len(call.Arguments) > 0 {
			if fn, ok := goja.AssertFunction(call.Arguments[0]); ok {
				msg := &messageBuilderRef{api: m}
				_, err := fn(goja.Undefined(), m.newMessageBuilderObject(msg))
				if err != nil {
					panic(err)
				}
				turns.AppendBlock(next.turn, turns.NewUserMultimodalBlock(msg.text, msg.images))
				return m.newSessionTurnBuilderObject(next)
			}
			turns.AppendBlock(next.turn, turns.NewUserTextBlock(call.Arguments[0].String()))
			return m.newSessionTurnBuilderObject(next)
		}
		turns.AppendBlock(next.turn, turns.NewUserTextBlock(""))
		return m.newSessionTurnBuilderObject(next)
	})
	m.mustSet(o, "assistant", func(call goja.FunctionCall) goja.Value {
		text := ""
		if len(call.Arguments) > 0 {
			text = call.Arguments[0].String()
		}
		next := ref.cloneFor(m)
		turns.AppendBlock(next.turn, turns.NewAssistantTextBlock(text))
		return m.newSessionTurnBuilderObject(next)
	})
	m.mustSet(o, "metadata", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(m.vm.NewTypeError("session.next().metadata requires key and value"))
		}
		key := strings.TrimSpace(call.Arguments[0].String())
		if key == "" {
			panic(m.vm.NewTypeError("session turn metadata key must not be empty"))
		}
		next := ref.cloneFor(m)
		_ = turns.TurnMetaKeyFromID[any](canonicalTurnMetaKey(key)).Set(&next.turn.Metadata, cloneJSONValue(call.Arguments[1].Export()))
		return m.newSessionTurnBuilderObject(next)
	})
	m.mustSet(o, "run", func(call goja.FunctionCall) goja.Value {
		opts, err := m.parseAgentRunOptions(ref.session.runDefaults, call.Arguments, 0)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		result, err := ref.session.runSync(ref.turn.Clone(), opts)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.newRunResultObject(result)
	})
	m.mustSet(o, "runAsync", func(call goja.FunctionCall) goja.Value {
		opts, err := m.parseAgentRunOptions(ref.session.runDefaults, call.Arguments, 0)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return ref.session.startAsync(ref.turn.Clone(), opts)
	})
	return o
}

func (r *sessionTurnBuilderRef) cloneFor(api *moduleRuntime) *sessionTurnBuilderRef {
	if r == nil {
		return &sessionTurnBuilderRef{api: api, turn: &turns.Turn{}}
	}
	turn := &turns.Turn{}
	if r.turn != nil {
		turn = r.turn.Clone()
	}
	return &sessionTurnBuilderRef{api: api, session: r.session, turn: turn}
}

func (s *agentSessionRef) configureBuilder(runScopedEventSinks []events.EventSink) error {
	builder, err := s.agent.newBuilderRef(runScopedEventSinks, s.selectedPersister()).buildEngineBuilder()
	if err != nil {
		return err
	}
	s.sess.Builder = builder
	return nil
}

func (s *agentSessionRef) selectedPersister() enginebuilder.TurnPersister {
	if s == nil || s.agent == nil {
		return nil
	}
	switch s.persistMode {
	case persistInherit:
		return s.agent.selectedPersister()
	case persistDisabled:
		return nil
	case persistExplicit, persistUseDefault:
		return s.persister
	default:
		return s.agent.selectedPersister()
	}
}

func (s *agentSessionRef) runSync(seed *turns.Turn, opts runOptions) (*runResultRef, error) {
	runScopedEventSinks, closers, err := s.agent.newRunScopedEventEmitterSinks()
	if err != nil {
		return nil, err
	}
	result, err := s.runBlockingOnOwner(seed, opts, runScopedEventSinks)
	if err != nil {
		closeRunScopedEventEmitterSinks(s.api.runtimeContext(), closers)
		return nil, err
	}
	s.agent.closeRunScopedEventEmitterSinksAfterOwnerQueue(closers)
	return result, nil
}

func (s *agentSessionRef) runBlockingOnOwner(seed *turns.Turn, opts runOptions, runScopedEventSinks []events.EventSink) (*runResultRef, error) {
	if s == nil || s.sess == nil {
		return nil, fmt.Errorf("session is not initialized")
	}
	if s.closed {
		return nil, fmt.Errorf("session is closed")
	}
	if seed == nil || len(seed.Blocks) == 0 {
		return nil, gosession.ErrSessionEmptyTurn
	}
	if s.sess.IsRunning() {
		return nil, gosession.ErrSessionAlreadyActive
	}
	if err := s.configureBuilder(runScopedEventSinks); err != nil {
		return nil, err
	}
	inputSnapshot := seed.Clone()
	stampTurnRuntimeMetadata(seed, s.runtimeMeta)
	effective := seed.Clone()
	s.sess.Append(seed)
	sr := &sessionRef{api: s.api, session: s.sess, runtimeMetadata: cloneJSONMap(s.runtimeMeta)}
	ctx, cancel, err := sr.buildRunContext(opts)
	if err != nil {
		return nil, err
	}
	defer cancel()
	if s.api != nil && s.api.runtimeOwner != nil {
		ctx = runtimeowner.OwnerContext(s.api.runtimeOwner, ctx)
	}
	if seed.ID == "" {
		seed.ID = uuid.NewString()
	}
	inferenceID := uuid.NewString()
	_ = turns.KeyTurnMetaSessionID.Set(&seed.Metadata, s.sess.SessionID)
	_ = turns.KeyTurnMetaInferenceID.Set(&seed.Metadata, inferenceID)
	runner, err := s.sess.Builder.Build(ctx, s.sess.SessionID)
	if err != nil {
		return nil, err
	}
	runCtx := gosession.WithSessionMeta(ctx, s.sess.SessionID, inferenceID)
	out, err := runner.RunInference(runCtx, seed)
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = seed
	}
	if out.ID == "" {
		out.ID = seed.ID
	}
	_ = turns.KeyTurnMetaSessionID.Set(&out.Metadata, s.sess.SessionID)
	_ = turns.KeyTurnMetaInferenceID.Set(&out.Metadata, inferenceID)
	if out != seed {
		*seed = *out
		out = seed
	}
	output, err := cloneRunOutput(out)
	if err != nil {
		return nil, err
	}
	return &runResultRef{api: s.api, inputTurn: inputSnapshot, effectiveTurn: effective, outputTurn: output}, nil
}

func (s *agentSessionRef) startAsync(seed *turns.Turn, opts runOptions) goja.Value {
	if _, err := s.api.requireBridge("session.runAsync"); err != nil {
		panic(s.api.vm.NewTypeError(err.Error()))
	}
	promise, resolve, reject := s.api.vm.NewPromise()
	handleObj := s.api.vm.NewObject()

	var activeHandle *gosession.ExecutionHandle
	var activeCancel context.CancelFunc
	cancelActive := func() {
		if activeHandle != nil {
			activeHandle.Cancel()
		}
		if activeCancel != nil {
			activeCancel()
		}
	}

	s.api.mustSet(handleObj, "promise", promise)
	s.api.mustSet(handleObj, "cancel", func(goja.FunctionCall) goja.Value {
		cancelActive()
		return goja.Undefined()
	})
	s.api.mustSet(handleObj, "close", func(goja.FunctionCall) goja.Value {
		cancelActive()
		return goja.Undefined()
	})

	started, err := s.startRun(seed, opts)
	if err != nil {
		s.rejectPromiseWithError(reject, err)
		return handleObj
	}
	activeHandle = started.handle
	activeCancel = started.cancel

	go func() {
		defer started.cancel()
		out, waitErr := started.handle.Wait()
		postErr := s.api.postOnOwner(s.api.runtimeContext(), "session.runAsync.settle", func(ctx context.Context) {
			defer closeRunScopedEventEmitterSinks(ctx, started.runScopedSinks)
			if waitErr != nil {
				s.rejectPromiseWithError(reject, waitErr)
				return
			}
			output, err := cloneRunOutput(out)
			if err != nil {
				s.rejectPromiseWithError(reject, err)
				return
			}
			_ = resolve(s.api.newRunResultObject(&runResultRef{api: s.api, inputTurn: started.inputSnapshot, effectiveTurn: started.effectiveTurn, outputTurn: output}))
		})
		if postErr != nil {
			closeRunScopedEventEmitterSinks(s.api.runtimeContext(), started.runScopedSinks)
			s.api.logger.Error().Err(postErr).Msg("session.runAsync: failed to settle promise on owner thread")
		}
	}()
	return handleObj
}

func (s *agentSessionRef) startRun(seed *turns.Turn, opts runOptions) (*startedSessionRun, error) {
	if s == nil || s.sess == nil {
		return nil, fmt.Errorf("session is not initialized")
	}
	if s.closed {
		return nil, fmt.Errorf("session is closed")
	}
	if seed == nil || len(seed.Blocks) == 0 {
		return nil, gosession.ErrSessionEmptyTurn
	}
	runScopedEventSinks, closers, err := s.agent.newRunScopedEventEmitterSinks()
	if err != nil {
		return nil, err
	}
	if err := s.configureBuilder(runScopedEventSinks); err != nil {
		closeRunScopedEventEmitterSinks(s.api.runtimeContext(), closers)
		return nil, err
	}
	inputSnapshot := seed.Clone()
	stampTurnRuntimeMetadata(seed, s.runtimeMeta)
	effective := seed.Clone()
	s.sess.Append(seed)
	sr := &sessionRef{api: s.api, session: s.sess, runtimeMetadata: cloneJSONMap(s.runtimeMeta)}
	ctx, cancel, err := sr.buildRunContext(opts)
	if err != nil {
		closeRunScopedEventEmitterSinks(s.api.runtimeContext(), closers)
		return nil, err
	}
	handle, err := s.sess.StartInference(ctx)
	if err != nil {
		cancel()
		closeRunScopedEventEmitterSinks(s.api.runtimeContext(), closers)
		return nil, err
	}
	return &startedSessionRun{handle: handle, inputSnapshot: inputSnapshot, effectiveTurn: effective, cancel: cancel, runScopedSinks: closers}, nil
}

func (s *agentSessionRef) rejectPromiseWithError(reject func(reason any) error, err error) {
	if s == nil || s.api == nil || reject == nil {
		return
	}
	if err == nil {
		err = fmt.Errorf("session run failed")
	}
	_ = reject(s.api.vm.NewGoError(err))
}

func (m *moduleRuntime) parseResumeLatestOptions(args []goja.Value, idx int) (TurnStoreQuery, bool, error) {
	q, err := m.parseTurnStoreQuery(args, idx)
	if err != nil {
		return TurnStoreQuery{}, false, err
	}
	if len(args) <= idx || args[idx] == nil || goja.IsUndefined(args[idx]) || goja.IsNull(args[idx]) {
		return q, false, nil
	}
	raw := decodeMap(args[idx].Export())
	if raw == nil {
		return q, false, fmt.Errorf("resumeLatest options must be an object")
	}
	return q, toBool(raw["required"], false), nil
}

func (m *moduleRuntime) defaultReadableTurnStoreRef() *turnStoreRef {
	if m == nil {
		return nil
	}
	if m.defaultTurnStore != nil {
		return &turnStoreRef{api: m, name: "default", store: m.defaultTurnStore}
	}
	if store := m.turnStores["default"]; store != nil {
		return &turnStoreRef{api: m, name: "default", store: store}
	}
	return nil
}

func numberAsInt(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int64:
		return int(x), true
	case float64:
		return int(x), true
	case float32:
		return int(x), true
	default:
		return 0, false
	}
}
