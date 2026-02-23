package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	gepaopt "github.com/go-go-golems/geppetto/pkg/optimizer/gepa"
	"github.com/pkg/errors"
)

const optimizerPluginAPIVersion = "gepa.optimizer/v1"

type optimizerPluginMeta struct {
	APIVersion string
	Kind       string
	ID         string
	Name       string
}

type pluginEvaluateOptions struct {
	Profile       string
	EngineOptions map[string]any
	Tags          map[string]any
}

type optimizerPlugin struct {
	rt       *jsRuntime
	meta     optimizerPluginMeta
	instance *goja.Object

	evaluateFn goja.Callable
	datasetFn  goja.Callable
}

func loadOptimizerPlugin(rt *jsRuntime, absScriptPath string, hostContext map[string]any) (*optimizerPlugin, optimizerPluginMeta, error) {
	if rt == nil || rt.vm == nil || rt.reqMod == nil {
		return nil, optimizerPluginMeta{}, errors.New("plugin loader: runtime is nil")
	}
	if strings.TrimSpace(absScriptPath) == "" {
		return nil, optimizerPluginMeta{}, errors.New("plugin loader: script path is empty")
	}

	// NOTE: require expects absolute path for local files when we use abs path.
	var exported goja.Value
	var err error
	exported, err = rt.reqMod.Require(absScriptPath)
	if err != nil {
		return nil, optimizerPluginMeta{}, errors.Wrap(err, "plugin loader: require script module")
	}

	descriptorObj := exported.ToObject(rt.vm)
	if descriptorObj == nil {
		return nil, optimizerPluginMeta{}, fmt.Errorf("plugin loader: script module did not export an object descriptor")
	}

	meta, err := decodeOptimizerPluginMeta(descriptorObj)
	if err != nil {
		return nil, optimizerPluginMeta{}, err
	}

	createVal := descriptorObj.Get("create")
	createFn, ok := goja.AssertFunction(createVal)
	if !ok {
		return nil, optimizerPluginMeta{}, fmt.Errorf("plugin loader: descriptor.create must be a function")
	}

	if hostContext == nil {
		hostContext = map[string]any{}
	}

	instanceVal, err := createFn(descriptorObj, rt.vm.ToValue(hostContext))
	if err != nil {
		return nil, optimizerPluginMeta{}, errors.Wrap(err, "plugin loader: descriptor.create failed")
	}
	instanceObj := instanceVal.ToObject(rt.vm)
	if instanceObj == nil {
		return nil, optimizerPluginMeta{}, fmt.Errorf("plugin loader: descriptor.create must return an object instance")
	}

	evaluateVal := instanceObj.Get("evaluate")
	evaluateFn, ok := goja.AssertFunction(evaluateVal)
	if !ok {
		return nil, optimizerPluginMeta{}, fmt.Errorf("plugin loader: plugin instance.evaluate must be a function")
	}

	// dataset() is optional if dataset file is provided, but we’ll attempt to bind it here.
	var datasetFn goja.Callable
	if dv := instanceObj.Get("dataset"); dv != nil && !goja.IsUndefined(dv) && !goja.IsNull(dv) {
		if fn, ok := goja.AssertFunction(dv); ok {
			datasetFn = fn
		}
	}
	if datasetFn == nil {
		if dv := instanceObj.Get("getDataset"); dv != nil && !goja.IsUndefined(dv) && !goja.IsNull(dv) {
			if fn, ok := goja.AssertFunction(dv); ok {
				datasetFn = fn
			}
		}
	}

	p := &optimizerPlugin{
		rt:         rt,
		meta:       meta,
		instance:   instanceObj,
		evaluateFn: evaluateFn,
		datasetFn:  datasetFn,
	}

	return p, meta, nil
}

func decodeOptimizerPluginMeta(descriptorObj *goja.Object) (optimizerPluginMeta, error) {

	apiVersion := strings.TrimSpace(descriptorObj.Get("apiVersion").String())
	kind := strings.TrimSpace(descriptorObj.Get("kind").String())
	id := strings.TrimSpace(descriptorObj.Get("id").String())
	name := strings.TrimSpace(descriptorObj.Get("name").String())

	if apiVersion == "" {
		return optimizerPluginMeta{}, fmt.Errorf("plugin loader: descriptor.apiVersion is required")
	}
	if apiVersion != optimizerPluginAPIVersion {
		return optimizerPluginMeta{}, fmt.Errorf("plugin loader: unsupported apiVersion %q (expected %q)", apiVersion, optimizerPluginAPIVersion)
	}
	if kind != "optimizer" {
		return optimizerPluginMeta{}, fmt.Errorf("plugin loader: descriptor.kind must be %q", "optimizer")
	}
	if id == "" {
		return optimizerPluginMeta{}, fmt.Errorf("plugin loader: descriptor.id is required")
	}
	if name == "" {
		return optimizerPluginMeta{}, fmt.Errorf("plugin loader: descriptor.name is required")
	}

	// Sanity: ensure descriptor.create exists.
	if cv := descriptorObj.Get("create"); cv == nil || goja.IsUndefined(cv) || goja.IsNull(cv) {
		return optimizerPluginMeta{}, fmt.Errorf("plugin loader: descriptor.create is required")
	}
	if _, ok := goja.AssertFunction(descriptorObj.Get("create")); !ok {
		return optimizerPluginMeta{}, fmt.Errorf("plugin loader: descriptor.create must be a function")
	}

	return optimizerPluginMeta{
		APIVersion: apiVersion,
		Kind:       kind,
		ID:         id,
		Name:       name,
	}, nil
}

func (p *optimizerPlugin) Dataset() ([]any, error) {
	if p == nil || p.rt == nil || p.instance == nil {
		return nil, fmt.Errorf("plugin dataset: plugin not initialized")
	}
	if p.datasetFn == nil {
		return nil, fmt.Errorf("plugin dataset: instance.dataset() not found (provide --dataset)")
	}

	ret, err := p.datasetFn(p.instance, goja.Undefined())
	if err != nil {
		return nil, errors.Wrap(err, "plugin dataset: call failed")
	}

	decoded, err := decodeJSReturnValue(ret)
	if err != nil {
		return nil, errors.Wrap(err, "plugin dataset: invalid return value")
	}
	arr, ok := decoded.([]any)
	if ok {
		return arr, nil
	}
	// handle []interface{} from json.Unmarshal etc.
	if arr2, ok := decoded.([]interface{}); ok {
		out := make([]any, 0, len(arr2))
		out = append(out, arr2...)
		return out, nil
	}
	return nil, fmt.Errorf("plugin dataset: expected array, got %T", decoded)
}

func (p *optimizerPlugin) Evaluate(
	candidate gepaopt.Candidate,
	exampleIndex int,
	example any,
	opts pluginEvaluateOptions,
) (gepaopt.EvalResult, error) {
	if p == nil || p.rt == nil || p.instance == nil || p.evaluateFn == nil {
		return gepaopt.EvalResult{}, fmt.Errorf("plugin evaluate: plugin not initialized")
	}

	input := map[string]any{
		"candidate":    candidate,
		"example":      example,
		"exampleIndex": exampleIndex,
	}
	options := map[string]any{
		"profile":       strings.TrimSpace(opts.Profile),
		"engineOptions": opts.EngineOptions,
		"tags":          opts.Tags,
	}

	ret, err := p.evaluateFn(p.instance, p.rt.vm.ToValue(input), p.rt.vm.ToValue(options))
	if err != nil {
		return gepaopt.EvalResult{}, errors.Wrap(err, "plugin evaluate: call failed")
	}

	decoded, err := decodeJSReturnValue(ret)
	if err != nil {
		return gepaopt.EvalResult{}, errors.Wrap(err, "plugin evaluate: invalid return value")
	}

	er, err := decodeEvalResult(decoded)
	if err != nil {
		return gepaopt.EvalResult{}, err
	}
	er.Raw = decoded
	return er, nil
}

// decodeJSReturnValue mirrors the cozo runner behavior:
// - if JS returns a string, attempt JSON parsing
// - if JS returns bytes, attempt JSON parsing
// - otherwise, return exported value
func decodeJSReturnValue(ret goja.Value) (any, error) {
	if ret == nil || goja.IsUndefined(ret) || goja.IsNull(ret) {
		return nil, fmt.Errorf("returned null/undefined")
	}
	if raw, ok := ret.Export().(string); ok {
		if strings.TrimSpace(raw) == "" {
			return nil, fmt.Errorf("returned empty string")
		}
		var jsonValue any
		if err := json.Unmarshal([]byte(raw), &jsonValue); err == nil {
			return jsonValue, nil
		}
		return raw, nil
	}
	if bytes, ok := ret.Export().([]uint8); ok {
		var jsonValue any
		if err := json.Unmarshal(bytes, &jsonValue); err == nil {
			return jsonValue, nil
		}
		return bytes, nil
	}
	return ret.Export(), nil
}

func decodeEvalResult(v any) (gepaopt.EvalResult, error) {
	switch x := v.(type) {
	case map[string]any:
		return decodeEvalResultFromMap(x)
	case float64:
		return gepaopt.EvalResult{Score: x}, nil
	case int:
		return gepaopt.EvalResult{Score: float64(x)}, nil
	default:
		return gepaopt.EvalResult{}, fmt.Errorf("evaluator must return an object with {score}, got %T", v)
	}
}

func decodeEvalResultFromMap(m map[string]any) (gepaopt.EvalResult, error) {
	scoreRaw, ok := m["score"]
	if !ok {
		// allow "value" fallback
		scoreRaw, ok = m["value"]
	}
	if !ok {
		return gepaopt.EvalResult{}, fmt.Errorf("evaluator return value missing required field: score")
	}

	score, err := toFloat(scoreRaw)
	if err != nil {
		return gepaopt.EvalResult{}, fmt.Errorf("invalid score: %w", err)
	}

	var objScores gepaopt.ObjectiveScores
	for _, key := range []string{"objectiveScores", "objectives"} {
		if v, ok := m[key]; ok && v != nil {
			objScores, _ = decodeObjectiveScores(v)
			break
		}
	}

	out := gepaopt.EvalResult{
		Score:      score,
		Objectives: objScores,
		Output:     m["output"],
		Feedback:   m["feedback"],
		Trace:      m["trace"],
	}

	if notes, ok := m["notes"].(string); ok {
		out.EvaluatorNotes = notes
	} else if notes, ok := m["evaluatorNotes"].(string); ok {
		out.EvaluatorNotes = notes
	}

	return out, nil
}

func decodeObjectiveScores(v any) (gepaopt.ObjectiveScores, error) {
	out := gepaopt.ObjectiveScores{}
	switch x := v.(type) {
	case map[string]any:
		for k, vv := range x {
			f, err := toFloat(vv)
			if err != nil {
				continue
			}
			out[k] = f
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty objective scores")
	}
	return out, nil
}

func toFloat(v any) (float64, error) {
	switch x := v.(type) {
	case float64:
		return x, nil
	case float32:
		return float64(x), nil
	case int:
		return float64(x), nil
	case int64:
		return float64(x), nil
	case json.Number:
		return x.Float64()
	case string:
		if strings.TrimSpace(x) == "" {
			return 0, fmt.Errorf("empty string")
		}
		num := json.Number(strings.TrimSpace(x))
		return num.Float64()
	default:
		return 0, fmt.Errorf("unsupported numeric type %T", v)
	}
}

// Ensure the require package is linked (some Go compilers prune unused imports).
var _ = require.Require
