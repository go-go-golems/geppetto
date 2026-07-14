---
Title: "Use renewable bearer credentials with OpenAI-compatible engines"
Slug: "use-renewable-bearer-credentials"
Short: "Inject host-owned renewable bearer credentials without placing refresh state in inference settings."
Topics:
- oauth
- credentials
- inference
Commands: []
Flags: []
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

Renewable bearer credentials let a host supply access tokens at request time while Geppetto continues to own provider request mechanics. This is useful for OAuth-style credentials that expire and rotate; it prevents refresh tokens from entering `InferenceSettings.APIKeys`, profile settings, logs, or JavaScript values.

## Build a host-owned source

The host implements `credentials.Store` and `credentials.Refresher`, then constructs one `credentials.RenewableBearerTokenSource`. The store owns persistence and atomic rotation; the refresher owns provider policy. Neither implementation may include credential material in errors.

```go
source, err := credentials.NewRenewableBearerTokenSource(store, refresher)
if err != nil {
    return err
}
factory := factory.NewStandardEngineFactory(factory.WithBearerTokenSource(source))
engine, err := factory.CreateEngine(inferenceSettings)
```

The source is authoritative for supported OpenAI-compatible engines. Static API keys remain a fallback only when no source is configured.

## Understand cancellation and 401 recovery

Each request can cancel its own wait for a token. Shared load, refresh, and persistence work continues independently so one canceled request does not abort other callers. A pre-stream provider 401 may trigger exactly one replacement-token replay when the source implements `UnauthorizedBearerTokenSource`; a second 401 is returned unchanged.

Forced refreshes are separated by an opaque, process-local keyed fingerprint of the rejected bearer. The bearer is never persisted or logged as a coordination key.

## JavaScript hosts

`require("geppetto").engine().inference(...).build()` currently creates an engine without a bearer-source injection point. Do not expose refresh tokens or a JavaScript token callback as a workaround. A host embedding JavaScript must create the engine in Go with `factory.WithBearerTokenSource` and pass the resulting engine into the runtime until a dedicated host-only injection API exists.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| Factory requires a static key | No bearer source was supplied | Construct the factory with `factory.WithBearerTokenSource(source)`. |
| Refresh material appears in settings | Host put OAuth state in `APIKeys` | Keep access/refresh state exclusively in the host store. |
| A request retries repeatedly after 401 | Caller implements its own replay loop | Rely on the provider path's single bounded replay. |
| JavaScript-built engine uses a static key | JS builder has no bearer-source option | Inject a Go-created engine; do not pass tokens through JS. |

## See Also

- [Wire provider credentials for JS and go runner](07-wire-provider-credentials-for-js-and-go-runner.md)
- [Inference Engines](../topics/06-inference-engines.md)
- [Profiles](../topics/01-profiles.md)
