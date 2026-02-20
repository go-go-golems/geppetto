package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSchemaOK(t *testing.T) {
	s := &schema{
		Namespace: "geppetto",
		BlockKinds: []blockKindSchema{
			{Const: "BlockKindUser", Value: "user"},
			{Const: "BlockKindOther", Value: "other"},
		},
		Keys: []keySchema{
			{Scope: "data", ValueConst: "FooValueKey", Value: "foo", TypedKey: "KeyFoo", TypeExpr: "string"},
			{Scope: "turn_meta", ValueConst: "BarValueKey", Value: "bar", TypedKey: "KeyBar", TypeExpr: "any"},
			{Scope: "block_meta", ValueConst: "BazValueKey", Value: "baz"},
		},
	}
	if err := validateSchema(s); err != nil {
		t.Fatalf("validateSchema() unexpected error: %v", err)
	}
}

func TestValidateSchemaRejectsDuplicates(t *testing.T) {
	t.Run("duplicate block const", func(t *testing.T) {
		s := &schema{
			Namespace: "geppetto",
			BlockKinds: []blockKindSchema{
				{Const: "BlockKindUser", Value: "user"},
				{Const: "BlockKindUser", Value: "other"},
			},
		}
		if err := validateSchema(s); err == nil {
			t.Fatalf("expected duplicate block const error")
		}
	})

	t.Run("duplicate block value", func(t *testing.T) {
		s := &schema{
			Namespace: "geppetto",
			BlockKinds: []blockKindSchema{
				{Const: "BlockKindUser", Value: "user"},
				{Const: "BlockKindOther", Value: "user"},
			},
		}
		if err := validateSchema(s); err == nil {
			t.Fatalf("expected duplicate block value error")
		}
	})

	t.Run("duplicate key value const", func(t *testing.T) {
		s := &schema{
			Namespace: "geppetto",
			BlockKinds: []blockKindSchema{
				{Const: "BlockKindUser", Value: "user"},
			},
			Keys: []keySchema{
				{Scope: "data", ValueConst: "FooValueKey", Value: "foo"},
				{Scope: "turn_meta", ValueConst: "FooValueKey", Value: "bar"},
			},
		}
		if err := validateSchema(s); err == nil {
			t.Fatalf("expected duplicate key value const error")
		}
	})

	t.Run("duplicate typed key", func(t *testing.T) {
		s := &schema{
			Namespace: "geppetto",
			BlockKinds: []blockKindSchema{
				{Const: "BlockKindUser", Value: "user"},
			},
			Keys: []keySchema{
				{Scope: "data", ValueConst: "FooValueKey", Value: "foo", TypedKey: "KeyX", TypeExpr: "string"},
				{Scope: "turn_meta", ValueConst: "BarValueKey", Value: "bar", TypedKey: "KeyX", TypeExpr: "string"},
			},
		}
		if err := validateSchema(s); err == nil {
			t.Fatalf("expected duplicate typed key error")
		}
	})
}

func TestValidateSchemaTypedKeyRequirements(t *testing.T) {
	s1 := &schema{
		Namespace: "geppetto",
		BlockKinds: []blockKindSchema{
			{Const: "BlockKindUser", Value: "user"},
		},
		Keys: []keySchema{
			{Scope: "data", ValueConst: "FooValueKey", Value: "foo", TypeExpr: "string"},
		},
	}
	if err := validateSchema(s1); err == nil {
		t.Fatalf("expected error when type_expr set without typed_key")
	}

	s2 := &schema{
		Namespace: "geppetto",
		BlockKinds: []blockKindSchema{
			{Const: "BlockKindUser", Value: "user"},
		},
		Keys: []keySchema{
			{Scope: "data", ValueConst: "FooValueKey", Value: "foo", TypedKey: "KeyFoo"},
		},
	}
	if err := validateSchema(s2); err == nil {
		t.Fatalf("expected error when typed_key set without type_expr")
	}
}

func TestFallbackKindPrefersOther(t *testing.T) {
	kinds := []blockKindSchema{
		{Const: "BlockKindUser", Value: "user"},
		{Const: "BlockKindOther", Value: "other"},
	}
	c, v := fallbackKind(kinds)
	if c != "BlockKindOther" || v != "other" {
		t.Fatalf("expected other fallback, got %q %q", c, v)
	}
}

func TestFallbackKindFallsBackToLast(t *testing.T) {
	kinds := []blockKindSchema{
		{Const: "BlockKindUser", Value: "user"},
		{Const: "BlockKindLast", Value: "last"},
	}
	c, v := fallbackKind(kinds)
	if c != "BlockKindLast" || v != "last" {
		t.Fatalf("expected last fallback, got %q %q", c, v)
	}
}

func TestToKeysRenderDataBuilders(t *testing.T) {
	s := &schema{
		Namespace: "geppetto",
		BlockKinds: []blockKindSchema{
			{Const: "BlockKindUser", Value: "user"},
		},
		Keys: []keySchema{
			{Scope: "data", ValueConst: "A", Value: "a", TypedKey: "KeyA", TypeExpr: "string"},
			{Scope: "turn_meta", ValueConst: "B", Value: "b", TypedKey: "KeyB", TypeExpr: "any"},
			{Scope: "block_meta", ValueConst: "C", Value: "c", TypedKey: "KeyC", TypeExpr: "string"},
		},
	}
	r := toKeysRenderData(s)
	if len(r.Data) != 1 || r.Data[0].Builder != "DataK" {
		t.Fatalf("unexpected data builder mapping: %+v", r.Data)
	}
	if len(r.TurnMeta) != 1 || r.TurnMeta[0].Builder != "TurnMetaK" {
		t.Fatalf("unexpected turn_meta builder mapping: %+v", r.TurnMeta)
	}
	if len(r.BlockMeta) != 1 || r.BlockMeta[0].Builder != "BlockMetaK" {
		t.Fatalf("unexpected block_meta builder mapping: %+v", r.BlockMeta)
	}
}

func TestToDTSRenderDataIncludesKindsAndKeys(t *testing.T) {
	s := &schema{
		Namespace: "geppetto",
		BlockKinds: []blockKindSchema{
			{Const: "BlockKindUser", Value: "user"},
			{Const: "BlockKindOther", Value: "other"},
		},
		Keys: []keySchema{
			{Scope: "data", ValueConst: "A", Value: "a"},
			{Scope: "turn_meta", ValueConst: "B", Value: "b"},
			{Scope: "block_meta", ValueConst: "C", Value: "c"},
		},
	}

	r := toDTSRenderData(s)
	if r.Namespace != "geppetto" {
		t.Fatalf("unexpected namespace %q", r.Namespace)
	}
	if len(r.BlockKinds) != 2 {
		t.Fatalf("expected 2 block kinds, got %d", len(r.BlockKinds))
	}
	if len(r.Data) != 1 || len(r.TurnMeta) != 1 || len(r.BlockMeta) != 1 {
		t.Fatalf("unexpected key grouping: data=%d turn_meta=%d block_meta=%d", len(r.Data), len(r.TurnMeta), len(r.BlockMeta))
	}
}

func TestRenderTemplateFile(t *testing.T) {
	dir := t.TempDir()
	tplPath := filepath.Join(dir, "x.tmpl")
	if err := os.WriteFile(tplPath, []byte(`{{ .Namespace }}-{{ len .BlockKinds }}`), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	out, err := renderTemplateFile(tplPath, dtsRenderData{
		Namespace:  "geppetto",
		BlockKinds: []blockKindSchema{{Const: "A", Value: "a"}},
	})
	if err != nil {
		t.Fatalf("renderTemplateFile() error: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "geppetto-1" {
		t.Fatalf("unexpected template output %q", got)
	}
}
