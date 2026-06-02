package geppetto

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

func newTurnBuilderTestRuntime(t *testing.T) (*goja.Runtime, *moduleRuntime) {
	t.Helper()
	vm := goja.New()
	mod := newRuntime(vm, Options{})
	exports := vm.NewObject()
	mod.installExports(exports)
	if err := vm.Set("gp", exports); err != nil {
		t.Fatalf("set gp: %v", err)
	}
	return vm, mod
}

func TestTurnBuilderCanContinueFromExistingTurn(t *testing.T) {
	vm, _ := newTurnBuilderTestRuntime(t)
	v, err := vm.RunString(`
		const base = gp.turn()
			.system("Be brief.")
			.user("First question")
			.build();
		const next = gp.turn(base)
			.assistant("First answer")
			.user("Second question")
			.build();
		JSON.stringify({
			baseBlocks: base.toJSON().blocks.map(b => b.role + ":" + b.payload.text),
			nextBlocks: next.toJSON().blocks.map(b => b.role + ":" + b.payload.text),
		});
	`)
	if err != nil {
		t.Fatalf("run JS: %v", err)
	}
	want := `{"baseBlocks":["system:Be brief.","user:First question"],"nextBlocks":["system:Be brief.","user:First question","assistant:First answer","user:Second question"]}`
	if got := v.String(); got != want {
		t.Fatalf("continued turn = %s, want %s", got, want)
	}
}

func TestTurnBuilderFromExistingTurnClearsIDButClonePreservesID(t *testing.T) {
	vm, mod := newTurnBuilderTestRuntime(t)
	base := &turns.Turn{ID: "turn-original"}
	turns.AppendBlock(base, turns.NewUserTextBlock("hello"))
	if err := vm.Set("base", mod.newTurnObject(&turnRef{api: mod, turn: base})); err != nil {
		t.Fatalf("set base: %v", err)
	}
	v, err := vm.RunString(`
		const cloned = base.clone();
		const continued = gp.turn(base).user("follow-up").build();
		JSON.stringify({
			baseID: base.toJSON().id,
			cloneID: cloned.toJSON().id,
			continuedID: continued.toJSON().id || "",
			continuedBlocks: continued.toJSON().blocks.length,
		});
	`)
	if err != nil {
		t.Fatalf("run JS: %v", err)
	}
	want := `{"baseID":"turn-original","cloneID":"turn-original","continuedID":"","continuedBlocks":2}`
	if got := v.String(); got != want {
		t.Fatalf("ID semantics = %s, want %s", got, want)
	}
}

func TestTurnBuilderRejectsPlainObjectBase(t *testing.T) {
	vm, _ := newTurnBuilderTestRuntime(t)
	v, err := vm.RunString(`
		let message = "";
		try {
			gp.turn({ blocks: [] });
		} catch (err) {
			message = String(err);
		}
		message;
	`)
	if err != nil {
		t.Fatalf("run JS: %v", err)
	}
	if got := v.String(); got == "" || got == "undefined" {
		t.Fatalf("expected plain object rejection, got %q", got)
	}
}

func TestTurnBuilderContinuesWithMultimodalUserMessage(t *testing.T) {
	vm, _ := newTurnBuilderTestRuntime(t)
	v, err := vm.RunString(`
		const base = gp.turn().system("Inspect images.").build();
		const next = gp.turn(base)
			.user(m => m.text("What is this?").imageURL("https://example.invalid/image.png"))
			.build();
		const snap = next.toJSON();
		JSON.stringify({
			roles: snap.blocks.map(b => b.role),
			text: snap.blocks[1].payload.text,
			url: snap.blocks[1].payload.images[0].url,
		});
	`)
	if err != nil {
		t.Fatalf("run JS: %v", err)
	}
	want := `{"roles":["system","user"],"text":"What is this?","url":"https://example.invalid/image.png"}`
	if got := v.String(); got != want {
		t.Fatalf("multimodal continuation = %s, want %s", got, want)
	}
}
