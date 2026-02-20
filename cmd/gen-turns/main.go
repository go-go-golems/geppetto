package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

type schema struct {
	Namespace  string            `yaml:"namespace"`
	BlockKinds []blockKindSchema `yaml:"block_kinds"`
	Keys       []keySchema       `yaml:"keys"`
}

type blockKindSchema struct {
	Const string `yaml:"const"`
	Value string `yaml:"value"`
}

type keySchema struct {
	Scope      string `yaml:"scope"`
	ValueConst string `yaml:"value_const"`
	Value      string `yaml:"value"`
	TypedKey   string `yaml:"typed_key"`
	TypeExpr   string `yaml:"type_expr"`
}

type keyRender struct {
	Scope      string
	ValueConst string
	Value      string
	TypedKey   string
	TypeExpr   string
	Builder    string
}

type kindsRenderData struct {
	BlockKinds          []blockKindSchema
	UnknownDefaultConst string
	UnknownValue        string
}

type keysRenderData struct {
	Namespace string
	Data      []keyRender
	TurnMeta  []keyRender
	BlockMeta []keyRender
}

type dtsRenderData struct {
	Namespace  string
	BlockKinds []blockKindSchema
	Data       []keyRender
	TurnMeta   []keyRender
	BlockMeta  []keyRender
}

func main() {
	var (
		schemaPath  = flag.String("schema", "", "Path to turns codegen schema YAML")
		outDir      = flag.String("out", "", "Output directory for Go outputs")
		section     = flag.String("section", "all", "Section to generate: all|kinds|keys|dts")
		dtsTemplate = flag.String("dts-template", "", "Path to turns .d.ts template")
		dtsOutPath  = flag.String("dts-out", "", "Path to generated turns .d.ts file")
	)
	flag.Parse()

	if *schemaPath == "" {
		fatalf("--schema is required")
	}
	if *section != "dts" && *outDir == "" {
		fatalf("--out is required")
	}

	s, err := loadSchema(*schemaPath)
	if err != nil {
		fatalf("load schema: %v", err)
	}

	if err := validateSchema(s); err != nil {
		fatalf("validate schema: %v", err)
	}

	switch *section {
	case "all", "kinds", "keys", "dts":
	default:
		fatalf("invalid --section %q (expected all|kinds|keys|dts)", *section)
	}

	needsGoOut := *section == "all" || *section == "kinds" || *section == "keys"
	needsDTS := *section == "all" || *section == "dts"
	if needsGoOut {
		if err := os.MkdirAll(*outDir, 0o755); err != nil {
			fatalf("mkdir out: %v", err)
		}
	}
	if needsDTS {
		if *dtsTemplate == "" {
			fatalf("--dts-template is required for section %q", *section)
		}
		if *dtsOutPath == "" {
			fatalf("--dts-out is required for section %q", *section)
		}
	}

	if *section == "all" || *section == "kinds" {
		fallbackConst, fallbackValue := fallbackKind(s.BlockKinds)
		kindsData := kindsRenderData{
			BlockKinds:          s.BlockKinds,
			UnknownDefaultConst: fallbackConst,
			UnknownValue:        fallbackValue,
		}
		src, err := renderTemplate(kindsTemplate, kindsData)
		if err != nil {
			fatalf("render block kinds: %v", err)
		}
		if err := writeFormatted(filepath.Join(*outDir, "block_kind_gen.go"), src); err != nil {
			fatalf("write block_kind_gen.go: %v", err)
		}
	}

	if *section == "all" || *section == "keys" {
		keysData := toKeysRenderData(s)
		src, err := renderTemplate(keysTemplate, keysData)
		if err != nil {
			fatalf("render keys: %v", err)
		}
		if err := writeFormatted(filepath.Join(*outDir, "keys_gen.go"), src); err != nil {
			fatalf("write keys_gen.go: %v", err)
		}
	}

	if needsDTS {
		data := toDTSRenderData(s)
		src, err := renderTemplateFile(*dtsTemplate, data)
		if err != nil {
			fatalf("render dts: %v", err)
		}
		if err := writeFile(*dtsOutPath, src); err != nil {
			fatalf("write dts output: %v", err)
		}
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

func validateSchema(s *schema) error {
	if s == nil {
		return fmt.Errorf("schema is nil")
	}
	if strings.TrimSpace(s.Namespace) == "" {
		return fmt.Errorf("namespace is required")
	}
	if len(s.BlockKinds) == 0 {
		return fmt.Errorf("at least one block kind is required")
	}

	kindConsts := map[string]struct{}{}
	kindValues := map[string]struct{}{}
	for i, k := range s.BlockKinds {
		if strings.TrimSpace(k.Const) == "" {
			return fmt.Errorf("block_kinds[%d].const is required", i)
		}
		if strings.TrimSpace(k.Value) == "" {
			return fmt.Errorf("block_kinds[%d].value is required", i)
		}
		if _, ok := kindConsts[k.Const]; ok {
			return fmt.Errorf("duplicate block kind const %q", k.Const)
		}
		kindConsts[k.Const] = struct{}{}
		if _, ok := kindValues[k.Value]; ok {
			return fmt.Errorf("duplicate block kind value %q", k.Value)
		}
		kindValues[k.Value] = struct{}{}
	}

	valueConsts := map[string]struct{}{}
	typedKeys := map[string]struct{}{}
	for i, k := range s.Keys {
		scope := strings.TrimSpace(k.Scope)
		if scope != "data" && scope != "turn_meta" && scope != "block_meta" {
			return fmt.Errorf("keys[%d].scope invalid: %q", i, k.Scope)
		}
		if strings.TrimSpace(k.ValueConst) == "" {
			return fmt.Errorf("keys[%d].value_const is required", i)
		}
		if strings.TrimSpace(k.Value) == "" {
			return fmt.Errorf("keys[%d].value is required", i)
		}
		if _, ok := valueConsts[k.ValueConst]; ok {
			return fmt.Errorf("duplicate key value_const %q", k.ValueConst)
		}
		valueConsts[k.ValueConst] = struct{}{}

		if strings.TrimSpace(k.TypedKey) == "" {
			if strings.TrimSpace(k.TypeExpr) != "" {
				return fmt.Errorf("keys[%d] has type_expr but no typed_key", i)
			}
			continue
		}
		if strings.TrimSpace(k.TypeExpr) == "" {
			return fmt.Errorf("keys[%d] has typed_key %q but empty type_expr", i, k.TypedKey)
		}
		if _, ok := typedKeys[k.TypedKey]; ok {
			return fmt.Errorf("duplicate typed_key %q", k.TypedKey)
		}
		typedKeys[k.TypedKey] = struct{}{}
	}

	return nil
}

func fallbackKind(blockKinds []blockKindSchema) (string, string) {
	for _, k := range blockKinds {
		if k.Value == "other" {
			return k.Const, k.Value
		}
	}
	k := blockKinds[len(blockKinds)-1]
	return k.Const, k.Value
}

func toKeysRenderData(s *schema) keysRenderData {
	out := keysRenderData{Namespace: s.Namespace}
	for _, k := range s.Keys {
		r := keyRender{
			Scope:      k.Scope,
			ValueConst: k.ValueConst,
			Value:      k.Value,
			TypedKey:   k.TypedKey,
			TypeExpr:   k.TypeExpr,
			Builder:    builderForScope(k.Scope),
		}
		switch k.Scope {
		case "data":
			out.Data = append(out.Data, r)
		case "turn_meta":
			out.TurnMeta = append(out.TurnMeta, r)
		case "block_meta":
			out.BlockMeta = append(out.BlockMeta, r)
		}
	}
	return out
}

func toDTSRenderData(s *schema) dtsRenderData {
	keysData := toKeysRenderData(s)
	return dtsRenderData{
		Namespace:  s.Namespace,
		BlockKinds: s.BlockKinds,
		Data:       keysData.Data,
		TurnMeta:   keysData.TurnMeta,
		BlockMeta:  keysData.BlockMeta,
	}
}

func builderForScope(scope string) string {
	switch scope {
	case "data":
		return "DataK"
	case "turn_meta":
		return "TurnMetaK"
	case "block_meta":
		return "BlockMetaK"
	default:
		return ""
	}
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

func renderTemplateFile(path string, data any) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return renderTemplate(string(b), data)
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
	_, _ = fmt.Fprintf(os.Stderr, "gen-turns: "+format+"\n", args...)
	os.Exit(1)
}

const kindsTemplate = `// Code generated by cmd/gen-turns. DO NOT EDIT.

package turns

import (
	"gopkg.in/yaml.v3"
	"strings"
)

// BlockKind represents the kind of a block within a Turn.
type BlockKind int

const (
{{- range $i, $k := .BlockKinds }}
	{{ $k.Const }} BlockKind = {{ $i }}
{{- end }}
)

// String returns a human-readable identifier for the BlockKind.
func (k BlockKind) String() string {
	switch k {
{{- range .BlockKinds }}
	case {{ .Const }}:
		return "{{ .Value }}"
{{- end }}
	default:
		return "{{ .UnknownValue }}"
	}
}

// YAML serialization for BlockKind using stable string names
func (k BlockKind) MarshalYAML() (interface{}, error) {
	return k.String(), nil
}

func (k *BlockKind) UnmarshalYAML(value *yaml.Node) error {
	if value == nil {
		*k = {{ .UnknownDefaultConst }}
		return nil
	}
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(s)) {
{{- range .BlockKinds }}
	case "{{ .Value }}":
		*k = {{ .Const }}
{{- end }}
	case "":
		*k = {{ .UnknownDefaultConst }}
	default:
		*k = {{ .UnknownDefaultConst }}
	}
	return nil
}
`

const keysTemplate = `// Code generated by cmd/gen-turns. DO NOT EDIT.

package turns

// Canonical namespace for geppetto-owned turn data and metadata keys.
const GeppettoNamespaceKey = "{{ .Namespace }}"

// Canonical value keys (scoped to GeppettoNamespaceKey).
const (
	// Turn.Data
{{- range .Data }}
	{{ .ValueConst }} = "{{ .Value }}"
{{- end }}

	// Turn.Metadata
{{- range .TurnMeta }}
	{{ .ValueConst }} = "{{ .Value }}"
{{- end }}

	// Block.Metadata
{{- range .BlockMeta }}
	{{ .ValueConst }} = "{{ .Value }}"
{{- end }}
)

// Typed keys for Turn.Data (geppetto-owned).
//
// Note: KeyToolConfig lives in ` + "`geppetto/pkg/inference/engine`" + ` to avoid the import cycle:
// turns -> engine (ToolConfig type) -> turns (Engine interface uses *turns.Turn)
var (
{{- range .Data }}
{{- if .TypedKey }}
	{{ .TypedKey }} = {{ .Builder }}[{{ .TypeExpr }}](GeppettoNamespaceKey, {{ .ValueConst }}, 1)
{{- end }}
{{- end }}
)

// Typed keys for Turn.Metadata (geppetto-owned).
var (
{{- range .TurnMeta }}
{{- if .TypedKey }}
	{{ .TypedKey }} = {{ .Builder }}[{{ .TypeExpr }}](GeppettoNamespaceKey, {{ .ValueConst }}, 1)
{{- end }}
{{- end }}
)

// Typed keys for Block.Metadata (geppetto-owned).
var (
{{- range .BlockMeta }}
{{- if .TypedKey }}
	{{ .TypedKey }} = {{ .Builder }}[{{ .TypeExpr }}](GeppettoNamespaceKey, {{ .ValueConst }}, 1)
{{- end }}
{{- end }}
)
`
