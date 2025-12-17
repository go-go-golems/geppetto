package turnsdatalint

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const (
	// DefaultTurnDataKeyType is the fully-qualified named type used by Geppetto for Turn.Data keys.
	DefaultTurnDataKeyType = "github.com/go-go-golems/geppetto/pkg/turns.TurnDataKey"
	// DefaultTurnMetadataKeyType is the fully-qualified named type used by Geppetto for Turn.Metadata keys.
	DefaultTurnMetadataKeyType = "github.com/go-go-golems/geppetto/pkg/turns.TurnMetadataKey"
	// DefaultBlockMetadataKeyType is the fully-qualified named type used by Geppetto for Block.Metadata keys.
	DefaultBlockMetadataKeyType = "github.com/go-go-golems/geppetto/pkg/turns.BlockMetadataKey"
	// DefaultRunMetadataKeyType is the fully-qualified named type used by Geppetto for Run.Metadata keys.
	DefaultRunMetadataKeyType = "github.com/go-go-golems/geppetto/pkg/turns.RunMetadataKey"
)

var (
	dataKeyTypeFlag          string
	turnMetadataKeyTypeFlag  string
	blockMetadataKeyTypeFlag string
	runMetadataKeyTypeFlag   string
)

// Analyzer enforces that Turn.Data[...] map access uses a const of the configured key type.
//
// This prevents ad-hoc conversions like turns.TurnDataKey("raw") or variables, encouraging use
// of canonical, typed constants (e.g., turns.DataKeyToolRegistry).
//
// Note: raw string literals like turn.Data["foo"] can compile in Go because untyped string
// constants may be implicitly converted to a defined string type; this analyzer flags them too.
var Analyzer = &analysis.Analyzer{
	Name:     "turnsdatalint",
	Doc:      "require Turn.{Data,Metadata}[...] and Block.{Payload,Metadata}[...] indexes to use const keys (no raw strings, conversions, or variables)",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func init() {
	Analyzer.Flags.StringVar(
		&dataKeyTypeFlag,
		"data-keytype",
		DefaultTurnDataKeyType,
		`fully-qualified named type like "github.com/go-go-golems/geppetto/pkg/turns.TurnDataKey"`,
	)
	Analyzer.Flags.StringVar(
		&turnMetadataKeyTypeFlag,
		"turn-metadata-keytype",
		DefaultTurnMetadataKeyType,
		`fully-qualified named type like "github.com/go-go-golems/geppetto/pkg/turns.TurnMetadataKey"`,
	)
	Analyzer.Flags.StringVar(
		&blockMetadataKeyTypeFlag,
		"block-metadata-keytype",
		DefaultBlockMetadataKeyType,
		`fully-qualified named type like "github.com/go-go-golems/geppetto/pkg/turns.BlockMetadataKey"`,
	)
	Analyzer.Flags.StringVar(
		&runMetadataKeyTypeFlag,
		"run-metadata-keytype",
		DefaultRunMetadataKeyType,
		`fully-qualified named type like "github.com/go-go-golems/geppetto/pkg/turns.RunMetadataKey"`,
	)
}

func run(pass *analysis.Pass) (any, error) {
	wantTypedKeyTypes := newWantedNamedTypes(
		dataKeyTypeFlag,
		turnMetadataKeyTypeFlag,
		blockMetadataKeyTypeFlag,
		runMetadataKeyTypeFlag,
	)

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{(*ast.IndexExpr)(nil)}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		idx := n.(*ast.IndexExpr)

		// Match <something>.{Data,Metadata,Payload}[...]
		sel, ok := idx.X.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil {
			return
		}
		switch sel.Sel.Name {
		case "Data", "Metadata", "Payload":
			// ok
		default:
			return
		}

		selection := pass.TypesInfo.Selections[sel]
		if selection == nil || selection.Kind() != types.FieldVal {
			return
		}

		m, ok := selection.Type().(*types.Map)
		if !ok {
			return
		}

		// Special-case Block.Payload which is map[string]any: require a const string key (no literals/vars).
		if sel.Sel.Name == "Payload" && isStringKeyMap(m) {
			if isAllowedConstStringKey(pass, idx.Index) {
				return
			}
			pass.Reportf(
				idx.Lbrack,
				`Payload key must be a const string (not a raw string literal, conversion, or variable)`,
			)
			return
		}

		// For typed-key maps (Turn.Data, Turn.Metadata, Block.Metadata, Run.Metadata), require const of the map key type.
		keyNamed, ok := m.Key().(*types.Named)
		if !ok {
			return
		}
		keyTypeStr := namedTypeToQualifiedString(keyNamed)
		if !wantTypedKeyTypes[keyTypeStr] {
			return
		}

		wantPkgPath, wantName, ok := splitQualifiedType(keyTypeStr)
		if !ok {
			return
		}

		if isAllowedConstKey(pass, idx.Index, wantPkgPath, wantName) {
			return
		}

		pass.Reportf(
			idx.Lbrack,
			`%s key must be a const of type %q (not a raw string literal, conversion, or variable)`,
			sel.Sel.Name,
			keyTypeStr,
		)
	})

	return nil, nil
}

func isAllowedConstKey(pass *analysis.Pass, e ast.Expr, wantPkgPath, wantName string) bool {
	e = unwrapParens(e)

	// allow unqualified const (Foo) or qualified const (pkg.Foo)
	switch t := e.(type) {
	case *ast.Ident:
		return objIsAllowedConst(pass.TypesInfo.ObjectOf(t), wantPkgPath, wantName)
	case *ast.SelectorExpr:
		// for pkg.Foo, the object is on Sel
		return objIsAllowedConst(pass.TypesInfo.ObjectOf(t.Sel), wantPkgPath, wantName)
	default:
		return false
	}
}

func objIsAllowedConst(obj types.Object, wantPkgPath, wantName string) bool {
	c, ok := obj.(*types.Const)
	if !ok {
		return false
	}

	named, ok := c.Type().(*types.Named)
	if !ok {
		return false
	}

	if wantName != "" && named.Obj().Name() != wantName {
		return false
	}

	if wantPkgPath != "" && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() != wantPkgPath {
		return false
	}

	return true
}

func namedTypeMatches(t types.Type, wantPkgPath, wantName string) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}

	if wantName != "" && named.Obj().Name() != wantName {
		return false
	}

	if wantPkgPath == "" {
		return true
	}

	p := named.Obj().Pkg()
	return p != nil && p.Path() == wantPkgPath
}

func unwrapParens(e ast.Expr) ast.Expr {
	for {
		p, ok := e.(*ast.ParenExpr)
		if !ok {
			return e
		}
		e = p.X
	}
}

func splitQualifiedType(s string) (string, string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", "", false
	}
	i := strings.LastIndex(s, ".")
	if i <= 0 || i >= len(s)-1 {
		return "", "", false
	}
	return s[:i], s[i+1:], true
}

func newWantedNamedTypes(qualifiedTypes ...string) map[string]bool {
	out := map[string]bool{}
	for _, qt := range qualifiedTypes {
		qt = strings.TrimSpace(qt)
		if qt == "" {
			continue
		}
		if _, _, ok := splitQualifiedType(qt); !ok {
			continue
		}
		out[qt] = true
	}
	return out
}

func namedTypeToQualifiedString(n *types.Named) string {
	if n == nil || n.Obj() == nil || n.Obj().Pkg() == nil {
		return ""
	}
	return n.Obj().Pkg().Path() + "." + n.Obj().Name()
}

func isStringKeyMap(m *types.Map) bool {
	if m == nil {
		return false
	}
	b, ok := m.Key().Underlying().(*types.Basic)
	return ok && b.Kind() == types.String
}

func isAllowedConstStringKey(pass *analysis.Pass, e ast.Expr) bool {
	e = unwrapParens(e)

	// allow unqualified const (Foo) or qualified const (pkg.Foo)
	switch t := e.(type) {
	case *ast.Ident:
		return objIsConstString(pass.TypesInfo.ObjectOf(t))
	case *ast.SelectorExpr:
		return objIsConstString(pass.TypesInfo.ObjectOf(t.Sel))
	default:
		return false
	}
}

func objIsConstString(obj types.Object) bool {
	c, ok := obj.(*types.Const)
	if !ok {
		return false
	}

	// Accept both untyped string constants and typed string constants.
	b, ok := c.Type().Underlying().(*types.Basic)
	if !ok {
		return false
	}
	return b.Info()&types.IsString != 0
}
