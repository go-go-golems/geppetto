---
Title: Diary
Ticket: PI-001-WEBCHAT-ENGINEBUILDER
Status: active
Topics:
    - pinocchio
    - webchat
    - refactor
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-22T13:11:53.300727517-05:00
WhatFor: ""
WhenToUse: ""
---

# Diary

## Goal

Track the investigation and design work for refactoring Pinocchio’s webchat `getOrCreateConv`
to stop using ad-hoc `build()` closures and instead use a reusable EngineBuilder-style pattern
inspired by go-go-mento (and compatible with Moments’ more complex needs), without writing code yet.

## Step 1: Create Ticket, Seed Docs, And Gather References

This step created the PI-001 ticket workspace and established the documentation scaffolding for
an in-depth analysis. The immediate goal was to “freeze” the problem statement and identify the
most load-bearing files in Pinocchio, go-go-mento, and Moments before proposing a refactor plan.

The main outcome is a new analysis document with a curated set of related source files so that
future steps (including eventual implementation) can proceed systematically and review can focus
on a small number of core artifacts.

### What I did
- Created the docmgr ticket `PI-001-WEBCHAT-ENGINEBUILDER`.
- Added an analysis doc and a diary doc for the ticket.
- Related the most relevant Pinocchio/go-go-mento/Moments/Geppetto files to the analysis doc.

### Why
- The current Pinocchio webchat creates engines/sinks/subscribers via per-callsite closures; this is
  hard to reason about, hard to reuse, and diverges from the stronger “builder/manager” patterns we
  already have in go-go-mento (and partially in Moments).
- Establishing a ticket + doc trail early keeps the refactor design reviewable and makes later
  implementation less error-prone.

### What worked
- `docmgr ticket create-ticket` created the workspace under `geppetto/ttmp/2026/01/22/`.
- `docmgr doc add` produced the analysis + diary docs with the expected frontmatter.
- `docmgr doc relate` successfully attached the intended cross-repo file references.

### What didn't work
- N/A (analysis-only step; no code changes and no tests required).

### What I learned
- The “EngineBuilder” pattern in go-go-mento is explicitly designed to keep Router handlers lean by
  centralizing engine + sink composition, and it cleanly separates “subscriber creation” as another
  dependency (SubscriberFactory). That split looks directly applicable to Pinocchio.
- Pinocchio already has a small `ParsedLayersEngineBuilder` abstraction (`pinocchio/pkg/inference/enginebuilder/parsed_layers.go`),
  but it currently doesn’t cover subscriber creation or sink wrapping and therefore can’t replace the
  `build()` closures in webchat yet.

### What was tricky to build
- N/A (no implementation in this step), but the main “tricky” aspect is scoping: we need to design a
  builder API that is reusable across Pinocchio and later Moments without prematurely importing all of
  Moments’ complexity (profiles, sink pipelines, persistence, step controller).

### What warrants a second pair of eyes
- Ensure the ticket scope is correct: PI-001 should focus on refactoring Pinocchio’s webchat builder
  plumbing (removing closures) and not accidentally turn into the broader “MO-001 port moments webchat”
  consolidation work.

### What should be done in the future
- Continue with a deep comparative analysis of:
  - Pinocchio’s current `getOrCreateConv` call graph and responsibilities.
  - go-go-mento’s `EngineBuilder` + `ConversationManager` responsibilities split.
  - Moments’ current build closure patterns (where it got more complex and why).

### Code review instructions
- Start at `geppetto/ttmp/2026/01/22/PI-001-WEBCHAT-ENGINEBUILDER--pinocchio-webchat-refactor-getorcreateconv-to-enginebuilder-pattern/analysis/01-simplify-getorcreateconv-via-enginebuilder-pinocchio-webchat.md`.
- Validate that `RelatedFiles` includes the intended core sources (Pinocchio webchat, go-go-mento builder/manager, Moments router, Geppetto session builder).

### Technical details
- Ticket created: `PI-001-WEBCHAT-ENGINEBUILDER`
- Docs created:
  - `geppetto/ttmp/2026/01/22/PI-001-WEBCHAT-ENGINEBUILDER--pinocchio-webchat-refactor-getorcreateconv-to-enginebuilder-pattern/analysis/01-simplify-getorcreateconv-via-enginebuilder-pinocchio-webchat.md`
  - `geppetto/ttmp/2026/01/22/PI-001-WEBCHAT-ENGINEBUILDER--pinocchio-webchat-refactor-getorcreateconv-to-enginebuilder-pattern/reference/01-diary.md`

## Step 2: Map Pinocchio’s Current getOrCreateConv Responsibilities

This step closely read Pinocchio’s current webchat router and conversation lifecycle to understand
exactly what the `build()` closures are responsible for today. The key outcome is a sharper problem
statement: `getOrCreateConv` is only a “create-once cache”, while the callsites’ closures contain
policy (profiles/overrides) and transport wiring (subscriber setup) that should be centralized.

This is important because Moments (and go-go-mento before it) already solved this class of problem
by separating “engine/sink composition” from “subscriber creation” and then letting a manager decide
when to rebuild (based on deterministic config signatures). Pinocchio is currently missing that
abstraction boundary.

### What I did
- Read Pinocchio webchat lifecycle:
  - `pinocchio/pkg/webchat/router.go` (WS join + /chat handlers).
  - `pinocchio/pkg/webchat/conversation.go` (`getOrCreateConv`, connection pool, stream startup).
  - `pinocchio/pkg/webchat/engine.go` (engine composition policy; middleware order).
- Identified all `build := func() (engine.Engine, *middleware.WatermillSink, message.Subscriber, error)` callsites.
- Noted exactly which responsibilities are duplicated and which are “policy decisions”.

### Why
- We need to remove `build()` closures from callsites without losing configurability (profiles, overrides).
- We need a design that can later be reused by Moments, where the same pattern exists but with more layers
  (sink pipelines, profile resolution, persistence, step controller).

### What worked
- The Pinocchio webchat code is compact and readable enough that responsibilities can be listed explicitly:
  - transport plumbing (Redis group/subscriber creation),
  - sink creation (Watermill sink bound to topic),
  - settings extraction (StepSettings from ParsedLayers),
  - policy application (profile defaults + request overrides),
  - engine composition (system prompt + middleware uses),
  - session creation (ToolLoopEngineBuilder + seed turn).

### What didn't work
- N/A (analysis-only).

### What I learned
- `getOrCreateConv` currently ignores profile/override changes once a conversation exists. That means
  a “WS join” can silently lock in engine settings, and later `/chat` requests with overrides will
  reuse the existing engine. This is the exact class of issue that go-go-mento’s `ConversationManager`
  signature check avoids.
- The current closure type forces `*middleware.WatermillSink` (not `events.EventSink`), which makes it
  harder to introduce sink wrapper pipelines without touching core types. go-go-mento returns
  `events.EventSink` and treats WatermillSink as just the base sink.

### What was tricky to build
- The biggest “sharp edge” is disentangling responsibilities without changing behavior:
  - WS join wants “default profile engine” even if no run is started yet.
  - /chat wants to apply overrides (system prompt, middleware list) for engine composition.
  - Both paths want consistent subscriber creation behavior under Redis vs in-memory transport.

### What warrants a second pair of eyes
- The conclusion that Pinocchio needs signature-based rebuild semantics (not just “create-once”) is a
  functional behavior question: if we introduce rebuilds, we must ensure this doesn’t break UI/client
  assumptions about session stability.

### What should be done in the future
- Compare Pinocchio’s closure responsibilities against go-go-mento and Moments, and propose a minimal
  “common denominator” interface set (EngineBuilder + SubscriberFactory + optional SinkBuilder).

### Code review instructions
- Read these files in order:
  - `pinocchio/pkg/webchat/router.go` (find the three `build := func() ...` blocks).
  - `pinocchio/pkg/webchat/conversation.go` (see how `buildEng()` is only used on create).
  - `pinocchio/pkg/webchat/engine.go` (policy for middleware application order).

### Technical details
- Pinocchio closure signature today:
  - `func() (engine.Engine, *middleware.WatermillSink, message.Subscriber, error)`
- Pinocchio conversation cache semantics:
  - “create once per conv_id; return existing conversation thereafter”

## Step 3: Compare Against go-go-mento and Moments (Builder + Manager Patterns)

This step examined the more mature patterns in go-go-mento and the “evolved but messy” reality in
Moments. The goal was to extract a reusable pattern that Pinocchio can adopt now (to remove the
closures) while staying compatible with Moments’ needs later (sink pipelines, identity/session
refresh, step controller, persistence/hydration).

The key conclusion is that go-go-mento already provides the right architectural split:
Engine composition is centralized (EngineBuilder + EngineConfig signatures) while subscriber creation
is a separate dependency (SubscriberFactory). Moments partially re-implemented this, but still wires
composition via ad-hoc closures and uses a weak “signature” check.

### What I did
- Read go-go-mento’s engine composition pipeline:
  - `go-go-mento/go/pkg/webchat/engine_builder.go`
  - `go-go-mento/go/pkg/webchat/engine_config.go`
- Read go-go-mento’s lifecycle orchestration:
  - `go-go-mento/go/pkg/webchat/conversation_manager.go`
- Read Moments’ current (messier) version of the same idea:
  - `moments/backend/pkg/webchat/router.go` (inline `build := func() ...` closures)
  - `moments/backend/pkg/webchat/conversation.go` (`getOrCreateConv` rebuild-on-profile change)
- Re-read Pinocchio’s existing `ParsedLayersEngineBuilder`:
  - `pinocchio/pkg/inference/enginebuilder/parsed_layers.go`

### Why
- Pinocchio should adopt the “good parts” (centralized build logic + signature-based rebuild) instead
  of expanding the per-callsite closure approach.
- If Pinocchio’s pattern is close enough to go-go-mento’s, Moments can later reuse it (or migrate
  gradually) instead of remaining a parallel, bespoke implementation.

### What worked
- go-go-mento’s design is explicit and debuggable:
  - `EngineConfig` is JSON-serializable and its `Signature()` is intentionally the JSON string (not a hash).
  - `ConversationManager.GetOrCreate` compares signatures and rebuilds engine/sink/subscriber when needed.
  - Subscriber creation is pluggable via `SubscriberFactory`, keeping the engine builder transport-agnostic.

### What didn't work
- Moments’ current rebuild signature is too weak:
  - `newSig := profileSlug + engineBuildSignatureSuffix` ignores most composition inputs and doesn’t
    directly encode overrides. It helps force recomposition in some scenarios, but it isn’t a principled
    “config signature” like go-go-mento’s.
- Moments still relies on ad-hoc closures at callsites (despite having logic inside `getOrCreateConv`
  to rebuild on signature changes).

### What I learned
- The “right” unit of reuse is not “a closure that returns (engine,sink,subscriber)”; it’s a small set of
  composable interfaces:
  - `EngineBuilder` (engine + sink composition, driven by a serializable config)
  - `SubscriberFactory` (transport-specific subscription strategy)
  - optionally `SinkBuilder` (Moments-style sink pipelines over a base Watermill sink)
- Pinocchio’s current `ParsedLayersEngineBuilder` is too narrow to replace the webchat closures because:
  - it doesn’t build a sink per conversation/topic,
  - it doesn’t support config signatures/rebuild decisions,
  - it doesn’t help with subscriber creation.

### What was tricky to build
- Reconciling terminology and scope:
  - go-go-mento’s “EngineBuilder” builds `(engine, eventsink)` and returns `EngineConfig`.
  - Geppetto’s `session.EngineBuilder` builds a runner for a session.
  - Pinocchio currently uses “builder” to mean “build engine + sink + subscriber closure”.
  We need to choose names/interfaces in Pinocchio that are unambiguous and align with existing usage.

### What warrants a second pair of eyes
- Whether Pinocchio should copy go-go-mento’s exact API surface (BuildConfig/BuildFromConfig) or adapt it:
  - Copying makes future Moments migration easier.
  - Adapting might better align with Geppetto’s `session.EngineBuilder` but risks divergence.

### What should be done in the future
- Write the PI-001 analysis doc with:
  - A proposed interface set for Pinocchio (mirroring go-go-mento where possible).
  - A migration plan that removes closures from `pinocchio/pkg/webchat/router.go`.
  - A “Moments adoption plan” describing how Moments could eventually swap its closures for the same builder/manager.

### Code review instructions
- Compare these implementations directly:
  - go-go-mento: `go-go-mento/go/pkg/webchat/engine_builder.go`
  - go-go-mento: `go-go-mento/go/pkg/webchat/conversation_manager.go`
  - moments: `moments/backend/pkg/webchat/conversation.go`
  - pinocchio: `pinocchio/pkg/webchat/router.go` and `pinocchio/pkg/webchat/conversation.go`

### Technical details
- go-go-mento: `EngineConfig.Signature()` returns JSON (human-debuggable).
- moments: signature is effectively `profileSlug + const` today (not input-complete).

## Step 4: Write The Refactor Analysis (Proposed To-Be Design)

This step distilled the findings from Pinocchio/go-go-mento/Moments into a concrete “To-Be” design:
Pinocchio should adopt a go-go-mento-style `EngineBuilder` and `EngineConfig` signature mechanism,
and move “get-or-create + rebuild-on-signature” into a manager layer. The analysis doc is intended
to be implementation-ready while still being reusable as a design reference for Moments migration.

The main deliverable is the PI-001 analysis doc with explicit responsibilities, diagrams, interfaces,
and a migration checklist.

### What I did
- Wrote the analysis doc:
  - `geppetto/ttmp/2026/01/22/PI-001-WEBCHAT-ENGINEBUILDER--pinocchio-webchat-refactor-getorcreateconv-to-enginebuilder-pattern/analysis/01-simplify-getorcreateconv-via-enginebuilder-pinocchio-webchat.md`
- Updated the ticket task list to reflect the proposed implementation checklist.

### Why
- We need a shared “north star” doc before writing code so that:
  - Pinocchio’s refactor doesn’t drift into unrelated webchat changes,
  - and Moments can later adopt the same pattern without yet another re-implementation.

### What worked
- The doc now captures:
  - the “as-is” responsibilities split (closures vs cache),
  - the “to-be” split (EngineBuilder + SubscriberFactory + ConversationManager),
  - and the rationale for storing `events.EventSink` instead of `*WatermillSink` on the conversation.

### What didn't work
- N/A (analysis-only; no implementation yet).

### What I learned
- The minimum reusable pattern for Pinocchio↔Moments is interface-level:
  - adopt the same conceptual split as go-go-mento (builder + manager + subscriber factory),
  - and keep Moments-only complexity in the manager layer, not inside engine composition.

### What was tricky to build
- The main tricky part is naming and compatibility:
  - “EngineBuilder” means different things in Geppetto (runner builder) vs go-go-mento (engine+sinks).
  The doc deliberately mirrors go-go-mento’s API shape for easier migration, while calling out that
  Geppetto’s session builder is still a separate concept.

### What warrants a second pair of eyes
- Confirm that the recommended change “store `events.EventSink` in Conversation” is acceptable in
  Pinocchio and doesn’t break any assumptions in the current webchat run loop wiring.

### What should be done in the future
- Upload the analysis doc (and diary) to reMarkable for review.
- Only after review, start implementing PI-001 tasks in small, testable slices.

### Code review instructions
- Read the analysis doc top-to-bottom and sanity check:
  - the identified Pinocchio bug/behavior (create-once ignores overrides),
  - the proposed interface set,
  - and the migration plan sequencing.

### Technical details
- Analysis doc path:
  - `geppetto/ttmp/2026/01/22/PI-001-WEBCHAT-ENGINEBUILDER--pinocchio-webchat-refactor-getorcreateconv-to-enginebuilder-pattern/analysis/01-simplify-getorcreateconv-via-enginebuilder-pinocchio-webchat.md`

## Step 5: Prepare reMarkable Upload (And Capture The Failure Mode)

This step attempted to upload the new ticket documents to reMarkable using the standard `remarquee`
bundle workflow. The upload failed due to DNS/network resolution in the current environment, so I
switched to generating a local PDF via `--pdf-only` as a fallback artifact that can be uploaded from
an environment with working network access.

This is still valuable: reviewers can read the exact PDF that would have been uploaded, and the
ticket retains a stable “shareable” artifact under `various/`.

### What I did
- Ran a dry-run bundle upload to confirm inputs and ToC order.
- Attempted the real upload and captured the exact error output.
- Generated a PDF-only bundle output in the ticket folder.

### Why
- The user requested uploading the analysis to reMarkable; we should at least produce the exact PDF
  artifact, even if the network upload must be performed elsewhere.

### What worked
- `remarquee upload bundle --dry-run ...` confirmed the bundle inputs and destination folder.
- `remarquee upload bundle --pdf-only ...` successfully generated a PDF at:
  - `geppetto/ttmp/2026/01/22/PI-001-WEBCHAT-ENGINEBUILDER--pinocchio-webchat-refactor-getorcreateconv-to-enginebuilder-pattern/various/PI-001 Webchat EngineBuilder Refactor.pdf`

### What didn't work
- Upload failed with DNS resolution errors:

  - `dial tcp: lookup internal.cloud.remarkable.com: no such host`
  - `dial tcp: lookup webapp-prod.cloud.remarkable.engineering: no such host`

### What I learned
- This environment can run `remarquee` and generate PDFs, but cannot currently resolve reMarkable
  cloud endpoints. The correct workflow here is “generate PDF locally, upload from host machine”.

### What was tricky to build
- N/A (tooling issue, not a design/implementation issue).

### What warrants a second pair of eyes
- If this DNS issue persists outside this environment, we may need to re-auth or check proxy/VPN/DNS,
  but the immediate fix is to run the upload on the host where `rmapi` already works.

### What should be done in the future
- Upload the generated PDF from a network-enabled environment using either:
  - `remarquee upload md ...` / `remarquee upload bundle ...`, or
  - the legacy `remarkable_upload.py` script if needed.

### Code review instructions
- Open the generated PDF and confirm formatting/ToC:
  - `geppetto/ttmp/2026/01/22/PI-001-WEBCHAT-ENGINEBUILDER--pinocchio-webchat-refactor-getorcreateconv-to-enginebuilder-pattern/various/PI-001 Webchat EngineBuilder Refactor.pdf`

### Technical details
- PDF generation command used (no upload):
  - `remarquee upload bundle ... --pdf-only --output-dir .../various --toc-depth 2`

## Step 6: Add Tasks For Removing toolbox (Before Implementing)

This step updates the PI-001 task list to explicitly include removing `toolbox` usage from Pinocchio’s
webchat runtime. This is meant to keep the refactor honest: if the new builder/manager wiring still
relies on implicit tool wiring via `toolbox`, we’ll keep accumulating magic, rather than converging on
a clear “tool factories → registry → tool loop” flow.

### What I did
- Added two PI-001 tasks to remove `toolbox` dependency and any remaining webchat/example wiring that
  relies on it.

### Why
- The new webchat design we’re moving toward should be explicit about tool registration and execution:
  tools are registered via factories into a `ToolRegistry`, and the tool loop is driven via the session
  runner. `toolbox` often becomes a catch-all that obscures where tools come from.

### What worked
- The tasks are now explicitly tracked in:
  - `geppetto/ttmp/2026/01/22/PI-001-WEBCHAT-ENGINEBUILDER--pinocchio-webchat-refactor-getorcreateconv-to-enginebuilder-pattern/tasks.md`

### What didn't work
- N/A

### What I learned
- Making the “remove toolbox” objective explicit early helps prevent scope creep during implementation.

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- Confirm the intended scope: “remove toolbox from webchat runtime” (and possibly webchat examples),
  not “delete geppetto/pkg/toolbox globally”.

### What should be done in the future
- Audit where `toolbox` is currently referenced from Pinocchio (and specifically webchat).

### Code review instructions
- Verify the new tasks exist under PI-001 tasks list.

### Technical details
- Added tasks:
  - “remove `geppetto/pkg/toolbox` dependency from webchat runtime”
  - “remove/replace any remaining `toolbox` wiring in webchat and examples”

## Step 7: Audit toolbox Usage (Pinocchio/Moments)

This step audited current usage of `toolbox` across Pinocchio and Moments to understand what “remove
toolbox” concretely means for PI-001. The surprising result is that Pinocchio’s current codebase does
not reference `geppetto/pkg/toolbox` at all (and Moments didn’t show direct references either). That
means the “toolbox removal” tasks are either:

- referring to older/deprecated tool middleware patterns in Geppetto, or
- referring to a future-direction cleanup (ensuring webchat does not reintroduce toolbox-style execution).

Either way, it’s safe to proceed with PI-001’s refactor without touching `geppetto/pkg/toolbox` today,
while keeping the tasks as guardrails.

### What I did
- Searched for `toolbox` references in:
  - `pinocchio/`
  - `moments/backend/`
  - and confirmed `geppetto/pkg/toolbox` exists but is not imported by Pinocchio.

### Why
- Before starting implementation, we need to know if “remove toolbox” is a mechanical refactor
  (remove imports/calls) or an architectural constraint (avoid adding it back).

### What worked
- Found no `toolbox` references under Pinocchio.
- Confirmed `geppetto/pkg/toolbox` is primarily used by the deprecated tool middleware layer.

### What didn't work
- N/A

### What I learned
- The main actionable “tooling cleanup” for PI-001 is not deleting toolbox code, but ensuring the new
  EngineBuilder/manager refactor remains centered on:
  - `tools.ToolRegistry` + tool loop execution (not toolbox/middleware execution),
  - explicit tool registration via factories.

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- Double-check whether any Pinocchio examples outside `pinocchio/` (or downstream consumers) are still
  depending on toolbox indirectly; the current repo grep suggests “no”.

### What should be done in the future
- Consider moving toolbox removal tasks to the Geppetto “remove tool middleware” ticket if the intent
  is to delete the old tool middleware layer entirely.

### Code review instructions
- Re-run these greps if needed:
  - `rg -n \"\\btoolbox\\b\" pinocchio -S`
  - `rg -n \"github.com/go-go-golems/geppetto/pkg/toolbox\" pinocchio -S`

### Technical details
- `geppetto/pkg/toolbox` exists (reflection-based toolbox used by `geppetto/pkg/inference/middleware/tool_middleware.go`).

## Step 8: Start implementation — EngineBuilder-style `getOrCreateConv` + ID surfacing

This step starts the concrete refactor in Pinocchio’s `pkg/webchat` to remove duplicated per-endpoint engine construction closures and replace them with a single “build config + build from config” flow. The immediate goal is to make `getOrCreateConv` able to rebuild the engine/sink/subscriber when the profile or overrides change, instead of silently reusing the first-created engine forever.

While touching this path, I also aligned the webchat API surface to return and log the canonical correlation IDs (`session_id`, `inference_id`, `turn_id`) in addition to the historical `run_id` field name where still used.

**Commit (code):** 2886551 — "webchat: return session/inference/turn ids; add engine config builder"

### What I did
- Added an `EngineConfig` type with a deterministic signature:
  - `pinocchio/pkg/webchat/engine_config.go`
- Added an EngineBuilder-style API on the router (BuildConfig + BuildFromConfig) and subscriber factory helper:
  - `pinocchio/pkg/webchat/engine_builder.go`
- Refactored `getOrCreateConv` to compute a config signature and rebuild when `profileSlug`/overrides change:
  - `pinocchio/pkg/webchat/conversation.go`
- Updated `/chat` endpoints to call `getOrCreateConv(convID, profileSlug, overrides)` instead of inlining build closures, and to return `session_id`/`inference_id`/`turn_id` for each run:
  - `pinocchio/pkg/webchat/router.go`
- Updated the web frontend to treat `session_id` as primary:
  - `pinocchio/cmd/web-chat/web/src/store.js`

### Why
- The old flow cached a `Conversation` per `conv_id` but never rebuilt on config changes, so a later request with different profile/overrides could still run with the wrong engine/sinks.
- Surfacing correlation IDs in responses makes it easier to:
  - attach a streaming client to the right session,
  - debug event filtering,
  - and eventually migrate away from the ambiguous `run_id` naming.

### What worked
- `cd pinocchio && go test ./... -count=1`

### What didn't work
- N/A (kept the change compile-safe; deeper behavioral tests are still missing).

### What I learned
- Pinocchio already had most of the primitives needed for the go-go-mento-style pattern; the main missing piece was turning per-route construction into a reusable config/signature + build step.

### What was tricky to build
- Subscriber lifecycle: for the in-memory subscriber (`r.router.Subscriber`) we must not close it when rebuilding a conversation, while Redis group subscribers can be closed and recreated.

### What warrants a second pair of eyes
- Concurrency correctness around rebuilding while a stream is running:
  - Ensure we never close the shared in-memory subscriber.
  - Ensure stream coordinator restart behavior can’t race with WS attachment/detachment.

### What should be done in the future
- Add focused tests around “same conv_id + different overrides” ⇒ engine rebuild happens and event filtering remains correct.
- Consider renaming `Conversation.RunID` to `SessionID` in Pinocchio to eliminate the last in-code ambiguity once API stability constraints allow it.

### Code review instructions
- Start with:
  - `pinocchio/pkg/webchat/engine_config.go`
  - `pinocchio/pkg/webchat/engine_builder.go`
  - `pinocchio/pkg/webchat/conversation.go`
  - `pinocchio/pkg/webchat/router.go`
- Validate with:
  - `cd pinocchio && go test ./... -count=1`

### Technical details
- Rebuild detection key: `EngineConfig.Signature()` (currently JSON-based; stable ordering via `encoding/json` rules for maps after normalization).
