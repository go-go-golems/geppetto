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


## 2026-02-14 - Slice 2: remove core profile endpoints and cookie policy

Removed core /api/chat/profile* endpoints and cookie-based resolver behavior; default resolver now uses generic runtime key semantics and debug API returns runtime_key. Updated tests accordingly.

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/webchat/engine_from_req.go — Removed profile registry/cookie resolver dependencies and switched to runtime key query
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/webchat/router.go — Removed core profile endpoints and updated WS logging/request comments
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/webchat/router_debug_api_test.go — Updated debug API assertions for runtime_key
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/webchat/router_debug_routes.go — Renamed debug payload field to runtime_key
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/web-agent-example/cmd/web-agent-example/engine_from_req.go — Simplified default runtime-key resolver path


## 2026-02-14 - Slice 3: app-owned profile policy in cmd/web-chat

Completed removal of profile registry/types from `pkg/webchat` core and moved profile policy into `cmd/web-chat` via app-owned resolver and handlers.

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/profile_policy.go — App-local profile registry, resolver, override policy, and profile endpoints
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/main.go — Router now wired with app resolver and handler registration
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/webchat/types.go — Removed core Profile/ProfileRegistry types
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/webchat/router_options.go — Removed WithProfileRegistry option
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/webchat/router.go — Removed AddProfile path in core
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/web-agent-example/cmd/web-agent-example/main.go — Removed stale AddProfile call


## 2026-02-14 - Slice 4: runtime-key naming + signature-only rebuild

Renamed remaining core runtime identity fields from profile-oriented names, switched rebuild checks to signature-only comparison, and updated debug UI mapping for `runtime_key`.

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/webchat/conversation.go — RuntimeKey/EngineConfigSignature fields and signature-only rebuild logic
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/webchat/engine_config.go — `runtime_key` in config/signature payload
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/webchat/send_queue.go — Queued request runtime key naming
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/webchat/router_debug_routes.go — Debug conversation payloads read runtime key from conversation state
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/web/src/debug-ui/api/debugApi.ts — Frontend mapping updated for backend `runtime_key`
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/web/src/debug-ui/mocks/msw/createDebugHandlers.ts — Mock payloads aligned with `runtime_key`


## 2026-02-14 - Slice 5: app-owned profile policy tests

Added dedicated tests in `cmd/web-chat` for profile resolver behavior and profile cookie API handlers, closing a migration validation gap after moving profile policy out of core.

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/profile_policy_test.go — New resolver + handler tests for default profile, override policy, and cookie get/set flows


## 2026-02-14 - Slice 6: remove legacy builder naming in web-agent-example

Renamed remaining resolver implementation symbols in `web-agent-example` away from old builder terminology and confirmed no runtime dependency on `/api/chat/profile*` endpoints.

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/web-agent-example/cmd/web-agent-example/engine_from_req.go — Resolver type/constructor renamed to request-resolver terminology
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/web-agent-example/cmd/web-agent-example/main.go — Updated resolver constructor call site


## 2026-02-14 - Slice 7: docs cutover to resolver-plan model

Updated core webchat docs to remove legacy builder/profile-registry guidance and document app-owned runtime/profile policy through `ConversationRequestResolver`.

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/doc/topics/webchat-framework-guide.md — Updated architecture and API examples for runtime-key resolver model
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/doc/topics/webchat-user-guide.md — Updated minimal wiring and customization guidance for app-owned policy
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/doc/tutorials/03-thirdparty-webchat-playbook.md — Removed `WithEngineFromReqBuilder` path and replaced with resolver-plan examples


## 2026-02-14 - Slice 8: ws hello runtime_key protobuf rename

Renamed WS hello protobuf field from `profile` to `runtime_key`, regenerated Go/TS code, and updated backend WS hello emission.

### Related Files

- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/proto/sem/base/ws.proto — `WsHelloV1.runtime_key`
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/sem/pb/proto/sem/base/ws.pb.go — Regenerated Go protobuf bindings
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/web/src/sem/pb/proto/sem/base/ws_pb.ts — Regenerated TS protobuf bindings
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/web/src/sem/pb/proto/sem/base/ws_pb.ts — Regenerated TS protobuf bindings
- /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/webchat/router.go — Updated hello frame construction to `RuntimeKey`
