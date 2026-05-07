# Changelog

## 2026-05-07

- Initial workspace created


## 2026-05-07

Created GP-CODE-REVIEW ticket, inventoried Geppetto package layout and recent observability work, wrote the intern-facing code review/onboarding guide, and related key Geppetto/Pinocchio/prior-diary files.

### Related Files

- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/pkg/observability/json.go — Evidence sanitizer issue cited in review
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/pkg/steps/ai/openai_responses/engine.go — Large stream engine refactor target cited in review
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-CODE-REVIEW--code-review-and-cleanup-guide-for-geppetto-observability-and-recent-runtime-integration/design-doc/01-geppetto-code-review-and-intern-onboarding-guide.md — Primary code review and intern onboarding deliverable
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-CODE-REVIEW--code-review-and-cleanup-guide-for-geppetto-observability-and-recent-runtime-integration/reference/01-diary.md — Investigation diary with commands and findings


## 2026-05-07

Validated GP-CODE-REVIEW with docmgr doctor; added vocabulary topics code-review, intern-onboarding, and observability, then doctor passed cleanly.

### Related Files

- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/vocabulary.yaml — Added ticket topics required for doctor validation


## 2026-05-07

Uploaded the GP-CODE-REVIEW guide/diary bundle to reMarkable at /ai/2026/05/07/GP-CODE-REVIEW/GP-CODE-REVIEW Geppetto Code Review and Intern Onboarding Guide.pdf and verified it with remarquee cloud ls.

### Related Files

- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-CODE-REVIEW--code-review-and-cleanup-guide-for-geppetto-observability-and-recent-runtime-integration/design-doc/01-geppetto-code-review-and-intern-onboarding-guide.md — Uploaded as part of reMarkable bundle
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-CODE-REVIEW--code-review-and-cleanup-guide-for-geppetto-observability-and-recent-runtime-integration/reference/01-diary.md — Uploaded as part of reMarkable bundle


## 2026-05-07

Uploaded an updated Final reMarkable bundle including the latest diary step: /ai/2026/05/07/GP-CODE-REVIEW/GP-CODE-REVIEW Geppetto Code Review and Intern Onboarding Guide Final.pdf.

### Related Files

- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-CODE-REVIEW--code-review-and-cleanup-guide-for-geppetto-observability-and-recent-runtime-integration/reference/01-diary.md — Updated with validation/upload step before final bundle upload


## 2026-05-07

Split OpenAI Responses engine into focused files across commits 5f28ea1, a79abcb, and 0e49aed. engine.go is now high-level orchestration; streaming, non-streaming, tool attachment, usage parsing, and stream event helpers live in separate files. Focused tests and full pre-commit test/lint passed for each code commit.

### Related Files

- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/pkg/steps/ai/openai_responses/engine.go — Reduced OpenAI Responses engine orchestration file
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/pkg/steps/ai/openai_responses/nonstreaming.go — Extracted non-streaming implementation
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/pkg/steps/ai/openai_responses/streaming.go — Extracted streaming implementation
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-CODE-REVIEW--code-review-and-cleanup-guide-for-geppetto-observability-and-recent-runtime-integration/reference/01-diary.md — Recorded split work


## 2026-05-07

Removed the custom observability evidence JSON transform layer in commit 1e55df3: deleted observability/json.go, simplified Config to trace level only, removed payload/redaction flags, and changed OpenAI Responses records to plain JSON for object/event/metadata evidence. Updated the guide and diary to record the simplification decision.

### Related Files

- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/pkg/cli/bootstrap/inference_observability.go — Removed payload/redaction CLI flags
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/pkg/observability/config.go — Config simplified to trace level only
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/pkg/observability/json.go — Deleted custom evidence JSON transform helper
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/pkg/steps/ai/openai_responses/observability.go — Plain JSON evidence capture
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-CODE-REVIEW--code-review-and-cleanup-guide-for-geppetto-observability-and-recent-runtime-integration/reference/01-diary.md — Recorded Step 5


## 2026-05-07

Split pkg/events/chat-events.go into domain-focused files in commit ce1149f: core contract/metadata/decoder stayed in chat-events.go; text, tool, log/info, and built-in provider events moved to separate files. Focused tests and full pre-commit test/lint passed.

### Related Files

- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/pkg/events/builtin_events.go — Built-in provider event definitions after split
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/pkg/events/chat-events.go — Core event contract/metadata/decoder after split
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/pkg/events/log_info_events.go — Log/info event definitions after split
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/pkg/events/text_events.go — Text event definitions after split
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/pkg/events/tool_events.go — Tool event definitions after split
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-CODE-REVIEW--code-review-and-cleanup-guide-for-geppetto-observability-and-recent-runtime-integration/reference/01-diary.md — Recorded Step 6 event definition split

