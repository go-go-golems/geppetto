---
status: active
intent: long-term
---

# Implementation diary

## 2026-02-03

### Step 1: Ticket setup + initial investigation

Created ticket docs (bug report, analysis, diary) and inspected webchat/session code to locate the seed-turn failure path.

#### What I did
- Created the ticket `PI-007-WEBCHAT-BACKEND-REFACTOR` and added docs.
- Read `pkg/webchat/router.go`, `pkg/webchat/conversation.go`, and `geppetto/pkg/inference/session/session.go`.
- Confirmed `Session.StartInference` fails when the latest turn has 0 blocks.
- Confirmed `AppendNewTurnFromUserPrompt` does not add blocks for empty prompts.

#### What I learned
- Webchat seeds the session with an empty turn and relies on user prompt to add blocks, which fails when prompt is empty.
- System prompt middleware is applied after StartInference validation, so it cannot fix an empty seed.

#### Next
- Seed the session with a system prompt block based on the profile configuration.
- Ensure the default profile uses system prompt text `"You are an assistant"`.

### Step 2: Implement seed turn + system prompt defaults

Implemented seed turn construction using the profile system prompt and added tests to prevent empty-seed regressions.

#### What I did
- Added `buildSeedTurn` and used it when creating `Session.Turns` for a new webchat conversation.
- Defaulted system prompts to "You are an assistant" when profiles provide none.
- Updated the default webchat profile prompt to "You are an assistant".
- Added tests to ensure empty prompts keep a system block in the turn.
- Ran `go test ./...` in `pinocchio`.

#### What worked
- Tests passed and the webchat package verified seed turn behavior.

#### Files touched
- `pinocchio/pkg/webchat/conversation.go`
- `pinocchio/pkg/webchat/engine_builder.go`
- `pinocchio/pkg/webchat/conversation_test.go`
- `pinocchio/cmd/web-chat/main.go`

#### Validation
- Command: `go test ./...`

### Step 3: Commit + hook validation

Committed the backend seed-turn fix and let the repo hooks run their full validation pipeline.

#### What I did
- Committed changes with message `webchat: seed session with system prompt`.
- Pre-commit hooks ran `go test ./...` and the lint pipeline (`go generate`, Vite build, golangci-lint).

#### What worked
- Commit succeeded and hooks passed.

#### Notes
- Hooks emitted npm deprecation/audit warnings during `go generate`.

#### Technical details
- Commit: `5e816d5`

### Step 4: Fix prompt payload mismatch

Investigated `prompt_len=0` and fixed the request payload mismatch between the frontend and backend.

#### What I did
- Updated the frontend chat payload to send `prompt` instead of `text`.
- Added a backend compatibility alias so `text` still maps to `prompt` if present.
- Committed the fix and let hooks run (Go tests + web checks + lint).

#### Files touched
- `pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx`
- `pinocchio/pkg/webchat/engine_from_req.go`

#### Validation
- Pre-commit hooks ran `go test ./...`, `npm run check`, and lint pipeline.

### Step 5: WS connection analysis

Reviewed the two WebSocket URLs and confirmed the token-based connection is Vite HMR, not the chat WS.

#### Notes
- The connection to `ws://localhost:5173/?token=...` includes `Sec-WebSocket-Protocol: vite-hmr`, indicating Vite’s dev server HMR socket.
- The chat socket uses `/ws?conv_id=...` and is proxied by Vite to the Go backend.

### Step 6: Disable remote Redux devtools

Disabled the remote Redux devtools enhancer to avoid connecting to the socketcluster endpoint.

#### What I did
- Removed the remote devtools enhancer/config and forced `devTools: false`.
- Committed the change.

#### Files touched
- `pinocchio/cmd/web-chat/web/src/store/store.ts`

#### Technical details
- Commit: `b53c262`
