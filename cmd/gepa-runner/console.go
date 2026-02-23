package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/dop251/goja"
)

func installConsoleAndHelpers(vm *goja.Runtime) error {
	if err := vm.Set("ENV", mapEnv()); err != nil {
		return err
	}

	console := vm.NewObject()
	_ = console.Set("log", func(call goja.FunctionCall) goja.Value {
		fmt.Println(joinArgs(call.Arguments))
		return goja.Undefined()
	})
	_ = console.Set("error", func(call goja.FunctionCall) goja.Value {
		fmt.Fprintln(os.Stderr, joinArgs(call.Arguments))
		return goja.Undefined()
	})
	if err := vm.Set("console", console); err != nil {
		return err
	}

	if _, err := vm.RunString(`
globalThis.assert = function assert(cond, msg) {
  if (!cond) {
    throw new Error(msg || "assertion failed");
  }
}
`); err != nil {
		return err
	}
	return nil
}

func mapEnv() map[string]string {
	out := map[string]string{}
	for _, kv := range os.Environ() {
		i := strings.Index(kv, "=")
		if i <= 0 {
			continue
		}
		out[kv[:i]] = kv[i+1:]
	}
	return out
}

func joinArgs(args []goja.Value) string {
	if len(args) == 0 {
		return ""
	}
	parts := make([]string, 0, len(args))
	for _, a := range args {
		if a == nil || goja.IsUndefined(a) || goja.IsNull(a) {
			continue
		}
		parts = append(parts, a.String())
	}
	return strings.Join(parts, " ")
}
