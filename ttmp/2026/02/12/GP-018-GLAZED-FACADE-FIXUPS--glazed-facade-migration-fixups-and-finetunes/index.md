---
Title: Glazed facade migration fixups and finetunes
Ticket: GP-018-GLAZED-FACADE-FIXUPS
Status: complete
Topics:
    - geppetto
    - glazed
    - migration
    - architecture
    - infrastructure
    - pinocchio
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/llm-runner/api.go
      Note: Run API fixes for missing-run status and streaming parse
    - Path: pkg/doc/tutorials/05-migrating-to-geppetto-sections-and-values.md
      Note: Primary migration help page for users
    - Path: pkg/sections/sections.go
      Note: Core geppetto sections middleware entrypoint
    - Path: pkg/security/outbound_url.go
      Note: Outbound URL security hardening for IPv6 zone-literal hosts
ExternalSources: []
Summary: Consolidated follow-up fixes and finetunes after the glazed facade migration, including symbol hard-cut, docs refresh, and validation.
LastUpdated: 2026-02-25T17:31:19.218395649-05:00
WhatFor: Track small but high-impact migration cleanups and ensure both geppetto and pinocchio are stable on sections/values APIs.
WhenToUse: Use this ticket to review facade migration hard-cut details, supporting commits, and validation evidence.
---





# Glazed facade migration fixups and finetunes

## Overview

This ticket captures the post-migration cleanup work that converted remaining Geppetto/Pinocchio code paths to the sections/values facade API and removed old compatibility shims. It also tracks the documentation sweep and release/lint stability checks needed after the hard-cut.

## Scope

- Hard-cut API migration in `geppetto` from layer/parameter naming to section/value naming.
- Consumer migration in `pinocchio` away from removed geppetto legacy symbols.
- Follow-up docs updates so default help pages show new symbols.
- Validation via `go test` and `make lint` in both repositories.

## Delivered Commits

- `pinocchio`: `95e0c4b5a42af101b87d604ca510fab7d5855c9d`
  - migrate pinocchio to geppetto sections/values facade
- `geppetto`: `53af798dca730ca7c4edd11bde5cdbd3627800c3`
  - migrate geppetto to sections/values facade hard-cut
- `geppetto`: `db090cce0430fbbc10e81c5a5d86e587c7d3460b`
  - fix security URL validation and llm-runner run parsing behavior

## Task Status

See [tasks.md](./tasks.md). All scoped tasks are checked complete.

## Detailed Journal

See [reference/01-diary.md](./reference/01-diary.md) for a retroactive detailed diary with prompt context, failures, decisions, and validation commands.

## Changelog

See [changelog.md](./changelog.md).
