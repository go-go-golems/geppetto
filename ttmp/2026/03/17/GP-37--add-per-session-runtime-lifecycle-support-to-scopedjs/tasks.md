# Tasks

## Ticket Setup

- [x] Create the `GP-37` ticket workspace under `geppetto/ttmp`
- [x] Review the current `scopedjs` runtime lifecycle code paths
- [x] Review the existing Geppetto session identity plumbing
- [x] Draft an intern-friendly design and implementation guide
- [x] Draft the GitHub issue body from the same material
- [x] File the upstream GitHub issue and link it back into this ticket
- [x] Run `docmgr doctor --root geppetto/ttmp --ticket GP-37 --stale-after 30`

## Future Implementation Slice 1: Public API Design

- [ ] Decide the public API shape for per-session support
- [ ] Keep registration-driven semantics honest and avoid reintroducing a misleading `StateMode` field on `EvalOptions`
- [ ] Choose whether per-session support is exposed as a new registrar helper, a new registration mode enum, or a small runtime manager object
- [ ] Define how callers provide or override the session key source
- [ ] Define failure behavior when no session identifier is present in context
- [ ] Update `scopedjs` documentation examples to show the final API

## Future Implementation Slice 2: Runtime Pool and Ownership Model

- [ ] Add an internal runtime-pool type keyed by session identifier
- [ ] Define the per-entry state structure, including runtime handle, manifest, creation time, last-used time, and any error or poison markers
- [ ] Ensure only one goroutine mutates or evaluates a given session runtime at a time
- [ ] Define cleanup hooks for explicit close, eviction, and shutdown
- [ ] Decide whether manifests are stored per runtime entry or shared from a describe hook

## Future Implementation Slice 3: Session Integration

- [ ] Use `session.SessionIDFromContext(ctx)` as the default session key source
- [ ] Allow advanced callers to provide a custom session key resolver when needed
- [ ] Decide whether empty session ID means hard error, fallback to per-call runtime, or opt-in downgrade behavior
- [ ] Document how this interacts with existing `session.Session` and toolloop execution flows

## Future Implementation Slice 4: Cleanup and Eviction

- [ ] Add idle-time eviction support
- [ ] Add maximum live session-runtime bounds or document why they are intentionally omitted
- [ ] Decide how to dispose of poisoned runtimes after fatal bootstrap or eval failures
- [ ] Decide whether eviction is synchronous on access, background, or both
- [ ] Add metrics/logging hooks if the package already has a local pattern for them

## Future Implementation Slice 5: Testing

- [ ] Add tests proving same-session calls observe retained state
- [ ] Add tests proving different sessions do not share state
- [ ] Add tests proving missing session ID behavior matches the chosen contract
- [ ] Add tests proving concurrent calls into the same session do not race or corrupt the runtime
- [ ] Add tests proving idle cleanup and shutdown close runtimes
- [ ] Add tests proving prebuilt and lazy registrars remain backward compatible

## Future Implementation Slice 6: Docs and Adoption

- [ ] Extend the `scopedjs` tutorial and examples once the final API lands
- [ ] Add a migration note explaining when to use prebuilt, lazy, and per-session runtime strategies
- [ ] Add an example that demonstrates session-scoped state safely
- [ ] Close out this ticket with validation notes and implementation diary updates
