package main

import (
	"strings"
	"testing"
)

func TestValidateSchemaOK(t *testing.T) {
	s := &schema{
		Namespace: "geppetto",
		Outputs: outputsSchema{
			TurnsBlockKindGo: "a", TurnsKeysGo: "b", EngineTurnkeysGo: "c", TurnsDTS: "d", GeppettoConstsGo: "e", GeppettoDTS: "f",
		},
		Templates:  templatesSchema{TurnsDTS: "x", GeppettoDTS: "y"},
		BlockKinds: []blockKindSchema{{Const: "BlockKindUser", Value: "user"}},
		KeyFamilies: keyFamiliesSchema{
			Data:      []keySchema{{ValueConst: "ToolConfigValueKey", Value: "tool_config"}},
			TurnMeta:  []keySchema{{ValueConst: "TurnMetaProviderValueKey", Value: "provider"}},
			BlockMeta: []keySchema{{ValueConst: "BlockMetaMiddlewareValueKey", Value: "middleware"}},
			RunMeta:   []keySchema{{ValueConst: "RunMetaKeyTraceID", Value: "trace_id", ConstType: "RunMetadataKey"}},
			Payload:   []keySchema{{ValueConst: "PayloadKeyText", Value: "text"}},
		},
		JSEnums: []jsEnumSchema{{
			Name:   "ToolChoice",
			Values: []jsEnumValueSchema{{JSKey: "AUTO", Value: "auto"}},
		}},
		JSExports: jsExportsSchema{
			ConstGroups: []jsConstGroupSchema{
				{Name: "BlockKind", Source: "block_kinds"},
				{Name: "ToolChoice", Source: "js_enum:ToolChoice"},
				{Name: "TurnDataKeys", Source: "key_family:data"},
			},
		},
	}
	if err := validateSchema(s); err != nil {
		t.Fatalf("validateSchema() unexpected error: %v", err)
	}
}

func TestValidateSchemaRejectsBadTypedOwner(t *testing.T) {
	s := &schema{
		Namespace: "geppetto",
		Outputs: outputsSchema{
			TurnsBlockKindGo: "a", TurnsKeysGo: "b", EngineTurnkeysGo: "c", TurnsDTS: "d", GeppettoConstsGo: "e", GeppettoDTS: "f",
		},
		Templates:  templatesSchema{TurnsDTS: "x", GeppettoDTS: "y"},
		BlockKinds: []blockKindSchema{{Const: "BlockKindUser", Value: "user"}},
		KeyFamilies: keyFamiliesSchema{
			Data: []keySchema{{
				ValueConst: "ToolConfigValueKey",
				Value:      "tool_config",
				TypedKey:   "KeyToolConfig",
				TypeExpr:   "ToolConfig",
				TypedOwner: "invalid",
			}},
		},
		JSEnums:   []jsEnumSchema{{Name: "ToolChoice", Values: []jsEnumValueSchema{{JSKey: "AUTO", Value: "auto"}}}},
		JSExports: jsExportsSchema{ConstGroups: []jsConstGroupSchema{{Name: "ToolChoice", Source: "js_enum:ToolChoice"}}},
	}
	if err := validateSchema(s); err == nil {
		t.Fatalf("expected typed_owner validation error")
	}
}

func TestBuildJSExportEnums(t *testing.T) {
	s := &schema{
		Namespace:  "geppetto",
		BlockKinds: []blockKindSchema{{Const: "BlockKindUser", Value: "user"}},
		KeyFamilies: keyFamiliesSchema{
			Data:    []keySchema{{ValueConst: "ToolConfigValueKey", Value: "tool_config"}},
			Payload: []keySchema{{ValueConst: "PayloadKeyText", Value: "text"}},
		},
		JSEnums: []jsEnumSchema{{
			Name: "ToolChoice",
			Doc:  "how",
			Values: []jsEnumValueSchema{{
				JSKey: "AUTO",
				Value: "auto",
			}},
		}},
		JSExports: jsExportsSchema{ConstGroups: []jsConstGroupSchema{
			{Name: "BlockKind", Source: "block_kinds"},
			{Name: "TurnDataKeys", Source: "key_family:data"},
			{Name: "PayloadKeys", Source: "key_family:payload"},
			{Name: "ToolChoice", Source: "js_enum:ToolChoice"},
		}},
	}
	enums, err := buildJSExportEnums(s)
	if err != nil {
		t.Fatalf("buildJSExportEnums() error: %v", err)
	}
	if len(enums) != 4 {
		t.Fatalf("expected 4 export groups, got %d", len(enums))
	}

	src, err := renderTemplate(jsConstsTemplate, jsConstsRenderData{Enums: enums})
	if err != nil {
		t.Fatalf("renderTemplate() error: %v", err)
	}
	out := string(src)
	if !strings.Contains(out, `m.mustSet(constsObj, "PayloadKeys", o)`) {
		t.Fatalf("expected payload keys export group")
	}
	if !strings.Contains(out, `m.mustSet(o, "TOOL_CONFIG", "tool_config")`) {
		t.Fatalf("expected key-family js key conversion")
	}
}

func TestValueToJSKey(t *testing.T) {
	cases := map[string]string{
		"session_id":         "SESSION_ID",
		"tool-call":          "TOOL_CALL",
		"  claude content  ": "CLAUDE_CONTENT",
		"9lives":             "K_9LIVES",
	}
	for in, want := range cases {
		if got := valueToJSKey(in); got != want {
			t.Fatalf("valueToJSKey(%q) = %q, want %q", in, got, want)
		}
	}
}
