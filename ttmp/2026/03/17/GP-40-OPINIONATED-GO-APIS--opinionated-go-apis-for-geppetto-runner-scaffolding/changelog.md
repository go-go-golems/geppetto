# Changelog

## 2026-03-17

- Initial workspace created
- Added an evidence-backed architecture analysis of Geppetto's session, enginebuilder, tool loop, tools, middleware, and profile composition surfaces
- Added downstream usage analysis for Pinocchio, CoinVault, CozoDB Editor, and Temporal Relationships
- Added a recommended design for an opinionated Go runner API plus phased implementation guidance and practical examples
- Added a Manuel-specific diary and an API sketch script for review and onboarding
- Updated the design doc after removal of core `AllowedTools` enforcement so registry filtering is treated as app-owned preparation logic rather than a Geppetto `ToolConfig` responsibility
- Updated the design doc again after GP-41, GP-43, and GP-45 so the proposed runner boundary is app-owned resolved runtime input (`StepSettings`, system prompt, middleware uses, tool registrars, runtime identity) rather than Geppetto-owned profile/runtime resolution.
- Added event-driven examples to the design doc showing how the opinionated API still supports streaming servers, SSE/WebSocket-style sinks, and async `Start(...)` flows.
- Added a concrete package-level implementation plan for `pkg/inference/runner`, including a phased build sequence, public API boundary, package layout, test plan, and recommended commit order.
- Replaced the old retrospective task checklist with a live implementation workboard that can track the runner build slice by slice.
- Added the initial `pkg/inference/runner` package skeleton with public runtime/request/result types, constructor options, package docs, and package-scoped validation errors.
- Added the first behavior slice for the runner package: function-tool registration helpers, runner-level default tool registrars, registry construction, registry filtering, and focused tests for the new helpers.
- Added the middleware and engine-assembly slice for the runner package: direct middleware handling, `middlewarecfg`-driven middleware-use resolution, system-prompt injection, reorder middleware insertion, and focused engine-wrapping tests.
- Validated the authored docs, related key evidence files, and uploaded the final bundle to reMarkable
