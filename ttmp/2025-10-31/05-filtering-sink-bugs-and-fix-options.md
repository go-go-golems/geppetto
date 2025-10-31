---
Title: FilteringSink Bugs and Fix Options
Slug: filtering-sink-bugs-and-fixes
Short: Clear list of current issues with hypotheses and multiple fix strategies
Date: 2025-10-31
Owners: geppetto/events
Status: Draft
---

## Overview

This document summarizes known issues in `FilteringSink` and proposes potential fixes discussed during the debate. Fixes are suggestions; some may be mutually exclusive or require spec decisions.

References:
- Implementation: `geppetto/pkg/events/structuredsink/filtering_sink.go`
- Tests: `geppetto/pkg/events/structuredsink/filtering_sink_test.go`


## FS-1: Open-tag split at '<' not recognized across final boundary

**Symptoms**
- When an open tag is split such that `<` is in one partial and "$name:dtype>" starts the next delta (possibly in final), the tag is not recognized.
- Expected: `OnStart` fires once tag completes, content is captured, and open/close tags are removed from filtered text.

**Repro (tests)**
- `TestFilteringSink_OpenTagSplit_BeforeDollar`
- `TestFilteringSink_OpenTagSplit_MultipleFragments`

**Hypothesis (Root Cause)**
- `handleFinal` processes only the raw tail (`full[len(rawSeen):]`).
- `openTagBuf` in `streamState` holds a leading `<` from a prior partial, but `scanAndFilter` only sees a tail starting with `$`, so it cannot complete `"<$...>"` and drops the partial tag.

**Fix Options**
- Option A: Track `bufferStartOffset` in `streamState` when `openTagBuf` first sees `<`. On final, reprocess from `bufferStartOffset` (not from `rawSeen.Len()`), rebuilding filtered output from there.
  - Pros: Correct incremental semantics; no duplication if carefully managed.
  - Cons: Requires bookkeeping and partial recomputation of filtered output.

- Option B: Prepend `openTagBuf.String()` to the next `delta` before calling `scanAndFilter` (for both partial and final).
  - Pros: Minimal invasive change; keeps scanning incremental.
  - Cons: Must ensure `openTagBuf` is cleared exactly once and `rawSeen`/`filteredCompletion` stay consistent; risk of double-counting if mishandled.

- Option C: On final, re-run `scanAndFilter` from the very beginning (or from last safe flush point) to rebuild `filteredCompletion` deterministically.
  - Pros: Simplest reasoning; guaranteed correctness.
  - Cons: O(stream length) recomputation; may be wasteful for long streams.

- Option D: Change open-text flushing policy: never emit `<` until we either complete `"<$...>"` or prove it's not a tag (i.e., flush only when prefix is not `"<$"`).
  - Pros: Ensures `<` is not leaked to filtered text; reduces need for retraction.
  - Cons: Adds small latency and introduces buffering semantics for `<`.

**Recommended**: B (short-term) + D (policy hardening). Consider A (offset-based partial recompute) for robust long-term behavior.


## FS-2: Trailing outside text lost on malformed close with case mismatch

**Symptoms**
- Input like: `prefix <$X:v1>abc</$x:v1> suffix`
- Expected: malformed block error (`success=false`) and filtered final "prefix  suffix" (captured content dropped, suffix preserved).
- Actual: suffix is lost in some cases; final "prefix ".

**Repro (test)**
- `TestFilteringSink_CaseSensitivityMismatch`

**Hypothesis (Root Cause)**
- While capturing, `scanAndFilter` does not emit outside text. The mismatched close `</$x:v1>` is treated as payload; the remaining " suffix" is seen while still capturing. At final, `flushMalformed` drops the captured content but no re-scan occurs to emit the trailing outside text from the same delta.

**Fix Options**
- Option A: After `flushMalformed` in `handleFinal`, re-run `scanAndFilter` on the remainder of the final `delta` that follows the end of the capture.
  - Pros: Preserves suffix correctly; minimal change to overall structure.
  - Cons: `scanAndFilter` must expose how much of `delta` was consumed while capturing.

- Option B: On final, reprocess from capture start (or from beginning) to ensure consistent filtered output including suffix.
  - Pros: Simplifies handling of malformed edge cases.
  - Cons: Reprocessing overhead.

- Option C: Extend `scanAndFilter` to return an additional `unprocessedTail` string when finalizing a malformed capture at final, which `handleFinal` appends/feeds back.
  - Pros: Explicit control; no recomputation.
  - Cons: API change to a core internal; more complexity.

**Recommended**: A (if we expose a consumed index) or B (simpler to implement now).


## FS-3: Test vs Spec on `</$x:v1>>` (extra '>')

**Symptoms**
- Ambiguity whether `</$x:v1>>` should close on first `>` or treat `</$x:v1>` as payload and close on second `>`.

**Repro (test)**
- `TestFilteringSink_CloseTagNearMiss_ExtraGt`

**Spec Decision**
- Close tag is exactly "</$name:dtype>"; close occurs on the first `>` completing that literal. Any subsequent `>` belongs to outside text.

**Fix Options**
- Option A: Keep implementation as-is; adjust tests to assert: payload is `abc`, filtered output contains `>middle</$x:v1> ...` as outside text.
- Option B: If we want to be lenient and include `</$x:v1>>` in payload, we must change the close detection algorithm to delay close until a non-`>` follows (not recommended; deviates from exact spec and increases complexity).

**Recommended**: A (clarify tests to match spec).


## FS-4: Final recomputation strategy

**Symptoms**
- Multiple edge cases (FS-1, FS-2) arise from tail-only processing in `handleFinal`.

**Fix Options**
- Option A: Tail-only processing (current) + precise state offsets (buffer start; consumed index). Highest complexity but most efficient.
- Option B: Partial recomputation from last safe anchor (e.g., from `bufferStartOffset`).
- Option C: Full recomputation at final; rebuild `filteredCompletion` once.

**Recommended**: Start with B (good balance). If too complex, C is acceptable; measure cost.


## FS-5: MaxCaptureBytes not enforced

**Symptoms**
- `Options.MaxCaptureBytes` exists but is not enforced.

**Repro (test)**
- `TestFilteringSink_MaxCaptureBytes_SkipIfNotImplemented` currently assumes success.

**Fix Options**
- Option A: Enforce during capture: if `payloadBuf.Len()` exceeds threshold, mark capture as malformed (policy-driven), emit `OnCompleted(false)`, and drop or reconstruct text per policy.
- Option B: Soft limit with warnings; continue capturing but log.

**Recommended**: A with clear policy mapping; add tests for limit exceeded.


## FS-6: Consistency guarantees for unknown extractors

**Status**
- Behavior is correct: unknown tags are forwarded verbatim, including payload and tags, both in partials and final.

**Action**
- Keep tests as guard rails: `UnknownExtractor_FlushesAsText`, `UnknownExtractor_SplitTag`, `UnknownExtractor_SplitCloseTag`.


## FS-7: Metadata propagation for typed events

**Status**
- `publishAll` populates missing metadata (ID/RunID/TurnID). Tests confirm.

**Action**
- None; keep as invariant.


## FS-8: Item context lifecycle

**Status**
- Per-item context is cancelled on completion and on malformed handling. Tests confirm.

**Action**
- None; keep as invariant.


## Priorities and Plan

1. FS-1 (Open-tag split at '<'): implement Option B (prepend `openTagBuf` on next delta) and D (flush policy), or Option B alone with careful state reset. Add regression tests (already present).
2. FS-2 (Suffix lost on malformed): implement Option B (reprocess from capture start on final) for correctness; consider Option A later for efficiency.
3. FS-3 (Spec/test alignment): keep exact close tag; ensure tests assert correct behavior.
4. FS-5 (MaxCaptureBytes): implement enforcement with policy mapping and tests.


## Follow-ups

- Benchmark common case vs edge cases to quantify any recomputation overhead.
- Document streaming semantics: buffering rules for `<`, tag split behavior, malformed policies.
- Add invariants as comments in `FilteringSink` to guide future maintenance.


## Backwards Compatibility (not required; candidates to remove)

This feature is new; backwards compatibility is not required. The current code includes a few convenience compat paths that we can simplify:

- Legacy malformed option mapping: `Options.OnMalformed` (string) maps to `Options.Malformed` (enum). We can remove the string field and require the enum.
- EventImpl fallbacks: `handleFinal` and `handlePartial` accept both typed `*EventFinal`/`*EventPartialCompletion` and raw `*events.EventImpl` with payload conversions (`ToText`, `ToPartialCompletion`). If the wider system always emits typed events, we can drop the raw fallbacks.
- `coalesce` helper: minor; keep or inline.

Proposed path: remove `OnMalformed` string and raw `EventImpl` fallbacks once downstreams are confirmed to emit typed events exclusively.
