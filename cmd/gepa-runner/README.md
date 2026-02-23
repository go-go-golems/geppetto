# gepa-runner

`gepa-runner` is a small CLI that runs a **GEPA-style reflective prompt evolution loop** on top of:

- **Geppetto** (Go inference + tooling runtime)
- **JS evaluation plugins** (goja + `require("geppetto")`)

It is meant to feel similar to the JS scripting approach in `cozo-relationship-js-runner`, but focused on **prompt optimization / benchmarking**.

## What this implements

This is a **GEPA-inspired** implementation:

- Minibatch sampling over a dataset
- Natural-language **reflection** (LLM proposes prompt edits based on traces/feedback)
- Simple Pareto selection when multi-objective scores are returned by the evaluator

It does **not** (yet) port every detail of the reference implementation (e.g., specialized crossover/merge logic),
but it is designed so you can add that on top of the same primitives.

## Quick start

1. Write a JS plugin that exports an optimizer descriptor:

```js
const gp = require("geppetto");
const plugins = require("geppetto/plugins");

module.exports = plugins.defineOptimizerPlugin({
  apiVersion: plugins.OPTIMIZER_PLUGIN_API_VERSION,
  kind: "optimizer",
  id: "my.task",
  name: "My Task",

  create(ctx) {
    return {
      dataset() { return [ /* examples */ ]; },
      evaluate(input, options) {
        const { candidate, example } = input;
        // Run geppetto inference, compare to expected output
        return { score: 1.0, output: "...", feedback: "..." };
      }
    };
  }
});
```

2. Run optimization:

```bash
gepa-runner optimize \
  --script ./cmd/gepa-runner/scripts/toy_math_optimizer.js \
  --seed "Answer the question." \
  --max-evals 200 \
  --batch-size 8 \
  --out-prompt best_prompt.txt \
  --out-report run_report.json \
  --profile 4o-mini
```

You can pass a dataset file instead of `dataset()`:

```bash
gepa-runner optimize \
  --script ./my_plugin.js \
  --dataset ./data/train.jsonl \
  --seed-file ./seed_prompt.txt
```

## Evaluator contract

The plugin instance must implement:

- `evaluate(input, options) -> object`
  - `input.candidate` is an object (e.g., `{prompt: "..."}`)
  - `input.example` is an example (any JSON value)
  - must return at least:
    - `score` (number, higher = better)
  - optional:
    - `objectiveScores` or `objectives` (object: `{name: number}`) for multi-objective Pareto selection
    - `output`, `feedback`, `trace` (any JSON) — included in reflection side-info

Optional:

- `dataset() -> array` (used if `--dataset` is not provided)

## Notes

- The optimizer currently mutates the `"prompt"` field (or falls back to the first key in `candidate`).
- Each `(candidate, example)` evaluation counts as **1** call against `--max-evals`.
