package geppetto

import (
	"fmt"

	"github.com/dop251/goja"
	enginefactory "github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
)

type engineBuilderRef struct {
	api      *moduleRuntime
	settings *inferenceSettingsRef
}

func (m *moduleRuntime) engineBuilder(call goja.FunctionCall) goja.Value {
	return m.newEngineBuilderObject(&engineBuilderRef{api: m})
}

func (m *moduleRuntime) newEngineBuilderObject(ref *engineBuilderRef) *goja.Object {
	if ref == nil {
		ref = &engineBuilderRef{api: m}
	}
	ref.api = m
	o := m.vm.NewObject()
	m.attachRef(o, ref.cloneFor(m))
	m.mustSet(o, "inference", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
			panic(m.vm.NewTypeError("engine().inference requires a registry-resolved InferenceSettings wrapper"))
		}
		settingsRef, err := m.requireInferenceSettingsRef(call.Arguments[0])
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		next := ref.cloneFor(m)
		next.settings = settingsRef.cloneFor(m)
		return m.newEngineBuilderObject(next)
	})
	m.mustSet(o, "build", func(goja.FunctionCall) goja.Value {
		if ref.settings == nil || ref.settings.settings == nil {
			panic(m.vm.NewGoError(fmt.Errorf("engine().build requires inference(settings) first")))
		}
		settings := cloneInferenceSettings(ref.settings.settings)
		ensureInferenceSettingsProviderDefaults(settings)
		eng, err := enginefactory.NewEngineFromSettings(settings)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		metadata := ref.settings.provenance.toMap()
		return m.newEngineObject(&engineRef{
			Name:      "inferenceSettings",
			Engine:    eng,
			ModelInfo: settings.ModelInfo.Clone(),
			Metadata:  metadata,
		})
	})
	return o
}

func (r *engineBuilderRef) cloneFor(api *moduleRuntime) *engineBuilderRef {
	if r == nil {
		return &engineBuilderRef{api: api}
	}
	var settings *inferenceSettingsRef
	if r.settings != nil {
		settings = r.settings.cloneFor(api)
	}
	return &engineBuilderRef{api: api, settings: settings}
}
