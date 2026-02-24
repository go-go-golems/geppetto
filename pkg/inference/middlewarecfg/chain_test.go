package middlewarecfg

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	gepmiddleware "github.com/go-go-golems/geppetto/pkg/inference/middleware"
	gepprofiles "github.com/go-go-golems/geppetto/pkg/profiles"
)

type chainTestDefinition struct {
	name    string
	buildFn func(context.Context, BuildDeps, any) (gepmiddleware.Middleware, error)
}

func (d *chainTestDefinition) Name() string {
	return d.name
}

func (d *chainTestDefinition) ConfigJSONSchema() map[string]any {
	return map[string]any{"type": "object"}
}

func (d *chainTestDefinition) Build(ctx context.Context, deps BuildDeps, cfg any) (gepmiddleware.Middleware, error) {
	if d.buildFn == nil {
		return func(next gepmiddleware.HandlerFunc) gepmiddleware.HandlerFunc { return next }, nil
	}
	return d.buildFn(ctx, deps, cfg)
}

func TestBuildChain_SkipsDisabledAndPreservesInputOrder(t *testing.T) {
	disabled := false
	calls := make([]string, 0, 2)
	configs := make([]map[string]any, 0, 2)

	defA := &chainTestDefinition{
		name: "a",
		buildFn: func(_ context.Context, _ BuildDeps, cfg any) (gepmiddleware.Middleware, error) {
			calls = append(calls, "a#first")
			m, _ := cfg.(map[string]any)
			configs = append(configs, m)
			return func(next gepmiddleware.HandlerFunc) gepmiddleware.HandlerFunc { return next }, nil
		},
	}
	defB := &chainTestDefinition{
		name: "b",
		buildFn: func(_ context.Context, _ BuildDeps, cfg any) (gepmiddleware.Middleware, error) {
			calls = append(calls, "b#skip")
			m, _ := cfg.(map[string]any)
			configs = append(configs, m)
			return func(next gepmiddleware.HandlerFunc) gepmiddleware.HandlerFunc { return next }, nil
		},
	}
	defC := &chainTestDefinition{
		name: "c",
		buildFn: func(_ context.Context, _ BuildDeps, cfg any) (gepmiddleware.Middleware, error) {
			calls = append(calls, "c#third")
			m, _ := cfg.(map[string]any)
			configs = append(configs, m)
			return func(next gepmiddleware.HandlerFunc) gepmiddleware.HandlerFunc { return next }, nil
		},
	}

	chain, err := BuildChain(context.Background(), BuildDeps{}, []ResolvedInstance{
		{
			Use: gepprofiles.MiddlewareUse{Name: "a", ID: "first"},
			Def: defA,
			Resolved: &ResolvedConfig{
				Config: map[string]any{"value": int64(1)},
			},
		},
		{
			Use: gepprofiles.MiddlewareUse{Name: "b", ID: "skip", Enabled: &disabled},
			Def: defB,
			Resolved: &ResolvedConfig{
				Config: map[string]any{"value": int64(2)},
			},
		},
		{
			Use: gepprofiles.MiddlewareUse{Name: "c", ID: "third"},
			Def: defC,
			Resolved: &ResolvedConfig{
				Config: map[string]any{"value": int64(3)},
			},
		},
	})
	if err != nil {
		t.Fatalf("BuildChain returned error: %v", err)
	}

	if got, want := len(chain), 2; got != want {
		t.Fatalf("chain length mismatch: got=%d want=%d", got, want)
	}

	wantCalls := []string{"a#first", "c#third"}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("build order mismatch: got=%v want=%v", calls, wantCalls)
	}

	wantConfigs := []map[string]any{
		{"value": int64(1)},
		{"value": int64(3)},
	}
	if !reflect.DeepEqual(configs, wantConfigs) {
		t.Fatalf("config pass-through mismatch: got=%v want=%v", configs, wantConfigs)
	}
}

func TestBuildChain_BuildErrorIncludesInstanceKey(t *testing.T) {
	def := &chainTestDefinition{
		name: "agentmode",
		buildFn: func(context.Context, BuildDeps, any) (gepmiddleware.Middleware, error) {
			return nil, errors.New("boom")
		},
	}

	_, err := BuildChain(context.Background(), BuildDeps{}, []ResolvedInstance{
		{
			Use: gepprofiles.MiddlewareUse{Name: "agentmode", ID: "primary"},
			Def: def,
			Resolved: &ResolvedConfig{
				Config: map[string]any{"mode": "safe"},
			},
		},
	})
	if err == nil {
		t.Fatalf("expected build error")
	}
	if !strings.Contains(err.Error(), "agentmode#primary") {
		t.Fatalf("expected instance key in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected wrapped build cause, got: %v", err)
	}
}

func TestBuildChain_RepeatedMiddlewareNamesWithUniqueIDs(t *testing.T) {
	received := make([]string, 0, 2)
	def := &chainTestDefinition{
		name: "agentmode",
		buildFn: func(_ context.Context, _ BuildDeps, cfg any) (gepmiddleware.Middleware, error) {
			m, _ := cfg.(map[string]any)
			mode, _ := m["mode"].(string)
			received = append(received, mode)
			return func(next gepmiddleware.HandlerFunc) gepmiddleware.HandlerFunc { return next }, nil
		},
	}

	chain, err := BuildChain(context.Background(), BuildDeps{}, []ResolvedInstance{
		{
			Use: gepprofiles.MiddlewareUse{Name: "agentmode", ID: "safe"},
			Def: def,
			Resolved: &ResolvedConfig{
				Config: map[string]any{"mode": "safe"},
			},
		},
		{
			Use: gepprofiles.MiddlewareUse{Name: "agentmode", ID: "aggressive"},
			Def: def,
			Resolved: &ResolvedConfig{
				Config: map[string]any{"mode": "aggressive"},
			},
		},
	})
	if err != nil {
		t.Fatalf("BuildChain returned error: %v", err)
	}
	if got, want := len(chain), 2; got != want {
		t.Fatalf("chain length mismatch: got=%d want=%d", got, want)
	}
	if got, want := received, []string{"safe", "aggressive"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("received mode ordering mismatch: got=%v want=%v", got, want)
	}
}

func TestMiddlewareInstanceKey_UsesIDWhenPresent(t *testing.T) {
	got := MiddlewareInstanceKey(gepprofiles.MiddlewareUse{Name: "agentmode", ID: "primary"}, 4)
	if got != "agentmode#primary" {
		t.Fatalf("instance key mismatch: got=%q want=%q", got, "agentmode#primary")
	}

	got = MiddlewareInstanceKey(gepprofiles.MiddlewareUse{Name: "agentmode"}, 4)
	if got != "agentmode[4]" {
		t.Fatalf("instance key mismatch: got=%q want=%q", got, "agentmode[4]")
	}
}
