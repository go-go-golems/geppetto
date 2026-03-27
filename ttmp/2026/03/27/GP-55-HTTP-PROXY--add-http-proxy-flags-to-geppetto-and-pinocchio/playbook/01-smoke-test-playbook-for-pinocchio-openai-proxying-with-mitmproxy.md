---
Title: Smoke test playbook for Pinocchio OpenAI proxying with mitmproxy
Ticket: GP-55-HTTP-PROXY
Status: active
Topics:
    - geppetto
    - pinocchio
    - glazed
    - config
    - inference
    - documentation
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/README.md
    - Path: pinocchio/examples/js/README.md
    - Path: pinocchio/examples/js/profiles/basic.yaml
    - Path: pinocchio/cmd/web-chat/main.go
    - Path: geppetto/pkg/steps/ai/settings/flags/client.yaml
    - Path: geppetto/pkg/steps/ai/settings/http_client.go
ExternalSources: []
Summary: Verified smoke-test procedure for routing Pinocchio example and web-chat traffic through mitmproxy and confirming gpt-5-mini OpenAI requests.
LastUpdated: 2026-03-27T11:05:00-04:00
WhatFor: Show a repeatable, low-ceremony way to prove that Pinocchio and web-chat send OpenAI traffic through the new ai-client proxy settings and that the selected profile resolves to gpt-5-mini on the wire.
WhenToUse: Use when validating proxy support after a code change, debugging customer reports about proxy routing, or onboarding someone who needs a concrete end-to-end verification procedure rather than an architectural explanation.
---

# Smoke test playbook for Pinocchio OpenAI proxying with mitmproxy

## Purpose

This playbook proves three concrete things:

- `pinocchio examples test` accepts the shared `ai-client` proxy flags and sends its OpenAI request through `mitmproxy`.
- The example profile registry at `pinocchio/examples/js/profiles/basic.yaml` resolves `assistant` to `gpt-5-mini`, and that model shows up in the intercepted request body.
- `web-chat` now exposes the same `ai-client` flags, but to hit the `assistant` profile you must post to `/chat/assistant`; posting to `/chat` uses the default runtime and will not prove the `gpt-5-mini` profile path.

The procedure below was exercised in this ticket workspace on 2026-03-27 with `tmux`, `uvx`, `mitmdump`, `pinocchio examples test`, and `web-chat web-chat`.

## Environment Assumptions

- Workspace root is `/home/manuel/workspaces/2026-03-27/add-geppetto-proxy`.
- `tmux` is installed.
- `uvx` is available so `mitmdump` can be launched without a persistent manual install.
- One valid OpenAI credential source is already configured for Pinocchio or Geppetto.
  - The playbook intentionally does not print or persist credentials.
  - Do not share raw `mitmproxy` flow files because they contain `Authorization` headers.
- The example registry file exists at `pinocchio/examples/js/profiles/basic.yaml`.
- Go trusts the `mitmproxy` certificate authority through `SSL_CERT_FILE`; without this, HTTPS interception will fail with an x509 trust error.

The relevant example profile chain is:

```yaml
slug: workspace
profiles:
  default:
    inference_settings:
      chat:
        api_type: openai
        engine: gpt-4o-mini
  assistant:
    stack:
      - profile_slug: default
    inference_settings:
      chat:
        engine: gpt-5-mini
```

That means `assistant` is the profile we use to prove `gpt-5-mini` on the wire.

## Commands

### 1. Confirm the CLI surfaces

```bash
cd /home/manuel/workspaces/2026-03-27/add-geppetto-proxy/pinocchio

go run ./cmd/pinocchio examples test --help --long-help | sed -n '95,125p'
go run ./cmd/web-chat web-chat --help --long-help | sed -n '40,70p'
```

You should see the shared `AI client flags` section, including:

- `--proxy-url`
- `--proxy-from-environment`
- `--timeout`
- `--organization`
- `--user-agent`

### 2. Create temporary capture paths

```bash
CONF_DIR="$(mktemp -d /tmp/gp55-mitm-conf.XXXXXX)"
FLOW_FILE="$(mktemp /tmp/gp55-openai-flow.XXXXXX)"

printf 'CONF_DIR=%s\nFLOW_FILE=%s\n' "$CONF_DIR" "$FLOW_FILE"
```

`CONF_DIR` will hold the generated mitm CA. `FLOW_FILE` will hold the raw intercepted flow data. Treat both as sensitive until deleted.

### 3. Start `mitmdump` inside `tmux`

```bash
cd /home/manuel/workspaces/2026-03-27/add-geppetto-proxy

tmux new-session -d -s gp55-mitm \
  "uvx --from mitmproxy mitmdump --set confdir=$CONF_DIR -p 8082 -w $FLOW_FILE"

sleep 2
tmux capture-pane -pt gp55-mitm:0.0
ls -1 "$CONF_DIR"
```

Expected proxy-pane output:

```text
[HH:MM:SS.mmm] HTTP(S) proxy listening at *:8082.
```

Expected certificate file:

```text
$CONF_DIR/mitmproxy-ca-cert.pem
```

### 4. Run the simple Pinocchio smoke test through the proxy

```bash
cd /home/manuel/workspaces/2026-03-27/add-geppetto-proxy/pinocchio

SSL_CERT_FILE="$CONF_DIR/mitmproxy-ca-cert.pem" \
go run ./cmd/pinocchio examples test \
  --profile-registries examples/js/profiles/basic.yaml \
  --profile assistant \
  --proxy-url http://127.0.0.1:8082 \
  --non-interactive \
  --what test \
  --pretend tester
```

Expected terminal result:

```text
Turing test
```

Expected proxy-pane evidence:

```bash
tmux capture-pane -pt gp55-mitm:0.0
```

The pane should contain a line like:

```text
POST https://api.openai.com/v1/chat/completions HTTP/2.0
```

### 5. Replay the flow safely without printing headers

Create a tiny replay addon that only prints non-sensitive request facts:

```bash
cat > /tmp/gp55-sanitize-addon.py <<'PY'
import json
from mitmproxy import http

class SafePrinter:
    def response(self, flow: http.HTTPFlow) -> None:
        if flow.request.host != "api.openai.com":
            return
        if flow.request.path != "/v1/chat/completions":
            return
        try:
            body = json.loads(flow.request.get_text())
        except Exception:
            body = {}
        messages = body.get("messages", [])
        roles = [m.get("role") for m in messages if isinstance(m, dict)]
        print(f"URL: {flow.request.pretty_url}")
        print(f"Model: {body.get('model')}")
        print(f"MessageRoles: {roles}")
        print(f"Stream: {body.get('stream')}")

addons = [SafePrinter()]
PY

uvx --from mitmproxy mitmdump -nr "$FLOW_FILE" -q -s /tmp/gp55-sanitize-addon.py
```

Expected safe replay output:

```text
URL: https://api.openai.com/v1/chat/completions
Model: gpt-5-mini
MessageRoles: ['system', 'user']
Stream: True
```

That is the cleanest proof that the `assistant` profile selected `gpt-5-mini` and that the request actually traversed the proxy.

### 6. Optional `web-chat` smoke path

Start the server with the same proxy settings:

```bash
cd /home/manuel/workspaces/2026-03-27/add-geppetto-proxy/pinocchio

tmux new-session -d -s gp55-webchat \
  "SSL_CERT_FILE=$CONF_DIR/mitmproxy-ca-cert.pem \
   go run ./cmd/web-chat web-chat \
     --addr 127.0.0.1:8090 \
     --profile-registries examples/js/profiles/basic.yaml \
     --profile assistant \
     --proxy-url http://127.0.0.1:8082"

sleep 3
tmux capture-pane -pt gp55-webchat:0.0
```

Expected server startup log:

```text
starting web-chat server addr=127.0.0.1:8090
```

First, demonstrate the default route behavior:

```bash
curl -sS -N \
  -H 'Content-Type: application/json' \
  -d '{"prompt":"What is 2+2? Answer in one word.","conv_id":"conv-smoke"}' \
  http://127.0.0.1:8090/chat
```

This returns a JSON object whose `runtime_fingerprint` shows `ai-engine":"gpt-4o-mini"`. That is expected. `/chat` uses the default runtime, not the `assistant` runtime.

Then hit the profile-specific route:

```bash
curl -sS -N \
  -H 'Content-Type: application/json' \
  -d '{"prompt":"What is 2+2? Answer in one word.","conv_id":"conv-assistant"}' \
  http://127.0.0.1:8090/chat/assistant
```

Expected JSON evidence from the immediate HTTP response:

- `profile_metadata.profile.slug` is `assistant`
- `runtime_fingerprint` contains `ai-engine":"gpt-5-mini"`
- `runtime_fingerprint` contains `proxy-url":"http://127.0.0.1:8082"`

Re-run the safe replay command:

```bash
uvx --from mitmproxy mitmdump -nr "$FLOW_FILE" -q -s /tmp/gp55-sanitize-addon.py
```

You should now see additional OpenAI requests, including one more `Model: gpt-5-mini` entry from the `web-chat` run.

### 7. Cleanup

```bash
tmux kill-session -t gp55-mitm
tmux kill-session -t gp55-webchat
python3 - <<PY
from pathlib import Path
import shutil
for raw in ["$FLOW_FILE", "$CONF_DIR", "/tmp/gp55-sanitize-addon.py"]:
    p = Path(raw)
    if p.is_dir():
        shutil.rmtree(p)
    elif p.exists():
        p.unlink()
PY
```

## Exit Criteria

- `pinocchio examples test` completes successfully through `--proxy-url`.
- `tmux capture-pane -pt gp55-mitm:0.0` shows `POST https://api.openai.com/v1/chat/completions`.
- Safe replay prints `Model: gpt-5-mini` for the `assistant` example run.
- `web-chat` help shows the same `AI client flags` surface.
- `curl http://127.0.0.1:8090/chat/assistant ...` returns a runtime fingerprint containing both `gpt-5-mini` and the configured proxy URL.

## Notes

- Raw flow files contain `Authorization` headers. Delete them when you finish and never attach them to tickets or chat.
- The first failed experiment in this ticket was running Pinocchio against `--proxy-url http://127.0.0.1:8080` before the proxy listener existed. The failure mode was `proxyconnect tcp: dial tcp 127.0.0.1:8080: connect: connection refused`.
- The second common failure mode is forgetting `SSL_CERT_FILE="$CONF_DIR/mitmproxy-ca-cert.pem"`, which prevents TLS interception.
- For `web-chat`, `--profile assistant` on server startup is not enough to prove the `assistant` runtime on its own. The request route matters. Use `/chat/assistant` if the goal is specifically to sniff `gpt-5-mini`.
