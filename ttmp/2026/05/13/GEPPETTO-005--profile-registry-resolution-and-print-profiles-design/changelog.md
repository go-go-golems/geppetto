# Changelog

## 2026-05-13

- Initial workspace created


## 2026-05-13

Created the profile registry resolution and --print-profiles implementation guide, including Geppetto/Pinocchio/CoinVault architecture, file references, proposed APIs, CLI UX, security constraints, and intern implementation plan.

### Related Files

- /home/manuel/workspaces/2026-05-13/coinvault-loop-analysis/geppetto/ttmp/2026/05/13/GEPPETTO-005--profile-registry-resolution-and-print-profiles-design/design-doc/01-profile-registry-resolution-and-print-profiles-implementation-guide.md — Primary design and implementation guide
- /home/manuel/workspaces/2026-05-13/coinvault-loop-analysis/geppetto/ttmp/2026/05/13/GEPPETTO-005--profile-registry-resolution-and-print-profiles-design/reference/01-investigation-diary.md — Chronological investigation diary


## 2026-05-13

Implemented the first profile introspection slice: Geppetto now has reusable report/redaction/rendering helpers and a profile-introspection section, and CoinVault chat send has an early-exit --print-profiles path before DB/LLM startup.

### Related Files

- /home/manuel/workspaces/2026-05-13/coinvault-loop-analysis/geppetto/../2026-03-16--gec-rag/cmd/coinvault/cmds/chat_send.go — Wires CoinVault chat send early-exit profile printing
- /home/manuel/workspaces/2026-05-13/coinvault-loop-analysis/geppetto/pkg/cli/bootstrap/profile_introspection.go — Reusable profile registry report builder
- /home/manuel/workspaces/2026-05-13/coinvault-loop-analysis/geppetto/pkg/cli/bootstrap/profile_introspection_test.go — Tests default/selected profile reporting
- /home/manuel/workspaces/2026-05-13/coinvault-loop-analysis/geppetto/pkg/sections/profile_introspection_section.go — Defines --print-profiles and related profile introspection flags

