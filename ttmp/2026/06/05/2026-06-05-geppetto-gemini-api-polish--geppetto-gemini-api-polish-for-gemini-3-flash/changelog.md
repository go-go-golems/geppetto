# Changelog

## 2026-06-05

- Initial workspace created


## 2026-06-05

Step 1: created Gemini API polish ticket, captured official Gemini/SDK sources, wrote intern guide and smoke plan

### Related Files

- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/analysis/01-smoke-test-plan-and-artifacts.md — Geppetto-first smoke test plan
- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/design-doc/01-gemini-api-polish-intern-guide.md — Intern-facing architecture/design/implementation guide
- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/reference/01-investigation-diary.md — Investigation diary Step 1


## 2026-06-05

Step 2: uploaded Gemini intern guide and added SDK capability probe proving legacy SDK lacks Gemini 3 fields

### Related Files

- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/design-doc/01-gemini-api-polish-intern-guide.md — Uploaded to reMarkable
- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/scripts/01-gemini-sdk-capability-probe.sh — Executable SDK capability probe
- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/scripts/artifacts/sdk-capability-probe.json — Probe output artifact


## 2026-06-05

Step 3: add direct Geppetto Gemini smoke runner and no-key skipped artifact

### Related Files

- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/analysis/01-smoke-test-plan-and-artifacts.md — Updated smoke runner instructions
- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/scripts/03-gemini-geppetto-smoke/main.go — Direct Geppetto smoke runner
- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/scripts/artifacts/plain-text-gemini-2.5-flash-summary.json — Structured no-key skip artifact


## 2026-06-05

Step 4: switch Gemini smoke runner from raw environment key lookup to profile-registry resolution and pass plain/tool/tool-loop direct smokes

### Related Files

- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/analysis/01-smoke-test-plan-and-artifacts.md — Updated profile-backed smoke instructions
- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/scripts/03-gemini-geppetto-smoke/main.go — Profile-backed smoke runner
- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/scripts/artifacts/plain-text-gemini-2.5-flash-summary.json — Plain-text smoke passed
- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/scripts/artifacts/tool-call-gemini-2.5-flash-summary.json — Tool-call smoke passed
- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/scripts/artifacts/tool-loop-gemini-2.5-flash-summary.json — Tool-loop smoke passed


## 2026-06-05

Step 5: add modern Gemini adapter fixture tests for thought signatures, provider tool IDs, thoughts usage, and replay; archive Gemini 3 profile 404 smokes

### Related Files

- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/go.mod — Modern SDK dependency
- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/pkg/steps/ai/gemini/modern_adapter.go — Modern Gemini adapter scaffold
- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/pkg/steps/ai/gemini/modern_adapter_test.go — Fixture tests for Gemini 3 semantics
- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/analysis/01-smoke-test-plan-and-artifacts.md — Updated smoke plan status
- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/scripts/artifacts/plain-text-gemini-3-pro-summary.json — Gemini 3 direct smoke provider/API failure artifact


## 2026-06-05

Step 6: wire live Gemini engine to google.golang.org/genai and pass direct gemini-2.5-flash plus gemini-3-flash-preview text/tool/tool-loop smokes

### Related Files

- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/pkg/steps/ai/gemini/engine_gemini.go — RunInference delegation
- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/pkg/steps/ai/gemini/modern_engine.go — Live modern Gemini SDK engine path
- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/pkg/steps/ai/settings/gemini/settings.go — Gemini thinking/API-version settings
- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/analysis/01-smoke-test-plan-and-artifacts.md — Updated smoke plan
- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/scripts/03-gemini-geppetto-smoke/main.go — Smoke runner model override and thinking flags
- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/scripts/artifacts/tool-loop-gemini-2.5-flash-gemini-3-flash-preview-summary.json — Gemini 3 Flash Preview direct tool-loop artifact


## 2026-06-05

Step 7: add local gemini-3-flash-preview profile, fix llm-proxy sparse profile merge, and pass Gemini-backed OpenAI-compatible proxy smokes

### Related Files

- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/pkg/steps/ai/gemini/modern_adapter.go — Tool-call argument replay fix
- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/analysis/01-smoke-test-plan-and-artifacts.md — Updated smoke status
- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/scripts/04-gemini-llm-proxy-smoke.py — Proxy smoke runner
- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/scripts/artifacts/llm-proxy-gemini-smoke-summary.json — All proxy smoke cases passed
- /home/manuel/workspaces/2026-06-04/llm-proxy/llm-proxy/pkg/profiles/resolver.go — Profile resolver base merge fix


## 2026-06-05

Step 8: prepare Gemini modernization commit and remove unused legacy helpers flagged by pre-commit lint

### Related Files

- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/pkg/steps/ai/gemini/engine_gemini.go — Removed unused legacy flat-part builder
- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/pkg/steps/ai/gemini/stream_helpers.go — Removed unused legacy stream iterator helper


## 2026-06-05

Step 8 follow-up: satisfy full lintmax by moving Gemini thought keys to metadata_keys.go and replacing Claude signature payload literals with a constant

### Related Files

- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/pkg/steps/ai/claude/engine_claude.go — Payload key lint cleanup
- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/pkg/steps/ai/claude/helpers.go — Payload key lint cleanup
- /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto/pkg/steps/ai/gemini/metadata_keys.go — Gemini metadata key definitions

