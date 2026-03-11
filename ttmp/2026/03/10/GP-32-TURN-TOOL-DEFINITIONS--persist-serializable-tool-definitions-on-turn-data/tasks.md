# Tasks

## TODO

- [x] Task 1: add a typed turn-data key for serializable tool definitions
  - Scope: `pkg/spec/geppetto_codegen.yaml`, generated key outputs, and any owning type definitions.
  - Acceptance criteria: Geppetto exposes a first-class typed key for `tool_definitions` on `Turn.Data`, with generated Go/TS/JS constants updated.

- [x] Task 2: define the canonical serializable tool-definition payload shape
  - Scope: `pkg/inference/engine` and/or `pkg/inference/tools`.
  - Acceptance criteria: there is one explicit persisted representation for tool advertisement fields (`name`, `description`, `parameters`, optional `examples/tags/version`) with runtime executor fields excluded or zeroed.

- [ ] Task 3: stamp tool definitions onto the turn inside the tool loop
  - Scope: `pkg/inference/toolloop/loop.go` and any helper used to map runtime registry entries into the persisted representation.
  - Acceptance criteria: turns that run with a registry also receive a serialized definitions snapshot on `Turn.Data` before the first inference call.

- [ ] Task 4: add serde and persistence regression coverage
  - Scope: `pkg/turns/serde/*`, tool loop tests, and focused turn-data round-trip tests.
  - Acceptance criteria: YAML/JSON round-trip preserves `tool_definitions`, and tool loop tests verify that persisted definitions are stamped alongside `tool_config`.

- [ ] Task 5: preserve provider advertisement from the live runtime registry
  - Scope: `pkg/steps/ai/openai/*`, `pkg/steps/ai/openai_responses/*`, `pkg/steps/ai/claude/*`, `pkg/steps/ai/gemini/*`.
  - Acceptance criteria: the implementation does not change engines to advertise from persisted `Turn.Data.tool_definitions`; the new turn key is documented and tested as informational/inspection data only.

- [ ] Task 6: preserve runtime execution via context-carried registry only
  - Scope: `pkg/inference/tools/context.go`, `pkg/inference/toolloop/loop.go`, execution paths.
  - Acceptance criteria: tool execution still depends on the live registry in context; persisted definitions remain inspection-only metadata.

- [ ] Task 7: update docs and JS codec surface for the new turn key
  - Scope: `pkg/js/modules/geppetto/codec.go`, generated docs/types, and relevant tutorials/topics.
  - Acceptance criteria: JS callers and docs can refer to the new turn-data key by name, and the turns/tools docs explain that persisted definitions are for inspection while the runtime registry remains authoritative for provider advertisement and execution.

## Working Order

- [x] Complete Tasks 1 and 2 in one commit because the new key depends on the persisted payload type.
- [ ] Complete Tasks 3 and 4 in one commit because the stamping path needs direct regression coverage.
- [ ] Complete Tasks 5 and 6 in one commit because they are authority-boundary non-regression tests.
- [ ] Complete Task 7 in a final cleanup/docs commit once the runtime behavior is locked.
