---
Title: Port Moments Webchat
Ticket: MO-001-PORT-MOMENTS-WEBCHAT
Status: complete
Topics:
    - webchat
    - moments
    - session-refactor
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/inference/toolhelpers/helpers.go
      Note: ToolExecutor injection + hook points
    - Path: go-go-mento/go/pkg/webchat/connection_pool.go
      Note: WS connection pooling patterns to port
    - Path: go-go-mento/go/pkg/webchat/step_controller.go
      Note: Step mode primitive to integrate
    - Path: go-go-mento/go/pkg/webchat/stream_coordinator.go
      Note: Better websocket/event streaming architecture to port
    - Path: pinocchio/pkg/webchat/conversation.go
      Note: Pinocchio per-conv reader/broadcast to refactor
    - Path: pinocchio/pkg/webchat/engine.go
      Note: Middleware ordering change (reverse application)
    - Path: pinocchio/pkg/webchat/router.go
      Note: Pinocchio webchat entrypoints to extend
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-25T17:34:02.174823149-05:00
WhatFor: ""
WhenToUse: ""
---



# Port Moments Webchat

## Overview

Port the moments/go-go-mento webchat architecture onto the MO-007 Session +
ExecutionHandle design, while converging with Pinocchio’s webchat so we keep one
“best-of-both” implementation (MO-007-native inference + production-grade
websocket/session manager).

## Key Links

- Analysis:
  - [Port go-go-mento webchat to Geppetto session design](./analysis/01-port-go-go-mento-webchat-to-geppetto-session-design.md)
- Design docs:
  - [Event versioning + ordering (from go-go-mento to pinocchio)](./design-doc/01-event-versioning-ordering-from-go-go-mento-to-pinocchio.md)
  - [Step controller integration (from go-go-mento to pinocchio)](./design-doc/02-step-controller-integration-from-go-go-mento-to-pinocchio.md)
- Diary:
  - [Diary](./reference/01-diary.md)
- Reference design:
  - `geppetto/ttmp/2026/01/21/MO-007-SESSION-REFACTOR--session-execution-refactor-unify-sinks-cancellation-tool-loop/design-doc/01-session-refactor-sessionid-enginebuilder-executionhandle.md`

## Status

Current status: **active**

## Topics

- webchat
- moments
- session-refactor
- architecture

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
