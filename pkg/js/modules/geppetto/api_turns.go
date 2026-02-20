package geppetto

import (
	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

func (m *moduleRuntime) turnsNormalize(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(m.vm.NewTypeError("turns.normalize requires turn"))
	}
	t, err := m.decodeTurnValue(call.Arguments[0])
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	normalized := t.Clone()
	if normalized == nil {
		normalized = &turns.Turn{}
	}
	v, err := m.encodeTurnValue(normalized)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return v
}

func (m *moduleRuntime) turnsNewTurn(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) == 0 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
		v, _ := m.encodeTurnValue(&turns.Turn{})
		return v
	}
	t, err := m.decodeTurnValue(call.Arguments[0])
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	v, err := m.encodeTurnValue(t)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return v
}

func (m *moduleRuntime) turnsAppendBlock(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(m.vm.NewTypeError("turns.appendBlock requires turn and block"))
	}
	t, err := m.decodeTurnValue(call.Arguments[0])
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	b, err := m.decodeBlock(call.Arguments[1].Export())
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	turns.AppendBlock(t, b)
	v, err := m.encodeTurnValue(t)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return v
}

func (m *moduleRuntime) turnsNewUserBlock(call goja.FunctionCall) goja.Value {
	text := ""
	if len(call.Arguments) > 0 {
		text = call.Arguments[0].String()
	}
	return m.toJSValue(m.encodeBlock(turns.NewUserTextBlock(text)))
}

func (m *moduleRuntime) turnsNewSystemBlock(call goja.FunctionCall) goja.Value {
	text := ""
	if len(call.Arguments) > 0 {
		text = call.Arguments[0].String()
	}
	return m.toJSValue(m.encodeBlock(turns.NewSystemTextBlock(text)))
}

func (m *moduleRuntime) turnsNewAssistantBlock(call goja.FunctionCall) goja.Value {
	text := ""
	if len(call.Arguments) > 0 {
		text = call.Arguments[0].String()
	}
	return m.toJSValue(m.encodeBlock(turns.NewAssistantTextBlock(text)))
}

func (m *moduleRuntime) turnsNewToolCallBlock(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 3 {
		panic(m.vm.NewTypeError("turns.newToolCallBlock requires id, name, args"))
	}
	id := call.Arguments[0].String()
	name := call.Arguments[1].String()
	args := call.Arguments[2].Export()
	return m.toJSValue(m.encodeBlock(turns.NewToolCallBlock(id, name, args)))
}

func (m *moduleRuntime) turnsNewToolUseBlock(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(m.vm.NewTypeError("turns.newToolUseBlock requires id, result[, error]"))
	}
	id := call.Arguments[0].String()
	result := call.Arguments[1].Export()
	errText := ""
	if len(call.Arguments) > 2 {
		errText = call.Arguments[2].String()
	}
	return m.toJSValue(m.encodeBlock(turns.NewToolUseBlockWithError(id, result, errText)))
}
