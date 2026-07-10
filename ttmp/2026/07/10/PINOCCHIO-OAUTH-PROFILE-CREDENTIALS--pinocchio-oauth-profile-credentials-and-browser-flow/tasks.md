# Tasks

## Phase G — Geppetto profile-agnostic OAuth primitives

- [x] Add standard OAuth client config validation, PKCE S256 generation, and authorization URL construction <!-- t:pga1 -->
- [x] Add authorization-code exchange and forced refresh-token grant with expiry normalization and explicit refresh-token rotation policy <!-- t:pga2 -->
- [x] Add fake token-endpoint tests for PKCE, exchange, forced refresh, rotation policy, and secret-free failures <!-- t:pga3 -->
- [x] Generate/check logcopter, run focused/full/race/lint validation, and document the Geppetto boundary <!-- t:pga4 -->

## Phase 0 — Pinocchio discovery and constraints

- [ ] Locate the Pinocchio profile model, resolver, profile display/export paths, and CLI command root <!-- t:p0a1 -->
- [ ] Select the initial OAuth provider and collect its documented authorization/token endpoints, PKCE, redirect URI, scope, and rotation requirements <!-- t:p0a2 -->
- [ ] Audit existing profile file permissions, Git ignore rules, backup behavior, and secret-redaction boundaries <!-- t:p0a3 -->
- [ ] Confirm the Geppetto source injection point used by the selected Pinocchio inference path <!-- t:p0a4 -->

## Phase 1 — Profile schema and secret-safe store

- [ ] Add versioned `oauth_bearer` profile schema with access token, refresh token, expiry, provider, endpoint, client ID, and scopes <!-- t:p1a1 -->
- [ ] Add parsing, validation, migration, and redacted profile-formatting tests <!-- t:p1a2 -->
- [ ] Implement locked atomic YAML credential updates with temporary-file `0600`, fsync, rename, and parent sync where supported <!-- t:p1a3 -->
- [ ] Reject unsafe profile permissions by default and add a documented owner-only repair workflow <!-- t:p1a4 -->
- [ ] Test rotation, write failure recovery, concurrent access, unrelated-profile preservation, and secret-free errors <!-- t:p1a5 -->

## Phase 2 — Provider token endpoint and Geppetto adapter

- [ ] Implement provider adapter interfaces for authorization-code exchange and refresh-token grant <!-- t:p2a1 -->
- [ ] Normalize expiry and provider-specific refresh-token rotation behavior <!-- t:p2a2 -->
- [ ] Implement Pinocchio `credentials.Store` and `credentials.Refresher` backed by the profile store <!-- t:p2a3 -->
- [ ] Inject `RenewableBearerTokenSource` into the selected OpenAI-compatible Geppetto factory/engine path <!-- t:p2a4 -->
- [ ] Add fake-token-server integration tests for proactive refresh and one bounded provider-401 retry <!-- t:p2a5 -->

## Phase 3 — Browser authorization-code login

- [ ] Add `pinocchio auth login --profile <name>` command skeleton and explicit OAuth profile selection <!-- t:p3a1 -->
- [ ] Implement PKCE S256, state/nonce, loopback-only ephemeral callback listener, exact callback path, and timeout <!-- t:p3a2 -->
- [ ] Exchange authorization code, atomically persist credentials, and present a sanitized completion response <!-- t:p3a3 -->
- [ ] Test state mismatch, provider error, duplicate callback, timeout, cancellation, and success <!-- t:p3a4 -->

## Phase 4 — Operations and delivery

- [ ] Document profile secrecy, permission repair, revoke/re-login recovery, and refresh failure behavior <!-- t:p4a1 -->
- [ ] Run focused/full/race/security validation and a local real-provider smoke without logging secrets <!-- t:p4a2 -->
- [ ] Relate implementation files, update diary/changelog, run `docmgr doctor`, and review all token-redaction paths <!-- t:p4a3 -->
