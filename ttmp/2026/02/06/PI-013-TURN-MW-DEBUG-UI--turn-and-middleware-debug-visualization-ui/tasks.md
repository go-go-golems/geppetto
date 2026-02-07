# Tasks

## Completed

- [x] Create ticket workspace `PI-013-TURN-MW-DEBUG-UI`
- [x] Author long-form analysis/specification document (designer handoff)
- [x] Author detailed frequent diary and backfill execution steps
- [x] Relate core source files to analysis/diary documents
- [x] Upload bundled analysis+diary document to reMarkable
- [x] Add standalone designer primer (turns/blocks/middlewares/structured events)
- [x] Upload standalone designer primer separately to reMarkable

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

## Next

- [ ] Review spec + primer with designer and prioritize MVP screen set
- [ ] Convert FR/IR/NFR requirements into implementation tasks
