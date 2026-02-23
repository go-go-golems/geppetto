package geppetto

import (
	"fmt"
	"math"
	"strings"

	"github.com/dop251/goja"
)

const extractorPluginAPIVersion = "cozo.extractor/v1"
const optimizerPluginAPIVersion = "gepa.optimizer/v1"

func (m *module) pluginsLoader(vm *goja.Runtime, moduleObj *goja.Object) {
	exports := moduleObj.Get("exports").(*goja.Object)
	mustSet := func(key string, v any) {
		if err := exports.Set(key, v); err != nil {
			panic(vm.NewGoError(fmt.Errorf("set %s: %w", key, err)))
		}
	}

	mustSet("EXTRACTOR_PLUGIN_API_VERSION", extractorPluginAPIVersion)
	mustSet("defineExtractorPlugin", func(call goja.FunctionCall) goja.Value {
		descriptor := call.Argument(0)
		if descriptor == nil || goja.IsUndefined(descriptor) || goja.IsNull(descriptor) {
			panic(vm.NewTypeError("plugin descriptor must be an object"))
		}
		descriptorObj := descriptor.ToObject(vm)
		if descriptorObj == nil || descriptorObj.ClassName() != "Object" {
			panic(vm.NewTypeError("plugin descriptor must be an object"))
		}

		apiVersion := readStringField(vm, descriptorObj, "apiVersion", false)
		if apiVersion == "" {
			apiVersion = extractorPluginAPIVersion
		}
		if apiVersion != extractorPluginAPIVersion {
			panic(vm.NewTypeError(
				"unsupported plugin descriptor apiVersion %q (expected %q)",
				apiVersion, extractorPluginAPIVersion,
			))
		}

		kind := readStringField(vm, descriptorObj, "kind", false)
		if kind == "" {
			kind = "extractor"
		}
		if kind != "extractor" {
			panic(vm.NewTypeError("plugin descriptor kind must be %q, got %q", "extractor", kind))
		}

		id := readStringField(vm, descriptorObj, "id", true)
		name := readStringField(vm, descriptorObj, "name", true)

		createVal := descriptorObj.Get("create")
		if _, ok := goja.AssertFunction(createVal); !ok {
			panic(vm.NewTypeError("plugin descriptor create must be a function"))
		}

		out := vm.NewObject()
		_ = out.Set("apiVersion", apiVersion)
		_ = out.Set("kind", kind)
		_ = out.Set("id", id)
		_ = out.Set("name", name)
		_ = out.Set("create", createVal)

		return freezeObject(vm, out)
	})

	mustSet("wrapExtractorRun", func(call goja.FunctionCall) goja.Value {
		runImplVal := call.Argument(0)
		runImpl, ok := goja.AssertFunction(runImplVal)
		if !ok {
			panic(vm.NewTypeError("plugin run must be a function"))
		}

		return vm.ToValue(func(call goja.FunctionCall) goja.Value {
			canonicalInput := canonicalizePluginRunInput(vm, call.Argument(0))
			options := normalizeRunOptions(vm, call.Argument(1))
			ret, err := runImpl(goja.Undefined(), canonicalInput, options)
			if err != nil {
				panic(err)
			}
			return ret
		})
	})

	mustSet("OPTIMIZER_PLUGIN_API_VERSION", optimizerPluginAPIVersion)
	mustSet("defineOptimizerPlugin", func(call goja.FunctionCall) goja.Value {
		descriptor := call.Argument(0)
		if descriptor == nil || goja.IsUndefined(descriptor) || goja.IsNull(descriptor) {
			panic(vm.NewTypeError("plugin descriptor must be an object"))
		}
		descriptorObj := descriptor.ToObject(vm)
		if descriptorObj == nil || descriptorObj.ClassName() != "Object" {
			panic(vm.NewTypeError("plugin descriptor must be an object"))
		}

		apiVersion := readStringField(vm, descriptorObj, "apiVersion", false)
		if apiVersion == "" {
			apiVersion = optimizerPluginAPIVersion
		}
		if apiVersion != optimizerPluginAPIVersion {
			panic(vm.NewTypeError(
				"unsupported plugin descriptor apiVersion %q (expected %q)",
				apiVersion, optimizerPluginAPIVersion,
			))
		}

		kind := readStringField(vm, descriptorObj, "kind", false)
		if kind == "" {
			kind = "optimizer"
		}
		if kind != "optimizer" {
			panic(vm.NewTypeError("plugin descriptor kind must be %q, got %q", "optimizer", kind))
		}

		id := readStringField(vm, descriptorObj, "id", true)
		name := readStringField(vm, descriptorObj, "name", true)

		createVal := descriptorObj.Get("create")
		if _, ok := goja.AssertFunction(createVal); !ok {
			panic(vm.NewTypeError("plugin descriptor create must be a function"))
		}

		out := vm.NewObject()
		_ = out.Set("apiVersion", apiVersion)
		_ = out.Set("kind", kind)
		_ = out.Set("id", id)
		_ = out.Set("name", name)
		_ = out.Set("create", createVal)

		return freezeObject(vm, out)
	})
}

func readStringField(vm *goja.Runtime, obj *goja.Object, key string, required bool) string {
	v := obj.Get(key)
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		if required {
			panic(vm.NewTypeError("plugin descriptor %s is required", key))
		}
		return ""
	}
	s, ok := v.Export().(string)
	if !ok {
		panic(vm.NewTypeError("plugin descriptor %s must be a string", key))
	}
	out := strings.TrimSpace(s)
	if required && out == "" {
		panic(vm.NewTypeError("plugin descriptor %s is required", key))
	}
	return out
}

func canonicalizePluginRunInput(vm *goja.Runtime, input goja.Value) goja.Value {
	if input == nil || goja.IsUndefined(input) || goja.IsNull(input) {
		panic(vm.NewTypeError("plugin run input must be an object"))
	}
	inputObj := input.ToObject(vm)
	if inputObj == nil || inputObj.ClassName() != "Object" {
		panic(vm.NewTypeError("plugin run input must be an object"))
	}

	transcriptVal := inputObj.Get("transcript")
	transcript, ok := transcriptVal.Export().(string)
	if !ok || strings.TrimSpace(transcript) == "" {
		panic(vm.NewTypeError("plugin run input.transcript must be a non-empty string"))
	}

	prompt := readOptionalInputString(inputObj.Get("prompt"))
	profile := readOptionalInputString(inputObj.Get("profile"))
	timeoutMs := readOptionalTimeout(inputObj.Get("timeoutMs"))
	engineOptions := cloneOptionalObject(vm, inputObj.Get("engineOptions"))

	out := vm.NewObject()
	_ = out.Set("transcript", transcript)
	_ = out.Set("prompt", prompt)
	_ = out.Set("profile", profile)
	_ = out.Set("timeoutMs", timeoutMs)
	_ = out.Set("engineOptions", engineOptions)
	return freezeObject(vm, out)
}

func readOptionalInputString(v goja.Value) string {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return ""
	}
	s, ok := v.Export().(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func readOptionalTimeout(v goja.Value) int {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return 120000
	}
	num, ok := v.Export().(float64)
	if !ok || math.IsNaN(num) || math.IsInf(num, 0) || num <= 0 {
		return 120000
	}
	return int(num)
}

func cloneOptionalObject(vm *goja.Runtime, v goja.Value) goja.Value {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return goja.Null()
	}
	obj := v.ToObject(vm)
	if obj == nil || obj.ClassName() != "Object" {
		return goja.Null()
	}
	copyObj := vm.NewObject()
	for _, key := range obj.Keys() {
		_ = copyObj.Set(key, obj.Get(key))
	}
	return freezeObject(vm, copyObj)
}

func normalizeRunOptions(vm *goja.Runtime, v goja.Value) goja.Value {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return vm.NewObject()
	}
	obj := v.ToObject(vm)
	if obj == nil || obj.ClassName() != "Object" {
		return vm.NewObject()
	}
	return obj
}

func freezeObject(vm *goja.Runtime, obj *goja.Object) goja.Value {
	objectVal := vm.Get("Object")
	objectObj := objectVal.ToObject(vm)
	freezeVal := objectObj.Get("freeze")
	freezeFn, ok := goja.AssertFunction(freezeVal)
	if !ok {
		return obj
	}
	ret, err := freezeFn(objectVal, obj)
	if err != nil {
		panic(err)
	}
	return ret
}
