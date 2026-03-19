# Changelog

## 2026-03-19

- Initial workspace created
- Added the migration analysis and task breakdown for removing `ParseEngineProfileRegistrySourceEntries` and pushing string-list decoding to Glazed where available.

## 2026-03-19

Completed task 1: switched the Glazed runner example helper contract to []string registry sources and adapted the flag-based runner-registry example at the caller boundary.

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/internal/runnerexample/inference_settings.go — Shared example helper now accepts []string directly
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/runner-glazed-registry-flags/main.go — Glazed example decodes profile-registries as TypeStringList
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/runner-registry/main.go — Flag-based example now adapts to the slice-based helper contract

