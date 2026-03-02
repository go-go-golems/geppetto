---
Title: Go runner max_tokens stop-reason propagation and loop continuation hardening
Ticket: MEN-TR-005
Status: active
Topics:
    - temporal-relationships
    - geppetto
    - stop-policy
    - claude
    - extraction
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/extractor/gorunner/run.go
      Note: Removed step settings API key inference from environment
    - Path: ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/design-doc/03-inference-result-implementation-plan-runinferencewithresult-wrapper-and-metadata-contract-migration.md
      Note: Latest implementation-plan design deliverable
    - Path: ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/design-doc/04-env-api-key-fallback-removal-postmortem.md
      Note: Detailed postmortem for env credential fallback removal
    - Path: ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/playbook/01-credential-and-provider-wiring-playbook-for-js-and-go-runner.md
      Note: Operational playbook for deterministic JS + go runner credential wiring
    - Path: ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/scripts/01-repro-max-tokens-stop-reason.sh
      Note: Repro no longer requires ANTHROPIC_API_KEY env preflight
ExternalSources: []
Summary: Ticket workspace for architecture research and implementation planning to make stop_reason propagation deterministic across streaming engines and Go loop control.
LastUpdated: 2026-03-02T15:40:20-05:00
WhatFor: Coordinate analysis, diary, and delivery artifacts for MEN-TR-005.
WhenToUse: Use when implementing or reviewing stop_reason contract fixes across geppetto engines and temporal-relationships loop policy.
---






# Go runner max_tokens stop-reason propagation and loop continuation hardening

## Overview

This ticket captures deep architecture analysis and implementation guidance for a cross-layer stop-reason contract issue. The core gap is that provider stream events can report `max_tokens`, while `turn.metadata.stop_reason` is not consistently populated by all engines, causing loop-stop ambiguity.

## Key Links

- Primary design guide (stop-reason propagation):
  - `design-doc/01-max-tokens-stop-reason-propagation-architecture-and-intern-implementation-guide.md`
- Secondary design guide (inference-result signaling architecture):
  - `design-doc/02-inference-result-signaling-architecture-study-turn-metadata-sections-events-and-alternative-contracts.md`
- Tertiary design guide (InferenceResult implementation plan):
  - `design-doc/03-inference-result-implementation-plan-runinferencewithresult-wrapper-and-metadata-contract-migration.md`
- Quaternary design guide (env fallback removal postmortem):
  - `design-doc/04-env-api-key-fallback-removal-postmortem.md`
- Operational credential/provider wiring playbook:
  - `playbook/01-credential-and-provider-wiring-playbook-for-js-and-go-runner.md`
- Investigation diary:
  - `reference/01-investigation-diary.md`
- Reproduction scripts:
  - `scripts/01-repro-max-tokens-stop-reason.sh`
  - `scripts/02-inventory-inference-result-signals.sh`

## Status

Current status: **active**

1. Research completed and documented.
2. Reproduction and inventory scripts created.
3. Bookkeeping and delivery pending completion checks.

## Tasks

See [tasks.md](./tasks.md) for current checklist.

## Changelog

See [changelog.md](./changelog.md) for step-by-step updates.

## Structure

- `design-doc/` architecture and implementation guidance
- `reference/` chronological diary
- `scripts/` reproducible experiments
- `sources/` raw ticket source material (if needed)
- `archive/` deprecated artifacts
