# Tasks

## Run → Session Removal Execution Checklist

### 0) Scope lock + baseline
- [x] Confirm migration policy for this ticket: hard remove `run_id` aliases (no compatibility fields) except intentional non-identifier verb uses (`RunLoop`, `go run`, etc.)
- [x] Refresh run-term inventory against current code after PI-018 changes and capture final target files
- [x] Record baseline API/CLI examples that still emit `run_id` (for before/after verification)

### 1) Backend core model rename
- [x] Rename `Conversation.RunID` → `Conversation.SessionID` and update constructors/usages
- [x] Rename run-centric router symbols (e.g. `startRunForPrompt`) to session/inference terminology
- [x] Remove `run_id` fields from `/chat` and queue responses, keep only `session_id`
- [x] Update runtime/log keys to `session_id` (remove legacy `run_id` logging where still tied to session)

### 2) Turn store + SQL migration
- [x] Rename `TurnSnapshot.RunID` → `TurnSnapshot.SessionID`
- [x] Rename `TurnQuery.RunID` → `TurnQuery.SessionID`
- [x] Migrate turn-store SQLite schema from `run_id` column/index names to `session_id`
- [x] Update query/build logic and tests for new session field names

### 3) AgentMode persistence/session references
- [x] Rename AgentMode model fields from `RunID` to `SessionID`
- [x] Migrate AgentMode SQLite schema/indexes from `run_id` to `session_id`
- [x] Update AgentMode service/middleware log/context usage to `session_id`

### 4) API surface cleanup
- [x] Remove `run_id` query parameter handling from `/turns`; accept only `session_id`
- [x] Update any remaining debug/webchat route params from `:runId` to `:sessionId`
- [x] Update response payload examples and handler output structs to session-only identifiers

### 5) Geppetto event metadata cleanup
- [x] Remove `LegacyRunID` compatibility behavior from `geppetto/pkg/events/chat-events.go`
- [x] Remove legacy `run_id` JSON/log emission tied to session metadata
- [x] Update serialization tests/contracts for session-only metadata fields

### 6) Frontend/session terminology migration
- [x] Update frontend state/query keys (`selectedRunId`, `runId`) to session equivalents
- [x] Update webchat/debug API client params and route builders to `session_id`
- [x] Update UI labels/tooltips from “run” to “session” where they refer to session identity

### 7) Documentation cleanup (code + ticket docs)
- [x] Update pinocchio docs/tutorials/READMEs to remove session-meaning `run_id` terminology
- [x] Update geppetto docs that still describe `run_id` as session alias
- [x] Update PI-013/PI-014 related design docs that still use debug-UI `run` terminology

### 8) Validation + safety checks
- [x] Run backend validation (`go build ./...`, relevant tests)
- [x] Run frontend validation (`npm run typecheck`, `npm run lint` / project standard checks)
- [x] Run repository grep gates proving no session-meaning `run_id` remains in code/contracts/docs (excluding allowed verb/process cases)

### 9) Ticket bookkeeping + handoff
- [x] Update PI-017 diary step-by-step with commands, failures, and commit hashes
- [x] Update PI-017 changelog with commit-by-commit entries
- [x] Relate all modified implementation files to ticket docs via `docmgr doc relate`
- [ ] Final review summary and close ticket

