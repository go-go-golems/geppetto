## Surfacing low-level provider data for development/logging

### Motivation
- During E2E and debugging sessions, we need access to raw provider signals (HTTP payloads, SSE frames, response objects) that are normally abstracted by engines. Exposing these without polluting public APIs or leaking into production code requires a deliberate design.

### Design goals
- Opt-in, development-only visibility of low-level data
- Preserve engine’s clean high-level Event API for normal consumers
- Avoid tight coupling to providers (OpenAI/Anthropic specifics)
- Consistent artifacts across engines where possible (request.json, events.ndjson, final_turn.yaml)

### Proposed design

1) Context-scoped DebugTap
- Define `DebugTap` interface in `pkg/inference/engine` (or `pkg/events`) that receives provider-raw breadcrumbs.
  - Example methods:
    - `OnHTTP(request *http.Request, body []byte)`
    - `OnHTTPResponse(resp *http.Response, body []byte)`
    - `OnSSE(event string, data []byte)`
    - `OnProviderObject(name string, v any)` // for decoded provider structs
- Provide `WithDebugTap(ctx, tap DebugTap) context.Context` and getter. Engines check for tap and emit breadcrumbs when available.
- Runner installs a simple `DebugTap` that mirrors to disk (`raw/request.json`, `raw/response-*.json`, `raw/sse.log`).

2) Engine Options: WithRawCapture(writer)
- Add `engine.WithRawCapture(dir string)` Option that instructs an engine to save raw inputs/outputs to a directory.
- Internally implemented via the same DebugTap, but convenient for tooling.
- Scope to dev tools and examples, not recommended for production.

3) Event extensions (non-breaking)
- For development, where allowed, attach minimal pointers to raw info via `EventMetadata.Extra`:
  - `extra.request_id`, `extra.response_id`, `extra.item_ids` (strings)
  - Never embed full payloads into events; keep events small.
- This lets the runner correlate NDJSON events with the raw folder.

4) Providers’ raw adapters
- Engines keep a small adapter layer to standardize tap calls:
  - Before sending: `tap.OnHTTP(req, body)`
  - On receive non-streaming: `tap.OnHTTPResponse(resp, body)`
  - While streaming: for each SSE line: `tap.OnSSE(event, data)`
  - When decoding a provider object: `tap.OnProviderObject("response.completed", obj)` (optional)

5) Storage layout (runner-friendly)
- out/
  - request.json (engine request body)
  - final_turn.yaml
  - events.ndjson
  - raw/
    - http-request.json
    - http-response-001.json
    - sse.log (verbatim lines)
    - provider-response.completed.json (optional decoded summary)

6) Privacy & security
- Default off. Require explicit flags/options in the runner.
- Redaction hooks for headers (Authorization), PII in provider payloads.
- Document that encrypted reasoning is preserved, not redacted.

### API sketch
```go
// pkg/inference/engine/debugtap.go
type DebugTap interface {
    OnHTTP(req *http.Request, body []byte)
    OnHTTPResponse(resp *http.Response, body []byte)
    OnSSE(event string, data []byte)
    OnProviderObject(name string, v any)
}

type debugTapKey struct{}

func WithDebugTap(ctx context.Context, tap DebugTap) context.Context {
    return context.WithValue(ctx, debugTapKey{}, tap)
}
func DebugTapFrom(ctx context.Context) (DebugTap, bool) {
    v := ctx.Value(debugTapKey{})
    if v == nil { return nil, false }
    t, ok := v.(DebugTap)
    return t, ok
}

// Engine option sugar
func WithRawCapture(dir string) engine.Option { /* create tap that writes to dir */ }
```

### Engine integration (OpenAI Responses example)
- Where we construct the HTTP request: call `tap.OnHTTP(req, body)`
- After non-2xx: `tap.OnHTTPResponse(resp, errorBody)`
- In SSE loop: for each line, after trimming, `tap.OnSSE(eventName, []byte(line))`
- When decoding `response.completed`: `tap.OnProviderObject("response.completed", decoded)`

### Runner UX
- Flags: `--raw-dir out/raw` enables raw capture; runner passes `WithRawCapture` to engine or installs a `DebugTap` on context.
- Report includes links to raw files. Events still flow as NDJSON for normal pretty/structured analysis.

### Benefits
- Keeps core Event API clean while enabling power-user introspection.
- Zero impact when unused (tap absent).
- Works for all engines with minimal per-provider code.

### Risks & mitigations
- Large files: use line-based SSE log; rotate big responses; allow `--raw-max-size`.
- Sensitive data: configurable header redaction; document implications.

### Conclusion
Adopt a context-scoped `DebugTap` with a convenience `WithRawCapture` option. Engines emit breadcrumbs opportunistically without changing their public surface. Runners choose when to capture and how to persist, enabling robust debugging and reproducible investigations.


