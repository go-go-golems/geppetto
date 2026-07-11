# Changelog

## 2026-07-10

- Initial workspace created


## 2026-07-10

Step 1: captured the sparse-profile runtime failure, provider matrix, normalization contract, test plan, and task sequence

### Related Files

- /home/manuel/workspaces/2026-07-10/fix-geppetto-inference-profiles/geppetto/pkg/js/modules/geppetto/api_engines.go — Shared normalization boundary analyzed


## 2026-07-10

Step 2: implemented provider-aware API/client/provider defaults, preserved explicit values and unsupported-provider rejection, added offline request-construction regression coverage, and validated with full tests and lint (commit d0557f7f)

### Related Files

- /home/manuel/workspaces/2026-07-10/fix-geppetto-inference-profiles/geppetto/pkg/js/modules/geppetto/api_engines.go — Shared normalization implementation


## 2026-07-10

All implementation tasks complete; provider-aware normalization and regression tests validated in commit d0557f7f


## 2026-07-10

Step 3: normalized canonical Claude credential, URL, and outbound-policy settings into missing `anthropic` alias keys so factory validation and Claude runtime lookup agree (commit 643d5313)

### Related Files

- /home/manuel/workspaces/2026-07-10/fix-geppetto-inference-profiles/geppetto/pkg/js/modules/geppetto/api_engines.go — Alias runtime-key normalization
- /home/manuel/workspaces/2026-07-10/fix-geppetto-inference-profiles/geppetto/pkg/js/modules/geppetto/api_agent_profile_test.go — Alias propagation regression coverage
