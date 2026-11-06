---
Title: 'Bound connections: atomic credential ownership and request execution'
Ticket: GEPPETTO-CREDSCOPE-001
Status: active
Topics:
    - security
    - credentials
    - providers
    - geppetto
DocType: design-doc
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: repo://pkg/steps/ai/connection/auth.go
      Note: Request-time authentication and refresh
    - Path: repo://pkg/steps/ai/connection/connection.go
      Note: Opaque request capability and shared executor
    - Path: repo://pkg/steps/ai/connection/connection_test.go
      Note: Adversarial ownership and routing tests
ExternalSources: []
Summary: ""
LastUpdated: 2026-11-06T17:05:09.738684029-05:00
WhatFor: ""
WhenToUse: ""
---

# Bound connections: atomic credential ownership and request execution

## Intent and scope

PR #445 introduced origin checks at the transport boundary. This second design makes the endpoint, authentication, URL policy and client a single immutable runtime capability. Code that encodes a chat, embedding, rerank or decision payload should not have to look up a key, choose a fallback URL, attach secrets or remember to pin a client.

The implementation uses existing configuration inputs (Pi entries and settings slots) as the declaration boundary. The named connections and secret URIs in the conversation were illustrative: no secret URI service exists here. A new registry or secret store is unnecessary to establish the runtime invariant. Existing settings are configuration input, never the authority to rebind an already constructed connection. There is no parallel legacy runtime path or compatibility adapter.

## Current failure surfaces

- `settings.APISettings` exposes independent `APIKeys` and `BaseUrls` maps.
- `openai/chat_stream.go` resolves those maps per request and carries raw keys beside a URL and client.
- Responses resolves the same slot repeatedly; its route resolver can choose another origin, after which the engine pins to that chosen origin.
- Claude creates a mutable API client with separate key, URL and HTTP client setters.
- Pi protects a client and source, then puts raw Anthropic/Gemini keys into public resolved settings.
- Embeddings combine top-level keys and URLs with local overrides independently. Pinning this merged result authorizes the accidental pair.
- Gemini inference already uses the modern `google.golang.org/genai` client, but obsolete legacy client setup still describes an independent cache client.
- Direct provider constructors can bypass factory-only guards.

## Runtime design

Add `pkg/steps/ai/connection` with an opaque `Connection`. A trusted configuration boundary calls `New(Config)` once with provider identity, declared base URL, authentication, outbound policy and host HTTP client. All fields of the constructed connection are private. Configuration, headers and the client struct are copied. A nil transport is captured at construction, rather than being re-read from mutable global state at send time.

`Authentication` is also opaque. Factories supply static tokens, renewable token sources, Anthropic modes, arbitrary declared headers, or the existing restricted transport middleware for Codex. Credential lookup and refresh remain at request time. Middleware can only write declared headers; it cannot replace a destination or request body. Existing source-specific scope checks remain authoritative: binding never widens a source's permitted origins.

```go
conn, err := connection.New(connection.Config{
    Provider: provider,
    BaseURL: declaredEndpoint,
    Auth: authentication,
    HTTPClient: hostClient,
    OutboundURL: policy,
})

request, err := conn.NewRequest(ctx, "POST", "chat/completions", body)
response, err := conn.Do(request)
```

`NewRequest` accepts a relative operation path. `Do` also accepts SDK-created requests, but validates their actual origin and outbound policy before consulting authentication. Authentication runs on a cloned request. The connection owns at most one pre-stream unauthorized replay with a replayable body and per-request attempt state. Concurrent requests never share mutable attempt state.

A private, guarded network transport checks every redirect hop. Host redirect callbacks cannot retarget an authenticated request outside the declaration's origin. `Host` overrides are rejected. Userinfo, query and fragment in a base URL are rejected: authentication has an explicit carrier and is never embedded in diagnostics or configuration-derived request URLs.

The origin is fixed from the declared source endpoint, before protocol routing. A provider route must remain in this origin; resolving a route does not authorize its origin. URI path boundaries are not an origin security boundary; tenant-specific path scopes remain a separate feature.

## SDK integration

`SDKClient()` returns an HTTP client facade whose private transport executes through the connection. The facade exposes neither the original client nor its transport. The connection owns redirects and authentication; an outer facade does not follow redirects a second time. SDKs receive an inert nonempty API-key placeholder where their constructor requires a key. The connection overwrites the placeholder with real authentication immediately before the network boundary. No engine or SDK holds the real key.

Modern Gemini generation and cache operations must share this client. Remove unused legacy client setup and verify the installed modern SDK's cache path with a recording transport. No SDK operation is permitted to fall back to an ambient client.

## Source resolution and merge rules

Settings slots are translated to an immutable connection at engine construction. The Responses precedence rule remains source selection: choose one slot, then obtain both fields from that slot. Engines accept an explicit `WithConnection` for Pi and other host-owned bindings; this bypasses settings credential resolution entirely. Ordinary model and inference settings remain separate.

Pi resolves provider endpoint, credentials and custom headers once, builds the connection, and captures it in the engine builder. Mutating `Resolved.Settings.API` afterward cannot move the credential. Anthropic and Gemini credentials no longer enter public resolved settings. Codex retains its canonical route and typed source, installed as restricted connection middleware.

For embeddings, a layer that overrides either credential field selects that layer's pair for that provider. A partial override is an error rather than implicit inheritance. Empty entries do not count as overrides. Likewise per-call credential overrides require a pair. Callers wanting to change only model/dimensions can continue to do so without affecting the connection. Adapter defaults are resolved only at the declaration boundary, before binding.

Factories pass host timeout/proxy/TLS settings and policy into the constructor; provider implementations use the connection. Direct constructors establish the same boundary and cannot bypass protection by skipping a factory.

## Guarantees and trust limits

- An existing binding cannot be repointed by changing a settings map, SDK route, redirect or header middleware.
- Secrets are absent from the encoder/engine request and diagnostic copies; only the connection and trusted credential sources own them.
- Unauthorized refresh is bounded, occurs before decoding and cannot replace the declared destination.
- Configuration must truthfully declare a credential's intended endpoint. A static API key does not encode its issuer's origin.
- The host's injected network transport, credential source and Go middleware are trusted code. An arbitrary transport can rewrite a URL after validation or send its own requests; no in-process Go wrapper can sandbox it.
- Origin checks do not pin DNS, validate proxy operators or enforce tenant path isolation.

## Implementation and commit plan

1. Commit this design and diary checkpoint.
2. Implement connection ownership, authentication, safe SDK integration and adversarial unit tests; commit.
3. Integrate engines, token counting and Pi so payload code consumes connections; commit.
4. Integrate remaining providers, remove independent credential-layer merges and extend cross-path regressions; commit.
5. Update help, design status and detailed diary; run relevant tests, race checks for the connection, full repository build/tests/lint and downstream checks; commit.

## Validation

Recording transports and marker credentials are used throughout; no real API requests are needed. Tests cover initial foreign destinations before credential acquisition, a route resolver choosing a foreign origin, redirects of each relevant status, hostile redirect callbacks, same-origin redirects, Host overrides, configuration mutation after binding, arbitrary secret headers, fresh credentials, one unauthorized replay, non-replayable bodies, cancellation, concurrent requests, SDK requests and the modern Gemini cache client. Existing 38 matrix rows must continue to exercise their intended paths, with new rows for token counting, direct constructors and source mutation. Embeddings tests must assert refusal of partial cross-layer pairs.

## Implementation status

Design checkpoint only. The subsequent diary entries and completion section will record the delivered behavior and exact validation results.
