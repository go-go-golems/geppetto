# Changelog

## 2026-03-19

- Initial workspace created
# Changelog

## 2026-03-19

Created the follow-up ticket for extracting the generic CLI bootstrap path from Pinocchio into Geppetto and parameterizing the application identity. Added the initial design/implementation guide covering the current code split, the proposed Geppetto package boundary, the app-configurable bootstrap surface, migration phases, and the main file inventory that will need to change.

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/19/GP-53-GEPPETTO-CLI-BOOTSTRAP--extract-generic-cli-bootstrap-path-to-geppetto-and-parameterize-app-name/design-doc/01-generic-geppetto-cli-bootstrap-extraction-and-app-name-parameterization-guide.md — Primary analysis and implementation guide for the extraction
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/profilebootstrap/profile_selection.go — Current Pinocchio-owned generic profile-selection candidate
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/profilebootstrap/engine_settings.go — Current Pinocchio-owned generic engine-settings candidate
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/sections.go — Existing Geppetto bootstrap path with Pinocchio-specific assumptions
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/internal/runnerexample/inference_settings.go — Current Geppetto registry-only helper path
