# Changelog

## 2026-02-12

- Initial workspace created.
- Added analysis doc: `analysis/01-migration-analysis-old-glazed-to-facade-packages-geppetto-then-pinocchio.md`.
- Added diary doc: `reference/01-diary.md`.
- Captured baseline validation logs and migration inventories under `sources/local/`:
  - legacy imports/symbols/tags/signatures
  - `make test` + `make lint` outputs for geppetto and pinocchio
  - failure extracts and per-repo count breakdowns
- Documented ordered implementation strategy: geppetto migration first, pinocchio second.

## 2026-02-12

Completed baseline migration analysis: validated glazed facade APIs, captured geppetto/pinocchio test+lint failures, generated exhaustive file/symbol inventories, and documented phased plan (geppetto first, then pinocchio).

### Related Files

- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/analysis/01-migration-analysis-old-glazed-to-facade-packages-geppetto-then-pinocchio.md — Primary migration analysis deliverable
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/reference/01-diary.md — Detailed implementation diary
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/14-failure-extracts.txt — Baseline failure evidence for planning

## 2026-02-12

Completed Pinocchio Phase 2 Task 1 by migrating `pkg/cmds/*` core command model/loader flow to `schema/fields/sources/values` in pinocchio commit `acd8533`, then recorded focused validation and blockers in ticket artifacts.

### Related Files

- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/pinocchio/pkg/cmds/cmd.go — values-based command runtime and default variable extraction
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/pinocchio/pkg/cmds/helpers/parse-helpers.go — source middleware migration for profile/config/env/default parsing
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/23-pinocchio-pkg-cmds-focused-pass.txt — focused package test evidence
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/24-pinocchio-pkg-cmds-helpers-blocker.txt — current missing-geppetto import blocker

## 2026-02-12

Completed Pinocchio Phase 2 Task 2 by migrating command implementation files under `cmd/pinocchio/cmds/*` to facade APIs (`fields`, `values`, `sources`) in pinocchio commit `826ba63`, including a local `pkg/filefilter` shim to decouple `catter` from `clay`'s legacy layer API.

### Related Files

- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/pinocchio/cmd/pinocchio/cmds/openai/openai.go — command section wiring and values-based decode migration
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/pinocchio/cmd/pinocchio/cmds/catter/cmds/print.go — catter command migration plus filefilter integration via local package
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/pinocchio/pkg/filefilter/section.go — facade-compatible section definition replacing layer-based parser dependency
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/26-pinocchio-cmd-impl-legacy-scan-after-task2.txt — post-migration local legacy-symbol scan
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/28-pinocchio-cmd-impl-aggregate-blockers.txt — aggregate command package blockers (`conversation`, `prompto`)

## 2026-02-12

Completed Pinocchio Phase 2 Task 3 by migrating webchat + redis settings decode paths to facade APIs in pinocchio commit `bc94338`.

### Related Files

- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/pinocchio/pkg/redisstream/redis_layer.go — redis section migrated from layer/parameter definitions to `schema.WithFields`
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/pinocchio/pkg/webchat/router.go — router settings and redis settings decode switched to `values.DecodeSectionInto`
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/pinocchio/cmd/web-chat/main.go — command flags migrated to `fields.New` and `WithSections`
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/29-pinocchio-webchat-redis-legacy-scan-after-task3.txt — post-migration scope scan
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/31-pinocchio-webchat-focused-blocker.txt — remaining `toolhelpers` blocker in `pkg/webchat`

## 2026-02-12

Completed Pinocchio Phase 2 Task 4 by migrating `cmd/examples/*` and `cmd/agents/simple-chat-agent/main.go` to facade command APIs in pinocchio commit `b349349`.

### Related Files

- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/pinocchio/cmd/examples/simple-chat/main.go — values-based command signature/decode and section wiring
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/pinocchio/cmd/examples/simple-redis-streaming-inference/main.go — facade migration plus sink wiring update via `events.WithEventSinks`
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/pinocchio/cmd/agents/simple-chat-agent/main.go — values/fields migration for agent command flags and decode
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/33-pinocchio-examples-agents-legacy-scan-after-task4.txt — post-migration scope scan
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/35-pinocchio-simple-redis-blocker.txt — current clay logging bootstrap blocker

## 2026-02-12

Completed post-Task-4 migration closure and full validation:

- pinocchio runtime migration in commit `9909af2`:
  - moved `toolhelpers` usage to `toolloop`
  - moved `toolcontext` usage to `tools.WithRegistry`/`tools.RegistryFrom`
  - migrated `RunID` references to turn metadata (`session_id`/`inference_id`) and `events.EventMetadata` (`SessionID`/`InferenceID`)
  - removed `engine.WithSink` usage by attaching sinks through `events.WithEventSinks`
- temporary prompto compatibility shim in commit `6c29260`
- prompto removed entirely in commit `c64e891`:
  - removed prompto command wiring and package
  - removed `github.com/go-go-golems/prompto` dependency and replace directive
  - deleted temporary vendored prompto tree under `third_party/prompto`
- geppetto analyzer script staticcheck fixes (QF1012) restored `make lint` pass.

Validation logs captured:

- `sources/local/47-geppetto-make-test-continue.txt`
- `sources/local/48-geppetto-make-lint-continue.txt`
- `sources/local/49-pinocchio-make-test-continue.txt`
- `sources/local/50-pinocchio-make-lint-continue.txt`
- `sources/local/51-pinocchio-make-test-after-porting-batch1.txt`
- `sources/local/52-pinocchio-make-lint-after-porting-batch1.txt`
- `sources/local/53-pinocchio-make-test-after-prompto-local-replace.txt`
- `sources/local/54-pinocchio-make-test-after-prompto-patch.txt`
- `sources/local/55-pinocchio-make-test-after-prompto-fix2.txt`
- `sources/local/56-pinocchio-make-lint-after-prompto-fix2.txt`
- `sources/local/57-geppetto-make-test-after-analyzer-lint-fix.txt`
- `sources/local/58-geppetto-make-lint-after-analyzer-lint-fix.txt`
- `sources/local/59-pinocchio-make-test-after-removing-prompto.txt`
- `sources/local/60-pinocchio-make-lint-after-removing-prompto.txt`

### Related Files

- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/pinocchio/pkg/geppettocompat/compat.go — compatibility helpers for middleware chaining and turn/session metadata IDs
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/pinocchio/pkg/webchat/loops.go — toolloop migration replacing removed toolhelpers API
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/pinocchio/pkg/middlewares/agentmode/middleware.go — event metadata and run/session migration
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/pinocchio/cmd/pinocchio/main.go — prompto command removal
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/scripts/glazed_migration_analyzer.go — staticcheck cleanup to restore geppetto lint pass
