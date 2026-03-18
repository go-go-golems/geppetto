# Changelog

## 2026-03-18

- Initial workspace created
- Completed the first research pass across the JS module, its docs, and the example scripts.
- Drafted the main design direction: add an additive `gp.runner` namespace instead of further growing the low-level `createBuilder` / `createSession` API.
- Identified the key current-state gap: profile resolution in JS already returns runtime metadata, but the module does not provide a first-class path that consumes that metadata into execution assembly.
- Rewrote the task board into concrete implementation phases after GP-47 landed, so the ticket can now proceed slice by slice on top of the cleaned runtime-metadata substrate.

## 2026-03-18

Completed the GP-46 research pass and wrote the main design guide plus diary for an opinionated JS runner layer above the current builder/session API.

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_sessions.go — Low-level session assembly analyzed
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/module.go — Export surface analyzed
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/18/GP-46-OPINIONATED-JS-APIS--opinionated-javascript-apis-for-geppetto/design-doc/01-opinionated-javascript-api-design-and-implementation-guide.md — Main deliverable
