# Changelog

## 2026-01-23

- Initial workspace created


## 2026-01-23

Step 1: Created ticket + diary doc

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/23/GP-09-PROFILE-ENGINE-BUILDER--profile-engine-builder/reference/01-diary.md — Established working diary and confirmed intent


## 2026-01-23

Step 2: Audited go-go-mento EngineBuilder/Router coupling and compared Pinocchio/Geppetto patterns

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/23/GP-09-PROFILE-ENGINE-BUILDER--profile-engine-builder/reference/01-diary.md — Recorded investigation findings and key issues (signature secrecy, tools override gap, unused profile fields)


## 2026-01-23

Step 3: Collected key docs and prior ticket references for EngineBuilder extraction

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/23/GP-09-PROFILE-ENGINE-BUILDER--profile-engine-builder/reference/01-diary.md — Added doc survey and highlighted PI-001 status update + doc drift


## 2026-01-23

Step 4: Wrote GP-09 design analysis (BuildEngineFromReq + ProfileEngineBuilder)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/23/GP-09-PROFILE-ENGINE-BUILDER--profile-engine-builder/reference/01-diary.md — Recorded step and review pointers
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/23/GP-09-PROFILE-ENGINE-BUILDER--profile-engine-builder/analysis/01-extract-profile-engine-builder-out-of-router.md — Main extraction proposal and incremental plan


## 2026-01-23

Step 5: Pivoted scope to Pinocchio + Moments (rolled back go-go-mento edits)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/23/GP-09-PROFILE-ENGINE-BUILDER--profile-engine-builder/reference/01-diary.md — Recorded scope change, rollback details, and test failures encountered during the abandoned go-go-mento attempt


## 2026-01-23

Step 6: Implemented Phase 1 + tools/config drive in Pinocchio (commit 3b8cae7)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/engine_from_req.go — Request-facing builder (BuildEngineFromReq-style)
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/engine_builder.go — Config now includes tools + override enforcement
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/router.go — Router delegates request policy + filters tool registry by config
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/engine_from_req_test.go — Request policy precedence tests
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/23/GP-09-PROFILE-ENGINE-BUILDER--profile-engine-builder/reference/01-diary.md — Step notes and review instructions


## 2026-01-23

Step 7: Implemented Phase 1 request policy extraction in Moments (commit fe3e9dcf)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/moments/backend/pkg/webchat/engine_from_req.go — Request-facing builder (BuildEngineFromReq-style)
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/moments/backend/pkg/webchat/router.go — Router delegates request policy for HTTP + WS
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/moments/backend/pkg/webchat/engine_from_req_test.go — Request policy precedence tests
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/23/GP-09-PROFILE-ENGINE-BUILDER--profile-engine-builder/reference/01-diary.md — Step notes and review instructions
