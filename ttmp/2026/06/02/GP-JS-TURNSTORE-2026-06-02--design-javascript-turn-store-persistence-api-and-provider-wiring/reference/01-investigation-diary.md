---
title: Investigation diary
doc_type: reference
status: active
intent: long-term
topics:
  - geppetto
  - goja
  - js-bindings
  - persistence
  - turns
owners:
  - manuel
created: 2026-06-02
updated: 2026-06-02
---

# Diary

## Goal

Record the investigation and design work for Geppetto/xgoja-facing turn-store integration.

## Step 1: Map the existing turn persistence seams

I investigated the existing Geppetto JS module, enginebuilder persistence hook, Pinocchio CLI persistence settings, and Pinocchio SQLite turn-store implementation. The main finding is that Geppetto already has the correct final-turn persistence seam, while Pinocchio already has the DSN-backed concrete store.

I then wrote an intern-oriented design guide that keeps concrete storage host-owned, adds wrapper-first JS store APIs, and uses provider/module configuration to make Pinocchio-style `--turns-dsn` storage easy to install as a default persister.

### Prompt Context

**User prompt (verbatim):** "Create 2 new tickets, one for each storage-related concern. The user specifically wants:
- A Geppetto/xgoja-facing turn-store integration that makes Pinocchio’s `--turns-dsn`-style storage easy to wire by default through module/provider configuration, possibly gated by an “enable storage” flag, and also exposed via API.
- A second related ticket, likely for the other storage/timeline concern, while noting the user believes timeline is probably not a Geppetto concern.
- For each ticket, create a detailed intern-oriented analysis/design/implementation guide with prose, bullets, pseudocode, diagrams, API references, and file references.
- Store the guide in the ticket and upload it to reMarkable."

**Assistant interpretation:** Create two docmgr tickets and write detailed design/implementation guides; this ticket covers Geppetto/xgoja turn-store persistence.

**Inferred user intent:** Make the next implementation step clear enough for an intern while preserving the Geppetto/Pinocchio ownership boundary.

**Commit (code):** N/A — documentation/design only.

### What I did
- Created docmgr ticket `GP-JS-TURNSTORE-2026-06-02`.
- Added design document `design-doc/01-javascript-turn-store-persistence-design-and-implementation-guide.md`.
- Read and cited Geppetto JS module, agent, session, and enginebuilder files.
- Read and cited Pinocchio CLI persistence settings and SQLite turn-store files.
- Proposed provider config, Go interfaces, JS wrapper APIs, implementation phases, and tests.

### Why
- The repository already has persistence hooks, but no JS-facing turn-store API or xgoja provider storage configuration contract.
- The design needs to make Pinocchio's DSN-backed turn storage easy to wire without making Geppetto import Pinocchio.

### What worked
- `Options.DefaultPersister` and `enginebuilder.WithPersister(...)` form a clean existing write path.
- Pinocchio's `chatstore.TurnStore` gives a concrete model for read/write operations and snapshot shape.
- The xgoja provider `HostServices` seam is already present and can mediate storage capabilities.

### What didn't work
- N/A. No commands failed for this ticket after creation.

### What I learned
- Geppetto's runner persists only on successful runs and treats persistence as best effort.
- Pinocchio currently maps session-like values into turn-store conversation IDs for resume behavior.
- The cleanest design is a Geppetto capability interface with host-owned concrete storage.

### What was tricky to build
- The main design tension is dependency direction: Geppetto needs to expose turn persistence, but importing Pinocchio's concrete SQLite store would make the lower-level inference module depend on the application CLI layer.
- I resolved this by proposing minimal Geppetto turn-store wrappers and optional provider host storage services.

### What warrants a second pair of eyes
- The exact JS query naming (`convId` vs `sessionId`) should be reviewed against Pinocchio resume semantics.
- The decision to keep persistence failures best-effort should be reviewed if JS callers need strict durability guarantees.

### What should be done in the future
- Implement the proposed `gp.turnStores` namespace and `agent.persistTo(...)` / `agent.persistDefault(...)` methods.
- Add provider config tests for `enableStorage` and host capability errors.
- Add a Pinocchio host adapter for `--turns-dsn` / `--turns-db` style config.

### Code review instructions
- Start with `geppetto/pkg/js/modules/geppetto/module.go`, `api_agent.go`, `api_sessions.go`, and `pkg/inference/toolloop/enginebuilder/builder.go`.
- Validate with future tests under `geppetto/pkg/js/modules/geppetto` and provider tests under `geppetto/pkg/js/modules/geppetto/provider`.

### Technical details
- Existing Geppetto seam: `enginebuilder.TurnPersister` and `Options.DefaultPersister`.
- Existing Pinocchio store: `pinocchio/pkg/persistence/chatstore/turn_store.go` and `turn_store_sqlite.go`.
- Existing CLI flags: `--turns-dsn` and `--turns-db` in `pinocchio/pkg/cmds/cmdlayers/helpers.go`.
