# Changelog

## 2026-07-10

- Initial workspace created


## 2026-07-10

Implemented host-injected renewable bearer credentials in 8ac6832e: cache/skew/singleflight/persist source plus OpenAI Chat, Responses, and factory wiring; focused/full tests and lint pass, full race has an unrelated existing JS runtime race.

### Related Files

- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/pkg/inference/engine/factory/factory.go — Factory integration
- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/pkg/steps/ai/credentials/bearer.go — Core credential lifecycle implementation

## 2026-07-10

Published the validated renewable credential design bundle to /ai/2026/07/10/GEPPETTO-REFRESHABLE-CREDENTIALS-387 after dry run; upload reported success.

### Related Files

- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/ttmp/2026/07/10/GEPPETTO-REFRESHABLE-CREDENTIALS-387--refreshable-bearer-credentials-for-openai-compatible-providers/design-doc/01-refreshable-bearer-credential-source-analysis-design-and-implementation-guide.md — Published intern-ready design guide
- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/ttmp/2026/07/10/GEPPETTO-REFRESHABLE-CREDENTIALS-387--refreshable-bearer-credentials-for-openai-compatible-providers/reference/01-implementation-diary.md — Delivery commands, output, and validation caveats

## 2026-07-10

Added Phase 4A bounded 401 recovery: opt-in renewable sources force-refresh/persist then Chat or Responses replay once before stream output; second 401 is returned.


## 2026-07-10

Regenerated required logcopter package declarations after check drift; logcopter-check now passes.

### Related Files

- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/pkg/js/modules/geppetto/provider/hostservicesexample/logcopter.go — Generated package logger required by logcopter-check
- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/pkg/steps/ai/imageparts/logcopter.go — Generated package logger required by logcopter-check

## 2026-07-13

Documented reviewed cancellation/forced-refresh hardening, keyed bearer fingerprints, and the host-only JavaScript injection boundary.

### Related Files

- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/pkg/doc/playbooks/08-use-renewable-bearer-credentials.md — Renewable credential operator guide
