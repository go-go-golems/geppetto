---
Title: Debate Round 5 (Q13-14 tooling+schema)
Ticket: 001-TYPED-ACCESSOR-TURN-DATA
Status: active
Topics:
    - geppetto
    - turns
    - go
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/analysis/turnsdatalint/analyzer.go
      Note: Current linter implementation (lines 32-45, 74-147)
    - Path: pkg/turns/keys.go
      Note: Canonical key definitions (const-based today)
    - Path: moments/backend/pkg/turnkeys/data_keys.go
      Note: Application-specific keys with namespace prefix
ExternalSources: []
Summary: "Debate Round 5: How should turnsdatalint evolve for new key models? Do we want a schema registry for key→type mappings?"
LastUpdated: 2025-12-20T02:45:00-05:00
WhatFor: Explore linter evolution strategies and schema registry trade-offs before committing to a tooling approach.
WhenToUse: Reference when drafting RFC or reviewing proposals for turnsdatalint enhancements and key validation strategies.
---

# Debate Round 5: Linter Evolution and Schema Registries (Q13-Q14)

## Participants

- **Mina "Make the Linter a Boundary"** (Tooling / Lint Enforcer)
- **Asha "Strong Types"** (Type-Safety Maximalist)
- **Priya "Go Specialist"** (Go generics + encoding expert)
- **Sam "Small Surface Area"** (API Minimalist)
- **Casey "Code Review"** (Maintainer / reviewer / bug-triage)

## Questions

13. **How should `turnsdatalint` evolve if we change key modeling?**
14. **Do we want a schema registry for keys?**

---

## Pre-Debate Research

### Research: Current linter capabilities

**Query:** What does `turnsdatalint` enforce today?

From `geppetto/pkg/analysis/turnsdatalint/analyzer.go:32-42`:
```go
// Analyzer enforces that Turn/Run/Block typed-key map access uses a typed key expression,
// and that Block.Payload (map[string]any) uses const string keys.
//
// For typed-key maps, this prevents raw string drift (e.g. t.Data["foo"]) while still allowing
// normal Go patterns like typed conversions, variables, and parameters.
//
// Note: raw string literals like turn.Data["foo"] can compile in Go because untyped string
// constants may be implicitly converted to a defined string type; this analyzer flags them too.
var Analyzer = &analysis.Analyzer{
	Name:     "turnsdatalint",
	Doc:      "require typed-key map indexes to use typed key expressions...",
```

**Finding:** The linter enforces:
- **Typed-key expressions** for `Data`/`Metadata` maps (no raw strings).
- **Const string keys** for `Block.Payload`.

It **does not** enforce:
- Key naming conventions (namespace, versioning).
- Canonical key definitions (keys must be defined in specific packages).
- Type correctness of values (e.g., "this key should store `engine.ToolConfig`").

---

**Query:** Where are canonical keys defined?

```bash
$ cd geppetto && wc -l pkg/turns/keys.go moments/backend/pkg/turnkeys/data_keys.go
  52 pkg/turns/keys.go
  52 moments/backend/pkg/turnkeys/data_keys.go
```

From `geppetto/pkg/turns/keys.go:45-51`:
```go
// Standard keys for Turn.Data map
const (
	DataKeyToolConfig            TurnDataKey = "tool_config"
	DataKeyAgentModeAllowedTools TurnDataKey = "agent_mode_allowed_tools"
	DataKeyAgentMode             TurnDataKey = "agent_mode"
	DataKeyResponsesServerTools  TurnDataKey = "responses_server_tools"
)
```

From `moments/backend/pkg/turnkeys/data_keys.go:6-10`:
```go
const (
	// PersonID is the person UUID for identity scope
	PersonID turns.TurnDataKey = "mento.person_id"
	// OrgID is the organization UUID for identity scope
	OrgID turns.TurnDataKey = "mento.org_id"
	// ...
)
```

**Finding:** Geppetto defines **4 keys** in `pkg/turns/keys.go`. Moments defines **20+ keys** in `pkg/turnkeys/`. Keys are **const** (not var), and there's **no central registry** mapping keys to expected types.

---

**Query:** How would we know what type a key expects?

**Current approach:** Read the code that uses the key.

Example: To find out what type `DataKeyToolConfig` expects, grep for usage:
```bash
$ cd geppetto && grep -n 'DataKeyToolConfig' pkg/inference/toolhelpers/helpers.go
304:	t.Data[turns.DataKeyToolConfig] = engine.ToolConfig{
```

**Finding:** You have to **grep** to discover the expected type. There's no single source of truth.

---

## Opening Statements

### Mina "Make the Linter a Boundary"

I looked at the current linter, and here's what it does well: **prevents raw string drift**.

From `geppetto/pkg/analysis/turnsdatalint/analyzer.go:35-36`:
```
// For typed-key maps, this prevents raw string drift (e.g. t.Data["foo"]) while still allowing
// normal Go patterns like typed conversions, variables, and parameters.
```

That's good. But here's what it **doesn't** do:
- Prevent ad-hoc key construction (`turns.TurnDataKey("oops")`).
- Enforce naming conventions (namespace, versioning).
- Enforce canonical key definitions (keys must be defined in `*/keys.go` or `*/turnkeys/*.go`).

**My answer to Q13:** `turnsdatalint` should evolve to enforce:

1. **Canonical keys only**: Ban ad-hoc key construction outside keys packages.
   - Rule: "`TurnDataKey(...)` conversions are only allowed in `*/keys.go` or `*/turnkeys/*.go` files."
   - This prevents scattered key definitions.

2. **Naming convention**: Enforce `"namespace.slug[@vN]"` format.
   - Rule: "All `TurnDataKey` const values must match `^[a-z]+\.[a-z_]+(@v\d+)?$`."
   - This ensures consistency (namespace + slug + optional version).

3. **Deprecation warnings**: Warn on usage of deprecated keys.
   - Rule: "If a key has a `// Deprecated: ...` comment, warn at usage sites."
   - This helps migrate away from old keys.

**Q14:** Do we want a schema registry? **No**. Here's why:

A schema registry would be a central file like:
```go
var KeyRegistry = map[TurnDataKey]reflect.Type{
    DataKeyToolConfig: reflect.TypeOf(engine.ToolConfig{}),
    // ...
}
```

But this has problems:
- **Maintenance burden**: Every time you add a key, you have to update the registry.
- **Import cycles**: The registry needs to import all value types (e.g., `engine.ToolConfig`), which creates dependencies.
- **Doesn't compose with generics**: Typed keys `Key[T]` already carry type information; a registry is redundant.

Instead, the **linter** is the registry. It enforces that keys are canonical, and the canonical key definitions (with comments) serve as documentation.

---

### Asha "Strong Types"

Mina, I agree with your linter evolution plan, but I want to go **further**.

If we introduce typed keys `Key[T]`, the linter should enforce:
- **Only canonical typed keys**: Ban `Key[T]{...}` construction outside keys packages.
- **Type consistency**: If a key is defined as `Key[engine.ToolConfig]`, the linter should warn if you try to `Set` a different type.

Wait, actually, the **compiler** already does that. If you define:
```go
var KeyToolConfig = Key[engine.ToolConfig]{id: ...}
```

And you try to do:
```go
t.Data.Set(turns.KeyToolConfig, "oops")  // compile error: wrong type
```

The compiler stops you. So we don't need a schema registry—**the type system is the registry**.

**My answer to Q13:** `turnsdatalint` should evolve to:
1. Ban ad-hoc `Key[T]{...}` construction (only allow canonical keys).
2. Enforce naming conventions (namespace, versioning) on the **key ID** (whether that's a string or a struct).

**Q14:** Do we want a schema registry? **No**, because **typed keys are the schema registry**. The key definition:
```go
var KeyToolConfig = Key[engine.ToolConfig]{id: MustDataKeyID("geppetto", "tool_config", 1)}
```

...tells you:
- **Key identity**: `"geppetto/tool_config@v1"` (from the ID).
- **Expected type**: `engine.ToolConfig` (from the `Key[T]` type parameter).

That's all the information a schema registry would provide, but it's **compile-time** instead of runtime.

---

### Priya "Go Specialist"

Let me talk about what's **feasible** for a linter to enforce.

**Q13:** Here's what a linter **can** do:
- Ban raw string indexing (`t.Data["foo"]`) — **already done**.
- Ban ad-hoc key construction (`TurnDataKey("oops")`) — **feasible** (check if the conversion appears in a keys file).
- Enforce naming conventions (`"namespace.slug@vN"`) — **feasible** (regex on const string values).
- Warn on deprecated keys — **feasible** (parse `// Deprecated:` comments).

Here's what a linter **cannot** do (easily):
- Enforce type consistency across `Set` and `Get` calls — requires **whole-program analysis** (expensive).
- Enforce that keys are defined in canonical packages — requires **package-level metadata** (complex).

So if we want strong enforcement, we need **typed keys** `Key[T]` (compiler does the work) **plus** linting (for naming conventions and deprecation).

**Q14:** Do we want a schema registry? **It depends on what problem we're solving.**

If the problem is: "I want to know what type a key expects," then **typed keys** `Key[T]` solve that. Jump to definition, see the type.

If the problem is: "I want to validate that all keys follow a naming convention," then **linting** solves that. No registry needed.

If the problem is: "I want to enforce that only certain packages can define keys," then we need **package-level linting** or a **build-time check** (e.g., a script that scans all key definitions and checks for collisions).

A runtime schema registry (like Mina described) is **not useful** for any of these problems. It's just extra maintenance.

---

### Sam "Small Surface Area"

I'm going to be the contrarian here.

**Q13:** I think we should **not** evolve `turnsdatalint` much. Here's why:

The current linter does **one thing well**: prevents raw string indexing. That's a clear, simple rule.

If we add more rules (naming conventions, canonical definitions, deprecation warnings), the linter becomes **complex**. And complexity means:
- Slower linting (more checks).
- More false positives (edge cases the linter doesn't handle).
- More maintenance (every time we change key modeling, we have to update the linter).

Instead, I think we should rely on **code review** and **documentation**:
- Document the naming convention (`"namespace.slug@vN"`) in `pkg/turns/keys.go`.
- Document the deprecation policy ("mark old keys with `// Deprecated: ...`").
- Trust developers to follow the convention.

**Q14:** Do we want a schema registry? **Absolutely not**. Here's why:

A schema registry is a **second source of truth**. You have:
1. The key definition (`var KeyToolConfig = ...`).
2. The registry entry (`KeyRegistry[KeyToolConfig] = ...`).

Now you have to keep them in sync. That's **double maintenance**.

If we use typed keys `Key[T]`, the key definition **is** the schema. No registry needed.

---

### Casey "Code Review"

I review a lot of PRs, and here's what I care about: **discoverability** and **preventing mistakes**.

**Q13:** I want `turnsdatalint` to evolve to enforce:

1. **Canonical keys only**: Ban ad-hoc key construction.
   - This prevents scattered key definitions.

2. **Naming convention**: Enforce `"namespace.slug[@vN]"` format.
   - This ensures consistency across packages.

3. **Deprecation warnings**: Warn on usage of deprecated keys.
   - This helps migrate away from old keys.

But I **don't** want the linter to be too strict. For example, if someone defines a test key locally:
```go
const MyTestKey turns.TurnDataKey = "test.my_feature"
```

...the linter should **warn** (not block) if it doesn't follow the convention. That way, developers can iterate quickly in tests without being blocked by linting.

**Q14:** Do we want a schema registry? **Yes, but not a runtime registry**.

Here's what I want: a **build-time registry** (a generated file or a script) that lists all keys and their expected types. Something like:
```go
// Generated by go generate
var AllKeys = []KeyInfo{
    {ID: "geppetto.tool_config@v1", Type: "engine.ToolConfig", Package: "geppetto/pkg/turns"},
    {ID: "mento.person_id@v1", Type: "uuid.UUID", Package: "moments/backend/pkg/turnkeys"},
    // ...
}
```

This serves as:
- **Documentation**: "What keys exist? What types do they expect?"
- **Collision detection**: "Are there duplicate keys across packages?"
- **Deprecation tracking**: "Which keys are deprecated?"

But it's **not** a runtime registry. It's a build artifact that we can use for linting, docs generation, and code review.

---

## Rebuttals

### Mina responds to Sam

Sam, you said "rely on code review and documentation." But here's the problem: **code review doesn't scale**.

Right now, if someone adds a new key:
```go
const MyKey turns.TurnDataKey = "my_key"
```

...I have to **manually check**:
- Does it follow the naming convention? (No, missing namespace.)
- Is it defined in the right place? (Maybe, depends on the file.)
- Does it collide with existing keys? (I have to grep to find out.)

That's **3 checks** I have to do for every new key. If the linter does it automatically, I save time and catch mistakes earlier.

Also, you said "complexity means slower linting." But `turnsdatalint` is already fast (it's a `go/analysis` analyzer). Adding a regex check for naming conventions is **trivial**—it won't slow down the linter noticeably.

---

### Asha responds to Priya

Priya, you said "a linter cannot enforce type consistency across `Set` and `Get` calls." But that's exactly why we need **typed keys** `Key[T]`.

With typed keys, the **compiler** enforces type consistency:
```go
var KeyToolConfig = Key[engine.ToolConfig]{...}

t.Data.Set(turns.KeyToolConfig, cfg)  // cfg must be engine.ToolConfig
val, ok, err := t.Data.Get(turns.KeyToolConfig)  // val is engine.ToolConfig
```

No linter needed. The type system does the work.

So the linter's job becomes **simpler**: just enforce that keys are canonical (defined in keys packages) and follow naming conventions. The type system handles the rest.

---

### Priya responds to Casey

Casey, you want a "build-time registry" (a generated file listing all keys). That's an interesting idea, but here's the challenge: **how do you generate it?**

You'd need a tool that:
1. Scans all Go files for `TurnDataKey` const definitions.
2. Extracts the key string and the const name.
3. Infers the expected type (by grepping for usage? by parsing type annotations?).
4. Generates a file with all keys.

That's **complex**. And it has to run on every build (or via `go generate`), which adds friction.

Here's a simpler approach: **use the linter as the registry**. The linter already scans all code. It can:
- Collect all key definitions.
- Check for collisions.
- Warn on deprecated keys.
- Output a report (JSON or text) listing all keys.

That way, the registry is **implicit** (derived from code) instead of **explicit** (a generated file you have to maintain).

---

### Sam responds to Mina

Mina, you said "code review doesn't scale." But here's my counter: **linting doesn't scale either**.

Every time we add a new rule to the linter, we add:
- **Complexity**: More code to maintain in the linter.
- **False positives**: Edge cases the linter doesn't handle.
- **Friction**: Developers blocked by linter errors they don't understand.

I've seen this happen in other projects. The linter starts simple ("ban raw strings"), then grows ("enforce naming conventions"), then grows more ("ban ad-hoc keys"), and eventually it's a **monster** that everyone hates.

I'd rather have **simple linting** (ban raw strings) and **good documentation** (naming conventions, deprecation policy) than a complex linter that tries to enforce everything.

---

### Casey responds to Sam

Sam, you're worried about linter complexity. I get that. But here's the thing: **the complexity is already there**—it's just hidden in code review.

Right now, when I review a PR that adds a new key, I have to:
1. Check the naming convention manually.
2. Check for collisions manually (grep).
3. Check that the key is defined in the right place manually.

That's **complexity**. If the linter does it automatically, the complexity is **centralized** (in the linter) instead of **distributed** (across every code review).

And about false positives: we can make the linter **configurable**. For example:
- Strict mode (CI): block on all violations.
- Permissive mode (local dev): warn on violations, don't block.

That way, developers can iterate quickly, but CI catches mistakes before merge.

---

## Moderator Summary

### Key Tensions

1. **Linter Evolution: Simple vs Comprehensive**
   - **Mina, Casey**: Evolve linter to enforce naming conventions, canonical keys, deprecation.
   - **Sam**: Keep linter simple (ban raw strings only); rely on code review and docs.
   - **Tension**: Comprehensive linting catches mistakes early but adds complexity. Simple linting is maintainable but relies on humans.

2. **Schema Registry: Runtime vs Build-Time vs None**
   - **Asha**: No registry needed (typed keys `Key[T]` are the schema).
   - **Casey**: Build-time registry (generated file listing all keys).
   - **Priya**: Linter as implicit registry (scan code, output report).
   - **Sam**: No registry (documentation is enough).
   - **Tension**: Registry provides discoverability but adds maintenance. Typed keys provide type info but not a "list of all keys".

3. **Type System vs Linting**
   - **Asha**: Type system enforces type consistency (typed keys `Key[T]`).
   - **Mina**: Linter enforces naming conventions and canonical definitions.
   - **Priya**: Both are needed (type system for types, linter for conventions).
   - **Tension**: Type system is compile-time but limited. Linter is flexible but runtime.

4. **Enforcement Strictness**
   - **Casey**: Configurable (strict in CI, permissive in local dev).
   - **Sam**: Minimal enforcement (trust developers).
   - **Mina**: Strong enforcement (catch mistakes early).

### Interesting Ideas

- **Linter enforces canonical keys** (Mina, Casey): Ban ad-hoc key construction outside keys packages.
- **Naming convention enforcement** (Mina, Casey): Regex check for `"namespace.slug[@vN]"` format.
- **Deprecation warnings** (Mina, Casey): Parse `// Deprecated:` comments and warn at usage sites.
- **Build-time registry** (Casey): Generated file listing all keys (for docs/collision detection).
- **Linter as implicit registry** (Priya): Linter scans code and outputs report (no generated file).
- **Configurable strictness** (Casey): Strict mode (CI) vs permissive mode (local dev).
- **Typed keys are the schema** (Asha): `Key[T]` carries type information; no separate registry needed.

### Open Questions

1. **Should we enforce naming conventions via linting or documentation?**
   - Mina/Casey: Linting (catch mistakes early).
   - Sam: Documentation (trust developers).

2. **Should we ban ad-hoc key construction?**
   - Mina/Casey/Asha: Yes (via linting or type system).
   - Sam: No (too strict).

3. **Do we need a build-time registry?**
   - Casey: Yes (for discoverability and collision detection).
   - Asha: No (typed keys are enough).
   - Priya: Maybe (linter can generate a report).
   - Sam: No (documentation is enough).

4. **How do we handle test keys?**
   - Casey: Allow `"test.*"` keys in test files only.
   - Sam: Don't enforce (trust developers).

5. **Should the linter be configurable (strict vs permissive)?**
   - Casey: Yes (strict in CI, permissive in local dev).
   - Mina: No (always strict).

### Consensus Points

- **Current linter is good at what it does**: Prevents raw string drift.
- **Typed keys `Key[T]` provide type information**: No separate registry needed for type consistency.
- **Linter should evolve if we add structure**: If we move to structured keys or add versioning, linter needs to adapt.
- **Documentation is important**: Even with strong linting, we need clear docs on naming conventions and deprecation policy.

### Next Steps

- **Prototype linter enhancements**: Add rules for naming conventions and canonical keys to `turnsdatalint`.
- **Prototype build-time registry**: Generate a report of all keys (for discoverability).
- **RFC**: Draft a proposal for linter evolution and schema registry approach.
- **Synthesis document**: Combine findings from all 5 debate rounds into a coherent design recommendation.

---

## Related

- Candidates: `reference/01-debate-candidates-typed-turn-data-metadata.md`
- Questions: `reference/02-debate-questions-typed-turn-data-metadata.md`
- Debate Round 1: `reference/08-debate-round-1-q1-3-typed-accessors.md`
- Debate Round 2: `reference/09-debate-round-2-q4-6-key-identity.md`
- Debate Round 3: `reference/10-debate-round-3-q7-9-api-surface.md`
- Debate Round 4: `reference/11-debate-round-4-q10-q12-serializability-failures.md`
- Ticket analysis: `analysis/01-opaque-turn-data-typed-get-t-accessors.md`
