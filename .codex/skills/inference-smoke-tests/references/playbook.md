# Inference Testing Playbook (geppetto + pinocchio)

This file is intentionally copy/paste-friendly. It is longer than SKILL.md so it lives in `references/`.

## geppetto examples

### OpenAI Responses thinking smoke (fast)

```bash
cd geppetto
go run ./cmd/examples/openai-tools test-openai-tools \
  --ai-api-type openai-responses \
  --ai-engine gpt-5-mini \
  --mode thinking \
  --prompt "What is 2+2?"
```

### Streaming inference (router + sink)

```bash
cd geppetto
go run ./cmd/examples/simple-streaming-inference \
  --pinocchio-profile 4o-mini \
  --prompt "Write one sentence about penguins." \
  --output-format text \
  --verbose \
  --log-level info
```

### Tool loop smoke

```bash
cd geppetto
go run ./cmd/examples/generic-tool-calling generic-tool-calling \
  --pinocchio-profile 4o-mini \
  "What's the weather in Paris and what is 2+2?" \
  --tools-enabled \
  --max-iterations 2 \
  --log-level info
```

### Claude tools smoke (tool calling)

```bash
cd geppetto
go run ./cmd/examples/claude-tools test-claude-tools \
  --ai-api-type claude \
  --ai-engine claude-haiku-4-5
```

## pinocchio examples

### Agent TUI smoke (tmux-driven)

Submit key: `Tab`  
Quit key: `Alt-q` (send as `M-q` in tmux)

```bash
cd pinocchio
rm -f /tmp/simple-chat-agent.log
tmux kill-session -t agent-smoke 2>/dev/null || true
tmux new-session -d -s agent-smoke \
  "go run ./cmd/agents/simple-chat-agent simple-chat-agent \
    --ai-api-type openai-responses \
    --ai-engine gpt-5-mini \
    --ai-max-response-tokens 256 \
    --openai-reasoning-summary auto \
    --log-level debug \
    --log-file /tmp/simple-chat-agent.log \
    --with-caller"

sleep 2
tmux send-keys -t agent-smoke 'hello' Tab
sleep 14
tmux send-keys -t agent-smoke M-q
sleep 1
tmux kill-session -t agent-smoke 2>/dev/null || true

tail -n 250 /tmp/simple-chat-agent.log
```

### Pinocchio TUI chat (manual)

This is a real-world multi-turn test (best done manually even if launched in tmux):

```bash
cd pinocchio
go run ./cmd/pinocchio code professional "hello" \
  --ai-api-type openai-responses \
  --ai-engine gpt-5-mini \
  --chat \
  --log-level debug --with-caller
```

Send 2+ messages and confirm:
- the second message completes (no stuck “generating”)
- no Responses validation errors like “reasoning item must be followed”

### Pinocchio webchat (manual)

```bash
cd pinocchio
go run ./cmd/web-chat --log-level debug --with-caller
```

Open the browser UI, send 2+ messages in one conversation, confirm no 400s and that tool events still show if enabled.

### Redis streaming inference (transport test)

Requires Redis.

```bash
cd pinocchio
go run ./cmd/examples/simple-redis-streaming-inference \
  --prompt "Stream this response via Redis." \
  --redis-enabled \
  --redis-addr localhost:6379 \
  --redis-group chat-ui \
  --redis-consumer ui-1 \
  --verbose \
  --log-level debug
```
