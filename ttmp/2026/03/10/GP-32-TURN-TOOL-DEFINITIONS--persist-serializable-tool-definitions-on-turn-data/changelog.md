# Changelog

## 2026-03-10

- Initial workspace created
- Added implementation plan for introducing persisted `tool_definitions` on `Turn.Data`
- Scoped the change as a complement to the existing context-carried runtime registry design: persisted definitions for advertisement and inspection, context registry for execution
- Updated the plan to keep provider advertisement strictly sourced from the live runtime registry; persisted `tool_definitions` are informational only
- Implemented the generated `tool_definitions` turn-data key and the `engine.ToolDefinitions` persisted snapshot contract
- Updated the persisted payload to use `ToolDefinitionSnapshot.Parameters map[string]any` so tool definitions survive typed-map YAML round trips
- Stamped persisted tool definitions in the tool loop and added serde regression coverage
- Added runtime authority-boundary tests for advertisement and execution, and introduced a shared context-only helper for advertised tool definitions
- Updated the JS codec/tests so the short `tool_definitions` key round-trips in the Goja module surface
