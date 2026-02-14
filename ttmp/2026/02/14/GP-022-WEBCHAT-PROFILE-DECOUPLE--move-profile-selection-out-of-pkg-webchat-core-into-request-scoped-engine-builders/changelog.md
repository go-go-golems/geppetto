# Changelog

## 2026-02-14

- Initial workspace created


## 2026-02-14 - Created profile decoupling design

Added a detailed design and migration analysis for removing profile semantics from pkg/webchat core and moving profile selection into app-owned request builders for web-chat and web-agent-example.

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/geppetto/ttmp/2026/02/14/GP-022-WEBCHAT-PROFILE-DECOUPLE--move-profile-selection-out-of-pkg-webchat-core-into-request-scoped-engine-builders/design-doc/01-profile-decoupling-analysis-and-migration-plan.md — Primary design document with API delta
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/geppetto/ttmp/2026/02/14/GP-022-WEBCHAT-PROFILE-DECOUPLE--move-profile-selection-out-of-pkg-webchat-core-into-request-scoped-engine-builders/tasks.md — Implementation task checklist derived from the design


## 2026-02-14 - Expanded resolver-plan architecture section

Added a 2+ page deep-dive section explaining the architectural evolution from BuildConfig/BuildFromConfig toward an app-owned ConversationRequestPlan resolver model, including request-data split, detailed pseudocode, migration strategy, and ASCII timeline diagram.

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/geppetto/ttmp/2026/02/14/GP-022-WEBCHAT-PROFILE-DECOUPLE--move-profile-selection-out-of-pkg-webchat-core-into-request-scoped-engine-builders/design-doc/01-profile-decoupling-analysis-and-migration-plan.md — Added detailed design evolution section with concrete API proposal and timeline diagram


## 2026-02-14 - Added concrete app migration map

Expanded the design doc with a concrete migration map for pinocchio/cmd/web-chat and web-agent-example, including file-level backend/frontend deltas, sequencing, and per-app acceptance criteria for the resolver-plan architecture.

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/geppetto/ttmp/2026/02/14/GP-022-WEBCHAT-PROFILE-DECOUPLE--move-profile-selection-out-of-pkg-webchat-core-into-request-scoped-engine-builders/design-doc/01-profile-decoupling-analysis-and-migration-plan.md — Added app migration map section with concrete change list


## 2026-02-14 - Added clean-cutover policy

Added an explicit decision section requiring a clean cutover to the ConversationRequestResolver/ConversationRequestPlan interface and retiring BuildEngineFromReq/WithEngineFromReqBuilder after app migration.

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/geppetto/ttmp/2026/02/14/GP-022-WEBCHAT-PROFILE-DECOUPLE--move-profile-selection-out-of-pkg-webchat-core-into-request-scoped-engine-builders/design-doc/01-profile-decoupling-analysis-and-migration-plan.md — Added explicit clean-cutover and old-interface retirement policy


## 2026-02-14 - Execution setup + task decomposition

Converted ticket tasks into a phased implementation checklist and initialized diary tracking for slice-by-slice execution, testing, and commit bookkeeping.

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/geppetto/ttmp/2026/02/14/GP-022-WEBCHAT-PROFILE-DECOUPLE--move-profile-selection-out-of-pkg-webchat-core-into-request-scoped-engine-builders/reference/01-diary.md — Initialized implementation diary with Step 1
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/geppetto/ttmp/2026/02/14/GP-022-WEBCHAT-PROFILE-DECOUPLE--move-profile-selection-out-of-pkg-webchat-core-into-request-scoped-engine-builders/tasks.md — Expanded phased implementation checklist


## 2026-02-14 - Slice 1: resolver API cutover

Implemented first code slice: replaced BuildEngineFromReq with ConversationRequestResolver.Resolve plan API in core webchat handlers and migrated web-agent-example call sites. Core webchat tests pass; web-agent-example full module test still blocked by baseline missing module deps.

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/webchat/engine_from_req.go — Replaced old builder interface with resolver-plan types and default resolver
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/webchat/router.go — Chat/WS handlers now consume request plans
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/webchat/router_options.go — New WithConversationRequestResolver router option
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/web-agent-example/cmd/web-agent-example/engine_from_req.go — Migrated custom resolver implementation
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/web-agent-example/cmd/web-agent-example/main.go — Switched option wiring to new resolver API

