# Tasks

## Completed

- [x] Create ticket workspace `PI-013-TURN-MW-DEBUG-UI`
- [x] Author long-form analysis/specification document (designer handoff)
- [x] Author detailed frequent diary and backfill execution steps
- [x] Relate core source files to analysis/diary documents
- [x] Upload bundled analysis+diary document to reMarkable
- [x] Add standalone designer primer (turns/blocks/middlewares/structured events)
- [x] Upload standalone designer primer separately to reMarkable
- [x] Author deep engineering review of architecture proposal `analysis/05`
- [x] Upload engineering review document to reMarkable
- [x] Update design doc `analysis/05` with post-review decisions (Critical 1, High 3/4, Medium 1/4)

## Documentation Improvements (from proposal 03)

- [x] D1: Add "Why 'Turn' instead of 'Message'" section to 08-turns.md (HIGH)
- [x] D2: Add "How Blocks Accumulate During Inference" section to 08-turns.md (HIGH)
- [x] D3: Add visual turn growth example to Multi-turn Sessions in 08-turns.md (HIGH)
- [x] D4: Add "Middleware as Composable Prompting" section to 09-middlewares.md (HIGH)
- [x] D5: Add block-mutating middleware example to 09-middlewares.md (HIGH)
- [x] D6: Add post-processing middleware example to 09-middlewares.md (HIGH)
- [x] D7: Add "Complete Runtime Flow" section to 06-inference-engines.md (MEDIUM)
- [x] D8: Add context-based DI explanation to 06-inference-engines.md (MEDIUM)
- [x] D9: Add structured sink cross-reference and correlation ID explanation to 04-events.md (LOWER)
- [x] D10: Create new topic doc for structured sinks / FilteringSink (MEDIUM)
- [x] D11: Create new topic doc for session management (LOWER)

## Pinocchio Webchat Doc Improvements (from proposal 04)

### New documents

- [x] N1: Create end-to-end "Adding a New Event Type" tutorial as webchat-adding-event-types.md (HIGH)
- [x] N2: Add Timeline Projector reference section to webchat-backend-internals.md (HIGH)
- [x] N3: Add "Where Events Go: The SEM Translation Layer" bridge section to geppetto 04-events.md (MEDIUM)

### Improvements to existing docs

- [x] E1: Add "What Is Webchat?" elevator pitch to webchat-overview.md (MEDIUM)
- [x] E2: Add SEM frame payload examples to webchat-sem-and-ui.md (MEDIUM)
- [x] E3: Add debugging section to webchat-sem-and-ui.md (LOWER)
- [x] E4: Add error handling patterns to webchat-frontend-integration.md (LOWER)

### Content to port from go-go-mento

- [x] P1: Port EngineBuilder reference to pinocchio docs (MEDIUM)
- [x] P2: Port Conversation Lifecycle reference to pinocchio docs (MEDIUM) — pinocchio uses Conversation+ConvManager instead of a separate InferenceState struct
- [x] P3: Port SEM widget catalog and "Adding a New Widget" guide to webchat-sem-and-ui.md (HIGH)

## Frontend Implementation (web-agent-example/cmd/web-agent-debug/web)

### Done

- [x] Project scaffold (package.json, vite, tsconfig, storybook config)
- [x] TypeScript types (ConversationSummary, TurnDetail, ParsedBlock, SemEvent, MwTrace, TimelineEntity)
- [x] RTK Query API layer (all /debug/* endpoints)
- [x] Redux store (uiSlice with selection state, view modes, filters)
- [x] MSW mock handlers + realistic mock data with standard block metadata
- [x] ConversationCard component + stories
- [x] BlockCard component with expandable metadata + stories
- [x] CorrelationIdBar component + stories
- [x] SessionList component + stories
- [x] TurnInspector component (phase tabs, turn metadata card) + stories
- [x] EventCard component + stories
- [x] MiddlewareChainView component + stories
- [x] TimelineEntityCard component + stories

### Screen 1: Session Overview (three-lane timeline)

- [x] TimelineLanes container component
- [x] StateTrackLane (turn snapshots as vertical cards)
- [x] EventTrackLane (SEM events as vertical list)
- [x] ProjectionLane (timeline entities)
- [x] NowMarker (live streaming indicator)
- [x] Lane synchronization (scroll sync, time alignment)
- [x] Stories for all lane components

### Screen 3: Snapshot Diff

- [x] DiffHeader (phaseA vs phaseB labels)
- [x] SideBySideBlocks container
- [x] DiffBlockRow (status: same|added|removed|changed|reordered)
- [x] MetadataDiff (key-level diff highlighting)
- [x] DiffSummaryBar (counts: +added -removed ~changed ↔reordered)
- [x] Identity-aware block matching (not index-only)
- [x] Stories for diff components

### Screen 5: Event Inspector

- [x] ViewModeTabs (Semantic | SEM | Raw Wire)
- [x] SemanticView (human-readable event card)
- [x] SemEnvelopeView (JSON viewer for SEM frame)
- [x] RawWireView (provider-native JSON)
- [x] CorrelatedNodesPanel (linked turn/block, prev/next events, projection entity)
- [x] TrustSignals (correlation checks)
- [x] Stories for all views

### Screen 6: Structured Sink View

- [ ] ThreeColumnLayout (input events | sink config | output state)
- [ ] SinkConfigPanel
- [ ] OutputStatePanel
- [ ] Stories

### Screen 7: FilterBar

- [x] Filter overlay component
- [x] Block kind filters
- [x] Event type filters
- [ ] Time range filters (deferred - need date picker)
- [x] Search input
- [x] Stories

### Screen 8: AnomalyPanel

- [x] Slide-out panel component
- [x] Anomaly list (orphan events, missing correlations, timing outliers)
- [x] Anomaly detail view
- [x] Stories

### AppShell & Routing

- [ ] AppShell layout (sidebar + main content + panels)
- [ ] React Router integration (all screen routes)
- [ ] Sidebar navigation state
- [ ] Responsive layout

### Live Features

- [ ] WebSocket connection for live SEM events
- [ ] Real-time turn updates
- [ ] Live NowMarker animation
- [ ] Reconnection handling

## Backend Implementation

- [ ] Execute PI-014 correlation/migration implementation ticket
- [ ] Execute PI-015 EventStore postmortem ticket
- [ ] Execute PI-016 SEM/event performance ticket

## Next

- [ ] Review spec + primer with designer and prioritize MVP screen set
- [ ] Convert FR/IR/NFR requirements into implementation tasks
