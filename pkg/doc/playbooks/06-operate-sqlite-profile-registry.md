---
Title: "Operate SQLite-backed profile registry"
Slug: operate-sqlite-profile-registry
Short: Runbook for operating, backing up, and recovering SQLite-backed profile registry storage.
Topics:
  - profiles
  - persistence
  - migration
  - pinocchio
Commands:
  - pinocchio
Flags:
  - profile-registry-dsn
  - profile-registry-db
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: Playbook
---

# Operate SQLite-backed profile registry

## Goal

Run pinocchio/geppetto apps with durable SQLite profile registry storage, and keep safe backup/recovery procedures for profile mutations.

## Before you start

- Confirm your server is wired to registry CRUD handlers (`/api/chat/profiles...`).
- Decide whether you configure by DB file path (`--profile-registry-db`) or full DSN (`--profile-registry-dsn`).
- Ensure only the service account can write the DB file and backup artifacts.

## Step 1: Start with durable profile storage

Preferred file-path form:

```bash
pinocchio web-chat --profile-registry-db ./data/profiles.db
```

Equivalent DSN form:

```bash
pinocchio web-chat \
  --profile-registry-dsn "file:./data/profiles.db?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on"
```

## Step 2: Verify the store is live

```bash
curl -s http://localhost:8080/api/chat/profiles | jq .
```

You should see a JSON profile list, not an empty response body or server error.

## Step 3: Back up before risky changes

Filesystem snapshot backup:

```bash
cp ./data/profiles.db ./data/profiles.db.bak.$(date +%Y%m%d-%H%M%S)
```

Optional SQL check:

```bash
sqlite3 ./data/profiles.db "SELECT slug, updated_at_ms FROM profile_registries ORDER BY slug;"
```

## Step 4: Restore procedure

1. Stop the service writing to the DB.
2. Replace the DB with a known-good backup.
3. Start service.
4. Validate profile list and send one profile-selected chat request.

Example:

```bash
cp ./data/profiles.db.bak.20260223-220000 ./data/profiles.db
```

## Step 5: Post-restore validation

- `GET /api/chat/profiles` returns expected slugs/default.
- `PATCH /api/chat/profiles/{slug}` with stale `expected_version` still returns `409`.
- Chat requests using explicit `profile` or `registry` still resolve correctly.

## Security and permissions

- Restrict DB file write permission to the app service user.
- Treat DB backups as sensitive artifacts because prompts/policy can include internal instructions.
- Keep retention bounded and documented.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `profile_registries` table missing | DB initialized outside profile store migration path | restart with app-managed SQLite store initialization |
| writes fail with busy/lock errors | concurrent writers and missing timeout/WAL settings | use DSN with WAL + busy timeout and avoid external long-running transactions |
| restored DB loads but profiles seem stale | restored snapshot older than expected | verify backup timestamp and repeat restore with newer snapshot |
| API returns 500 after restore | payload/schema corruption in DB row | inspect `payload_json`, restore cleaner snapshot, then reapply desired changes |

## See Also

- [Profile Registry in Geppetto](../topics/01-profiles.md)
- [Migration playbook: legacy profiles.yaml to registry format](05-migrate-legacy-profiles-yaml-to-registry.md)
