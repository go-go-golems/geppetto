---
Title: "Operate SQLite-backed profile registry"
Slug: operate-sqlite-profile-registry
Short: Runbook for operating, backing up, and recovering SQLite-backed profile registry storage.
Topics:
  - profiles
  - persistence
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

Run pinocchio/geppetto apps with durable SQLite profile registry storage, verify schema discovery endpoints, and keep safe backup/recovery procedures for profile mutations.

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

Verify schema discovery endpoints:

```bash
curl -s http://localhost:8080/api/chat/schemas/middlewares | jq .
curl -s http://localhost:8080/api/chat/schemas/extensions | jq .
```

You should get JSON arrays. If the arrays are empty unexpectedly, check application startup wiring for middleware definitions and extension schema registration.

Quick contract checks:

```bash
curl -s http://localhost:8080/api/chat/schemas/middlewares \
  | jq '.[0] | {name,version,display_name,description}'

curl -s http://localhost:8080/api/chat/schemas/extensions \
  | jq 'map(select(.key | startswith("middleware."))) | .[0].key'
```

Expected:

- middleware schema items expose `name`, `version`, `display_name`, `description`, and `schema`,
- extension schema keys use typed-key format (for example `middleware.agentmode_config@v1`).

## Step 3: Validate write-time middleware checks

Unknown middleware names should hard-fail:

```bash
curl -s -X POST http://localhost:8080/api/chat/profiles \
  -H 'content-type: application/json' \
  -d '{"slug":"bad-mw","runtime":{"middlewares":[{"name":"does_not_exist"}]}}'
```

Expect HTTP `400` with a `validation error` mentioning `runtime.middlewares[0].name`.

Schema-invalid middleware payloads should also hard-fail:

```bash
curl -s -X POST http://localhost:8080/api/chat/profiles \
  -H 'content-type: application/json' \
  -d '{"slug":"bad-config","runtime":{"middlewares":[{"name":"agentmode","config":{"unknown":"x"}}]}}'
```

Expect HTTP `400` with a `validation error` mentioning `runtime.middlewares[0].config`.

## Step 3b: Verify typed-key middleware payload write shape (hard cutover)

Middleware config is persisted in profile `extensions` under middleware typed keys.
There is no migration command or fallback route in this flow.

```bash
curl -s -X POST http://localhost:8080/api/chat/profiles \
  -H 'content-type: application/json' \
  -d '{
    "slug":"analyst",
    "runtime":{"middlewares":[{"name":"agentmode","id":"default","enabled":true}]},
    "extensions":{
      "middleware.agentmode_config@v1":{
        "instances":{"id:default":{"default_mode":"financial_analyst"}}
      }
    }
  }' | jq .
```

## Step 4: Back up before risky changes

Filesystem snapshot backup:

```bash
cp ./data/profiles.db ./data/profiles.db.bak.$(date +%Y%m%d-%H%M%S)
```

Optional SQL check:

```bash
sqlite3 ./data/profiles.db "SELECT slug, updated_at_ms FROM profile_registries ORDER BY slug;"
```

## Step 5: Restore procedure

1. Stop the service writing to the DB.
2. Replace the DB with a known-good backup.
3. Start service.
4. Validate profile list and send one profile-selected chat request.

Example:

```bash
cp ./data/profiles.db.bak.20260223-220000 ./data/profiles.db
```

## Step 6: Post-restore validation

- `GET /api/chat/profiles` returns expected slugs/default.
- `GET /api/chat/schemas/middlewares` and `GET /api/chat/schemas/extensions` return expected schema catalogs.
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
| `validation error (runtime.middlewares[*].name)` on create/update | middleware name not registered in app runtime | check middleware definition registry wiring at server bootstrap |
| `validation error (runtime.middlewares[*].config)` on create/update | config payload does not satisfy middleware schema | fetch schema from `/api/chat/schemas/middlewares` and correct payload |

## See Also

- [Profile Registry in Geppetto](../topics/01-profiles.md)
