//go:build geppetto_js_hardcut_contract
// +build geppetto_js_hardcut_contract

package geppetto

import "testing"

// TestHardCutPublicSurfaceContract locks the intended hard-cut Geppetto JS
// surface before implementation work starts. It is behind an explicit build tag
// so the current compatibility implementation can keep passing the default
// suite while the new wrapper-first API is developed.
//
// Phase-0 contract notes:
//   - InferenceSettings objects are produced by Geppetto inference profile
//     registry resolution only; there is no public gp.inferenceSettings()
//     builder in the first pass.
//   - Accepted gp.inferenceProfiles.load(...) source forms are intended to be:
//     YAML path, yaml:PATH, yaml://PATH, SQLite path, sqlite:PATH, and
//     sqlite-dsn:DSN.
//   - Pinocchio unified config documents are intentionally out of scope for
//     gp.inferenceProfiles.load(...).
func TestHardCutPublicSurfaceContract(t *testing.T) {
	rt := newJSRuntime(t, Options{})

	mustRunJS(t, rt, `
		const gp = require("geppetto");

		function assert(cond, msg) {
			if (!cond) throw new Error(msg);
		}
		function hasOwn(obj, key) {
			return Object.prototype.hasOwnProperty.call(obj, key);
		}

		const required = [
			"version",
			"consts",
			"agent",
			"inferenceProfiles",
			"turn",
			"engine",
			"tool",
			"toolRegistry",
			"schema",
		];
		for (const key of required) {
			assert(hasOwn(gp, key), "missing hard-cut export: " + key);
		}

		const removedTopLevel = [
			"chat",
			"inferenceSettings",
			"createBuilder",
			"createSession",
			"runInference",
			"profiles",
			"engines",
			"turns",
			"runner",
			"schemas",
			"middlewares",
			"tools",
			"embeddings",
			"unsafe",
			"events",
		];
		for (const key of removedTopLevel) {
			assert(!hasOwn(gp, key), "legacy export should be absent: " + key);
		}

		if (gp.turns) {
			assert(!hasOwn(gp.turns, "newTurn"), "legacy turns.newTurn should be absent");
		}
		if (gp.engines) {
			assert(!hasOwn(gp.engines, "fromConfig"), "legacy engines.fromConfig should be absent");
		}
		if (gp.runner) {
			assert(!hasOwn(gp.runner, "run"), "legacy runner.run should be absent");
		}
	`)
}
