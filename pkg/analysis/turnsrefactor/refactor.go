package turnsrefactor

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/imports"

	pkgerrors "github.com/pkg/errors"
)

// Config configures the refactoring run.
type Config struct {
	PackagePatterns []string
	Write           bool
	Verify          bool
	AllowNoop       bool

	Stdout io.Writer
	Stderr io.Writer
}

// StringSliceFlag is a repeatable flag for package patterns.
type StringSliceFlag struct {
	Values []string
}

func (s *StringSliceFlag) String() string {
	return fmt.Sprintf("%v", s.Values)
}

func (s *StringSliceFlag) Set(v string) error {
	s.Values = append(s.Values, v)
	return nil
}

type rewriteKind string

const (
	rkDataGet      rewriteKind = "DataGet"
	rkDataSet      rewriteKind = "DataSet"
	rkMetadataGet  rewriteKind = "MetadataGet"
	rkMetadataSet  rewriteKind = "MetadataSet"
	rkBlockMetaGet rewriteKind = "BlockMetadataGet"
	rkBlockMetaSet rewriteKind = "BlockMetadataSet"
	targetTurnsPkg             = "github.com/go-go-golems/geppetto/pkg/turns"
)

type stats struct {
	filesChanged int
	rewrites     map[rewriteKind]int
}

func Run(cfg Config) error {
	if cfg.Stdout == nil {
		cfg.Stdout = io.Discard
	}
	if cfg.Stderr == nil {
		cfg.Stderr = io.Discard
	}
	if len(cfg.PackagePatterns) == 0 {
		cfg.PackagePatterns = []string{"./..."}
	}

	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedModule,
		// Respect go.work / module context; Run from caller CWD.
	}, cfg.PackagePatterns...)
	if err != nil {
		return pkgerrors.Wrap(err, "load packages")
	}
	if packages.PrintErrors(pkgs) > 0 {
		return pkgerrors.Errorf("package load errors")
	}

	st := stats{rewrites: map[rewriteKind]int{}}
	changedFiles := map[string][]byte{}

	for _, pkg := range pkgs {
		for i, file := range pkg.Syntax {
			filename := pkg.CompiledGoFiles[i]

			changed, out, perFile, err := rewriteFile(pkg.Fset, pkg.TypesInfo, file, filename)
			if err != nil {
				return pkgerrors.Wrapf(err, "rewrite %s", filename)
			}
			if !changed {
				continue
			}

			st.filesChanged++
			for k, v := range perFile {
				st.rewrites[k] += v
			}
			changedFiles[filename] = out
		}
	}

	if len(changedFiles) == 0 && !cfg.AllowNoop {
		if !cfg.Verify {
			return pkgerrors.Errorf("no rewrites applied (use --allow-noop to ignore)")
		}
	}

	// Apply imports.Process + optionally write.
	for filename, src := range changedFiles {
		processed, err := imports.Process(filename, src, &imports.Options{
			Comments:  true,
			TabWidth:  8,
			TabIndent: true,
		})
		if err != nil {
			return pkgerrors.Wrapf(err, "imports.Process %s", filename)
		}

		changedFiles[filename] = processed

		if cfg.Write {
			if err := os.WriteFile(filename, processed, 0o644); err != nil {
				return pkgerrors.Wrapf(err, "write %s", filename)
			}
		}
	}

	fmt.Fprintf(cfg.Stdout, "turnsrefactor: files changed=%d\n", st.filesChanged)
	for _, k := range []rewriteKind{rkDataGet, rkDataSet, rkMetadataGet, rkMetadataSet, rkBlockMetaGet, rkBlockMetaSet} {
		if st.rewrites[k] > 0 {
			fmt.Fprintf(cfg.Stdout, "  %s: %d\n", k, st.rewrites[k])
		}
	}
	if !cfg.Write {
		fmt.Fprintln(cfg.Stdout, "  (dry-run) pass -w to write changes")
	}

	if cfg.Verify {
		// Verification: ensure no targeted calls remain anywhere in the scanned packages.
		// This is intentionally a cheap textual check so it still works in a post-migration
		// world where the legacy API may no longer exist.
		seen := map[string]struct{}{}
		for _, pkg := range pkgs {
			for i := range pkg.CompiledGoFiles {
				filename := pkg.CompiledGoFiles[i]
				if _, ok := seen[filename]; ok {
					continue
				}
				seen[filename] = struct{}{}

				if src, ok := changedFiles[filename]; ok {
					if err := verifyNoTargetCalls(filename, src); err != nil {
						return err
					}
					continue
				}

				src, err := os.ReadFile(filename)
				if err != nil {
					return pkgerrors.Wrapf(err, "verify read %s", filename)
				}
				if err := verifyNoTargetCalls(filename, src); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func rewriteFile(fset *token.FileSet, info *types.Info, f *ast.File, filename string) (bool, []byte, map[rewriteKind]int, error) {
	perFile := map[rewriteKind]int{}
	changed := false

	ast.Inspect(f, func(n ast.Node) bool {
		ce, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		rk, newCall, ok, err := rewriteCallExpr(info, ce)
		if err != nil {
			// hard fail: we don't want partial migrations with silent skips.
			panic(pkgerrors.Wrapf(err, "rewrite call in %s", filename))
		}
		if !ok {
			return true
		}

		// Mutate the node in place.
		*ce = *newCall
		changed = true
		perFile[rk]++
		return true
	})

	if !changed {
		return false, nil, perFile, nil
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, f); err != nil {
		return false, nil, perFile, pkgerrors.Wrap(err, "format file")
	}

	// Make sure we end with newline (imports.Process expects it).
	out := buf.Bytes()
	if len(out) == 0 || out[len(out)-1] != '\n' {
		out = append(out, '\n')
	}
	return true, out, perFile, nil
}

func rewriteCallExpr(info *types.Info, call *ast.CallExpr) (rewriteKind, *ast.CallExpr, bool, error) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", nil, false, nil
	}
	obj, ok := info.Uses[sel.Sel]
	if !ok {
		return "", nil, false, nil
	}
	fn, ok := obj.(*types.Func)
	if !ok {
		return "", nil, false, nil
	}
	pkg := fn.Pkg()
	if pkg == nil || pkg.Path() != targetTurnsPkg {
		return "", nil, false, nil
	}

	name := fn.Name()
	switch name {
	case string(rkDataGet):
		return rewriteGet(rkDataGet, "Get", call)
	case string(rkMetadataGet):
		return rewriteGet(rkMetadataGet, "Get", call)
	case string(rkBlockMetaGet):
		return rewriteGet(rkBlockMetaGet, "Get", call)
	case string(rkDataSet):
		return rewriteSet(rkDataSet, "Set", call)
	case string(rkMetadataSet):
		return rewriteSet(rkMetadataSet, "Set", call)
	case string(rkBlockMetaSet):
		return rewriteSet(rkBlockMetaSet, "Set", call)
	default:
		return "", nil, false, nil
	}
}

func rewriteGet(kind rewriteKind, method string, call *ast.CallExpr) (rewriteKind, *ast.CallExpr, bool, error) {
	// turns.DataGet(store, key) -> key.Get(store)
	if len(call.Args) != 2 {
		return "", nil, false, pkgerrors.Errorf("%s: expected 2 args, got %d", kind, len(call.Args))
	}
	store := call.Args[0]
	key := call.Args[1]

	newCall := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   key,
			Sel: ast.NewIdent(method),
		},
		Args: []ast.Expr{store},
	}
	return kind, newCall, true, nil
}

func rewriteSet(kind rewriteKind, method string, call *ast.CallExpr) (rewriteKind, *ast.CallExpr, bool, error) {
	// turns.DataSet(storePtr, key, value) -> key.Set(storePtr, value)
	if len(call.Args) != 3 {
		return "", nil, false, pkgerrors.Errorf("%s: expected 3 args, got %d", kind, len(call.Args))
	}
	storePtr := call.Args[0]
	key := call.Args[1]
	value := call.Args[2]

	newCall := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   key,
			Sel: ast.NewIdent(method),
		},
		Args: []ast.Expr{storePtr, value},
	}
	return kind, newCall, true, nil
}

func verifyNoTargetCalls(filename string, src []byte) error {
	// Cheap textual check is fine here because we're only verifying the tool output.
	// This catches stragglers due to missed rewrites or tool bugs.
	//
	// We only check for the explicit selector spellings; type-based verification can
	// be added later by loading packages again.
	needles := [][]byte{
		[]byte(".DataGet("),
		[]byte(".DataSet("),
		[]byte(".MetadataGet("),
		[]byte(".MetadataSet("),
		[]byte(".BlockMetadataGet("),
		[]byte(".BlockMetadataSet("),
	}
	for _, n := range needles {
		if bytes.Contains(src, n) {
			rel := filename
			if wd, err := os.Getwd(); err == nil {
				if r, err2 := filepath.Rel(wd, filename); err2 == nil {
					rel = r
				}
			}
			return pkgerrors.Errorf("verify failed: %s still contains %q", rel, string(n))
		}
	}
	return nil
}
