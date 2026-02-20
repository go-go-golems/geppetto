package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

type schema struct {
	Enums []enumSchema `yaml:"enums"`
}

type turnsSchema struct {
	BlockKinds []turnsBlockKindSchema `yaml:"block_kinds"`
	Keys       []turnsKeySchema       `yaml:"keys"`
}

type enumSchema struct {
	Name   string            `yaml:"name"`
	Doc    string            `yaml:"doc"`
	Values []enumValueSchema `yaml:"values"`
}

type enumValueSchema struct {
	JSKey   string `yaml:"js_key"`
	Value   string `yaml:"value"`
	GoConst string `yaml:"go_const"`
}

type turnsBlockKindSchema struct {
	Value string `yaml:"value"`
}

type turnsKeySchema struct {
	Scope string `yaml:"scope"`
	Value string `yaml:"value"`
}

var (
	identifierRE      = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	nonAlnumRE        = regexp.MustCompile(`[^A-Z0-9]+`)
	multiUnderscoreRE = regexp.MustCompile(`_+`)
)

func main() {
	var (
		schemaPath      = flag.String("schema", "", "Path to JS API codegen schema YAML")
		turnsSchemaPath = flag.String("turns-schema", "", "Optional path to turns codegen schema YAML for importing turns constants")
		templatePath    = flag.String("dts-template", "", "Path to geppetto.d.ts template")
		goOutPath       = flag.String("go-out", "", "Path to generated Go file (consts_gen.go)")
		tsOutPath       = flag.String("ts-out", "", "Path to generated TypeScript declaration file")
	)
	flag.Parse()

	if strings.TrimSpace(*schemaPath) == "" {
		fatalf("--schema is required")
	}
	if strings.TrimSpace(*templatePath) == "" {
		fatalf("--dts-template is required")
	}
	if strings.TrimSpace(*goOutPath) == "" {
		fatalf("--go-out is required")
	}
	if strings.TrimSpace(*tsOutPath) == "" {
		fatalf("--ts-out is required")
	}

	s, err := loadSchema(*schemaPath)
	if err != nil {
		fatalf("load schema: %v", err)
	}
	if strings.TrimSpace(*turnsSchemaPath) != "" {
		ts, err := loadTurnsSchema(*turnsSchemaPath)
		if err != nil {
			fatalf("load turns schema: %v", err)
		}
		s.Enums = mergeEnums(s.Enums, importedTurnsEnums(ts))
	}
	if err := validateSchema(s); err != nil {
		fatalf("validate schema: %v", err)
	}

	goSrc, err := renderTemplate(goConstsTemplate, s)
	if err != nil {
		fatalf("render Go output: %v", err)
	}
	if err := writeFormatted(*goOutPath, goSrc); err != nil {
		fatalf("write Go output: %v", err)
	}

	tsSrc, err := renderTemplateFile(*templatePath, s)
	if err != nil {
		fatalf("render TypeScript output: %v", err)
	}
	if err := writeFile(*tsOutPath, tsSrc); err != nil {
		fatalf("write TypeScript output: %v", err)
	}
}

func loadSchema(path string) (*schema, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s schema
	if err := yaml.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func loadTurnsSchema(path string) (*turnsSchema, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s turnsSchema
	if err := yaml.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func importedTurnsEnums(s *turnsSchema) []enumSchema {
	if s == nil {
		return nil
	}

	return []enumSchema{
		{
			Name:   "BlockKind",
			Doc:    "The kind of a block within a Turn (from turns schema)",
			Values: enumValuesFromBlockKinds(s.BlockKinds),
		},
		{
			Name:   "TurnDataKeys",
			Doc:    "Canonical Turn.Data value keys (from turns schema)",
			Values: enumValuesFromKeysByScope(s.Keys, "data"),
		},
		{
			Name:   "MetadataKeys",
			Doc:    "Standard turn metadata key names (from turns schema)",
			Values: enumValuesFromKeysByScope(s.Keys, "turn_meta"),
		},
		{
			Name:   "TurnMetadataKeys",
			Doc:    "Canonical Turn.Metadata value keys (from turns schema)",
			Values: enumValuesFromKeysByScope(s.Keys, "turn_meta"),
		},
		{
			Name:   "BlockMetadataKeys",
			Doc:    "Canonical Block.Metadata value keys (from turns schema)",
			Values: enumValuesFromKeysByScope(s.Keys, "block_meta"),
		},
	}
}

func enumValuesFromBlockKinds(kinds []turnsBlockKindSchema) []enumValueSchema {
	out := make([]enumValueSchema, 0, len(kinds))
	for _, k := range kinds {
		v := strings.TrimSpace(k.Value)
		if v == "" {
			continue
		}
		out = append(out, enumValueSchema{
			JSKey: valueToJSKey(v),
			Value: v,
		})
	}
	return out
}

func enumValuesFromKeysByScope(keys []turnsKeySchema, scope string) []enumValueSchema {
	out := make([]enumValueSchema, 0, len(keys))
	for _, k := range keys {
		if strings.TrimSpace(k.Scope) != scope {
			continue
		}
		v := strings.TrimSpace(k.Value)
		if v == "" {
			continue
		}
		out = append(out, enumValueSchema{
			JSKey: valueToJSKey(v),
			Value: v,
		})
	}
	return out
}

func mergeEnums(base, imported []enumSchema) []enumSchema {
	if len(imported) == 0 {
		return base
	}
	override := make(map[string]struct{}, len(imported))
	for _, e := range imported {
		override[e.Name] = struct{}{}
	}

	out := make([]enumSchema, 0, len(base)+len(imported))
	for _, e := range base {
		if _, ok := override[e.Name]; ok {
			continue
		}
		out = append(out, e)
	}
	out = append(out, imported...)
	return out
}

func valueToJSKey(v string) string {
	s := strings.ToUpper(strings.TrimSpace(v))
	s = nonAlnumRE.ReplaceAllString(s, "_")
	s = multiUnderscoreRE.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if s == "" {
		return "VALUE"
	}
	if s[0] >= '0' && s[0] <= '9' {
		return "K_" + s
	}
	return s
}

func validateSchema(s *schema) error {
	if s == nil {
		return fmt.Errorf("schema is nil")
	}
	if len(s.Enums) == 0 {
		return fmt.Errorf("at least one enum is required")
	}

	enumNames := map[string]struct{}{}
	for i, e := range s.Enums {
		if strings.TrimSpace(e.Name) == "" {
			return fmt.Errorf("enums[%d].name is required", i)
		}
		if !identifierRE.MatchString(e.Name) {
			return fmt.Errorf("enums[%d].name %q is not a valid identifier", i, e.Name)
		}
		if _, ok := enumNames[e.Name]; ok {
			return fmt.Errorf("duplicate enum name %q", e.Name)
		}
		enumNames[e.Name] = struct{}{}

		if len(e.Values) == 0 {
			return fmt.Errorf("enums[%d].values must have at least one value", i)
		}
		keys := map[string]struct{}{}
		values := map[string]struct{}{}
		for j, v := range e.Values {
			if strings.TrimSpace(v.JSKey) == "" {
				return fmt.Errorf("enums[%d].values[%d].js_key is required", i, j)
			}
			if !identifierRE.MatchString(v.JSKey) {
				return fmt.Errorf("enums[%d].values[%d].js_key %q is not a valid identifier", i, j, v.JSKey)
			}
			if _, ok := keys[v.JSKey]; ok {
				return fmt.Errorf("duplicate js_key %q in enum %q", v.JSKey, e.Name)
			}
			keys[v.JSKey] = struct{}{}

			if strings.TrimSpace(v.Value) == "" {
				return fmt.Errorf("enums[%d].values[%d].value is required", i, j)
			}
			if _, ok := values[v.Value]; ok {
				return fmt.Errorf("duplicate value %q in enum %q", v.Value, e.Name)
			}
			values[v.Value] = struct{}{}
		}
	}
	return nil
}

func renderTemplateFile(path string, data any) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return renderTemplate(string(b), data)
}

func renderTemplate(tplSrc string, data any) ([]byte, error) {
	tpl, err := template.New("gen").Parse(tplSrc)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	if err := tpl.Execute(&b, data); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func writeFormatted(path string, src []byte) error {
	formatted, err := format.Source(src)
	if err != nil {
		return fmt.Errorf("format source for %s: %w", path, err)
	}
	return writeFile(path, formatted)
}

func writeFile(path string, src []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return os.WriteFile(path, src, 0o644)
}

func fatalf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, "gen-js-api: "+format+"\n", args...)
	os.Exit(1)
}

const goConstsTemplate = `// Code generated by cmd/gen-js-api. DO NOT EDIT.

package geppetto

import "github.com/dop251/goja"

// installConsts installs the gp.consts namespace on the module exports.
func (m *moduleRuntime) installConsts(exports *goja.Object) {
	constsObj := m.vm.NewObject()
{{- range .Enums }}

	// {{ .Name }}{{ if .Doc }} - {{ .Doc }}{{ end }}
	{
		o := m.vm.NewObject()
{{- range .Values }}
		m.mustSet(o, "{{ .JSKey }}", "{{ .Value }}")
{{- end }}
		m.mustSet(constsObj, "{{ .Name }}", o)
	}
{{- end }}

	m.mustSet(exports, "consts", constsObj)
}
`
