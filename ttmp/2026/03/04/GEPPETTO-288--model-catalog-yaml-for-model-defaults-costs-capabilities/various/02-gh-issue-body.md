## Summary

Add a YAML-backed **Model Catalog** to Geppetto that maps `ai-engine` (model slug) → default settings, limits, capabilities (thinking / reasoning controls), and pricing. Support a **local override YAML** so operators can add or patch model entries without waiting for a release.

This enables “model-aware defaults” like:
- selecting `ai-api-type` automatically (e.g. OpenAI reasoning models default to `openai-responses`),
- setting a sensible default `ai-max-response-tokens` per model,
- advertising supported thinking levels and sanitizing unsupported params,
- (later) computing per-call cost from usage tokens + catalog pricing.

## Motivation

Today Geppetto relies on duplicated heuristics (prefix checks like `o1/o3/o4/gpt-5`) scattered across engine factory, provider request builders, and JS module code. There’s no canonical data source for model limits/defaults/pricing/capabilities.

## Proposed Approach (v1)

- Introduce `pkg/models` (name TBD) with:
  - embedded built-in catalog YAML (ships with binaries),
  - optional local override YAML loaded from `${XDG_CONFIG_HOME:-~/.config}/pinocchio/models.yaml` (and/or env override),
  - deterministic merge/validation rules.
- Add a StepSettings normalization step that:
  - resolves `ai-api-type=auto` to a concrete provider using the catalog,
  - applies catalog defaults when the corresponding StepSettings fields are unset,
  - sanitizes or errors when the chosen model disallows certain params (e.g. temperature/top_p on reasoning models).
- Replace duplicated “reasoning model” heuristics with catalog-driven checks (keep heuristic fallback for unknown slugs).

## Acceptance Criteria

- A built-in model catalog is available in code and can be queried by exact model slug.
- A local override file can:
  - add a new model slug,
  - patch defaults/limits/capabilities for an existing slug,
  - and takes precedence deterministically.
- `ai-api-type=auto` is supported and resolves to a concrete provider before engine creation.
- `ai-max-response-tokens` defaults can be applied per model (when unset).
- Unit tests cover catalog load + merge + normalization behavior.
- No regression for unknown models: fallback heuristics still work.

## Implementation Notes

- Research / design doc is attached as a comment (file: `ttmp/2026-03-04/01-model-catalog-design-and-implementation-guide.md`).

## Task Breakdown

- [ ] Add `pkg/models` types + loader + merge + validation
- [ ] Add embedded `catalog.yaml` (minimal starter set of models)
- [ ] Add local override loading + env override option
- [ ] Add `ApiTypeAuto` + update chat flag schema to include `auto`
- [ ] Normalize StepSettings prior to engine creation (central hook)
- [ ] Migrate duplicated heuristics to catalog usage (with fallback)
- [ ] Add tests

