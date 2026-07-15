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

## Step 2: Validate, commit, and deliver the research bundle

The design, diary, source map, task list, ticket index, and changelog were validated and committed as one focused documentation phase. The finished material was then bundled into a single PDF with a table of contents and uploaded to the ticket-specific reMarkable directory.

The delivery bundle intentionally contains only reviewable prose and the redacted evidence map. It does not include provider configuration files, Pi auth storage, source-code archives, or any credential-bearing test fixture.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Validate the completed research ticket, record the delivery evidence, and publish an accessible review bundle.

**Inferred user intent:** Make the resulting technical design straightforward to review away from the terminal while preserving the credential boundary.

**Commit (docs):** f9b0d2cc — "docs: design Pi subscription credential adapters"

### What I did

- Ran frontmatter validation on the design and diary documents.
- Ran `docmgr doctor --ticket GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS --stale-after 30`; all checks passed.
- Committed the ticket documentation, source map, tasks, index, and changelog.
- Ran the required reMarkable bundle dry run with design, diary, and source map inputs.
- Uploaded the resulting bundle to `/ai/2026/07/14/GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS`.

### Why

The deliverable must be auditable before it is published. Bundling the guide with its diary and evidence index gives a reviewer the architectural conclusions, the reasoning trail, and the source references in one navigable artifact without bundling local secret storage.

### What worked

- Both document frontmatters validated successfully.
- Ticket doctor reported all checks passed.
- The dry run identified the correct three inputs and target path.
- The upload completed successfully: `OK: uploaded Geppetto Pi Subscription Credential Adapters.pdf -> /ai/2026/07/14/GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS`.

### What didn't work

N/A. No credential-bearing artifact was requested or needed for the bundle.

### What I learned

A source map is sufficient for a research deliverable when the original local sources include credential-adjacent behavior. It preserves reproducibility and reviewability without duplicating machine-specific files or secret values into a portable PDF.

### What was tricky to build

The bundle needed to include enough evidence for an intern to verify conclusions without including the installed provider code or local auth storage. The solution was to include a short redacted source map alongside precise paths and line ranges in the design document, then bundle that map with the diary and guide.

### What warrants a second pair of eyes

- Verify that the proposed experimental Codex transport is desirable before opening an implementation ticket.
- Verify the selected reMarkable directory is the desired long-term location for this Geppetto research packet.

### What should be done in the future

- Create a follow-up implementation ticket only after the Codex/Claude contract and host ownership questions are accepted.
- Keep a future live-smoke record separate from this research document and redact it to outcome-level evidence.

### Code review instructions

- Review the PDF bundle or the three source Markdown files together.
- Confirm that the guide names no secret values and that the source map explicitly excludes auth-file contents.
- Run `docmgr doctor --ticket GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS --stale-after 30` before changing ticket status.

### Technical details

The bundle command used `remarquee upload bundle` with `--toc-depth 2` and `--non-interactive`. The dry run completed before the actual upload, as required by the ticket delivery workflow.

## Step 3: Reframe lifecycle ownership for reuse beyond Pinocchio

The first design correctly kept Pi storage out of Geppetto, but assigned too much provider login and refresh behavior to every host. The requested architecture instead makes Geppetto a reusable provider-credential library: it supplies supported provider protocol mechanics and secure lifecycle building blocks, while Pinocchio remains one concrete application binding those building blocks to direct YAML and its CLI.

The revised guide now distinguishes provider mechanics from application policy. Geppetto may implement PKCE/state, exchange, refresh, redacted status, local delete, typed request authentication, and provider transports. Pinocchio decides where the credential is stored, which profile is selected, how a browser is presented, whether a Pi record may be migrated, and how output is rendered.

### Prompt Context

**User prompt (verbatim):** "Ok, update the design, reupload to remarkable"

**Assistant interpretation:** Amend the research ticket to make reusable Geppetto lifecycle/provider primitives the primary architecture, then deliver the revised documentation bundle.

**Inferred user intent:** Ensure package users besides Pinocchio can safely reuse provider login, refresh, status, logout, and transport functionality without taking a dependency on Pi or Pinocchio storage.

### What I did

- Rewrote the executive architecture, decisions, diagrams, contracts, implementation phases, risks, and intern checklist in the design guide.
- Added a proposed Geppetto lifecycle surface consisting of `Store`, `ProviderFlow`, `Login`, `StatusOf`, `Logout`, and a host-provided `Presenter`.
- Documented Pinocchio’s role as a direct-YAML store adapter and Glazed CLI/UI binding, with optional explicit Pi migration.
- Reopened the ticket documentation status and added follow-up delivery bookkeeping before publishing the revised bundle.

### Why

Provider PKCE, code exchange, refresh rotation, redacted status, and typed provider transport behavior are reusable Go functionality. Reimplementing them in each host would create duplicate security-sensitive code. Conversely, allowing Geppetto to discover local files or make browser/consent choices would make the library application-specific and unsafe.

### What worked

- The existing design already separated Pi’s private file format from Geppetto, so the revision could preserve all secret-boundary and transport-contract safeguards.
- The new split maps cleanly onto existing Pinocchio components: its direct YAML store remains host-owned, while the future provider protocol implementation can be shared.

### What didn't work

The first version’s statement that the host should own the provider refresh protocol was too restrictive for the intended reusable package API. It would require every Geppetto host to recreate OAuth-sensitive behavior and did not match the desired relationship between Geppetto and Pinocchio.

### What I learned

The correct stable abstraction boundary is not “host owns all credentials” versus “library owns all credentials.” It is: Geppetto owns reusable provider and lifecycle mechanics; hosts own credential placement, identity selection, interaction, and consent policy.

### What was tricky to build

The key design challenge was preserving the security reason for host ownership while moving reusable protocol code into Geppetto. The resolved boundary uses host-injected `Store` and `Presenter` interfaces. That lets the library perform a provider flow without selecting a filesystem path or launching a browser, and lets Pinocchio reuse the flow without exposing a credential capability to profiles or JavaScript.

### What warrants a second pair of eyes

- Review the proposed lifecycle API for correct error/redaction and concurrency semantics before public API commitment.
- Review whether a generic explicit-path file-store helper belongs in Geppetto or should remain a host implementation.
- Review each provider’s terms and transport stability before a provider package is shipped.

### What should be done in the future

- Turn the proposed lifecycle interfaces into a focused Geppetto implementation ticket with fake flow/store tests.
- Follow with a Pinocchio binding ticket that replaces duplicated lifecycle protocol code without changing its direct-YAML ownership model.
- Design any Pi migration separately, with explicit consent and no shared mutable credential ownership.

### Code review instructions

- Start at the revised Executive summary and Section 5 of the design guide.
- Verify that `ProviderFlow` is provider-specific but `Store` and `Presenter` remain host-injected.
- Verify that no filesystem discovery, browser launch, profile selection, or JavaScript secret capability is moved into Geppetto.
- Run `docmgr doctor --ticket GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS --stale-after 30` after the delivery updates.

### Technical details

The revised architecture retains the existing exact-loopback/PKCE-S256, state-validation, locking, atomic rotation, URL-before-token, bounded-401-replay, and no-JavaScript-credential invariants. It moves only reusable provider/lifecycle mechanics into Geppetto.

## Step 4: Validate and publish the revised architecture

The amended guide passed frontmatter and ticket health validation, then was rendered as a new reMarkable bundle. It uses a distinct bundle name so the prior research artifact is preserved rather than force-overwriting a potentially annotated document.

The new bundle presents the corrected reusable-lifecycle architecture alongside the diary and redacted source map. The reMarkable output remains free of credentials and provider auth-file contents.

### Prompt Context

**User prompt (verbatim):** (see Step 3)

**Assistant interpretation:** Complete the revised design’s validation and make the corrected architecture available on reMarkable.

**Inferred user intent:** Ensure the portable review artifact reflects the intended relationship: Geppetto is reusable provider/lifecycle infrastructure and Pinocchio is one storage/UI client.

### What I did

- Ran `docmgr doctor --ticket GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS --stale-after 30`; all checks passed.
- Ran the required dry run for the revised three-document bundle.
- Uploaded `Geppetto Provider Credential Lifecycle Adapters.pdf` to the ticket directory without force-overwriting the prior bundle.

### Why

A different document name avoids silently deleting annotations on the earlier upload while making the architectural revision independently reviewable.

### What worked

- Ticket doctor passed cleanly.
- The dry run listed the design, diary, and redacted source map with the expected target.
- Upload completed successfully: `OK: uploaded Geppetto Provider Credential Lifecycle Adapters.pdf -> /ai/2026/07/14/GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS`.

### What didn't work

N/A. The replacement bundle was uploaded under a new clear name rather than using `--force`, because a force overwrite can discard existing annotations.

### What I learned

Reuploading a revised review packet does not require destructive replacement. Naming the corrected bundle after its actual architectural content preserves both the original research record and the corrected decision.

### What was tricky to build

The desired behavior was a reupload, but the upload tool’s `--force` option deletes an existing document and annotations. The solution was to retain the same ticket directory while choosing a distinct bundle name; this provides the updated material without making an irreversible assumption about annotations.

### What warrants a second pair of eyes

- Confirm whether the earlier Pi-focused bundle should eventually be archived or kept alongside the revised lifecycle-focused packet.
- Review the public API proposal before any Geppetto implementation starts.

### What should be done in the future

- Open a focused implementation ticket for Geppetto lifecycle primitives and provider modules when the API direction is accepted.
- Open a separate Pinocchio adapter/migration ticket afterward.

### Code review instructions

- Read the reMarkable bundle named `Geppetto Provider Credential Lifecycle Adapters` or its three Markdown inputs.
- Confirm the guide assigns provider flow mechanics to Geppetto and storage/UI policy to hosts.
- Verify no secret-bearing file was included in the bundle.

### Technical details

The upload used `remarquee upload bundle` with `--toc-depth 2` and `--non-interactive`, targeting `/ai/2026/07/14/GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS`.

## Step 5: Replace the isolated Codex engine with restricted shared-core middleware

The previous revision placed reusable lifecycle behavior in Geppetto but still proposed a dedicated Codex engine. The architecture now instead treats Codex as a trusted route resolver and request/response middleware installed on Geppetto’s shared OpenAI Responses core. This preserves the core’s request construction, URL validation, retry orchestration, and stream ownership while containing Codex-only credential headers in one small provider component.

The middleware is deliberately narrower than an arbitrary request hook. It receives a read-only post-validation request context and a header writer constrained by engine-declared names; it cannot rewrite URLs, bodies, host/framing headers, or streaming response bodies. The response hook can classify a response and request one bounded retry, but the engine core remains responsible for closing, replaying, and decoding the stream.

### Prompt Context

**User prompt (verbatim):** "ok, update the design ticket with this"

**Assistant interpretation:** Record the shared engine core plus restricted provider route/middleware architecture as the ticket’s proposed implementation direction.

**Inferred user intent:** Avoid a large duplicated Codex engine while keeping provider-specific credential header injection secure, explicit, and reusable by OpenAI and Anthropic engine paths.

### What I did

- Replaced the dedicated-Codex-engine decision with a trusted Codex route resolver and request/response middleware over the shared Responses core.
- Added proposed `RouteResolver`, `Middleware`, `HeaderWriter`, request context, response metadata, and bounded response-decision contracts.
- Replaced Codex and Claude pseudocode with middleware ordering and security invariants.
- Reworked implementation phases, fake-server tests, test matrix, and review questions around shared core behavior and optional stream codecs.

### Why

Codex credential handling needs only a few provider-specific headers, a fixed route, and a 401 refresh classification. Duplicating the entire OpenAI Responses engine would duplicate cancellation, request construction, retry, and stream logic that should remain consistent across providers.

### What worked

- The current design’s strict Go-only credential boundary maps directly to middleware installed through typed engine options.
- Separating route resolution, header injection, and stream decoding makes the security order explicit: final URL validation precedes credential acquisition, and response-body ownership remains with the core.

### What didn't work

The earlier dedicated-engine proposal would likely duplicate hundreds or thousands of existing OpenAI Responses lines solely to carry Codex route and header behavior. A fully mutable `func(*http.Request)` middleware alternative was also rejected because it could bypass URL validation or mutate framing/body behavior.

### What I learned

A reusable middleware seam is safe only when it is not a generic request mutator. The appropriate abstraction exposes a read-only request context, a header-only writer with a static allowlist, and structured response decisions; it never hands middleware the raw stream body or settings-derived configuration.

### What was tricky to build

The difficult part was retaining a shared core without making Codex credential behavior invisible or overly powerful. The solution is a two-stage pipeline: the route resolver runs before final URL validation, then the credential middleware runs only after validation. This prevents a token-bearing middleware from selecting its destination, while still avoiding a duplicated engine.

### What warrants a second pair of eyes

- Review the final public/internal package boundary for the middleware seam before implementation.
- Review header-conflict precedence when multiple trusted middlewares are installed.
- Review the exact response-decision API so middleware cannot cause unbounded retries or consume a stream.

### What should be done in the future

- Build middleware ordering and header-writer fake tests before extracting production code.
- Prove Codex request/SSE compatibility with the existing Responses core; add a provider stream codec only if evidence requires it.
- Apply the same constrained middleware model to Anthropic only after its contract tests are complete.

### Code review instructions

- Start with Sections 5.1, 5.3, and 5.4 of the design guide.
- Verify that `RouteResolver` executes before final URL validation and `Middleware.BeforeRequest` after it.
- Verify middleware cannot alter URL/body/stream and that only core-owned code performs retries and decoding.
- Run `docmgr doctor --ticket GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS --stale-after 30` after ticket bookkeeping.

### Technical details

The proposed Codex adapter consists of a fixed route resolver, a typed `CodexCredentialSource`, an allowlisted `HeaderWriter`, and `AfterResponse` classification. The shared Responses core creates and validates the request, calls middleware, and performs at most one pre-output replay.
