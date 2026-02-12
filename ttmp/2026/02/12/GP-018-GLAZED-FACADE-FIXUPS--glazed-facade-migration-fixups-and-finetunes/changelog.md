# Changelog

## 2026-02-12

- Ticket `GP-018-GLAZED-FACADE-FIXUPS` created to track migration fixups and finetunes.
- Completed pinocchio migration to geppetto `sections`/`values` API and committed as `95e0c4b5a42af101b87d604ca510fab7d5855c9d`.
- Completed geppetto hard-cut migration, removed old `pkg/layers` compatibility surface, and committed as `53af798dca730ca7c4edd11bde5cdbd3627800c3`.
- Added and integrated migration tutorial: `pkg/doc/tutorials/05-migrating-to-geppetto-sections-and-values.md`.
- Scrubbed existing geppetto docs to replace stale `*FromParsedLayers` and old symbol references.
- Validation captured:
  - `geppetto`: `go test ./...`, `make lint`
  - `pinocchio`: `go test ./...`, pre-commit test/lint hooks during commit
