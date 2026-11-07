---
Title: Migrate to connection-only inference construction
Slug: migrate-connection-only-inference
Short: Update engine constructors, token counters, and Responses settings to use one explicit API connection.
Topics:
- inference
- credentials
- migration
Commands: []
Flags: []
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

Inference engines and token counters now require a connection as their first argument. Settings factories resolve credentials and endpoints; payload constructors consume the result. There are no compatibility wrappers for the removed constructor options.

## Update direct Go callers

Previously, an engine could resolve credentials itself:

```go
// Before
engine, err := openai.NewOpenAIEngine(s, openai.WithBearerTokenSource(source))
```

Resolve the binding at the source boundary, then pass it explicitly. These snippets assume imports and surrounding error handling; `source` is a host-owned token source, or nil for the selected settings key.

```go
// After
c, err := connectionfactory.New(s, source)
if err != nil {
    return err
}
engine, err := openai.NewOpenAIEngine(c, s)
```

Apply the same argument order to the other constructors:

```go
responses, err := openai_responses.NewEngine(c, s)
claudeEngine, err := claude.NewClaudeEngine(c, s)
geminiEngine, err := gemini.NewGeminiEngine(c, s)
counter, err := openai_responses.NewTokenCounter(c, s)
claudeCounter, err := claude.NewTokenCounter(c, s)
```

Each line is an independent example: use a connection with the selected protocol's routes and authentication. Nil connections fail immediately. Keep observer and payload options on the engine; move custom authentication, headers, routes and HTTP policy into connection construction. Remove engine `WithConnection`, bearer options, and token-counter connection/auth options.

Existing generic factory callers can continue using `factory.NewEngineFromSettings(s)`, or `factory.NewStandardEngineFactory(factory.WithBearerTokenSource(source))`. Factories resolve the connection before constructing the payload engine. A host that already owns a connection can still pass `factory.WithConnection(c)` to the factory.

## Make Responses configuration explicit

Rename the protocol and its Go constant to `open-responses` / `types.ApiTypeOpenResponses`. The old `openai-responses` alias is removed. Responses no longer searches OpenAI or legacy slots for credentials, URLs, or outbound policy.

Declare the canonical pair, even when calling OpenAI itself:

```yaml
inference_settings:
  chat:
    api_type: open-responses
    engine: gpt-5-mini
  api:
    api_keys:
      open-responses-api-key: <key for this endpoint>
    base_urls:
      open-responses-base-url: https://api.openai.com/v1
```

Move relevant policy entries to the `open-responses` key too. Keep credentials in your existing trusted configuration source; do not put real keys into committed examples. Ordinary `openai` chat configuration is unchanged. Pi's external catalog protocol name `openai-responses` is still supported and translates to the canonical internal protocol; it is not a settings fallback.

## Replace custom Responses transport options

Remove `WithRequestTransport`. For Codex, construct its canonical connection directly:

```go
c, err := openaicodex.NewConnection(source, openaicodex.Options{}, hostClient)
if err != nil {
    return err
}
engine, err := openai_responses.NewEngine(c, s)
```

Other integrations use `connection.New` with explicit routes and authentication. Pi performs this binding itself. Embedding constructors already take connections; this change does not redesign direct rerank or decision convenience constructors.

## Verify the migration

Search callers for the removed options and alias, compile the application, then run its ordinary tests. Verify at least one configured request per protocol with a recording transport. Confirm the intended endpoint receives authentication and a foreign redirect is refused. Run socket-based tests in an environment that permits local test servers.

| Problem | Cause | Fix |
|---|---|---|
| Missing constructor argument | Old direct constructor call | Pass the resolved connection first |
| Missing Responses key or URL | Reliance on another settings slot | Declare the complete canonical pair |
| Unknown operation | Connection lacks a protocol route | Register the payload client's routes |

## See Also

- `geppetto-credential-scope`: connection ownership, route tables and trust limits.
- `use-renewable-bearer-credentials`: host-owned credential renewal.
