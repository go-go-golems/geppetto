---
Title: Model Sources: Pi Agent Directory and Engine Profiles
Slug: geppetto-model-sources
Short: List and resolve models from Pi's agent directory and from engine profile registries with one reference syntax.
Topics:
- geppetto
- credentials
- providers
- profiles
Commands: []
Flags: []
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

An application that runs models usually has credentials in more than one place. Pi, the coding agent, keeps logins and API keys in its agent directory (`auth.json`, `models.json`, `models-store.json`). Pinocchio and other Geppetto applications keep engine profiles in a registry such as `~/.config/pinocchio/profiles.yaml`. Package `pkg/inference/modelsource` puts both behind one interface: list the models that can run now, and resolve a reference into an engine.

## References

A reference names a source and a model in it.

| Reference | Source |
|---|---|
| `pi:lunaroute/deepseek-4.1-flash` | Pi agent directory, provider `lunaroute`, model `deepseek-4.1-flash` |
| `profile:claude` | engine profile `claude`, found by registry precedence |
| `profile:team/claude` | engine profile `claude` in registry `team` |
| `lunaroute/deepseek-4.1-flash` | no prefix: offered to each source in order |

A prefixed reference goes only to the source with that prefix. An unprefixed reference is tried in source order; the first source that knows it answers. `errors.Is(err, modelsource.ErrUnknownModel)` reports that no source knows a reference.

## Loading the default sources

```go
set, notes, err := modelsource.Load(ctx, modelsource.LoadOptions{})
if err != nil {
    return err
}
defer set.Close()
for _, n := range notes {
    log.Info().Msg(n) // e.g. "profiles: ~/.config/pinocchio/profiles.yaml not found"
}

models, err := set.Models(ctx) // only models with a credential
r, err := set.Resolve(ctx, "pi:lunaroute/deepseek-4.1-flash", modelsource.Options{Thinking: "low"})
eng, err := r.Engine()
```

`Load` reads engine profiles first (`LoadOptions.ProfileSources`, default `<UserConfigDir>/<AppName>/profiles.yaml` with `AppName` defaulting to `pinocchio`), then Pi (`LoadOptions.PiDir`, default `$PI_CODING_AGENT_DIR` or `~/.pi/agent`). A source whose files do not exist is skipped and named in `notes`. A source that exists but cannot be parsed is an error.

To build a set by hand, use `modelsource.Chain(modelsource.FromProfiles(registry), modelsource.FromPi(agentDir))`.

## What a resolved model carries

`Resolved.Model` describes the model without secrets: reference, label, API, context window, maximum output tokens, reasoning and image support, thinking levels and, when a profile declares it, cost per million tokens. `Resolved.Settings` are payload settings with credential maps removed. `Resolved.Engine()` returns an engine whose connection is already bound to the credential's own endpoint (see `geppetto-credential-scope`); callers never handle the key.

`Options.MaxTokens` caps one response. `Options.Thinking` is a Pi thinking level (`off`, `minimal`, `low`, `medium`, `high`, `xhigh`); Pi maps it per model and profiles ignore it, because a profile already configures its provider's reasoning settings.

## Behaviour of each source

- **Pi** lists only models whose provider has a credential (an `auth.json` entry or a `models.json` `apiKey`) and whose API Geppetto can speak. OAuth credentials are refreshed under a lock that Pi itself respects.
- **Profiles** list every profile whose resolved settings name an engine; base layers without an engine are not listed. A profile is resolved by merging it over default inference settings, so it may set only what differs. Credentials are checked when the engine is built, not when the profile is listed.

## See Also

- `geppetto-credential-scope`: bound connections and why engines never see credentials.
- `migrate-connection-only-inference`: the constructor change these engines depend on.
