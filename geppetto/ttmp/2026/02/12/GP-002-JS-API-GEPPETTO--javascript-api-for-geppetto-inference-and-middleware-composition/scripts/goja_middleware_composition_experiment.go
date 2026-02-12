//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/dop251/goja"
)

func mustRun(vm *goja.Runtime, src string) {
	if _, err := vm.RunString(src); err != nil {
		panic(err)
	}
}

func mustFn(vm *goja.Runtime, name string) goja.Callable {
	fn, ok := goja.AssertFunction(vm.Get(name))
	if !ok {
		panic(fmt.Sprintf("%s is not a function", name))
	}
	return fn
}

func main() {
	vm := goja.New()

	vm.Set("goPostProcess", func(turn map[string]any) map[string]any {
		blocks, ok := turn["blocks"].([]any)
		if !ok || len(blocks) == 0 {
			return turn
		}
		last, ok := blocks[len(blocks)-1].(map[string]any)
		if !ok {
			return turn
		}
		payload, ok := last["payload"].(map[string]any)
		if !ok {
			return turn
		}
		text, _ := payload["text"].(string)
		payload["text"] = text + " [go-post]"
		return turn
	})

	mustRun(vm, `
function compose(base, mws) {
  let h = base;
  for (let i = mws.length - 1; i >= 0; i--) {
    h = mws[i](h);
  }
  return h;
}

function baseHandler(_ctx, turn) {
  turn.blocks.push({ kind: "llm_text", role: "assistant", payload: { text: "hello world" } });
  return turn;
}

function jsUppercase(next) {
  return function(ctx, turn) {
    const out = next(ctx, turn);
    for (const b of out.blocks) {
      if (b.kind === "llm_text" && b.payload && typeof b.payload.text === "string") {
        b.payload.text = b.payload.text.toUpperCase();
      }
    }
    return out;
  };
}

function jsTag(next) {
  return function(ctx, turn) {
    const out = next(ctx, turn);
    out.metadata = out.metadata || {};
    out.metadata.tag = "js-tagged";
    return out;
  };
}

function jsGoAdapter(next) {
  return function(ctx, turn) {
    const out = next(ctx, turn);
    return goPostProcess(out);
  };
}

function jsThrower(_next) {
  return function(_ctx, _turn) {
    throw new Error("middleware exploded");
  };
}

function runHappyPath() {
  const handler = compose(baseHandler, [jsUppercase, jsTag, jsGoAdapter]);
  return handler({}, { blocks: [], metadata: {} });
}

function runErrorPath() {
  const handler = compose(baseHandler, [jsThrower]);
  return handler({}, { blocks: [], metadata: {} });
}

function runHotLoop(n) {
  const handler = compose(baseHandler, [jsUppercase, jsTag, jsGoAdapter]);
  let c = 0;
  for (let i = 0; i < n; i++) {
    const out = handler({}, { blocks: [], metadata: {} });
    const last = out.blocks[out.blocks.length - 1];
    c += (last.payload.text || "").length;
  }
  return c;
}
`)

	runHappy := mustFn(vm, "runHappyPath")
	runError := mustFn(vm, "runErrorPath")
	runHot := mustFn(vm, "runHotLoop")

	val, err := runHappy(goja.Undefined())
	if err != nil {
		panic(err)
	}

	exported := val.Export().(map[string]any)
	blocks := exported["blocks"].([]any)
	last := blocks[len(blocks)-1].(map[string]any)
	payload := last["payload"].(map[string]any)
	text := payload["text"].(string)
	meta := exported["metadata"].(map[string]any)

	fmt.Println("=== Goja Middleware Composition Experiment ===")
	fmt.Printf("happy-path final text: %q\n", text)
	fmt.Printf("happy-path metadata.tag: %v\n", meta["tag"])

	_, err = runError(goja.Undefined())
	if err == nil {
		fmt.Println("error-path: expected an error but got nil")
	} else {
		fmt.Printf("error-path error: %s\n", strings.TrimSpace(err.Error()))
	}

	const n = 30000
	start := time.Now()
	v, err := runHot(goja.Undefined(), vm.ToValue(n))
	if err != nil {
		panic(err)
	}
	dur := time.Since(start)
	fmt.Printf("hot-loop runHotLoop(%d) => %d in %s (%.2f us/op)\n", n, v.ToInteger(), dur, float64(dur.Microseconds())/float64(n))
}
