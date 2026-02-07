# Tasks

## Run → Session Removal Execution Checklist

### 0) Scope lock + baseline
- [x] Confirm migration policy for this ticket: hard remove `run_id` aliases (no compatibility fields) except intentional non-identifier verb uses (`RunLoop`, `go run`, etc.)
- [x] Refresh run-term inventory against current code after PI-018 changes and capture final target files
- [x] Record baseline API/CLI examples that still emit `run_id` (for before/after verification)

### 1) Backend core model rename
- [ ] Rename `Conversation.RunID` → `Conversation.SessionID` and update constructors/usages
- [ ] Rename run-centric router symbols (e.g. `startRunForPrompt`) to session/inference terminology
- [ ] Remove `run_id` fields from `/chat` and queue responses, keep only `session_id`
- [ ] Update runtime/log keys to `session_id` (remove legacy `run_id` logging where still tied to session)

### 2) Turn store + SQL migration
- [ ] Rename `TurnSnapshot.RunID` → `TurnSnapshot.SessionID`
- [ ] Rename `TurnQuery.RunID` → `TurnQuery.SessionID`
- [ ] Migrate turn-store SQLite schema from `run_id` column/index names to `session_id`
- [ ] Update query/build logic and tests for new session field names

### 3) AgentMode persistence/session references
- [ ] Rename AgentMode model fields from `RunID` to `SessionID`
- [ ] Migrate AgentMode SQLite schema/indexes from `run_id` to `session_id`
- [ ] Update AgentMode service/middleware log/context usage to `session_id`

### 4) API surface cleanup
- [ ] Remove `run_id` query parameter handling from `/turns`; accept only `session_id`
- [ ] Update any remaining debug/webchat route params from `:runId` to `:sessionId`
- [ ] Update response payload examples and handler output structs to session-only identifiers

### 5) Geppetto event metadata cleanup
- [ ] Remove `LegacyRunID` compatibility behavior from `geppetto/pkg/events/chat-events.go`
- [ ] Remove legacy `run_id` JSON/log emission tied to session metadata
- [ ] Update serialization tests/contracts for session-only metadata fields

### 6) Frontend/session terminology migration
- [ ] Update frontend state/query keys (`selectedRunId`, `runId`) to session equivalents
- [ ] Update webchat/debug API client params and route builders to `session_id`
- [ ] Update UI labels/tooltips from “run” to “session” where they refer to session identity

### 7) Documentation cleanup (code + ticket docs)
- [ ] Update pinocchio docs/tutorials/READMEs to remove session-meaning `run_id` terminology
- [ ] Update geppetto docs that still describe `run_id` as session alias
- [ ] Update PI-013/PI-014 related design docs that still use debug-UI `run` terminology

### 8) Validation + safety checks
- [ ] Run backend validation (`go build ./...`, relevant tests)
- [ ] Run frontend validation (`npm run typecheck`, `npm run lint` / project standard checks)
- [ ] Run repository grep gates proving no session-meaning `run_id` remains in code/contracts/docs (excluding allowed verb/process cases)

### 9) Ticket bookkeeping + handoff
- [ ] Update PI-017 diary step-by-step with commands, failures, and commit hashes
- [ ] Update PI-017 changelog with commit-by-commit entries
- [ ] Relate all modified implementation files to ticket docs via `docmgr doc relate`
- [ ] Final review summary and close ticket

