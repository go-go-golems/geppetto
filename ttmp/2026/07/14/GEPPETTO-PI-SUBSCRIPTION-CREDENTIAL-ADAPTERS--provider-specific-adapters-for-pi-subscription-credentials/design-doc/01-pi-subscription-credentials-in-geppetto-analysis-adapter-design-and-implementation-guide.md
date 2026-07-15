---
Title: 'Pi subscription credentials in Geppetto: analysis, adapter design, and implementation guide'
Ticket: GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS
Status: active
Topics:
    - geppetto
    - oauth
    - credentials
    - security
DocType: design-doc
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: repo://pkg/inference/engine/factory/factory.go
      Note: Current engine-specific source propagation and validation
    - Path: repo://pkg/js/modules/geppetto/api_engine_builder.go
      Note: Go-only JavaScript source injection boundary
    - Path: repo://pkg/steps/ai/claude/api/completion.go
      Note: Claude static x-api-key request headers
    - Path: repo://pkg/steps/ai/claude/engine_claude.go
      Note: Current static Claude credential construction
    - Path: repo://pkg/steps/ai/credentials/bearer.go
      Note: Existing host-owned renewable bearer contract and invariants
    - Path: repo://pkg/steps/ai/openai_responses/streaming.go
      Note: Current bearer-only Responses request/replay behavior
    - Path: repo://ttmp/2026/07/14/GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS--provider-specific-adapters-for-pi-subscription-credentials/sources/01-local-pi-and-geppetto-source-map.md
      Note: Redacted local Pi provider and Geppetto evidence index
ExternalSources: []
Summary: Evidence-backed design for safely supporting provider-specific Pi subscription credential transports without making Geppetto own Pi storage or exposing credentials to JavaScript.
LastUpdated: 2026-07-14T21:52:00-04:00
WhatFor: Orient an engineer implementing provider-specific renewable credential adapters and transports in and around Geppetto.
WhenToUse: Use when evaluating Pi-originated subscription credentials, custom OAuth transports, or a future Geppetto request authentication capability.
---


# Pi subscription credentials in Geppetto

## Executive summary

Geppetto already has a secure, host-injected renewable bearer mechanism. Today a host owns persistence and refresh protocol while Geppetto asks for a bearer immediately before a supported OpenAI Chat or Responses request. This boundary is deliberately narrow and correct for ordinary OpenAI-compatible services, where `Authorization: Bearer <access-token>` is the whole authentication contract.

Installed Pi code shows that the three providers initially described as “OAuth” are not the same integration problem. OpenAI Codex is a genuine refreshable ChatGPT subscription flow, but its inference transport is the ChatGPT backend with a Codex-specific path and additional account, originator, beta, and request-ID headers. Anthropic Claude Pro/Max is also a genuine refreshable OAuth flow, but it targets the Anthropic Messages protocol while Geppetto’s Claude engine currently owns a static `x-api-key` client. Umans is not renewable OAuth in the Pi extension at all: `/login` persists an API key in OAuth-shaped fields, and its refresh function returns the same value.

Therefore this work must **not** copy Pi credentials into Geppetto profiles, make Geppetto parse `~/.pi/agent/auth.json`, or add “generic OAuth” claims based only on a token shape. The recommended path is a two-layer design:

1. **Geppetto provides reusable, provider-specific lifecycle and transport primitives.** These include PKCE/state/code-exchange and refresh mechanics for supported OAuth providers, typed redacted status and local deletion, store/lock/atomic-rotation helpers, and Go-only request-auth capabilities. It remains storage-location-, CLI-, browser-launch-, and Pi-file-agnostic.
2. **A host (initially Pinocchio) binds those primitives to its own persistence and user experience.** It selects a provider and profile, supplies its direct-YAML store, launches a browser or presents a device code, formats status, and applies consent/import policy. It never exposes token material, refresh callbacks, account identity, or source-selection metadata to profiles or JavaScript.

OpenAI Codex should use the existing OpenAI Responses core through a trusted Codex route resolver and request/response middleware, not through profile-configured paths or header maps. Claude should gain the same restricted middleware seam only after its OAuth request-header semantics are captured in a fake-server contract. Umans belongs in Geppetto’s provider catalog as an Anthropic Messages/API-key middleware/adapter, not a renewable OAuth adapter.

## 1. Problem statement and scope

### 1.1 The question

A machine running Pi already has local provider records for several login-based providers. Can a Geppetto host reuse a selected subscription credential safely and renew it at request time, while retaining the security invariants established by Geppetto’s renewable bearer work?

The answer is **potentially, provider by provider**. A record containing fields named `access`, `refresh`, and `expires` only establishes that Pi stores a credential in a common shape. It does not establish the inference endpoint, request framing, required companion headers, token audience, service terms, or compatibility with an existing Geppetto engine.

### 1.2 Goals

This design defines how to:

- classify the actual provider/transport contracts evidenced by installed Pi code;
- separate reusable Geppetto provider/lifecycle mechanics from host-owned credential placement and UX;
- add only the lifecycle and runtime seams required by a validated provider transport;
- keep all secret and credential-adjacent state out of profile YAML, settings dumps, events, logs, JavaScript, and CLI output;
- produce a phased implementation plan that an intern can execute with fake-server tests before any account-backed smoke.

### 1.3 Non-goals

This ticket does **not** authorize implementation of a provider adapter, a real account request, or a new browser-login command. It also does not:

- read, copy, print, hash, upload, or commit a value from Pi’s auth file;
- make Geppetto own Pi’s file format, profile path, browser-launch policy, or local CLI;
- claim the ChatGPT backend is a stable public OpenAI API contract;
- treat the Umans API-key shim as OAuth;
- expose a bearer, refresh callback, account identifier, raw headers, or credential-source selector to JavaScript;
- add a broad arbitrary HTTP-request mutator that could override validated URLs or security headers.

### 1.4 Terminology

| Term | Meaning in this guide |
|---|---|
| **host** | The embedding application, such as Pinocchio, that constructs Geppetto engines. |
| **credential store** | A host-selected secret-bearing persistence implementation. Geppetto supplies store contracts and safe lifecycle helpers; Pinocchio binds its direct-YAML profile extension. Pi’s auth file is one optional external import source, not a Geppetto data format. |
| **renewable bearer** | An access token plus an optional refresh token and expiry, acquired at request time through a host source. |
| **transport contract** | The full endpoint, path, HTTP method, payload, headers, response stream, and retry behavior expected by a provider. |
| **provider adapter** | Trusted Go-only code that binds a provider route resolver, restricted request/response middleware, credential source, and optional stream codec to a shared Geppetto engine core. |
| **engine core** | Geppetto’s shared provider-protocol implementation, such as OpenAI Responses or Anthropic Messages, which owns request construction, URL validation, send/retry orchestration, and stream consumption. |
| **metadata** | Non-token companion values such as an account identifier. It is still private runtime state and must not be shown to JavaScript or diagnostics. |

## 2. Current-state architecture

### 2.1 Geppetto credential boundary

`pkg/steps/ai/credentials/bearer.go:21-83` defines `credentials.Request`, `Credential`, `Store`, `Refresher`, and `BearerTokenSource`. The public source interface returns exactly one string:

```go
// pkg/steps/ai/credentials/bearer.go
type BearerTokenSource interface {
    BearerToken(context.Context, Request) (string, error)
}
```

The narrowness is intentional. The generic `RenewableBearerTokenSource` loads credentials through a host `Store`, refreshes through a host `Refresher`, saves a rotation before caching it, and emits redacted availability errors. It does not parse profile YAML or know provider token endpoints. Its request identity is the non-secret pair `{Provider, BaseURL}`.

```text
             host owns secrets and provider protocol

  +----------------+       +------------------+       +----------------+
  | host Store     |<----->| host Refresher   |<----->| provider OAuth |
  +--------+-------+       +------------------+       +----------------+
           |                         ^
           v                         |
  +-------------------------------+  |
  | RenewableBearerTokenSource    |--+
  | cache + singleflight + redact |
  +---------------+---------------+
                  |
                  | bearer at outbound request time
                  v
  +---------------+---------------+
  | Geppetto engine               |
  | validated URL -> request      |
  +-------------------------------+
```

The existing source is appropriate only if the provider request needs a normal bearer header and the target engine understands the protocol.

### 2.2 Supported engine paths

`pkg/inference/engine/factory/factory.go:134-151` forwards a configured bearer source to OpenAI Chat and OpenAI Responses. `pkg/steps/ai/openai/chat_stream.go:96-105,133-168` and `pkg/steps/ai/openai_responses/streaming.go:148-185` resolve the source after URL validation and set `Authorization: Bearer ...`. Each path permits one opt-in pre-stream 401 refresh/replay using `UnauthorizedBearerTokenSource`.

The factory intentionally does **not** forward the source to Claude. `pkg/steps/ai/claude/engine_claude.go:72-86` looks up a static API key. `pkg/steps/ai/claude/api/completion.go:94-99` sends that value as `x-api-key` and sets the Anthropic version header. This matters: Claude is not merely another OpenAI-compatible bearer path.

### 2.3 JavaScript boundary

`pkg/js/modules/geppetto/api_engine_builder.go:63-73` keeps a host-configured bearer source inside `moduleRuntime` in Go. JavaScript receives an engine builder, not the source. That same boundary remains mandatory for any richer provider-specific capability:

```text
JavaScript settings/profile -> Go engine construction -> Go-only source/adapter -> HTTP request
                                 ^
                                 |
                    no token, header map, account id,
                    refresh callback, or selector crosses here
```

### 2.4 Pi credential behavior

Pi’s `AuthStorage` uses provider registrations to log in, stores OAuth credential records, and locks its backing file while refreshing expired records (`dist/core/auth-storage.js:318-425`). This is useful evidence for the current local client, but it is not a stable Geppetto interface. Geppetto must not import Pi’s JavaScript implementation or directly adopt Pi’s disk format.

## 3. Provider contract classification

### 3.1 OpenAI Codex: renewable OAuth, custom ChatGPT transport

Pi’s OpenAI Codex provider is explicitly bound to `https://chatgpt.com/backend-api` (`dist/providers/openai-codex.js:6-16`). Its OAuth implementation supports PKCE authorization-code login and device-code login, refreshes through the OpenAI OAuth token service, and derives a ChatGPT account identifier from an access-token claim (`dist/utils/oauth/openai-codex.js:22-35,112-125,319-338`).

The inference request is not a normal public OpenAI Responses request. Pi resolves a `/codex/responses` path and adds all of the following in its Codex response implementation (`dist/api/openai-codex-responses.js:402-409,1163-1210`):

- the bearer authorization header;
- a ChatGPT account header derived from the current access token;
- an originator header;
- experimental Responses/SSE headers;
- request/session identifiers where applicable.

| Required capability | Existing Geppetto OpenAI Responses | Codex requirement | Result |
|---|---:|---:|---|
| Request-time bearer | Yes | Yes | Reusable concept |
| One pre-stream 401 replay | Yes | Likely useful | Must be contract-tested |
| `/responses` target | Yes | `/codex/responses` | Incompatible without a dedicated target strategy |
| Account header | No | Required by Pi transport | Missing capability |
| Originator/beta headers | No | Required by Pi transport | Missing capability |
| Codex request/stream mapping | Standard Responses | Pi-specific implementation | Must be audited, not assumed identical |

**Conclusion:** do not configure the generic OpenAI Responses engine with a ChatGPT base URL and a Pi bearer. That would route a token to a validated but semantically wrong target and omit companion headers. Treat Codex as a distinct transport adapter.

### 3.2 Anthropic Claude subscription: renewable OAuth, Anthropic Messages transport

Pi’s Anthropic provider implements PKCE authorization-code exchange and refresh-token renewal (`dist/utils/oauth/anthropic.js:159-185,290-315`). The scopes include inference and Claude Code-session capability. This proves a refreshable client flow exists in Pi.

Geppetto already speaks the Anthropic Messages family through the Claude engine, which is promising. However, it currently carries a static key through a client field and sends `x-api-key` (`pkg/steps/ai/claude/engine_claude.go:72-86`, `pkg/steps/ai/claude/api/completion.go:94-99`). It has no request-time source and no validated OAuth-specific header profile.

**Conclusion:** Claude is not blocked by an absent protocol engine; it is blocked by absent dynamic credential plumbing and unverified subscription-auth request semantics. First add a fake-server test that states the exact headers and endpoint expected by the approved provider contract. Only then design `WithBearerTokenSource`-like behavior for Claude. Never assume an access token can be substituted into `x-api-key` because both are strings.

### 3.3 Umans: API-key persistence shim, Anthropic Messages transport

The installed Umans Pi extension asks for an API key, copies it into access/refresh-shaped values, gives it a very long expiry, and returns it unchanged from “refresh” (`pi-provider-umans/index.ts:539-577`). Its own README says the gateway uses Anthropic Messages at `/v1/messages`, with `anthropic-version`, rather than OpenAI Chat Completions (`README.md:39-43`).

**Conclusion:** Umans is not evidence for refresh-token support. It is an API-key configuration use case. It may later be compatible with Geppetto’s Claude/Messages engine after a protocol test, but it belongs to static-key/provider compatibility work and must not drive the renewable OAuth API.

### 3.4 Additional candidate: Kimi Code

The installed Kimi Code extension implements device authorization and refresh token grants and can select an OpenAI-like coding endpoint. It is relevant evidence that Pi extensions can carry extra model/transport metadata beyond `access`, `refresh`, and `expires`. It is deliberately out of the first implementation scope because its custom stream handler and protocol-specific headers require the same evidence-first evaluation as Codex.

## 4. Gap analysis

### 4.1 Why `BearerTokenSource` is insufficient for Codex

A bearer source answers only “what token may I send to this already-decided request?” Codex also needs a specialized request target and companion headers derived from the current credential. The token-only source deliberately cannot express those capabilities.

Adding an arbitrary `func(*http.Request)` callback to settings would be unsafe:

- settings are profile-derived, serializable, cloneable, and inspectable;
- an arbitrary callback could rewrite a URL after outbound validation and exfiltrate a bearer;
- a callback could override `Host`, `Authorization`, `Content-Length`, or tracing headers;
- JavaScript could accidentally gain a way to select or observe private transport behavior.

### 4.2 Why Geppetto must not read Pi’s auth file

Reading Pi’s file from Geppetto would make a reusable library depend on a local tool’s private schema, lock implementation, migration behavior, and provider registrations. Geppetto should instead provide the reusable provider protocol and lifecycle mechanics while the host chooses credential placement, user interaction, and consent policy.

The correct dependency direction is:

```text
Pinocchio or another host
  ├── binds a Geppetto store to its direct-YAML/profile location
  ├── launches browser or presents device code and renders CLI status
  ├── decides whether a Pi record may be explicitly imported/migrated
  └── injects Go-only runtime capability into Go and JavaScript engine builders

Geppetto
  ├── owns supported provider PKCE, exchange, refresh, status, and local-delete primitives
  ├── validates outbound target before credential release
  ├── implements provider protocol and bounded replay behavior
  └── never discovers host credential files, chooses a profile, or launches a browser
```

### 4.3 Why no generic “OAuth provider” abstraction

OAuth tells us how to acquire and refresh tokens. It does not define the inference HTTP protocol. The provider needs an adapter when any of these vary:

- endpoint host or path;
- header names and mandatory metadata;
- message wire format and streaming events;
- model discovery or model aliases;
- request retry/idempotency semantics;
- refresh behavior after a server-side rejection.

A generic `OAuthProvider` in Geppetto would collapse these concerns and invite unsafe configuration. Keep protocol code explicit.

## 5. Proposed architecture

### 5.1 Architectural decision

Implement reusable provider lifecycle packages and provider transports in Geppetto, then bind them to host-selected stores and UI. Each lifecycle package owns its supported provider’s OAuth/API-key protocol behavior; each host owns where credentials live, how a login is presented, and whether another application’s record may be imported.

Do **not** turn the current generic OpenAI Responses engine into a configurable Codex client by adding profile keys for endpoint suffixes or header maps.

```text
                     host-owned binding and interaction policy
 +----------------------------------------------------------------+
 | Pinocchio or another application                                |
 |  - direct-YAML, keychain, database, or in-memory Store binding  |
 |  - profile/provider selection, consent, CLI/browser UI          |
 |  - optional explicit Pi migration                               |
 +-------------------------------+--------------------------------+
                                 |
             typed Store + browser/device-code callbacks; no settings/JS secrets
                                 v
 +-------------------------------+--------------------------------+
 | Geppetto credentials/providers                                  |
 |  - PKCE/state, authorization URL, code exchange, refresh        |
 |  - locking/atomic rotation helpers, redacted Status, Delete     |
 |  - typed private runtime credential metadata                     |
 +-------------------------------+--------------------------------+
                                 |
                   Go-only typed capability, never settings/JS
                                 v
 +-------------------------------+--------------------------------+
 | Geppetto shared engine cores + provider adapters                |
 |  OpenAI Responses core + Codex route/middleware                 |
 |  Anthropic Messages core + Claude auth middleware               |
 |  Anthropic Messages core + Umans API-key middleware             |
 |  OpenAI engines + ordinary bearer middleware                    |
 +-------------------------------+--------------------------------+
                                 |
                         validated outbound HTTP request
                                 v
                           provider inference API
```

### Decision: Geppetto owns reusable lifecycle; hosts own binding and policy

- **Context:** OAuth exchange/refresh and request contracts are provider behavior worth sharing, while disk location, profile schema, browser launch, and consent are application behavior.
- **Options considered:** Put all lifecycle protocol in every host; parse Pi storage in Geppetto; provide provider lifecycle packages plus host-injected storage/UI bindings.
- **Decision:** Geppetto provides provider-specific lifecycle packages, generic lifecycle/store helpers, and provider engines. Hosts supply a selected `Store`, browser/device-code presentation, selected identity, and import policy.
- **Rationale:** This avoids duplicating PKCE/refresh/status/delete mechanics in every Go application without coupling Geppetto to Pi or Pinocchio storage.
- **Consequences:** Geppetto needs stable, documented interfaces for store operations and interactive login callbacks. Pinocchio can retain its direct-YAML extension as a store adapter and CLI binding. A future host can instead use a keychain or database without reimplementing provider protocol.
- **Status:** proposed.

### Decision: Codex is a trusted middleware adapter over the Responses core

- **Context:** Pi’s Codex transport has a different request path and required companion headers, but may share request framing and streaming semantics with OpenAI Responses.
- **Options considered:** Set a custom base URL in OpenAI Responses; add arbitrary profile header maps; copy a dedicated Codex engine; add a restricted provider route/middleware adapter to the shared Responses core.
- **Decision:** Add a Go-only Codex route resolver and restricted request/response middleware to the Responses core after fake-server tests prove which request and SSE behavior is shared.
- **Rationale:** The core retains URL validation, request send/retry orchestration, cancellation, and stream ownership. The Codex middleware contains the few Codex-only headers and typed credential behavior, avoiding both hundreds of duplicated lines and an unsafe public request-mutator API.
- **Consequences:** The middleware seam must be intentionally constrained and provider-installed; profile settings and JavaScript cannot configure it. A provider-specific stream codec remains available if Codex events differ.
- **Status:** proposed.

### Decision: Add dynamic Claude authentication only after header-contract proof

- **Context:** Geppetto has a Claude Messages engine but assumes static `x-api-key` authentication.
- **Options considered:** Reuse bearer source blindly as an API key; add dynamic source plus an explicit auth strategy; defer Claude.
- **Decision:** First write a fake-server contract for the accepted subscription request; then add a typed Claude auth strategy if the server behavior is confirmed.
- **Rationale:** Header semantics affect authorization and compatibility. A string token’s existence is insufficient evidence.
- **Consequences:** Claude implementation remains a separate phase from Codex. Token counting and all non-streaming Claude paths must be audited together.
- **Status:** proposed.

### Decision: Treat Umans as static-key/Messages work

- **Context:** The installed extension performs no token exchange or rotation.
- **Options considered:** Model it as OAuth because of field names; adapt it through renewable bearer; keep it in API-key protocol work.
- **Decision:** Keep Umans out of renewable OAuth scope.
- **Rationale:** The implementation evidence says refresh is a no-op and the API is Anthropic Messages.
- **Consequences:** Any Umans support should validate the Messages protocol and API-key header behavior, with no refresh tests or lifecycle claims.
- **Status:** proposed.

### 5.2 Lifecycle, store, and interaction contracts

The existing `BearerTokenSource` stays unchanged for ordinary OpenAI-compatible providers. Do not break every embedding host by replacing it. Add new interfaces only with a concrete consumer and tests.

Geppetto should expose a small provider-lifecycle surface that hosts can compose instead of reimplementing token handling:

```go
// Proposed: package credentials
// Values are never formatted by these APIs and must remain out of settings/JS.
type Store interface {
    Load(context.Context, Key) (Credential, error)
    Save(context.Context, Key, Credential) error // atomically replaces rotations
    Delete(context.Context, Key) error
}

type Status struct {
    State     State // Missing, Ready, Expiring, Expired, RefreshFailed
    ExpiresAt time.Time
    Renewable bool
}

type ProviderFlow interface {
    BeginLogin(context.Context, LoginRequest) (AuthorizationRequest, error)
    CompleteLogin(context.Context, LoginCompletion) (Credential, error)
    Refresh(context.Context, Credential) (Credential, error)
}

func Login(ctx context.Context, store Store, key Key, flow ProviderFlow, presenter Presenter) (Status, error)
func StatusOf(ctx context.Context, store Store, key Key, now time.Time) (Status, error)
func Logout(ctx context.Context, store Store, key Key) error
```

`Presenter` is deliberately host-controlled: it can open a browser, display a URL/device code, or integrate with another UI. The flow validates PKCE state and the exact callback redirect; it does not launch a browser. `Logout` is local deletion. A separately named, provider-specific `Revoke` operation may be added only after client-authentication and endpoint semantics are documented.

`Store` describes persistence semantics, not a required serialization format. Geppetto should provide an in-memory implementation and reusable locking/atomic-update helpers. A filesystem implementation, if added, must require an explicit caller-owned path and codec; it must never discover a profile file or default to Pi storage. Pinocchio’s direct YAML extension remains a host adapter over this contract.

### 5.3 Go-only route and request/response middleware contracts

The engine core owns the HTTP request and response body. A provider adapter receives neither a mutable request URL/body nor a stream body. This avoids a general `func(*http.Request)` escape hatch while allowing a few credential-dependent headers to be injected consistently across OpenAI and Anthropic engines.

```go
// Proposed: package transport
// RequestContext is read-only and contains the already validated final URL.
type RequestContext struct {
    Provider  string
    Operation string
    URL       url.URL
}

type HeaderWriter interface {
    Set(name, value string) error // permits only engine-declared header names
}

type Middleware interface {
    BeforeRequest(context.Context, RequestContext, HeaderWriter) (Attempt, error)
    AfterResponse(context.Context, RequestContext, Attempt, ResponseMetadata) (ResponseDecision, error)
}

type RouteResolver interface {
    Resolve(baseURL *url.URL, operation string) (*url.URL, error)
}
```

The core calls `RouteResolver.Resolve`, validates the resulting final URL, builds the request, then runs `BeforeRequest`. `AfterResponse` receives status and safe response metadata before any stream output is emitted; it can request a single bounded retry but cannot consume, replace, or rewrite the response body. The engine owns close/retry/decode behavior.

`HeaderWriter` is configured with a static allowlist supplied by the engine/provider adapter. It rejects `Host`, transfer/framing headers, and undeclared names; values are never formatted into errors, trace events, or JavaScript values. Middleware is installed only through typed Go engine options, never from profile settings or JavaScript.

A Codex middleware is constructed only from a typed `CodexCredentialSource`. It resolves the current bearer and private account metadata, sets the sanctioned Codex header names, and requests a forced credential refresh only for the first pre-stream 401. The source and opaque `Attempt` state never cross the engine boundary. A Geppetto provider lifecycle package derives needed metadata from its current credential or provider result; the host store persists it only if the typed provider record requires it.

### 5.4 Codex middleware flow pseudocode

```go
// Proposed: package providers/openaicodex
func (m *CredentialMiddleware) BeforeRequest(
    ctx context.Context, req transport.RequestContext, headers transport.HeaderWriter,
) (transport.Attempt, error) {
    credential, err := m.source.CodexCredential(ctx, credentials.Request{
        Provider: "openai-codex", BaseURL: req.URL.Scheme + "://" + req.URL.Host,
    })
    if err != nil {
        return nil, credentials.RedactError(err)
    }
    if err := headers.Set("Authorization", "Bearer "+credential.bearerToken); err != nil { return nil, err }
    if err := headers.Set("chatgpt-account-id", credential.accountID); err != nil { return nil, err }
    if err := headers.Set("originator", codexOriginator); err != nil { return nil, err }
    if err := headers.Set("OpenAI-Beta", codexResponsesBeta); err != nil { return nil, err }
    return newAttempt(credential), nil // opaque and never logged
}

func (m *CredentialMiddleware) AfterResponse(
    ctx context.Context, req transport.RequestContext, attempt transport.Attempt, response transport.ResponseMetadata,
) (transport.ResponseDecision, error) {
    if response.StatusCode != http.StatusUnauthorized || response.StreamStarted {
        return transport.Continue, nil
    }
    return m.source.ForceRefreshAfterUnauthorized(ctx, attempt) // core permits one retry total
}

// Inside the shared Responses core:
target := adapter.Route.Resolve(baseURL, "responses")
validateOutboundURL(target)                 // before BeforeRequest
request := buildResponsesRequest(ctx, target, input)
attempt := middleware.BeforeRequest(ctx, requestContext(target), request.headers)
response := client.Do(request)
decision := middleware.AfterResponse(ctx, requestContext(target), attempt, responseMetadata(response))
return core.retryOnceOrDecodeResponsesStream(response, decision)
```

Key invariants:

1. The route resolver produces the Codex target and the core validates the final URL before credential middleware runs.
2. Codex header injection is in trusted middleware installed only for the Codex provider adapter, never in profiles or JavaScript.
3. `HeaderWriter` allowlists the Codex header names; no middleware can alter URL, host, framing, request body, or stream body.
4. Never include a token, account value, raw provider response, or refresh error in a returned error.
5. The core, not middleware, closes/replays the request exactly once on a pre-output 401.

### 5.5 Claude and Umans middleware flow

Claude and Umans use the same engine-level middleware seam after their request contracts are proven. Claude’s future OAuth middleware resolves only the approved dynamic authentication form and requests refresh after an eligible 401. Umans’ static API-key middleware resolves its host-provided key and sets only its approved Anthropic Messages header form; it never advertises refresh support.

The Anthropic Messages core must invoke the middleware on streaming, non-streaming, and token-count requests. Do not implement the middleware until a fake-server contract establishes the exact allowed header names, required beta/client headers, and retry semantics. As with Codex, it is a typed Go engine option and cannot be supplied by settings or JavaScript.

## 6. Implementation plan

### Phase 0 — Define the reusable lifecycle API and freeze provider evidence

**Files to create/update**

- This ticket’s `sources/01-local-pi-and-geppetto-source-map.md`.
- A provider-contract fixture directory, for example `pkg/steps/ai/openai_codex/testdata/`.
- A provider decision record in the implementation PR.

**Actions**

1. Pin the exact Pi and extension version used as evidence in test documentation.
2. Extract only public protocol constants and sanitized fixture shapes; never save a local credential file.
3. Write acceptance criteria per provider:
   - target path;
   - required static and dynamic header names (not values);
   - request body expectations;
   - SSE event mapping;
   - token refresh/replay behavior;
   - all prohibited logging/serialization paths.
4. Write lifecycle-contract tests for `Store`, `Login`, `StatusOf`, local `Logout`, and a fake `Presenter`; verify that provider flow code never selects a file path or writes UI output.

**Exit criterion:** fake-server tests and lifecycle tests can be written without an account credential or a Pinocchio profile.

### Phase 1 — Middleware and Codex contract tests before implementation

**Likely files**

- `pkg/steps/ai/transport/middleware.go` (new, or an existing internal transport package)
- `pkg/steps/ai/openai_responses/*_test.go`
- `pkg/steps/ai/providers/openaicodex/*_test.go` (new)
- `pkg/inference/engine/factory/factory_test.go`

**Tests**

- a route resolver resolves the Codex response path and cannot escape the configured host;
- the core validates the final URL before it calls middleware;
- `HeaderWriter` permits declared names and rejects URL/host/framing/undeclared-header mutation;
- a Codex middleware cannot be installed for ordinary OpenAI and an ordinary bearer source cannot construct Codex middleware;
- a current credential yields one request; an expired credential performs one locked refresh and persistence;
- first pre-stream 401 requests one retry; a second 401 or post-output error does not;
- account metadata and bearer never occur in errors, trace events, or JS values;
- JavaScript builder works only when the host installs the provider adapter in Go, with no exposure API.

**Exit criterion:** fake servers validate middleware ordering, the complete outbound request shape, and simulated stream behavior without an account credential.

### Phase 2 — Implement the shared engine middleware seam and Codex adapter

**Likely files**

- `pkg/steps/ai/transport/middleware.go` (new)
- `pkg/steps/ai/openai_responses/streaming.go`
- `pkg/steps/ai/openai_responses/engine.go`
- `pkg/steps/ai/providers/openaicodex/route.go` (new)
- `pkg/steps/ai/providers/openaicodex/middleware.go` (new)
- `pkg/inference/engine/factory/factory.go`

**Implementation notes**

- The Responses core owns final URL validation, request construction, HTTP execution, retry bounds, response-body close, and stream decode.
- Codex owns only the route resolver, typed credential/header middleware, and an optional stream codec if tests prove Responses SSE is not compatible.
- Keep middleware a typed Go engine option. It must not become `InferenceSettings.API` data, a profile header map, or a JavaScript value.
- Make model selection explicit and conservative; model aliases change more frequently than the transport boundary.

**Exit criterion:** unit and race tests pass with fake stores/refreshers, a fake middleware, and a fake inference service.

### Phase 3 — Implement Geppetto lifecycle/provider packages

This phase belongs in Geppetto. It turns the current generic renewable-bearer mechanics into composable lifecycle primitives and adds supported provider protocol modules.

**Responsibilities**

- provide `Store` semantics, in-memory/fake stores, and locking/atomic-rotation helpers;
- provide redacted `StatusOf` and idempotent local `Logout`/delete;
- provide typed PKCE/state/authorization-code flow support with host-provided presentation;
- implement only verified provider modules: OpenAI/Codex, Anthropic, and Umans API-key configuration;
- map a provider lifecycle result into a Go-only engine source without serializing it into profiles;
- preserve rotated credential persistence before returning refreshed access;
- inject the source into both Go and JavaScript engine construction paths in Go.

**Do not do**

- parse the entire Pi auth file as a convenient generic store;
- make an `auth.json` path configurable in profile YAML;
- have Geppetto choose a host’s credential path, profile, browser behavior, or consent policy;
- use JavaScript to choose the local credential record;
- automatically reuse a current user’s Pi login without an explicit host import/migration policy.

**Exit criterion:** provider and lifecycle integration tests use an in-memory or temporary fake store and fake presenter; they never use the real local credential file.

### Phase 4 — Bind Pinocchio’s direct-YAML lifecycle and optional Pi migration

This phase belongs in Pinocchio or another host repository.

**Responsibilities**

- adapt the existing `extensions."pinocchio.oauth@v1"` direct-YAML persistence to Geppetto’s `Store` contract while retaining Pinocchio’s locking and atomic write guarantees;
- implement the existing Glazed login/status/logout commands by calling Geppetto lifecycle APIs and formatting only redacted status;
- select profile/provider and browser presentation in Pinocchio, never in Geppetto;
- make any Pi import an explicit user-directed migration into Pinocchio-owned storage, with no silent linking or shared mutable ownership.

**Exit criterion:** Pinocchio tests prove the selected profile receives a host-owned store binding and neither Go nor JavaScript exposes credential capability.

### Phase 5 — Claude audit and dynamic-auth design

**Likely files**

- `pkg/steps/ai/claude/engine_claude.go`
- `pkg/steps/ai/claude/api/completion.go`
- `pkg/steps/ai/claude/api/messages.go`
- `pkg/steps/ai/claude/token_count.go`
- `pkg/inference/engine/factory/factory.go`

Audit every outbound Claude path before adding source support. The token-count path is independently static today, so changing only streaming inference would create inconsistent authentication behavior.

**Exit criterion:** an approved, evidence-backed Claude request-auth strategy passes fake-server coverage for Messages, streaming, and token counting. If no stable contract is available, defer rather than guessing.

### Phase 6 — Optional controlled live smoke

A real smoke is permitted only after a reviewer accepts the provider’s current contract and the host owner explicitly approves use of a selected account. The smoke must:

- use a non-destructive minimal request;
- send no credentials to terminal output, test logs, source files, ticket documents, or reMarkable PDFs;
- record only success/failure class, HTTP status category, provider version, and redacted timing;
- avoid persistent profile modifications;
- include a clear rollback/logout/removal plan.

## 7. Testing and validation strategy

### 7.1 Unit tests

| Area | Assertions |
|---|---|
| Lifecycle/store | login state/PKCE validation, atomic rotation-before-return, idempotent local delete, redacted status, cancellation isolation, no token-bearing errors |
| Credential source | cache hit, expiry, refresh skew, rotated refresh persistence, cancellation isolation, no token-bearing errors |
| Route resolver | normalized base URL produces only approved provider target path before validation |
| Middleware headers | required keys present; undeclared, host, URL, framing, and body mutation rejected; no source headers in event/debug data |
| Middleware response | exactly one core-owned replay before output; middleware cannot consume stream bodies or request a second replay |
| Claude/Umans middleware | request-time resolution applies identically to Messages and token count |
| Factory | source presence authorizes only engines that explicitly support it |
| JavaScript | builder succeeds with Go source; script cannot receive source, token, headers, expiry, or account metadata |

### 7.2 Fake-server integration tests

Use `httptest.NewTLSServer` and an injected HTTP client. The handler should capture only request method, path, header **names**, and a sanitized body shape. Never make test failures print full headers.

```go
func TestCodexFirst401RefreshesOnce(t *testing.T) {
    server := newTLSServer(func(r *http.Request) response {
        assertPath(t, r, "/backend-api/codex/responses")
        assertHeaderPresent(t, r, "Authorization")
        assertHeaderPresent(t, r, "chatgpt-account-id")
        assertHeaderPresent(t, r, "originator")
        return sequence(http.StatusUnauthorized, sseSuccess())
    })

    source := newFakeCodexSource(/* stale then replacement; values never logged */)
    engine := newResponsesEngine(server.URL,
        withRoute(codexRoute{}),
        withMiddleware(newCodexMiddleware(source)),
    )
    _, err := engine.RunInference(context.Background(), userTurn("ping"))
    require.NoError(t, err)
    require.Equal(t, 2, server.RequestCount())
    require.Equal(t, 1, source.ForcedRefreshCount())
}
```

### 7.3 Commands

Run from the standalone Geppetto module:

```bash
cd /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto
GOWORK=off go test ./... -count=1
GOWORK=off go test -race ./pkg/steps/ai/credentials ./pkg/steps/ai/transport ./pkg/steps/ai/openai_responses ./pkg/steps/ai/providers/openaicodex ./pkg/steps/ai/claude ./pkg/inference/engine/factory ./pkg/js/modules/geppetto -count=1
GOWORK=off make logcopter-check
GOWORK=off make gosec
```

Replace the proposed `transport` and `providers/openaicodex` packages in the race command with their actual locations once implementation begins. A full repository race run remains useful but may contain unrelated baseline failures; report them separately rather than masking them.

## 8. Risks, alternatives, and open questions

### 8.1 Risks

- **Provider contract drift:** Pi implementation may change before Geppetto implementation. Pin versions and test a captured request shape.
- **Terms/account policy:** A token that Pi can use is not automatic approval for another application to use it. Require explicit host policy and user consent.
- **Header leakage:** Account headers and bearer values are both private runtime state. Treat them as sensitive even if they are not cryptographic secrets.
- **Partial coverage:** A dynamic source added only to streaming inference would leave token counting or non-streaming requests stale.
- **Unsafe abstraction:** Arbitrary profile header maps or mutable request callbacks create an endpoint-exfiltration path. The middleware must remain header-restricted, trusted, and Go-only.
- **Refresh races:** Two processes operating on the same external store need an agreed lock and atomic-write policy.
- **Overreaching lifecycle API:** A Geppetto file store that silently selects paths or a login helper that launches browsers would erase the necessary application-policy boundary.

### 8.2 Alternatives rejected

**Treat every Pi OAuth-shaped record as a `BearerTokenSource`.** Rejected because Umans demonstrates that field names do not establish a renewable OAuth protocol, and Codex needs more than a bearer.

**Embed Pi JavaScript or parse Pi storage in Geppetto.** Rejected because it violates language/runtime boundaries and turns a reusable provider-lifecycle library into a Pi-specific credential owner.

**Add `headers` and `path` maps to profiles.** Rejected because settings are serializable and user-controlled; these maps would make it easy to alter security-sensitive request behavior and leak credentials.

**Use the generic OpenAI Responses engine for Codex immediately through base-URL/profile configuration.** Rejected because the installed Pi code explicitly uses a different endpoint path and required headers. The revised proposal instead adds a tested trusted route/middleware adapter to the shared core.

### 8.3 Open questions for review

1. Is there a supported, stable provider policy for the ChatGPT Codex backend outside Pi/Codex clients?
2. What exact Claude subscription request headers and endpoint behavior should Geppetto support, if any?
3. Should Pinocchio offer an explicit one-time Pi import, or should Pi use require a dedicated local credential broker?
4. Does Codex share enough request/SSE behavior to use the Responses core with a route/middleware adapter, or does it require an optional provider stream codec?
5. Which model aliases and request fields are mandatory for a valid Codex request beyond endpoint and headers?
6. What user consent and audit trail are required before an application reuses a local subscription credential?

## 9. Intern checklist

Before changing a line of production code, an intern should be able to answer all of these:

- Which host-selected store owns the persistent credential? If the answer is “Geppetto discovers profile YAML,” stop: that is wrong.
- Which shared engine core sends the actual request, and which trusted provider route/middleware adapter is installed? Read URL construction, header allowlists, retry loop, and tests.
- Does the provider use an OpenAI protocol, Anthropic Messages, or a custom one? Do not infer this from a model name.
- Are all required headers known from source or provider documentation? Record the source and write a fake-server assertion.
- Is the final URL validated before a token source is called?
- Does refresh persist a rotated refresh token before returning a replacement access token?
- Can JavaScript inspect any credential-adjacent data? It must not.
- Does every outbound path use the same dynamic-auth strategy, including token counting?
- Is a real account smoke explicitly approved and secret-safe? If not, use fixtures only.

## 10. References

### Geppetto source

- `pkg/steps/ai/credentials/bearer.go:21-83,124-167,265-297`
- `pkg/inference/engine/factory/factory.go:134-151,205-260`
- `pkg/steps/ai/openai/chat_stream.go:96-105,133-168`
- `pkg/steps/ai/openai_responses/streaming.go:148-185`
- `pkg/steps/ai/claude/engine_claude.go:72-86`
- `pkg/steps/ai/claude/api/completion.go:94-99`
- `pkg/js/modules/geppetto/api_engine_builder.go:63-73`
- Existing design: `ttmp/2026/07/10/GEPPETTO-REFRESHABLE-CREDENTIALS-387--refreshable-bearer-credentials-for-openai-compatible-providers/design-doc/01-refreshable-bearer-credential-source-analysis-design-and-implementation-guide.md`

### Local Pi/extension evidence

- `sources/01-local-pi-and-geppetto-source-map.md`
- `/home/manuel/.nvm/versions/node/v22.22.1/lib/node_modules/@earendil-works/pi-coding-agent/docs/providers.md`
- `/home/manuel/.nvm/versions/node/v22.22.1/lib/node_modules/@earendil-works/pi-coding-agent/node_modules/@earendil-works/pi-ai/dist/utils/oauth/openai-codex.js`
- `/home/manuel/.nvm/versions/node/v22.22.1/lib/node_modules/@earendil-works/pi-coding-agent/node_modules/@earendil-works/pi-ai/dist/api/openai-codex-responses.js`
- `/home/manuel/.nvm/versions/node/v22.22.1/lib/node_modules/@earendil-works/pi-coding-agent/node_modules/@earendil-works/pi-ai/dist/utils/oauth/anthropic.js`
- `/home/manuel/.pi/agent/npm/node_modules/pi-provider-umans/index.ts`
- `/home/manuel/.pi/agent/npm/node_modules/pi-provider-umans/README.md`
