---
Title: 'Refreshable bearer credential source: analysis, design, and implementation guide'
Ticket: GEPPETTO-REFRESHABLE-CREDENTIALS-387
Status: active
Topics:
    - oauth
    - credentials
    - inference
DocType: design-doc
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: repo://pkg/inference/engine/factory/factory.go
      Note: |-
        Factory validation and option propagation
        Source-aware validation and option propagation
    - Path: repo://pkg/steps/ai/credentials/bearer.go
      Note: |-
        Public renewable bearer source contract and cache/refresh implementation
        Public host-injected credential contracts, caching, refresh, persistence, and redaction
    - Path: repo://pkg/steps/ai/credentials/bearer_test.go
      Note: Renewal invariants and concurrent refresh evidence
    - Path: repo://pkg/steps/ai/openai/chat_stream.go
      Note: |-
        OpenAI-compatible Chat request-time bearer resolution and header injection
        Chat request-time credential lookup after outbound URL validation
    - Path: repo://pkg/steps/ai/openai_responses/engine.go
      Note: OpenAI Responses request-time bearer resolution
    - Path: repo://pkg/steps/ai/openai_responses/provider_settings.go
      Note: Responses request-time source and static fallback
ExternalSources:
    - https://github.com/go-go-golems/geppetto/issues/387
Summary: Design and implementation record for host-owned renewable bearer credentials used at outbound OpenAI-compatible inference request time.
LastUpdated: 2026-07-10T20:15:00-04:00
WhatFor: Implement, operate, and review expiring OAuth access credentials without storing refresh material in profiles.
WhenToUse: Use when embedding Geppetto with OAuth-backed OpenAI-compatible providers or another host-owned renewable bearer credential system.
---


# Refreshable bearer credential source: analysis, design, and implementation guide

## 1. Executive summary

Geppetto historically models provider authentication as a static string in `InferenceSettings.API.APIKeys`. That is correct for conventional long-lived API keys, but it fails for OAuth-backed OpenAI-compatible providers: an access token expires while a long-running process continues using the old profile value. The visible symptom is a provider `401` even though the host still possesses refresh material and could obtain a new access token.

This implementation adds a public, host-injected credential subsystem at `pkg/steps/ai/credentials`. It retains OAuth refresh-token storage and provider-specific protocol requests in the host application. Geppetto asks the source for a usable bearer token immediately before an outbound OpenAI Chat or OpenAI Responses request. The supplied renewable implementation performs an inexpensive in-memory validity check on normal requests, refreshes only when expiry is inside a configurable skew, persists a rotated credential through the host store, and collapses concurrent refreshes for one provider/base-URL identity.

The central security boundary is deliberate: refresh tokens never become fields in profile YAML, `InferenceSettings`, profile debug output, observability data, or Geppetto logs. Hosts opt in through engine/factory options. Existing static API-key profiles retain current behavior when no source is configured.

## 2. Problem statement and scope

### 2.1 What failed in the motivating integration

A real llm-proxy BYOK smoke routed an `umans-flash` request through Geppetto’s OpenAI-compatible engine. The profile contained a static bearer string that was no longer accepted by the upstream provider. Replacing the local value with the current host credential made both ordinary and SSE inference succeed. The proxy, model mapping, encrypted BYOK credential, and SSE parser were therefore not the cause; static credential lifetime was.

Manual profile editing is an emergency recovery mechanism, not a suitable runtime design. It requires an operator to know that a token expired, copy sensitive data, and restart or re-resolve configuration. It also couples every host to a mutable profile file even when the host already has a secure credential store.

### 2.2 In-scope behavior

This work covers OpenAI-compatible inference request paths:

- `pkg/steps/ai/openai` Chat Completions streaming path;
- `pkg/steps/ai/openai_responses` Responses streaming path;
- `pkg/inference/engine/factory` construction and validation for those engines;
- a generic renewable bearer source that hosts can use with their own credential storage and OAuth refresher.

The source calls are request-time. A healthy cached token performs only local map/mutex/time operations; it does **not** contact an OAuth server for every inference request.

### 2.3 Explicit non-goals

This release intentionally does **not**:

- read `~/.pi/agent/auth.json` or any other host credential file;
- implement Umans, OpenAI, Anthropic, or another vendor’s OAuth refresh endpoint;
- serialize `refresh_token`, `access_token`, or expiry data into engine profile YAML;
- persist credentials in Geppetto;
- retry a completed request after provider `401` automatically;
- extend Claude/Gemini/embedding/transcription paths in this first slice;
- provide a browser OAuth login flow.

The non-goals keep Geppetto a reusable inference library. A host knows where its credentials live, how they are encrypted, and which OAuth endpoint/client policy applies. Geppetto knows only how to request a usable bearer string at the boundary where it creates an outbound HTTP request.

## 3. Current-state architecture, with evidence

### 3.1 Settings are static configuration

`pkg/steps/ai/settings/settings-inference.go:43-67` defines:

```go
type APISettings struct {
    APIKeys  map[string]string
    BaseUrls map[string]string
    AllowHTTP map[string]bool
    AllowLocalNetworks map[string]bool
}
```

The maps merge and clone as profile settings. This is a good representation for non-renewable configuration but is unsuitable for refresh material: it is serializable, copied by `Clone`, and used by profile/debug tooling. The new source is deliberately not a field on `APISettings`.

### 3.2 Factory validation previously required static key presence

`pkg/inference/engine/factory/factory.go:119-140` selects an engine from `Chat.ApiType`. `validateOpenAISettings` at approximately lines 203-230 previously required `<api_type>-api-key` (with Responses aliases) before engine construction. That would reject a valid host-injected OAuth source before its first request.

The factory now stores an optional `credentials.BearerTokenSource`. For OpenAI-compatible engines, source presence authorizes construction without a static key; required base-URL validation remains unchanged. Static validation remains exactly in force when no source is supplied.

### 3.3 Chat Completions snapshots a profile key

Before this change, `pkg/steps/ai/openai/chat_stream.go:54-80` read the static API key into `chatStreamConfig.apiKey`. `openChatCompletionStream` then set `Authorization: Bearer <apiKey>` at lines 83-102. The key was selected once per inference but could not be refreshed.

The path now has this ordering:

```text
settings base URL
  -> ValidateOutboundURL
  -> ensure HTTP client
  -> BearerTokenSource.BearerToken(ctx, Request)
  -> construct HTTP request and set Authorization header
```

URL validation happens before source invocation. This prevents an unvalidated/malicious endpoint configuration from causing a host source to release a bearer credential.

### 3.4 OpenAI Responses has an independent path

`pkg/steps/ai/openai_responses/engine.go` builds `POST /responses` and used `responsesAPIKey` from `provider_settings.go`. `streaming.go:147-155` writes the static bearer header. The new option and request-time resolver cover this path too. The Responses request uses the canonical `open-responses` provider identity when aliases such as `openai` or `openai-responses` normalize to it.

### 3.5 Existing redaction is a constraint, not a solution

`pkg/cli/bootstrap/profile_introspection.go:442-449` recognizes keys containing `token`, `secret`, `credential`, `authorization`, and API-key variants as sensitive. That protects profile-facing diagnostic structures. It would not make a refresh token safe to add to `APISettings`; avoiding serialized settings is the stronger invariant.

## 4. Terms and security model

| Term | Meaning | Secret? |
| --- | --- | --- |
| access token | Bearer value sent in an outbound `Authorization` header | Yes |
| refresh token | Long-lived credential presented to a provider token endpoint to obtain a replacement access token | Yes |
| credential | Access token, optional refresh token, and access-token expiry held by the host store | Yes |
| provider request | Non-secret `Provider`/`BaseURL` identity used to choose a credential | No, though hosts should avoid embedding user info in URLs |
| source | `BearerTokenSource`, which returns an access token at outbound request time | Interface, not secret |
| store | Host adapter that loads/saves credentials securely | Host-owned |
| refresher | Host adapter that implements vendor OAuth/refresh protocol | Host-owned |
| skew | Interval before nominal expiry during which the token is proactively refreshed | No |

The flow is intentionally split between three owners:

```text
+----------------------+        +-----------------------+
| Host application     |        | Geppetto              |
|----------------------|        |-----------------------|
| encrypted credential | Load   | Renewable source      |
| store                |<------>| cache + expiry check  |
| OAuth refresher      | Refresh| singleflight          |
| provider-specific    |<------>| request-time token    |
+----------------------+        +-----------+-----------+
                                            |
                                            | Authorization: Bearer <access>
                                            v
                                   +-----------------------+
                                   | Provider endpoint     |
                                   | OpenAI-compatible API |
                                   +-----------------------+
```

Geppetto does not log a `Credential`, a header, a source return value, or a host error. The public source reduces host errors to `ErrUnavailable` with a provider and operation; it intentionally does not wrap a potentially secret-bearing error string.

## 5. Proposed public API

### 5.1 Minimal engine-facing interface

The provider engines need only this small interface:

```go
type BearerTokenSource interface {
    BearerToken(ctx context.Context, request Request) (string, error)
}

type Request struct {
    Provider string
    BaseURL  string
}
```

A host may implement this directly if it already owns an OAuth cache. The method is called before each outbound provider request; it should normally return a cached valid access token in constant local time.

### 5.2 Reusable renewable implementation

`RenewableBearerTokenSource` composes host-owned persistence and refresh adapters:

```go
type Credential struct {
    AccessToken  string
    RefreshToken string
    ExpiresAt    time.Time
}

type Store interface {
    Load(ctx context.Context, request Request) (Credential, error)
    Save(ctx context.Context, request Request, credential Credential) error
}

type Refresher interface {
    Refresh(ctx context.Context, request Request, previous Credential) (Credential, error)
}

func NewRenewableBearerTokenSource(
    store Store,
    refresher Refresher,
    opts ...RenewableOption,
) (*RenewableBearerTokenSource, error)
```

`Store.Save` is required after a successful refresh. OAuth servers can rotate refresh tokens; using a new access token without persisting its paired refresh value can strand the application on the next expiry. On save failure, this implementation returns a redacted failure and does not cache the new credential.

### 5.3 Engine and factory integration APIs

Direct constructors opt in with provider-specific options:

```go
openai.NewOpenAIEngine(settings,
    openai.WithBearerTokenSource(source),
)

openai_responses.NewEngine(settings,
    openai_responses.WithBearerTokenSource(source),
)
```

The normal application-level path uses one factory option:

```go
factory := enginefactory.NewStandardEngineFactory(
    enginefactory.WithBearerTokenSource(source),
)
engine, err := factory.CreateEngine(resolvedSettings)
```

The source is authoritative when configured. This avoids a subtle fallback in which a refresh failure silently sends an old static bearer. When no source is configured, the preexisting static API-key behavior remains in place.

## 6. Detailed runtime flow

### 6.1 Normal cache-hit request

```text
RunInference(ctx)
  |
  +-> validate configured outbound URL
  |
  +-> source.BearerToken(ctx, {provider, baseURL})
        |
        +-> cached credential usable after skew? -- yes --> access token
  |
  +-> POST provider request with Authorization header
```

Pseudocode:

```go
credential, ok := source.cache[key]
if ok && credential.Usable(now(), refreshSkew) {
    return credential.AccessToken // no OAuth network traffic
}
```

### 6.2 Expired or near-expiry request

```text
caller A                       caller B..N
--------                       -----------
BearerToken()                  BearerToken()
  |                               |
  +--> singleflight key <---------+
  |                               wait, with each caller's context
  v
Load current credential
  |
usable after skew? -- no --> Refresher.Refresh(previous)
                                  |
                                  v
                         Store.Save(rotated credential)
                                  |
                                  v
                            cache + return access token
```

Pseudocode:

```go
result := group.DoChan(key, func() {
    current := cacheOrStoreLoad(key)
    if current.Usable(now, skew) { return current }
    replacement := refresher.Refresh(ctx, request, current)
    require(replacement.Usable(now, 0))
    store.Save(ctx, request, replacement)
    cache[key] = replacement
    return replacement
})

select {
case <-ctx.Done(): return "", ctx.Err()
case result := <-result: return result.AccessToken, result.Err
}
```

The waiting caller can cancel without changing the cache or interrupting its own wait. In the present implementation the first caller’s context drives the shared refresh operation; a future revision may add a source-owned bounded refresh context if deployments need refresh completion to outlive a canceled leader.

### 6.3 Failure behavior

- Missing provider identity: reject before a store/refresher call.
- Store load failure: return `ErrUnavailable{Operation: "credential load"}`.
- Refresh failure: return `ErrUnavailable{Operation: "credential refresh"}`.
- Refresher returns empty/expired credential: reject it.
- Store save failure: reject it and do not cache it.
- Source returns an error to an engine: engine returns a generic provider/source resolution error rather than wrapping arbitrary host error text.
- Static key with no source: preserve current static request behavior.

There is intentionally no automatic retry after `401`. An inference request may be billable or have observable side effects. A host can invalidate a source after an application-specific credential error, then make an explicit retry decision with its own idempotency policy.

## 7. Design decisions

### Decision: Keep refresh material out of `APISettings`

- **Context:** `APISettings` is YAML-backed, cloned, and consumed by diagnostics.
- **Options considered:** Add `refresh_token` fields to settings; place a runtime interface in settings; use an engine option.
- **Decision:** Use a runtime engine/factory option and a package-level interface.
- **Rationale:** The option is non-serializable and makes credential ownership explicit.
- **Consequences:** Hosts must wire the source during engine construction; profile-only configuration cannot enable refresh by itself.
- **Status:** accepted.

### Decision: Source is authoritative when configured

- **Context:** Falling back to a static API key after source failure can silently send stale/revoked credentials.
- **Options considered:** Source then static fallback; static then source; source-only when present.
- **Decision:** Source-only when present; static-only when absent.
- **Rationale:** Failure is visible, deterministic, and does not weaken the host’s intended credential policy.
- **Consequences:** Factory validation must accept an absent static key when source is configured.
- **Status:** accepted.

### Decision: Cache by provider plus base URL

- **Context:** Multiple OpenAI-compatible services often share `api_type: openai` but require distinct access tokens.
- **Options considered:** Key by API type only; key by model; key by provider/base URL.
- **Decision:** Key by normalized provider and base URL.
- **Rationale:** It separates services such as a custom OpenAI-compatible endpoint and OpenAI itself without binding to profile YAML or a model name.
- **Consequences:** Host stores need the same identity convention; callers should not include secret user-info in base URLs.
- **Status:** accepted.

### Decision: Persist before caching refreshed credentials

- **Context:** Refresh-token rotation means in-memory success alone can lose the only next refresh capability.
- **Options considered:** Cache then save asynchronously; save then cache; never persist in Geppetto.
- **Decision:** Invoke host `Save` synchronously, then cache only on success.
- **Rationale:** Fails closed and preserves recoverability after process restart.
- **Consequences:** Refresh latency includes host persistence. Hosts should make save bounded and durable according to their risk model.
- **Status:** accepted.

### Decision: No implicit 401 replay

- **Context:** A provider may have accepted work before returning/losing a response; retry can duplicate cost or effects.
- **Options considered:** Refresh and retry every 401; refresh only proactively; add host-controlled retry later.
- **Decision:** Proactive expiry refresh only.
- **Rationale:** It is safe by default and isolates credential validity from inference idempotency.
- **Consequences:** Hosts may explicitly invalidate/retry only where their application contract proves it safe.
- **Status:** accepted.

## 8. Implementation map

| File | Responsibility | Change |
| --- | --- | --- |
| `pkg/steps/ai/credentials/bearer.go` | Host-facing credential API and reusable cache/refresh source | New |
| `pkg/steps/ai/credentials/bearer_test.go` | Cache, expiry, persistence, concurrency, cancellation, redaction tests | New |
| `pkg/steps/ai/openai/engine_openai.go` | Chat engine state and request-time resolution call | Modified |
| `pkg/steps/ai/openai/observability.go` | `WithBearerTokenSource` engine option | Modified |
| `pkg/steps/ai/openai/chat_stream.go` | URL validation, static/source resolution, header injection | Modified |
| `pkg/steps/ai/openai_responses/engine.go` | Responses request-time source lookup | Modified |
| `pkg/steps/ai/openai_responses/observability.go` | Responses source option | Modified |
| `pkg/steps/ai/openai_responses/provider_settings.go` | Responses provider identity/base URL/static fallback | Modified |
| `pkg/inference/engine/factory/factory.go` | Factory option propagation and source-aware static-key validation | Modified |

## 9. Host integration example

A host owns the actual OAuth protocol and secret store. The following is pseudocode; it must use the host’s encrypted persistence, not profile YAML:

```go
store := &encryptedCredentialStore{ /* host-owned */ }
refresher := &umansOAuthRefresher{
    tokenURL: "https://provider.example/oauth/token",
    client:   hardenedHTTPClient,
}
source, err := credentials.NewRenewableBearerTokenSource(
    store,
    refresher,
    credentials.WithRefreshSkew(45*time.Second),
)
if err != nil { return err }

factory := enginefactory.NewStandardEngineFactory(
    enginefactory.WithBearerTokenSource(source),
)
engine, err := factory.CreateEngine(resolvedProfile.FinalInferenceSettings)
```

A refresher should:

1. use `previous.RefreshToken` only in a provider TLS request;
2. parse access token, optional rotated refresh token, and expiry;
3. retain the previous refresh token only if the provider explicitly does not return a replacement;
4. return errors with provider-safe categories, not response bodies containing credentials;
5. honor the supplied context deadline;
6. use an outbound URL pinned/configured by the host, not profile data.

## 10. Testing and validation strategy

### 10.1 Source unit tests

`bearer_test.go` covers:

- valid credential cache hit: one load, no refresh, no save;
- expired credential: refresh and save a rotated refresh token;
- concurrent expired callers: one refresh and one save;
- canceled waiter: returns `context.Canceled` while other callers can complete;
- host save failure: access/refresh values do not appear in errors and the unpersisted replacement is not cached.

### 10.2 Provider wiring tests

- OpenAI Chat engine test checks that a dynamic source wins over an intentionally different static key and receives `{Provider: "openai", BaseURL: ...}`.
- Responses settings test checks the canonical `open-responses` identity and verifies source errors are not copied into returned errors.
- Factory test deletes `openai-api-key`, injects a source, and proves engine construction plus outbound authorization succeeds.
- Existing OpenAI stream configuration tests retain static fallback coverage.

### 10.3 Required commands

```bash
cd /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto
gofmt -w pkg/steps/ai/credentials/*.go pkg/steps/ai/openai/*.go \
  pkg/steps/ai/openai_responses/*.go pkg/inference/engine/factory/*.go
GOWORK=off go test ./pkg/steps/ai/credentials ./pkg/steps/ai/openai \
  ./pkg/steps/ai/openai_responses ./pkg/inference/engine/factory -count=1
GOWORK=off go test ./... -count=1
GOWORK=off go test -race ./pkg/steps/ai/credentials ./pkg/steps/ai/openai \
  ./pkg/steps/ai/openai_responses ./pkg/inference/engine/factory -count=1
GOWORK=off make lint logcopter-check
GOWORK=off make gosec
GOWORK=off GOTOOLCHAIN=go1.26.5 go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

Run full `go test -race ./...` as a repository release gate too. If an unrelated baseline race fails, preserve its exact package/test/stack evidence in the diary rather than silently substituting the focused race command as an equivalent pass.

## 11. Operational guidance

### 11.1 Cost model

A valid cached token incurs no OAuth request. The normal per-inference cost is a mutex-protected lookup and time comparison. An OAuth refresh occurs only on first load with an expired credential, after expiry, inside refresh skew, or after an explicit cache invalidation.

### 11.2 Expiry skew

Use a skew that exceeds clock drift and expected request setup time. Thirty seconds is the default. Hosts with high request latency or short-lived tokens may choose a larger value; setting it too large causes needless refreshes.

### 11.3 Rotation and recovery

A successful refresh is not complete until the store persists the returned credential. Back up the host credential store according to its own security policy. Never put the credential store into Geppetto’s profile backup/export flow.

### 11.4 Revocation and provider 401

A host that learns a credential is revoked can call `Invalidate(request)` after updating/revoking persistent state. Geppetto will not retry the failed inference automatically. A host can decide to make an explicit retry only for a proven idempotent workflow.

## 12. Risks, alternatives, and follow-up work

### 12.1 Risks requiring review

- The first caller’s context is the shared refresh context. A canceled leader can cause a shared refresh to fail for other waiters; source-owned bounded refresh context is a possible future enhancement.
- `Provider + BaseURL` assumes host identity mapping is stable. Multi-tenant hosts may need a custom `BearerTokenSource` that includes an account identity outside this generic cache.
- The source trusts host adapters not to include secret values in errors. The built-in source suppresses host error text, but a custom source can violate that contract.
- Responses token-count and non-chat OpenAI features still use their own static-key configuration paths; they should be audited before claiming full provider-wide renewable authentication.

### 12.2 Alternatives rejected

**Mutable profile rewrite at expiry.** This is simple but unsafe for concurrent readers, loses source-of-truth ownership, and persists secrets in a configuration surface.

**Refresh token fields in YAML.** This makes long-lived credentials easy to serialize, clone, inspect, and accidentally commit.

**A global Geppetto credential singleton.** Global state causes test pollution and prevents applications from using separate accounts/providers safely.

**Automatic 401 retry.** It can duplicate inference cost and side effects.

### 12.3 Next implementation slices

1. Add a concrete host-side Umans adapter in the consuming application once its OAuth token endpoint and rotation semantics are documented.
2. Audit Responses token-count and other provider HTTP paths for optional source support.
3. Consider a source-owned bounded refresh context and metrics hooks.
4. Add integration coverage in llm-proxy with an injected source and a fake OAuth token endpoint; keep provider credentials out of browser and proxy logs.

## 13. References

### Geppetto source

- `pkg/steps/ai/settings/settings-inference.go:43-67` — static API configuration.
- `pkg/steps/ai/openai/chat_stream.go:54-122` — Chat endpoint validation, credential resolution, and bearer header injection.
- `pkg/steps/ai/openai/engine_openai.go` — Chat `RunInference` entry point.
- `pkg/steps/ai/openai_responses/engine.go` — Responses `RunInference` entry point.
- `pkg/steps/ai/openai_responses/provider_settings.go` — Responses aliases and base URL resolution.
- `pkg/steps/ai/openai_responses/streaming.go:147-155` — Responses HTTP bearer header.
- `pkg/inference/engine/factory/factory.go` — option propagation and settings validation.
- `pkg/cli/bootstrap/profile_introspection.go:442-449` — existing sensitive profile-key redaction.

### Tracking issue and downstream evidence

- [Geppetto #387 — Add refreshable bearer credentials for OpenAI-compatible providers](https://github.com/go-go-golems/geppetto/issues/387)
- `llm-proxy` ticket `LLM-PROXY-CROSS-DOMAIN-CHATBOT`, Steps 10–12 — real stale-Umans failure, profile refresh, and successful encrypted BYOK/SSE smoke.
