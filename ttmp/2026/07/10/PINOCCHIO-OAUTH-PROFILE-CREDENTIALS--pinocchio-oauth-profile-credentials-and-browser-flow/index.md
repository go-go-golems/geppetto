---
Title: Pinocchio OAuth profile credentials and browser flow
Ticket: PINOCCHIO-OAUTH-PROFILE-CREDENTIALS
Status: active
Topics:
    - pinocchio
    - oauth
    - credentials
    - profiles
DocType: index
Intent: long-term
Owners:
    - manuel
RelatedFiles: []
ExternalSources: []
Summary: 'Follow-on host integration plan for profile-backed OAuth credentials, refresh endpoint support, and browser authorization.'
LastUpdated: 2026-07-10T16:40:53.96980044-04:00
WhatFor: 'Implement Pinocchio-owned OAuth credential acquisition, persistence, and Geppetto injection.'
WhenToUse: 'Use when planning or reviewing profile YAML token storage, OAuth refresh, or browser login.'
---

# Pinocchio OAuth profile credentials and browser flow

## Overview

Pinocchio will own OAuth credential lifecycle for profiles that opt into `auth.kind: oauth_bearer`: Authorization Code with PKCE browser login, profile YAML access/refresh/expiry persistence, refresh-endpoint calls, and injection of a renewable source into Geppetto. The ticket is design-ready; implementation starts with locating the Pinocchio profile and CLI packages plus confirming the initial provider contract.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- pinocchio
- oauth
- credentials
- profiles

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
