# Tasks

## Phase 0 — Evidence and design

- [x] Map static API-key configuration, factory validation, OpenAI Chat, OpenAI Responses, and redaction boundaries <!-- t:p0a1 -->
- [x] Search existing Geppetto issues and file [#387](https://github.com/go-go-golems/geppetto/issues/387) with the discovered gap and security constraints <!-- t:p0a2 -->
- [x] Define a public host-injected credential contract that does not serialize refresh material into profile settings <!-- t:p0a3 -->
- [x] Write an intern-oriented analysis, design, API, flow, test, and operations guide <!-- t:p0a4 -->

## Phase 1 — Renewable source implementation

- [x] Add a public `credentials.Request`, `Credential`, `Store`, `Refresher`, and `BearerTokenSource` contract <!-- t:p1a1 -->
- [x] Add an in-memory cache keyed by non-secret provider/base-URL identity <!-- t:p1a2 -->
- [x] Implement expiry-plus-skew validation and host-owned refresh/persistence <!-- t:p1a3 -->
- [x] Collapse concurrent refreshes per credential key with `singleflight` <!-- t:p1a4 -->
- [x] Make waiting requests honor context cancellation <!-- t:p1a5 -->
- [x] Return redacted availability failures and never retain an unpersisted rotated credential <!-- t:p1a6 -->
- [x] Add source unit tests for cache hit, rotate/persist, concurrency, cancellation, and redaction <!-- t:p1a7 -->

## Phase 2 — Provider and factory wiring

- [x] Add request-time bearer-source engine options for OpenAI Chat and OpenAI Responses <!-- t:p2a1 -->
- [x] Resolve and validate the provider URL before asking the source for a bearer value <!-- t:p2a2 -->
- [x] Preserve static API-key fallback when no source is configured <!-- t:p2a3 -->
- [x] Make a configured source authoritative and permit factory construction without a static OpenAI-compatible key <!-- t:p2a4 -->
- [x] Add direct-engine and factory tests proving request-time source precedence and Responses source/error behavior <!-- t:p2a5 -->

## Phase 3 — Validation, documentation, and delivery

- [x] Generate/check logcopter output for the new package <!-- t:p3a1 -->
- [x] Run focused, full, race, formatting, lint, glazed-lint, logcopter, and vulnerability validation <!-- t:p3a2 -->
- [x] Commit implementation and record exact validation evidence in the diary <!-- t:p3a3 -->
- [x] Update ticket file relations, changelog, and task status <!-- t:p3a4 -->
- [x] Run `docmgr doctor` cleanly <!-- t:p3a5 -->
- [x] Dry-run then upload the guide/diary/tasks bundle to reMarkable and verify the remote listing <!-- t:p3a6 -->
