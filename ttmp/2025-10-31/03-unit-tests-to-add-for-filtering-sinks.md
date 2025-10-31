---
Title: Unit Test Plan for FilteringSink
Slug: unit-tests-filtering-sink
Short: Comprehensive tests to validate tag parsing across partials, policies, and context/metadata behavior.
Date: 2025-10-31
Owners: geppetto/events
Status: Draft
---

## Overview

This guide describes a complete set of unit tests for `FilteringSink` (`geppetto/pkg/events/structuredsink/filtering_sink.go`). It provides enough context for a new developer to understand the streaming model, the structured tag format, and how to validate correctness across tricky partial boundaries and edge cases.

`FilteringSink` sits in the streaming path and:
- Detects structured blocks in text streams using XML-like tags: `<$name:dtype>` … `</$name:dtype>`.
- Removes structured blocks from the forwarded “filtered text” (what the UI shows).
- Starts an extractor session for each block and streams raw payload to the extractor via `OnRaw` and `OnCompleted`.
- Emits typed events returned by the extractor alongside filtered text events.

In v2, the sink only understands tags (not YAML/JSON); extractors own parsing. The sink maintains a small state machine and a lag buffer to safely detect the closing tag without leaking close-tag bytes into payload or UI.


## Quick glossary

- **Partial**: `EventPartialCompletion` with `Delta` and cumulative `Completion`.
- **Final**: `EventFinal` containing the full final text of the stream.
- **Extractor**: Registers for `(name, dtype)`; per block it receives `OnStart`, then `OnRaw` chunks, then `OnCompleted(raw, success, err)`.
- **Malformed policy**: Controls behavior when a block is unclosed at final. Options: error-events, forward-raw, ignore.
- **Filtered text**: Text forwarded downstream after removing structured blocks.


## Test utilities (recommended patterns)

- Simple downstream sink to capture forwarded events for assertions:

```go
type eventCollector struct{ list []events.Event }
func (c *eventCollector) PublishEvent(ev events.Event) error { c.list = append(c.list, ev); return nil }
```

- Minimal test extractor for a specific `(name, dtype)` capturing lifecycle:

```go
type testExtractor struct{ name, dtype string; last *testSession }
func (e *testExtractor) Name() string     { return e.name }
func (e *testExtractor) DataType() string { return e.dtype }
func (e *testExtractor) NewSession(ctx context.Context, meta events.EventMetadata, itemID string) ExtractorSession {
    s := &testSession{ctx: ctx, itemID: itemID}
    e.last = s
    return s
}

type testSession struct {
    ctx       context.Context
    itemID    string
    started   bool
    rawChunks []string
    completed bool
    finalRaw  string
    success   bool
}
func (s *testSession) OnStart(ctx context.Context) []events.Event { s.started = true; return nil }
func (s *testSession) OnRaw(ctx context.Context, b []byte) []events.Event { s.rawChunks = append(s.rawChunks, string(b)); return nil }
func (s *testSession) OnCompleted(ctx context.Context, raw []byte, ok bool, err error) []events.Event {
    s.completed, s.finalRaw, s.success = true, string(raw), ok
    return nil
}
```

- Helpers to collect forwarded partial deltas and final text:

```go
func collectTextParts(list []events.Event) (partials []string, final string) {
    for _, ev := range list {
        switch v := ev.(type) {
        case *events.EventPartialCompletion:
            partials = append(partials, v.Delta)
        case *events.EventFinal:
            final = v.Text
        case *events.EventImpl:
            if pc, ok := v.ToPartialCompletion(); ok { partials = append(partials, pc.Delta) }
            if tf, ok := v.ToText(); ok { final = tf.Text }
        }
    }
    return
}
```


## Test categories and detailed cases

Below are prioritized tests, each with purpose, setup, and expected assertions.

### 1) Open-tag split matrix

Purpose: Prove tag recognition is robust when the open tag arrives in fragments across partials.

Setup: Register extractor for `(x, v1)`. Send outside text followed by broken-up tag pieces until `<$x:v1>` is complete.

Split boundaries to cover (each as a separate test):
- `<` | `$` | `x` | `:` | `v1` | `>`; examples of parts: `"<"`, `"$x"`, `":v1"`, `">"`.

Assertions:
- `OnStart` fires exactly once once tag completes.
- No open-tag bytes appear in forwarded text.
- Filtered final equals outside text only.

### 2) Close-tag near-misses

Purpose: Ensure payload strings resembling a close tag don’t prematurely close.

Setup: Payload includes strings like `"</$x:v1!>"`, `"</$x:v1>>"` before the real `</$x:v1>`.

Assertions:
- No early completion; `OnRaw` includes those near-miss strings.
- Completion only when exact `</$x:v1>` appears.

### 3) Case sensitivity mismatch

Purpose: Close tag must match exactly. Mismatched case doesn’t close and should be treated as malformed on final.

Setup: Open with `<$X:v1>`, then attempt `</$x:v1>` (lowercase).

Assertions:
- No close on mismatch; at final, `OnCompleted(success=false, err)` is called per policy.
- Filtered final adheres to default policy (drop captured content).

### 4) Empty payload

Purpose: Validate lifecycle when block is empty.

Setup: `<$x:v1></$x:v1>`.

Assertions:
- `OnStart` then `OnCompleted(success=true, finalRaw=="")`.
- No tag text appears in filtered final.

### 5) Back-to-back blocks without spacing

Purpose: Ensure two blocks are handled sequentially with correct sequencing and no text leakage.

Setup: `<$x:v1>a</$x:v1><$x:v1>b</$x:v1>`.

Assertions:
- Two sessions, `itemID` suffixes `:1` and `:2`.
- `finalRaw` values `"a"` and `"b"`.
- Filtered output empty (if no outside text).

### 6) Interleaved blocks for different extractors

Purpose: Route to the correct extractor and avoid cross-talk.

Setup: Register `(a,v1)` and `(b,v1)`. Stream `…<$a:v1>…</$a:v1> mid <$b:v1>…</$b:v1>…`.

Assertions:
- Each extractor gets its own lifecycle; no chunks from the other block.
- Filtered text preserves only outside content.

### 7) Unknown extractor: full tag flush

Purpose: When no extractor is registered, ensure the entire tag (and payload) is forwarded verbatim as text.

Setup:
- Single-part case: `X <$unknown:v1>abc</$unknown:v1> Y`.
- Split-across-partials variant: split the open tag or close tag over boundaries.

Assertions:
- No extractor `OnStart`/`OnCompleted`.
- Filtered final equals the original text, including tags.

### 8) Malformed policies (unclosed at final)

Purpose: Verify all policies: `error-events`, `forward-raw`, `ignore`.

Setup: Stream `"before <$x:v1>payload"` and finish without a close tag. Run three tests, each with different options.

Assertions:
- `error-events`: `OnCompleted(success=false)` called; filtered final drops captured region.
- `forward-raw`: reconstructed block text reinserted into filtered final; `OnCompleted(success=false)`.
- `ignore`: captured text dropped; `OnCompleted(success=false)`.

### 9) Final-only inputs

Purpose: Validate behavior when there are no partials.

Cases:
- Valid block entirely in final.
- Malformed (no close tag) in final.
- Unknown extractor in final.

Assertions:
- Same as above for lifecycle and filtered final; no partial deltas emitted.

### 10) Metadata propagation for typed events

Purpose: Ensure events emitted by extractors receive stream metadata if missing.

Setup: Test extractor returns `events.EventImpl{}` with zero metadata in `OnStart` or `OnCompleted`.

Assertions:
- After `PublishEvent`, captured typed events have `Metadata.ID == streamID` (and RunID/TurnID copied).

### 11) Item context lifecycle (cancellation)

Purpose: Context for an item must be canceled on completion or malformed handling.

Setup: Extractor stores `ctx.Done()` and exposes a `done` channel. After close (or final malformed), verify the channel closes within a short timeout.

Assertions:
- Not canceled before close; canceled after close/final.

### 12) Multiple streams interleaved

Purpose: Ensure independence across two `Metadata.ID`s.

Setup: Alternate partials between stream A and B, each opening and closing a block.

Assertions:
- Each stream gets its own session and filtered output; no cross-contamination.
- Final events per stream contain only that stream’s filtered text.

### 13) Zero-length deltas stability

Purpose: Ensure empty partials don’t break state machine or accumulate spurious text.

Setup: Inject `EventPartialCompletion` with `Delta==""` both outside and inside capture.

Assertions:
- No panics; `Completion` evolves correctly; filtered output remains correct.

### 14) MaxCaptureBytes (future-ready)

Purpose: When enforced, oversized payloads should fail gracefully.

Setup: Set `Options.MaxCaptureBytes` to a small number; stream a larger payload.

Assertions:
- The sink drops or marks the block as failed according to design (add once implemented). For now, mark this as skipped if the option is not yet enforced.


## Implementation patterns and helpers

- Prefer small helpers to assemble streams:

```go
func feedParts(t *testing.T, sink *FilteringSink, meta events.EventMetadata, parts []string) string {
    completion := ""
    for _, p := range parts {
        completion += p
        require.NoError(t, sink.PublishEvent(events.NewPartialCompletionEvent(meta, p, completion)))
    }
    require.NoError(t, sink.PublishEvent(events.NewFinalEvent(meta, completion)))
    return completion
}
```

- Common assertions:
- No leakage of `<$…>` or `</$…>` into filtered output.
- `finalRaw` equals exactly the payload (no tag text) for successful completions.
- `OnCompleted(success=false)` occurs exactly once for malformed.
- `itemID` sequence increments per block: `streamID:1`, `streamID:2`, …


## Priorities to add next

1) Open-tag split matrix (complete set of boundaries)
2) Close-tag near-misses inside payload
3) Metadata propagation for typed events (zero metadata → filled by sink)
4) Item context cancellation test using a channel and timeout
5) Interleaved two-stream test


## Notes for reviewers

- The sink uses a lag buffer to hold the last `closeTagLen-1` bytes during capture so no close-tag fragment leaks into payload. Tests that split the close tag across partials are essential to guard against regressions.
- Unknown extractors must leave text untouched; ensure tests verify full tag + payload are forwarded.
- Keep tests deterministic and avoid timeouts unless required for context-cancel checks.


