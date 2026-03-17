# Changelog

## 2026-03-16

- Moved the canonical ticket workspace from `pinocchio/ttmp` to `geppetto/ttmp`.
- Reason: the primary design and implementation surface is a Geppetto token-count API, while Pinocchio is the consumer/integration layer.

## 2026-03-05

- Implemented provider-native token counting in `geppetto` for OpenAI Responses and Anthropic Messages count endpoints.
- Added a shared token-count result interface plus a provider-dispatch factory in `geppetto/pkg/inference/tokencount`.
- Extracted Claude message projection logic so inference and count requests share the same Turn-to-message conversion path.
- Extended `pinocchio tokens count` with `--count-mode estimate|api|auto` and geppetto-aware middleware/config loading.
- Added command tests for estimate, provider-backed OpenAI counting, and automatic fallback to local estimation.
- Added an embedded help page for token count modes and updated the ticket diary/tasks to reflect the implementation work.

## 2026-03-05

- Initial workspace created
- Added an evidence-backed design and implementation guide for provider-native token counting across `geppetto` and `pinocchio`.
- Added an investigation diary capturing repository discovery, API research, and the recommended architecture.
- Validated the ticket with `docmgr doctor` and uploaded the documentation bundle to reMarkable at `/ai/2026/03/05/PI-03-TOKEN-COUNT-APIS`.

## 2026-03-05

Add detailed design guide and diary for provider-native OpenAI/Claude token counting across geppetto and pinocchio.

### Related Files

- /home/manuel/workspaces/2026-03-05/add-token-count-api/pinocchio/ttmp/2026/03/05/PI-03-TOKEN-COUNT-APIS--add-provider-native-token-counting-apis-to-geppetto-and-expose-them-via-pinocchio-tokens-count/design-doc/01-provider-native-token-counting-for-geppetto-and-pinocchio.md — Primary implementation guide
- /home/manuel/workspaces/2026-03-05/add-token-count-api/pinocchio/ttmp/2026/03/05/PI-03-TOKEN-COUNT-APIS--add-provider-native-token-counting-apis-to-geppetto-and-expose-them-via-pinocchio-tokens-count/reference/01-investigation-diary.md — Chronological research log
- /home/manuel/workspaces/2026-03-05/add-token-count-api/pinocchio/ttmp/2026/03/05/PI-03-TOKEN-COUNT-APIS--add-provider-native-token-counting-apis-to-geppetto-and-expose-them-via-pinocchio-tokens-count/tasks.md — Execution checklist for later implementation
