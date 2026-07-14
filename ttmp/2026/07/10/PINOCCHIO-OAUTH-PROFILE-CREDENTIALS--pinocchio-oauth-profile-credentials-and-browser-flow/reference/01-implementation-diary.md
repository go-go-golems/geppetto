---
Title: Implementation diary
Ticket: PINOCCHIO-OAUTH-PROFILE-CREDENTIALS
Status: active
Topics:
    - pinocchio
    - oauth
    - credentials
    - profiles
    - security
DocType: reference
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: repo://pkg/js/runtime/runtime_test.go
      Note: Known unrelated repository-wide race failure
    - Path: repo://pkg/steps/ai/credentials/bearer.go
      Note: Existing Geppetto host-facing source contract
    - Path: repo://pkg/steps/ai/credentials/oauth/oauth.go
      Note: Implemented in commit 9e0d93b4
    - Path: repo://pkg/steps/ai/credentials/oauth/oauth_test.go
      Note: Protocol and redaction validation evidence
ExternalSources:
    - https://github.com/go-go-golems/geppetto/issues/387
Summary: Chronological plan and evidence for Pinocchio OAuth profile credential lifecycle work.
LastUpdated: 2026-07-10T20:55:00-04:00
WhatFor: Continue or review the profile-backed OAuth credential implementation.
WhenToUse: Use when implementing Pinocchio OAuth persistence, refresh calls, browser login, or Geppetto injection.
---


# Implementation diary

## Goal

Record the design and future implementation of Pinocchio-owned OAuth credentials in profile YAML, including a provider refresh endpoint and a browser authorization-code login flow that supplies Geppetto’s renewable bearer source.

## Step 1: Split host OAuth lifecycle from Geppetto’s generic source

This ticket was created after the generic Geppetto credential source gained a deliberately bounded OpenAI-compatible 401 retry. The generic library supplies request-time authorization mechanics; it does not own the user’s OAuth grant, profile file, token endpoint, or browser callback. Pinocchio is the correct host boundary for those responsibilities.

The user explicitly chose profile YAML as the first credential store. The proposed design therefore treats the profile as secret material: it requires owner-only permissions, atomic whole-document replacement, redacted output, and no inclusion in Git, diagnostics, ticket attachments, or reMarkable uploads.

### Prompt Context

**User prompt (verbatim):** "alright, only add 4A, and create a new docmgr ticket for 4D and persisting credentials, and adding support for refresh endpoint."

**Assistant interpretation:** Limit the current Geppetto change to OpenAI Chat/Responses one-time 401 refresh/replay, and create a separate planning ticket for Pinocchio OAuth browser login, profile-backed credentials, and refresh-endpoint support.

**Inferred user intent:** Keep the reusable library change small while establishing a concrete, durable host implementation path that stores refresh/access/expiry credentials in the user’s chosen profile format.

### What I did

- Created ticket `PINOCCHIO-OAUTH-PROFILE-CREDENTIALS`.
- Created this diary and a detailed design document.
- Defined the proposed profile OAuth block: access token, refresh token, RFC 3339 expiry, token endpoint, client ID, scopes, provider, and explicit `oauth_bearer` kind.
- Defined the file-lock, temporary-file, `fsync`, atomic rename, and `0600` persistence protocol.
- Defined Authorization Code with PKCE S256, loopback callback, state validation, and a short-lived listener.
- Defined a provider-adapter seam for code exchange and refresh grants, including provider-owned refresh-token rotation semantics.
- Defined the bridge from Pinocchio profile store/refresher to Geppetto’s `credentials.RenewableBearerTokenSource`.

### Why

- The request changes the original #387 non-goal: a host now needs an actual secret store and OAuth protocol implementation.
- Combining profile storage/browser security with the generic Geppetto package would couple a reusable inference library to one application’s configuration and provider policy.
- A ticket-level design prevents accidental token leakage through apparently harmless YAML introspection, logs, test fixtures, docs, or upload bundles.

### What worked

- The existing Geppetto source contracts map directly to the host design: `Store` performs profile persistence and `Refresher` performs provider protocol calls.
- The current source’s persistence-before-cache invariant aligns with token-rotation safety: a new access token is not used until the matching refresh material is durably stored.

### What didn't work

N/A — this ticket currently records design only. Pinocchio repository/package discovery and provider-specific OAuth endpoint verification are intentionally tracked as first implementation tasks rather than guessed from the Geppetto worktree.

### What I learned

- “YAML-backed” is a storage format decision, not a relaxation of secret-handling requirements.
- The 401 retry in Geppetto needs the profile store/refresher to force a provider refresh even if the old credential has a future expiry.
- OAuth token responses can omit a refresh token or rotate one; only a provider adapter can decide whether retaining the prior token is valid.

### What was tricky to build

The design must reconcile user-visible YAML with atomic secret rotation. Updating only an in-memory access token would work until process restart, while saving individual fields separately could leave an access token paired with the wrong refresh token. The proposed store updates the whole credential tuple in one locked, atomic document replacement and preserves the old file on any write failure.

### What warrants a second pair of eyes

- Verify the actual Pinocchio profile parser and command package before assigning file-level implementation work.
- Verify the chosen provider’s OAuth redirect, PKCE, scope, token endpoint, and rotation documentation.
- Decide whether comment/order preservation is required for profile writes.
- Review the threat model for plaintext owner-readable YAML versus a future encrypted/keyring backend.

### What should be done in the future

- Complete the discovery, schema, YAML store, refresh adapter, browser login, and Geppetto injection phases in `tasks.md`.
- Add real-provider testing only after local secret files are excluded from all logs and artifacts.

### Code review instructions

Start with the design document’s “Architecture and ownership,” “Safe YAML persistence,” and “Browser OAuth flow” sections. Then review Geppetto `pkg/steps/ai/credentials/bearer.go` to understand the host interfaces the future Pinocchio work must implement.

### Technical details

No access token, refresh token, expiry value, authorization code, client secret, or profile file content was read or recorded in this ticket.

## Step 2: Move standard OAuth protocol mechanics into Geppetto

The user provided a clean Pinocchio worktree and asked to update the design before implementing the Geppetto side. Inspection confirmed both modules already resolve `golang.org/x/oauth2`, while neither contains the requested OAuth profile lifecycle implementation. The design now separates pure, reusable protocol mechanics from Pinocchio-owned profile, browser, and persistence policy.

The new Geppetto package is intentionally not an OAuth application. It has no YAML parser, filesystem access, callback server, browser opener, provider registry, or profile name. It accepts explicit endpoints/client/scopes, produces `credentials.Credential`, and gives Pinocchio a tested building block for a later store/refresher adapter.

### Prompt Context

**User prompt (verbatim):** "btw I cloned /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/pinocchio (worktree).

Update the deisgn doc, then implement the geppetto side of things."

**Assistant interpretation:** Use the new clean Pinocchio worktree to validate the ownership boundary, revise the design to place reusable standard OAuth protocol operations in Geppetto, then implement and test that Geppetto package only.

**Inferred user intent:** Avoid overloading Pinocchio with reusable protocol code while retaining Pinocchio ownership of profile YAML and CLI/browser behavior.

**Commit (code):** `9e0d93b4c7dff3f7282ed87b4abb938c214a8b84` — "feat: add reusable OAuth credential client"

### What I did

- Inspected clean Geppetto and Pinocchio worktrees and confirmed both already resolve `golang.org/x/oauth2`.
- Revised the design to define the Geppetto/Pinocchio boundary and a new Geppetto OAuth primitive phase.
- Added `pkg/steps/ai/credentials/oauth` with:
  - explicit OAuth client config validation;
  - RFC 7636 PKCE S256 verifier/challenge creation;
  - authorization URL construction with state, PKCE, and offline access request;
  - authorization-code exchange through `golang.org/x/oauth2`;
  - direct forced refresh-token grant execution so a 401 can refresh even before recorded expiry;
  - expiry normalization, configurable omitted-refresh-token policy, and redacted errors.
- Added an in-process fake token-endpoint test suite for PKCE, code exchange, forced refresh, strict rotation, and endpoint-error redaction.

### Why

- Standard OAuth grants are useful to every Geppetto host; profile YAML and a browser listener are not.
- A direct refresh grant is required for the existing 401 recovery feature. `oauth2.TokenSource` may retain an omitted refresh token internally and cannot reveal whether rotation occurred, so it cannot implement an explicit strict-rotation policy.

### What worked

Focused protocol and existing credential/engine tests pass after the direct refresh implementation:

```text
GOWORK=off go test ./pkg/steps/ai/credentials/oauth -count=1
GOWORK=off go test ./pkg/steps/ai/credentials ./pkg/steps/ai/credentials/oauth ./pkg/steps/ai/openai ./pkg/steps/ai/openai_responses ./pkg/inference/engine/factory -count=1
```

Full `GOWORK=off go test ./... -count=1`, focused race coverage, `GOWORK=off make lint logcopter-check`, and `GOWORK=off make gosec` also passed. The pre-commit hook reran full tests and lint successfully for code commit `9e0d93b4`.

### What didn't work

The first multi-symbol Go documentation command was malformed:

```text
Usage of [go] doc:
	go doc
	go doc <pkg>
	go doc <sym>[.<methodOrField>]
```

I reran one symbol at a time and verified `GenerateVerifier`, `S256ChallengeFromVerifier`, `VerifierOption`, `S256ChallengeOption`, and `Config.Exchange` before coding.

The first strict-rotation test also failed because `oauth2.Config.TokenSource` automatically restores a prior refresh token when a response omits `refresh_token`:

```text
--- FAIL: TestRefreshCanRequireRotatedRefreshTokenAndRedactsEndpointFailure
    oauth_test.go:127: expected redacted missing-refresh error, got <nil>
```

I replaced that call with a direct standard `application/x-www-form-urlencoded` refresh grant parser. It preserves redaction while allowing Pinocchio to select either preserve-previous or require-replacement rotation policy.

Full `GOWORK=off go test -race ./... -count=1` again failed only at the existing `pkg/js/runtime.TestNewRuntime_DefaultJSEventsInitializerLogsListenerErrors` bytes.Buffer/zerolog race. The new OAuth package and all changed credential/engine packages passed focused `-race` coverage.

### What I learned

- The `golang.org/x/oauth2` PKCE helpers are sufficient for verifier/challenge construction and code exchange.
- A standard refresh response may omit `refresh_token`; policy belongs in the consumer, but the protocol helper must retain enough raw response distinction to enforce it.
- Forcing a refresh after a provider 401 cannot rely on an expiry-aware token source because the rejected token can have a future recorded expiry.

### What was tricky to build

The hardest boundary is not the HTTP POST; it is retaining a correct distinction between “the provider returned no new refresh token” and “a client library preserved the old token for convenience.” The initial library implementation erased that distinction, which would make strict token-rotation providers unsafe. The direct refresh parser preserves the distinction, converts only the required token fields, and returns generic errors instead of endpoint bodies or credentials.

### What warrants a second pair of eyes

- Review whether a future provider needs a client-auth style other than basic authentication with secret or form `client_id` without secret.
- Review whether `access_type=offline` should remain a default authorization parameter or be made a Pinocchio provider-adapter option.
- Review whether the pure client should enforce HTTPS in production while retaining injectable test transport/loopback compatibility.

### What should be done in the future

- Finish this Geppetto phase’s full/race/lint/logcopter validation and commit it.
- Use the new primitive from the Pinocchio profile store/refresher and browser-login implementation, not from Geppetto settings.
- Add provider-specific wrappers only after collecting their documented endpoint, rotation, scope, and client-auth behavior.

### Code review instructions

Start with `pkg/steps/ai/credentials/oauth/oauth.go`, then read `oauth_test.go`. Verify that no config/profile/browser code enters the package, refresh is forced independent of expiry, and every network failure is redacted. Run the focused commands above, then the ticket’s full validation commands.

### Technical details

The package uses `oauth2.Config.Exchange` only for standard authorization-code exchange. Its refresh path posts `grant_type=refresh_token` directly, decodes access/refresh/expiry fields, and applies `PreservePreviousRefreshToken` or `RequireReplacementRefreshToken` explicitly.
