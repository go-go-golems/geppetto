package main

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type turnsSchema struct {
	Namespace  string           `yaml:"namespace"`
	BlockKinds []turnsBlockKind `yaml:"block_kinds"`
	Keys       []turnsKeySchema `yaml:"keys"`
}

type turnsBlockKind struct {
	Const string `yaml:"const"`
	Value string `yaml:"value"`
}

type turnsKeySchema struct {
	Scope string `yaml:"scope"`
	Value string `yaml:"value"`
}

type jsSchema struct {
	Enums []jsEnum `yaml:"enums"`
}

type jsEnum struct {
	Name   string        `yaml:"name"`
	Values []jsEnumValue `yaml:"values"`
}

type jsEnumValue struct {
	JSKey string `yaml:"js_key"`
	Value string `yaml:"value"`
}

var (
	payloadLineRE = regexp.MustCompile(`(?m)^\s*(PayloadKey[A-Za-z0-9_]+)\s*=\s*"([^"]+)"`)
	runMetaLineRE = regexp.MustCompile(`(?m)^\s*(RunMetaKey[A-Za-z0-9_]+)\s+RunMetadataKey\s*=\s*"([^"]+)"`)
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintf(os.Stderr, "usage: %s <turns-schema> <js-schema> <turns-keys-go>\n", os.Args[0])
		os.Exit(2)
	}

	turnsPath := os.Args[1]
	jsPath := os.Args[2]
	keysPath := os.Args[3]

	ts := mustLoadTurns(turnsPath)
	js := mustLoadJS(jsPath)
	keysSrc := mustRead(keysPath)

	fmt.Println("== Codegen Surface Inventory ==")
	fmt.Printf("turns.namespace: %s\n", ts.Namespace)
	fmt.Printf("turns.block_kinds: %d\n", len(ts.BlockKinds))

	byScope := map[string][]string{}
	for _, k := range ts.Keys {
		byScope[k.Scope] = append(byScope[k.Scope], k.Value)
	}
	for _, scope := range []string{"data", "turn_meta", "block_meta"} {
		vals := append([]string(nil), byScope[scope]...)
		sort.Strings(vals)
		fmt.Printf("turns.keys[%s]: %d\n", scope, len(vals))
	}

	enumNames := make([]string, 0, len(js.Enums))
	for _, e := range js.Enums {
		enumNames = append(enumNames, e.Name)
	}
	sort.Strings(enumNames)
	fmt.Printf("js.enums: %d\n", len(enumNames))
	fmt.Printf("js.enum.names: %s\n", strings.Join(enumNames, ", "))

	payloadMatches := payloadLineRE.FindAllStringSubmatch(keysSrc, -1)
	runMetaMatches := runMetaLineRE.FindAllStringSubmatch(keysSrc, -1)
	fmt.Printf("manual.payload_keys(keys.go): %d\n", len(payloadMatches))
	fmt.Printf("manual.run_meta_keys(keys.go): %d\n", len(runMetaMatches))

	fmt.Println("\n== Families not in turns_codegen.yaml today ==")
	fmt.Println("- payload keys (Block.Payload map string keys) remain manual in pkg/turns/keys.go")
	fmt.Println("- run metadata keys remain manual in pkg/turns/keys.go")

	fmt.Println("\n== Suggested unified families ==")
	fmt.Println("- block_kinds")
	fmt.Println("- key_families.data")
	fmt.Println("- key_families.turn_meta")
	fmt.Println("- key_families.block_meta")
	fmt.Println("- key_families.run_meta")
	fmt.Println("- payload_keys")
	fmt.Println("- js_enums (JS-specific only)")
	fmt.Println("- js_exports (which families/enums are exported to gp.consts)")
}

func mustRead(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func mustLoadTurns(path string) *turnsSchema {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var s turnsSchema
	if err := yaml.Unmarshal(b, &s); err != nil {
		panic(err)
	}
	return &s
}

func mustLoadJS(path string) *jsSchema {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var s jsSchema
	if err := yaml.Unmarshal(b, &s); err != nil {
		panic(err)
	}
	return &s
}
