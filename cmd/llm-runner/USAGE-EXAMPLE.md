# LLM Runner Usage Example

This guide demonstrates how to use llm-runner to run inference, capture artifacts including logs, and visualize them with the web UI.

## Prerequisites

1. Set your OpenAI API key:
   ```bash
   export OPENAI_API_KEY="your-api-key-here"
   ```

2. Build llm-runner:
   ```bash
   cd geppetto
   go build ./cmd/llm-runner
   ```

## Running Inference with Full Capture

Run inference with all capture options enabled:

```bash
./llm-runner run \
  --in ./cmd/llm-runner/fixtures/simple.yaml \
  --out ./cmd/llm-runner/out/demo \
  --cassette ./cmd/llm-runner/out/demo/cassettes/simple \
  --record \
  --model o4-mini \
  --stream \
  --raw \
  --capture-logs \
  --echo-events
```

This will create the following artifacts in `./cmd/llm-runner/out/demo/`:
- `input_turn.yaml` - The normalized input turn
- `final_turn.yaml` - The final turn after inference
- `events.ndjson` - Stream of inference events
- `logs.jsonl` - Application logs in JSON Lines format
- `raw/` - Raw HTTP requests, responses, SSE logs, and provider objects
- `cassettes/` - VCR cassette for replay

## Generating a Report

Generate a markdown report from the artifacts:

```bash
./llm-runner report --out ./cmd/llm-runner/out/demo
```

This creates `report.md` with a summary of the run.

## Visualizing with Web UI

Launch the web interface to browse all artifacts:

```bash
./llm-runner serve --out ./cmd/llm-runner/out --port 8080
```

Then open http://localhost:8080 in your browser.

### Web UI Features

1. **Directory Navigation**: Browse different artifact directories (e.g., demo, test runs)
2. **File Browser**: View all files in a selected directory
3. **Syntax Highlighting**: Automatic syntax highlighting for YAML, JSON, NDJSON, and logs
4. **Interactive Navigation**: Click to view file contents with HTMX for smooth updates
5. **Log Inspection**: View captured zerolog output in JSON Lines format

### Debugging with Logs

The `logs.jsonl` file contains structured logs that help debug:
- API calls and responses
- Internal state transitions
- Error messages with context
- Timing information

Example log entry:
```json
{"level":"info","time":"2025-10-21T10:30:45Z","message":"Starting inference","turn_id":"turn_simple","model":"o4-mini"}
```

## Replaying Cassettes

To replay a previously recorded session without hitting the API:

```bash
./llm-runner run \
  --in ./cmd/llm-runner/fixtures/simple.yaml \
  --out ./cmd/llm-runner/out/replay \
  --cassette ./cmd/llm-runner/out/demo/cassettes/simple \
  --capture-logs
```

Note: `--record` is omitted, so VCR will replay from the cassette.

## Multi-turn Conversations

Run follow-up turns to test conversation flow:

```bash
./llm-runner run \
  --in ./cmd/llm-runner/fixtures/simple.yaml \
  --out ./cmd/llm-runner/out/multi \
  --second \
  --second-user "Tell me more" \
  --raw \
  --capture-logs \
  --record
```

This creates additional artifacts:
- `final_turn_1.yaml` - Turn after adding follow-up
- `events-2.ndjson` - Events for second inference
- `final_turn_2.yaml` - Final turn after second inference

## Tips

1. **Debugging Issues**: Use `--raw` and `--capture-logs` together for full observability
2. **Compare Runs**: Use different `--out` directories to compare multiple runs
3. **Development**: Use `--log-level trace` for verbose logging
4. **Performance**: Disable `--capture-logs` for production runs if logs aren't needed
5. **Web UI**: Keep the server running while doing multiple runs to see updates

## Example Workflow

```bash
# 1. Run initial inference with full capture
./llm-runner run --in fixtures/simple.yaml --out out/run1 --record --raw --capture-logs

# 2. Review in web UI
./llm-runner serve --out out --port 8080 &

# 3. Make changes and run again
./llm-runner run --in fixtures/simple.yaml --out out/run2 --record --raw --capture-logs

# 4. Compare runs in the web UI
# Navigate between run1 and run2 directories to see differences

# 5. Generate reports
./llm-runner report --out out/run1
./llm-runner report --out out/run2

# 6. Stop the server
killall llm-runner
```

