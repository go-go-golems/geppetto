---
Title: Robustifying Structured Event Parsing/Filtering in FilteringSink
Slug: review-filtering-sink-code
Short: A pragmatic refactor plan to make tag parsing across partials correct, simpler, and testable.
Date: 2025-10-31
Owners: geppetto/events
Status: Draft
---

## Purpose

This document proposes a focused cleanup and hardening of the structured-data parsing/filtering path implemented in `geppetto/pkg/events/structuredsink/filtering_sink.go`. The goal is to make the sink reliable across partial boundaries, easier to reason about, and well-tested, while preserving current external behavior and keeping extractors in control of payload parsing.

The audience is a developer new to this part of the codebase. We explain the context, highlight current issues, define clear goals/non-goals, and outline a concrete, incremental refactor with tests and a rollout plan.


## Context: What the FilteringSink does today

- The sink intercepts streaming events (partials and final) and detects XML-like structured blocks delimited by `<$name:dtype>` … `</$name:dtype>`.
- When such a block is detected, the raw payload between tags is routed to a registered extractor session for `(name, dtype)`.
- The sink removes the structured block text from the forwarded “filtered” text stream (UI text) and publishes any typed events emitted by extractors in parallel.

References:
- Implementation: `geppetto/pkg/events/structuredsink/filtering_sink.go`
- Overview document: `geppetto/pkg/doc/topics/11-structured-data-event-sinks.md`


## Current Implementation: High-level shape

- Tag detection is done in a character-by-character loop inside `scanAndFilter` with several string builders on the `streamState`:
  - `openTagBuf` accumulates possible `<$...>` for the open tag.
  - `payloadBuf` gathers raw bytes for the active capture.
  - `closeTagBuf` holds a sliding window to detect `</$name:dtype>`.
  - `filteredCompletion` accumulates the filtered output forwarded to UI.
  - `rawSeen` tracks full text seen to compute final-delta.
- Open tag uses a regex (`reOpen`) via `tryParseOpenTag`; close tag uses a manual sliding window string compare.
- Extractor sessions are started on open tag, receive `OnRaw` for streamed bytes, and are completed when a matching close tag is detected or the stream ends.


## Problems Observed (from production and code review)

1. Close-tag leakage across partials
   - Symptom: `</$...>` bytes sometimes appear in the forwarded filtered text or make it into extractor `OnRaw` chunks.
   - Root cause: detection trims current-delta buffers, but cannot retract bytes already forwarded in previous partials. Close tag split across partials (`</` then `$name:dtype>`) is particularly error-prone.

2. Asymmetric parsing strategies increase complexity
   - Open tag is regex-parsed on a fully buffered candidate; close tag is string-compared via a sliding window. Mixed approaches increase branching and edge cases.

3. Implicit state hidden in buffer lengths
   - Whether we are “buffering potential open tag” is inferred from `openTagBuf.Len() > 0` rather than an explicit state. This makes the state machine harder to reason about and test.

4. Redundant buffers and overlapping responsibilities
   - We maintain three-plus builders (`openTagBuf`, `payloadBuf`, `closeTagBuf`, `filteredCompletion`) with overlapping lifetime semantics. This makes correctness and cleanup tricky.

5. Cleanup inconsistencies
   - The code resets a different set of buffers when opening/closing tags, leading to subtle inconsistencies after malformed blocks or early cancellations.

6. Options naming/semantics
   - `OnMalformed` == "forward-raw" reconstructs a tag/payload string, not original bytes. Naming suggests byte-for-byte fidelity which we do not guarantee.

7. Small clarity issues
   - Custom `itoa/strconv` reimplementations add noise; `coalesce` is unnecessary; `MaxCaptureBytes` isn’t enforced.


## Goals

- Correctness across partial boundaries: never leak open/close tag bytes into filtered output or extractor payloads.
- Simpler, explicit state machine with symmetric open/close handling.
- Retain the current external API and extractor ownership of payload parsing.
- Maintain or improve performance; avoid regex in hot path.
- Tighten semantics and naming of options.
- Comprehensive tests for boundary cases and malformed input.


## Non-Goals

- Changing extractor contracts or YAML/JSON parsing helpers.
- Introducing heavyweight parser libraries.
- Supporting nested structured blocks (still out of scope).


## Proposed Design

### 1) Explicit FSM with unified transitions

Introduce an explicit state enum on `streamState`:

- `Idle`
- `ScanningOpenTag` (building `<$name:dtype>`) 
- `Capturing` (streaming payload)
- `ScanningCloseTag` (implemented as a lookbehind ring while still in Capturing; see below)

Notes:
- We remain single-state from the caller’s perspective: `Capturing` covers both payload and close-tag detection. Internally, we add a small lookbehind buffer to robustly detect the close tag without leaking bytes.


### 2) Unified open/close tag parsing without regex

Replace regex open-tag parsing with a small hand-rolled scanner that:
- Confirms the `<$` prefix.
- Reads `name` until `:`; validates allowed charset `[A-Za-z0-9_-]`.
- Reads `dtype` until `>`; validates `[A-Za-z0-9._-]`.
- Emits `(name, dtype)` or returns “need more bytes” / “invalid”.

For close tag, compute the exact expected string once: `"</$" + name + ":" + dtype + ">"`.


### 3) Lookbehind lag buffer during capture (fixes close-tag leakage)

When in `Capturing`, maintain a fixed-size ring buffer `lagBuf` with capacity `K = len(expectedClose) - 1`.

- For each incoming byte `b`:
  - Append to `lagBuf`.
  - If `lagBuf` exceeds `K`, emit the oldest byte to extractor `OnRaw` and append the same byte to `payloadBuf` (this ensures extractor never receives the last `K` bytes until we are certain they aren’t a close tag prefix).
- In parallel, maintain a sliding compare over `lagBuf + current byte` to detect the close tag. If a full close tag is detected:
  - Do NOT emit any remaining `lagBuf` bytes; they are part of the close tag and should be dropped.
  - Finalize the session with the accumulated `payloadBuf` only (which contains no close-tag bytes).
  - Reset capture state.

Impact:
- This small lag fully prevents close-tag bytes from being forwarded to either extractor payload or filtered text, even when the close tag is split across partials.


### 4) Buffer simplification

Keep only these per-stream builders:
- `filteredCompletion` (UI text; outside of captures + any reconstructed text per policy)
- `openTagBuf` (small; only while scanning for open tag)
- `payloadBuf` (raw payload for current capture)
- `lagBuf` (small ring; see above)

Remove `closeTagBuf` and any redundant per-delta accumulators.


### 5) Cleanup rules

- On successful close: clear `openTagBuf`, `payloadBuf`, `lagBuf`; cancel `itemCtx`.
- On malformed at stream end: apply `OnMalformed` policy; clear all capture buffers; cancel `itemCtx`.
- On unknown extractor for a valid open tag: flush `openTagBuf` as normal text and stay `Idle`.


### 6) Option semantics and naming

Replace `OnMalformed string` with a typed enum:

```go
type MalformedPolicy int
const (
  MalformedErrorEvents MalformedPolicy = iota // call OnCompleted(success=false), do not reinsert
  MalformedReconstructText                    // reinsert reconstructed block into filtered text and call OnCompleted(false)
  MalformedIgnore                             // drop captured payload and call OnCompleted(false)
)
```

Rationale: clearer names and explicit type prevent misuse.

Optional future: add `FilterMode` to support alternative UXs (remove/keep/placeholder). Default remains current behavior (remove structured text from filtered output).


### 7) Small code cleanups

- Use `strconv.Itoa` for item IDs; remove custom `itoa/strconv` helpers.
- Remove `coalesce` helper; returning empty slices is fine.
- If we keep `MaxCaptureBytes` in `Options`, enforce it in `Capturing` with a hard cap to prevent unbounded memory.


## Algorithm sketch (pseudocode)

```text
state := Idle
for each byte b in delta:
  switch state:
    case Idle:
      if b == '<': start openTagBuf; state = ScanningOpenTag; else write b -> filteredCompletion

    case ScanningOpenTag:
      append b -> openTagBuf
      parse openTagBuf with small scanner
        - need more: continue
        - invalid: flush openTagBuf to filteredCompletion; state=Idle
        - valid(name,dtype):
            if extractor exists: start itemCtx, session; clear openTagBuf; init payloadBuf, lagBuf; state=Capturing
            else: flush openTagBuf to filteredCompletion; clear; state=Idle

    case Capturing:
      push b into lagBuf; if lagBuf length > K: emit oldest -> payloadBuf and session.OnRaw
      if detectCloseTag(lagBuf + b):
        // drop lagBuf bytes; do not emit
        finalize: session.OnCompleted(success=true, raw=payloadBuf)
        cleanup; state=Idle
```


## Performance considerations

- Remove regex usage in hot path; prefer a small deterministic scanner.
- Keep `lagBuf` capacity tiny (tens of bytes max) — bounded overhead.
- Reuse builders per-stream; avoid per-byte allocations.
- Pre-size `filteredCompletion` and `payloadBuf` when feasible.


## Error handling and context

- Keep stream-level context and item-level context as today; cancel item context on completion or malformed handling.
- The sink remains synchronous; extractors should perform any long-running work in goroutines using the provided context.
- Emit errors to extractors via `OnCompleted(success=false, err)`; avoid sink-side logging except when `Debug` is enabled.


## Backwards compatibility and migration

- Constructors remain, but `Options` changes:
  - `OnMalformed string` -> `MalformedPolicy` (with migration shim that maps old string values to enum).
  - `MaxCaptureBytes` enforced if set; previously a no-op.
- Event types and extractor interfaces unchanged.


## Testing strategy

1. Unit tests for `scanAndFilter` (or new scanner) covering:
   - Open/close tags in one delta
   - Close tag split across multiple partials at every boundary position
   - Non-matching text that resembles tags (e.g., `<<$name:dtype>`, `<$name:dtype>>`)
   - Unknown extractor case (open tag text flushed as normal)
   - Malformed block at stream end for each policy variant
   - `MaxCaptureBytes` limit behavior

2. Property tests
   - Randomly split a known well-formed block across N partials and verify no leakage of tag bytes into payload or filtered output.

3. Fuzz tests
   - Random byte streams including tag-like fragments; assert no panics and invariants hold.

4. Concurrency/cancellation tests
   - Ensure item context is cancelled on completion and no goroutine leaks in extractors (via test extractors that record use).

5. Benchmarks
   - Compare old vs new for typical token rates; ensure we’re on par or faster.


## Rollout plan

1. Implement behind a package-level feature flag or constructor option (default off in first PR).
2. Add tests and benchmarks.
3. Enable by default after burn-in; keep fallback path for one release.
4. Update `11-structured-data-event-sinks.md` with clarified semantics and examples.


## Risks and mitigations

- Risk: Subtle regressions in rare partial boundary placements.
  - Mitigation: Property/fuzz testing; exhaustive boundary tests at close-tag length.
- Risk: Performance regression from added lag buffer.
  - Mitigation: Keep `K` small; microbench; preallocate builders.
- Risk: Behavior changes around malformed handling.
  - Mitigation: Map legacy strings to new enum; document exact behavior.


## Open questions

- Do we want a `FilterMode` (remove|keep|placeholder) now or later?
- Should we enforce a code fence for YAML/JSON at the sink layer (format-aware) or keep that strictly extractor-owned?
- Should `MaxCaptureBytes` default to a safe finite value?


## Summary

We keep the core architecture (sink routes, extractors parse), but replace an error-prone, asymmetric tag detector with a small, explicit state machine and a minimal lookbehind buffer. This removes end-tag leakage, simplifies reasoning, aligns behavior across partials, and gives us a robust foundation backed by thorough tests. The changes are incremental, maintain external contracts, and clarify option semantics.


