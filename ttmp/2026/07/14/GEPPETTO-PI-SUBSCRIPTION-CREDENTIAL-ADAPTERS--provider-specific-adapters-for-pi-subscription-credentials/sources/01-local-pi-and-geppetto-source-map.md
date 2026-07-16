# Local Pi and Geppetto source map

This is an evidence index, not a credential export. It was created without reading, copying, hashing, or printing any value from `~/.pi/agent/auth.json`. Field names and provider behavior below come from installed Pi provider code and Geppetto source.

## Pi authentication framework

- `.../pi-coding-agent/dist/core/auth-storage.js:318-425`
  - registers provider login output as an OAuth credential;
  - locks the backing file while refreshing expired credentials;
  - returns a provider-derived API key/bearer at request time;
  - reloads after a refresh error in case another process completed the refresh.
- `.../pi-coding-agent/docs/providers.md`
  - documents subscription OAuth storage in `~/.pi/agent/auth.json` and automatic refresh.

## OpenAI Codex

- `.../pi-ai/dist/utils/oauth/openai-codex.js:22-35,112-125,319-338,450-535`
  - defines OpenAI authorization-code PKCE and device-code login;
  - exchanges and refreshes access/refresh credentials;
  - derives a ChatGPT account identifier from a token claim.
- `.../pi-ai/dist/providers/openai-codex.js:6-16`
  - registers the provider against the ChatGPT backend rather than the public OpenAI API host.
- `.../pi-ai/dist/api/openai-codex-responses.js:402-409,1163-1210`
  - targets `/codex/responses`;
  - sets `Authorization`, an account header, an originator header, and experimental Responses/SSE headers.

## Anthropic Claude subscription

- `.../pi-ai/dist/utils/oauth/anthropic.js:20-20,159-185,290-315`
  - implements authorization-code + PKCE and refresh-token grant handling;
  - requests scopes including inference and Claude Code session capability.

## Umans

- `~/.pi/agent/npm/node_modules/pi-provider-umans/index.ts:539-577`
  - `/login umans` prompts for an API key, stores it in access/refresh-shaped fields, and performs a no-op “refresh”.
- `~/.pi/agent/npm/node_modules/pi-provider-umans/index.ts:340-377,410-447`
  - direct extension side-calls use `POST /v1/messages` with the Anthropic version header and both `x-api-key` and bearer authorization header names; request bodies use Anthropic Messages blocks.
- `~/.pi/agent/npm/node_modules/pi-provider-umans/index.ts:564-569`
  - registers Pi inference as `api: "anthropic-messages"` with `authHeader: true`.
- `~/.pi/agent/npm/node_modules/pi-provider-umans/README.md:23-47`
  - identifies the gateway as Anthropic Messages compatible at `/v1/messages`, not OpenAI Chat/Responses compatible.

## Geppetto renewable-credential boundary

- `pkg/steps/ai/credentials/bearer.go:21-83,124-167,265-297`
  - defines `Request`, `Credential`, host-owned `Store`/`Refresher`, and the token-only `BearerTokenSource` interface;
  - keeps credential data out of inference settings and logs;
  - implements request-time load/refresh/caching.
- `pkg/inference/engine/factory/factory.go:134-151,205-253`
  - propagates a bearer source only to OpenAI Chat and Responses engines;
  - validates Claude independently.
- `pkg/steps/ai/openai/chat_stream.go:96-105,133-168`
  - resolves a bearer at request time, sets only `Authorization: Bearer`, and permits one pre-stream 401 replay for an opt-in source.
- `pkg/steps/ai/openai_responses/streaming.go:148-185`
  - has the analogous Responses-only bearer behavior.
- `pkg/steps/ai/claude/engine_claude.go:72-86` and `pkg/steps/ai/claude/api/completion.go:94-99`
  - read a static API key and send it as `x-api-key`; no renewable source option is currently present.
- `pkg/js/modules/geppetto/api_engine_builder.go:63-73`
  - keeps configured bearer sources in trusted Go runtime state and never represents them in JavaScript.

## Interpretation limits

Pi source proves how Pi currently implements its own subscription clients. It does not grant Geppetto permission to treat private/undocumented endpoints as stable public contracts. The design therefore requires fake-server protocol tests, explicit adapter selection, and an opt-in real smoke only after a provider-specific review accepts the current contract.
