package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	gp "github.com/go-go-golems/geppetto/pkg/js/modules/geppetto"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
)

func main() {
	scriptPath := flag.String("script", "", "Path to JavaScript file to execute")
	printResult := flag.Bool("print-result", false, "Print top-level JS return value as JSON")
	listGoTools := flag.Bool("list-go-tools", false, "List built-in Go tools exposed to JS and exit")
	flag.Parse()

	goRegistry, err := buildGoToolRegistry()
	if err != nil {
		fatalf("failed to build go tool registry: %v", err)
	}

	if *listGoTools {
		for _, t := range goRegistry.ListTools() {
			fmt.Println(t.Name)
		}
		return
	}

	if strings.TrimSpace(*scriptPath) == "" {
		fatalf("--script is required")
	}

	scriptBytes, err := os.ReadFile(*scriptPath)
	if err != nil {
		fatalf("failed to read script %q: %v", *scriptPath, err)
	}

	loop := eventloop.NewEventLoop()
	go loop.Start()
	defer loop.Stop()

	vm := goja.New()
	runner := runtimeowner.NewRunner(vm, loop, runtimeowner.Options{
		Name:          "geppetto-js-lab",
		RecoverPanics: true,
	})
	installConsole(vm)
	installHelpers(vm)

	reg := require.NewRegistry()
	gp.Register(reg, gp.Options{
		Runner:         runner,
		GoToolRegistry: goRegistry,
	})
	reg.Enable(vm)

	result, err := vm.RunScript(filepath.Base(*scriptPath), string(scriptBytes))
	if err != nil {
		fatalf("js execution failed (%s): %v", *scriptPath, err)
	}

	if *printResult && result != nil && !goja.IsUndefined(result) && !goja.IsNull(result) {
		blob, err := json.MarshalIndent(result.Export(), "", "  ")
		if err != nil {
			fatalf("failed to marshal result: %v", err)
		}
		fmt.Println(string(blob))
	}

	fmt.Printf("PASS: %s\n", filepath.Base(*scriptPath))
}

func buildGoToolRegistry() (tools.ToolRegistry, error) {
	reg := tools.NewInMemoryToolRegistry()

	type goDoubleInput struct {
		N int `json:"n" jsonschema:"required,description=Number to double"`
	}
	doubleDef, err := tools.NewToolFromFunc(
		"go_double",
		"Double a number and return {value}",
		func(in goDoubleInput) (map[string]any, error) {
			return map[string]any{"value": in.N * 2}, nil
		},
	)
	if err != nil {
		return nil, err
	}
	if err := reg.RegisterTool("go_double", *doubleDef); err != nil {
		return nil, err
	}

	type goConcatInput struct {
		A string `json:"a" jsonschema:"required,description=First string"`
		B string `json:"b" jsonschema:"required,description=Second string"`
	}
	concatDef, err := tools.NewToolFromFunc(
		"go_concat",
		"Concatenate two strings and return {value}",
		func(in goConcatInput) (map[string]any, error) {
			return map[string]any{"value": in.A + in.B}, nil
		},
	)
	if err != nil {
		return nil, err
	}
	if err := reg.RegisterTool("go_concat", *concatDef); err != nil {
		return nil, err
	}

	return reg, nil
}

func installHelpers(vm *goja.Runtime) {
	_ = vm.Set("ENV", mapEnv())

	_ = vm.Set("sleep", func(ms int64) {
		if ms <= 0 {
			return
		}
		time.Sleep(time.Duration(ms) * time.Millisecond)
	})

	_, err := vm.RunString(`
globalThis.assert = function assert(cond, msg) {
  if (!cond) {
    throw new Error(msg || "assertion failed");
  }
};
`)
	if err != nil {
		panic(err)
	}
}

func mapEnv() map[string]string {
	out := map[string]string{}
	for _, kv := range os.Environ() {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			continue
		}
		out[parts[0]] = parts[1]
	}
	return out
}

func installConsole(vm *goja.Runtime) {
	console := vm.NewObject()
	_ = console.Set("log", func(call goja.FunctionCall) goja.Value {
		fmt.Fprintln(os.Stdout, joinArgs(call.Arguments))
		return goja.Undefined()
	})
	_ = console.Set("error", func(call goja.FunctionCall) goja.Value {
		fmt.Fprintln(os.Stderr, joinArgs(call.Arguments))
		return goja.Undefined()
	})
	_ = vm.Set("console", console)
}

func joinArgs(args []goja.Value) string {
	if len(args) == 0 {
		return ""
	}
	out := make([]string, 0, len(args))
	for _, arg := range args {
		switch {
		case arg == nil || goja.IsUndefined(arg):
			out = append(out, "undefined")
		case goja.IsNull(arg):
			out = append(out, "null")
		default:
			exp := arg.Export()
			if b, err := json.Marshal(exp); err == nil && json.Valid(b) {
				out = append(out, string(b))
			} else {
				out = append(out, fmt.Sprint(exp))
			}
		}
	}
	return strings.Join(out, " ")
}

func fatalf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
