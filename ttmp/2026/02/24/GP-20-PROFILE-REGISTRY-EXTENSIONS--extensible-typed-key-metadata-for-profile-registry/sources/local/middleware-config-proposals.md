---
Title: Imported Source - Middleware Config Proposals
Ticket: GP-20-PROFILE-REGISTRY-EXTENSIONS
Status: active
Topics:
    - architecture
    - geppetto
    - pinocchio
    - chat
    - backend
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources:
    - /tmp/middleware-config-proposals.md
Summary: Imported external proposal document used as design input for middleware configuration unification.
LastUpdated: 2026-02-24T23:58:00-05:00
WhatFor: Preserve the imported proposal in ticket-local sources for auditability and future reference.
WhenToUse: Use when reviewing original proposal text during implementation/planning.
---

You’re running into the classic Go tension:

* **Runtime middleware** wants to stay the idiomatic `type Middleware func(HandlerFunc) HandlerFunc`.
* **Configurability + introspection** needs a *thing* with methods: schema, defaults, validation, build-time deps, etc.

So the move is: **keep `middleware.Middleware` as-is**, but introduce a *separate, first-class “middleware definition” concept* that can (a) describe config, and (b) instantiate the actual func middleware.

Below are several designs, from minimal to “this scales to UI + profiles + layered overrides”, all compatible with the existing `Middleware` func type.

---

## Design goals (what we’re optimizing for)

1. **Self-describing parameters**

   * list of fields, types, defaults, help text, required-ness, etc.
2. **Layered configuration with provenance**

   * defaults → profile → config file → env → flags → request override
   * keep per-field “where did this value come from?” (glazed already has `FieldValue.Log`)
3. **Works with “middleware is just a func”**

   * no need to change `middleware.Middleware` type
4. **Profile-friendly**

   * a profile can ship a middleware chain + default config
   * consumers can “refine” config without replacing the whole chain
5. **Allows build-time deps**

   * e.g. logger, tool registry, event sinks, database handles, etc.

---

## Proposal 1: Minimal “Descriptor + Factory” registry (config is a map)

This is the lowest ceremony: each middleware registers a descriptor containing:

* name
* description
* schema (glazed `schema.Section`, or your own minimal schema struct)
* factory that takes `map[string]any` and returns the middleware func

### Core types

```go
// pkg/inference/middlewarecfg/registry.go
package middlewarecfg

import (
	"context"
	"fmt"

	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
)

type BuildDeps struct {
	// Add what you need: logger, tool registry, etc.
}

type Definition struct {
	Name        string
	Description string
	Section     schema.Section               // glazed section describing config
	Build       func(ctx context.Context, deps BuildDeps, cfg map[string]any) (middleware.Middleware, error)
}

type Registry struct {
	defs map[string]Definition
}

func NewRegistry(defs ...Definition) *Registry {
	r := &Registry{defs: map[string]Definition{}}
	for _, d := range defs {
		r.defs[d.Name] = d
	}
	return r
}

func (r *Registry) Get(name string) (Definition, bool) {
	d, ok := r.defs[name]
	return d, ok
}

func (r *Registry) Build(ctx context.Context, deps BuildDeps, name string, cfg map[string]any) (middleware.Middleware, error) {
	d, ok := r.defs[name]
	if !ok {
		return nil, fmt.Errorf("unknown middleware: %s", name)
	}
	return d.Build(ctx, deps, cfg)
}
```

### Example: systemPrompt middleware

```go
import (
	"context"
	"fmt"

	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
)

func SystemPromptDefinition() (middlewarecfg.Definition, error) {
	sec, err := schema.NewSection(
		"mw-systemprompt",
		"System prompt middleware",
		schema.WithPrefix("mw-systemprompt-"),
		schema.WithFields(
			fields.New("prompt", fields.TypeString, fields.WithHelp("System prompt text")),
		),
	)
	if err != nil {
		return middlewarecfg.Definition{}, err
	}

	return middlewarecfg.Definition{
		Name:        "systemPrompt",
		Description: "Ensures a fixed system prompt block exists (adds or replaces).",
		Section:     sec,
		Build: func(ctx context.Context, deps middlewarecfg.BuildDeps, cfg map[string]any) (middleware.Middleware, error) {
			prompt, _ := cfg["prompt"].(string)
			if prompt == "" {
				return nil, fmt.Errorf("systemPrompt.prompt is required")
			}
			return middleware.NewSystemPromptMiddleware(prompt), nil
		},
	}, nil
}
```

### Pros / cons

✅ Very simple
✅ Keeps middleware as func
✅ Easy to register lots of middleware quickly
❌ Config parsing/validation is ad-hoc unless you also run it through glazed fields
❌ No strong typing; you’ll end up writing a bunch of `toInt/toString` helpers
❌ Layering + provenance is not “built-in”; you still have to wire glazed Values yourself

---

## Proposal 2: Typed configurable middleware + Glazed Values layering (recommended)

This is the “feels like glazed commands” approach:

* Each middleware defines a **config struct**.
* Each middleware provides a **glazed section** describing that struct’s fields.
* At runtime, you build a `schema.Schema` containing the sections for the active middleware instances.
* You then run `sources.Execute(...)` to layer defaults/profile/env/flags/overrides into `*values.Values`.
* Finally, each middleware decodes its section into its config struct and builds the actual `middleware.Middleware` func.

This directly gives you:

* a self describing schema
* layered sources
* per-field provenance (glazed `FieldValue.Log`)
* type safety in the builder

### A concrete registry design

```go
// pkg/inference/middlewarecfg/typed.go
package middlewarecfg

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

type BuildDeps struct {
	// Logger zerolog.Logger
	// ToolRegistry tools.ToolRegistry
	// ...
}

type Use struct {
	Name   string
	ID     string // optional stable instance key; default = Name
	Config any    // profile-provided config blob (often map[string]any)
}

func (u Use) InstanceKey() string {
	if strings.TrimSpace(u.ID) != "" {
		return strings.TrimSpace(u.ID)
	}
	return strings.TrimSpace(u.Name)
}

// Canonical slug/prefix for a middleware instance.
// (You can make this more strict: kebab-case, etc.)
func SectionSlug(instanceKey string) string   { return "mw-" + strings.ToLower(instanceKey) }
func SectionPrefix(instanceKey string) string { return "mw-" + strings.ToLower(instanceKey) + "-" }

type Definition interface {
	Name() string
	Description() string

	// Section for a particular instance. Instance key matters if multiple instances exist.
	Section(instanceKey string) (schema.Section, error)

	// Build using decoded values for this instance.
	Build(ctx context.Context, deps BuildDeps, parsed *values.Values, instanceKey string) (middleware.Middleware, error)
}

type Registry struct {
	defs map[string]Definition
}

func NewRegistry(defs ...Definition) *Registry {
	r := &Registry{defs: map[string]Definition{}}
	for _, d := range defs {
		r.Register(d)
	}
	return r
}

func (r *Registry) Register(d Definition) {
	r.defs[d.Name()] = d
}

func (r *Registry) SchemaForUses(uses []Use) (*schema.Schema, error) {
	s := schema.NewSchema()
	seen := map[string]struct{}{}
	for _, u := range uses {
		def, ok := r.defs[u.Name]
		if !ok {
			return nil, fmt.Errorf("unknown middleware: %s", u.Name)
		}
		key := u.InstanceKey()
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("duplicate middleware instance key: %s", key)
		}
		seen[key] = struct{}{}

		sec, err := def.Section(key)
		if err != nil {
			return nil, err
		}
		s.Set(sec.GetSlug(), sec)
	}
	return s, nil
}

func (r *Registry) BuildChain(ctx context.Context, deps BuildDeps, uses []Use, parsed *values.Values) ([]middleware.Middleware, error) {
	out := make([]middleware.Middleware, 0, len(uses))
	for _, u := range uses {
		def, ok := r.defs[u.Name]
		if !ok {
			return nil, fmt.Errorf("unknown middleware: %s", u.Name)
		}
		mw, err := def.Build(ctx, deps, parsed, u.InstanceKey())
		if err != nil {
			return nil, fmt.Errorf("middleware %s: %w", u.Name, err)
		}
		out = append(out, mw)
	}
	return out, nil
}
```

### A generic typed definition helper

```go
// pkg/inference/middlewarecfg/typed_definition.go
package middlewarecfg

import (
	"context"
	"fmt"

	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

type TypedDefinition[C any] struct {
	name        string
	description string

	// Builds the glazed section describing the config.
	section func(instanceKey string) (schema.Section, error)

	// Returns a default config value.
	defaults func() C

	// Builds the runtime middleware func.
	build func(ctx context.Context, deps BuildDeps, cfg C) (middleware.Middleware, error)
}

func (d *TypedDefinition[C]) Name() string        { return d.name }
func (d *TypedDefinition[C]) Description() string { return d.description }
func (d *TypedDefinition[C]) Section(instanceKey string) (schema.Section, error) {
	return d.section(instanceKey)
}

func (d *TypedDefinition[C]) Build(ctx context.Context, deps BuildDeps, parsed *values.Values, instanceKey string) (middleware.Middleware, error) {
	cfg := d.defaults()
	slug := SectionSlug(instanceKey)
	if err := parsed.DecodeSectionInto(slug, &cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	return d.build(ctx, deps, cfg)
}
```

### Example: systemPrompt middleware (typed)

```go
package builtins

import (
	"context"

	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/middlewarecfg"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
)

type SystemPromptConfig struct {
	Prompt string `glazed:"prompt" json:"prompt" yaml:"prompt"`
}

func SystemPrompt() middlewarecfg.Definition {
	return &middlewarecfg.TypedDefinition[SystemPromptConfig]{
		name:        "systemPrompt",
		description: "Ensure a fixed system block exists; replace or insert.",
		defaults: func() SystemPromptConfig {
			return SystemPromptConfig{Prompt: ""}
		},
		section: func(instanceKey string) (schema.Section, error) {
			return schema.NewSection(
				middlewarecfg.SectionSlug(instanceKey),
				"System Prompt",
				schema.WithPrefix(middlewarecfg.SectionPrefix(instanceKey)),
				schema.WithFields(
					fields.New("prompt", fields.TypeString,
						fields.WithHelp("System prompt to inject/replace"),
					),
				),
			)
		},
		build: func(ctx context.Context, deps middlewarecfg.BuildDeps, cfg SystemPromptConfig) (middleware.Middleware, error) {
			return middleware.NewSystemPromptMiddleware(cfg.Prompt), nil
		},
	}
}
```

### Example: turnLogging middleware (typed)

```go
type TurnLoggingConfig struct {
	Enabled bool `glazed:"enabled" json:"enabled" yaml:"enabled"`
	// Could add more toggles later, without changing the core architecture.
}

func TurnLogging() middlewarecfg.Definition {
	return &middlewarecfg.TypedDefinition[TurnLoggingConfig]{
		name:        "turnLogging",
		description: "Logs turn metadata before/after inference",
		defaults: func() TurnLoggingConfig {
			return TurnLoggingConfig{Enabled: true}
		},
		section: func(instanceKey string) (schema.Section, error) {
			return schema.NewSection(
				middlewarecfg.SectionSlug(instanceKey),
				"Turn Logging",
				schema.WithPrefix(middlewarecfg.SectionPrefix(instanceKey)),
				schema.WithFields(
					fields.New("enabled", fields.TypeBool,
						fields.WithDefault(true),
						fields.WithHelp("Enable this middleware"),
					),
				),
			)
		},
		build: func(ctx context.Context, deps middlewarecfg.BuildDeps, cfg TurnLoggingConfig) (middleware.Middleware, error) {
			if !cfg.Enabled {
				// No-op middleware (or return nil and have caller skip).
				return func(next middleware.HandlerFunc) middleware.HandlerFunc { return next }, nil
			}
			// deps.Logger would be used here; simplified:
			return middleware.NewTurnLoggingMiddleware(/* deps.Logger */ middlewarecfgNoLogger()), nil
		},
	}
}
```

(You’d likely make `BuildDeps` include `zerolog.Logger` and pass it in.)

### Layering: defaults → profile → env/flags → request overrides

This is where glazed shines.

Assume you have a profile runtime spec:

```go
uses := []middlewarecfg.Use{
	{Name: "systemPrompt", Config: map[string]any{"prompt": "You are helpful."}},
	{Name: "turnLogging", Config: map[string]any{"enabled": true}},
}
```

Convert the per-middleware `Config` blobs into a map keyed by section slug:

```go
func ProfileUsesToSectionMap(uses []middlewarecfg.Use) (map[string]map[string]any, error) {
	out := map[string]map[string]any{}
	for _, u := range uses {
		if u.Config == nil {
			continue
		}
		cfg, ok := u.Config.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s config must be map[string]any, got %T", u.Name, u.Config)
		}
		out[middlewarecfg.SectionSlug(u.InstanceKey())] = cfg
	}
	return out, nil
}
```

Now build schema + parse layered values:

```go
mwSchema, _ := reg.SchemaForUses(uses)
parsed := values.New()

profileMap, _ := ProfileUsesToSectionMap(uses)

// requestOverridesMap can be another map[string]map[string]any (same shape)
// built from HTTP body, UI state, etc.
requestOverridesMap := map[string]map[string]any{
	"mw-systemprompt": {"prompt": "You are EXTRA helpful."},
}

_ = sources.Execute(
	mwSchema,
	parsed,
	sources.FromCobra(cmd, fields.WithSource("cobra")),
	sources.FromEnv("PINOCCHIO", fields.WithSource("env")),
	sources.FromMap(requestOverridesMap, fields.WithSource("request")),
	sources.FromMap(profileMap, fields.WithSource("profile")),
	sources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
)
```

Now you can:

* decode + build middlewares
* inspect provenance logs per field (`parsed.GetField("mw-systemprompt", "prompt").Log`)

Finally build chain:

```go
mws, _ := reg.BuildChain(ctx, deps, uses, parsed)
// builder.WithMiddlewares(mws...) or middleware.Chain(base, mws...)
```

### Why this is the best “glazed-like” fit

✅ Each middleware is “like a mini command section”
✅ Schema is introspectable + serializable (glazed already marshals schema)
✅ Layering is uniform and provenance is already implemented
✅ Build-time dependencies are explicit (`BuildDeps`)
✅ Supports multiple instances (via `Use.ID`) with stable keys/prefixes
✅ Easy to expose in a UI: ship schema + current values + logs

---

## Proposal 3: Make “middleware instances” first-class, support refine/patch semantics

If “refine” means more than “override a few fields”, you’ll want a patch model that can do:

* enable/disable specific middleware
* reorder
* add/remove middlewares
* override config for a specific instance

### Extend `MiddlewareUse`

Right now you have:

```go
type MiddlewareUse struct {
	Name   string `json:"name" yaml:"name"`
	Config any    `json:"config,omitempty" yaml:"config,omitempty"`
}
```

Add stable instance identity + enabled:

```go
type MiddlewareUse struct {
	Name    string `json:"name" yaml:"name"`
	ID      string `json:"id,omitempty" yaml:"id,omitempty"`           // stable instance key
	Enabled *bool  `json:"enabled,omitempty" yaml:"enabled,omitempty"` // nil = default true
	Config  any    `json:"config,omitempty" yaml:"config,omitempty"`
}
```

### Patch format example

```yaml
middlewaresPatch:
  disable: ["turnLogging"]        # by ID (fallback to name)
  set:
    systemPrompt:
      prompt: "Overridden prompt"
  add:
    - name: "someOtherMw"
      id: "other1"
      config:
        foo: 123
  move:
    - id: "other1"
      before: "systemPrompt"
```

### Apply patches

Pseudocode:

```go
func ApplyMiddlewarePatch(base []MiddlewareUse, patch Patch) ([]MiddlewareUse, error) {
	// 1) index by ID (or name if no ID)
	// 2) disable/enable toggles
	// 3) apply config "set" as merge into existing config map
	// 4) add new uses
	// 5) reorder (move ops)
	// return new slice
}
```

Then feed resulting uses into Proposal 2’s registry+glazed pipeline for config resolution.

### Pros / cons

✅ Matches “profile chain is a template; runtime refines it”
✅ Lets UI safely offer reorder/add/remove operations
✅ IDs solve “multiple instances of same middleware type” cleanly
❌ More machinery (patch parsing, merge semantics, conflict resolution)

This is worth it if your “refine” requirement includes more than tweaking params.

---

## Proposal 4: JSON-Schema-first middleware contract (UI/JS/plugin friendly)

If you want middleware to be configurable not only via CLI but also via:

* JS plugins (you already have `MiddlewareFactory(options map[string]any)`)
* Web UI configuration forms
* API consumers

…then emitting **JSON Schema** is super useful.

Geppetto already uses `invopop/jsonschema` in-tree; you can make middleware configs “tool-like”.

### Shape

Each middleware definition provides:

* `ConfigJSONSchema() *jsonschema.Schema`
* `Build(cfg any) (middleware.Middleware, error)` (or typed)

And optionally also a glazed section derived from the struct schema.

#### Example

```go
type Definition interface {
	Name() string
	Description() string
	ConfigJSONSchema() *jsonschema.Schema
	BuildFromAny(ctx context.Context, deps Deps, cfg any) (middleware.Middleware, error)
}
```

Implementation uses a typed struct and reflect-based schema:

```go
type SystemPromptConfig struct {
	Prompt string `json:"prompt" jsonschema:"required,description=System prompt text"`
}

func (d *SystemPromptDef) ConfigJSONSchema() *jsonschema.Schema {
	r := jsonschema.Reflector{}
	s := r.Reflect(SystemPromptConfig{})
	return s
}

func (d *SystemPromptDef) BuildFromAny(ctx context.Context, deps Deps, cfg any) (middleware.Middleware, error) {
	// marshal/unmarshal cfg into SystemPromptConfig
	// validate required fields
	// return middleware.NewSystemPromptMiddleware(...)
}
```

Then:

* UI renders JSON schema forms
* JS engine can introspect schema and validate before sending options
* CLI can still be supported by converting JSON schema → glazed fields (you’d write a translator once)

### Pros / cons

✅ Best for cross-boundary interoperability (HTTP/JS/UI)
✅ You can validate configs generically
❌ You either lose glazed’s layered provenance, or you need to keep glazed alongside it
❌ Schema conversion JSONSchema ↔ glazed takes work (one-time, but non-trivial)

A strong hybrid is:

* **glazed schema** is the canonical internal config model (for layering/provenance)
* **jsonschema** is exported for UI/JS (generated from the same config struct)

---

## What I’d implement first (practical path)

If your main use cases are **profiles + CLI/env/config layering** and you want “glazed-like” behavior:

1. Implement **Proposal 2** (typed middleware definitions + registry + glazed layering).
2. Add `MiddlewareUse.ID` if you foresee multiple instances or patch operations (Proposal 3-lite).
3. Later, add JSON Schema export if you want UI/JS configuration to be rich (Proposal 4).

That gives you something that feels *exactly like* glazed command sections: each middleware is a section; the runtime schema is the union of sections for active middleware instances; values are layered and logged.

---

## Bonus: How this plugs into your existing profile runtime spec

You already have:

```go
type RuntimeSpec struct {
	SystemPrompt string
	Middlewares []profiles.MiddlewareUse // Name + Config
	Tools []string
}
```

Two integration styles:

### Style A: keep SystemPrompt separate (current behavior)

* `SystemPrompt` stays a top-level runtime value
* middleware chain includes “other middlewares”
* fine, but systemPrompt config isn’t part of the same layered system

### Style B: treat system prompt as just another configurable middleware

* profile sets it via middleware use:

  ```yaml
  middlewares:
    - name: systemPrompt
      config:
        prompt: "You are ..."
  ```
* runtime builder always includes it (or profile chain includes it)
* now *everything* about prompting techniques is consistent and introspectable

If the goal is “middleware as composable prompting techniques”, Style B is conceptually cleaner.

---

If you want, I can also sketch the exact changes to **pinocchio’s** `RuntimeComposer` (the one currently doing `mwFactories[u.Name](u.Config)`) to use Proposal 2 end-to-end (schema build → layered parse → typed decode → instantiate), but the core design above should already make the shape of that refactor obvious.
