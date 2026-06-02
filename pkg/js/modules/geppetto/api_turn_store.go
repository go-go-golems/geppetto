package geppetto

import (
	"context"
	"fmt"
	"strings"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

// TurnStore is the host-facing storage capability exposed to JavaScript as a
// Go-owned wrapper. Concrete storage (SQLite, remote stores, etc.) remains
// host-owned; Geppetto only requires final-turn persistence plus readback
// operations useful to JS scripts.
type TurnStore interface {
	PersistTurn(ctx context.Context, t *turns.Turn) error
	ListTurns(ctx context.Context, q TurnStoreQuery) ([]TurnStoreSnapshot, error)
	LoadLatestTurn(ctx context.Context, q TurnStoreQuery) (*TurnStoreSnapshot, error)
	Close() error
}

// StorageOptions groups host-provided turn-store capabilities for xgoja
// provider integration.
type StorageOptions struct {
	Default TurnStore
	Stores  map[string]TurnStore
}

// TurnStoreQuery is the storage query shape accepted from JavaScript.
type TurnStoreQuery struct {
	ConvID    string
	SessionID string
	Phase     string
	SinceMs   int64
	Limit     int
}

// TurnStoreSnapshot is a storage readback row. Turn is optional so host
// adapters can expose metadata even when payload decoding fails or is deferred.
type TurnStoreSnapshot struct {
	ConvID      string
	SessionID   string
	TurnID      string
	Phase       string
	RuntimeKey  string
	InferenceID string
	CreatedAtMs int64
	Turn        *turns.Turn
}

type turnStoreRef struct {
	api   *moduleRuntime
	name  string
	store TurnStore
}

func (r *turnStoreRef) PersistTurn(ctx context.Context, t *turns.Turn) error {
	if r == nil || r.store == nil {
		return fmt.Errorf("turn store is not configured")
	}
	return r.store.PersistTurn(ctx, t)
}

func (m *moduleRuntime) installTurnStoresNamespace(exports *goja.Object) {
	ns := m.vm.NewObject()
	m.mustSet(ns, "default", func(goja.FunctionCall) goja.Value {
		if m.defaultTurnStore == nil {
			panic(m.vm.NewGoError(fmt.Errorf("geppetto turnStores.default is not configured")))
		}
		return m.newTurnStoreObject(&turnStoreRef{api: m, name: "default", store: m.defaultTurnStore})
	})
	m.mustSet(ns, "get", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
			panic(m.vm.NewTypeError("turnStores.get requires a store name"))
		}
		name := strings.TrimSpace(call.Arguments[0].String())
		if name == "" {
			panic(m.vm.NewTypeError("turnStores.get store name must not be empty"))
		}
		store := m.turnStores[name]
		if store == nil {
			panic(m.vm.NewGoError(fmt.Errorf("geppetto turn store %q is not configured", name)))
		}
		return m.newTurnStoreObject(&turnStoreRef{api: m, name: name, store: store})
	})
	m.mustSet(exports, "turnStores", ns)
}

func (m *moduleRuntime) newTurnStoreObject(ref *turnStoreRef) *goja.Object {
	if ref == nil {
		ref = &turnStoreRef{}
	}
	ref.api = m
	o := m.vm.NewObject()
	m.attachRef(o, ref)
	m.mustSet(o, "name", func(goja.FunctionCall) goja.Value { return m.vm.ToValue(ref.name) })
	m.mustSet(o, "list", func(call goja.FunctionCall) goja.Value {
		q, err := m.parseTurnStoreQuery(call.Arguments, 0)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		if ref.store == nil {
			panic(m.vm.NewGoError(fmt.Errorf("turn store %q is not configured", ref.name)))
		}
		items, err := ref.store.ListTurns(m.runtimeContext(), q)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		out := make([]any, 0, len(items))
		for i := range items {
			out = append(out, m.newTurnStoreSnapshotObject(items[i]))
		}
		return m.toJSValue(out)
	})
	m.mustSet(o, "loadLatest", func(call goja.FunctionCall) goja.Value {
		q, err := m.parseTurnStoreQuery(call.Arguments, 0)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		if ref.store == nil {
			panic(m.vm.NewGoError(fmt.Errorf("turn store %q is not configured", ref.name)))
		}
		item, err := ref.store.LoadLatestTurn(m.runtimeContext(), q)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		if item == nil {
			return goja.Null()
		}
		return m.newTurnStoreSnapshotObject(*item)
	})
	m.mustSet(o, "close", func(goja.FunctionCall) goja.Value {
		if ref.store == nil {
			return goja.Undefined()
		}
		if err := ref.store.Close(); err != nil {
			panic(m.vm.NewGoError(err))
		}
		return goja.Undefined()
	})
	return o
}

func (m *moduleRuntime) newTurnStoreSnapshotObject(s TurnStoreSnapshot) *goja.Object {
	o := m.vm.NewObject()
	m.mustSet(o, "convId", s.ConvID)
	m.mustSet(o, "sessionId", s.SessionID)
	m.mustSet(o, "turnId", s.TurnID)
	m.mustSet(o, "phase", s.Phase)
	m.mustSet(o, "runtimeKey", s.RuntimeKey)
	m.mustSet(o, "inferenceId", s.InferenceID)
	m.mustSet(o, "createdAtMs", s.CreatedAtMs)
	if s.Turn != nil {
		m.mustSet(o, "turn", m.newTurnObject(&turnRef{api: m, turn: s.Turn.Clone()}))
	} else {
		m.mustSet(o, "turn", goja.Null())
	}
	m.mustSet(o, "toJSON", func(goja.FunctionCall) goja.Value {
		var turnValue any
		if s.Turn != nil {
			turnValue = m.encodeTurn(s.Turn)
		}
		return m.toJSValue(map[string]any{
			"convId":      s.ConvID,
			"sessionId":   s.SessionID,
			"turnId":      s.TurnID,
			"phase":       s.Phase,
			"runtimeKey":  s.RuntimeKey,
			"inferenceId": s.InferenceID,
			"createdAtMs": s.CreatedAtMs,
			"turn":        turnValue,
		})
	})
	return o
}

func (m *moduleRuntime) requireTurnStoreRef(v goja.Value) (*turnStoreRef, error) {
	ref := m.getRef(v)
	store, ok := ref.(*turnStoreRef)
	if !ok || store == nil || store.store == nil {
		return nil, fmt.Errorf("expected Go-owned TurnStore wrapper, got %T (value: %v)", ref, v)
	}
	store.api = m
	return store, nil
}

func (m *moduleRuntime) defaultTurnPersister() (*turnStoreRef, error) {
	if m.defaultTurnStore != nil {
		return &turnStoreRef{api: m, name: "default", store: m.defaultTurnStore}, nil
	}
	if store := m.turnStores["default"]; store != nil {
		return &turnStoreRef{api: m, name: "default", store: store}, nil
	}
	if m.defaultPersister != nil {
		return &turnStoreRef{api: m, name: "default", store: persisterOnlyTurnStore{persister: m.defaultPersister}}, nil
	}
	return nil, fmt.Errorf("geppetto default turn persistence is not configured")
}

type persisterOnlyTurnStore struct {
	persister interface {
		PersistTurn(context.Context, *turns.Turn) error
	}
}

func (s persisterOnlyTurnStore) PersistTurn(ctx context.Context, t *turns.Turn) error {
	return s.persister.PersistTurn(ctx, t)
}

func (persisterOnlyTurnStore) ListTurns(context.Context, TurnStoreQuery) ([]TurnStoreSnapshot, error) {
	return nil, fmt.Errorf("default persister does not support turn store reads")
}

func (persisterOnlyTurnStore) LoadLatestTurn(context.Context, TurnStoreQuery) (*TurnStoreSnapshot, error) {
	return nil, fmt.Errorf("default persister does not support turn store reads")
}

func (persisterOnlyTurnStore) Close() error { return nil }

func (m *moduleRuntime) parseTurnStoreQuery(args []goja.Value, idx int) (TurnStoreQuery, error) {
	q := TurnStoreQuery{}
	if len(args) <= idx || args[idx] == nil || goja.IsUndefined(args[idx]) || goja.IsNull(args[idx]) {
		return q, nil
	}
	raw := decodeMap(args[idx].Export())
	if raw == nil {
		return q, fmt.Errorf("turn store query must be an object")
	}
	q.ConvID = decodeString(raw["convId"])
	if q.ConvID == "" {
		q.ConvID = decodeString(raw["conversationId"])
	}
	q.SessionID = decodeString(raw["sessionId"])
	if q.SessionID == "" {
		q.SessionID = decodeString(raw["sessionID"])
	}
	q.Phase = decodeString(raw["phase"])
	q.SinceMs = int64(toInt(raw["sinceMs"], 0))
	q.Limit = toInt(raw["limit"], 0)
	return q, nil
}
