### Proposal v2 (Breaking): Tag-only Sink; Extractor-owned Fences and Parsing

Date: 2025-10-29
Scope: Redesign `FilteringSink` to handle only XML-like tags; all code-fence detection and content parsing moves to extractors. This replaces the prior compatibility draft.

---

#### Summary (what changes)
- Sink only detects `<$name:dtype>` and `</$name:dtype>` and filters everything between from user-visible text.
- Sink no longer detects fence languages or parses YAML/JSON at any time (no snapshots, no final parse).
- Sink does not store or propagate any fence info; extractors infer and handle fences/languages themselves.
- New extractor interface (no `OnUpdate`):

```go
type ExtractorSession interface {
    OnStart(ctx context.Context) []events.Event
    OnRaw(ctx context.Context, chunk []byte) []events.Event
    OnCompleted(ctx context.Context, raw []byte, success bool, err error) []events.Event
}
```

- Options simplified:

```go
type Options struct {
    MaxCaptureBytes int
    OnMalformed     MalformedPolicy // enum
    Debug           bool
}
```

- State simplified: remove fence-related buffers/flags; keep an opaque `payloadBuf` for final raw.
- Enforce `MaxCaptureBytes`; on exceed, call `OnCompleted(rawSoFar, false, ErrTooLarge)` and apply `OnMalformed` policy.

#### Extractor responsibilities
- Detect/strip code fences inside the raw payload (e.g., ```yaml|json ... ```), decide language, and parse accordingly.
- Optionally debounce parsing using helper controllers (newline/bytes cadence), budgets (size/time/attempt caps), and emit typed events.

#### Helper toolkit (package `.../parsehelpers`)
- `StripCodeFence(s string) (lang string, body []byte)`
- Debounced controllers: `NewDebouncedYAML[T]`, `NewDebouncedJSON[T]` with `DebounceConfig{ SnapshotEveryBytes, SnapshotOnNewline, ParseTimeout, MaxBytes }`
- Final-only helpers: `FinalOnlyYAML[T]`, `FinalOnlyJSON[T]`
- Event helpers for consistent metadata propagation and delta indexing

#### Next steps (breaking v2)
- Remove `AcceptFenceLangs`, `parseYAML`, fence buffers, and snapshot logic from sink
- Implement new extractor interface, update examples and docs
- Ship helper toolkit with metrics (attempts, successes, failures, bytes)

---



