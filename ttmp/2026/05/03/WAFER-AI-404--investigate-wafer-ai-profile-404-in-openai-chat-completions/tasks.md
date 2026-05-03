# Tasks

## Completed

- [x] Create docmgr ticket workspace for Wafer AI 404 investigation.
- [x] Inspect local Wafer profiles with API keys redacted.
- [x] Trace Geppetto OpenAI chat endpoint construction in source code.
- [x] Capture `--print-inference-settings` evidence for `wafer-deepseek-v4-pro`.
- [x] Reproduce live Pinocchio 404 with current profile.
- [x] Probe Wafer endpoint behavior directly with redacted curl evidence.
- [x] Write analysis and implementation guide.
- [x] Write chronological investigation diary.

## Follow-up implementation tasks

- [ ] Back up and edit `~/.config/pinocchio/profiles.yaml` so Wafer `openai-base-url` values use `https://pass.wafer.ai/v1`.
- [ ] Add debug logging for the computed OpenAI chat completions endpoint.
- [ ] Add warning/tests for `openai-base-url` values ending in `/chat/completions`.
- [ ] Decide whether CLI base URL flags should override profile values.
