---
Title: Credential Scope — Which Hosts a Provider Key Can Reach
Slug: geppetto-credential-scope
Short: Every provider credential in Geppetto is scoped to the origin it was configured for, and the HTTP client that carries it refuses every other origin, including after redirects.
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

# Credential scope

A provider credential in Geppetto (an API key, a bearer token, or a secret header from Pi's `models.json`) may only be sent to the **origin** it was configured for: the scheme, host and port of its base URL. Two mechanisms enforce this:

- **The key and its endpoint are resolved together**, from the same configuration entry.
- **The HTTP client that carries the key is pinned** to that origin. It refuses any request to another origin before anything is sent, including requests produced by redirects.

A refused request returns an error that wraps `security.ErrOriginNotAllowed`:

```text
request to https://other.example refused (allowed: https://gateway.example:443): origin not allowed for this credential
```

The message names the target by scheme and host only, never its query, because Gemini sends its key as `?key=`.

## Where the scope comes from

| Credential source | Scope |
|---|---|
| Profile slot (`openai-api-key`, `claude-api-key`, `cohere-api-key`, …) | the base URL in the **same slot** (`openai-base-url`, …), or that slot's documented default |
| Pi `Resolve` / `Decider` | the model's `baseUrl`, else the provider's own default, else an error |
| `credentials.Scoped(source, origins…)` | the given origins; a token is not produced for any other `Request.BaseURL` |

The Pi defaults are keyed by **provider**, not by wire protocol:

| Provider | Default |
|---|---|
| `openai` | `https://api.openai.com/v1` |
| `anthropic` | `https://api.anthropic.com` |
| `google` | `https://generativelanguage.googleapis.com` |
| `openai-codex` | `https://chatgpt.com/backend-api` |
| `lunaroute` | `https://gw.lunaroute.com/v1` |

Any other provider must set `baseUrl` in `models.json`. Otherwise `Resolve` and `Decider` fail with `*pi.NoEndpointError` and nothing is sent. (Before this rule, an `openai-completions` provider without `baseUrl` sent its key to `api.openai.com`.)

## Behaviour to be aware of

- **Redirects to another origin fail.** Same-origin redirects still work. If a provider legitimately redirects API calls elsewhere, configure its final URL.
- **OpenAI Responses reads the key and the base URL from one slot.**
  - An `open-responses-api-key` needs `open-responses-base-url`.
  - `openai-responses-*` may fall back to `openai-base-url`.
  - Only the OpenAI-named slots default to `api.openai.com`.
- **OpenAI embeddings honour `openai-base-url`**, as chat always did, and apply the outbound URL policy. A local OpenAI-compatible embeddings server needs `allow_http.embeddings` / `allow_local_networks.embeddings`.

## For code that sends requests

- To build a client for a settings slot, use `settings.EnsurePinnedHTTPClient(clientSettings, baseURL)` instead of `EnsureHTTPClient`. Pin to the URL actually used, including an adapter's own default when the profile names none.
- For other code, `security.PinClient(base, origins…)` pins any `*http.Client`. It returns a copy and never modifies `base`.
- The regression matrix `pkg/steps/ai/credscope` drives every credential path with a marker secret, and asserts that the marker never reaches another origin. A new engine or adapter that attaches a credential must add a row.
