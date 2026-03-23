package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type options struct {
	SchemaPath string
	Check      bool
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "gen-dts: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	opts, err := parseOptions(args)
	if err != nil {
		return err
	}

	absSchemaPath, err := filepath.Abs(opts.SchemaPath)
	if err != nil {
		return fmt.Errorf("resolve schema path %q: %w", opts.SchemaPath, err)
	}
	repoRoot, err := findRepoRoot(absSchemaPath)
	if err != nil {
		return err
	}

	if !opts.Check {
		return runGenMeta(repoRoot, absSchemaPath)
	}

	checkSchemaPath, checkOutputPath, expectedOutputPath, cleanup, err := prepareCheckSchema(absSchemaPath)
	if err != nil {
		return err
	}
	defer cleanup()

	if err := runGenMeta(repoRoot, checkSchemaPath); err != nil {
		return err
	}

	expected, err := os.ReadFile(expectedOutputPath)
	if err != nil {
		return fmt.Errorf("read expected output %q: %w", expectedOutputPath, err)
	}
	actual, err := os.ReadFile(checkOutputPath)
	if err != nil {
		return fmt.Errorf("read generated check output %q: %w", checkOutputPath, err)
	}

	if !bytes.Equal(expected, actual) {
		return fmt.Errorf("--check failed: generated output differs from %s", expectedOutputPath)
	}
	return nil
}

func parseOptions(args []string) (options, error) {
	fs := flag.NewFlagSet("gen-dts", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	schemaPath := fs.String("schema", "pkg/spec/geppetto_codegen.yaml", "Path to geppetto codegen schema")
	check := fs.Bool("check", false, "Check mode: fail if generated d.ts differs from current output")

	if err := fs.Parse(args); err != nil {
		return options{}, err
	}

	return options{
		SchemaPath: *schemaPath,
		Check:      *check,
	}, nil
}

func runGenMeta(repoRoot, schemaPath string) error {
	cmd := exec.Command("go", "run", "./cmd/tools/gen-meta", "--schema", schemaPath, "--section", "js-dts")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		if len(out) == 0 {
			return fmt.Errorf("run gen-meta: %w", err)
		}
		return fmt.Errorf("run gen-meta: %w\n%s", err, out)
	}
	return nil
}

func findRepoRoot(schemaPath string) (string, error) {
	dir := filepath.Dir(schemaPath)
	for {
		marker := filepath.Join(dir, "cmd", "tools", "gen-meta", "main.go")
		if st, err := os.Stat(marker); err == nil && !st.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("could not find geppetto repo root from schema path %q", schemaPath)
}

func prepareCheckSchema(absSchemaPath string) (string, string, string, func(), error) {
	schemaBytes, err := os.ReadFile(absSchemaPath)
	if err != nil {
		return "", "", "", nil, fmt.Errorf("read schema %q: %w", absSchemaPath, err)
	}

	var raw map[string]any
	if err := yaml.Unmarshal(schemaBytes, &raw); err != nil {
		return "", "", "", nil, fmt.Errorf("parse schema %q: %w", absSchemaPath, err)
	}

	outputsAny, ok := raw["outputs"]
	if !ok {
		return "", "", "", nil, errors.New("schema outputs is missing")
	}
	outputs, ok := outputsAny.(map[string]any)
	if !ok {
		return "", "", "", nil, errors.New("schema outputs is not a map")
	}

	outAny, ok := outputs["geppetto_dts"]
	if !ok {
		return "", "", "", nil, errors.New("schema outputs.geppetto_dts is missing")
	}
	outString, ok := outAny.(string)
	if !ok {
		return "", "", "", nil, errors.New("schema outputs.geppetto_dts is not a string")
	}
	if outString == "" {
		return "", "", "", nil, errors.New("schema outputs.geppetto_dts is empty")
	}

	schemaDir := filepath.Dir(absSchemaPath)
	expectedOutputPath := outString
	if !filepath.IsAbs(expectedOutputPath) {
		expectedOutputPath = filepath.Join(schemaDir, expectedOutputPath)
	}
	expectedOutputPath = filepath.Clean(expectedOutputPath)

	tmpOutputFile, err := os.CreateTemp("", "geppetto-dts-check-*.d.ts")
	if err != nil {
		return "", "", "", nil, fmt.Errorf("create temp d.ts output: %w", err)
	}
	checkOutputPath := tmpOutputFile.Name()
	if err := tmpOutputFile.Close(); err != nil {
		return "", "", "", nil, fmt.Errorf("close temp d.ts output file: %w", err)
	}

	outputs["geppetto_dts"] = checkOutputPath
	checkSchemaBytes, err := yaml.Marshal(raw)
	if err != nil {
		return "", "", "", nil, fmt.Errorf("marshal check schema: %w", err)
	}

	tmpSchemaFile, err := os.CreateTemp(schemaDir, ".geppetto-dts-check-*.yaml")
	if err != nil {
		return "", "", "", nil, fmt.Errorf("create temp schema in %q: %w", schemaDir, err)
	}
	checkSchemaPath := tmpSchemaFile.Name()
	if _, err := tmpSchemaFile.Write(checkSchemaBytes); err != nil {
		_ = tmpSchemaFile.Close()
		return "", "", "", nil, fmt.Errorf("write temp schema %q: %w", checkSchemaPath, err)
	}
	if err := tmpSchemaFile.Close(); err != nil {
		return "", "", "", nil, fmt.Errorf("close temp schema %q: %w", checkSchemaPath, err)
	}

	cleanup := func() {
		_ = os.Remove(checkSchemaPath)
		_ = os.Remove(checkOutputPath)
	}
	return checkSchemaPath, checkOutputPath, expectedOutputPath, cleanup, nil
}
