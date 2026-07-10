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
    - Path: repo://pkg/steps/ai/credentials/bearer.go
      Note: Existing Geppetto host-facing source contract
ExternalSources:
    - https://github.com/go-go-golems/geppetto/issues/387
Summary: 'Chronological plan and evidence for Pinocchio OAuth profile credential lifecycle work.'
LastUpdated: 2026-07-10T20:55:00-04:00
WhatFor: 'Continue or review the profile-backed OAuth credential implementation.'
WhenToUse: 'Use when implementing Pinocchio OAuth persistence, refresh calls, browser login, or Geppetto injection.'
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
