---
Title: Cross-repository release checklist for Geppetto observability consumers
Ticket: GP-CODE-REVIEW
DocType: playbook
Topics:
  - release
  - dependency-alignment
  - observability
  - pinocchio
  - sessionstream
Status: active
LastUpdated: 2026-05-07
---

# Cross-repository release checklist for Geppetto observability consumers

This checklist resolves the release risk called out in section 6.8 of the GP-CODE-REVIEW guide: Pinocchio's web-chat debug path consumes local Geppetto observability APIs and local Sessionstream observer APIs, while `make lintmax` intentionally validates with `GOWORK=off` against pinned module versions.

## Current dependency floor

At the time this checklist was written, Pinocchio pins:

- `github.com/go-go-golems/geppetto v0.11.20`
- `github.com/go-go-golems/sessionstream v0.0.4`

Pinocchio requires newer releases that include:

- Geppetto:
  - `github.com/go-go-golems/geppetto/pkg/observability`
  - OpenAI Responses observer options and provider-ID event propagation
- Sessionstream:
  - pipeline observer APIs such as `PipelineRecord` and `WithPipelineObserver`
  - websocket transport observer APIs such as `TransportRecord`, `TimelineEntitySummary`, `WithTransportObserver`, and transport stage constants used by web-chat debug tests

## Release order

1. Finish and validate the Sessionstream observer API.
2. Tag or otherwise publish the Sessionstream version that contains the observer API.
3. Update Pinocchio to depend on that Sessionstream version.
4. Finish and validate the Geppetto observability API.
5. Tag or otherwise publish the Geppetto version that contains `pkg/observability` and OpenAI Responses instrumentation.
6. Update Pinocchio to depend on that Geppetto version.
7. Re-run Pinocchio validation with workspace mode disabled.
8. Remove any temporary `replace` directives before release.

## Pre-tag validation

Run from each repository before tagging.

### Sessionstream

```bash
cd sessionstream
git status --short
go test ./...
```

The status should contain only intentional files. Do not tag with partially generated websocket transport changes.

### Geppetto

```bash
cd geppetto
git status --short
go test ./...
make lintmax
```

If `make lintmax` is not the repository's current lint target, use the repository pre-commit lint command instead.

### Pinocchio workspace sanity check

```bash
cd pinocchio
git status --short
go test ./cmd/web-chat ./cmd/web-chat/app ./pkg/chatapp/plugins -count=1
```

This confirms the integration still works before switching to pinned-module validation.

## Pinocchio dependency update

After Sessionstream and Geppetto are published:

```bash
cd pinocchio
go get github.com/go-go-golems/sessionstream@<published-sessionstream-version>
go get github.com/go-go-golems/geppetto@<published-geppetto-version>
go mod tidy
```

Review the module diff carefully:

```bash
git diff -- go.mod go.sum
```

Only expected dependency changes should appear.

## GOWORK=off release gate

Run Pinocchio without workspace assistance:

```bash
cd pinocchio
GOWORK=off go test ./...
GOWORK=off go build ./...
make lintmax
```

`make lintmax` already sets `GOWORK=off` for golangci-lint in this branch, but running explicit `GOWORK=off go test ./...` first gives a faster dependency-resolution failure if a tag is missing required APIs.

## Web-chat debug smoke gate

Once pinned-module validation passes, run the focused debug path that uses both dependency updates:

```bash
cd pinocchio
go test ./cmd/web-chat ./cmd/web-chat/app ./pkg/chatapp/plugins -count=1
```

If credentials are available, also run the browser-backed GP-OBSERVABILITY smoke and confirm the SQLite export contains:

- `geppetto_record_count > 0`
- `geppetto_provider_events > 0`
- `geppetto_emitted_events > 0`
- `backend_item_id` and `frontend_item_id` populated in `geppetto_reasoning_to_frontend` for reasoning deltas

## Temporary replace policy

Temporary local `replace` directives are acceptable only for branch-local diagnosis. Before release:

```bash
cd pinocchio
go mod edit -dropreplace github.com/go-go-golems/geppetto || true
go mod edit -dropreplace github.com/go-go-golems/sessionstream || true
go mod tidy
GOWORK=off go test ./...
make lintmax
```

Do not merge release branches with local filesystem replaces.

## Failure triage

| Symptom | Likely cause | Fix |
| --- | --- | --- |
| `no required module provides package github.com/go-go-golems/geppetto/pkg/observability` | Pinocchio still points at a Geppetto version before observability was added | Publish/update Geppetto dependency |
| `undefined: sessionstream.PipelineRecord` | Pinocchio still points at a Sessionstream version before pipeline observers | Publish/update Sessionstream dependency |
| `undefined: wstransport.TransportRecord` | Pinocchio still points at a Sessionstream version before websocket transport observers | Publish/update Sessionstream dependency |
| Workspace tests pass but `GOWORK=off` fails | Local workspace contains APIs absent from pinned versions | Update `go.mod`/`go.sum` to published versions and rerun |
| Generated websocket files dirty unexpectedly | A generation hook or stale generated output rewrote Sessionstream files | Re-run generation intentionally in Sessionstream, validate there, and commit/tag from a clean state |

## Exit criteria

The release risk is resolved when:

- Pinocchio `go.mod` points to published Geppetto and Sessionstream versions with the required APIs.
- Pinocchio passes `GOWORK=off go test ./...`.
- Pinocchio passes `make lintmax` without `--no-verify`.
- No local filesystem `replace` directives remain.
- The web-chat debug/reconcile tests pass against pinned modules.
