---
Title: Reusable provider credential lifecycle and transport adapters
Ticket: GEPPETTO-PI-SUBSCRIPTION-CREDENTIAL-ADAPTERS
Status: complete
Topics:
    - geppetto
    - oauth
    - credentials
    - security
DocType: index
Intent: long-term
Owners:
    - manuel
RelatedFiles: []
ExternalSources: []
Summary: Geppetto supplies reusable provider lifecycle and transport primitives; hosts such as Pinocchio supply storage binding, user experience, and import policy.
LastUpdated: 2026-07-14T20:29:03.036021202-04:00
WhatFor: Define provider credential lifecycle ownership between Geppetto and embedding applications.
WhenToUse: Use when implementing reusable login, refresh, status, logout, storage, or provider transport support.
---




# Reusable provider credential lifecycle and transport adapters

## Overview

This ticket defines a reusable provider-credential architecture: Geppetto supplies provider protocol, lifecycle, and transport primitives; hosts such as Pinocchio bind those primitives to their selected storage, browser/CLI experience, and consent policy. Pi is evidence and a possible explicit migration source, never Geppetto’s storage contract.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- geppetto
- oauth
- credentials
- security

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
