---
Title: DB Profile Store Ops Notes and GP-01 Release Notes
Ticket: GP-01-ADD-PROFILE-REGISTRY
Status: active
Topics:
  - architecture
  - geppetto
  - pinocchio
  - chat
  - inference
  - persistence
  - migration
  - backend
  - frontend
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles:
  - Path: geppetto/pkg/profiles/sqlite_store.go
    Note: SQLite storage schema, DSN helper, and persistence semantics.
  - Path: pinocchio/cmd/web-chat/main.go
    Note: Glazed profile registry DB/DSN flags used for runtime server configuration.
  - Path: pinocchio/cmd/web-chat/profile_policy.go
    Note: SQLite profile service bootstrap and default registry handling.
  - Path: pinocchio/pkg/webchat/http/profile_api.go
    Note: CRUD API behavior and status code mapping.
  - Path: geppetto/pkg/doc/topics/01-profiles.md
    Note: Registry-first profile behavior and migration guidance.
ExternalSources: []
Summary: Operational runbook for DB-backed profile registries plus GP-01 release notes covering deprecations and rollout/fallback posture.
LastUpdated: 2026-02-23T22:03:00-05:00
WhatFor: Close GP01-903 and GP01-904 with concrete backup/recovery/permissions instructions and release messaging.
WhenToUse: Use when operating SQLite-backed profile storage, planning rollout, or communicating profile-registry migration impacts.
---

# DB Profile Store Ops Notes and GP-01 Release Notes

## Purpose

Provide a repeatable operational procedure for SQLite-backed profile registries and publish final release notes for GP-01.

## Environment Assumptions

- You run `pinocchio web-chat` or another app that wires `geppetto/pkg/profiles.SQLiteProfileStore`.
- You can pass standard Glazed flags on server startup.
- You have filesystem access to the SQLite DB path used by profile registry storage.
- You can run a short maintenance window for backup/restore operations.

## Commands

```bash
# 1) Start web-chat with durable SQLite profile registry storage
pinocchio web-chat \
  --profile-registry-db ./data/profiles.db

# Equivalent DSN form (advanced)
pinocchio web-chat \
  --profile-registry-dsn "file:./data/profiles.db?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on"

# 2) Verify profile API availability
curl -s http://localhost:8080/api/chat/profiles | jq .

# 3) Snapshot backup (filesystem-level copy)
cp ./data/profiles.db ./data/profiles.db.bak.$(date +%Y%m%d-%H%M%S)

# 4) Optional SQL-level verification
sqlite3 ./data/profiles.db "SELECT slug, updated_at_ms FROM profile_registries ORDER BY slug;"

# 5) Create a restore point before risky profile edits
cp ./data/profiles.db ./data/profiles.db.pre-change

# 6) Restore from backup (service stopped)
cp ./data/profiles.db.bak.20260223-220000 ./data/profiles.db
```

## Exit Criteria

- `/api/chat/profiles` responds with expected profiles after startup.
- Backups are timestamped and restorable.
- SQL verification confirms `profile_registries` rows exist and update timestamps move after mutations.
- Restore drill can recover known-good state without schema or slug mismatch errors.

## Notes

### Backup and Recovery Guidance (GP01-903)

- The SQLite store uses one JSON payload per registry row in table `profile_registries`.
- DSN defaults include WAL mode and busy timeout; keep these settings for concurrent access resilience.
- Prefer backup while process is quiescent. If not possible, perform SQLite-consistent backup (`.backup` command) instead of raw copy.
- Always test restore in a non-production environment before relying on a backup procedure.

Recommended restore sequence:

1. Stop service writing to profile DB.
2. Replace DB file with backup snapshot.
3. Restart service.
4. Validate with `GET /api/chat/profiles` and at least one chat request using explicit profile.

### Permissions Guidance

- Restrict write permissions on profile DB to the service user account.
- Restrict backup artifact access because profile prompts/policies may contain sensitive internal instructions.
- Keep backup retention bounded (for example 7/14/30-day tiers) and document deletion policy.

### GP-01 Release Notes (GP01-904)

#### Summary of delivered behavior

- Registry-first profile domain is now canonical across geppetto + pinocchio webchat.
- Reusable profile CRUD HTTP routes are available and mounted in pinocchio and go-go-os server paths.
- Profile version now contributes to runtime fingerprint/rebuild behavior.
- Legacy profile-map YAML can be migrated via `pinocchio profiles migrate-legacy`.

#### Deprecations and migration posture

- Profile-first selection is the recommended workflow.
- Direct low-level flags (`--ai-engine`, `--ai-api-type`) remain supported as transitional override mechanisms, not as primary UX.
- Old app-local profile structures were removed in favor of shared registry interfaces and reusable handlers.

#### Fallback strategy

- No runtime environment toggle path is used for middleware switching.
- Rollback is handled operationally at release level:
  - rollback binary/image version,
  - restore known-good profile store snapshot if needed,
  - rerun profile regression/e2e checks before re-promoting.

#### Validation checklist for release sign-off

- CRUD API smoke tests pass (`list/create/update/delete/default`).
- Resolver precedence tests pass (`path/body/query/runtime/cookie/default`).
- Runtime rebuild-on-profile-version-change tests pass.
- Go-Go-OS profile selection e2e scenarios pass.
