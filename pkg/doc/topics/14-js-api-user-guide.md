---
Title: Geppetto JavaScript API User Guide
Slug: geppetto-js-api-user-guide
Short: Practical guide to composing engines, middlewares, tools, and sessions from JavaScript.
Topics:
- geppetto
- javascript
- goja
- user-guide
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Application
---

This guide is focused on writing JS files and validating behavior by executing those JS files directly.

If you need exact signatures for every method, use [JS API Reference](13-js-api-reference.md).

## JS-First Workflow

1. Write a JS script file.
2. Put assertions in the script (`assert(...)`).
3. Run it with:

```bash
go run ./cmd/examples/geppetto-js-lab --script <your-script.js>
```

4. Treat non-zero exit as failure and fix the script or pipeline setup.

## Suggested Project Layout

```text
examples/js/geppetto/
  01_turns_and_blocks.js
  02_session_echo.js
  03_middleware_composition.js
  04_tools_and_toolloop.js
  05_go_tools_from_js.js
  06_live_profile_inference.js
```

You can copy these and branch them into your own scenario files.

## Workflow 1: Start with Turns and Blocks

Goal: verify payload and metadata shape before adding engines.

Run:

```bash
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/01_turns_and_blocks.js
```

Use this stage to lock your turn schema and metadata keys.

## Workflow 2: Add Deterministic Session Inference

Goal: verify session lifecycle and history with no provider variability.

Run:

```bash
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/02_session_echo.js
```

Keep this deterministic until basic state flow is stable.

## Workflow 3: Compose JS and Go Middlewares

Goal: verify middleware order and turn mutation behavior.

Run:

```bash
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/03_middleware_composition.js
```

Default Go middleware names available here:

- `systemPrompt`
- `reorderToolResults`
- `turnLogging`

## Workflow 4: Register JS Tools and Enable Toolloop

Goal: verify tool-call to tool-use lifecycle using pure JS tools.

Run:

```bash
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/04_tools_and_toolloop.js
```

## Workflow 5: Import and Call Go Tools from JS

Goal: confirm hybrid registry behavior (`JS tools + Go tools`).

Run:

```bash
go run ./cmd/examples/geppetto-js-lab --list-go-tools
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/05_go_tools_from_js.js
```

## Workflow 6: Optional Live Provider Inference

Goal: final external smoke check after deterministic scripts pass.

Run:

```bash
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/06_live_profile_inference.js
```

The script skips cleanly if no Gemini key is set (`GEMINI_API_KEY` or `GOOGLE_API_KEY`).

## Recommended Iteration Loop

1. Keep one script per behavior slice.
2. Use assertions for observable outcomes (block kinds, payload content, metadata changes).
3. Commit only scripts that are executable without manual edits.
4. Add live-provider scripts only after deterministic script set is green.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `module geppetto not found` | host runtime did not register module | use `geppetto-js-lab` or register via `gp.Register(reg, opts)` |
| `no go tool registry configured` | script calls `useGoTools` in host without Go registry | run with `geppetto-js-lab` or configure `Options.GoToolRegistry` |
| tool loop does not execute | registry not bound to builder | call `.withTools(reg, { enabled: true })` |
| unstable output in live script | provider variability | keep deterministic checks in `echo`/`fromFunction` scripts |

## See Also

- [JS API Reference](13-js-api-reference.md)
- [JS API Getting Started Tutorial](../tutorials/05-js-api-getting-started.md)
- [Tools](07-tools.md)
- [Middlewares](09-middlewares.md)
- [Sessions](10-sessions.md)
