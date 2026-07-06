---
Title: 'Geppetto dependencies: SSRF escape hatch and streaming include_usage'
Ticket: LLM-PROXY-BYOK-GEPETTO
Status: active
Topics:
    - byok
    - geppetto
    - llm-proxy
    - security
    - inference
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-07-05/llm-proxy-byok/geppetto/pkg/security/outbound_url.go
      Note: ValidateOutboundURL + OutboundURLOptions — the SSRF guard primitive
    - Path: /home/manuel/workspaces/2026-07-05/llm-proxy-byok/geppetto/pkg/steps/ai/claude/api/completion.go
      Note: Hard-codes AllowHTTP: false at the Claude provider call sites (lines 91, 152)
    - Path: /home/manuel/workspaces/2026-07-05/llm-proxy-byok/geppetto/pkg/steps/ai/openai_responses/engine.go
      Note: Hard-codes AllowHTTP: false at the OpenAI Responses call site (line 104)
    - Path: /home/manuel/workspaces/2026-07-05/llm-proxy-byok/geppetto/pkg/embeddings/ollama.go
      Note: Already opts into AllowHTTP+AllowLocalNetworks — proves the mechanism is reachable
    - Path: /home/manuel/workspaces/2026-07-05/llm-proxy-byok/llm-proxy/pkg/runtime/chat_service.go
      Note: Streaming goroutine that owns result.Usage — include_usage plumbing point
ExternalSources: []
Summary: Two geppetto-side blockers for the BYOK effort: (1) the SSRF guard hard-codes AllowHTTP:false on every LLM provider path with no per-profile escape hatch, blocking local plain-HTTP fake providers in tests; (2) stream_options.include_usage is unsupported, so streaming clients cannot observe usage on the wire.
LastUpdated: 2026-07-06T11:30:00-04:00
WhatFor: Capture the geppetto dependencies that block local-provider testing and on-wire streaming usage, with concrete fix proposals for both geppetto and llm-proxy.
WhenToUse: Read when unblocking local-provider smoke tests or implementing stream_options.include_usage.
---

# Geppetto dependencies: SSRF escape hatch and streaming include_usage

## Executive Summary

Two geppetto-side concerns block or limit the BYOK effort and were discovered during the LLM-PROXY-BYOK implementation (diary Steps 3 and 5). This ticket tracks resolving both:

1. **SSRF escape hatch for local testing.** Geppetto's `security.ValidateOutboundURL` is well-designed — it takes an `OutboundURLOptions{AllowHTTP, AllowLocalNetworks}` struct — but every LLM provider call site hard-codes `AllowHTTP: false` with no way to override it per profile or per provider. This makes it impossible to point a profile at a local plain-HTTP fake provider for smoke testing. (The Ollama embeddings provider already opts into `AllowHTTP: true, AllowLocalNetworks: true`, proving the mechanism is reachable — it is just not exposed for LLM providers.)
2. **`stream_options.include_usage`.** Streaming responses carry no usage object on the wire. The authoritative token counts exist only in the `InferenceResult` returned by `geppettoengine.RunInferenceWithResult`. The OpenAI streaming API defines `stream_options.include_usage` to emit a final SSE frame carrying `usage`; llm-proxy does not implement it, so streaming BYOK consumers cannot observe their own spend on the wire (they can only read it via the control-plane usage API after the fact).

## Problem Statement

### 1. SSRF guard blocks local-provider testing

During LLM-PROXY-BYOK Phase 2, the planned tmux smoke test against a local plain-HTTP fake provider failed:

```
invalid chat completion URL: http scheme is not allowed
```

Root cause: every LLM provider path in geppetto calls `security.ValidateOutboundURL(url, security.OutboundURLOptions{AllowHTTP: false})` with a hard-coded literal. There is no env var, flag, or profile setting to relax this. The relevant call sites:

- `geppetto/pkg/steps/ai/claude/api/completion.go:91` and `:152` — `AllowHTTP: false`
- `geppetto/pkg/steps/ai/openai_responses/engine.go:104` — `AllowHTTP: false`
- `geppetto/pkg/steps/ai/openai_responses/token_count.go:72` — `AllowHTTP: false`

Contrast with `geppetto/pkg/embeddings/ollama.go:52`, which legitimately sets `AllowHTTP: true, AllowLocalNetworks: true` because Ollama runs locally. The primitive supports the opt-in; the LLM providers just never wire it up.

**Impact:** the only way to test the full HTTP → middleware → service → engine → metering chain against a fake upstream is an in-process fake engine implementing `EngineWithResult` (which is what `pkg/byok/integration_test.go` does). That is excellent for CI but does not exercise the real HTTP client path, the real SSE parser, or the real provider URL handling. A local HTTP fake provider remains valuable for integration and operator smoke testing.

### 2. `stream_options.include_usage` unsupported

The OpenAI Chat Completions streaming API supports `stream_options: { include_usage: true }`, which causes the server to emit a final SSE chunk whose `usage` field is populated. llm-proxy currently ignores this option. The proxy's SSE chunks have no `usage` field, and the authoritative numbers exist only in the `InferenceResult` returned inside the streaming goroutine after the stream completes (`pkg/runtime/chat_service.go`).

**Impact:** BYOK consumers that stream cannot observe their own token usage on the wire. They must poll the control-plane `/api/usage` endpoint after the fact. This is a documented gap in the LLM-PROXY-BYOK diary (Step 3, "What should be done in the future") and the PROJ note's open questions.

## Proposed Solution

### 1. SSRF escape hatch — profile-level opt-in

Thread `OutboundURLOptions` from profile settings so a profile can opt into `AllowHTTP` and/or `AllowLocalNetworks`. This mirrors how the Ollama embeddings provider already works and keeps the default secure (deny HTTP + local networks).

**Geppetto side:**
- Add optional fields to the provider/connection settings struct that feeds `ValidateOutboundURL`, e.g. `AllowHTTP bool` and `AllowLocalNetworks bool` on the relevant `ClientSettings`/`API` settings, defaulting to `false`.
- Replace the hard-coded `security.OutboundURLOptions{AllowHTTP: false}` literals at the LLM provider call sites with a value derived from those settings.
- Keep the default `false` so production behavior is unchanged unless a profile explicitly opts in.

**llm-proxy side:**
- Expose the two booleans as profile YAML fields (e.g. under `api:` or a new `security:` block) so a dev profile can point at `http://127.0.0.1:PORT`.
- Document the dev-only nature loudly; this must never be the default for a profile that resolves a real provider key.

**Alternative / complement — in-process fake engine:**
- The in-process fake `EngineWithResult` pattern (`pkg/byok/integration_test.go`) remains the preferred CI approach because it is deterministic and needs no port. This ticket should also document that pattern as the canonical test seam so future contributors do not re-derive it.

### 2. `stream_options.include_usage` — emit a final usage frame

**llm-proxy side (primary):**
- In the streaming handlers (`chat_service.go`, `completion_service.go`), inspect the request for `stream_options.include_usage` (and the legacy `include_usage` if relevant).
- After the upstream stream completes and `result.Usage` is available (the same point where `meter.Recorder` is called), emit one final SSE chunk carrying the `usage` object, then the `data: [DONE]` terminator.
- The usage object must be mapped to the OpenAI wire shape (prompt_tokens, completion_tokens, total_tokens, and cached token fields where applicable), reusing the same mapping the meter uses to avoid drift.

**Geppetto side (verify only):**
- Confirm `RunInferenceWithResult` reliably returns `result.Usage` for streaming completions across all engines (it does for the engines llm-proxy uses today). No geppetto change is expected unless an engine fails to populate usage.

## Design Decisions

- **Profile-level opt-in over a global dev flag.** A global `--allow-http-providers` flag is too broad: it would relax the guard for every profile. Per-profile opt-in matches the existing Ollama precedent and keeps blast radius small.
- **Default remains deny.** The hard-coded `false` is the correct production default. The escape hatch is strictly additive for dev/test profiles.
- **include_usage is llm-proxy's responsibility.** The usage data already reaches llm-proxy via `result.Usage`; the gap is purely that llm-proxy does not serialize it into a final SSE frame. No geppetto streaming change is required.
- **Reuse the meter's usage mapping.** The ledger and the on-wire usage frame must agree, so both should derive from the same `turns.InferenceUsage` → wire-usage helper.

## Alternatives Considered

- **Global env var `GEPPETTO_ALLOW_HTTP=true`.** Rejected: too broad, and it would silently weaken security for every profile in the process.
- **HTTPS test server with a trusted self-signed cert.** Considered for testing. Rejected as the primary path: it adds cert-management overhead to every smoke loop and does not help operators who legitimately want to point a profile at a local gateway. Viable as an additional option but not a substitute for the opt-in.
- **Skip on-wire usage; rely on the control-plane usage API.** Rejected: streaming clients (especially third-party sites in Phase 4) should be able to observe their own spend without a second round-trip to a different plane.

## Implementation Plan

1. **Geppetto: thread OutboundURLOptions from settings.** Add `AllowHTTP`/`AllowLocalNetworks` to the provider settings structs; replace the hard-coded literals at the Claude, OpenAI Responses, and OpenAI Chat call sites. Default false. Add unit tests asserting the opt-in is honored and the default still denies.
2. **llm-proxy: expose profile security fields.** Add YAML fields; wire them into the settings that reach the engine. Add a dev profile in `examples/` pointing at `http://127.0.0.1`.
3. **llm-proxy: implement `stream_options.include_usage`.** Parse the request option; after stream completion, emit a final usage SSE chunk then `[DONE]`. Add an integration test asserting the final frame carries usage matching the ledger row.
4. **Document the in-process fake-engine test seam** in this ticket's reference docs so the canonical CI pattern is discoverable.
5. **Operator smoke:** stand up a local plain-HTTP fake provider and drive the full HTTP path with a dev profile, confirming the SSRF opt-in works end to end.

## Open Questions

- Should the `AllowLocalNetworks` opt-in be separately gated from `AllowHTTP`, or combined into a single `dev: true` profile flag? (Lean: keep them separate to match the existing `OutboundURLOptions` shape.)
- Should `include_usage` also be emitted for the OpenAI Responses API streaming shape, or only Chat Completions? (Lean: both, where the upstream API defines it.)
- Does any geppetto engine fail to populate `result.Usage` for streaming? (Verify during implementation; none known today.)

## References

- LLM-PROXY-BYOK diary Step 3 (SSRF blocker) and Step 5 (include_usage future work)
- PROJ note open questions: `stream_options.include_usage` support
- Geppetto: `pkg/security/outbound_url.go`, `pkg/steps/ai/claude/api/completion.go`, `pkg/steps/ai/openai_responses/engine.go`, `pkg/embeddings/ollama.go`
- llm-proxy: `pkg/runtime/chat_service.go`, `pkg/byok/meter/meter.go`, `pkg/byok/integration_test.go`
