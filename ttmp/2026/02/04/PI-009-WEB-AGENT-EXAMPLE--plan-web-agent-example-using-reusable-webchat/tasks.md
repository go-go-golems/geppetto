# Tasks

## TODO

- [ ] Task 1: Baseline repo + scaffolding audit
- [ ] Task 1.1: Inventory current web-agent-example layout (cmd/, pkg/, static/, web/).
- [ ] Task 1.2: Identify missing dependencies to run a standalone server (Go modules, embed paths, assets).
- [ ] Task 1.3: Decide whether pinocchio refactors are required for external webchat (capture changes).

- [x] Task 2: Custom thinking-mode events (web-agent-example)
- [x] Task 2.1: Create new event types + payload structs with a unique namespace (e.g., webagent.thinking.*).
- [x] Task 2.2: Register event factories and add basic serialization tests.

- [x] Task 3: Custom thinking-mode middleware (web-agent-example)
- [x] Task 3.1: Implement middleware to emit started/update/completed events around inference.
- [x] Task 3.2: Wire middleware into the router assembly in web-agent-example main.

- [x] Task 4: SEM translation + timeline projection for custom events (pinocchio/pkg/webchat or extension)
- [x] Task 4.1: Add SEM mapping for custom events into websocket frames.
- [x] Task 4.2: Add timeline projection branch to map custom events into a custom entity kind.
- [x] Task 4.3: Validate durable timeline store writing for custom entity kind.

- [x] Task 5: Frontend custom UI (web-agent-example web app)
- [x] Task 5.1: Create custom ThinkingModeCard and register renderer for custom entity kind.
- [x] Task 5.2: Add thinking-mode switch in Composer (custom slot override).
- [x] Task 5.3: Serialize selected mode into POST overrides (extend or wrap ChatWidget send).

- [x] Task 6: Build + embed frontend assets
- [x] Task 6.1: Build web app to static/dist.
- [x] Task 6.2: Embed static assets in web-agent-example Go binary.

- [x] Task 7: Runtime validation + correlation
- [x] Task 7.1: Run server + web app in tmux.
- [x] Task 7.2: Use Playwright to exercise UI and capture timeline behavior.
- [x] Task 7.3: Query timeline store to correlate with UI results.
- [ ] Task 7.4: Ask user to manually test critical flows.

- [ ] Task 8: Cleanup + docs
- [ ] Task 8.1: Update diary and changelog after each task.
- [ ] Task 8.2: Capture any required pinocchio refactors and document them.
- [ ] Task 8.3: Commit code and docs in logical chunks.

- [ ] Task 9: Redux-first state integration in web-agent-example
- [x] Task 9.1: Ensure timeline/chat runtime state is driven by the `@pwchat` Redux store (`timelineSlice`, app slice), not parallel local stores.
- [x] Task 9.2: Keep only ephemeral UI control state local (transient composer toggles); move durable UI/session state to Redux actions/selectors.
- [ ] Task 9.3: Validate hydration + websocket merge behavior still correct with Redux as the single source of truth.

- [x] Task 10: Protobuf contract for Go↔TS SEM boundary
- [x] Task 10.1: Define/confirm protobuf message schemas for custom middleware SEM payloads at `pinocchio/proto/sem/middleware/*`.
- [x] Task 10.2: Ensure Go SEM emission uses generated protobuf + `protojson` serialization for payloads.
- [x] Task 10.3: Ensure TS SEM handlers decode generated schemas with `fromJson` and map into timeline entities deterministically.
- [x] Task 10.4: Add/refresh boundary checks (Go tests and TS typecheck coverage) for schema drift detection.

- [x] Task 11: Update PI-009 guide with Redux + protobuf boundary requirements
- [x] Task 11.1: Add explicit architecture constraints and file-level references for Redux state ownership.
- [x] Task 11.2: Add explicit Go/TS protobuf boundary flow and decode/encode snippet patterns.

## DONE

- [x] Confirm web-agent-example repo renames (no XXX remaining)
- [x] Write analysis guide for building web-agent-example using reusable webchat frontend/backend
- [x] Relate key source/docs to the analysis and diary
- [x] Upload analysis guide to reMarkable
