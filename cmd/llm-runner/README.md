# llm-runner

A generic CLI to run LLM interactions using Geppetto turns/blocks.

- Loads a Turn (or Fixture with follow-ups) from YAML
- Runs against OpenAI Responses API (streaming by default)
- Records artifacts: input/final turns, NDJSON events, optional raw provider data
- Generates a Markdown report

## Install / Build

```bash
cd geppetto && go build ./cmd/llm-runner
```

## Usage

```bash
llm-runner run \
  --in ./cmd/llm-runner/fixtures/simple.yaml \
  --out ./cmd/llm-runner/out \
  --cassette ./cmd/llm-runner/out/cassettes/simple \
  --record \
  --model o4-mini \
  --stream \
  --echo-events \
  --raw
```

- Requires `OPENAI_API_KEY` in the environment
- When `--record` is set, HTTP is recorded to the cassette; omit to replay

### Flags
- `--in`: Path to input YAML. Can be either a raw `turns.Turn` or a fixture doc with `turn` and `followups`.
- `--out`: Output directory (default: `out`)
- `--cassette`: VCR cassette base path (without `.yaml`)
- `--record`: Record HTTP instead of replaying
- `--model`: Responses model id (default: `o4-mini`)
- `--stream`: Stream events (default: true)
- `--echo-events`: Echo NDJSON events to stdout while recording
- `--second`, `--second-user`: Append an extra user message follow-up
- `--raw`: Capture raw provider data under `out/raw`

## Fixtures

Fixture document format:

```yaml
version: 1
turn:
  id: turn_simple
  blocks:
    - kind: system
      role: system
      payload: { text: You are a LLM. }
    - kind: user
      role: user
      payload: { text: Say hi. }
followups:
  - kind: user
    role: user
    payload: { text: Hello again }
```

- You can also pass a raw `turns.Turn` YAML without the `turn` wrapper

## Artifacts

- `input_turn.yaml`: The normalized input Turn
- `final_turn.yaml`: Final turn after first run
- `final_turn_N.yaml`: Before and after turns for follow-ups
- `events.ndjson`, `events-2.ndjson`, ...: Event streams per run
- `report.md`: Markdown summary of model, turns, and event timeline
- `raw/` (with `--raw`):
  - `turn-N-http-request.json`, `turn-N-http-response.json`
  - `turn-N-sse.log`
  - `turn-N-provider-000001-output.reasoning.json`, ... (sequenced provider objects)

## Report

Generate a report from an artifacts directory:

```bash
llm-runner report --out ./cmd/llm-runner/out
```

## Notes

- Multi-turn runs currently reproduce a 400 error when reasoning lacks a proper follower; raw capture helps diagnose sequencing.
- Uses Glazed for CLI plumbing and logging. Set `--log-level trace` via Viper config or env to debug.
