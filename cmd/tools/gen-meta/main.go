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
	Version     int               `yaml:"version"`
	Namespace   string            `yaml:"namespace"`
	Outputs     outputsSchema     `yaml:"outputs"`
	Templates   templatesSchema   `yaml:"templates"`
	BlockKinds  []blockKindSchema `yaml:"block_kinds"`
	KeyFamilies keyFamiliesSchema `yaml:"key_families"`
	JSEnums     []jsEnumSchema    `yaml:"js_enums"`
	JSExports   jsExportsSchema   `yaml:"js_exports"`
}

type outputsSchema struct {
	TurnsBlockKindGo string `yaml:"turns_block_kind_go"`
	TurnsKeysGo      string `yaml:"turns_keys_go"`
	EngineTurnkeysGo string `yaml:"engine_turnkeys_go"`
	TurnsDTS         string `yaml:"turns_dts"`
	GeppettoConstsGo string `yaml:"geppetto_consts_go"`
	GeppettoDTS      string `yaml:"geppetto_dts"`
}

type templatesSchema struct {
	TurnsDTS    string `yaml:"turns_dts"`
	GeppettoDTS string `yaml:"geppetto_dts"`
}

type blockKindSchema struct {
	Const string `yaml:"const"`
	Value string `yaml:"value"`
}

type keyFamiliesSchema struct {
	Data      []keySchema `yaml:"data"`
	TurnMeta  []keySchema `yaml:"turn_meta"`
	BlockMeta []keySchema `yaml:"block_meta"`
	RunMeta   []keySchema `yaml:"run_meta"`
	Payload   []keySchema `yaml:"payload"`
}

type keySchema struct {
	ValueConst string `yaml:"value_const"`
	Value      string `yaml:"value"`
	ConstType  string `yaml:"const_type"`
	TypedKey   string `yaml:"typed_key"`
	TypeExpr   string `yaml:"type_expr"`
	TypedOwner string `yaml:"typed_owner"`
}

type jsEnumSchema struct {
	Name   string              `yaml:"name"`
	Doc    string              `yaml:"doc"`
	Values []jsEnumValueSchema `yaml:"values"`
}

type jsEnumValueSchema struct {
	JSKey string `yaml:"js_key"`
	Value string `yaml:"value"`
}

type jsExportsSchema struct {
	ConstGroups []jsConstGroupSchema `yaml:"const_groups"`
}

type jsConstGroupSchema struct {
	Name   string `yaml:"name"`
	Doc    string `yaml:"doc"`
	Source string `yaml:"source"`
}

type keyRender struct {
	ValueConst string
	Value      string
	ConstType  string
	TypedKey   string
	TypeExpr   string
	Builder    string
}

type turnsKeysRenderData struct {
	Namespace      string
	Data           []keyRender
	TurnMeta       []keyRender
	BlockMeta      []keyRender
	RunMeta        []keyRender
	Payload        []keyRender
	DataTurns      []keyRender
	TurnMetaTurns  []keyRender
	BlockMetaTurns []keyRender
}

type turnsDTSRenderData struct {
	Namespace  string
	BlockKinds []blockKindSchema
	Data       []keyRender
	TurnMeta   []keyRender
	BlockMeta  []keyRender
	RunMeta    []keyRender
	Payload    []keyRender
}

type engineTurnkeysRenderData struct {
	Namespace string
	Data      []keyRender
}

type jsEnumRender struct {
	Name   string
	Doc    string
	Values []jsEnumValueSchema
}

type jsConstsRenderData struct {
	Enums []jsEnumRender
}

var (
	identifierRE      = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	nonAlnumRE        = regexp.MustCompile(`[^A-Z0-9]+`)
	multiUnderscoreRE = regexp.MustCompile(`_+`)
)

func main() {
	var (
		schemaPath = flag.String("schema", "", "Path to unified geppetto codegen schema")
		section    = flag.String("section", "all", "Section to generate: all|turns-go|engine-go|turns-dts|js-go|js-dts")
	)
	flag.Parse()

	if strings.TrimSpace(*schemaPath) == "" {
		fatalf("--schema is required")
	}

	s, err := loadSchema(*schemaPath)
	if err != nil {
		fatalf("load schema: %v", err)
	}
	if err := validateSchema(s); err != nil {
		fatalf("validate schema: %v", err)
	}

	switch *section {
	case "all", "turns-go", "engine-go", "turns-dts", "js-go", "js-dts":
	default:
		fatalf("invalid --section %q", *section)
	}

	if *section == "all" || *section == "turns-go" {
		if err := generateTurnsGo(*schemaPath, s); err != nil {
			fatalf("generate turns-go: %v", err)
		}
	}
	if *section == "all" || *section == "engine-go" {
		if err := generateEngineGo(*schemaPath, s); err != nil {
			fatalf("generate engine-go: %v", err)
		}
	}
	if *section == "all" || *section == "turns-dts" {
		if err := generateTurnsDTS(*schemaPath, s); err != nil {
			fatalf("generate turns-dts: %v", err)
		}
	}
	if *section == "all" || *section == "js-go" {
		if err := generateJSConstsGo(*schemaPath, s); err != nil {
			fatalf("generate js-go: %v", err)
		}
	}
	if *section == "all" || *section == "js-dts" {
		if err := generateJSDTS(*schemaPath, s); err != nil {
			fatalf("generate js-dts: %v", err)
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
	if strings.TrimSpace(s.Outputs.TurnsBlockKindGo) == "" || strings.TrimSpace(s.Outputs.TurnsKeysGo) == "" || strings.TrimSpace(s.Outputs.EngineTurnkeysGo) == "" || strings.TrimSpace(s.Outputs.TurnsDTS) == "" || strings.TrimSpace(s.Outputs.GeppettoConstsGo) == "" || strings.TrimSpace(s.Outputs.GeppettoDTS) == "" {
		return fmt.Errorf("all outputs paths are required")
	}
	if strings.TrimSpace(s.Templates.TurnsDTS) == "" || strings.TrimSpace(s.Templates.GeppettoDTS) == "" {
		return fmt.Errorf("all template paths are required")
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
		if _, ok := kindValues[k.Value]; ok {
			return fmt.Errorf("duplicate block kind value %q", k.Value)
		}
		kindConsts[k.Const] = struct{}{}
		kindValues[k.Value] = struct{}{}
	}

	keyGroups := map[string][]keySchema{
		"data":       s.KeyFamilies.Data,
		"turn_meta":  s.KeyFamilies.TurnMeta,
		"block_meta": s.KeyFamilies.BlockMeta,
		"run_meta":   s.KeyFamilies.RunMeta,
		"payload":    s.KeyFamilies.Payload,
	}

	valueConsts := map[string]struct{}{}
	typedKeys := map[string]struct{}{}
	for scope, list := range keyGroups {
		for i, k := range list {
			if strings.TrimSpace(k.ValueConst) == "" {
				return fmt.Errorf("%s[%d].value_const is required", scope, i)
			}
			if strings.TrimSpace(k.Value) == "" {
				return fmt.Errorf("%s[%d].value is required", scope, i)
			}
			if _, ok := valueConsts[k.ValueConst]; ok {
				return fmt.Errorf("duplicate key value_const %q", k.ValueConst)
			}
			valueConsts[k.ValueConst] = struct{}{}

			hasTyped := strings.TrimSpace(k.TypedKey) != ""
			hasTypeExpr := strings.TrimSpace(k.TypeExpr) != ""
			hasOwner := strings.TrimSpace(k.TypedOwner) != ""
			if hasTyped || hasTypeExpr || hasOwner {
				if !hasTyped || !hasTypeExpr || !hasOwner {
					return fmt.Errorf("%s[%d] typed_key, type_expr, and typed_owner must be set together", scope, i)
				}
				if k.TypedOwner != "turns" && k.TypedOwner != "engine" {
					return fmt.Errorf("%s[%d].typed_owner must be turns|engine, got %q", scope, i, k.TypedOwner)
				}
				if scope != "data" && k.TypedOwner == "engine" {
					return fmt.Errorf("%s[%d]: engine typed_owner is only valid for data keys", scope, i)
				}
				if _, ok := typedKeys[k.TypedKey]; ok {
					return fmt.Errorf("duplicate typed_key %q", k.TypedKey)
				}
				typedKeys[k.TypedKey] = struct{}{}
			}
		}
	}

	enumNames := map[string]struct{}{}
	jsEnumByName := map[string]struct{}{}
	for i, e := range s.JSEnums {
		if strings.TrimSpace(e.Name) == "" {
			return fmt.Errorf("js_enums[%d].name is required", i)
		}
		if !identifierRE.MatchString(e.Name) {
			return fmt.Errorf("js_enums[%d].name invalid: %q", i, e.Name)
		}
		if _, ok := enumNames[e.Name]; ok {
			return fmt.Errorf("duplicate js enum name %q", e.Name)
		}
		enumNames[e.Name] = struct{}{}
		jsEnumByName[e.Name] = struct{}{}
		if len(e.Values) == 0 {
			return fmt.Errorf("js_enums[%d].values empty", i)
		}
		keys := map[string]struct{}{}
		vals := map[string]struct{}{}
		for j, v := range e.Values {
			if strings.TrimSpace(v.JSKey) == "" || strings.TrimSpace(v.Value) == "" {
				return fmt.Errorf("js_enums[%d].values[%d] js_key/value required", i, j)
			}
			if !identifierRE.MatchString(v.JSKey) {
				return fmt.Errorf("js_enums[%d].values[%d].js_key invalid: %q", i, j, v.JSKey)
			}
			if _, ok := keys[v.JSKey]; ok {
				return fmt.Errorf("duplicate js_key %q in enum %q", v.JSKey, e.Name)
			}
			if _, ok := vals[v.Value]; ok {
				return fmt.Errorf("duplicate value %q in enum %q", v.Value, e.Name)
			}
			keys[v.JSKey] = struct{}{}
			vals[v.Value] = struct{}{}
		}
	}

	if len(s.JSExports.ConstGroups) == 0 {
		return fmt.Errorf("js_exports.const_groups must not be empty")
	}
	groupNames := map[string]struct{}{}
	for i, g := range s.JSExports.ConstGroups {
		if strings.TrimSpace(g.Name) == "" {
			return fmt.Errorf("js_exports.const_groups[%d].name is required", i)
		}
		if !identifierRE.MatchString(g.Name) {
			return fmt.Errorf("js_exports.const_groups[%d].name invalid: %q", i, g.Name)
		}
		if _, ok := groupNames[g.Name]; ok {
			return fmt.Errorf("duplicate js export group %q", g.Name)
		}
		groupNames[g.Name] = struct{}{}
		src := strings.TrimSpace(g.Source)
		switch {
		case src == "block_kinds":
		case strings.HasPrefix(src, "key_family:"):
			fam := strings.TrimPrefix(src, "key_family:")
			if _, ok := keyGroups[fam]; !ok {
				return fmt.Errorf("js export group %q references unknown key family %q", g.Name, fam)
			}
		case strings.HasPrefix(src, "js_enum:"):
			en := strings.TrimPrefix(src, "js_enum:")
			if _, ok := jsEnumByName[en]; !ok {
				return fmt.Errorf("js export group %q references unknown js enum %q", g.Name, en)
			}
		default:
			return fmt.Errorf("js export group %q has invalid source %q", g.Name, src)
		}
	}

	return nil
}

func generateTurnsGo(schemaPath string, s *schema) error {
	fallbackConst, fallbackValue := fallbackKind(s.BlockKinds)
	kindsData := struct {
		BlockKinds          []blockKindSchema
		UnknownDefaultConst string
		UnknownValue        string
	}{
		BlockKinds:          s.BlockKinds,
		UnknownDefaultConst: fallbackConst,
		UnknownValue:        fallbackValue,
	}
	kindsSrc, err := renderTemplate(kindsTemplate, kindsData)
	if err != nil {
		return err
	}
	if err := writeFormatted(resolveFromSchema(schemaPath, s.Outputs.TurnsBlockKindGo), kindsSrc); err != nil {
		return err
	}

	keysData := toTurnsKeysRenderData(s)
	keysSrc, err := renderTemplate(turnsKeysTemplate, keysData)
	if err != nil {
		return err
	}
	return writeFormatted(resolveFromSchema(schemaPath, s.Outputs.TurnsKeysGo), keysSrc)
}

func generateEngineGo(schemaPath string, s *schema) error {
	data := toEngineTurnkeysRenderData(s)
	src, err := renderTemplate(engineTurnkeysTemplate, data)
	if err != nil {
		return err
	}
	return writeFormatted(resolveFromSchema(schemaPath, s.Outputs.EngineTurnkeysGo), src)
}

func generateTurnsDTS(schemaPath string, s *schema) error {
	data := toTurnsDTSRenderData(s)
	tplPath := resolveFromSchema(schemaPath, s.Templates.TurnsDTS)
	src, err := renderTemplateFile(tplPath, data)
	if err != nil {
		return err
	}
	return writeFile(resolveFromSchema(schemaPath, s.Outputs.TurnsDTS), src)
}

func generateJSConstsGo(schemaPath string, s *schema) error {
	enums, err := buildJSExportEnums(s)
	if err != nil {
		return err
	}
	src, err := renderTemplate(jsConstsTemplate, jsConstsRenderData{Enums: enums})
	if err != nil {
		return err
	}
	return writeFormatted(resolveFromSchema(schemaPath, s.Outputs.GeppettoConstsGo), src)
}

func generateJSDTS(schemaPath string, s *schema) error {
	enums, err := buildJSExportEnums(s)
	if err != nil {
		return err
	}
	tplPath := resolveFromSchema(schemaPath, s.Templates.GeppettoDTS)
	src, err := renderTemplateFile(tplPath, jsConstsRenderData{Enums: enums})
	if err != nil {
		return err
	}
	return writeFile(resolveFromSchema(schemaPath, s.Outputs.GeppettoDTS), src)
}

func toTurnsKeysRenderData(s *schema) turnsKeysRenderData {
	data := convertKeys(s.KeyFamilies.Data, "data")
	turnMeta := convertKeys(s.KeyFamilies.TurnMeta, "turn_meta")
	blockMeta := convertKeys(s.KeyFamilies.BlockMeta, "block_meta")
	runMeta := convertKeys(s.KeyFamilies.RunMeta, "run_meta")
	payload := convertKeys(s.KeyFamilies.Payload, "payload")

	return turnsKeysRenderData{
		Namespace:      s.Namespace,
		Data:           data,
		TurnMeta:       turnMeta,
		BlockMeta:      blockMeta,
		RunMeta:        runMeta,
		Payload:        payload,
		DataTurns:      filterByOwner(data, s.KeyFamilies.Data, "turns"),
		TurnMetaTurns:  filterByOwner(turnMeta, s.KeyFamilies.TurnMeta, "turns"),
		BlockMetaTurns: filterByOwner(blockMeta, s.KeyFamilies.BlockMeta, "turns"),
	}
}

func toEngineTurnkeysRenderData(s *schema) engineTurnkeysRenderData {
	all := convertKeys(s.KeyFamilies.Data, "data")
	return engineTurnkeysRenderData{
		Namespace: s.Namespace,
		Data:      filterByOwner(all, s.KeyFamilies.Data, "engine"),
	}
}

func toTurnsDTSRenderData(s *schema) turnsDTSRenderData {
	return turnsDTSRenderData{
		Namespace:  s.Namespace,
		BlockKinds: s.BlockKinds,
		Data:       convertKeys(s.KeyFamilies.Data, "data"),
		TurnMeta:   convertKeys(s.KeyFamilies.TurnMeta, "turn_meta"),
		BlockMeta:  convertKeys(s.KeyFamilies.BlockMeta, "block_meta"),
		RunMeta:    convertKeys(s.KeyFamilies.RunMeta, "run_meta"),
		Payload:    convertKeys(s.KeyFamilies.Payload, "payload"),
	}
}

func convertKeys(keys []keySchema, scope string) []keyRender {
	out := make([]keyRender, 0, len(keys))
	for _, k := range keys {
		out = append(out, keyRender{
			ValueConst: k.ValueConst,
			Value:      k.Value,
			ConstType:  k.ConstType,
			TypedKey:   k.TypedKey,
			TypeExpr:   k.TypeExpr,
			Builder:    builderForScope(scope),
		})
	}
	return out
}

func filterByOwner(rendered []keyRender, source []keySchema, owner string) []keyRender {
	out := make([]keyRender, 0)
	for i := range source {
		if source[i].TypedOwner == owner && source[i].TypedKey != "" {
			out = append(out, rendered[i])
		}
	}
	return out
}

func buildJSExportEnums(s *schema) ([]jsEnumRender, error) {
	enumByName := map[string]jsEnumSchema{}
	for _, e := range s.JSEnums {
		enumByName[e.Name] = e
	}
	familyByName := map[string][]keySchema{
		"data":       s.KeyFamilies.Data,
		"turn_meta":  s.KeyFamilies.TurnMeta,
		"block_meta": s.KeyFamilies.BlockMeta,
		"run_meta":   s.KeyFamilies.RunMeta,
		"payload":    s.KeyFamilies.Payload,
	}

	out := make([]jsEnumRender, 0, len(s.JSExports.ConstGroups))
	for _, g := range s.JSExports.ConstGroups {
		group := jsEnumRender{Name: g.Name, Doc: g.Doc}
		src := strings.TrimSpace(g.Source)
		switch {
		case src == "block_kinds":
			if group.Doc == "" {
				group.Doc = "Block kinds"
			}
			for _, bk := range s.BlockKinds {
				group.Values = append(group.Values, jsEnumValueSchema{JSKey: valueToJSKey(bk.Value), Value: bk.Value})
			}
		case strings.HasPrefix(src, "key_family:"):
			fam := strings.TrimPrefix(src, "key_family:")
			if group.Doc == "" {
				group.Doc = fmt.Sprintf("Canonical %s keys", fam)
			}
			for _, k := range familyByName[fam] {
				group.Values = append(group.Values, jsEnumValueSchema{JSKey: valueToJSKey(k.Value), Value: k.Value})
			}
		case strings.HasPrefix(src, "js_enum:"):
			enName := strings.TrimPrefix(src, "js_enum:")
			e, ok := enumByName[enName]
			if !ok {
				return nil, fmt.Errorf("unknown js enum %q", enName)
			}
			if group.Doc == "" {
				group.Doc = e.Doc
			}
			group.Values = append(group.Values, e.Values...)
		default:
			return nil, fmt.Errorf("unknown group source %q", src)
		}
		if len(group.Values) == 0 {
			return nil, fmt.Errorf("js export group %q resolved to zero values", g.Name)
		}
		out = append(out, group)
	}
	return out, nil
}

func fallbackKind(blockKinds []blockKindSchema) (string, string) {
	for _, k := range blockKinds {
		if k.Value == "other" {
			return k.Const, k.Value
		}
	}
	last := blockKinds[len(blockKinds)-1]
	return last.Const, last.Value
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

func resolveFromSchema(schemaPath, rel string) string {
	if filepath.IsAbs(rel) {
		return rel
	}
	return filepath.Clean(filepath.Join(filepath.Dir(schemaPath), rel))
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
	_, _ = fmt.Fprintf(os.Stderr, "gen-meta: "+format+"\n", args...)
	os.Exit(1)
}

const kindsTemplate = `// Code generated by cmd/gen-meta. DO NOT EDIT.

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

const turnsKeysTemplate = `// Code generated by cmd/gen-meta. DO NOT EDIT.

package turns

// Canonical namespace for geppetto-owned turn data and metadata keys.
const GeppettoNamespaceKey = "{{ .Namespace }}"

// Canonical keys used in Block.Payload maps.
const (
{{- range .Payload }}
	{{- if .ConstType }}
	{{ .ValueConst }} {{ .ConstType }} = "{{ .Value }}"
	{{- else }}
	{{ .ValueConst }} = "{{ .Value }}"
	{{- end }}
{{- end }}
)

// Canonical keys used in Run.Metadata maps.
const (
{{- range .RunMeta }}
	{{- if .ConstType }}
	{{ .ValueConst }} {{ .ConstType }} = "{{ .Value }}"
	{{- else }}
	{{ .ValueConst }} = "{{ .Value }}"
	{{- end }}
{{- end }}
)

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

// Typed keys for Turn.Data owned by turns package.
var (
{{- range .DataTurns }}
	{{ .TypedKey }} = {{ .Builder }}[{{ .TypeExpr }}](GeppettoNamespaceKey, {{ .ValueConst }}, 1)
{{- end }}
)

// Typed keys for Turn.Metadata owned by turns package.
var (
{{- range .TurnMetaTurns }}
	{{ .TypedKey }} = {{ .Builder }}[{{ .TypeExpr }}](GeppettoNamespaceKey, {{ .ValueConst }}, 1)
{{- end }}
)

// Typed keys for Block.Metadata owned by turns package.
var (
{{- range .BlockMetaTurns }}
	{{ .TypedKey }} = {{ .Builder }}[{{ .TypeExpr }}](GeppettoNamespaceKey, {{ .ValueConst }}, 1)
{{- end }}
)
`

const engineTurnkeysTemplate = `// Code generated by cmd/gen-meta. DO NOT EDIT.

package engine

import "github.com/go-go-golems/geppetto/pkg/turns"

// Typed turn keys owned by inference/engine package.
var (
{{- range .Data }}
	{{ .TypedKey }} = turns.DataK[{{ .TypeExpr }}](turns.GeppettoNamespaceKey, turns.{{ .ValueConst }}, 1)
{{- end }}
)
`

const jsConstsTemplate = `// Code generated by cmd/gen-meta. DO NOT EDIT.

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
