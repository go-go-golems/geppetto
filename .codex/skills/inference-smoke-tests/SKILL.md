---
name: inference-smoke-tests
description: "Run repeatable inference smoke tests using geppetto/pinocchio example binaries (single-pass, streaming, tool-loop, OpenAI Responses thinking) including tmux-driven TUI tests. Use when refactors touch InferenceState/Session/EngineBuilder, tool calling loop, event sinks, provider request formatting, or when you need a quick 'does inference still work?' checklist."
---

# Inference Smoke Tests

## Quick Start (Recommended)

Run the fast suite (geppetto non-TUI + pinocchio agent TUI) via the bundled script:

```bash
bash geppetto/.codex/skills/inference-smoke-tests/scripts/run_smoke.sh --quick
```

If you need the full manual checklist, open:

`geppetto/.codex/skills/inference-smoke-tests/references/playbook.md`

## Preconditions

- Ensure `OPENAI_API_KEY` is set (for OpenAI Chat + OpenAI Responses).
- Ensure Claude credentials are available (e.g. `ANTHROPIC_API_KEY`) if you want the Claude tool-calling smoke step to pass.
- Ensure `tmux` is installed (required for non-interactive TUI runs).
- Expect costs: these tests make real API calls.

## Workflow Decision Tree

1) Validate provider “thinking” streaming (Responses)?
- Run `geppetto/cmd/examples/openai-tools` in `--mode thinking`.

2) Validate tool loop orchestration?
- Run `geppetto/cmd/examples/generic-tool-calling`.

3) Validate Bubble Tea TUI event flow (thinking deltas + final)?
- Run `pinocchio/cmd/agents/simple-chat-agent` in tmux.

4) Validate Claude tool calling?
- Run `geppetto/cmd/examples/claude-tools` with `--ai-api-type claude --ai-engine claude-haiku-4-5`.

5) Validate multi-turn chat state persistence?
- Run pinocchio TUI chat in tmux (manual) and/or pinocchio webchat in browser (manual).

## What “Benefits From InferenceState” (Rules of Thumb)

Already benefits (multi-turn, cancel-sensitive, tool-loop, strict provider validation):
- pinocchio TUI chat (`pinocchio/cmd/pinocchio … --chat`)
- pinocchio agent TUI (`pinocchio/cmd/agents/simple-chat-agent …`)
- pinocchio webchat (`pinocchio/cmd/web-chat`)
- geppetto example runners that execute via `geppetto/pkg/inference/core.Session`

Could benefit (optional; mainly consistency/cancel):
- `pinocchio/cmd/examples/simple-redis-streaming-inference` (transport-focused; currently `eng.RunInference` direct)
- `pinocchio/cmd/examples/simple-chat` (exercises PinocchioCommand runner; could benefit indirectly if that runner standardizes on `InferenceState`)

Does not apply (not an inference runner):
- `geppetto/cmd/examples/citations-event-stream`

## Troubleshooting (Common Failure Modes)

### “OpenAI Responses 400” errors
- Re-run with higher logging:
  - Add `--log-level debug --with-caller` where supported.
- Confirm you’re using the correct provider mode:
  - `--ai-api-type openai-responses`
- If the error mentions invalid parameter support (e.g., `temperature` unsupported), it’s model-dependent; reduce parameters and retry.

### TUI doesn’t submit the prompt
- Some TUIs submit on `Tab` (not `Enter`).
- Always capture logs to a file and confirm inference actually ran (look for `EventPartialCompletionStart`, `EventFinal`).

## References

When you need copy/paste commands for the full sweep, read:
- `geppetto/.codex/skills/inference-smoke-tests/references/playbook.md`

When you need to find new example entry points, search:

```bash
rg -n "cmd/examples" -S geppetto/cmd/examples pinocchio/cmd/examples
rg -n "cmd/agents" -S pinocchio/cmd/agents
```
