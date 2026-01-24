---
Title: Port Pinocchio Webchat to React (Moments parity)
Ticket: PI-003-PORT-TO-REACT
Status: active
Topics:
    - react
    - webchat
    - moments
    - pinocchio
    - geppetto
    - frontend
    - backend
    - websocket
    - redux
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Research ticket documenting the Moments/go-go-mento React webchat architecture (SEM events, sink extractors, WebSocket streaming, Redux timeline widgets) for a Pinocchio React port."
LastUpdated: 2026-01-24T13:52:51.501151796-05:00
WhatFor: "Central entrypoint for PI-003 research artifacts and pointers."
WhenToUse: "Start here to find the architecture doc and the research diary."
---

# Port Pinocchio Webchat to React (Moments parity)

## Overview

This ticket captures a deep, code-grounded analysis of how Moments and go-go-mento implement a React streaming chat UI with rich timeline widgets driven by SEM events over WebSocket, and how the Go backend turns Geppetto events (including structuredsink extractor outputs) into those SEM frames.

## Key Links

- Architecture doc: `analysis/01-moments-react-chat-widget-architecture.md`
- Pinocchio implementation plan: `design-doc/01-pinocchio-react-webchat-refactor-plan.md`
- Research diary: `reference/01-diary.md`
- Prior art (in-repo): `go-go-mento/docs/reference/webchat/`

## Status

Current status: **active**

## Topics

- react
- webchat
- moments
- pinocchio
- geppetto
- frontend
- backend
- websocket
- redux

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
