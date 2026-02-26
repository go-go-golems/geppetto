# Changelog

## 2026-02-12

- Ticket `GP-018-GLAZED-FACADE-FIXUPS` created to track migration fixups and finetunes.
- Completed pinocchio migration to geppetto `sections`/`values` API and committed as `95e0c4b5a42af101b87d604ca510fab7d5855c9d`.
- Completed geppetto hard-cut migration, removed old `pkg/layers` compatibility surface, and committed as `53af798dca730ca7c4edd11bde5cdbd3627800c3`.
- Added and integrated migration tutorial: `pkg/doc/tutorials/05-migrating-to-geppetto-sections-and-values.md`.
- Scrubbed existing geppetto docs to replace stale `*FromParsedLayers` and old symbol references.
- Fixed outbound URL validation to correctly handle IPv6 zone-literal hosts and prevent local-network bypass when local networks are disallowed.
- Updated `GetRunHandler` to return `404` for missing run directories instead of `500`.
- Refactored run event/log parsing to stream from constrained file readers instead of buffering full files in memory.
- Added tests for:
  - zoned IPv6 URL validation behavior
  - `GetRunHandler` missing-run status mapping
  - NDJSON event parsing via streaming readers
- Committed implementation as `db090cce0430fbbc10e81c5a5d86e587c7d3460b`.
- Validation captured:
  - `geppetto`: `go test ./...`, `make lint`
  - `pinocchio`: `go test ./...`, pre-commit test/lint hooks during commit

## 2026-02-25

Ticket closed

