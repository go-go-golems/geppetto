# Tasks

## TODO


- [x] Define JS hook contract for toolloop lifecycle: beforeToolCall, afterToolCall, onToolError
- [x] Define hook context payload schema (turn snapshot, tool name, args, result/error, attempt)
- [x] Define hook return protocol (continue, modify args, retry, abort) with validation
- [x] Implement hook registration surfaces in builder/tools APIs
- [x] Wire hooks into tool execution path around Go and JS tool invocation
- [x] Implement retry/timeout policy controls and deterministic max-attempt enforcement
- [x] Define and implement callback error handling policy (fail-open vs fail-closed)
- [x] Add tests for success path, mutation path, retry path, and error/abort path
- [x] Create smoke script demonstrating hook-driven arg rewrite and retry
- [x] Run tests/script and update diary/changelog with behavior traces
