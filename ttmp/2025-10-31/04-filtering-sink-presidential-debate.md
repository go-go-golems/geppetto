---
Title: FilteringSink Implementation Debate - Presidential Style
Slug: filtering-sink-debate
Short: 10 questions about tag parsing failures, debated by code components and developers
Date: 2025-10-31
Owners: geppetto/events
Status: Draft
---

## Debate Participants

**Code Components (personified):**
- **OpenTagBuf** - The buffer that accumulates potential open tag fragments
- **LagBuffer** - The buffer that prevents close-tag leakage
- **HandleFinal** - The function processing final events
- **ScanAndFilter** - The core byte-by-byte parser

**Developers:**
- **Correctness Carl** - Believes in precise spec compliance
- **Performance Petra** - Obsessed with zero-copy and efficiency
- **Simplicity Sam** - "If it's complex, it's wrong"
- **UX Uma** - Cares only about user-visible behavior
- **Test-Driven Tina** - Tests define reality
- **Legacy Larry** - "We can't break existing behavior!"

---

## Question 1: Why doesn't openTagBuf persist across the final event boundary?

**Round 1:**

**OpenTagBuf**: "Look, I'm a `strings.Builder` living in `streamState`. I accumulate `<`, then `$`, then more chars. But when `handleFinal` calls `scanAndFilter` with just the *tail delta*, I start fresh! The previous partials already processed `<`, but the tail is `$x:v1>...`. Nobody's telling scanAndFilter I exist!"

**HandleFinal**: "Not my fault! I compute `delta = full[len(prefix):]` to avoid reprocessing. That's *correct* - we already forwarded filtered output for `<`. If I reprocess the whole stream, we'd emit duplicate filtered deltas!"

**Correctness Carl**: "This is a STATE MACHINE BUG. The tag isn't complete until we see `>`. If the tag spans partial/final boundary, state must carry over. The spec says 'tag recognition is robust across fragments' - this fails that!"

**Performance Petra**: "Wait wait wait. If we reprocess from the beginning on every final, that's O(n²) for a stream with n partials. That's UNACCEPTABLE. We need incremental parsing!"

**Simplicity Sam**: "You're all overthinking this. Just flush openTagBuf to output before processing the tail. Problem solved in one line!"

**UX Uma**: "From the user's perspective, does `<$x:v1>data</$x:v1>` in ONE partial work? Yes. Does it work split at `<` then rest? No. That's INCONSISTENT and users will file bugs."

**Round 2:**

**ScanAndFilter**: "I'm innocent here. I process what I'm given. If HandleFinal gives me a delta starting with `$x:v1>`, that doesn't start with `<$`, so I flush it immediately. The openTagBuf has `<` but that's from a PREVIOUS invocation!"

**Test-Driven Tina**: "The test DOCUMENTS the requirement: split at `<` should work. If it doesn't, that's a P0 bug. Ship the test as XFAIL and fix it in the next sprint."

**Legacy Larry**: "Has anyone shipped code that splits tags at `<`? If not, maybe this is YAGNI. Let's document 'tags must not split before `$`' and move on."

**OpenTagBuf**: "I could survive across scanAndFilter calls - I already do! The issue is HandleFinal processes a tail that doesn't include my buffered `<`. We need to EITHER reprocess from where I started buffering OR flush me into the delta."

**Correctness Carl**: "The fix is clear: track buffering offset. When we buffer `<` at position N, HandleFinal should reprocess from N, not from rawSeen.Len()."

**HandleFinal**: "Fine, but that means I need to track TWO positions: rawSeen (for normal text) and bufferStart (for tag accumulation). That complicates the interface!"

---

## Question 2: Is the lag buffer closing too early on `</$x:v1>>` or working correctly?

**Round 1:**

**LagBuffer**: "I'm designed to hold `closeTagLen-1` bytes to detect the close tag without leaking it into payload. When I have 9 bytes `</$x:v1` and see `>`, I match! That's my JOB. The next `>` is outside the tag - not my problem!"

**Correctness Carl**: "The test expects `</$x:v1>>middle` in the payload. That means the SECOND `>` should close it, not the first. But wait... that doesn't make sense either. The spec says `</$name:dtype>` closes, and that's 10 bytes ending with the first `>`."

**Test-Driven Tina**: "Look at the test assertion: `assert.Contains(ex.last.finalRaw, '</$x:v1>>middle')`. This REQUIRES we treat `</$x:v1>` as PAYLOAD, not as close tag. That's WRONG. The test is broken!"

**UX Uma**: "What does a user WANT? If they write `<$x:v1>data</$x:v1>>more`, they probably mean the tag closes at first `>` and `>more` is outside text. Closing at second `>` would be INSANE."

**Simplicity Sam**: "The lag buffer is too clever. Just scan for the literal string `</$name:dtype>`. If you see it, close. Done. No buffering needed."

**Performance Petra**: "Sam, that's O(closeTagLen) per character! With lag buffer, we check in O(1). The current behavior is CORRECT AND FAST."

**Round 2:**

**ScanAndFilter**: "The algorithm is: when lagBuf has N-1 bytes and next char completes the close tag, match and close. For `</$x:v1>`, that's exactly what happens. The second `>` arrives AFTER we've already closed and reset state."

**LagBuffer**: "If the test wants `</$x:v1>>` to NOT close until the second `>`, then the close tag would need to BE `</$x:v1>>`. But that breaks the spec. The test is WRONG."

**Test-Driven Tina**: "I'll revise the test. The correct assertion is: first block closes on first `>`, payload is `abc`, filtered output has `>middle</$x:v1> suffix` as text. That's THREE assertions, not one."

**Correctness Carl**: "Wait, if the first `>` closes block 1, then `>middle</$x:v1>` is outside text, but `</$x:v1>` looks like a MALFORMED close tag without an open! We need to handle that too!"

**Legacy Larry**: "We're going down a rabbit hole. Real-world LLMs don't generate `</$x:v1>>`. Let's add a note that double-close-chars are undefined behavior."

**UX Uma**: "No way. Users will copy-paste structured blocks. We MUST handle this gracefully, even if 'gracefully' means 'forward it as text because it's malformed'."

---

## Question 3: Should we reprocess the entire stream on final or just the tail?

**Round 1:**

**HandleFinal**: "Currently I compute `delta = full[len(rawSeen):]` to process only new bytes. This prevents duplicate output and is O(1) in memory. Reprocessing everything would mean building filtered output from scratch!"

**Performance Petra**: "EXACTLY. Reprocessing is O(n×m) where n=number of partials, m=average length. For a 10k token response with 500 partials, that's 5 MILLION character operations. HARD NO."

**Correctness Carl**: "But tail-only processing loses state! OpenTagBuf has `<`, the tail has `$x:v1>...`, and they never meet. We need state continuity."

**Simplicity Sam**: "Make scanAndFilter idempotent. Process the whole stream every time. Keep state, but verify it matches. If there's drift, we catch bugs early."

**Test-Driven Tina**: "The test doesn't care about algorithmic complexity. It cares about correctness. If reprocessing is the only way to pass the test, DO IT, then optimize later."

**ScanAndFilter**: "I maintain openTagBuf across calls! It's IN streamState, which persists. The problem is HandleFinal gives me a substring that doesn't include the buffered prefix. I can't work with incomplete information!"

**Round 2:**

**OpenTagBuf**: "Here's a compromise: flush me to filtered output at end of each partial UNLESS I contain a valid tag prefix `<$`. Then persist me. HandleFinal's tail includes everything after my last flush point."

**LagBuffer**: "I do this already! I persist across scanAndFilter invocations. Why can't OpenTagBuf do the same?"

**HandleFinal**: "LagBuffer is different - it's only active during capture state, which happens within scanAndFilter. OpenTagBuf spans MULTIPLE scanAndFilter calls across partials AND final."

**Legacy Larry**: "In v1, did we have this problem? What did we do differently?"

**Correctness Carl**: "V1 didn't exist! This is net-new code. We're making it correct from the start."

**UX Uma**: "Users don't care about v1 or v2. They paste `<$thinking:v1>I wonder...</$thinking:v1>` and expect it to work whether it comes in one chunk or fifty. THAT'S the requirement."

---

## Question 4: Should case-sensitive close tag matching be an error or should it find the correct close tag?

**Round 1:**

**Correctness Carl**: "The spec is clear: tags are case-sensitive. `<$X:v1>` opens, `</$X:v1>` closes. `</$x:v1>` is NOT a match and should be treated as payload text, waiting for the real close tag."

**UX Uma**: "But we HIT final event without seeing `</$X:v1>`, so the block is malformed. The user made a typo. We should emit an error event AND show them what went wrong."

**Test-Driven Tina**: "The test says `success=false` and `final='prefix  suffix'`. That means we drop the captured content and emit error. That's REASONABLE."

**Simplicity Sam**: "Why is case sensitivity even a thing? Make tags case-insensitive and this whole problem disappears."

**LagBuffer**: "I do EXACT string matching on `expectedClose`, which is `</$X:v1>`. When I see `</$x:v1>`, it doesn't match, so I treat it as payload. Working as designed!"

**Legacy Larry**: "Are there existing tags in production? What case do they use? Let's not break them."

**Round 2:**

**ScanAndFilter**: "I store `expectedClose` based on the open tag's actual case. If open is `<$X:v1>`, expectedClose is `</$X:v1>`. When scanning payload, I look for THAT string. Case mismatch means no match - simple."

**Performance Petra**: "Case-insensitive matching is expensive - we'd need lowercase normalization on every byte comparison. Current approach is O(1) per char."

**Correctness Carl**: "The test expects final to be `prefix  suffix`. But the current impl only processes up to the malformed close, so suffix gets dropped! That's the REAL bug - malformed handling doesn't preserve trailing text."

**HandleFinal**: "Oh! When flushMalformed runs, I append its output to filteredCompletion, then immediately emit final. But if there's text AFTER the malformed close tag, that's in the tail... wait, no, we already processed it. Hmm."

**UX Uma**: "From user perspective: they write `<$X:v1>abc</$x:v1> suffix`, the block fails to close properly, they should see `prefix  suffix` in output (block dropped) AND an error event saying 'unclosed tag <$X:v1>'."

**Test-Driven Tina**: "The test currently FAILS because final is `prefix ` not `prefix  suffix`. The suffix is getting lost. THAT'S what we need to fix, not the case sensitivity logic."

---

## Question 5: Should we optimize for common case (tags in single partial) or worst case (tags split across many)?

**Round 1:**

**Performance Petra**: "99% of tags arrive in ONE partial. LLMs stream character-by-character but buffering means partials are 50-500 chars. Tags are 20-50 chars. Optimize for the common case!"

**Correctness Carl**: "But the 1% edge case where tags split is exactly when bugs hide! We MUST handle it correctly even if it's slower."

**Test-Driven Tina**: "We have tests for both. They're separate concerns. Single-partial path can be fast, split-partial path can be correct. Use different code paths!"

**Simplicity Sam**: "ONE code path. It's either always correct or sometimes buggy. I don't care if single-partial is 10% slower if it means zero bugs."

**UX Uma**: "Users won't notice 10% slower parsing. They WILL notice tags mysteriously not parsing when split at character 47 instead of 48. Correctness wins."

**ScanAndFilter**: "I process byte-by-byte EVERY time. There's no fast path. Whether the tag arrives in one delta or ten, I do the same work per byte."

**Round 2:**

**OpenTagBuf**: "I could check: if delta starts with `<$` and ends with `>` and parses validly, FAST PATH - jump straight to capture state. Otherwise, accumulate byte-by-byte."

**LagBuffer**: "I ALWAYS buffer N-1 bytes. Even if the close tag is all in one delta. That's overhead for the common case!"

**Performance Petra**: "Exactly! For single-partial tags, we could detect `</$name:dtype>` as a substring and close immediately, skipping lag buffer."

**Correctness Carl**: "And introduce TWO code paths that must both be correct and tested. Now we have 2× the bugs."

**Legacy Larry**: "The current code works for single-partial tags - those tests pass! Don't break what works while fixing edge cases."

**HandleFinal**: "The real cost is state management. If we optimize for single-partial, we need special case code in ME to detect 'entire stream was one partial, skip state preservation'. That's messy."

---

## Question 6: Is `feedParts` helper hiding bugs by being too convenient?

**Round 1:**

**Test-Driven Tina**: "feedParts assembles completion strings correctly and calls events in order. It's a GOOD abstraction that prevents test bugs."

**Simplicity Sam**: "But it HIDES the fact that completion is cumulative! Some tests need to manually build completion to test edge cases. feedParts is a crutch."

**Correctness Carl**: "Look at the failing tests - they ALL use feedParts. Maybe feedParts is computing deltas wrong?"

**OpenTagBuf**: "No, I checked. feedParts builds completion as `part1 + part2 + ...` and emits deltas `part1`, `part2`, etc. That's CORRECT."

**HandleFinal**: "The issue is feedParts calls final with full completion. But if a test manually builds partials with wrong completion values, it could expose bugs in MY delta computation."

**UX Uma**: "Users don't call feedParts. They call the real API. Tests should use the real API too."

**Round 2:**

**Test-Driven Tina**: "Fine. We'll add variants: `feedPartsWithWrongCompletion`, `feedPartsThenWaitThenFinal`, etc. These test error handling."

**Performance Petra**: "More test helpers means more code to maintain. Keep it simple - one helper for happy path, manual event construction for edge cases."

**ScanAndFilter**: "I don't care how tests call me. I process delta strings. If delta is `<` one time and `$x:v1>` next time, I should handle it. That's what the test is checking."

**Simplicity Sam**: "The BUG is not in feedParts. The bug is in HandleFinal not preserving openTagBuf state across final boundary. feedParts just EXPOSES it."

**Legacy Larry**: "We've used feedParts-style helpers in other tests for years. They're fine. Don't bikeshed."

**Correctness Carl**: "Agreed. Move on. The helper is correct, the implementation is buggy."

---

## Question 7: Should we buffer entire tags before processing or stream them byte-by-byte?

**Round 1:**

**Performance Petra**: "Byte-by-byte is CACHE FRIENDLY. We process 64 bytes, they fit in L1 cache, we're done. Buffering entire tags means allocating, writing, then re-reading."

**Simplicity Sam**: "Buffering makes state machine simple: outside-tag, inside-tag. Two states! Byte-by-byte needs: idle, accumulating-open, capturing, accumulating-close. Four states! More states = more bugs."

**ScanAndFilter**: "I already do byte-by-byte. It works! The issue is state spanning deltas, not the granularity of processing."

**LagBuffer**: "I buffer N-1 bytes to avoid close-tag leakage. That's ESSENTIAL. Without me, we'd emit `</$x:v1` as payload then eat the `>` as close marker. Users would see corrupted payloads!"

**Correctness Carl**: "Buffering entire tags means we can validate them before emitting OnStart. Byte-by-byte means we emit OnStart when seeing `>`, but the tag might be malformed."

**UX Uma**: "Emit OnStart as soon as we see `<$name:dtype>`. If it's malformed later, emit OnCompleted(success=false). Don't delay signals to the UI."

**Round 2:**

**OpenTagBuf**: "I'm a compromise - I buffer potential tags UNTIL they complete or prove invalid. Once I see `>`, I parse and either transition to capture or flush as text."

**HandleFinal**: "The problem with buffering entire tags is: what if a tag is 10KB? We hold 10KB in memory before processing. Byte-by-byte streams it."

**Test-Driven Tina**: "Tests don't care about implementation strategy. They care about: given THIS input sequence, produce THIS output. Both strategies could pass if implemented correctly."

**Performance Petra**: "Real-world tags are 100 bytes max. Buffering 100 bytes is FREE. Let's special-case: if we see `<$`, buffer up to 1KB looking for `>`. If not found, flush as text."

**Simplicity Sam**: "Now you have magic number 1KB and special-case logic. More complexity!"

**Legacy Larry**: "Current approach has been fine for the tests that pass. Don't rewrite the whole engine to fix 3 failing tests."

---

## Question 8: Should final event recompute filtered output from scratch or trust partials?

**Round 1:**

**HandleFinal**: "I trust partials! They emitted filtered deltas, I forwarded them. Final just adds the tail. Recomputing would mean I can't trust my OWN output. That's insane."

**Correctness Carl**: "But if partials had bugs in their filtered output, final would inherit those bugs. Recomputing is a SAFETY NET."

**Test-Driven Tina**: "We test partials separately. If they're buggy, those tests fail. Final can trust them. Don't test the same thing twice."

**UX Uma**: "Users see partials in real-time, then final as confirmation. If final CHANGES the output, that's jarring. They saw 'Hello ' then 'world' then final is 'Hi world'?! Confusing!"

**Performance Petra**: "Recomputing filtered output on final means O(n) work where n=total stream length. We already did that work during partials!"

**ScanAndFilter**: "I build filtered output incrementally. Each delta produces a filtered delta. Concatenating them gives the final result. Math works out."

**Round 2:**

**HandleFinal**: "BUT, what if state transitions happened that affected earlier output? Like we buffered `<` in a partial, then final completes the tag. We need to REMOVE that `<` from earlier filtered output!"

**Correctness Carl**: "EXACTLY. That's why openTagBuf preservation matters. We tentatively emitted `<` as text, but when tag completes, we need to retract it."

**Simplicity Sam**: "Don't emit text tentatively. Buffer it until you KNOW it's not a tag. Only emit confirmed text."

**UX Uma**: "But that adds latency! If we buffer `<` waiting to see if `$` follows, users see lag. Stream it immediately, retract if needed."

**Test-Driven Tina**: "The test shows we DON'T emit `<` in filtered output. When tag completes across partials, both open and close tags disappear. So HandleFinal must handle retraction."

**Legacy Larry**: "This is getting complicated. Can we just document: 'tags must be in single partial for real-time streaming'?"

---

## Question 9: What's the performance impact of these fixes?

**Round 1:**

**Performance Petra**: "Fix #1: Preserve openTagBuf state. Cost: zero - it's already in streamState. Fix #2: Track buffer offset. Cost: one integer. Fix #3: Handle retraction. Cost: potentially rewriting filtered output. That's O(n) where n=buffered text. EXPENSIVE."

**Correctness Carl**: "Who cares? Correctness first, optimize later. If we can't handle a split tag correctly, the feature is BROKEN."

**Test-Driven Tina**: "Measure it. Run benchmarks. If it's <1% overhead, ship it. If it's >10%, optimize. Don't guess."

**Simplicity Sam**: "The simplest fix - reprocess whole stream on final - is also probably fast enough. Modern CPUs process megabytes per millisecond."

**UX Uma**: "Users have 50ms perception threshold. If parsing adds 10ms, they won't notice. If it adds 100ms, they will. Measure!"

**ScanAndFilter**: "I process ~1 char per nanosecond on my machine. A 10k token stream is 10μs. Even with 2× overhead from retraction, that's 20μs. Unmeasurable."

**Round 2:**

**Performance Petra**: "Wait, I calculated wrong. If we DON'T emit tentative text, there's no retraction needed. We only emit confirmed filtered output. Cost is memory for buffering, not CPU."

**HandleFinal**: "Buffering means filteredCompletion lags behind rawSeen. At final, we flush buffers and catch up. One-time cost, only for split tags."

**LagBuffer**: "I already buffer N-1 bytes with no measurable overhead. OpenTagBuf buffering 20 bytes worst-case is the same."

**Legacy Larry**: "So the performance argument is moot? Good. Let's focus on correctness."

**Correctness Carl**: "Exactly. Fix the bugs, measure performance, optimize if needed. Don't prematurely optimize."

**Test-Driven Tina**: "I'll add a benchmark: BenchmarkSinglePartialTag, BenchmarkSplitTagAcross100Partials. We'll see the real cost."

---

## Question 10: Should we ship with failing tests or delay until all pass?

**Round 1:**

**Test-Driven Tina**: "NEVER ship with failing tests. They're red for a reason. Fix them or delete them."

**Correctness Carl**: "Agree. Failing tests mean known bugs. Document them, file issues, but don't merge code that fails its own tests."

**Legacy Larry**: "But the feature WORKS for 90% of cases! Ship it, mark the edge-case tests as TODO, fix them in v2."

**UX Uma**: "What if a user hits the 10% case? They'll file a bug. Better to fix it now than deal with support tickets."

**Simplicity Sam**: "The tests aren't that complex. Two of them are duplicates (split at `<` vs split into many). Fix the root cause and they'll all pass."

**Performance Petra**: "I care about shipping performant code. If the fix makes common cases slow, I'd rather ship with edge-case bugs and optimize later."

**Round 2:**

**HandleFinal**: "I'm the one who needs fixing. Give me a weekend and I'll handle openTagBuf preservation correctly. Don't ship broken code on my account."

**ScanAndFilter**: "Or refactor me to handle deltas differently. I can track buffer offsets. It's not hard, just needs careful thought."

**OpenTagBuf**: "Please don't delete me! I'm useful! Just... use me correctly."

**LagBuffer**: "I'm fine, right? RIGHT?!"

**Test-Driven Tina**: "Mark the tests as XFAIL with `t.Skip()` and a TODO comment. Ship the feature, file issues for each failing test, fix them in the next sprint. Compromise."

**Legacy Larry**: "That's reasonable. We shipped v1 with known limitations. We can do the same for v2."

**Correctness Carl**: "*Heavy sigh* Fine. But I want those issues filed TODAY and assigned to next sprint. No letting this drift."

**UX Uma**: "And we document the limitation: 'Tags may not parse correctly if split at `<` character. This is a known issue and will be fixed.' Users deserve transparency."

**Simplicity Sam**: "Or we could just... fix it now? It's probably a 20-line change."

**Performance Petra**: "Famous last words."

---

## Debate Conclusions

**Consensus points:**
1. OpenTagBuf state needs preservation across partial/final boundary
2. LagBuffer is working correctly
3. Case sensitivity is correct; suffix-dropping is the bug
4. Performance impact is likely negligible
5. Ship with documented limitations if fixes aren't trivial

**No consensus:**
1. Whether to reprocess stream on final or use incremental approach
2. Fast-path optimization for common case
3. Test helper philosophy
4. Ship-now vs fix-first priority

**Next steps:**
1. File issues for each failing test
2. Prototype HandleFinal fix for openTagBuf preservation
3. Run benchmarks to measure performance
4. Decide ship vs delay based on fix complexity

**Moderator**: "And with that, we're out of time! Thank you to all our participants, especially to the code components for their insights into their own implementation. Voters, remember: there are no wrong answers in software architecture, only expensive ones. Good night!"

