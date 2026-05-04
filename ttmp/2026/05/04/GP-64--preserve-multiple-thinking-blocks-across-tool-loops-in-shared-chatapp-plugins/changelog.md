# Changelog

## 2026-05-04

- Initial workspace created


## 2026-05-04

Created analysis and implementation guide. Diagnosis: shared ReasoningPlugin uses one thinking entity ID per assistant message, causing multiple thinking phases across tool loops to overwrite each other in sessionstream/Redux and hydrate as only the final block. Recommended fix: segment-aware reasoning entity IDs.


## 2026-05-04

Updated design with addendum: assistant text messages can suffer the same segment-identity folding as thinking blocks. Broadened recommendation from reasoning-only segmentation to assistant transcript segment identity across thinking/text/tool/warning rows.


## 2026-05-04

Implemented fresh-cutover segmented transcript rows in pinocchio and CoinVault, removed legacy runtime-debug compatibility, and validated backend/frontend tests.


## 2026-05-04

Revised design guidance to make GP-64 a fresh cutover: no old IDs, old schemas, dual parser branches, or legacy CoinVault runtime-debug compatibility.


## 2026-05-04

Completed GP-64 browser hydration smoke test against a multi-tool wafer-qwen3.5-397b CoinVault conversation; verified three distinct thinking rows around two tool calls and no console errors.


## 2026-05-04

Uploaded refreshed GP-64 implementation and validation bundle to reMarkable under /ai/2026/05/04/GP-64.

