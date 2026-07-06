---
Title: Investigation diary
Ticket: LLM-PROXY-BYOK-GEPETTO
Status: active
Topics:
    - byok
    - geppetto
    - llm-proxy
    - security
    - inference
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Chronological diary for the geppetto-dependencies ticket, starting from the assessment of the LLM-PROXY-BYOK implementation and the two geppetto-side blockers it surfaced.
LastUpdated: 2026-07-06T11:35:00-04:00
WhatFor: Preserve the investigation trail so future sessions can resume the geppetto SSRF escape hatch and include_usage work without re-deriving context.
WhenToUse: Read before resuming work on this ticket; append an entry per work session.
---

# Investigation diary

## Goal

Track the geppetto-side dependencies that block or limit the BYOK effort: the SSRF guard that prevents local-provider testing, and the missing `stream_options.include_usage` streaming support.

## Step 1: Ticket creation from the BYOK assessment

This ticket was created during a post-implementation assessment of the LLM-PROXY-BYOK work. The assessment reviewed the byok-host prior-art tickets, the llm-proxy implementation diaries (Steps 1–6), the PROJ note, and the current code state (builds clean, tests green, 0 TODOs, ~3,800 LOC, branch `task/llm-proxy-byok` local-only). Two geppetto-side concerns were identified as worth their own ticket because they span both the geppetto and llm-proxy repos and are not pure llm-proxy changes.

### Prompt Context

**User prompt (verbatim):** "create a geppetto ticket to address the issues with using geppetto packaging to test, and for include_usage maybe?"

**Assistant interpretation:** Create a docmgr ticket (in the llm-proxy workspace) tracking the geppetto-side blockers: the SSRF guard that blocks local plain-HTTP fake providers in tests, and the missing `stream_options.include_usage` streaming support.

**Inferred user intent:** Separate the geppetto-cross-cutting concerns from the llm-proxy-internal hardening so each can be scoped and picked up independently.

### What I did
- Verified the geppetto SSRF guard source: `pkg/security/outbound_url.go` defines `ValidateOutboundURL(rawURL, OutboundURLOptions{AllowHTTP, AllowLocalNetworks})` — the primitive is well-designed and supports opt-in.
- Confirmed every LLM provider call site hard-codes `AllowHTTP: false`: `pkg/steps/ai/claude/api/completion.go:91,:152`, `pkg/steps/ai/openai_responses/engine.go:104`, `pkg/steps/ai/openai_responses/token_count.go:72`.
- Confirmed `pkg/embeddings/ollama.go:52` already opts into `AllowHTTP: true, AllowLocalNetworks: true` — proving the mechanism is reachable, just not exposed for LLM providers.
- Created ticket `LLM-PROXY-BYOK-GEPETTO` with design doc and this diary.
- Wrote the design doc covering both issues with file/line-anchored evidence, proposed profile-level opt-in for the SSRF hatch, and the llm-proxy-side `include_usage` implementation plan.

### Why
- These two concerns are not pure llm-proxy changes: the SSRF hatch requires a geppetto change (threading `OutboundURLOptions` from settings), and `include_usage` spans the geppetto streaming result and the llm-proxy SSE frame. They deserve a focused ticket rather than being buried in a generic hardening list.

### What worked
- The geppetto source confirmed the diagnosis from the LLM-PROXY-BYOK diary Step 3 verbatim: the primitive supports opt-in, the callers just hard-code `false`.

### What didn't work
- N/A. This was a ticket-creation and design-writing step.

### What I learned
- The Ollama embeddings provider is the existing precedent for opting into local HTTP — the fix is "do what Ollama does, but make it profile-configurable for LLM providers."

### What was tricky to build
- Scoping the `include_usage` work correctly: the usage data already reaches llm-proxy via `result.Usage`, so the gap is purely llm-proxy's SSE frame emission, not a geppetto streaming change. Getting that boundary right keeps the geppetto side of this ticket small (verify only).

### What warrants a second pair of eyes
- Whether the SSRF opt-in should be two separate booleans (`AllowHTTP`, `AllowLocalNetworks`) matching the existing struct, or a single `dev: true` profile flag. The design doc leans separate to match the primitive.

### What should be done in the future
- Implement the geppetto-side settings threading (Step 1 of the design doc plan).
- Consider filing a geppetto GitHub issue for the SSRF escape hatch if the geppetto maintainers want to track it upstream (the glazed issues were filed; geppetto was kept as a docmgr ticket per the user's request).

### Code review instructions
- Start with the design doc: `design-doc/01-geppetto-dependencies-ssrf-escape-hatch-and-streaming-include-usage.md`.
- Verify the call-site evidence: `rg -n "AllowHTTP" geppetto/pkg/`.

### Technical details
- Geppetto repo: `/home/manuel/workspaces/2026-07-05/llm-proxy-byok/geppetto` (remote `go-go-golems/geppetto`).
- llm-proxy streaming plumbing point: `pkg/runtime/chat_service.go` (the goroutine that owns `result.Usage`).

## Step 2: Implement profile-owned outbound URL validation opt-ins

The first Geppetto-side blocker is implemented. Provider URL validation still denies plain HTTP and local-network targets by default, but a profile can now explicitly opt in through `inference_settings.api.allow_http` and `inference_settings.api.allow_local_networks`. The opt-in is keyed by provider/API type and supports both raw provider keys (for YAML readability, e.g. `openai: true`) and suffixed keys (for compatibility with Geppetto's existing `openai-base-url` / `openai-api-key` naming pattern).

The change removes the hard-coded `AllowHTTP: false` literals from OpenAI-compatible Chat Completions, OpenAI Responses, and Claude provider paths. This unblocks local fake-provider testing without weakening the default SSRF posture.

### Prompt Context

**User prompt (verbatim):** "push and PR. Then work on GEPETTO changes (move that ticket to geppetto/)"

**Assistant interpretation:** Push the llm-proxy BYOK branch and open a PR, move the Geppetto follow-up ticket from the llm-proxy docmgr workspace into the Geppetto repo, then implement the Geppetto-side changes starting with the outbound URL validation escape hatch.

**Inferred user intent:** Keep follow-up ownership in the correct repo and turn the assessment ticket into concrete Geppetto code, not just planning.

**Commit (code):** ece5bb07 — "security: make provider URL validation opt-in configurable"

### What I did
- Moved `LLM-PROXY-BYOK-GEPETTO` from `llm-proxy/ttmp` into `geppetto/ttmp` and added the missing `byok` vocabulary topic.
- Added `APISettings.AllowHTTP` and `APISettings.AllowLocalNetworks` maps.
- Added `settings.OutboundURLOptions` / `OutboundURLOptionsForKeys` helper functions.
- Threaded the helper through:
  - `pkg/steps/ai/openai/chat_stream.go`
  - `pkg/steps/ai/openai_responses/engine.go`
  - `pkg/steps/ai/openai_responses/token_count.go`
  - `pkg/steps/ai/claude/engine_claude.go`
  - `pkg/steps/ai/claude/token_count.go`
  - `pkg/steps/ai/claude/api/{completion.go,messages.go}` via `Client.SetOutboundURLOptions`.
- Added tests for default-deny behavior, YAML decoding, suffixed/raw provider keys, Responses alias fallback, OpenAI Chat local HTTP opt-in, and Claude client options.
- Ran focused tests and then full pre-commit validation.

### Why
- The BYOK llm-proxy integration needs a way to test the real provider HTTP path against local fake providers without globally weakening SSRF protections.
- A profile-level opt-in keeps production defaults secure and scopes any relaxation to the profile that explicitly asks for it.

### What worked
- `GOWORK=off go test ./...` passed.
- Geppetto's pre-commit hook passed: full tests plus lint/glazed-lint vet tooling.
- The existing `OutboundURLOptions` primitive was sufficient; no changes to `pkg/security/outbound_url.go` were needed.

### What didn't work
- The first `git commit` attempt timed out because the pre-commit hook runs the full repository test/lint suite and exceeded the initial 120-second command timeout. I re-ran the same commit with a longer timeout; the hook completed successfully and the commit landed.
- A focused test run initially failed because I removed the `security` import from `openai/chat_stream.go`, but the file still calls `security.ValidateOutboundURL`. Restored the import and re-ran the focused tests successfully.

### What I learned
- OpenAI Responses needs alias-aware option resolution just like its API key and base URL resolution. `responsesOutboundURLOptions` now checks `open-responses`, `openai-responses`, then `openai`.
- The cleanest YAML shape is provider-keyed maps:
  - `api.allow_http.openai: true`
  - `api.allow_local_networks.openai: true`
  while still accepting suffixed keys like `openai-allow-http` for naming consistency.

### What was tricky to build
- The tricky boundary was not the security primitive; it was threading the option into all provider paths without accidentally broadening defaults. Claude wraps URL validation inside an API client, so the engine and token counter now set `Client.SetOutboundURLOptions(...)` immediately after constructing the client. OpenAI Responses has multiple aliases, so it needed a small package-local helper rather than a single API type lookup.

### What warrants a second pair of eyes
- Whether maps keyed by provider/API type are the right long-term config shape, or whether the project eventually wants a nested structure under each base URL. The current shape matches existing `api_keys` / `base_urls` conventions.
- Whether CLI flag support should be added later via wildcard `glazed` maps. This implementation focuses on profile YAML support because the immediate need is local-provider test profiles.

### What should be done in the future
- Add an operator smoke with a local plain-HTTP fake provider and a profile that opts into `allow_http` and `allow_local_networks`.
- Continue with `stream_options.include_usage` after the SSRF/local-test path is proven end-to-end.

### Code review instructions
- Start with `pkg/steps/ai/settings/outbound_url.go` and `pkg/steps/ai/settings/settings-inference.go`.
- Then review call sites in OpenAI Chat, OpenAI Responses, and Claude.
- Validate with:
  - `GOWORK=off go test ./pkg/steps/ai/settings ./pkg/steps/ai/openai ./pkg/steps/ai/openai_responses ./pkg/steps/ai/claude/... ./pkg/security`
  - `GOWORK=off go test ./...`

### Technical details
- YAML example:
  ```yaml
  api:
    base_urls:
      openai-base-url: http://127.0.0.1:9999/v1
    allow_http:
      openai: true
    allow_local_networks:
      openai: true
  ```
- Responses alias order: `open-responses`, `openai-responses`, `openai`.
