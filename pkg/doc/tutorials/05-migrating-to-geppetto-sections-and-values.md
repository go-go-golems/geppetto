---
Title: Migrate to Geppetto Sections and Values (No Compatibility Layer)
Slug: geppetto-migrate-sections-values
Short: Hard-cut migration guide from old Geppetto layer/parameter symbols to the new section/value facade API.
Topics:
- geppetto
- migration
- tutorial
- facade
- glazed
- sections
- values
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This tutorial shows how to migrate existing Geppetto integrations from the old layer/parameter APIs to the new section/value facade APIs, with no backward compatibility shims. It is intended for maintainers who want to complete a hard-cut migration and remove deprecated symbol usage.

What changes conceptually:
- `layers` becomes `sections` for schema definitions.
- `parameters` becomes `values` for parsed runtime data.
- helper APIs that accepted `ParsedLayers` now accept `ParsedValues`.

## Scope and Preconditions

Use this guide when your code currently imports old Geppetto symbols such as:
- `github.com/go-go-golems/geppetto/pkg/layers`
- `factory.NewEngineFromParsedLayers(...)`
- `settings.NewStepSettingsFromParsedLayers(...)`
- `stepSettings.UpdateFromParsedLayers(...)`
- `embeddings.NewSettingsFactoryFromParsedLayers(...)`

Before starting:
- update to a Geppetto version that exports `pkg/sections` and `...FromParsedValues` APIs.
- ensure your Glazed usage is already on `schema`/`values` in command pipelines.

## Symbol Mapping

Use this mapping as the canonical replacement list.

| Old symbol | New symbol |
|---|---|
| `geppetto/pkg/layers` | `geppetto/pkg/sections` |
| `CreateGeppettoLayers` | `CreateGeppettoSections` |
| `NewEngineFromParsedLayers` | `NewEngineFromParsedValues` |
| `NewStepSettingsFromParsedLayers` | `NewStepSettingsFromParsedValues` |
| `(*StepSettings).UpdateFromParsedLayers` | `(*StepSettings).UpdateFromParsedValues` |
| `embeddings.NewSettingsFactoryFromParsedLayers` | `embeddings.NewSettingsFactoryFromParsedValues` |
| `settings.NewChatParameterLayer` | `settings.NewChatValueSection` |
| `settings.NewClientParameterLayer` | `settings.NewClientValueSection` |
| `openai.NewParameterLayer` | `openai.NewValueSection` |
| `claude.NewParameterLayer` | `claude.NewValueSection` |
| `gemini.NewParameterLayer` | `gemini.NewValueSection` |
| `ollama.NewParameterLayer` | `ollama.NewValueSection` |
| `config.NewEmbeddingsParameterLayer` | `config.NewEmbeddingsValueSection` |

## Import Migration

Before:

```go
import geppettolayers "github.com/go-go-golems/geppetto/pkg/layers"
```

After:

```go
import geppettosections "github.com/go-go-golems/geppetto/pkg/sections"
```

## Command Schema Wiring

Before:

```go
geppettoLayers, err := geppettolayers.CreateGeppettoLayers()
if err != nil { return err }

desc := cmds.NewCommandDescription(
    "run",
    cmds.WithLayers(geppettoLayers...),
)
```

After:

```go
geppettoSections, err := geppettosections.CreateGeppettoSections()
if err != nil { return err }

desc := cmds.NewCommandDescription(
    "run",
    cmds.WithSections(geppettoSections...),
)
```

## Parsed Data and Engine Factory

Before:

```go
eng, err := factory.NewEngineFromParsedLayers(parsedLayers)
if err != nil { return err }
```

After:

```go
eng, err := factory.NewEngineFromParsedValues(parsedValues)
if err != nil { return err }
```

If you build settings first:

Before:

```go
stepSettings, err := settings.NewStepSettingsFromParsedLayers(parsedLayers)
if err != nil { return err }
```

After:

```go
stepSettings, err := settings.NewStepSettingsFromParsedValues(parsedValues)
if err != nil { return err }
```

Before:

```go
if err := stepSettings.UpdateFromParsedLayers(parsedLayers); err != nil {
    return err
}
```

After:

```go
if err := stepSettings.UpdateFromParsedValues(parsedValues); err != nil {
    return err
}
```

## Embeddings Factory Migration

Before:

```go
factory, err := embeddings.NewSettingsFactoryFromParsedLayers(parsedLayers)
if err != nil { return err }
```

After:

```go
factory, err := embeddings.NewSettingsFactoryFromParsedValues(parsedValues)
if err != nil { return err }
```

## Provider Section Constructors

When you directly assemble sections, switch constructor names:

```go
chat, _ := settings.NewChatValueSection()
client, _ := settings.NewClientValueSection()
openaiSection, _ := openai.NewValueSection()
claudeSection, _ := claude.NewValueSection()
geminiSection, _ := gemini.NewValueSection()
ollamaSection, _ := ollama.NewValueSection()
embeddingsSection, _ := config.NewEmbeddingsValueSection()
```

## Validation Checklist

After replacement, verify all of the following:
- no imports of `geppetto/pkg/layers`.
- no calls to `*FromParsedLayers`.
- no calls to `*ParameterLayer` constructors for Geppetto settings.
- project builds and tests with current Geppetto and Glazed.
- `go mod tidy` removes stale module edges introduced by deleted old symbols.

Useful checks:

```bash
rg -n "pkg/layers|FromParsedLayers|ParameterLayer" .
go test ./...
make lint
```

## Troubleshooting

| Problem | Cause | Fix |
|---|---|---|
| `undefined: factory.NewEngineFromParsedLayers` | Old helper removed | replace with `factory.NewEngineFromParsedValues` |
| `module ... does not contain package .../pkg/layers` | Import path removed | import `.../pkg/sections` and update calls |
| decode/update errors after rename | mixed `parsedLayers` and `parsedValues` variables | standardize on `*values.Values` and new helper names |

## See Also

- `../tutorials/01-streaming-inference-with-tools.md`
- `../topics/06-inference-engines.md`
- `../topics/06-embeddings.md`
