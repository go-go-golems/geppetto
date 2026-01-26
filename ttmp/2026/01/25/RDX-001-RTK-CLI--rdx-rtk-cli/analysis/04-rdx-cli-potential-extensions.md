---
Title: RDX CLI Potential Extensions
Ticket: RDX-001-RTK-CLI
Status: active
Topics: []
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../../../tmp/remote-devtools/package/src/devTools.ts
      Note: Supported DevTools message types
    - Path: ../../../../../../../../../../../../tmp/remotedev-server/package/README.md
      Note: Report store and GraphQL details
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-26T00:00:00-05:00
WhatFor: ""
WhenToUse: ""
---


# RDX CLI Potential Extensions

## Goal

Document additional features that could make the RDX CLI a full-featured companion to Redux DevTools, with guidance on feasibility and required protocol hooks.

## Extensions by Category

### 1) Historical Sessions and Reports (RDX-002-HIST-REPORTS)

**What:** Query stored reports from the remotedev server (GraphQL) and replay action history.  
**Why:** Enables post-mortem debugging and sharing sessions.  
**How:** Add a `report list` and `report show <id>` command that queries the GraphQL endpoint and optionally converts reports into `STATE`/`ACTION` streams.  
**Dependencies:** remotedev-server report store and GraphQL endpoint.

### 2) Time Travel and Dispatch Controls (RDX-003-DISPATCH-SYNC)

**What:** Add commands that emit DevTools control messages (`DISPATCH`, `UPDATE`, `IMPORT`, `SYNC`).  
**Why:** Enables reset, rollback, jump-to-state, and action replay from the CLI.  
**How:** Implement `rdx dispatch` and `rdx jump --index N`.  
**Dependencies:** Full lifted state model decoding and monitor channel support.

### 3) Action History and Filtering (Unassigned)

**What:** Maintain a rolling action buffer, support filtering by type.  
**Why:** Faster workflows for large action streams.  
**How:** Add `--action-filter` and `--actions-limit` flags.  
**Dependencies:** None beyond existing action decoding.

### 4) State Diff and Snapshot Tools (RDX-004-DIFF-SNAPSHOT)

**What:** Provide diff summaries between consecutive states.  
**Why:** Helps understand state transitions quickly.  
**How:** Add `--diff` to `tail`/`watch`, optionally output JSON Patch.  
**Dependencies:** Local diff library.

### 5) RTK Query Introspection (RDX-005-RTKQ-INTROSPECTION)

**What:** Specialized commands to summarize RTK Query cache entries.  
**Why:** RTK Query state is large and nested; targeted views are valuable.  
**How:** `rdx rtkq list`, `rdx rtkq show <endpoint>`.  
**Dependencies:** Knowledge of RTK Query state schema.

### 6) Multi-Instance Session Management (RDX-006-MULTI-INSTANCE)

**What:** Tag sessions by app name or environment; aggregate multiple instances.  
**Why:** Useful in multi-app or multi-tab workflows.  
**How:** CLI-side tagging, grouping in `list` output.  
**Dependencies:** Consistent app metadata in relay messages.

### 7) Connection Management and Profiles (Unassigned)

**What:** Persistent config profiles (`~/.rdx/config.yaml`).  
**Why:** Easier switching between servers/environments.  
**How:** Add `rdx profile add` / `rdx profile use`.  
**Dependencies:** Glazed config layers or custom config loader.

### 8) Security and Auth (Unassigned)

**What:** TLS, token auth, and secure transport modes.  
**Why:** Remote debugging across machines.  
**How:** Add `--server wss://...` support and token headers.  
**Dependencies:** SocketCluster server TLS config.

### 9) UX Improvements (Unassigned)

**What:** Live TUI or log-format output; colored action types.  
**Why:** Better readability for streaming output.  
**How:** Add a `--tui` mode using bubbletea or a new `rdx ui` subcommand.  
**Dependencies:** TUI framework integration.

### 10) Mobile/React Native Debugging Helpers (Unassigned)

**What:** Helpers for `adb reverse` and Hermes-specific configuration hints.  
**Why:** Remote DevTools in RN frequently fails without guidance.  
**How:** `rdx rn setup` command that prints platform-specific steps.  
**Dependencies:** Platform knowledge from RN docs and community solutions.

## Implementation Prioritization

1) Report store access (high value, moderate effort)  
2) Time travel commands (medium value, high effort)  
3) Action filtering and diffs (high value, low effort)  
4) RTK Query introspection (medium value, medium effort)  
5) TUI mode (nice-to-have)

## References

- https://github.com/zalmoxisus/remotedev-server
- https://github.com/reduxjs/redux-devtools/blob/main/docs/Integrations/Remote.md
- https://www.npmjs.com/package/@redux-devtools/cli
- https://www.scaler.com/topics/react/redux-devtools/
- https://stackoverflow.com/questions/76595014/redux-devtools-with-expo-49-beta-react-native-hermes-engine
