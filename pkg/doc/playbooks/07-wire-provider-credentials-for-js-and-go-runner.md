---
Title: "Wire provider credentials for JS and go runner"
Slug: wire-provider-credentials-js-go-runner
Short: Resolve provider credentials through Geppetto/Pinocchio profiles and pass final settings to engines.
Topics:
  - profiles
  - configuration
  - javascript
  - extraction
Commands:
  - geppetto
  - temporal-relationships
Flags:
  - profile
  - profile-registries
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

# Wire provider credentials for JS and go runner

Provider credentials belong in Geppetto/Pinocchio profile registries. Applications should resolve the selected profile stack, receive final merged inference settings, and pass those settings to engine or embedding factories.

Use the user's registry by default:

```text
~/.config/pinocchio/profiles.yaml
```

Go applications should expose profile selection rather than provider-key flags:

```go
resolved, err := profilebootstrap.ResolveCLIEngineSettings(ctx, parsedValues)
if err != nil {
    return err
}
defer resolved.Close()

engine, err := factory.NewEngineFromInferenceSettings(resolved.FinalInferenceSettings)
if err != nil {
    return err
}
```

JS applications should use profile-aware APIs or runtime settings produced from a resolved profile. If a command needs to override chat-layer details, use the Geppetto chat/profile flags so the final settings still come from the same merge path.

Do not read provider keys directly in application code. Keep static keys in profile base layers, such as an OpenAI base profile, and stack model-specific chat or embedding profiles on top.

Renewable OAuth-style credentials are different: the host owns their store and refresher, injects `factory.WithBearerTokenSource`, and must not serialize refresh material into inference settings or profile API-key maps. See [Use renewable bearer credentials with OpenAI-compatible engines](08-use-renewable-bearer-credentials.md).
