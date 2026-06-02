# Changelog

## 2026-06-01

- Initial workspace created


## 2026-06-01

Created EventEmitter streaming-events design package: evidence script/source, intern-ready implementation guide, and investigation diary

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-STREAM-EVENTS-2026-06-01--design-geppetto-js-streaming-events-via-go-go-goja-event-emitter/design-doc/01-geppetto-js-streaming-events-design-and-implementation-guide.md — Primary design deliverable
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-STREAM-EVENTS-2026-06-01--design-geppetto-js-streaming-events-via-go-go-goja-event-emitter/reference/01-investigation-diary.md — Chronological investigation diary
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-STREAM-EVENTS-2026-06-01--design-geppetto-js-streaming-events-via-go-go-goja-event-emitter/sources/01-code-evidence.md — Line-numbered evidence excerpts


## 2026-06-01

Validated ticket hygiene after replacing unknown topic metadata and adding frontmatter to generated code evidence

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-STREAM-EVENTS-2026-06-01--design-geppetto-js-streaming-events-via-go-go-goja-event-emitter/scripts/01-collect-evidence.sh — Regenerates evidence with frontmatter
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-STREAM-EVENTS-2026-06-01--design-geppetto-js-streaming-events-via-go-go-goja-event-emitter/sources/01-code-evidence.md — Docmgr-compatible evidence file


## 2026-06-01

Uploaded the Geppetto JS EventEmitter streaming design bundle to reMarkable and verified the remote listing

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-STREAM-EVENTS-2026-06-01--design-geppetto-js-streaming-events-via-go-go-goja-event-emitter/design-doc/01-geppetto-js-streaming-events-design-and-implementation-guide.md — Included in uploaded bundle
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-STREAM-EVENTS-2026-06-01--design-geppetto-js-streaming-events-via-go-go-goja-event-emitter/reference/01-investigation-diary.md — Included in uploaded bundle


## 2026-06-01

Clarified the streaming design to prefer builder-level EventEmitter sinks first and documented the handle.on late-listener race

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-STREAM-EVENTS-2026-06-01--design-geppetto-js-streaming-events-via-go-go-goja-event-emitter/design-doc/01-geppetto-js-streaming-events-design-and-implementation-guide.md — Updated EventEmitter API and implementation phase ordering
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-STREAM-EVENTS-2026-06-01--design-geppetto-js-streaming-events-via-go-go-goja-event-emitter/reference/01-investigation-diary.md — Recorded API correction


## 2026-06-01

Clarified that synchronous agent.run cannot provide live JavaScript EventEmitter streaming because it blocks the runtime owner; a non-blocking stream/runAsync path is still needed

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-STREAM-EVENTS-2026-06-01--design-geppetto-js-streaming-events-via-go-go-goja-event-emitter/design-doc/01-geppetto-js-streaming-events-design-and-implementation-guide.md — Updated execution semantics section
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-STREAM-EVENTS-2026-06-01--design-geppetto-js-streaming-events-via-go-go-goja-event-emitter/reference/01-investigation-diary.md — Recorded run-vs-stream clarification


## 2026-06-01

Implemented builder-level EventEmitter sinks and agent.runAsync with typed jsevents manager resolver, tests, declarations, docs, and example runner promise waiting

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_agent.go — runAsync implementation and cancellation
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_event_emitters.go — EventEmitter-backed EventSink adapter
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/runtime/runtime.go — Installs jsevents manager and passes typed resolver
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-STREAM-EVENTS-2026-06-01--design-geppetto-js-streaming-events-via-go-go-goja-event-emitter/reference/01-investigation-diary.md — Recorded implementation diary


## 2026-06-01

Committed EventEmitter runAsync implementation (commit 35c994e570bfb7caaecf4aba7fbc7bac7aae8f3c)

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_agent.go — runAsync implementation commit
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_event_emitters.go — EventEmitter sink implementation commit
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-STREAM-EVENTS-2026-06-01--design-geppetto-js-streaming-events-via-go-go-goja-event-emitter/tasks.md — Marked implementation phase commits


## 2026-06-01

Added JavaScript EventEmitter runAsync examples and expanded JS API/user/tutorial docs

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/examples/js/geppetto/32_event_emitter_progress_summary.js — Single-run EventEmitter progress summary example
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/examples/js/geppetto/33_event_emitter_multiturn_run_async.js — Multi-turn runAsync EventEmitter example
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/doc/topics/13-js-api-reference.md — Expanded public JS API reference for runAsync events
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/doc/topics/14-js-api-user-guide.md — Added live-events usage section
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/doc/tutorials/05-js-api-getting-started.md — Added runAsync tutorial step


## 2026-06-02

Fixed geppetto-js-run promise detection so async EventEmitter examples wait for completion and surface missing profile errors

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/cmd/examples/geppetto-js-run/main.go — Exports returned JS Promise values on the owner thread before waiting
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-STREAM-EVENTS-2026-06-01--design-geppetto-js-streaming-events-via-go-go-goja-event-emitter/reference/01-investigation-diary.md — Recorded no-output diagnosis


## 2026-06-02

Aligned go-go-goja dependency to v0.7.2 for HostServices AssetResolver compatibility while fixing promise waiting

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/go.mod — go-go-goja v0.7.2 dependency alignment
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/provider/provider_test.go — fake host implements AssetResolver


## 2026-06-02

Added intern-focused EventEmitter runAsync code review with findings and cleanup plan

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_agent.go — Reviewed runAsync owner-thread and cancellation issues
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_event_emitters.go — Reviewed EventEmitter sink lifecycle issues
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-STREAM-EVENTS-2026-06-01--design-geppetto-js-streaming-events-via-go-go-goja-event-emitter/analysis/01-eventemitter-runasync-code-review-and-intern-guide.md — Detailed EventEmitter runAsync review


## 2026-06-02

Uploaded EventEmitter runAsync code review to reMarkable at /ai/2026/06/01/GP-GOJA-STREAM-EVENTS-2026-06-01

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-STREAM-EVENTS-2026-06-01--design-geppetto-js-streaming-events-via-go-go-goja-event-emitter/analysis/01-eventemitter-runasync-code-review-and-intern-guide.md — Uploaded review source


## 2026-06-02

Added research logbook cataloging useful, stale, and update-needed resources for EventEmitter runAsync work

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-STREAM-EVENTS-2026-06-01--design-geppetto-js-streaming-events-via-go-go-goja-event-emitter/reference/01-investigation-diary.md — Recorded logbook creation
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-STREAM-EVENTS-2026-06-01--design-geppetto-js-streaming-events-via-go-go-goja-event-emitter/reference/02-research-logbook.md — Research logbook source


## 2026-06-02

Uploaded research logbook to reMarkable at /ai/2026/06/01/GP-GOJA-STREAM-EVENTS-2026-06-01

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-STREAM-EVENTS-2026-06-01--design-geppetto-js-streaming-events-via-go-go-goja-event-emitter/reference/02-research-logbook.md — Uploaded research logbook

