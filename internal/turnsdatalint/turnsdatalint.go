package turnsdatalint

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const defaultKeyType = "github.com/go-go-golems/geppetto/pkg/turns.TurnDataKey"

var keyTypeFlag string

// Analyzer enforces that Turn.Data[...] map access uses a const of the configured key type.
//
// This prevents ad-hoc conversions like turns.TurnDataKey("raw") or variables, encouraging use
// of canonical, typed constants (e.g., turns.DataKeyToolRegistry).
var Analyzer = &analysis.Analyzer{
	Name:     "turnsdatalint",
	Doc:      "require Turn.Data[...] indexes to use a const of the configured key type (not a conversion or variable)",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func init() {
	Analyzer.Flags.StringVar(
		&keyTypeFlag,
		"keytype",
		defaultKeyType,
		`fully-qualified named type like "github.com/go-go-golems/geppetto/pkg/turns.TurnDataKey"`,
	)
}

func run(pass *analysis.Pass) (any, error) {
	wantPkgPath, wantName, ok := splitQualifiedType(keyTypeFlag)
	if !ok {
		return nil, nil
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{(*ast.IndexExpr)(nil)}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		idx := n.(*ast.IndexExpr)

		// Match <something>.Data[...]
		sel, ok := idx.X.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != "Data" {
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
		if !namedTypeMatches(m.Key(), wantPkgPath, wantName) {
			return
		}

		// Key must be a const of the desired named type.
		if isAllowedConstKey(pass, idx.Index, wantPkgPath, wantName) {
			return
		}

		pass.Reportf(
			idx.Lbrack,
			`Turn.Data key must be a const of type %q (not a conversion or variable)`,
			keyTypeFlag,
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
