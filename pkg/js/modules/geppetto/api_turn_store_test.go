package geppetto

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/dop251/goja"
	inferenceengine "github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

type turnStoreTestEngine struct{}

var _ inferenceengine.Engine = (*turnStoreTestEngine)(nil)

func (e *turnStoreTestEngine) RunInference(_ context.Context, t *turns.Turn) (*turns.Turn, error) {
	out := &turns.Turn{}
	if t != nil {
		out = t.Clone()
	}
	turns.AppendBlock(out, turns.NewAssistantTextBlock("stored answer"))
	return out, nil
}

type recordingTurnStore struct {
	mu      sync.Mutex
	name    string
	closed  bool
	persist []*turns.Turn
}

var _ TurnStore = (*recordingTurnStore)(nil)

func (s *recordingTurnStore) PersistTurn(_ context.Context, t *turns.Turn) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return fmt.Errorf("store %s is closed", s.name)
	}
	s.persist = append(s.persist, t.Clone())
	return nil
}

func (s *recordingTurnStore) ListTurns(_ context.Context, q TurnStoreQuery) ([]TurnStoreSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	limit := q.Limit
	if limit <= 0 || limit > len(s.persist) {
		limit = len(s.persist)
	}
	out := make([]TurnStoreSnapshot, 0, limit)
	start := len(s.persist) - limit
	for _, t := range s.persist[start:] {
		out = append(out, snapshotFromTurn(t, q.Phase))
	}
	return out, nil
}

func (s *recordingTurnStore) LoadLatestTurn(_ context.Context, q TurnStoreQuery) (*TurnStoreSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.persist) == 0 {
		return nil, nil
	}
	snap := snapshotFromTurn(s.persist[len(s.persist)-1], q.Phase)
	return &snap, nil
}

func (s *recordingTurnStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

func (s *recordingTurnStore) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.persist)
}

func snapshotFromTurn(t *turns.Turn, phase string) TurnStoreSnapshot {
	if phase == "" {
		phase = "final"
	}
	var sessionID, inferenceID string
	if t != nil {
		if v, ok, err := turns.KeyTurnMetaSessionID.Get(t.Metadata); err == nil && ok {
			sessionID = v
		}
		if v, ok, err := turns.KeyTurnMetaInferenceID.Get(t.Metadata); err == nil && ok {
			inferenceID = v
		}
	}
	return TurnStoreSnapshot{
		ConvID:      sessionID,
		SessionID:   sessionID,
		TurnID:      t.ID,
		Phase:       phase,
		RuntimeKey:  "test-runtime",
		InferenceID: inferenceID,
		CreatedAtMs: time.Now().UnixMilli(),
		Turn:        t.Clone(),
	}
}

func TestTurnStoresNamespaceReturnsConfiguredStores(t *testing.T) {
	defaultStore := &recordingTurnStore{name: "default"}
	namedStore := &recordingTurnStore{name: "durable"}
	rt := newJSRuntime(t, Options{
		EnableStorage:    true,
		DefaultTurnStore: defaultStore,
		TurnStores: map[string]TurnStore{
			"durable": namedStore,
		},
	})
	got := mustEvalExprExport(t, rt, `(() => {
		const gp = require("geppetto");
		return JSON.stringify({
			top: Object.keys(gp.turnStores).sort(),
			defaultName: gp.turnStores.default().name(),
			durableName: gp.turnStores.get("durable").name(),
		});
	})()`)
	want := `{"top":["default","get"],"defaultName":"default","durableName":"durable"}`
	if got != want {
		t.Fatalf("turnStores namespace = %v, want %s", got, want)
	}
}

func TestTurnStoresDefaultMissingErrors(t *testing.T) {
	rt := newJSRuntime(t, Options{})
	_, err := rt.runtimeOwner.Call(context.Background(), "test.missingDefaultTurnStore", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, runErr := vm.RunString(`require("geppetto").turnStores.default()`)
		return nil, runErr
	})
	if err == nil {
		t.Fatalf("expected missing default turn store error")
	}
}

func TestAgentPersistToStoreAndReadLatest(t *testing.T) {
	store := &recordingTurnStore{name: "durable"}
	rt := newJSRuntime(t, Options{EnableStorage: true, TurnStores: map[string]TurnStore{"durable": store}})
	_, err := rt.runtimeOwner.Call(context.Background(), "test.persistToStore", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if setErr := vm.Set("fakeEngine", &turnStoreTestEngine{}); setErr != nil {
			return nil, setErr
		}
		_, runErr := vm.RunString(`
			const gp = require("geppetto");
			const store = gp.turnStores.get("durable");
			const agent = gp.agent().engine(globalThis.fakeEngine).store(store).build();
			const session = agent.session().id("persist-store-test").build();
			const result = session.next().user("persist me").run();
			const latest = store.loadLatest({ phase: "final" });
			globalThis.persistReadback = JSON.stringify({
				text: result.text(),
				latestText: latest.turn.toJSON().blocks[latest.turn.toJSON().blocks.length - 1].payload.text,
				turnId: latest.turnId.length > 0,
				sessionId: latest.sessionId.length > 0,
			});
		`)
		return nil, runErr
	})
	if err != nil {
		t.Fatalf("persistTo script failed: %v", err)
	}
	if store.count() != 1 {
		t.Fatalf("persist count = %d, want 1", store.count())
	}
	got := mustEvalExprExport(t, rt, `globalThis.persistReadback`)
	want := `{"text":"stored answer","latestText":"stored answer","turnId":true,"sessionId":true}`
	if got != want {
		t.Fatalf("readback = %v, want %s", got, want)
	}
}

func TestAgentPersistToRejectsPlainObjects(t *testing.T) {
	rt := newJSRuntime(t, Options{})
	_, err := rt.runtimeOwner.Call(context.Background(), "test.persistToRejectsPlainObjects", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if setErr := vm.Set("fakeEngine", &turnStoreTestEngine{}); setErr != nil {
			return nil, setErr
		}
		_, runErr := vm.RunString(`require("geppetto").agent().engine(globalThis.fakeEngine).persistTo({})`)
		return nil, runErr
	})
	if err == nil {
		t.Fatalf("expected persistTo({}) to reject plain objects")
	}
}

func TestAgentPersistToNullDisablesHostDefaultPersister(t *testing.T) {
	store := &recordingTurnStore{name: "default"}
	rt := newJSRuntime(t, Options{DefaultPersister: store})
	_, err := rt.runtimeOwner.Call(context.Background(), "test.persistToNull", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if setErr := vm.Set("fakeEngine", &turnStoreTestEngine{}); setErr != nil {
			return nil, setErr
		}
		_, runErr := vm.RunString(`
			const gp = require("geppetto");
			const agent = gp.agent().engine(globalThis.fakeEngine).persistTo(null).build();
			const session = agent.session().id("persist-null-test").build();
			session.next().user("do not persist").run();
		`)
		return nil, runErr
	})
	if err != nil {
		t.Fatalf("persistTo(null) script failed: %v", err)
	}
	if store.count() != 0 {
		t.Fatalf("default persister count = %d, want 0", store.count())
	}
}

func TestAgentRunAsyncPersistsToStore(t *testing.T) {
	store := &recordingTurnStore{name: "durable"}
	rt := newJSRuntime(t, Options{EnableStorage: true, TurnStores: map[string]TurnStore{"durable": store}})
	_, err := rt.runtimeOwner.Call(context.Background(), "test.runAsyncPersistToStore", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if setErr := vm.Set("fakeEngine", &turnStoreTestEngine{}); setErr != nil {
			return nil, setErr
		}
		_, runErr := vm.RunString(`
			const gp = require("geppetto");
			globalThis.asyncPersistDone = false;
			const store = gp.turnStores.get("durable");
			const agent = gp.agent().engine(globalThis.fakeEngine).store(store).build();
			const session = agent.session().id("persist-async-test").build();
			session.next().user("persist async").runAsync().promise.then(
				result => { globalThis.asyncPersistDone = true; globalThis.asyncPersistText = result.text(); },
				err => { globalThis.asyncPersistDone = true; globalThis.asyncPersistError = String(err); },
			);
		`)
		return nil, runErr
	})
	if err != nil {
		t.Fatalf("runAsync persist script failed: %v", err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if store.count() == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if store.count() != 1 {
		t.Fatalf("runAsync persist count = %d, want 1", store.count())
	}
	got := mustEvalExprExport(t, rt, `JSON.stringify({done: globalThis.asyncPersistDone, text: globalThis.asyncPersistText, error: globalThis.asyncPersistError})`)
	want := `{"done":true,"text":"stored answer"}`
	if got != want {
		t.Fatalf("runAsync state = %v, want %s", got, want)
	}
}
