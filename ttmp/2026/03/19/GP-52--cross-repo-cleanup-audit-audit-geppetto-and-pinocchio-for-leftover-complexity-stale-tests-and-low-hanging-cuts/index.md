---
Title: CROSS-REPO-CLEANUP-AUDIT -- audit geppetto and pinocchio for leftover complexity stale tests and low-hanging cuts
Ticket: GP-52
Status: active
Topics:
    - cleanup
    - architecture
    - geppetto
    - pinocchio
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Cross-repo audit of Geppetto and Pinocchio focused on leftover complexity, stale compatibility layers, and test surfaces that can likely be simplified or removed."
LastUpdated: 2026-03-19T14:22:00-04:00
WhatFor: "This ticket stores the cross-repo cleanup audit for Geppetto and Pinocchio, including architecture context, candidate deletions, test triage, and the investigation diary needed to continue the cleanup later."
WhenToUse: "Use when planning maintenance work across Geppetto and Pinocchio, onboarding a new contributor into the current architecture, or deciding whether older compatibility and test surfaces should be preserved or removed."
---

# CROSS-REPO-CLEANUP-AUDIT -- audit geppetto and pinocchio for leftover complexity stale tests and low-hanging cuts

## Overview

This ticket captures a cross-repository audit of historical complexity in `geppetto/` and `pinocchio/`. The deliverable is intentionally written for a new intern or tired reviewer: it explains what each repository is for, how the two fit together, which parts of the current system appear to be long-lived residue versus legitimate core complexity, and which tests look worth keeping, splitting, or cutting once migrations are complete.

The central architectural conclusion is simple. Geppetto should own reusable runtime/profile machinery. Pinocchio should own application defaults, product-facing composition, and app-specific compatibility policy. The strongest cleanup opportunities are the places where that ownership boundary has blurred over time.

## Key Links

- Design doc: [design-doc/01-geppetto-and-pinocchio-cleanup-audit-and-intern-guide.md](./design-doc/01-geppetto-and-pinocchio-cleanup-audit-and-intern-guide.md)
- Investigation diary: [reference/01-investigation-diary.md](./reference/01-investigation-diary.md)
- Tasks: [tasks.md](./tasks.md)
- Changelog: [changelog.md](./changelog.md)

## Status

Current status: **active**

## Topics

- cleanup
- tech-debt
- geppetto
- pinocchio

## Tasks

See [tasks.md](./tasks.md) for the current task list and suggested follow-up cleanup tracks.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions on the audit packet.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
