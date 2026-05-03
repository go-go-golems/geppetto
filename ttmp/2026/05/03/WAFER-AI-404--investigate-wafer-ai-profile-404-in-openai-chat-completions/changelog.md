# Changelog

## 2026-05-03

- Initial workspace created.
- Added redacted settings, live 404, and curl endpoint probe evidence for the Wafer AI profile failure.
- Wrote the Wafer AI OpenAI-compatible 404 analysis and implementation guide.
- Wrote the chronological investigation diary.
- Updated tasks with completed investigation steps and follow-up implementation items.

## 2026-05-03

Completed Wafer AI 404 investigation: profile stores full chat-completions endpoint while Geppetto appends /chat/completions, causing double-path 404; wrote guide and diary.

### Related Files

- /home/manuel/workspaces/2026-05-03/fix-404-wafer-ai/geppetto/ttmp/2026/05/03/WAFER-AI-404--investigate-wafer-ai-profile-404-in-openai-chat-completions/design-doc/01-wafer-ai-openai-compatible-404-analysis-and-implementation-guide.md — Primary analysis and implementation guide
- /home/manuel/workspaces/2026-05-03/fix-404-wafer-ai/geppetto/ttmp/2026/05/03/WAFER-AI-404--investigate-wafer-ai-profile-404-in-openai-chat-completions/reference/01-investigation-diary.md — Chronological investigation diary


## 2026-05-03

Validated ticket with docmgr doctor and uploaded the analysis bundle to reMarkable at /ai/2026/05/03/WAFER-AI-404.

### Related Files

- /home/manuel/workspaces/2026-05-03/fix-404-wafer-ai/geppetto/ttmp/2026/05/03/WAFER-AI-404--investigate-wafer-ai-profile-404-in-openai-chat-completions/reference/01-investigation-diary.md — Added validation and reMarkable delivery step


## 2026-05-03

Implemented the follow-up: added OpenAI-compatible 404 base-URL diagnostics, tested them, backed up and refactored local Wafer profiles to use wafer-base stack, and confirmed live Wafer streaming succeeds.

### Related Files

- /home/manuel/.config/pinocchio/profiles.yaml — Local profile stack refactor
- /home/manuel/workspaces/2026-05-03/fix-404-wafer-ai/geppetto/pkg/steps/ai/openai/chat_stream.go — 404 hint implementation
- /home/manuel/workspaces/2026-05-03/fix-404-wafer-ai/geppetto/pkg/steps/ai/openai/chat_stream_test.go — 404 hint regression tests


## 2026-05-03

Uploaded updated analysis bundle to reMarkable as 'WAFER-AI-404 Wafer AI 404 analysis updated'.

### Related Files

- /home/manuel/workspaces/2026-05-03/fix-404-wafer-ai/geppetto/ttmp/2026/05/03/WAFER-AI-404--investigate-wafer-ai-profile-404-in-openai-chat-completions/design-doc/01-wafer-ai-openai-compatible-404-analysis-and-implementation-guide.md — Updated bundle content


## 2026-05-03

Added Defuddle-extracted DeepSeek thinking mode documentation and Kagi search provenance to sources for thinking-level analysis.

### Related Files

- /home/manuel/workspaces/2026-05-03/fix-404-wafer-ai/geppetto/ttmp/2026/05/03/WAFER-AI-404--investigate-wafer-ai-profile-404-in-openai-chat-completions/sources/04-deepseek-thinking-mode-defuddle.md — DeepSeek thinking mode source
- /home/manuel/workspaces/2026-05-03/fix-404-wafer-ai/geppetto/ttmp/2026/05/03/WAFER-AI-404--investigate-wafer-ai-profile-404-in-openai-chat-completions/sources/05-kagi-deepseek-v4-thinking-search.md — Search provenance


## 2026-05-03

Refactored all local Pinocchio profiles to provider base profiles, backed up profiles.yaml, validated every leaf profile, and added a redacted audit source.

### Related Files

- /home/manuel/.config/pinocchio/profiles.yaml — Full provider base profile refactor
- /home/manuel/workspaces/2026-05-03/fix-404-wafer-ai/geppetto/ttmp/2026/05/03/WAFER-AI-404--investigate-wafer-ai-profile-404-in-openai-chat-completions/sources/06-profiles-provider-base-refactor-redacted.md — Redacted profile refactor summary

