package turnsdatalint

import (
	"go/ast"
	"go/token"
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

// Analyzer enforces that Turn/Run/Block typed-key map access uses a typed key expression,
// and that Block.Payload (map[string]any) uses const string keys.
//
// For typed-key maps, this prevents raw string drift (e.g. t.Data["foo"]) while still allowing
// normal Go patterns like typed conversions, variables, and parameters.
//
// Note: raw string literals like turn.Data["foo"] can compile in Go because untyped string
// constants may be implicitly converted to a defined string type; this analyzer flags them too.
var Analyzer = &analysis.Analyzer{
	Name:     "turnsdatalint",
	Doc:      "require typed-key map indexes to use typed key expressions (no raw string literals or untyped string constants); require Block.Payload indexes to use const strings",
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
		keyNamed := asNamedType(m.Key())
		if keyNamed == nil {
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

		if isAllowedTypedKeyExpr(pass, idx.Index, wantPkgPath, wantName) {
			return
		}

		pass.Reportf(
			idx.Lbrack,
			`%s key must be of type %q (not a raw string literal or untyped string constant)`,
			sel.Sel.Name,
			keyTypeStr,
		)
	})

	return nil, nil
}

func isAllowedTypedKeyExpr(pass *analysis.Pass, e ast.Expr, wantPkgPath, wantName string) bool {
	e = unwrapParens(e)

	// Disallow raw string literals even if the type checker can implicitly convert them.
	if lit, ok := e.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		return false
	}

	// Disallow untyped string const identifiers/selectors (e.g. const k = "foo"; t.Data[k]).
	if isUntypedStringConstExpr(pass, e) {
		return false
	}

	tv, ok := pass.TypesInfo.Types[e]
	if !ok || tv.Type == nil {
		return false
	}

	named := asNamedType(tv.Type)
	if named == nil || named.Obj() == nil {
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

func asNamedType(t types.Type) *types.Named {
	switch tt := t.(type) {
	case *types.Named:
		return tt
	case *types.Alias:
		return asNamedType(tt.Rhs())
	default:
		return nil
	}
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

func isUntypedStringConstExpr(pass *analysis.Pass, e ast.Expr) bool {
	e = unwrapParens(e)

	switch t := e.(type) {
	case *ast.Ident:
		return objIsUntypedStringConst(pass.TypesInfo.ObjectOf(t))
	case *ast.SelectorExpr:
		return objIsUntypedStringConst(pass.TypesInfo.ObjectOf(t.Sel))
	default:
		return false
	}
}

func objIsUntypedStringConst(obj types.Object) bool {
	c, ok := obj.(*types.Const)
	if !ok {
		return false
	}

	b, ok := c.Type().Underlying().(*types.Basic)
	if !ok {
		return false
	}

	return b.Kind() == types.UntypedString
}
