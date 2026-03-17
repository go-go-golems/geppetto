package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/go-go-golems/geppetto/pkg/inference/tools/scopedjs"
)

func main() {
	ctx := context.Background()
	spec := scopedjs.EnvironmentSpec[struct{}, struct{}]{
		RuntimeLabel: "repro",
		Tool:         scopedjs.ToolDefinitionSpec{Name: "eval_repro"},
		DefaultEval:  scopedjs.DefaultEvalOptions(),
		Configure: func(ctx context.Context, b *scopedjs.Builder, _ struct{}) (struct{}, error) {
			return struct{}{}, nil
		},
	}

	handle, err := scopedjs.BuildRuntime(ctx, spec, struct{}{})
	if err != nil {
		fatalf("BuildRuntime failed: %v", err)
	}
	defer func() { _ = handle.Cleanup() }()

	cases := []struct {
		Name string
		Code string
	}{
		{
			Name: "string rejection",
			Code: `await Promise.reject("boom")`,
		},
		{
			Name: "error rejection",
			Code: `await Promise.reject(new Error("boom"))`,
		},
		{
			Name: "throw error",
			Code: `throw new Error("boom")`,
		},
	}

	for _, tc := range cases {
		out, err := scopedjs.RunEval(ctx, handle.Runtime, scopedjs.EvalInput{Code: tc.Code}, scopedjs.DefaultEvalOptions())
		if err != nil {
			fatalf("%s: RunEval returned error: %v", tc.Name, err)
		}
		raw, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			fatalf("%s: marshal output: %v", tc.Name, err)
		}
		fmt.Printf("== %s ==\n%s\n\n", tc.Name, raw)
	}
}

func fatalf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
