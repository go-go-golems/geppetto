---
Title: Investigation diary
Ticket: GEPPETTO-005
DocType: reference
Status: active
Intent: long-term
Topics:
  - geppetto
  - pinocchio
  - coinvault
  - profiles
  - cli
  - configuration
Owners:
  - manuel
Created: 2026-05-13
Updated: 2026-05-13
---

# Investigation diary

## 2026-05-13 — Initial survey and design capture

The investigation started from a practical CLI usability question: where should a `--print-profiles` flag live so a user can see where profiles came from, how profile registries were loaded, and how profile stacks/config overlays were resolved.

I surveyed three repositories from the workspace root:

- `geppetto/`
- `pinocchio/`
- `2026-03-16--gec-rag/` (CoinVault)

The important findings were:

1. Geppetto owns the core engine-profile model and registry abstractions in `pkg/engineprofiles`.
2. Geppetto also owns a reusable CLI bootstrap layer in `pkg/cli/bootstrap`, including profile settings, config loading, default profile registry discovery, profile registry chain construction, and base+profile inference-settings merging.
3. Pinocchio mostly delegates generic profile CLI bootstrapping to Geppetto, but extends it with Pinocchio-specific config documents, inline profiles, and composed imported+inline registries.
4. CoinVault currently mounts Geppetto's generic `profile-settings` section but resolves profile sources using its own `cmd/coinvault/cmds/profile_settings.go` and `internal/webchat.OpenInferenceProfiles(...)` helper instead of the Geppetto bootstrap runtime. CoinVault also has a separate application-profile layer for prompts/tools.
5. The most reusable home for a `--print-profiles` capability is therefore Geppetto, but CoinVault will need a small adapter/wiring step because it does not currently use Geppetto's full CLI bootstrap path.

I then created this ticket in `geppetto/ttmp`:

```bash
docmgr --root ttmp ticket create-ticket \
  --ticket GEPPETTO-005 \
  --title "Profile registry resolution and print-profiles design" \
  --topics geppetto,pinocchio,coinvault,profiles,cli,configuration
```

I created two documents:

- `design-doc/01-profile-registry-resolution-and-print-profiles-implementation-guide.md`
- `reference/01-investigation-diary.md`

The design guide records the current architecture, file references, resolution order, API proposal, command UX, pseudocode, and a concrete implementation plan for a new intern.

## 2026-05-13 — Implemented the first reusable profile introspection slice

I implemented the first useful slice of the `--print-profiles` design.

In Geppetto I added a reusable profile introspection section and report builder:

- `pkg/sections/profile_introspection_section.go`
- `pkg/cli/bootstrap/profile_introspection.go`
- `pkg/cli/bootstrap/profile_introspection_test.go`

The report builder can summarize source entries, loaded registries, profiles, selected/default profiles, optional stack lineage, and optional redacted merged inference settings. The redaction pass replaces sensitive keys such as API keys, tokens, secrets, credentials, passwords, and authorization fields with `***REDACTED***`.

In CoinVault I wired `coinvault chat send --print-profiles` as an early-exit path before database startup and before any LLM run. This lets a user inspect profile registry behavior even when MySQL is not configured. The implementation uses CoinVault's existing `resolveProfileSettings` and `webchat.OpenInferenceProfiles(...)` bridge, then calls Geppetto's reusable report builder/renderer.

Validation commands:

```bash
cd geppetto
go test ./pkg/cli/bootstrap ./pkg/sections ./pkg/engineprofiles -count=1

cd ../2026-03-16--gec-rag
go test ./cmd/coinvault/cmds -count=1
GOWORK=off go test ./cmd/coinvault/cmds -count=1
go build -tags embed -o bin/coinvault ./cmd/coinvault

env -u COINVAULT_HOST -u COINVAULT_DATABASE -u COINVAULT_USER -u COINVAULT_PASSWORD \
  -u GEC_MYSQL_HOST -u GEC_MYSQL_DATABASE -u GEC_MYSQL_RO_USER -u GEC_MYSQL_RO_PASSWORD \
  bin/coinvault chat send \
  --profile-registries ./profile-registry.local.yaml \
  --profile gpt-5-low \
  --print-profiles \
  --run-timeout 1s
```

The smoke test printed profile sources, default/selected profile, registry summaries, and profile summaries without requiring application database settings. A JSON `--print-profile-resolution` run showed `api_keys` redacted.

Pinocchio's inline-profile wrapper remains a follow-up task because it should call Pinocchio's own `profilebootstrap.ResolveCLIProfileRuntime(...)` so `.pinocchio.yml` inline profiles are included.
