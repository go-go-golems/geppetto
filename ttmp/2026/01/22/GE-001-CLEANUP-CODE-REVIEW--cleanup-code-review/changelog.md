# Changelog

## 2026-01-22

- Initial workspace created


## 2026-01-22

Remove ParsedLayersEngineBuilder indirection: examples/agent now construct engines directly via engine/factory.NewEngineFromParsedLayers; deleted unused builder packages.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/cmd/examples/claude-tools/main.go — Construct engine directly (remove builder)
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/cmd/examples/generic-tool-calling/main.go — Construct engine directly (remove builder)
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/cmd/examples/middleware-inference/main.go — Construct engine directly (remove builder)
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/cmd/examples/openai-tools/main.go — Construct engine directly; keep sink wiring
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/cmd/examples/simple-inference/main.go — Construct engine directly (remove builder)
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/cmd/examples/simple-streaming-inference/main.go — Construct engine directly (remove builder)
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/agents/simple-chat-agent/main.go — Construct engine directly (remove builder package)

