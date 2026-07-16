# Changelog

## 2026-07-14

- Initial workspace created


## 2026-07-14

Captured Pi provider and Geppetto transport evidence; wrote a provider-specific adapter design that keeps Pi storage host-owned and credentials out of settings and JavaScript.

### Related Files

- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/ttmp/2026/07/14/GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS--provider-specific-adapters-for-pi-subscription-credentials/design-doc/01-pi-subscription-credentials-in-geppetto-analysis-adapter-design-and-implementation-guide.md — Intern implementation guide
- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/ttmp/2026/07/14/GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS--provider-specific-adapters-for-pi-subscription-credentials/sources/01-local-pi-and-geppetto-source-map.md — Redacted source map


## 2026-07-14

Validated and delivered the redacted Pi subscription credential adapter guide as a reMarkable bundle after a successful dry run.

### Related Files

- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/ttmp/2026/07/14/GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS--provider-specific-adapters-for-pi-subscription-credentials/reference/01-investigation-diary.md — Validation and delivery evidence


## 2026-07-14

Ticket closed


## 2026-07-14

Reframed credential ownership: Geppetto provides reusable provider lifecycle and transport primitives; hosts bind stores, UI, profile selection, and import policy.

### Related Files

- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/ttmp/2026/07/14/GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS--provider-specific-adapters-for-pi-subscription-credentials/design-doc/01-pi-subscription-credentials-in-geppetto-analysis-adapter-design-and-implementation-guide.md — Revised lifecycle ownership and implementation plan
- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/ttmp/2026/07/14/GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS--provider-specific-adapters-for-pi-subscription-credentials/reference/01-investigation-diary.md — Architecture correction rationale


## 2026-07-14

Validated and re-uploaded the revised Geppetto provider credential lifecycle architecture without overwriting the prior bundle.

### Related Files

- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/ttmp/2026/07/14/GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS--provider-specific-adapters-for-pi-subscription-credentials/reference/01-investigation-diary.md — Revision validation and reMarkable delivery evidence


## 2026-07-14

Ticket closed


## 2026-07-14

Replaced the dedicated Codex engine proposal with a shared Responses core plus validated route resolver and restricted provider request/response middleware.

### Related Files

- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/ttmp/2026/07/14/GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS--provider-specific-adapters-for-pi-subscription-credentials/design-doc/01-pi-subscription-credentials-in-geppetto-analysis-adapter-design-and-implementation-guide.md — Middleware architecture and security ordering
- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/ttmp/2026/07/14/GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS--provider-specific-adapters-for-pi-subscription-credentials/reference/01-investigation-diary.md — Decision rationale and review guidance


## 2026-07-14

Ticket closed


## 2026-07-14

Reopened the ticket as the implementation tracker; added phased tasks for restricted middleware, provider adapters, reusable lifecycle primitives, and Pinocchio binding.

### Related Files

- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/ttmp/2026/07/14/GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS--provider-specific-adapters-for-pi-subscription-credentials/design-doc/01-pi-subscription-credentials-in-geppetto-analysis-adapter-design-and-implementation-guide.md — Implementation phases and acceptance criteria
- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/ttmp/2026/07/14/GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS--provider-specific-adapters-for-pi-subscription-credentials/tasks.md — Phased implementation task plan


## 2026-07-15

Published a clean design-only PDF for the shared-core provider middleware architecture without overwriting earlier review bundles.

### Related Files

- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/ttmp/2026/07/14/GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS--provider-specific-adapters-for-pi-subscription-credentials/reference/01-investigation-diary.md — Clean reMarkable delivery evidence


## 2026-07-16

Implemented and race-tested restricted provider transport contracts with validated route resolution, allowlisted header injection, opaque attempts, and core-governed retry decisions.

### Related Files

- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/pkg/steps/ai/transport/transport.go — New provider transport contracts
- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/pkg/steps/ai/transport/transport_test.go — Security and ordering tests


## 2026-07-16

Wired OpenAI Responses through the restricted provider transport pipeline and added a fake-server-tested typed Codex route and credential middleware.

### Related Files

- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/pkg/steps/ai/openai_responses/request_transport.go — Shared core integration
- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/pkg/steps/ai/providers/openaicodex/codex.go — Codex adapter


## 2026-07-16

Completed reusable lifecycle state/logout, Umans dual-auth binding, Anthropic OAuth header mode, token-count source propagation, OAuth state primitives, and Pinocchio Claude profile binding; live smoke remains approval-gated.

### Related Files

- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/pkg/steps/ai/claude/api/completion.go — Explicit Umans and Anthropic OAuth header modes
- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/pkg/steps/ai/credentials/oauth/state.go — OAuth state primitive
- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/pinocchio/pkg/cmds/profilebootstrap/oauth.go — Host binding

