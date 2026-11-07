---
Title: Bound API Connections and Credential Scope
Slug: geppetto-credential-scope
Short: Pass one connection containing the endpoint, routes, authentication, headers and HTTP policy to any AI API client.
Topics:
- geppetto
- security
- credentials
- providers
Commands: []
Flags: []
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

An authenticated API call uses one `connection.Connection`. It owns the provider's base URL, named routes, authentication, default headers, URL policy and HTTP client. Payload code supplies an operation and request body. It does not look up keys, select endpoint defaults, refresh credentials or construct a client again.

The connection fixes the allowed **origin** (scheme, host and port) from its source declaration before routing. It checks the actual destination before credential lookup and checks every redirect hop before sending. A route resolver or later settings change cannot grant permission to send the credential to another origin.

## Construct once at the credential source

Create the connection where the credential and its intended endpoint are known together. Pass a host-owned HTTP client here to retain its timeout, proxy and TLS settings. Do not read a key and URL from unrelated fallback chains.

```go
c, err := connection.New(connection.Config{
    Provider: "gateway",
    BaseURL:  "https://gateway.example/v1",
    Routes: connection.Routes{
        "chat":         connection.Relative("chat/completions"),
        "responses":    connection.Relative("responses"),
        "input-tokens": connection.Relative("responses/input_tokens"),
        "transcribe":   connection.Relative("audio/transcriptions"),
    },
    Auth:       connection.StaticBearer(apiKey),
    Headers:    http.Header{"X-Team": {"research"}},
    HTTPClient: hostClient,
})
if err != nil {
    return err
}
```

`Headers` supplies copied default headers. Authentication uses declared sensitive header carriers and overwrites its own carriers at send time. Arbitrary middleware can set only its declared headers; it cannot change the destination or body. A connection's fields are private, and it exposes no secret, mutable URL, client setter or underlying transport getter.

For a renewable source, use `connection.Bearer(source)`. Other header carriers use `connection.Token(source, connection.TokenHeaders{"x-api-key": ""})`; an empty prefix places the token directly in that header. Anthropic gateway dual auth uses both `x-api-key` and `Authorization: Bearer `. `claude.Authentication(source, true)` supplies subscription OAuth carriers and identity headers. `umans.NewConnection(key, client)` binds Umans credentials to its own issuer.

## Make an authenticated request

The operation name selects a registered route below the base URL. Relative routes reject absolute URLs, queries and traversal segments. The connection also checks SDK-created requests against its fixed origin.

```go
req, err := c.NewRequest(ctx, http.MethodPost, "chat", bytes.NewReader(payload))
if err != nil {
    return err
}
req.Header.Set("Content-Type", "application/json")
resp, err := c.Do(req)
if err != nil {
    return err
}
defer resp.Body.Close()
```

`Do` owns at most one unauthorized replay before decoding when the authentication source supports refresh and the body is replayable. It keeps retry state per call, so a connection can be shared by concurrent clients. `DoWithObserver` supplies a redacted diagnostic request; the returned response's request metadata also redacts declared authentication and default headers.

A refused origin wraps `security.ErrOriginNotAllowed`; use `errors.Is` to classify it. Same-origin redirects work unless `RejectRedirects` is set. Strict decision/rerank source constructors and OAuth token grants reject all redirects. Host redirect callbacks cannot bypass the final origin check.

## Share the same setup across API families

Choose the protocol/model options separately from the connection. Each client below sends through the same request boundary.

| API | Connection input |
|---|---|
| OpenAI chat | `openai.NewOpenAIEngine(c, s)` |
| OpenAI Responses | `openai_responses.NewEngine(c, s)` |
| Anthropic | `claude.NewClaudeEngine(c, s)`; low-level `api.NewClient(c)` |
| Gemini | `gemini.NewGeminiEngine(c, s)`; `geminisdk.NewClient(ctx, c, version)` for Models, Caches, Files, counting and embeddings |
| OpenAI embeddings | `embeddings.NewOpenAIProvider(c, model, dimensions)` |
| Cohere embeddings | `embeddings.NewCohereProvider(c, model, dimensions)` |
| Ollama embeddings | `embeddings.NewOllamaProvider(c, model, dimensions)` |
| Cohere / llama.cpp / Qwen3 rerank | provider `Options{Connection: c, Model: model}` |
| System One decisions | `systemone.Options{Connection: c, Model: model}` |
| Whisper transcription | `openai.NewTranscriptionClient(c)` |
| Token counters | `NewTokenCounter(c, s)`, returning a counter and error |
| Generic engine factory | `factory.NewStandardEngineFactory(factory.WithConnection(c))` |
| Generic counter factory | `factory.NewFromSettings(s, factory.WithConnection(c))` |
| Pi | `resolved.Engine()` or reuse the opaque `resolved.Connection()` |

Register the routes the chosen native client uses: OpenAI chat provides `openai.Routes()`, Responses `openai_responses.Routes()`, Anthropic `api.Routes()`, embeddings `CohereRoutes()` / `OllamaRoutes()`, and rerank/decision providers their `Routes()`. SDK factories apply the same bound origin to every SDK request. OpenAI's SDK holds no real API key; Gemini receives only an inert constructor placeholder, replaced by the connection's auth before sending.

Settings factories are the configuration entrypoints: `connectionfactory.New(s, source)` resolves one declared slot and its provider-owned defaults. Inference and token-counter constructors require the resulting connection as their first argument and copy only payload settings. They do not accept independent authentication options. Generic factories can resolve settings or accept one complete connection.

## Keep credential pairs atomic in configuration

An embedding-local override must provide both its key and its URL. Per-call embedding credential overrides have the same rule. Model/dimension-only overrides do not affect authority.

```yaml
inference_settings:
  embeddings:
    type: openai
    engine: text-embedding-3-small
    api_keys:
      openai-api-key: <gateway key>
    base_urls:
      openai-base-url: https://gateway.example/v1
```

Profile stacks also replace existing authenticated bindings as a pair. A key-only or URL-only change is refused; blank entries preserve the original pair. A model-only profile can inherit a complete provider binding unchanged. A host that supplies already merged raw settings is responsible for preserving that pairing before calling Geppetto.

Responses reads only the `open-responses` slot and requires both `open-responses-api-key` and `open-responses-base-url` unless an external source supplies the credential. It does not search the `openai` slot or the removed `openai-responses` alias. Its outbound policy also comes only from the canonical slot. OpenAI embeddings use the same `openai-base-url` declaration as chat. Local hosted-compatible servers need explicit HTTP/local-network policy; anonymous local Ollama retains its local provider policy.

Canonical runtime endpoints live in `providers/defaults`, keyed by issuer/provider rather than protocol. They include OpenAI, Anthropic, Google, Codex, LunaRoute, Cohere and Umans. Pi providers outside that table must name their endpoint. OAuth grant clients bind the configured token endpoint separately from the inference endpoint.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| Origin refusal | Route, SDK or redirect chose another origin | Declare the intended final endpoint at the source; use a separate binding for another origin |
| Unknown route | The connection has no named operation | Register the client's route table when constructing the connection |
| Partial credential override | Key and URL come from different configuration layers | Supply the complete pair or inherit the existing binding unchanged |
| Connection already owns authentication | Both a connection and separate auth inputs were supplied | Pass the connection with only model/payload options |
| Local server refused | HTTP/private-network policy is absent | Opt in at the trusted source declaration |
| Missing key during construction | Source resolution produced no credential | Configure the selected slot or pass a complete connection |

The binding enforces configured authority. An opaque key does not reveal which host issued it; the configuration source must declare the intended endpoint truthfully. Injected Go transports, credential sources and middleware are trusted code. Origin scope does not enforce tenant path isolation or DNS/proxy trust.

## See Also

- `migrate-connection-only-inference`: constructor and configuration migration examples.

- `pkg/steps/ai/connection/example_test.go`: compiled request and shared-client examples.
- `pkg/steps/ai/credscope`: the 64-path recording matrix; add a row for each new authenticated API path.
- The `embeddings`, `inference-engines` and `geppetto-decision-package` help topics.
