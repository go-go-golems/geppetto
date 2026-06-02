package geppetto

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

type turnRef struct {
	api  *moduleRuntime
	turn *turns.Turn
}

type turnBuilderRef struct {
	api  *moduleRuntime
	turn *turns.Turn
}

func (m *moduleRuntime) turnBuilder(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) == 0 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
		return m.newTurnBuilderObject(&turnBuilderRef{api: m, turn: &turns.Turn{}})
	}
	base, err := m.requireTurnRef(call.Arguments[0])
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	turn := base.turn.Clone()
	// A builder created from an existing turn represents a continuation value,
	// not an exact clone. Clear the copied ID so persistence/hydration keyed by
	// Turn.ID cannot overwrite the previous turn. Use turn.clone() when exact
	// identity preservation is desired.
	turn.ID = ""
	return m.newTurnBuilderObject(&turnBuilderRef{api: m, turn: turn})
}

type messageBuilderRef struct {
	api    *moduleRuntime
	text   string
	images []map[string]any
}

func (m *moduleRuntime) newTurnBuilderObject(ref *turnBuilderRef) *goja.Object {
	if ref == nil {
		ref = &turnBuilderRef{api: m, turn: &turns.Turn{}}
	}
	ref.api = m
	if ref.turn == nil {
		ref.turn = &turns.Turn{}
	}
	o := m.vm.NewObject()
	m.attachRef(o, ref.cloneFor(m))
	m.mustSet(o, "system", func(call goja.FunctionCall) goja.Value {
		text := ""
		if len(call.Arguments) > 0 {
			text = call.Arguments[0].String()
		}
		next := ref.cloneFor(m)
		turns.AppendBlock(next.turn, turns.NewSystemTextBlock(text))
		return m.newTurnBuilderObject(next)
	})
	m.mustSet(o, "user", func(call goja.FunctionCall) goja.Value {
		next := ref.cloneFor(m)
		if len(call.Arguments) > 0 {
			if fn, ok := goja.AssertFunction(call.Arguments[0]); ok {
				msg := &messageBuilderRef{api: m}
				_, err := fn(goja.Undefined(), m.newMessageBuilderObject(msg))
				if err != nil {
					panic(err)
				}
				turns.AppendBlock(next.turn, turns.NewUserMultimodalBlock(msg.text, msg.images))
				return m.newTurnBuilderObject(next)
			}
			turns.AppendBlock(next.turn, turns.NewUserTextBlock(call.Arguments[0].String()))
			return m.newTurnBuilderObject(next)
		}
		turns.AppendBlock(next.turn, turns.NewUserTextBlock(""))
		return m.newTurnBuilderObject(next)
	})
	m.mustSet(o, "assistant", func(call goja.FunctionCall) goja.Value {
		text := ""
		if len(call.Arguments) > 0 {
			text = call.Arguments[0].String()
		}
		next := ref.cloneFor(m)
		turns.AppendBlock(next.turn, turns.NewAssistantTextBlock(text))
		return m.newTurnBuilderObject(next)
	})
	m.mustSet(o, "metadata", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(m.vm.NewTypeError("turn().metadata requires key and value"))
		}
		key := strings.TrimSpace(call.Arguments[0].String())
		if key == "" {
			panic(m.vm.NewTypeError("turn metadata key must not be empty"))
		}
		next := ref.cloneFor(m)
		_ = turns.TurnMetaKeyFromID[any](canonicalTurnMetaKey(key)).Set(&next.turn.Metadata, cloneJSONValue(call.Arguments[1].Export()))
		return m.newTurnBuilderObject(next)
	})
	m.mustSet(o, "build", func(goja.FunctionCall) goja.Value {
		return m.newTurnObject(&turnRef{api: m, turn: ref.turn.Clone()})
	})
	return o
}

func (r *turnBuilderRef) cloneFor(api *moduleRuntime) *turnBuilderRef {
	if r == nil {
		return &turnBuilderRef{api: api, turn: &turns.Turn{}}
	}
	return &turnBuilderRef{api: api, turn: r.turn.Clone()}
}

func (m *moduleRuntime) newTurnObject(ref *turnRef) *goja.Object {
	if ref == nil {
		ref = &turnRef{api: m, turn: &turns.Turn{}}
	}
	ref.api = m
	if ref.turn == nil {
		ref.turn = &turns.Turn{}
	}
	o := m.vm.NewObject()
	m.attachRef(o, ref.cloneFor(m))
	m.mustSet(o, "toJSON", func(goja.FunctionCall) goja.Value {
		return m.toJSValue(m.encodeTurn(ref.turn.Clone()))
	})
	m.mustSet(o, "clone", func(goja.FunctionCall) goja.Value {
		return m.newTurnObject(ref.cloneFor(m))
	})
	return o
}

func (m *moduleRuntime) newMessageBuilderObject(ref *messageBuilderRef) *goja.Object {
	if ref == nil {
		ref = &messageBuilderRef{api: m}
	}
	ref.api = m
	o := m.vm.NewObject()
	m.mustSet(o, "text", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			if ref.text != "" {
				ref.text += "\n"
			}
			ref.text += call.Arguments[0].String()
		}
		return o
	})
	m.mustSet(o, "imageURL", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(m.vm.NewTypeError("imageURL requires url"))
		}
		img := map[string]any{"url": call.Arguments[0].String()}
		if len(call.Arguments) > 1 {
			if opts := decodeMap(call.Arguments[1].Export()); opts != nil {
				for k, v := range opts {
					img[k] = cloneJSONValue(v)
				}
			}
		}
		ref.images = append(ref.images, img)
		return o
	})
	m.mustSet(o, "imageFile", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(m.vm.NewTypeError("imageFile requires path"))
		}
		path := call.Arguments[0].String()
		b, err := os.ReadFile(path)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		mediaType := mediaTypeForImagePath(path)
		img := map[string]any{"media_type": mediaType, "content": base64.StdEncoding.EncodeToString(b)}
		ref.images = append(ref.images, img)
		return o
	})
	m.mustSet(o, "imageBytes", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(m.vm.NewTypeError("imageBytes requires base64/string content"))
		}
		mediaType := "application/octet-stream"
		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) && !goja.IsNull(call.Arguments[1]) {
			mediaType = call.Arguments[1].String()
		}
		ref.images = append(ref.images, map[string]any{"media_type": mediaType, "content": call.Arguments[0].String()})
		return o
	})
	return o
}

func mediaTypeForImagePath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

func (m *moduleRuntime) requireTurnRef(v goja.Value) (*turnRef, error) {
	ref := m.getRef(v)
	switch x := ref.(type) {
	case *turnRef:
		return x.cloneFor(m), nil
	case *turns.Turn:
		return &turnRef{api: m, turn: x.Clone()}, nil
	default:
		return nil, fmt.Errorf("expected Go-owned Turn wrapper, got %T", ref)
	}
}

func (r *turnRef) cloneFor(api *moduleRuntime) *turnRef {
	if r == nil {
		return &turnRef{api: api, turn: &turns.Turn{}}
	}
	return &turnRef{api: api, turn: r.turn.Clone()}
}
