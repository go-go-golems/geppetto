### Migration Guide: v2 Tag-only Sink and Extractor-owned Parsing

Date: 2025-10-29
Scope: Update `FilteringSink` and all extractors/examples to the v2 design: sink handles XML-like tags only; extractors own fence detection and parsing.

---

### 1) What’s changing (high-level)

- Sink no longer detects code fences or parses YAML/JSON at any time (no mid-stream snapshots, no final parse).
- Sink detects only `<$name:dtype>` … `</$name:dtype>` and filters everything between from user-visible text.
- Extractors receive raw bytes inside the tag window via `OnRaw([]byte)` and the full accumulated payload via `OnCompleted(raw []byte, ...)`.
- Fence language handling (```yaml / ```json, etc.) and parsing are fully owned by extractors (with helper toolkit support).

---

### 2) Breaking API changes

- `ExtractorSession` interface (remove `OnUpdate`, change delta signature, change completed payload):

```go
// OLD
type ExtractorSession interface {
    OnStart(ctx context.Context) []events.Event
    OnDelta(ctx context.Context, raw string) []events.Event
    OnUpdate(ctx context.Context, snapshot map[string]any, parseErr error) []events.Event
    OnCompleted(ctx context.Context, final map[string]any, success bool, err error) []events.Event
}

// NEW (v2)
type ExtractorSession interface {
    OnStart(ctx context.Context) []events.Event
    OnRaw(ctx context.Context, chunk []byte) []events.Event
    OnCompleted(ctx context.Context, raw []byte, success bool, err error) []events.Event
}
```

- `Options` struct: remove fence and sink-parsing related fields. Keep minimal options (example):

```go
type Options struct {
    MaxCaptureBytes int
    OnMalformed     MalformedPolicy // enum
    Debug           bool
}
```

- Sink internals: remove `tryDetectFenceOpen`, fence buffers/flags, `parseYAML`, and snapshot logic.

---

### 3) Step-by-step: Refactor the sink

1. Update `ExtractorSession` type in `geppetto/pkg/events/structuredsink/filtering_sink.go`:
   - Replace the old interface with the new one shown above.
   - Rename all call sites: `OnDelta(...)` → `OnRaw(...)` and pass `[]byte` (prefer coalescing per incoming delta, not per character).
   - Remove all `OnUpdate(...)` invocations.

2. Simplify `Options`:
   - Delete `AcceptFenceLangs`, `EmitRawDeltas`, `EmitParsedSnapshots`.
   - Convert `OnMalformed` to a typed enum if not already.

3. Remove fence detection and parsing:
   - Delete `tryDetectFenceOpen(...)` and related buffer/flags (`fenceBuf`, `inFence`, `fenceOpened`, `fenceLangOK`, `awaitingCloseTag` used for fence state).
   - Delete `parseYAML(...)` and all snapshot/final parsing code paths.

4. Capture semantics (tag-only):
   - On open tag: start capture, initialize `payloadBuf` (opaque bytes), create/emit `OnStart`.
   - During capture: append the incoming delta bytes to `payloadBuf` and call `OnRaw(ctx, deltaBytes)` once per input delta (not per char).
   - On close tag: call `OnCompleted(ctx, payloadBufBytes, true, nil)`, clear capture state, continue scanning.
   - On final with capture open: call `OnCompleted(ctx, payloadBufBytes, false, ErrUnclosedBlock)`, apply `OnMalformed` policy, then clear state.

5. Limits and policy:
   - Enforce `MaxCaptureBytes` on `payloadBuf.Len()`. If exceeded, call `OnCompleted(ctx, payloadSoFar, false, ErrTooLarge)` and apply `OnMalformed`.
   - Preserve event ordering: publish filtered text first, then any typed events returned by extractor methods for that delta.

6. Logging/observability:
   - Avoid logging raw payload in debug; prefer sizes and state transitions.
   - Keep existing metadata propagation behavior in `publishAll`.

---

### 4) Update extractors to v2

For each extractor implementation:

1. Change interface methods:
   - Rename `OnDelta(ctx, raw string)` → `OnRaw(ctx, chunk []byte)`; adapt caller sites to pass bytes.
   - Remove `OnUpdate(...)` entirely; any snapshot logic moves to a helper-driven debounce inside the extractor.
   - Change `OnCompleted(ctx, final map[string]any, ...)` → `OnCompleted(ctx, raw []byte, ...)`.

2. Handle fences and parsing in the extractor:
   - If your prompts emit fenced payloads, first strip the fence and detect the language.
   - Then parse into your struct (YAML/JSON/TOML) either on completion or debounced mid-stream.

Minimal example (final-only YAML):

```go
type MyPayload struct { Title string `yaml:"title"`; URLs []string `yaml:"urls"` }

type MySession struct { buf bytes.Buffer }

func (s *MySession) OnStart(ctx context.Context) []events.Event { return nil }

func (s *MySession) OnRaw(ctx context.Context, chunk []byte) []events.Event {
    s.buf.Write(chunk)
    return nil
}

func (s *MySession) OnCompleted(ctx context.Context, raw []byte, success bool, err error) []events.Event {
    if !success || err != nil { return []events.Event{ newErrorEvent(err) } }
    // Strip code fence (if present) and parse
    lang, body := parsehelpers.StripCodeFenceBytes(raw)
    _ = lang // optional: branch on lang ("yaml", "json", ...)
    var p MyPayload
    if perr := yaml.Unmarshal(body, &p); perr != nil { return []events.Event{ newErrorEvent(perr) } }
    return []events.Event{ newCompletedEvent(p) }
}
```

Debounced snapshots variant (extractor-owned):

```go
ctrl := parsehelpers.NewDebouncedYAML[MyPayload](parsehelpers.DebounceConfig{
    SnapshotEveryBytes: 1024,
    SnapshotOnNewline:  true,
    ParseTimeout:       50 * time.Millisecond,
    MaxBytes:           64 << 10,
})

func (s *MySession) OnRaw(ctx context.Context, chunk []byte) []events.Event {
    if snap, err := ctrl.FeedBytes(chunk); snap != nil {
        return []events.Event{ newUpdateEvent(*snap, err) }
    }
    return nil
}

func (s *MySession) OnCompleted(ctx context.Context, raw []byte, success bool, err error) []events.Event {
    if !success || err != nil { return []events.Event{ newErrorEvent(err) } }
    p, perr := ctrl.FinalBytes(raw)
    if perr != nil { return []events.Event{ newErrorEvent(perr) } }
    return []events.Event{ newCompletedEvent(*p) }
}
```

---

### 5) Introduce helper toolkit (`geppetto/pkg/events/structuredsink/parsehelpers`)

Recommended minimal APIs:

```go
// StripCodeFenceBytes detects ```lang\n ... \n``` around the payload; returns lang (lowercased) and body bytes.
func StripCodeFenceBytes(b []byte) (lang string, body []byte)

type DebounceConfig struct {
    SnapshotEveryBytes int
    SnapshotOnNewline  bool
    ParseTimeout       time.Duration
    MaxBytes           int
}

// NewDebouncedYAML returns a controller that attempts YAML parse per cadence.
func NewDebouncedYAML[T any](cfg DebounceConfig) *YAMLController[T]

// Controller API
func (c *YAMLController[T]) FeedBytes(chunk []byte) (*T, error)
func (c *YAMLController[T]) FinalBytes(raw []byte) (*T, error)

// Similarly for JSON
func NewDebouncedJSON[T any](cfg DebounceConfig) *JSONController[T]
```

Notes:
- Enforce size/time budgets in controllers; return typed errors when exceeded.
- Provide small event constructors that auto-fill metadata consistently.

---

### 6) Example app migration (`cmd/examples/citations-event-stream`)

1. Update extractor types to the new interface (replace `OnDelta`/`OnUpdate` with `OnRaw`; change `OnCompleted` signature).
2. Move any YAML parsing from the sink-driven snapshots/final into the extractor using `parsehelpers`.
3. If the example emitted per-character deltas, consider coalescing by line/chunk before calling `OnRaw` (optional; helpers handle small chunks).
4. Re-run and verify that UI-visible text is unaffected and typed events still flow with correct metadata.

---

### 7) Testing and validation

- Update tests to remove expectations around `OnUpdate` and sink-side parsing.
- Add tests for:
  - Tag split across deltas → correct capture window and filtering.
  - Unclosed tag at final → `OnCompleted(..., success=false)` and policy behavior.
  - Size overflow → early `OnCompleted(..., false, ErrTooLarge)`.
  - Extractor fence detection: `StripCodeFenceBytes` correctness (yaml/json/empty header).
  - Debounced parsing cadence and budgets.

---

### 8) Rollout plan

1. Apply sink refactor and compile; fix call sites.
2. Migrate extractors one by one; compile after each.
3. Update example app; manually validate typed event flows.
4. Add/adjust tests; run CI.
5. Monitor resource usage; expect reduced sink CPU/alloc.

---

### 9) Removal/deprecation checklist

- [ ] Remove `AcceptFenceLangs`, `EmitParsedSnapshots`, `EmitRawDeltas` from `Options`
- [ ] Delete `tryDetectFenceOpen`, fence buffers/flags, and related code paths
- [ ] Delete `parseYAML` and all snapshot/final parsing in the sink
- [ ] Remove `OnUpdate` from `ExtractorSession` and all implementations
- [ ] Replace `OnDelta` usages with `OnRaw([]byte)`
- [ ] Enforce `MaxCaptureBytes` and unify malformed handling on unclosed/overflow

---

### 10) FAQ

- Q: Can we still support JSON or TOML?
  - A: Yes—extractors decide. Use `StripCodeFenceBytes` to detect language and parse accordingly. JSON often parses under YAML, but for strictness use JSON helpers.

- Q: How do we get snapshots mid-stream now?
  - A: Use the debounced controllers in your extractor. The sink no longer emits snapshots.

- Q: Does the sink still guarantee event ordering?
  - A: Yes. Filtered text first, then typed events returned by the extractor for the same delta.


