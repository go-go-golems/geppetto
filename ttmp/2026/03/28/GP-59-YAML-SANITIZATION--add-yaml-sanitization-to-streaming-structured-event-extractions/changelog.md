# Changelog

## 2026-03-28

- Initial workspace created


## 2026-03-28

Created the GP-59 ticket, confirmed the feature belongs in Geppetto's structured-sink parsehelpers layer rather than Pinocchio, and wrote an intern-oriented design and diary covering architecture, API shape, testing, and rollout.

### Related Files

- /home/manuel/workspaces/2026-03-28/sanitize-yaml-structured-events/geppetto/ttmp/2026/03/28/GP-59-YAML-SANITIZATION--add-yaml-sanitization-to-streaming-structured-event-extractions/design-doc/01-intern-guide-to-adding-optional-by-default-yaml-sanitization-to-streaming-structured-event-extractions.md — primary implementation guide
- /home/manuel/workspaces/2026-03-28/sanitize-yaml-structured-events/geppetto/ttmp/2026/03/28/GP-59-YAML-SANITIZATION--add-yaml-sanitization-to-streaming-structured-event-extractions/reference/01-investigation-diary.md — chronological investigation record


## 2026-03-28

Validated the ticket with docmgr doctor and uploaded the bundled ticket packet to reMarkable under /ai/2026/03/28/GP-59-YAML-SANITIZATION.

### Related Files

- /home/manuel/workspaces/2026-03-28/sanitize-yaml-structured-events/geppetto/ttmp/2026/03/28/GP-59-YAML-SANITIZATION--add-yaml-sanitization-to-streaming-structured-event-extractions/tasks.md — ticket status updated after validation and delivery


## 2026-03-30

Implemented default-on YAML sanitization in Geppetto structured-sink parsehelpers using `github.com/go-go-golems/sanitize/pkg/yaml`, added helper-focused tests for sanitized and opt-out behavior, and updated the structured-sink docs/tutorials to show the shipped helper API (`YAMLController`, `FeedBytes`, `FinalBytes`, `sanitize_yaml`).

### Related Files

- /home/manuel/workspaces/2026-03-28/sanitize-yaml-structured-events/geppetto/pkg/events/structuredsink/parsehelpers/helpers.go — added sanitize-aware normalization and config defaults
- /home/manuel/workspaces/2026-03-28/sanitize-yaml-structured-events/geppetto/pkg/events/structuredsink/parsehelpers/helpers_test.go — added default-on and opt-out parsing coverage
- /home/manuel/workspaces/2026-03-28/sanitize-yaml-structured-events/geppetto/pkg/doc/topics/11-structured-sinks.md — updated public helper examples
- /home/manuel/workspaces/2026-03-28/sanitize-yaml-structured-events/geppetto/pkg/doc/playbooks/03-progressive-structured-data.md — updated playbook examples
- /home/manuel/workspaces/2026-03-28/sanitize-yaml-structured-events/geppetto/pkg/doc/tutorials/04-structured-data-extraction.md — updated tutorial examples

Fixed a follow-up semantic bug where whole-document trimming could change valid YAML block-scalar values before unmarshal. The helper now preserves original body bytes unless sanitization is explicitly enabled, and regression tests cover trailing-newline preservation.

### Related Files

- /home/manuel/workspaces/2026-03-28/sanitize-yaml-structured-events/geppetto/pkg/events/structuredsink/parsehelpers/helpers.go — removed unconditional whole-document trimming before unmarshal
- /home/manuel/workspaces/2026-03-28/sanitize-yaml-structured-events/geppetto/pkg/events/structuredsink/parsehelpers/helpers_test.go — added block-scalar regression coverage
