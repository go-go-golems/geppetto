---
Title: 'Analysis: where Turn YAML serde is used (geppetto vs moments vs pinocchio)'
Ticket: 003-REVISE-DATA-ACCESS-API
Status: active
Topics:
    - geppetto
    - turns
    - go
    - architecture
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/cmd/llm-runner/api.go
      Note: |-
        Reads/parses input_turn.yaml and final_turn*.yaml to render the run UI (real usage path).
        Real usage: parses turn YAML artifacts to render run viewer UI
    - Path: geppetto/cmd/llm-runner/main.go
      Note: |-
        CLI that writes input_turn.yaml artifacts (real usage path).
        Real usage: writes input_turn.yaml
    - Path: geppetto/pkg/inference/fixtures/fixtures.go
      Note: |-
        Loads input turns from YAML and writes turn YAML artifacts during fixture runs (real usage path).
        Real usage: reads Turn YAML inputs and writes YAML artifacts
    - Path: geppetto/pkg/turns/serde/serde.go
      Note: |-
        Defines ToYAML/FromYAML/SaveTurnYAML/LoadTurnYAML.
        Defines the YAML import/export surface
    - Path: geppetto/pkg/turns/serde/serde_test.go
      Note: Tests YAML round-trip contract (not production usage but constrains design).
ExternalSources: []
Summary: YAML Turn serde is used in geppetto (fixtures + llm-runner artifacts/UI) and tests/docs; no direct usage found in moments or pinocchio runtime code in this workspace.
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: Assess whether YAML-friendly snapshots are a real product requirement or mostly a debugging/artifact feature, to inform the RawMessage vs any/registry decision.
WhenToUse: Use when deciding storage/serde strategy for Turn.Data/Metadata and whether to prioritize YAML readability vs typed round-trip fidelity/performance.
---


# Analysis: where Turn YAML serde is used (geppetto vs moments vs pinocchio)

## Question

Is the current `geppetto/pkg/turns/serde` YAML import/export used in real runtime flows (especially in moments/pinocchio), or is it primarily a “human readable snapshot/debugging” feature?

## What was searched

Repository-wide searches for:

- `serde.ToYAML`, `serde.FromYAML`, `serde.SaveTurnYAML`, `serde.LoadTurnYAML`
- the import path `github.com/go-go-golems/geppetto/pkg/turns/serde`
- raw symbols `ToYAML(`/`FromYAML(` etc.

## Findings

### 1) Geppetto: **Yes, used in real flows**

In this workspace, YAML serde is not just tests—it is used by geppetto tooling.

#### A) `geppetto/pkg/inference/fixtures` (load inputs + write artifacts)

`fixtures.LoadFixtureOrTurn` supports loading a Turn from YAML (fallback path), and `fixtures.ExecuteFixture` writes `input_turn.yaml` and multiple `final_turn*.yaml` artifacts during runs.

This is used for:

- reproducible fixtures / recordings
- debugging inference pipelines
- artifact-driven reports

#### B) `geppetto/cmd/llm-runner` (CLI + run viewer UI)

The `llm-runner` command:

- writes `input_turn.yaml` via `serde.SaveTurnYAML`
- later reads/parses those YAML turn artifacts in `api.go` via `serde.FromYAML` to render a run viewer UI

This creates a “YAML as artifact interchange” loop:

- produce YAML snapshots during a run
- consume them later for inspection/reporting/UI

So YAML readability matters most for *developer experience* of these tools, not for production inference correctness.

### 2) Moments: **No direct runtime usage found**

Within this workspace, there are no imports/calls to `geppetto/pkg/turns/serde` from `moments/` code (excluding moments’ `ttmp` docs).

That suggests moments does not currently depend on YAML Turn serde in production paths.

### 3) Pinocchio: **No direct runtime usage found**

Within this workspace, there are no imports/calls to `geppetto/pkg/turns/serde` from `pinocchio/` code.

Pinocchio appears to use its own persistence/serialization approaches and is not coupled to this YAML contract.

## Implications for the design (RawMessage vs any+registry)

### If YAML readability is “not that important”

Based on usage, YAML serde appears primarily:

- artifact generation for developer workflows (fixtures + run viewer)
- tests/docs

Not a moments/pinocchio runtime dependency.

So it’s reasonable to de-prioritize “human friendly YAML” as a core product requirement, *if* you’re comfortable with geppetto developer tooling changing.

### If we keep geppetto tooling expectations

If you care about the `llm-runner` artifact loop remaining readable and stable, then:

- `any` storage keeps readable YAML easy, but loses typed round-trip for structs unless we add a registry/codec (or accept map-shaped decoded values).
- `json.RawMessage` storage restores typed round-trip but requires the “RawMessage ↔ YAML bridge” to keep YAML readable.

## Recommendations

- **Clarify intent**:
  - Is YAML an artifact/debug format only? (then readability can be “nice to have”)
  - Or is YAML a supported interchange format for workflows? (then it’s a contract)
- **If YAML is debugging-only**:
  - Prefer correctness + simplicity for typed round-trip (`json.RawMessage` or codec registry), and accept that YAML may be lossy/ugly.
- **If YAML is a developer tooling contract (llm-runner/fixtures)**:
  - Keep YAML output readable either by:
    - staying with `any` + optional per-key registry, or
    - adopting `json.RawMessage` + explicit YAML bridge.

## Appendix: concrete usage sites (high signal)

- `geppetto/pkg/inference/fixtures/fixtures.go`
  - loads: `serde.FromYAML`
  - writes: `serde.SaveTurnYAML` to `input_turn.yaml`, `final_turn.yaml`, `final_turn_N.yaml`
- `geppetto/cmd/llm-runner/main.go`
  - writes `input_turn.yaml` via `serde.SaveTurnYAML`
- `geppetto/cmd/llm-runner/api.go`
  - reads/parses `input_turn.yaml` and `final_turn*.yaml` via `serde.FromYAML`
