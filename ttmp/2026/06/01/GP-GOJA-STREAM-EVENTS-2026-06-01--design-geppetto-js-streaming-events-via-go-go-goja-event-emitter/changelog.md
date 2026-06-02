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

