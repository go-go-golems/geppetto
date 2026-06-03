package geppetto

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/dop251/goja"
	inferenceengine "github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

type sessionEchoEngine struct{}

var _ inferenceengine.Engine = (*sessionEchoEngine)(nil)

func (e *sessionEchoEngine) RunInference(_ context.Context, t *turns.Turn) (*turns.Turn, error) {
	out := &turns.Turn{}
	if t != nil {
		out = t.Clone()
	}
	users := []string{}
	assistants := []string{}
	for _, block := range out.Blocks {
		text, _ := block.Payload[turns.PayloadKeyText].(string)
		switch block.Role {
		case turns.RoleUser:
			users = append(users, text)
		case turns.RoleAssistant:
			assistants = append(assistants, text)
		}
	}
	turns.AppendBlock(out, turns.NewAssistantTextBlock("users="+strings.Join(users, ",")+" assistants="+strings.Join(assistants, ",")))
	return out, nil
}

func TestAgentSessionRunsMultipleTurnsWithHistory(t *testing.T) {
	rt := newJSRuntime(t, Options{})
	_, err := rt.runtimeOwner.Call(context.Background(), "test.sessionMultiturn", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if setErr := vm.Set("fakeEngine", &sessionEchoEngine{}); setErr != nil {
			return nil, setErr
		}
		_, runErr := vm.RunString(`
			const gp = require("geppetto");
			const agent = gp.agent().engine(globalThis.fakeEngine).build();
			const session = agent.session().id("chat-a").build();
			const r1 = session.next().system("be terse").user("one").run();
			const firstId = r1.outputTurn().toJSON().id;
			const r2 = session.next().user("two").run();
			const second = r2.outputTurn().toJSON();
			globalThis.sessionResult = JSON.stringify({
				id: session.id(),
				turnCount: session.turnCount(),
				latestText: r2.text(),
				firstIdPreserved: session.turn(0).toJSON().id === firstId,
				newTurnId: second.id !== firstId && second.id.length > 0,
				inputBlocks: r2.inputTurn().toJSON().blocks.length,
			});
		`)
		return nil, runErr
	})
	if err != nil {
		t.Fatalf("session multiturn script failed: %v", err)
	}
	got := mustEvalExprExport(t, rt, `globalThis.sessionResult`)
	want := `{"id":"chat-a","turnCount":2,"latestText":"users=one assistants=\nusers=one,two assistants=users=one assistants=","firstIdPreserved":true,"newTurnId":true,"inputBlocks":4}`
	if got != want {
		t.Fatalf("session result = %v, want %s", got, want)
	}
}

func TestAgentSessionBaseAndFork(t *testing.T) {
	rt := newJSRuntime(t, Options{})
	_, err := rt.runtimeOwner.Call(context.Background(), "test.sessionFork", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if setErr := vm.Set("fakeEngine", &sessionEchoEngine{}); setErr != nil {
			return nil, setErr
		}
		_, runErr := vm.RunString(`
			const gp = require("geppetto");
			const agent = gp.agent().engine(globalThis.fakeEngine).build();
			const main = agent.session().id("main").build();
			const r1 = main.next().user("root").run();
			const baseId = r1.outputTurn().toJSON().id;
			const fork = main.fork().id("fork").build();
			const imported = fork.latestTurn().toJSON();
			const r2 = fork.next().user("branch").run();
			globalThis.forkResult = JSON.stringify({
				mainTurns: main.turnCount(),
				forkTurns: fork.turnCount(),
				importedBaseIdPreserved: imported.id === baseId,
				forkSessionId: fork.id(),
				latestText: r2.text(),
				forkNewTurnId: r2.outputTurn().toJSON().id !== baseId,
			});
		`)
		return nil, runErr
	})
	if err != nil {
		t.Fatalf("session fork script failed: %v", err)
	}
	got := mustEvalExprExport(t, rt, `globalThis.forkResult`)
	want := `{"mainTurns":1,"forkTurns":2,"importedBaseIdPreserved":true,"forkSessionId":"fork","latestText":"users=root assistants=\nusers=root,branch assistants=users=root assistants=","forkNewTurnId":true}`
	if got != want {
		t.Fatalf("fork result = %v, want %s", got, want)
	}
}

func TestAgentSessionForkAtTurnWrapper(t *testing.T) {
	rt := newJSRuntime(t, Options{})
	_, err := rt.runtimeOwner.Call(context.Background(), "test.sessionForkAtTurn", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if setErr := vm.Set("fakeEngine", &sessionEchoEngine{}); setErr != nil {
			return nil, setErr
		}
		_, runErr := vm.RunString(`
			const gp = require("geppetto");
			const agent = gp.agent().engine(globalThis.fakeEngine).build();
			const main = agent.session().id("main-at").build();
			main.next().user("first").run();
			main.next().user("second").run();
			const firstTurn = main.turn(0);
			const fork = main.fork({ at: firstTurn }).id("fork-at").build();
			const result = fork.next().user("branch").run();
			globalThis.forkAtResult = JSON.stringify({
				forkTurns: fork.turnCount(),
				importedBaseIdPreserved: fork.turn(0).toJSON().id === firstTurn.toJSON().id,
				latestText: result.text(),
			});
		`)
		return nil, runErr
	})
	if err != nil {
		t.Fatalf("session fork at wrapper script failed: %v", err)
	}
	got := mustEvalExprExport(t, rt, `globalThis.forkAtResult`)
	want := `{"forkTurns":2,"importedBaseIdPreserved":true,"latestText":"users=first assistants=\nusers=first,branch assistants=users=first assistants="}`
	if got != want {
		t.Fatalf("fork-at result = %v, want %s", got, want)
	}
}

func TestAgentSessionResumeLatestFromStore(t *testing.T) {
	store := &recordingTurnStore{name: "default"}
	rt := newJSRuntime(t, Options{DefaultTurnStore: store})
	_, err := rt.runtimeOwner.Call(context.Background(), "test.sessionResume", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if setErr := vm.Set("fakeEngine", &sessionEchoEngine{}); setErr != nil {
			return nil, setErr
		}
		_, runErr := vm.RunString(`
			const gp = require("geppetto");
			const store = gp.turnStores.default();
			const agent = gp.agent().engine(globalThis.fakeEngine).build();
			const first = agent.session().id("resume-me").store(store).build();
			first.next().user("persisted").run();
			const resumed = agent.session().id("resume-me").store(store).resumeLatest().build();
			const r2 = resumed.next().user("continued").run();
			globalThis.resumeResult = JSON.stringify({
				storeCount: store.list({ sessionId: "resume-me" }).length,
				turnCount: resumed.turnCount(),
				latestText: r2.text(),
			});
		`)
		return nil, runErr
	})
	if err != nil {
		t.Fatalf("session resume script failed: %v", err)
	}
	got := mustEvalExprExport(t, rt, `globalThis.resumeResult`)
	want := `{"storeCount":2,"turnCount":2,"latestText":"users=persisted assistants=\nusers=persisted,continued assistants=users=persisted assistants="}`
	if got != want {
		t.Fatalf("resume result = %v, want %s", got, want)
	}
}

func TestAgentSessionRunAsync(t *testing.T) {
	rt := newJSRuntime(t, Options{})
	_, err := rt.runtimeOwner.Call(context.Background(), "test.sessionRunAsync", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if setErr := vm.Set("fakeEngine", &sessionEchoEngine{}); setErr != nil {
			return nil, setErr
		}
		_, runErr := vm.RunString(`
			const gp = require("geppetto");
			globalThis.sessionAsyncDone = false;
			const agent = gp.agent().engine(globalThis.fakeEngine).build();
			const session = agent.session().id("async-session").build();
			session.next().user("async").runAsync().promise.then(
				result => { globalThis.sessionAsyncDone = true; globalThis.sessionAsyncText = result.text(); globalThis.sessionAsyncTurns = session.turnCount(); },
				err => { globalThis.sessionAsyncDone = true; globalThis.sessionAsyncError = String(err); },
			);
		`)
		return nil, runErr
	})
	if err != nil {
		t.Fatalf("session runAsync script failed: %v", err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		v := mustEvalExprExport(t, rt, `globalThis.sessionAsyncDone === true`)
		if done, ok := v.(bool); ok && done {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	got := mustEvalExprExport(t, rt, `JSON.stringify({done: globalThis.sessionAsyncDone, text: globalThis.sessionAsyncText, turns: globalThis.sessionAsyncTurns, error: globalThis.sessionAsyncError})`)
	want := `{"done":true,"text":"users=async assistants=","turns":1}`
	if got != want {
		t.Fatalf("runAsync result = %v, want %s", got, want)
	}
}
