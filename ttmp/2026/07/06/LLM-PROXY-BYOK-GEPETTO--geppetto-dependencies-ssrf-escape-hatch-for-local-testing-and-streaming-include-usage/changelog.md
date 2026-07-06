# Changelog

## 2026-07-06

- Initial workspace created


## 2026-07-06

Step 2: implemented profile-owned outbound URL validation opt-ins for OpenAI Chat, OpenAI Responses, and Claude; defaults remain fail-closed; full go test ./... and pre-commit lint passed (commit ece5bb07).

### Related Files

- /home/manuel/workspaces/2026-07-05/llm-proxy-byok/geppetto/pkg/steps/ai/claude/api/completion.go — Claude API client carries outbound URL options
- /home/manuel/workspaces/2026-07-05/llm-proxy-byok/geppetto/pkg/steps/ai/openai/chat_stream.go — OpenAI-compatible Chat validates provider URL with profile-owned outbound options
- /home/manuel/workspaces/2026-07-05/llm-proxy-byok/geppetto/pkg/steps/ai/openai_responses/provider_settings.go — Responses alias-aware outbound option resolution
- /home/manuel/workspaces/2026-07-05/llm-proxy-byok/geppetto/pkg/steps/ai/settings/outbound_url.go — New helper mapping profile API settings to security.OutboundURLOptions
- /home/manuel/workspaces/2026-07-05/llm-proxy-byok/geppetto/pkg/steps/ai/settings/settings-inference.go — APISettings now exposes allow_http and allow_local_networks maps

