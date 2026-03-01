package geppetto

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"testing"
)

type dtsExportSurface struct {
	TopLevel []string
	Grouped  map[string][]string
}

var (
	reExportConst         = regexp.MustCompile(`(?m)^\s*export const ([A-Za-z_][A-Za-z0-9_]*)\s*:\s*`)
	reExportFunction      = regexp.MustCompile(`(?m)^\s*export function ([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	reObjectLevelProperty = regexp.MustCompile(`(?m)^\s{8}([A-Za-z_][A-Za-z0-9_]*)\s*(?:\(|:)`)
)

func TestGeneratedDTSMatchesRuntimeExportSurface(t *testing.T) {
	dtsPath := geppettoDTSPath(t)
	expected, err := parseDTSSurfaceFile(dtsPath)
	if err != nil {
		t.Fatalf("failed parsing generated d.ts (%s): %v", dtsPath, err)
	}

	rt := newJSRuntime(t, Options{})
	assertSameSet(
		t,
		"geppetto top-level exports",
		expected.TopLevel,
		runtimeObjectKeys(t, rt, `require("geppetto")`),
	)

	for _, namespace := range []string{"consts", "turns", "engines", "profiles", "schemas", "middlewares", "tools"} {
		want, ok := expected.Grouped[namespace]
		if !ok {
			t.Fatalf("generated d.ts does not contain export object for %q", namespace)
		}
		assertSameSet(
			t,
			fmt.Sprintf("geppetto.%s exports", namespace),
			want,
			runtimeObjectKeys(t, rt, fmt.Sprintf(`require("geppetto").%s`, namespace)),
		)
	}
}

func geppettoDTSPath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to determine current test file path")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "doc", "types", "geppetto.d.ts")
}

func runtimeObjectKeys(t *testing.T, rt *jsRuntime, expr string) []string {
	t.Helper()
	exported := mustEvalExprExport(t, rt, fmt.Sprintf(`Object.keys(%s).sort()`, expr))
	return toStringSlice(t, exported)
}

func toStringSlice(t *testing.T, v any) []string {
	t.Helper()
	switch vv := v.(type) {
	case []string:
		return append([]string(nil), vv...)
	case []any:
		out := make([]string, 0, len(vv))
		for _, item := range vv {
			s, ok := item.(string)
			if !ok {
				t.Fatalf("expected []string-compatible value, got element %T in %T", item, v)
			}
			out = append(out, s)
		}
		return out
	default:
		t.Fatalf("expected []string-compatible value, got %T", v)
		return nil
	}
}

func parseDTSSurfaceFile(path string) (dtsExportSurface, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return dtsExportSurface{}, err
	}
	return parseDTSSurface(string(b))
}

func parseDTSSurface(src string) (dtsExportSurface, error) {
	topSet := map[string]struct{}{}
	grouped := map[string][]string{}

	for _, match := range reExportConst.FindAllStringSubmatch(src, -1) {
		topSet[match[1]] = struct{}{}
	}
	for _, match := range reExportFunction.FindAllStringSubmatch(src, -1) {
		topSet[match[1]] = struct{}{}
	}

	for _, loc := range reExportConst.FindAllStringSubmatchIndex(src, -1) {
		name := src[loc[2]:loc[3]]
		valueStart := loc[1]
		for valueStart < len(src) {
			c := src[valueStart]
			if c == ' ' || c == '\t' {
				valueStart++
				continue
			}
			break
		}
		if valueStart >= len(src) || src[valueStart] != '{' {
			continue
		}
		valueEnd, err := findMatchingBrace(src, valueStart)
		if err != nil {
			return dtsExportSurface{}, fmt.Errorf("parse object export %q: %w", name, err)
		}
		grouped[name] = objectProperties(src[valueStart+1 : valueEnd])
	}

	return dtsExportSurface{
		TopLevel: sortedSet(topSet),
		Grouped:  grouped,
	}, nil
}

func findMatchingBrace(src string, open int) (int, error) {
	depth := 0
	for i := open; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return -1, fmt.Errorf("unclosed object literal")
}

func objectProperties(body string) []string {
	set := map[string]struct{}{}
	for _, match := range reObjectLevelProperty.FindAllStringSubmatch(body, -1) {
		set[match[1]] = struct{}{}
	}
	return sortedSet(set)
}

func sortedSet(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	slices.Sort(out)
	return out
}

func assertSameSet(t *testing.T, label string, expected, actual []string) {
	t.Helper()
	slices.Sort(expected)
	slices.Sort(actual)

	missing := make([]string, 0)
	extra := make([]string, 0)
	expectedSet := map[string]struct{}{}
	actualSet := map[string]struct{}{}

	for _, item := range expected {
		expectedSet[item] = struct{}{}
	}
	for _, item := range actual {
		actualSet[item] = struct{}{}
	}
	for _, item := range expected {
		if _, ok := actualSet[item]; !ok {
			missing = append(missing, item)
		}
	}
	for _, item := range actual {
		if _, ok := expectedSet[item]; !ok {
			extra = append(extra, item)
		}
	}

	if len(missing) == 0 && len(extra) == 0 {
		return
	}

	t.Fatalf(
		"%s mismatch\nmissing: %v\nextra: %v\nexpected: %v\nactual: %v",
		label,
		missing,
		extra,
		expected,
		actual,
	)
}
