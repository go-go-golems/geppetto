---
Title: OAuth profile credential persistence and refresh endpoint design
Ticket: PINOCCHIO-OAUTH-PROFILE-CREDENTIALS
Status: active
Topics:
    - pinocchio
    - oauth
    - credentials
    - profiles
    - security
DocType: design-doc
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: repo://pkg/inference/engine/factory/factory.go
      Note: Planned injection point for the Pinocchio-created credential source
    - Path: repo://pkg/steps/ai/credentials/bearer.go
      Note: |-
        Geppetto host-facing renewable credential contract to be consumed by this work
        Host contract Pinocchio will implement for profile-backed OAuth credentials
        Host contract the future Pinocchio store and refresher will implement
    - Path: repo://pkg/steps/ai/credentials/oauth/oauth.go
      Note: Reusable profile-agnostic OAuth PKCE, code exchange, and forced refresh client
    - Path: repo://pkg/steps/ai/credentials/oauth/oauth_test.go
      Note: Fake token-endpoint coverage for protocol, rotation, and redaction
ExternalSources:
    - https://github.com/go-go-golems/geppetto/issues/387
Summary: Split design for Geppetto OAuth protocol primitives and Pinocchio profile-backed credential, browser, and injection ownership.
LastUpdated: 2026-07-10T20:55:00-04:00
WhatFor: Plan the host integration that supplies persisted renewable OAuth credentials to Geppetto.
WhenToUse: Use before implementing Pinocchio profile OAuth authentication or a provider refresh adapter.
---



# OAuth profile credential persistence and refresh endpoint design

## Executive summary

Geppetto now exposes a reusable request-time bearer credential seam, including a renewable implementation and one bounded OpenAI-compatible 401 replay. It will also own a pure OAuth protocol package: PKCE values, authorization URL construction, authorization-code exchange, refresh-token grant execution, expiry normalization, and redacted errors. The package is configuration- and UI-agnostic: it accepts explicit OAuth inputs and returns a `credentials.Credential`.

Pinocchio remains the host integration layer. It owns the browser/loopback listener, selected provider/client/scopes, profile schema, durable YAML persistence, file permissions, and construction/injection of the Geppetto source. This separation permits another Geppetto host to reuse standard OAuth protocol mechanics without inheriting Pinocchio’s profile format or command UX.

The requested storage model is explicit: profile YAML contains the OAuth access token, refresh token, and expiry metadata. That is acceptable only as an opt-in secret-bearing profile format with mode `0600`, atomic updates, redacted display/export paths, and a clear warning that Git, sync services, backups, and ticket attachments must never receive the file. A later optional encrypted/keyring backend may improve at-rest protection, but it must not delay a correct, narrowly-scoped first implementation.

## Problem statement and scope

A static API key in a profile cannot represent an OAuth credential lifecycle. A provider can reject an access token even when the user has a valid refresh token locally; manually copying a replacement token is fragile and defeats long-running automation. The user needs a browser login path for the initial grant and a refresh-endpoint path for renewal and token rotation.

### In scope

- A Geppetto pure OAuth protocol package for PKCE, authorization URL construction, code exchange, forced refresh, expiry normalization, and redacted failures.
- A versioned Pinocchio profile YAML schema for OAuth bearer credentials.
- Strict `0600` file permissions and atomic, locked credential updates.
- Authorization Code with PKCE S256 browser login through a loopback callback.
- State/nonce validation, callback timeout, authorization-code exchange, and refresh-token exchange.
- Refresh-token rotation persistence before the new access token is handed to Geppetto.
- A Pinocchio adapter that configures Geppetto’s pure OAuth client and implements `credentials.Store` plus `credentials.Refresher`.
- Exact integration point that injects a source into the selected OpenAI-compatible Geppetto engine/factory.
- Redaction, diagnostics, migration, and test strategy.

### Out of scope

- Changing Geppetto `APISettings` to own OAuth configuration, profiles, or browser behavior.
- Writing tokens to shell output, generated docs, reMarkable bundles, telemetry, debug taps, or profile introspection output.
- Supporting every provider’s bespoke OAuth variant in the initial release.
- Browser-hosted OAuth callbacks, device code flow, and client-credential grant unless a selected provider requires them.
- Claude/Gemini/embedding/transcription authentication integration; those require separate provider-header and replay audits.

## Architecture and ownership

```text
+------------------+       loopback callback       +------------------+
| User's browser   | <---------------------------> | Pinocchio CLI    |
+------------------+  authorization code + state    | auth login       |
          |                                         +--------+---------+
          | authorization request (PKCE S256)                |
          v                                                  | atomic 0600 save
+------------------+  code/refresh-token exchange    +-------v----------------+
| OAuth provider   | <------------------------------ | Pinocchio profile YAML |
+------------------+                                 | oauth credential block |
          ^                                         +-------+----------------+
          | refresh grant                                    |
          |                                                  | Store / Refresher
          |                                         +--------v---------+
          +-----------------------------------------| Geppetto source  |
                                                    | request-time auth |
                                                    +--------+---------+
                                                             |
                                                             v
                                                    OpenAI-compatible API
```

### Boundary rules

- **Geppetto owns standard OAuth protocol mechanics.** Its pure package builds PKCE authorization requests, exchanges authorization codes, sends refresh grants, and converts standard token responses into `credentials.Credential`. It owns neither a profile path nor a browser listener.
- **Pinocchio owns secrets and application policy.** It knows profile locations, selected issuer/client/scopes, permission policy, browser callback lifecycle, and which credential gets persisted.
- **Geppetto owns inference request timing.** It calls the injected source immediately before a provider request and may ask the source for one replacement after a provider-originated 401.
- **The profile file is secret material.** It is not a normal portable configuration artifact even though it is YAML.
- **Provider adapters own non-standard quirks.** A standard OAuth client is reusable; a provider-specific wrapper remains responsible for non-standard parameters, response fields, or rotation semantics.

## Profile schema

The exact profile model must be located during implementation, but its OAuth block should have one stable shape and explicit schema semantics:

```yaml
name: umans-base
api_type: openai
base_url: https://api.example.invalid/v1
auth:
  kind: oauth_bearer
  provider: umans
  token_endpoint: https://issuer.example.invalid/oauth/token
  client_id: public-cli-client
  scopes:
    - inference
  access_token: <secret>
  refresh_token: <secret>
  expires_at: "2026-07-10T21:30:00Z"
```

Rules:

1. `access_token`, `refresh_token`, and `expires_at` are a single update unit. A persisted rotated refresh token must never be split from its matching access token.
2. `expires_at` is RFC 3339 UTC. A missing expiry is allowed only when a provider explicitly issues non-expiring bearer credentials.
3. `token_endpoint`, `client_id`, scopes, and provider name are configuration—not secrets—but must still be validated against the selected provider adapter.
4. A profile with `auth.kind: oauth_bearer` must not also silently fall back to `api_key`; ambiguous credential precedence is an error.
5. Missing/invalid token state produces a redacted actionable error such as “run `pinocchio auth login --profile umans-base`”, never a token echo.

## Safe YAML persistence

A YAML-backed store is viable only if it has transactional behavior at the file level:

```text
lock(profile.yaml.lock)
  read + parse current YAML
  locate exact named profile
  validate replacement credential
  replace only auth.access_token, auth.refresh_token, auth.expires_at
  serialize complete document to same-directory temporary file (0600)
  fsync temporary file
  rename temporary file over profile YAML
  fsync parent directory where supported
unlock
```

The store must preserve unrelated profiles, comments where the parser can preserve them, and any unknown fields. If comment-preserving round-trip YAML is unavailable, this is a design decision to record before implementation rather than silently reformat user configuration.

Startup and every save check permissions. If the profile is group/world-readable, Pinocchio should refuse to operate by default and tell the user to restrict it; a narrowly documented repair command may set `0600`. Backups and recovery files must also be owner-readable only and never remain as plaintext temporary files after a successful rename.

## Browser OAuth flow

The initial grant uses Authorization Code with PKCE S256 and a local loopback listener:

```text
pinocchio auth login --profile <name>
  validate selected oauth_bearer profile and local callback policy
  generate cryptographic state, nonce, code_verifier, code_challenge=S256(verifier)
  bind 127.0.0.1 on an ephemeral port before opening browser
  build authorization URL with exact redirect_uri, state, nonce, challenge, scopes
  open browser; print only sanitized authorization host/instructions if necessary
  accept exactly one GET callback before a short deadline
  require exact callback path and constant-time state equality
  reject provider error, missing code, duplicate callback, or mismatched state
  exchange code + verifier at configured token endpoint
  validate token response and calculate expires_at
  persist credential atomically through the profile store
  close listener and erase state/verifier from memory as practical
```

The loopback listener must bind loopback only, use a random state, and return a minimal browser completion page without token values. It must not accept wildcard redirect URIs or forward the authorization code to another process.

## Refresh endpoint support

Geppetto provides the protocol-level authorization-code and forced refresh operations. A Pinocchio provider adapter supplies selected profile configuration and any documented non-standard parameters, then maps the result into its profile store. The protocol result contains `access_token`, optional rotated `refresh_token`, and `expires_at`.

```go
client, err := oauth.NewClient(oauth.Config{
    AuthorizationURL: profile.Auth.AuthorizationURL,
    TokenURL:         profile.Auth.TokenURL,
    ClientID:         profile.Auth.ClientID,
    RedirectURL:      redirectURL,
    Scopes:           profile.Auth.Scopes,
}, oauth.WithRefreshTokenPolicy(oauth.PreservePreviousRefreshToken))

replacement, err := client.Refresh(ctx, previous)
credential, err := client.ExchangeAuthorizationCode(ctx, code, pkce)
```

Pinocchio’s provider adapter configures this client and can add documented provider-specific behavior outside the standard path. It chooses whether an omitted refresh token preserves the old value or is rejected because that provider rotates on every grant.

Refresh pseudocode:

```text
credential = store.Load(profile identity)
if credential is usable outside skew:
    return credential.access_token
response = adapter.Refresh(profile OAuth config, credential)
replacement = normalize + validate response
if response omitted refresh_token:
    replacement.refresh_token = credential.refresh_token  # only if provider contract permits
store.Save(profile identity, replacement)                 # atomic and 0600
return replacement.access_token
```

The “retain previous refresh token when omitted” rule must be adapter-controlled. Some providers rotate tokens strictly; blindly retaining an old token could violate their protocol.

## Geppetto integration

Pinocchio creates the existing `credentials.RenewableBearerTokenSource` with its YAML profile store and selected provider refresher, then passes it using `factory.WithBearerTokenSource(source)` or the equivalent engine option. Geppetto performs the cheap cached lookup per request. On its bounded provider 401 retry, it calls the optional unauthorized-source extension; the source forces a refresh and saves the replacement before it returns a retry token.

No Geppetto settings object receives refresh material. The profile resolver must construct the host source after parsing the secret-bearing profile but before factory validation/configuration emits any diagnostic representation.

## Decision records

### Decision: Pinocchio profile YAML is the initial durable store

- **Status:** proposed.
- **Context:** User explicitly wants access, refresh, and expiry data in profile YAML.
- **Options:** YAML with `0600`; OS keychain; encrypted file; external secret manager.
- **Decision:** Implement YAML with strict permissions and atomic updates first, while retaining a store interface that permits later migration.
- **Consequences:** Users must treat profiles as secrets and keep them out of repositories, backups, and attachments.

### Decision: Authorization Code with PKCE loopback login

- **Status:** proposed.
- **Context:** A public CLI must obtain an initial user grant without embedding a client secret.
- **Decision:** Use a loopback redirect, PKCE S256, state, exact callback validation, and a finite listener lifetime.
- **Consequences:** Selected provider clients must permit a loopback redirect; headless environments need a documented manual URL/open-browser path rather than a weaker callback policy.

### Decision: Geppetto protocol primitive plus Pinocchio provider adapter

- **Status:** accepted for the Geppetto primitive; proposed for the first Pinocchio adapter.
- **Context:** Standard OAuth authorization-code and refresh-token grants are reusable, while endpoint selection, custom parameters, and rotation semantics vary by provider.
- **Decision:** Geppetto implements a pure standard OAuth client with explicit config and redacted failures. Pinocchio wraps it for each selected provider and retains policy/quirk handling.
- **Consequences:** The protocol code is reusable by other Geppetto hosts, but no provider endpoint is guessed and no profile behavior leaks into Geppetto.

## Implementation phases

1. **Geppetto OAuth primitive:** implement/test PKCE, authorization URL, authorization-code exchange, forced refresh, expiry normalization, explicit rotation policy, and redacted failures.
2. **Pinocchio discovery and schema:** locate profile model/resolver/output surfaces; select initial provider; add fixtures and migration rules.
3. **Secret-safe YAML store:** locking, parse/update/write, permissions, race tests, redaction tests, and failure recovery.
4. **Pinocchio OAuth adapter and browser login:** configure the Geppetto primitive, add loopback callback orchestration, code exchange, cancellation/timeout, and sanitized operator UX.
5. **Geppetto wiring:** build/inject source for OAuth profiles; prove proactive expiry refresh and one bounded 401 replay end to end.
6. **Operations:** docs, revoke/logout behavior, migration/recovery guide, and permission repair workflow.

## Validation plan

- Unit tests for schema parsing, redacted formatting, file mode rejection, atomic save, rotation, and unrelated-profile preservation.
- Fake OAuth server tests for code exchange, refresh, refresh-token omission/rotation, provider errors, expiry skew, and no secret values in errors.
- Browser callback tests for state mismatch, duplicate callback, timeout, wrong path, and successful PKCE request parameters.
- Geppetto integration test proving a profile-backed source refreshes and retries one provider 401 without exposing secrets.
- Race tests for simultaneous inference refresh plus profile writes.
- Manual test with the selected provider only after credentials are stored in a local `0600` file excluded from Git and test output.

## Risks and open questions

- Which Pinocchio repository/package owns profile parsing and the CLI command must be established in Phase 1.
- A YAML parser may not preserve comments/order; decide whether preservation is a user-visible requirement.
- Determine the initial provider’s documented redirect URI, scopes, token endpoint, and refresh-rotation rules before coding.
- Decide whether `client_id` belongs in the profile or a provider registry. It is not secret but may be centrally controlled.
- Establish behavior for expired/revoked refresh credentials: fail closed and direct the user to `auth login`; never downgrade to a stale static API key.

## References

- `pkg/steps/ai/credentials/bearer.go` — Geppetto public source/store/refresher contracts and forced refresh after provider 401.
- `pkg/steps/ai/credentials/oauth/oauth.go` — profile-agnostic OAuth protocol primitives implemented in this ticket.
- `pkg/steps/ai/credentials/oauth/oauth_test.go` — fake token-endpoint and PKCE/redaction test coverage.
- `pkg/inference/engine/factory/factory.go` — source injection into OpenAI-compatible engines.
- Geppetto issue [#387](https://github.com/go-go-golems/geppetto/issues/387) — original generic credential lifecycle issue.
