package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

var legacyImports = map[string]struct{}{
	"github.com/go-go-golems/glazed/pkg/cmds/layers":       {},
	"github.com/go-go-golems/glazed/pkg/cmds/parameters":   {},
	"github.com/go-go-golems/glazed/pkg/cmds/middlewares":  {},
	"github.com/go-go-golems/glazed/pkg/cmds/parsedlayers": {},
}

type ImportHit struct {
	Module     string `json:"module"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	Column     int    `json:"column"`
	Alias      string `json:"alias"`
	ImportPath string `json:"importPath"`
}

type SelectorHit struct {
	Module       string `json:"module"`
	File         string `json:"file"`
	Line         int    `json:"line"`
	Column       int    `json:"column"`
	Alias        string `json:"alias"`
	Symbol       string `json:"symbol"`
	ResolvedFrom string `json:"resolvedFrom"`
}

type TagHit struct {
	Module   string `json:"module"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	TagKey   string `json:"tagKey"`
	TagValue string `json:"tagValue"`
	Field    string `json:"field"`
}

type SignatureHotspot struct {
	Module         string   `json:"module"`
	File           string   `json:"file"`
	Line           int      `json:"line"`
	Column         int      `json:"column"`
	Function       string   `json:"function"`
	Signature      string   `json:"signature"`
	GoplsRefCount  int      `json:"goplsRefCount"`
	GoplsRefSample []string `json:"goplsRefSample,omitempty"`
	GoplsError     string   `json:"goplsError,omitempty"`
}

type Summary struct {
	ModulesScanned    []string `json:"modulesScanned"`
	GoFilesScanned    int      `json:"goFilesScanned"`
	ImportHits        int      `json:"importHits"`
	SelectorHits      int      `json:"selectorHits"`
	TagHits           int      `json:"tagHits"`
	SignatureHotspots int      `json:"signatureHotspots"`
	GoplsEnriched     int      `json:"goplsEnriched"`
}

type Report struct {
	GeneratedAt       string             `json:"generatedAt"`
	RepoRoot          string             `json:"repoRoot"`
	LegacyImports     []string           `json:"legacyImports"`
	Summary           Summary            `json:"summary"`
	ImportHits        []ImportHit        `json:"importHits"`
	SelectorHits      []SelectorHit      `json:"selectorHits"`
	TagHits           []TagHit           `json:"tagHits"`
	SignatureHotspots []SignatureHotspot `json:"signatureHotspots"`
}

type analyzerConfig struct {
	repoRoot      string
	modules       []string
	outJSON       string
	outMarkdown   string
	includeGopls  bool
	maxGoplsCalls int
	goplsTimeout  time.Duration
}

func main() {
	cfg := parseFlags()

	report, err := analyze(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "analyze failed: %v\n", err)
		os.Exit(1)
	}

	if err := writeJSON(cfg.outJSON, report); err != nil {
		fmt.Fprintf(os.Stderr, "write json failed: %v\n", err)
		os.Exit(1)
	}
	if err := writeMarkdown(cfg.outMarkdown, report, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "write markdown failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("ok: scanned modules=%v go_files=%d import_hits=%d selector_hits=%d tag_hits=%d signature_hotspots=%d gopls_enriched=%d\n",
		report.Summary.ModulesScanned,
		report.Summary.GoFilesScanned,
		report.Summary.ImportHits,
		report.Summary.SelectorHits,
		report.Summary.TagHits,
		report.Summary.SignatureHotspots,
		report.Summary.GoplsEnriched,
	)
	fmt.Printf("ok: wrote %s\n", cfg.outJSON)
	fmt.Printf("ok: wrote %s\n", cfg.outMarkdown)
}

func parseFlags() analyzerConfig {
	var (
		modulesCSV   string
		repoRoot     string
		outJSON      string
		outMarkdown  string
		includeGopls bool
		maxGopls     int
		goplsTimeout string
	)

	flag.StringVar(&repoRoot, "repo-root", "", "absolute path to workspace root (contains geppetto/, pinocchio/, glazed/)")
	flag.StringVar(&modulesCSV, "modules", "geppetto,pinocchio", "comma-separated module directories under repo-root")
	flag.StringVar(&outJSON, "out-json", "", "output JSON report path")
	flag.StringVar(&outMarkdown, "out-md", "", "output Markdown report path")
	flag.BoolVar(&includeGopls, "include-gopls", true, "enrich signature hotspots with `gopls references` counts")
	flag.IntVar(&maxGopls, "max-gopls-calls", 60, "maximum number of hotspots to query with gopls")
	flag.StringVar(&goplsTimeout, "gopls-timeout", "12s", "timeout per gopls call")
	flag.Parse()

	if repoRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to get cwd: %v\n", err)
			os.Exit(1)
		}
		repoRoot = cwd
	}
	repoRoot, _ = filepath.Abs(repoRoot)

	modules := []string{}
	for _, m := range strings.Split(modulesCSV, ",") {
		m = strings.TrimSpace(m)
		if m != "" {
			modules = append(modules, m)
		}
	}
	if len(modules) == 0 {
		fmt.Fprintln(os.Stderr, "no modules provided")
		os.Exit(1)
	}

	timeoutDur, err := time.ParseDuration(goplsTimeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid gopls-timeout: %v\n", err)
		os.Exit(1)
	}

	if outJSON == "" {
		outJSON = filepath.Join(repoRoot, "migration-report.json")
	}
	if outMarkdown == "" {
		outMarkdown = filepath.Join(repoRoot, "migration-report.md")
	}

	return analyzerConfig{
		repoRoot:      repoRoot,
		modules:       modules,
		outJSON:       outJSON,
		outMarkdown:   outMarkdown,
		includeGopls:  includeGopls,
		maxGoplsCalls: maxGopls,
		goplsTimeout:  timeoutDur,
	}
}

func analyze(cfg analyzerConfig) (*Report, error) {
	report := &Report{
		GeneratedAt: time.Now().Format(time.RFC3339),
		RepoRoot:    cfg.repoRoot,
		Summary: Summary{
			ModulesScanned: append([]string{}, cfg.modules...),
		},
	}
	for k := range legacyImports {
		report.LegacyImports = append(report.LegacyImports, k)
	}
	sort.Strings(report.LegacyImports)

	for _, module := range cfg.modules {
		moduleRoot := filepath.Join(cfg.repoRoot, module)
		if _, err := os.Stat(moduleRoot); err != nil {
			return nil, fmt.Errorf("module root missing %s: %w", moduleRoot, err)
		}
		if err := scanModule(module, moduleRoot, report); err != nil {
			return nil, err
		}
	}

	sort.Slice(report.ImportHits, func(i, j int) bool {
		return lessPos(report.ImportHits[i].File, report.ImportHits[i].Line, report.ImportHits[i].Column, report.ImportHits[j].File, report.ImportHits[j].Line, report.ImportHits[j].Column)
	})
	sort.Slice(report.SelectorHits, func(i, j int) bool {
		return lessPos(report.SelectorHits[i].File, report.SelectorHits[i].Line, report.SelectorHits[i].Column, report.SelectorHits[j].File, report.SelectorHits[j].Line, report.SelectorHits[j].Column)
	})
	sort.Slice(report.TagHits, func(i, j int) bool {
		return lessPos(report.TagHits[i].File, report.TagHits[i].Line, report.TagHits[i].Column, report.TagHits[j].File, report.TagHits[j].Line, report.TagHits[j].Column)
	})
	sort.Slice(report.SignatureHotspots, func(i, j int) bool {
		return lessPos(report.SignatureHotspots[i].File, report.SignatureHotspots[i].Line, report.SignatureHotspots[i].Column, report.SignatureHotspots[j].File, report.SignatureHotspots[j].Line, report.SignatureHotspots[j].Column)
	})

	report.Summary.ImportHits = len(report.ImportHits)
	report.Summary.SelectorHits = len(report.SelectorHits)
	report.Summary.TagHits = len(report.TagHits)
	report.Summary.SignatureHotspots = len(report.SignatureHotspots)

	if cfg.includeGopls {
		enriched := enrichWithGopls(cfg, report.SignatureHotspots)
		report.Summary.GoplsEnriched = enriched
	}

	return report, nil
}

func scanModule(module, moduleRoot string, report *Report) error {
	return filepath.WalkDir(moduleRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		report.Summary.GoFilesScanned++
		return scanGoFile(module, moduleRoot, path, report)
	})
}

func skipDir(name string) bool {
	switch name {
	case ".git", "vendor", "node_modules", "dist", "build", "tmp", "ttmp":
		return true
	default:
		return false
	}
}

func scanGoFile(module, moduleRoot, filePath string, report *Report) error {
	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse %s: %w", filePath, err)
	}

	rel, _ := filepath.Rel(moduleRoot, filePath)
	rel = filepath.ToSlash(rel)

	aliasToImport := map[string]string{}
	for _, imp := range fileNode.Imports {
		importPath, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		if _, ok := legacyImports[importPath]; !ok {
			continue
		}

		alias := importAlias(imp, importPath)
		aliasToImport[alias] = importPath

		p := fset.Position(imp.Pos())
		report.ImportHits = append(report.ImportHits, ImportHit{
			Module:     module,
			File:       rel,
			Line:       p.Line,
			Column:     p.Column,
			Alias:      alias,
			ImportPath: importPath,
		})
	}

	ast.Inspect(fileNode, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			ident, ok := x.X.(*ast.Ident)
			if !ok {
				return true
			}
			importPath, ok := aliasToImport[ident.Name]
			if !ok {
				return true
			}
			p := fset.Position(x.Sel.Pos())
			report.SelectorHits = append(report.SelectorHits, SelectorHit{
				Module:       module,
				File:         rel,
				Line:         p.Line,
				Column:       p.Column,
				Alias:        ident.Name,
				Symbol:       x.Sel.Name,
				ResolvedFrom: importPath,
			})
		case *ast.Field:
			if x.Tag == nil {
				return true
			}
			rawTag := strings.Trim(x.Tag.Value, "`")
			tag := reflect.StructTag(rawTag)
			keys := []string{"glazed.parameter", "glazed.layer", "glazed.default", "glazed.help"}
			for _, k := range keys {
				v := tag.Get(k)
				if v == "" {
					continue
				}
				p := fset.Position(x.Tag.Pos())
				fieldName := ""
				if len(x.Names) > 0 {
					fieldName = x.Names[0].Name
				}
				report.TagHits = append(report.TagHits, TagHit{
					Module:   module,
					File:     rel,
					Line:     p.Line,
					Column:   p.Column,
					TagKey:   k,
					TagValue: v,
					Field:    fieldName,
				})
			}
		case *ast.FuncDecl:
			signature := signatureString(fset, x)
			if !strings.Contains(signature, "layers.") &&
				!strings.Contains(signature, "parameters.") &&
				!strings.Contains(signature, "middlewares.") &&
				!strings.Contains(signature, "parsedlayers.") {
				return true
			}
			p := fset.Position(x.Name.Pos())
			fnName := x.Name.Name
			if x.Recv != nil && len(x.Recv.List) > 0 {
				fnName = receiverName(fset, x.Recv.List[0].Type) + "." + fnName
			}
			report.SignatureHotspots = append(report.SignatureHotspots, SignatureHotspot{
				Module:    module,
				File:      rel,
				Line:      p.Line,
				Column:    p.Column,
				Function:  fnName,
				Signature: signature,
			})
		}
		return true
	})

	return nil
}

func importAlias(imp *ast.ImportSpec, importPath string) string {
	if imp.Name != nil && imp.Name.Name != "" {
		return imp.Name.Name
	}
	parts := strings.Split(importPath, "/")
	return parts[len(parts)-1]
}

func signatureString(fset *token.FileSet, fn *ast.FuncDecl) string {
	buf := &bytes.Buffer{}
	if err := printer.Fprint(buf, fset, fn.Type); err != nil {
		return ""
	}
	return buf.String()
}

func receiverName(fset *token.FileSet, expr ast.Expr) string {
	buf := &bytes.Buffer{}
	if err := printer.Fprint(buf, fset, expr); err != nil {
		return "recv"
	}
	s := buf.String()
	s = strings.TrimPrefix(s, "*")
	return s
}

func enrichWithGopls(cfg analyzerConfig, hotspots []SignatureHotspot) int {
	if len(hotspots) == 0 {
		return 0
	}

	moduleRoots := map[string]string{}
	for _, module := range cfg.modules {
		moduleRoots[module] = filepath.Join(cfg.repoRoot, module)
	}

	limit := cfg.maxGoplsCalls
	if limit <= 0 || limit > len(hotspots) {
		limit = len(hotspots)
	}

	enriched := 0
	for i := 0; i < limit; i++ {
		h := &hotspots[i]
		moduleRoot, ok := moduleRoots[h.Module]
		if !ok {
			h.GoplsError = "module root not found"
			continue
		}
		absFile := filepath.Join(moduleRoot, filepath.FromSlash(h.File))
		position := fmt.Sprintf("%s:%d:%d", absFile, h.Line, h.Column)
		refs, err := runGoplsReferences(moduleRoot, position, cfg.goplsTimeout)
		if err != nil {
			h.GoplsError = err.Error()
			continue
		}
		h.GoplsRefCount = len(refs)
		if len(refs) > 10 {
			h.GoplsRefSample = append([]string{}, refs[:10]...)
		} else {
			h.GoplsRefSample = refs
		}
		enriched++
	}
	return enriched
}

func runGoplsReferences(moduleRoot, position string, timeout time.Duration) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gopls", "references", position)
	cmd.Dir = moduleRoot
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gopls references failed (%s): %s", position, strings.TrimSpace(string(out)))
	}

	lines := []string{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip log lines, keep location lines.
		if strings.HasPrefix(line, "20") && strings.Contains(line, "Error:") {
			continue
		}
		lines = append(lines, line)
	}
	return lines, nil
}

func writeJSON(path string, report *Report) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func writeMarkdown(path string, report *Report, cfg analyzerConfig) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString("# Glazed Migration Analyzer Report\n\n")
	b.WriteString(fmt.Sprintf("- Generated: `%s`\n", report.GeneratedAt))
	b.WriteString(fmt.Sprintf("- Repo root: `%s`\n", report.RepoRoot))
	b.WriteString(fmt.Sprintf("- Modules: `%s`\n", strings.Join(report.Summary.ModulesScanned, ", ")))
	b.WriteString(fmt.Sprintf("- gopls enrichment: `%v` (max calls: `%d`, timeout: `%s`)\n\n", cfg.includeGopls, cfg.maxGoplsCalls, cfg.goplsTimeout))

	b.WriteString("## Summary\n\n")
	b.WriteString(fmt.Sprintf("- Go files scanned: `%d`\n", report.Summary.GoFilesScanned))
	b.WriteString(fmt.Sprintf("- Legacy import hits: `%d`\n", report.Summary.ImportHits))
	b.WriteString(fmt.Sprintf("- Legacy selector hits: `%d`\n", report.Summary.SelectorHits))
	b.WriteString(fmt.Sprintf("- Legacy tag hits: `%d`\n", report.Summary.TagHits))
	b.WriteString(fmt.Sprintf("- Signature hotspots: `%d`\n", report.Summary.SignatureHotspots))
	b.WriteString(fmt.Sprintf("- Hotspots enriched with gopls: `%d`\n\n", report.Summary.GoplsEnriched))

	b.WriteString("## Legacy Imports\n\n")
	for _, imp := range report.LegacyImports {
		b.WriteString(fmt.Sprintf("- `%s`\n", imp))
	}
	b.WriteString("\n")

	writeTopFilesSection(&b, "Top files by legacy selector usage", topSelectorFiles(report.SelectorHits))
	writeTopTagKeysSection(&b, report.TagHits)
	writeHotspotSection(&b, report.SignatureHotspots)

	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func topSelectorFiles(hits []SelectorHit) [][2]string {
	counts := map[string]int{}
	for _, h := range hits {
		k := h.Module + "/" + h.File
		counts[k]++
	}
	type kv struct {
		k string
		v int
	}
	items := []kv{}
	for k, v := range counts {
		items = append(items, kv{k: k, v: v})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].v == items[j].v {
			return items[i].k < items[j].k
		}
		return items[i].v > items[j].v
	})
	out := [][2]string{}
	for _, it := range items {
		out = append(out, [2]string{it.k, strconv.Itoa(it.v)})
	}
	return out
}

func writeTopFilesSection(b *strings.Builder, title string, rows [][2]string) {
	b.WriteString("## " + title + "\n\n")
	if len(rows) == 0 {
		b.WriteString("_none_\n\n")
		return
	}
	b.WriteString("| File | Count |\n")
	b.WriteString("| --- | ---: |\n")
	limit := len(rows)
	if limit > 20 {
		limit = 20
	}
	for i := 0; i < limit; i++ {
		b.WriteString(fmt.Sprintf("| `%s` | %s |\n", rows[i][0], rows[i][1]))
	}
	b.WriteString("\n")
}

func writeTopTagKeysSection(b *strings.Builder, tags []TagHit) {
	counts := map[string]int{}
	for _, t := range tags {
		counts[t.TagKey]++
	}
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	b.WriteString("## Legacy Tag Keys\n\n")
	if len(keys) == 0 {
		b.WriteString("_none_\n\n")
		return
	}
	for _, k := range keys {
		b.WriteString(fmt.Sprintf("- `%s`: `%d`\n", k, counts[k]))
	}
	b.WriteString("\n")
}

func writeHotspotSection(b *strings.Builder, hotspots []SignatureHotspot) {
	b.WriteString("## Signature Hotspots\n\n")
	if len(hotspots) == 0 {
		b.WriteString("_none_\n")
		return
	}
	b.WriteString("| Function | Location | gopls refs | Notes |\n")
	b.WriteString("| --- | --- | ---: | --- |\n")
	limit := len(hotspots)
	if limit > 40 {
		limit = 40
	}
	for i := 0; i < limit; i++ {
		h := hotspots[i]
		loc := fmt.Sprintf("%s/%s:%d:%d", h.Module, h.File, h.Line, h.Column)
		notes := ""
		if h.GoplsError != "" {
			notes = "gopls error"
		}
		b.WriteString(fmt.Sprintf("| `%s` | `%s` | %d | %s |\n", h.Function, loc, h.GoplsRefCount, notes))
	}
	b.WriteString("\n")
}

func lessPos(fileA string, lineA int, colA int, fileB string, lineB int, colB int) bool {
	if fileA != fileB {
		return fileA < fileB
	}
	if lineA != lineB {
		return lineA < lineB
	}
	return colA < colB
}
