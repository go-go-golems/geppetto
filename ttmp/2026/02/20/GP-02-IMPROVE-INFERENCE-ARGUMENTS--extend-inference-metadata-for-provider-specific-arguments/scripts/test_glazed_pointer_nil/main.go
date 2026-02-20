// Experiment: Does glazed's DecodeSectionInto leave pointer fields nil
// when no default is provided in the YAML definition?
//
// Tests three scenarios:
//  1. Flag with explicit default (default: 42) → pointer should be non-nil
//  2. Flag with no default key at all → pointer should stay nil
//  3. Flag with default: 0 → pointer should be non-nil (pointing to 0)
//
// Run: go run ./ttmp/2026/02/20/GP-02-IMPROVE-INFERENCE-ARGUMENTS--extend-inference-metadata-for-provider-specific-arguments/scripts/test_glazed_pointer_nil/
package main

import (
	"fmt"
	"os"

	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

// Target struct with pointer fields — mirrors InferenceConfig's pattern.
type TestConfig struct {
	WithDefault    *int     `glazed:"with-default"`
	NoDefault      *int     `glazed:"no-default"`
	ZeroDefault    *int     `glazed:"zero-default"`
	FloatNoDefault *float64 `glazed:"float-no-default"`
	StrNoDefault   *string  `glazed:"str-no-default"`
	StrEmpty       *string  `glazed:"str-empty-default"`
}

var testYAML = []byte(`
slug: test-section
name: Test section
flags:
  - name: with-default
    type: int
    help: Has an explicit non-zero default
    default: 42
  - name: no-default
    type: int
    help: No default key at all
  - name: zero-default
    type: int
    help: Has explicit default of 0
    default: 0
  - name: float-no-default
    type: float
    help: Float with no default
  - name: str-no-default
    type: string
    help: String with no default
  - name: str-empty-default
    type: string
    help: String with empty default
    default: ""
`)

func run() error {
	// Create section from YAML
	section, err := schema.NewSectionFromYAML(testYAML)
	if err != nil {
		return fmt.Errorf("NewSectionFromYAML: %w", err)
	}

	// --- Test 1: InitializeStructFromFieldDefaults ---
	fmt.Println("=== Test 1: InitializeStructFromFieldDefaults ===")
	cfg1 := &TestConfig{}
	if err := section.InitializeStructFromFieldDefaults(cfg1); err != nil {
		return fmt.Errorf("InitializeStructFromFieldDefaults: %w", err)
	}
	printConfig("After InitializeStructFromFieldDefaults", cfg1)

	// --- Test 2: DecodeSectionInto with only defaults (no user input) ---
	fmt.Println("\n=== Test 2: DecodeSectionInto (defaults only, no CLI/config) ===")
	s := schema.NewSchema(schema.WithSections(section))
	parsed := values.New()
	err = sources.Execute(
		s,
		parsed,
		sources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
	)
	if err != nil {
		return fmt.Errorf("sources.Execute: %w", err)
	}
	cfg2 := &TestConfig{}
	if err := parsed.DecodeSectionInto("test-section", cfg2); err != nil {
		return fmt.Errorf("DecodeSectionInto: %w", err)
	}
	printConfig("After DecodeSectionInto (defaults only)", cfg2)

	return nil
}

func printConfig(label string, cfg *TestConfig) {
	fmt.Printf("\n%s:\n", label)
	fmt.Printf("  WithDefault    (*int, default:42):  %s\n", ptrStr(cfg.WithDefault))
	fmt.Printf("  NoDefault      (*int, no default):  %s\n", ptrStr(cfg.NoDefault))
	fmt.Printf("  ZeroDefault    (*int, default:0):   %s\n", ptrStr(cfg.ZeroDefault))
	fmt.Printf("  FloatNoDefault (*float64, no def):  %s\n", ptrFloatStr(cfg.FloatNoDefault))
	fmt.Printf("  StrNoDefault   (*string, no def):   %s\n", ptrStringStr(cfg.StrNoDefault))
	fmt.Printf("  StrEmpty       (*string, def:\"\"):   %s\n", ptrStringStr(cfg.StrEmpty))
}

func ptrStr(p *int) string {
	if p == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%d (non-nil)", *p)
}

func ptrFloatStr(p *float64) string {
	if p == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%g (non-nil)", *p)
}

func ptrStringStr(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%q (non-nil)", *p)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}
