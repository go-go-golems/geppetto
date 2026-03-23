package main

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPrepareCheckSchemaPreservesSchemaContent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.yaml")
	schemaContent := `
version: 1
namespace: geppetto
outputs:
  geppetto_dts: out/geppetto.d.ts
templates:
  geppetto_dts: spec/geppetto.d.ts.tmpl
`
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	checkSchemaPath, checkOutputPath, expectedOutputPath, cleanup, err := prepareCheckSchema(schemaPath)
	if err != nil {
		t.Fatalf("prepare check schema: %v", err)
	}
	defer cleanup()

	if expectedOutputPath != filepath.Join(dir, "out", "geppetto.d.ts") {
		t.Fatalf("unexpected expected output path: %s", expectedOutputPath)
	}
	if _, err := os.Stat(checkOutputPath); err != nil {
		t.Fatalf("expected temp output file to exist: %v", err)
	}

	checkBytes, err := os.ReadFile(checkSchemaPath)
	if err != nil {
		t.Fatalf("read check schema: %v", err)
	}
	var parsed map[string]any
	if err := yaml.Unmarshal(checkBytes, &parsed); err != nil {
		t.Fatalf("parse check schema: %v", err)
	}
	if parsed["namespace"] != "geppetto" {
		t.Fatalf("namespace was not preserved: %v", parsed["namespace"])
	}

	outputs, ok := parsed["outputs"].(map[string]any)
	if !ok {
		t.Fatalf("outputs missing or wrong type")
	}
	gotOutput, ok := outputs["geppetto_dts"].(string)
	if !ok {
		t.Fatalf("outputs.geppetto_dts missing or wrong type")
	}
	if gotOutput != checkOutputPath {
		t.Fatalf("expected rewritten geppetto_dts output %s, got %s", checkOutputPath, gotOutput)
	}
}

func TestFindRepoRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	schemaPath := filepath.Join(root, "pkg", "spec", "schema.yaml")
	if err := os.MkdirAll(filepath.Dir(schemaPath), 0o755); err != nil {
		t.Fatalf("mkdir schema dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "cmd", "tools", "gen-meta"), 0o755); err != nil {
		t.Fatalf("mkdir gen-meta dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "tools", "gen-meta", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write marker file: %v", err)
	}

	found, err := findRepoRoot(schemaPath)
	if err != nil {
		t.Fatalf("find repo root: %v", err)
	}
	if found != root {
		t.Fatalf("expected root %s, got %s", root, found)
	}
}
