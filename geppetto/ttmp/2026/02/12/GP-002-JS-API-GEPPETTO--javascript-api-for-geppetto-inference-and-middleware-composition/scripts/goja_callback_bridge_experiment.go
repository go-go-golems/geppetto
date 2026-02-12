//go:build ignore
// +build ignore

package main

import (
	"fmt"
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

func callInt(fn goja.Callable, n int) (int64, error) {
	v, err := fn(goja.Undefined(), goja.New().ToValue(n))
	if err != nil {
		return 0, err
	}
	return v.ToInteger(), nil
}

func main() {
	vm := goja.New()

	vm.Set("goInc", func(x int64) int64 {
		return x + 1
	})

	vm.Set("goConsumeBlock", func(block map[string]any) int64 {
		payload, _ := block["payload"].(map[string]any)
		text, _ := payload["text"].(string)
		return int64(len(text))
	})

	mustRun(vm, `
function pureJsLoop(n) {
  let x = 0;
  function localInc(v) { return v + 1; }
  for (let i = 0; i < n; i++) {
    x = localInc(x);
  }
  return x;
}

function goCallLoop(n) {
  let x = 0;
  for (let i = 0; i < n; i++) {
    x = goInc(x);
  }
  return x;
}

function goObjectLoop(n) {
  let total = 0;
  for (let i = 0; i < n; i++) {
    total += goConsumeBlock({
      kind: "llm_text",
      payload: { text: "hello-" + i },
      metadata: { idx: i }
    });
  }
  return total;
}
`)

	pure := mustFn(vm, "pureJsLoop")
	goCall := mustFn(vm, "goCallLoop")
	goObj := mustFn(vm, "goObjectLoop")

	const nPure = 250000
	const nGoCall = 250000
	const nGoObj = 25000

	startPure := time.Now()
	vPure, err := pure(goja.Undefined(), vm.ToValue(nPure))
	if err != nil {
		panic(err)
	}
	dPure := time.Since(startPure)

	startGoCall := time.Now()
	vGoCall, err := goCall(goja.Undefined(), vm.ToValue(nGoCall))
	if err != nil {
		panic(err)
	}
	dGoCall := time.Since(startGoCall)

	startGoObj := time.Now()
	vGoObj, err := goObj(goja.Undefined(), vm.ToValue(nGoObj))
	if err != nil {
		panic(err)
	}
	dGoObj := time.Since(startGoObj)

	fmt.Println("=== Goja Callback Bridge Experiment ===")
	fmt.Printf("pureJsLoop(%d) => %d in %s\n", nPure, vPure.ToInteger(), dPure)
	fmt.Printf("goCallLoop(%d) => %d in %s\n", nGoCall, vGoCall.ToInteger(), dGoCall)
	fmt.Printf("goObjectLoop(%d) => %d in %s\n", nGoObj, vGoObj.ToInteger(), dGoObj)

	perPure := float64(dPure.Nanoseconds()) / float64(nPure)
	perGoCall := float64(dGoCall.Nanoseconds()) / float64(nGoCall)
	perGoObj := float64(dGoObj.Nanoseconds()) / float64(nGoObj)

	fmt.Printf("per-call pure-js: %.2f ns\n", perPure)
	fmt.Printf("per-call js->go scalar: %.2f ns (x%.2f vs pure)\n", perGoCall, perGoCall/perPure)
	fmt.Printf("per-call js->go object: %.2f ns (x%.2f vs pure)\n", perGoObj, perGoObj/perPure)

	_ = callInt
}
