package middlewarecfg

import (
	"context"
	"testing"

	gepmiddleware "github.com/go-go-golems/geppetto/pkg/inference/middleware"
)

type testDefinition struct {
	name string
}

func (d *testDefinition) Name() string {
	return d.name
}

func (d *testDefinition) ConfigJSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
	}
}

func (d *testDefinition) Build(context.Context, BuildDeps, any) (gepmiddleware.Middleware, error) {
	return nil, nil
}

func TestInMemoryDefinitionRegistry_RegisterAndLookup(t *testing.T) {
	reg := NewInMemoryDefinitionRegistry()
	def := &testDefinition{name: "agentmode"}

	if err := reg.RegisterDefinition(def); err != nil {
		t.Fatalf("RegisterDefinition returned error: %v", err)
	}

	got, ok := reg.GetDefinition("agentmode")
	if !ok {
		t.Fatalf("expected definition to be found")
	}
	if got.Name() != "agentmode" {
		t.Fatalf("definition name mismatch: got=%q want=%q", got.Name(), "agentmode")
	}
}

func TestInMemoryDefinitionRegistry_DuplicateRegistrationRejected(t *testing.T) {
	reg := NewInMemoryDefinitionRegistry()
	if err := reg.RegisterDefinition(&testDefinition{name: "agentmode"}); err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}
	err := reg.RegisterDefinition(&testDefinition{name: "agentmode"})
	if err == nil {
		t.Fatalf("expected duplicate registration error")
	}
}

func TestInMemoryDefinitionRegistry_ListDefinitionsDeterministic(t *testing.T) {
	reg := NewInMemoryDefinitionRegistry()
	for _, def := range []Definition{
		&testDefinition{name: "zeta"},
		&testDefinition{name: "alpha"},
		&testDefinition{name: "beta"},
	} {
		if err := reg.RegisterDefinition(def); err != nil {
			t.Fatalf("RegisterDefinition returned error: %v", err)
		}
	}

	list := reg.ListDefinitions()
	if len(list) != 3 {
		t.Fatalf("expected 3 definitions, got %d", len(list))
	}

	got := []string{list[0].Name(), list[1].Name(), list[2].Name()}
	want := []string{"alpha", "beta", "zeta"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sorted names mismatch at index %d: got=%q want=%q", i, got[i], want[i])
		}
	}
}

func TestInMemoryDefinitionRegistry_RejectsNilAndEmptyNames(t *testing.T) {
	reg := NewInMemoryDefinitionRegistry()

	if err := reg.RegisterDefinition(nil); err == nil {
		t.Fatalf("expected nil definition error")
	}

	if err := reg.RegisterDefinition(&testDefinition{name: "  "}); err == nil {
		t.Fatalf("expected empty name error")
	}

	var typedNil *testDefinition
	if err := reg.RegisterDefinition(typedNil); err == nil {
		t.Fatalf("expected typed nil definition error")
	}
}
