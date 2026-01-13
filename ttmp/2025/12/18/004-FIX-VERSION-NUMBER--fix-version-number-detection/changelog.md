# Changelog

## 2025-12-18

- Initial workspace created


## 2025-12-18

Created analysis document and diary documenting version detection issues. Root cause: tags v0.5.6/v0.5.7 on parallel branch not reachable from HEAD. GitHub releases separate from tags.

### Related Files

- /home/manuel/workspaces/2025-12-01/integrate-moments-persistence/geppetto/.github/workflows/release.yml — Release workflow
- /home/manuel/workspaces/2025-12-01/integrate-moments-persistence/geppetto/Makefile — Uses svu current


## 2025-12-18

Workflow analysis: release.yml is manual-only (tag trigger disabled in 70b2a2b). Added tag-release-notes.yml to create artifact-free GitHub Releases on v* tags and allow manual backfill.

### Related Files

- /home/manuel/workspaces/2025-12-01/integrate-moments-persistence/geppetto/.github/workflows/release.yml — Manual-only goreleaser workflow
- /home/manuel/workspaces/2025-12-01/integrate-moments-persistence/geppetto/.github/workflows/tag-release-notes.yml — Artifact-free GitHub Release creation workflow
- /home/manuel/workspaces/2025-12-01/integrate-moments-persistence/geppetto/.goreleaser.yaml — GoReleaser config may not target an existing main package


## 2026-01-05

Ticket closed

