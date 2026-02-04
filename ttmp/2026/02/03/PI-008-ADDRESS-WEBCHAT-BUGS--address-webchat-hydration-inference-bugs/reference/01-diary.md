---
Title: Diary
Ticket: PI-008-ADDRESS-WEBCHAT-BUGS
Status: active
Topics:
    - webchat
    - backend
    - frontend
    - bugfix
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/ttmp/2026/02/03/PI-008-ADDRESS-WEBCHAT-BUGS--address-webchat-hydration-inference-bugs/scripts/fetch_timeline.py
      Note: Fetch /timeline payload for hydration analysis
    - Path: geppetto/ttmp/2026/02/03/PI-008-ADDRESS-WEBCHAT-BUGS--address-webchat-hydration-inference-bugs/scripts/inspect_timeline_db.py
      Note: Inspect SQLite timeline versions for conv
    - Path: geppetto/ttmp/2026/02/03/PI-008-ADDRESS-WEBCHAT-BUGS--address-webchat-hydration-inference-bugs/scripts/print_timeline_order.py
      Note: Print hydrated entity order for conv
    - Path: pinocchio/pkg/webchat/stream_coordinator.go
      Note: Time-based seq fallback for missing stream IDs (commit fd7c65c)
    - Path: pinocchio/pkg/webchat/stream_coordinator_test.go
      Note: Updated fallback seq test (commit fd7c65c)
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-03T21:00:00-05:00
WhatFor: ""
WhenToUse: ""
---


# Diary

## Goal

Track investigation and fixes for webchat hydration and inference ordering bugs, capturing evidence and implementation steps.

## Step 1: Capture Hydration Ordering Evidence

I gathered on-disk evidence for the hydration ordering issue by fetching the `/timeline` JSON and inspecting the SQLite timeline store. This established that assistant events were stored with tiny sequence numbers while user messages were stored with time-based versions, explaining why user messages appear at the end during hydration.

### Prompt Context

**User prompt (verbatim):** "Ok, let's use MCP and CLI tools to evaluate the hydration and timeline behavior of await fetch(\"http://localhost:8080/timeline?conv_id=cac78e87-88e0-4258-b1a0-ec60d200d53e\", {
    \"credentials\": \"include\",
    \"headers\": {
        \"User-Agent\": \"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:147.0) Gecko/20100101 Firefox/147.0\",
        \"Accept\": \"*/*\",
        \"Accept-Language\": \"en-US,en;q=0.9\",
        \"Sec-Fetch-Dest\": \"empty\",
        \"Sec-Fetch-Mode\": \"cors\",
        \"Sec-Fetch-Site\": \"same-origin\",
        \"Priority\": \"u=4\"
    },
    \"referrer\": \"http://localhost:8080/?conv_id=cac78e87-88e0-4258-b1a0-ec60d200d53e\",
    \"method\": \"GET\",
    \"mode\": \"cors\"
});"

**Assistant interpretation:** Use local tooling to fetch the timeline payload and analyze entity ordering to understand the hydration bug.

**Inferred user intent:** Identify the ordering mismatch causing user messages to appear below assistant responses, and do so with concrete evidence.

**Commit (code):** N/A

### What I did
- Created ticket `PI-008-ADDRESS-WEBCHAT-BUGS` and added a reference diary.
- Saved CLI scripts in the ticket for timeline fetching and inspection.
- Captured `/timeline` payload for conv `cac78e87-88e0-4258-b1a0-ec60d200d53e` into `sources/timeline_cac78e87.json`.
- Inspected `/tmp/test.db` to enumerate timeline entity versions.

### Why
- Needed proof of the version ordering in storage vs. hydration output to pinpoint the mismatch.

### What worked
- JSON hydration order and SQLite versions showed user messages at massive time-based versions while assistant events had tiny incremental versions.

### What didn't work
- N/A

### What I learned
- When stream IDs are missing, the StreamCoordinator fallback uses small local counters, producing versions that sort before time-based user messages.

### What was tricky to build
- Correlating the stored version ordering with the hydrated JSON ordering required looking at both `/timeline` output and raw SQLite rows.

### What warrants a second pair of eyes
- Confirm that all environments missing `xid`/`redis_xid` metadata should adopt time-based sequence generation.

### What should be done in the future
- N/A

### Code review instructions
- Review scripts in `geppetto/ttmp/2026/02/03/PI-008-ADDRESS-WEBCHAT-BUGS--address-webchat-hydration-inference-bugs/scripts/`.
- Inspect `sources/timeline_cac78e87.json` and `/tmp/test.db` for version ordering.

### Technical details
- Assistant messages stored with versions like `11`, `41`, `65` while user messages were stored at `1770169937948000000+`, causing hydration order to list assistant messages first.

## Step 2: Use Time-Based Seq Fallback for StreamCoordinator

I updated the StreamCoordinator sequence generation to ensure that when stream IDs are missing, event sequences are time-based and monotonic. This aligns assistant event versions with the time-based user message versions so hydration ordering matches actual conversation order.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Fix the root cause by generating proper sequence numbers for assistant events even when stream IDs are absent.

**Inferred user intent:** Ensure hydration and streaming display order is correct (user messages before assistant replies).

**Commit (code):** fd7c65c — "Use time-based seq when stream IDs missing"

### What I did
- Added `StreamCoordinator.nextSeq` to produce monotonic time-based sequence numbers when stream IDs are missing.
- Updated the fallback test to assert time-based sequences rather than `1`.
- Ran `go test ./pkg/webchat -count=1`; pre-commit ran repo-wide tests, codegen, lint, and vet.

### Why
- The timeline store orders by version, and assistant events without stream IDs were using tiny counter-based versions, placing user messages at the end of hydration.

### What worked
- Tests passed with the new seq generation, and the logic now aligns assistant event versions with time-based user entries.

### What didn't work
- N/A

### What I learned
- The fallback sequencing logic needs to share the same scale as time-based upserts to maintain consistent ordering.

### What was tricky to build
- Preserving monotonic ordering even when stream IDs are present, while preventing regressions when the fallback path is used.

### What warrants a second pair of eyes
- Confirm whether mixing stream-ID-derived sequences with time-based fallback should always prefer monotonic ordering over raw stream IDs.

### What should be done in the future
- N/A

### Code review instructions
- Start at `pinocchio/pkg/webchat/stream_coordinator.go` and `pinocchio/pkg/webchat/stream_coordinator_test.go`.
- Validate with `go test ./pinocchio/pkg/webchat -count=1`.

### Technical details
- Fallback seq now uses `time.Now().UnixMilli() * 1_000_000` with monotonic enforcement, replacing the small local counter default.
