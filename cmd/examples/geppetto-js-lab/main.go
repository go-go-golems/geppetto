package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/middlewarecfg"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	gp "github.com/go-go-golems/geppetto/pkg/js/modules/geppetto"
	"github.com/go-go-golems/geppetto/pkg/profiles"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
)

func main() {
	scriptPath := flag.String("script", "", "Path to JavaScript file to execute")
	profileRegistries := flag.String("profile-registries", "", "Comma-separated profile registry sources (yaml/sqlite/sqlite-dsn)")
	seedProfileSQLite := flag.String("seed-profile-sqlite", "", "Seed a demo writable sqlite profile registry at this path")
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

	if dbPath := strings.TrimSpace(*seedProfileSQLite); dbPath != "" {
		if err := seedDemoProfileSQLite(dbPath); err != nil {
			fatalf("failed to seed sqlite profile registry %q: %v", dbPath, err)
		}
		fmt.Printf("Seeded profile registry sqlite: %s\n", dbPath)
		if strings.TrimSpace(*scriptPath) == "" {
			return
		}
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

	profileRegistry, profileRegistryWriter, closer, err := loadProfileRegistryStack(*profileRegistries)
	if err != nil {
		fatalf("failed to load profile registries: %v", err)
	}
	if closer != nil {
		defer func() {
			_ = closer.Close()
		}()
	}

	middlewareSchemas, extensionCodecs, extensionSchemas, err := buildDemoSchemaProviders()
	if err != nil {
		fatalf("failed to build schema providers: %v", err)
	}

	reg := require.NewRegistry()
	gp.Register(reg, gp.Options{
		Runner:                runner,
		GoToolRegistry:        goRegistry,
		ProfileRegistry:       profileRegistry,
		ProfileRegistryWriter: profileRegistryWriter,
		MiddlewareSchemas:     middlewareSchemas,
		ExtensionCodecs:       extensionCodecs,
		ExtensionSchemas:      extensionSchemas,
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

func loadProfileRegistryStack(raw string) (profiles.RegistryReader, profiles.RegistryWriter, io.Closer, error) {
	entries, err := profiles.ParseProfileRegistrySourceEntries(raw)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(entries) == 0 {
		return nil, nil, nil, nil
	}
	specs, err := profiles.ParseRegistrySourceSpecs(entries)
	if err != nil {
		return nil, nil, nil, err
	}
	chain, err := profiles.NewChainedRegistryFromSourceSpecs(context.Background(), specs)
	if err != nil {
		return nil, nil, nil, err
	}
	return chain, chain, chain, nil
}

func seedDemoProfileSQLite(path string) error {
	dsn, err := profiles.SQLiteProfileDSNForFile(path)
	if err != nil {
		return err
	}
	store, err := profiles.NewSQLiteProfileStore(dsn, profiles.MustRegistrySlug("workspace-db"))
	if err != nil {
		return err
	}
	defer func() {
		_ = store.Close()
	}()

	reg := &profiles.ProfileRegistry{
		Slug:               profiles.MustRegistrySlug("workspace-db"),
		DefaultProfileSlug: profiles.MustProfileSlug("default"),
		Profiles: map[profiles.ProfileSlug]*profiles.Profile{
			profiles.MustProfileSlug("default"): {
				Slug:        profiles.MustProfileSlug("default"),
				DisplayName: "Workspace default",
				Runtime: profiles.RuntimeSpec{
					StepSettingsPatch: map[string]any{
						"ai-chat": map[string]any{
							"ai-api-type": "openai",
							"ai-engine":   "gpt-4o-mini",
						},
						"api": map[string]any{
							"openai-api-key": "seed-openai-key",
						},
					},
					SystemPrompt: "You are the workspace default assistant.",
				},
				Policy: profiles.PolicySpec{
					AllowOverrides: true,
				},
			},
			profiles.MustProfileSlug("assistant"): {
				Slug: profiles.MustProfileSlug("assistant"),
				Stack: []profiles.ProfileRef{
					{ProfileSlug: profiles.MustProfileSlug("default")},
				},
				Runtime: profiles.RuntimeSpec{
					StepSettingsPatch: map[string]any{
						"ai-chat": map[string]any{
							"ai-engine": "gpt-4.1-nano",
						},
					},
					SystemPrompt: "You are the workspace assistant profile.",
				},
			},
		},
	}

	return store.UpsertRegistry(context.Background(), reg, profiles.SaveOptions{
		Actor:  "geppetto-js-lab",
		Source: "seed-profile-sqlite",
	})
}

func buildDemoSchemaProviders() (
	middlewarecfg.DefinitionRegistry,
	profiles.ExtensionCodecRegistry,
	map[string]map[string]any,
	error,
) {
	mwRegistry := middlewarecfg.NewInMemoryDefinitionRegistry()
	definitions := []demoMiddlewareDefinition{
		{
			name:        "retry",
			displayName: "Retry",
			description: "Retry middleware policy config",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"maxAttempts": map[string]any{"type": "integer", "minimum": 1},
					"backoffMs":   map[string]any{"type": "integer", "minimum": 0},
				},
			},
		},
		{
			name:        "agentmode",
			displayName: "Agent Mode",
			description: "Agent mode routing config",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"mode": map[string]any{"type": "string"},
				},
			},
		},
		{
			name:        "telemetry",
			displayName: "Telemetry",
			description: "Telemetry enrichment config",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"level": map[string]any{"type": "string"},
				},
			},
		},
	}
	for _, def := range definitions {
		if err := mwRegistry.RegisterDefinition(def); err != nil {
			return nil, nil, nil, err
		}
	}

	extRegistry, err := profiles.NewInMemoryExtensionCodecRegistry(
		demoExtensionCodec{
			key:         profiles.MustExtensionKey("demo.analytics@v1"),
			displayName: "Demo Analytics",
			description: "Runtime analytics extension payload",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"enabled": map[string]any{"type": "boolean"},
					"sink":    map[string]any{"type": "string"},
				},
			},
		},
		demoExtensionCodec{
			key:         profiles.MustExtensionKey("demo.safety@v1"),
			displayName: "Demo Safety",
			description: "Safety policy extension payload",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"strategy": map[string]any{"type": "string"},
				},
			},
		},
	)
	if err != nil {
		return nil, nil, nil, err
	}

	staticSchemas := map[string]map[string]any{
		"host.notes@v1": {
			"type": "object",
			"properties": map[string]any{
				"value": map[string]any{"type": "string"},
			},
		},
	}

	return mwRegistry, extRegistry, staticSchemas, nil
}

type demoMiddlewareDefinition struct {
	name        string
	displayName string
	description string
	schema      map[string]any
}

func (d demoMiddlewareDefinition) Name() string {
	return d.name
}

func (d demoMiddlewareDefinition) ConfigJSONSchema() map[string]any {
	return cloneSchemaMap(d.schema)
}

func (d demoMiddlewareDefinition) Build(context.Context, middlewarecfg.BuildDeps, any) (middleware.Middleware, error) {
	return func(next middleware.HandlerFunc) middleware.HandlerFunc {
		return next
	}, nil
}

type demoExtensionCodec struct {
	key         profiles.ExtensionKey
	displayName string
	description string
	schema      map[string]any
}

func (c demoExtensionCodec) Key() profiles.ExtensionKey {
	return c.key
}

func (c demoExtensionCodec) Decode(raw any) (any, error) {
	return raw, nil
}

func (c demoExtensionCodec) JSONSchema() map[string]any {
	return cloneSchemaMap(c.schema)
}

func (c demoExtensionCodec) ExtensionDisplayName() string {
	return c.displayName
}

func (c demoExtensionCodec) ExtensionDescription() string {
	return c.description
}

func cloneSchemaMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	b, err := json.Marshal(in)
	if err != nil {
		return in
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return in
	}
	return out
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
