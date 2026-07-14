---
Title: Investigation diary
Ticket: GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS
Status: active
Topics:
    - geppetto
    - oauth
    - credentials
    - security
DocType: reference
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: repo://ttmp/2026/07/14/GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS--provider-specific-adapters-for-pi-subscription-credentials/design-doc/01-pi-subscription-credentials-in-geppetto-analysis-adapter-design-and-implementation-guide.md
      Note: Primary evidence-backed architecture and plan
    - Path: repo://ttmp/2026/07/14/GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS--provider-specific-adapters-for-pi-subscription-credentials/sources/01-local-pi-and-geppetto-source-map.md
      Note: Provider source evidence summary
ExternalSources: []
Summary: Chronological evidence for designing provider-specific Pi subscription credential support without exposing credential material.
LastUpdated: 2026-07-14T21:52:00-04:00
WhatFor: Record the research, design decisions, validation, and delivery of the Pi subscription credential adapter guide.
WhenToUse: Read before resuming implementation or assessing whether a provider contract is safe to support.
---


# Diary

## Goal

Capture the evidence and design required to evaluate Pi-managed provider credentials in Geppetto without turning a token-shaped local record into an assumed inference contract.

## Step 1: Establish provider contracts and Geppetto boundaries

The investigation began after recognizing that public marketing/API documentation was not sufficient evidence for the locally installed Pi providers. I examined Pi’s installed authentication and provider implementations using a redacted structural view of the auth store and source-code references, then compared those behaviors against Geppetto’s existing renewable bearer, OpenAI, Responses, Claude, factory, and JavaScript seams.

The result is a deliberately provider-specific design. OpenAI Codex and Claude have real renewable flows in Pi, but their transports do not fit the existing bearer-only OpenAI injection path unchanged. Umans’ Pi record looks OAuth-shaped but is an API key persisted through a no-op refresh shim. The guide makes these distinctions explicit and requires fake-server contracts before any real account request.

### Prompt Context

**User prompt (verbatim):** "Ok, create a new docmgr ticket (in geppetto) and analyze and research and design and document and plan this.

Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new Geppetto ticket containing an evidence-backed, intern-oriented analysis and implementation plan for safely supporting Pi-originated provider credentials, then publish the documentation as a reMarkable bundle.

**Inferred user intent:** Replace assumption-based provider triage with a clear technical roadmap that explains the current architecture, security boundaries, protocol differences, and a safe path to implementation.

### What I did

- Created `GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS` with a design document, this diary, tasks, and a local source map.
- Inspected Pi `AuthStorage` refresh locking and provider registration behavior without printing any credential values.
- Inspected Pi’s installed OpenAI Codex OAuth and transport code, including PKCE/device login, refresh behavior, Codex response path construction, and required header names.
- Inspected Pi’s Anthropic OAuth flow and Umans extension behavior.
- Inspected Geppetto’s renewable bearer interfaces, factory propagation, OpenAI/Responses request construction, Claude static-key request construction, and Go-only JavaScript source injection boundary.
- Wrote an implementation guide with diagrams, typed API sketches, pseudocode, decisions, phased plan, tests, risks, and review questions.

### Why

The existing Geppetto renewable bearer abstraction intentionally models only a token returned at request time. That is enough for OpenAI-compatible services but not automatically enough for a provider that requires a special endpoint, account metadata, extra headers, or a separate message protocol. The design needed to distinguish credential acquisition from the complete inference transport.

### What worked

- Pi source gives concrete evidence that its auth store serializes login results and performs locked refreshes for expired OAuth records.
- Pi source establishes that OpenAI Codex uses a ChatGPT backend and Codex-specific request mechanics rather than a generic OpenAI Responses endpoint.
- Pi source establishes that Claude uses renewable OAuth while Geppetto’s Claude engine currently uses static `x-api-key` behavior.
- Pi’s Umans extension explicitly documents API-key prompting and a no-op refresh, resolving the earlier ambiguity.
- Geppetto’s existing host-injected bearer and Go-only JavaScript design provide a strong boundary to preserve.

### What didn't work

The earlier public-document-only provider triage was incomplete: it did not inspect the installed Pi provider implementations, so it incorrectly treated the absence of a public generic contract as evidence that no local provider flow existed. The correction is documented as an evidence distinction: Pi implements usable flows, but those flows still do not prove compatibility with Geppetto’s existing generic engines.

No real provider request was attempted. This was intentional: no selected provider has yet passed a fake-server contract suite, host ownership review, and explicit account-use approval.

### What I learned

- OAuth-shaped storage fields do not define an inference transport or even prove a real refresh protocol.
- The OpenAI Codex case requires more than bearer renewal: the target path and companion headers are part of the protocol contract.
- A provider-specific adapter can be secure only when URL validation happens before it releases credential material and no runtime capability becomes profile or JavaScript data.
- Umans should be designed as an Anthropic Messages/API-key compatibility case, not a renewable OAuth integration.

### What was tricky to build

The main difficulty was separating what source evidence proves from what it merely suggests. Pi’s source proves how Pi currently logs in, refreshes, and sends requests; it does not guarantee that the upstream endpoint is a stable external contract or that Geppetto can safely send the same requests. The design resolves this by requiring provider-specific fake-server tests before code and explicit user approval before any live smoke.

A second sharp edge is interface design. Extending `BearerTokenSource` into a general arbitrary request callback would let credential logic alter a request after URL validation, creating an exfiltration risk. The proposed approach instead uses a dedicated engine for Codex and only typed, copied, engine-controlled authentication data where a shared seam is justified.

### What warrants a second pair of eyes

- Review whether ChatGPT Codex transport support is appropriate in Geppetto at all given endpoint stability and account policy.
- Review whether a host should ever directly read Pi’s private auth storage rather than using a broker owned by Pi.
- Review the eventual Claude subscription request-header contract against an approved provider source before dynamic Claude authentication is implemented.
- Review all token-count and non-streaming paths together; partial renewable support would produce inconsistent behavior.

### What should be done in the future

- Implement Phase 0 and Phase 1 fake-server contract tests before adding a Codex transport.
- Design any Pi storage bridge in Pinocchio or another host, never in Geppetto core.
- Audit Claude request semantics and all Claude outbound paths before adding a dynamic source.
- Run a secret-safe, explicitly authorized live smoke only after the corresponding provider contract is accepted.

### Code review instructions

- Start with `sources/01-local-pi-and-geppetto-source-map.md` and Sections 2–5 of the design document.
- Compare `pkg/steps/ai/credentials/bearer.go` with the Codex and Claude source maps to understand why a bearer string is not the complete solution.
- Verify the proposed API remains Go-only and that no profile or JavaScript API is proposed for tokens, headers, account metadata, or refresh callbacks.
- Validate documentation metadata with `docmgr doctor --ticket GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS --stale-after 30`.

### Technical details

Evidence commands included:

```bash
nl -ba pkg/steps/ai/credentials/bearer.go | sed -n '1,300p'
rg -n 'BearerTokenSource|NewEngineFromSettings' pkg/inference/engine/factory/factory.go -C 5
rg -n 'Authorization|x-api-key|apiKey|NewClaudeEngine' pkg/steps/ai/claude -g '*.go' -C 3
```

Pi evidence was read from its installed source and summarized only in `sources/01-local-pi-and-geppetto-source-map.md`. No access token, refresh token, authorization code, client secret, account value, or local auth-file value was copied into this ticket.
