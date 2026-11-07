---
Title: Understanding and Using the Decision Package
Slug: geppetto-decision-package
Short: Decision models (System One, Jev-style) in Geppetto — typed noul/choice/score questions, the LunaRoute adapter, profiles, Pi credentials and the JavaScript API.
Topics:
- geppetto
- decisions
- classification
- retrieval
- ai
- providers
- security
Commands: []
Flags:
- decision-type
- decision-engine
- systemone-api-key
- systemone-base-url
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

# Decision models in Geppetto

A decision model answers typed questions about a *state* without generating text. You declare each question's answer space, and the model returns a probability distribution over exactly that space. There is nothing to parse, an answer cannot fall outside the space, and every answer carries probabilities. LunaRoute serves such a model, `djev`, on its System One endpoint. TypeSafe's Jev uses the same question model.

`pkg/decision` is Geppetto's fourth model-service primitive, after inference (`pkg/inference/engine`), embeddings (`pkg/embeddings`) and rerank (`pkg/rerank`). It follows rerank's layering: core types and validation, a provider adapter, a settings factory, profile wiring, and a JavaScript binding.

## Question types

| Type | Declares | Answer fields |
|---|---|---|
| `noul` | a proposition; optional `True`/`False` descriptions | `Probability` that it is true |
| `choice` | 1–255 `Options` (name, description), in order | `Choice`, `Confidence`, `Probabilities` per option |
| `score` | 1–10 ordered `Levels` | fractional `Score`, `Confidence`, `LevelProbabilities` |

All questions in one request share the state, which the provider counts once. Batching many questions into one request is cheaper and barely slower than asking one. On djev, twelve passage-relevance questions took about 0.6 s.

Option order matters to these models, so `Options` is a slice and the adapter sends it in slice order.

## Go API

```go
import (
    "github.com/go-go-golems/geppetto/pkg/decision"
    "github.com/go-go-golems/geppetto/pkg/decision/systemone"
)

p, err := systemone.New(systemone.Options{APIKey: key, Model: "djev"})
resp, err := p.Decide(ctx, decision.Request{
    State: map[string]any{"query": q, "corpus": "engineering design notes"},
    Questions: map[string]decision.Question{
        "route": {Type: decision.Choice, Instructions: "Which retrieval strategy fits?",
            Options: []decision.Option{{Name: "keyword", Description: "identifiers"}, {Name: "semantic", Description: "concepts"}}},
        "needs": {Type: decision.Noul, Instructions: "Does answering require documents?"},
        "complexity": {Type: decision.Score, Instructions: "How complex?",
            Levels: []string{"lookup", "single fact", "mechanism", "synthesis"}},
    },
})
resp.Answers["route"].Choice        // "semantic"
resp.Answers["needs"].Probability   // 0.91
decision.Gate(resp.Answers["needs"].Probability, 0.8, 0.5) // decision.Act
```

`Decide` validates the request (question ids `[A-Za-z0-9_.-]{1,64}`, type-specific fields only, option and level limits, data-URL images) and the response (one answer per question, of its type, with a finite probability for every declared option or level).

Errors wrap sentinels, which you test with `errors.Is`:

- `ErrInvalidRequest`
- `ErrInvalidResponse`
- `ErrUnavailable`
- `ErrRateLimited`
- `ErrNotGranted`
- `ErrModelNotFound`
- `ErrRequestTooLarge`
- `ErrResponseTooLarge`

Non-2xx responses are a `*decision.ProviderError` with the provider's `Code`, `RequestID` and `RetryAfter`.

## Pi credentials

```go
a, _ := pi.LoadAgentDir(pi.DefaultDir(os.Getenv(pi.EnvAgentDir)))
d, _ := a.Decider(ctx, "lunaroute/djev", pi.DecisionOptions{})
```

`Decider` uses the provider's base URL, any `models.json` headers, and Pi's credential (`auth.json`, then `models.json` `apiKey`). The endpoint comes from the model's provider: its `baseUrl`, else that provider's own default (`lunaroute` → `https://gw.lunaroute.com/v1`), else `*pi.NoEndpointError`. The decider's client is pinned to that origin (see the `geppetto-credential-scope` help topic). To try it:

```bash
go run ./cmd/examples/decision-pi-smoke run --query "how does the rebalance law preserve placements?"
go run ./cmd/examples/decision-pi-smoke run --query "..." --passage "first passage" --passage "second passage"
```

## Profiles

```yaml
profiles:
  djev:
    inference_settings:
      api:
        api_keys:
          systemone-api-key: ${LUNAROUTE_API_KEY}
        # base_urls: { systemone-base-url: https://gw.lunaroute.com/v1 }
      decision:
        type: systemone
        engine: djev
```

`factory.NewSettingsFactoryFromInferenceSettings(s).NewProvider()` builds the provider. Plain HTTP and local addresses need `allow_http.decision` and `allow_local_networks.decision`. CLI flags: `--decision-type`, `--decision-engine`, `--decision-max-request-bytes`, `--decision-max-response-bytes`, `--systemone-api-key`, `--systemone-base-url`.

## JavaScript

```js
const gp = require("geppetto");
const d = gp.decider(gp.inferenceProfiles.load(path).resolve("djev"));
const r = d.decide({
  state: { query },
  questions: {
    route: { type: "choice", instructions: "Which index?", options: { keyword: "identifiers", semantic: "concepts" } },
    needs: { type: "noul", instructions: "Needs documents?", true: "corpus facts" },
    hard:  { type: "score", instructions: "How hard?", levels: ["easy", "medium", "hard"] },
  },
});
r.answers.route.choice; r.answers.needs.probability; r.answers.hard.levelProbabilities;
const h = d.decideAsync(req); // { promise, cancel, close }
```

Choice options given as an object keep JavaScript insertion order. Unknown keys are rejected.

## Limits and cautions

- Decision models do not generate text. They are also weak at arithmetic, counting, and date comparison.
- Option order changes probabilities. Test critical questions with the options reversed.
- What the state says matters. Describing the corpus in the state moved a "needs retrieval?" probability from 0.08 to 0.99 in our probes.
- The state is untrusted input. Treat a decision as an input to policy, not as an authorization.
