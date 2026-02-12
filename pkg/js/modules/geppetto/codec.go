package geppetto

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

var (
	knownKinds = []turns.BlockKind{
		turns.BlockKindUser,
		turns.BlockKindLLMText,
		turns.BlockKindToolCall,
		turns.BlockKindToolUse,
		turns.BlockKindSystem,
		turns.BlockKindReasoning,
		turns.BlockKindOther,
	}

	turnDataShortToID = map[string]turns.TurnDataKey{
		turns.ToolConfigValueKey:            turns.TurnDataKey(engine.KeyToolConfig.String()),
		turns.AgentModeAllowedToolsValueKey: turns.TurnDataKey(turns.KeyAgentModeAllowedTools.String()),
		turns.AgentModeValueKey:             turns.TurnDataKey(turns.KeyAgentMode.String()),
		turns.ResponsesServerToolsValueKey:  turns.TurnDataKey(turns.KeyResponsesServerTools.String()),
	}
	turnMetaShortToID = map[string]turns.TurnMetadataKey{
		turns.TurnMetaProviderValueKey:    turns.TurnMetadataKey(turns.KeyTurnMetaProvider.String()),
		turns.TurnMetaRuntimeValueKey:     turns.TurnMetadataKey(turns.KeyTurnMetaRuntime.String()),
		turns.TurnMetaSessionIDValueKey:   turns.TurnMetadataKey(turns.KeyTurnMetaSessionID.String()),
		turns.TurnMetaInferenceIDValueKey: turns.TurnMetadataKey(turns.KeyTurnMetaInferenceID.String()),
		turns.TurnMetaTraceIDValueKey:     turns.TurnMetadataKey(turns.KeyTurnMetaTraceID.String()),
		turns.TurnMetaUsageValueKey:       turns.TurnMetadataKey(turns.KeyTurnMetaUsage.String()),
		turns.TurnMetaStopReasonValueKey:  turns.TurnMetadataKey(turns.KeyTurnMetaStopReason.String()),
		turns.TurnMetaModelValueKey:       turns.TurnMetadataKey(turns.KeyTurnMetaModel.String()),
	}
	blockMetaShortToID = map[string]turns.BlockMetadataKey{
		turns.BlockMetaClaudeOriginalContentValueKey: turns.BlockMetadataKey(turns.KeyBlockMetaClaudeOriginalContent.String()),
		turns.BlockMetaToolCallsValueKey:             turns.BlockMetadataKey(turns.KeyBlockMetaToolCalls.String()),
		turns.BlockMetaMiddlewareValueKey:            turns.BlockMetadataKey(turns.KeyBlockMetaMiddleware.String()),
		turns.BlockMetaAgentModeTagValueKey:          turns.BlockMetadataKey(turns.KeyBlockMetaAgentModeTag.String()),
		turns.BlockMetaAgentModeValueKey:             turns.BlockMetadataKey(turns.KeyBlockMetaAgentMode.String()),
	}

	turnDataIDToShort  = reverseTurnDataMap(turnDataShortToID)
	turnMetaIDToShort  = reverseTurnMetaMap(turnMetaShortToID)
	blockMetaIDToShort = reverseBlockMetaMap(blockMetaShortToID)
)

func reverseTurnDataMap(in map[string]turns.TurnDataKey) map[turns.TurnDataKey]string {
	out := make(map[turns.TurnDataKey]string, len(in))
	for k, v := range in {
		out[v] = k
	}
	return out
}

func reverseTurnMetaMap(in map[string]turns.TurnMetadataKey) map[turns.TurnMetadataKey]string {
	out := make(map[turns.TurnMetadataKey]string, len(in))
	for k, v := range in {
		out[v] = k
	}
	return out
}

func reverseBlockMetaMap(in map[string]turns.BlockMetadataKey) map[turns.BlockMetadataKey]string {
	out := make(map[turns.BlockMetadataKey]string, len(in))
	for k, v := range in {
		out[v] = k
	}
	return out
}

func parseBlockKind(raw string) turns.BlockKind {
	s := strings.ToLower(strings.TrimSpace(raw))
	for _, k := range knownKinds {
		if k.String() == s {
			return k
		}
	}
	return turns.BlockKindOther
}

func decodeString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	default:
		return ""
	}
}

func decodeMap(v any) map[string]any {
	if v == nil {
		return nil
	}
	switch m := v.(type) {
	case map[string]any:
		return m
	default:
		return nil
	}
}

func decodeSlice(v any) []any {
	if v == nil {
		return nil
	}
	switch a := v.(type) {
	case []any:
		return a
	default:
		return nil
	}
}

func cloneJSONValue(v any) any {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return v
	}
	return out
}

func canonicalTurnDataKey(raw string) turns.TurnDataKey {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if k, ok := turnDataShortToID[raw]; ok {
		return k
	}
	if strings.Contains(raw, "@v") {
		return turns.TurnDataKey(raw)
	}
	return turns.NewTurnDataKey(turns.GeppettoNamespaceKey, raw, 1)
}

func canonicalTurnMetaKey(raw string) turns.TurnMetadataKey {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if k, ok := turnMetaShortToID[raw]; ok {
		return k
	}
	if strings.Contains(raw, "@v") {
		return turns.TurnMetadataKey(raw)
	}
	return turns.NewTurnMetadataKey(turns.GeppettoNamespaceKey, raw, 1)
}

func canonicalBlockMetaKey(raw string) turns.BlockMetadataKey {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if k, ok := blockMetaShortToID[raw]; ok {
		return k
	}
	if strings.Contains(raw, "@v") {
		return turns.BlockMetadataKey(raw)
	}
	return turns.NewBlockMetadataKey(turns.GeppettoNamespaceKey, raw, 1)
}

func (m *moduleRuntime) decodeTurnData(raw any) (turns.Data, error) {
	var out turns.Data
	dataMap := decodeMap(raw)
	if len(dataMap) == 0 {
		return out, nil
	}
	for k, v := range dataMap {
		id := canonicalTurnDataKey(k)
		if id == "" {
			continue
		}
		if err := turns.DataKeyFromID[any](id).Set(&out, cloneJSONValue(v)); err != nil {
			return turns.Data{}, fmt.Errorf("decode turn data key %q: %w", k, err)
		}
	}
	return out, nil
}

func (m *moduleRuntime) decodeTurnMeta(raw any) (turns.Metadata, error) {
	var out turns.Metadata
	metaMap := decodeMap(raw)
	if len(metaMap) == 0 {
		return out, nil
	}
	for k, v := range metaMap {
		id := canonicalTurnMetaKey(k)
		if id == "" {
			continue
		}
		if err := turns.TurnMetaKeyFromID[any](id).Set(&out, cloneJSONValue(v)); err != nil {
			return turns.Metadata{}, fmt.Errorf("decode turn metadata key %q: %w", k, err)
		}
	}
	return out, nil
}

func (m *moduleRuntime) decodeBlockMeta(raw any) (turns.BlockMetadata, error) {
	var out turns.BlockMetadata
	metaMap := decodeMap(raw)
	if len(metaMap) == 0 {
		return out, nil
	}
	for k, v := range metaMap {
		id := canonicalBlockMetaKey(k)
		if id == "" {
			continue
		}
		if err := turns.BlockMetaKeyFromID[any](id).Set(&out, cloneJSONValue(v)); err != nil {
			return turns.BlockMetadata{}, fmt.Errorf("decode block metadata key %q: %w", k, err)
		}
	}
	return out, nil
}

func (m *moduleRuntime) decodeBlock(raw any) (turns.Block, error) {
	obj := decodeMap(raw)
	if obj == nil {
		return turns.Block{}, fmt.Errorf("block must be an object")
	}
	b := turns.Block{
		ID:   decodeString(obj["id"]),
		Kind: parseBlockKind(decodeString(obj["kind"])),
		Role: decodeString(obj["role"]),
	}

	if payload := decodeMap(obj["payload"]); payload != nil {
		b.Payload = payload
	} else {
		// Accept top-level "text"/"args"/"result" shorthands when payload is omitted.
		p := map[string]any{}
		if v, ok := obj[turns.PayloadKeyText]; ok {
			p[turns.PayloadKeyText] = v
		}
		if v, ok := obj[turns.PayloadKeyArgs]; ok {
			p[turns.PayloadKeyArgs] = v
		}
		if v, ok := obj[turns.PayloadKeyResult]; ok {
			p[turns.PayloadKeyResult] = v
		}
		if v, ok := obj[turns.PayloadKeyID]; ok {
			p[turns.PayloadKeyID] = v
		}
		if v, ok := obj[turns.PayloadKeyName]; ok {
			p[turns.PayloadKeyName] = v
		}
		if v, ok := obj[turns.PayloadKeyError]; ok {
			p[turns.PayloadKeyError] = v
		}
		if len(p) > 0 {
			b.Payload = p
		}
	}

	meta, err := m.decodeBlockMeta(obj["metadata"])
	if err != nil {
		return turns.Block{}, err
	}
	b.Metadata = meta
	return b, nil
}

func (m *moduleRuntime) decodeTurn(raw any) (*turns.Turn, error) {
	obj := decodeMap(raw)
	if obj == nil {
		return nil, fmt.Errorf("turn must be an object")
	}
	t := &turns.Turn{
		ID: decodeString(obj["id"]),
	}
	for _, b := range decodeSlice(obj["blocks"]) {
		decoded, err := m.decodeBlock(b)
		if err != nil {
			return nil, err
		}
		t.Blocks = append(t.Blocks, decoded)
	}
	meta, err := m.decodeTurnMeta(obj["metadata"])
	if err != nil {
		return nil, err
	}
	data, err := m.decodeTurnData(obj["data"])
	if err != nil {
		return nil, err
	}
	t.Metadata = meta
	t.Data = data
	return t, nil
}

func (m *moduleRuntime) decodeTurnValue(v goja.Value) (*turns.Turn, error) {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return &turns.Turn{}, nil
	}
	return m.decodeTurn(v.Export())
}

func (m *moduleRuntime) encodeTurnData(d turns.Data) map[string]any {
	out := map[string]any{}
	d.Range(func(k turns.TurnDataKey, v any) bool {
		if short, ok := turnDataIDToShort[k]; ok {
			out[short] = v
		} else {
			out[k.String()] = v
		}
		return true
	})
	return out
}

func (m *moduleRuntime) encodeTurnMeta(meta turns.Metadata) map[string]any {
	out := map[string]any{}
	meta.Range(func(k turns.TurnMetadataKey, v any) bool {
		if short, ok := turnMetaIDToShort[k]; ok {
			out[short] = v
		} else {
			out[k.String()] = v
		}
		return true
	})
	return out
}

func (m *moduleRuntime) encodeBlockMeta(meta turns.BlockMetadata) map[string]any {
	out := map[string]any{}
	meta.Range(func(k turns.BlockMetadataKey, v any) bool {
		if short, ok := blockMetaIDToShort[k]; ok {
			out[short] = v
		} else {
			out[k.String()] = v
		}
		return true
	})
	return out
}

func (m *moduleRuntime) encodeBlock(b turns.Block) map[string]any {
	out := map[string]any{
		"id":      b.ID,
		"kind":    b.Kind.String(),
		"role":    b.Role,
		"payload": b.Payload,
	}
	if meta := m.encodeBlockMeta(b.Metadata); len(meta) > 0 {
		out["metadata"] = meta
	}
	return out
}

func (m *moduleRuntime) encodeTurn(t *turns.Turn) map[string]any {
	if t == nil {
		return map[string]any{
			"id":       "",
			"blocks":   []any{},
			"metadata": map[string]any{},
			"data":     map[string]any{},
		}
	}
	blocks := make([]any, 0, len(t.Blocks))
	for _, b := range t.Blocks {
		blocks = append(blocks, m.encodeBlock(b))
	}
	out := map[string]any{
		"id":     t.ID,
		"blocks": blocks,
	}
	if meta := m.encodeTurnMeta(t.Metadata); len(meta) > 0 {
		out["metadata"] = meta
	}
	if data := m.encodeTurnData(t.Data); len(data) > 0 {
		out["data"] = data
	}
	return out
}

func (m *moduleRuntime) encodeTurnValue(t *turns.Turn) (goja.Value, error) {
	return m.toJSValue(m.encodeTurn(t)), nil
}

func (m *moduleRuntime) toJSValue(v any) goja.Value {
	if v == nil {
		return goja.Null()
	}
	switch x := v.(type) {
	case goja.Value:
		return x
	case map[string]any:
		obj := m.vm.NewObject()
		for k, vv := range x {
			_ = obj.Set(k, m.toJSValue(vv))
		}
		return obj
	case []any:
		arr := m.vm.NewArray()
		for i, vv := range x {
			_ = arr.Set(strconv.Itoa(i), m.toJSValue(vv))
		}
		return arr
	case []string:
		arr := m.vm.NewArray()
		for i, vv := range x {
			_ = arr.Set(strconv.Itoa(i), vv)
		}
		return arr
	default:
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			arr := m.vm.NewArray()
			for i := 0; i < rv.Len(); i++ {
				_ = arr.Set(strconv.Itoa(i), m.toJSValue(rv.Index(i).Interface()))
			}
			return arr
		}
		if rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
			obj := m.vm.NewObject()
			iter := rv.MapRange()
			for iter.Next() {
				k := iter.Key().String()
				_ = obj.Set(k, m.toJSValue(iter.Value().Interface()))
			}
			return obj
		}
		return m.vm.ToValue(v)
	}
}
