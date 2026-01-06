---
Title: Debate Round 2 (Q4-6 key identity)
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
    - Path: pkg/turns/key_types.go
      Note: Current typed string key definitions (TurnDataKey, etc.)
    - Path: pkg/turns/keys.go
      Note: Canonical const keys in Geppetto (lines 47-50)
    - Path: moments/backend/pkg/turnkeys/data_keys.go
      Note: Application-specific keys with namespace prefix "mento.*"
    - Path: pkg/analysis/turnsdatalint/analyzer.go
      Note: Linter enforces typed-key expressions
ExternalSources: []
Summary: "Debate Round 2: Should keys be structured {vs,slug,version} or encoded strings? Where does versioning live? How do application-specific keys coexist?"
LastUpdated: 2025-12-20T01:15:00-05:00
WhatFor: Explore key identity modeling (structured vs encoded), versioning strategies, and multi-package key organization before committing to an API design.
WhenToUse: Reference when drafting RFC or reviewing proposals for key identity and versioning policy.
---

# Debate Round 2: Key Identity and Versioning (Q4-Q6)

## Participants

- **Asha "Strong Types"** (Type-Safety Maximalist)
- **Mina "Make the Linter a Boundary"** (Tooling / Lint Enforcer)
- **Priya "Go Specialist"** (Go generics + encoding expert)
- **Casey "Code Review"** (Maintainer / reviewer / bug-triage)
- **Jordan "Just Let Me Ship"** (Application Engineer / API consumer)

## Questions

4. **Should key identity be a structured type or an encoded string convention?**
5. **Where should versioning live?**
6. **Where do "application-specific keys" live, and how do we prevent drift?**

---

## Pre-Debate Research

### Research: Current key modeling

**Query:** What are the current key types?

From `geppetto/pkg/turns/key_types.go:3-13`:
```go
// TurnDataKey is a typed string key for Turn.Data map.
type TurnDataKey string

// TurnMetadataKey is a typed string key for Turn.Metadata map.
type TurnMetadataKey string

// BlockMetadataKey is a typed string key for Block.Metadata map.
type BlockMetadataKey string

// RunMetadataKey is a typed string key for Run.Metadata map.
type RunMetadataKey string
```

**Finding:** Keys are **typed strings**. No struct, no fields—just `type TurnDataKey string`.

---

**Query:** What do canonical keys look like in Geppetto?

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

**Finding:** Keys are **simple strings** like `"tool_config"`. No namespace prefix, no version suffix.

---

**Query:** How does Moments (application layer) define keys?

From `moments/backend/pkg/turnkeys/data_keys.go:6-46`:
```go
const (
	// PersonID is the person UUID for identity scope
	PersonID turns.TurnDataKey = "mento.person_id"
	// OrgID is the organization UUID for identity scope
	OrgID turns.TurnDataKey = "mento.org_id"
	// PromptSlugPrefix is the namespace prefix for prompt resolution
	PromptSlugPrefix turns.TurnDataKey = "mento.prompt_slug_prefix"
	// ...
	// TeamSuggestions is the team suggestions data
	TeamSuggestions turns.TurnDataKey = "mento.team.suggestions"
	// ...
	// PersonIDLegacy is legacy person_id key (for compatibility)
	PersonIDLegacy turns.TurnDataKey = "person_id"
	// PersonIDCamelCase is camelCase variant (for compatibility)
	PersonIDCamelCase turns.TurnDataKey = "personId"
)
```

**Finding:** Moments uses a **namespace prefix** (`"mento.*"`) and defines keys in a separate package (`moments/backend/pkg/turnkeys`). There are **legacy keys** for compatibility (`person_id`, `personId`).

---

**Query:** Is there any versioning in current keys?

```bash
$ cd geppetto && grep -r '@v\d\|_v\d\|version' pkg/turns/keys.go moments/backend/pkg/turnkeys/
```

**Finding:** **No versioning** in key strings. No `@v1`, `_v2`, or similar.

---

**Query:** What does the linter enforce?

From `geppetto/pkg/analysis/turnsdatalint/analyzer.go:32-39`:
```
// Analyzer enforces that Turn/Run/Block typed-key map access uses a typed key expression,
// and that Block.Payload (map[string]any) uses const string keys.
//
// For typed-key maps, this prevents raw string drift (e.g. t.Data["foo"]) while still allowing
// normal Go patterns like typed conversions, variables, and parameters.
```

**Finding:** The linter prevents **raw string indexing** (`t.Data["foo"]`) but doesn't enforce key naming conventions (namespace, versioning, etc.).

---

## Opening Statements

### Asha "Strong Types"

I looked at the current keys, and here's what I see: **no structure, no versioning, no enforcement beyond "don't use raw strings"**.

Look at `geppetto/pkg/turns/keys.go:47-50`:
```go
DataKeyToolConfig            TurnDataKey = "tool_config"
DataKeyAgentModeAllowedTools TurnDataKey = "agent_mode_allowed_tools"
```

These are just strings. There's nothing stopping someone from defining:
```go
DataKeyToolConfigV2 TurnDataKey = "tool_config_v2"
```

And now we have **two** keys for the same concept, with no way to know which is current.

**My answer to Q4:** Key identity should be a **structured type**:
```go
type TurnDataKeyID struct {
	vs      string  // namespace: "geppetto", "mento", etc.
	slug    string  // identifier: "tool_config", "person_id"
	version uint16  // version: 1, 2, 3, ...
}
```

This makes versioning **explicit** and **enforceable**. You can't create a key without all three parts.

**Q5:** Versioning should be **in the key identity** (`@v2`). When you need to change the shape of a value, you bump the version and define a new key. Old code keeps using `v1`, new code uses `v2`.

**Q6:** Application-specific keys should live in their own packages (like `moments/backend/pkg/turnkeys`), but they should use the **same structured type**. That way, all keys have a namespace, and we can enforce "no collisions" via linting or a central registry.

The current pattern (typed strings) is too loose. We need more structure.

---

### Mina "Make the Linter a Boundary"

Asha, I hear you, but I think you're overengineering. Let me show you what we **already have** that works:

1. **Typed keys** (`type TurnDataKey string`) prevent raw string drift—the linter catches `t.Data["foo"]`.
2. **Const definitions** (`DataKeyToolConfig`) give us a single source of truth.
3. **Namespace prefixes** (like `"mento.*"` in Moments) prevent collisions.

Look at `moments/backend/pkg/turnkeys/data_keys.go:8-10`:
```go
PersonID turns.TurnDataKey = "mento.person_id"
OrgID turns.TurnDataKey = "mento.org_id"
```

That's a **convention**, and it works. We don't need a struct to enforce it—we just need **linting**.

**My answer to Q4:** Keep keys as **encoded strings** with a **canonical format**:
- Format: `<namespace>.<slug>[@v<version>]`
- Example: `"geppetto.tool_config@v1"`, `"mento.person_id@v1"`

**Q5:** Versioning should be **optional** in the string encoding. If you don't specify a version, it's implicitly `v1`. When you need to evolve a key, you add `@v2` to the string.

**Q6:** Application-specific keys live in their own packages (like `turnkeys`), and we enforce the namespace convention via linting:
- New lint rule: "All `TurnDataKey` const definitions must match `^[a-z]+\.[a-z_]+(@v\d+)?$`".
- This prevents ad-hoc keys like `"oops"` or `"tool_config"` (no namespace).

The advantage of this approach: **no breaking changes**. We can evolve the linter incrementally, and existing keys keep working.

---

### Priya "Go Specialist"

Let me talk about what Go can and can't do, because this matters for both approaches.

**Asha's structured type:**
```go
type TurnDataKeyID struct {
	vs      string
	slug    string
	version uint16
}
```

This is **comparable** (can be a map key), and we can implement `encoding.TextMarshaler` / `encoding.TextUnmarshaler` to serialize it as `"vs/slug@vN"` for YAML. That's good.

But here's the catch: **Go doesn't support `const` for structs**. So canonical keys become **vars**:
```go
var KeyToolConfig = TurnDataKeyID{vs: "geppetto", slug: "tool_config", version: 1}
```

That means `turnsdatalint` needs to evolve from "enforce const keys" to "enforce canonical var keys". That's doable, but it's a change.

**Mina's encoded string:**
```go
type TurnDataKey string
const DataKeyToolConfig TurnDataKey = "geppetto.tool_config@v1"
```

This keeps `const`, which is nice. But now the linter has to **parse the string** to enforce the format. That's also doable (regex or split on `.` and `@`), but it's runtime validation, not compile-time.

**My answer to Q4:** I'm **neutral** on structured vs encoded. Both work. The trade-off is:
- **Structured**: compile-time enforcement, but no `const` (vars only).
- **Encoded**: `const` support, but runtime validation (linter parses strings).

**Q5:** Versioning should be **in the key identity**, whether that's a struct field or a string suffix. Don't do implicit versioning (renaming keys) because it's invisible and error-prone.

**Q6:** Application-specific keys should use the **same type** (whether `TurnDataKeyID` or `TurnDataKey`), and we should enforce namespace prefixes via linting. Moments already does this with `"mento.*"`.

One more thing: if we move to structured keys, we need to handle **YAML round-trips**. The key needs to serialize as a readable string (`"mento/person_id@v1"`), not a struct (`{vs: "mento", slug: "person_id", version: 1}`). That means implementing `MarshalText` / `UnmarshalText`, which is straightforward but adds complexity.

---

### Casey "Code Review"

I review a lot of PRs, and here's what I care about: **discoverability** and **preventing mistakes**.

Right now, if I see:
```go
t.Data[turnkeys.PersonID] = userID
```

I can jump to `turnkeys.PersonID` and see:
```go
PersonID turns.TurnDataKey = "mento.person_id"
```

That's good. But I have **no idea** what version this key is, or if there's a newer version I should be using.

**My answer to Q4:** I prefer **structured keys** (`TurnDataKeyID`), but I want them to be **self-documenting**:
```go
var KeyPersonID = TurnDataKeyID{
	vs:      "mento",
	slug:    "person_id",
	version: 1,
}
```

When I jump to definition, I see the namespace, slug, and version **immediately**. That's better than parsing a string.

**Q5:** Versioning should be **explicit** in the key identity. When someone bumps a version, they should define a new key:
```go
var KeyPersonIDV2 = TurnDataKeyID{vs: "mento", slug: "person_id", version: 2}
```

And we should have a **migration guide** that says: "If you're using `KeyPersonID`, migrate to `KeyPersonIDV2` by 2026-Q1."

**Q6:** Application-specific keys should live in their own packages, but we need a **central registry** (or at least a linting rule) to prevent collisions. For example:
- Geppetto keys use namespace `"geppetto"`.
- Moments keys use namespace `"mento"`.
- Pinocchio keys use namespace `"pinocchio"`.

We should enforce this via linting: "All keys in `geppetto/pkg/turns` must use namespace `geppetto`; all keys in `moments/backend/pkg/turnkeys` must use namespace `mento`."

One more thing: I want a **deprecation policy**. If we introduce `KeyPersonIDV2`, we should mark `KeyPersonID` as deprecated (with a comment or a linter warning), so new code doesn't accidentally use the old key.

---

### Jordan "Just Let Me Ship"

Okay, I'm the person who actually writes application code. Let me tell you what I care about:

**Q4:** I don't care if keys are structs or strings, as long as I can **copy/paste** them. Right now, I do:
```go
t.Data[turnkeys.PersonID] = userID
```

If we move to structured keys, I'll do:
```go
t.Data.Set(turnkeys.KeyPersonID, userID)
```

That's fine. But if the new pattern is:
```go
t.Data.Set(turns.NewKey("mento", "person_id", 1), userID)
```

That's **way** worse. I don't want to remember the namespace, slug, and version every time I use a key.

**Q5:** Versioning should be **invisible** to me unless I'm explicitly migrating. If I'm using `KeyPersonID`, and someone bumps it to `v2`, I should get a **linter warning** that says:
```
turnkeys.PersonID is deprecated; use turnkeys.PersonIDV2 instead
```

But I shouldn't have to think about versions when I'm just reading a config.

**Q6:** Application-specific keys should live in a **single package** per application. For Moments, that's `moments/backend/pkg/turnkeys`. I don't want to grep across 10 different packages to find where a key is defined.

Also, about Mina's "namespace prefix" idea: I like it, but I want it to be **automatic**. If I define a key in `moments/backend/pkg/turnkeys`, the namespace should be `"mento"` by default. I shouldn't have to type it every time.

---

## Rebuttals

### Asha responds to Mina

Mina, you said "we don't need a struct to enforce it—we just need linting." But linting **is** enforcement! You're proposing a lint rule that parses strings to check the format. That's runtime validation.

My structured type gives you **compile-time** enforcement. If you try to create a key without a namespace, the compiler stops you:
```go
var KeyOops = TurnDataKeyID{slug: "oops"} // compile error: vs is required
```

With your approach, the linter has to parse `"oops"` and say "this doesn't match the format". That's weaker.

Also, you said "no breaking changes". But if we add versioning to the string format (`@v1`), **all existing keys** need to be updated. That's a breaking change.

With structured keys, we can add versioning **gradually**:
1. Add the `version` field to `TurnDataKeyID` (default to `1` if unset).
2. Update keys one by one.
3. Eventually enforce "version must be set" via linting.

That's a **smoother migration**.

---

### Mina responds to Asha

Asha, you're right that linting is runtime validation. But here's the thing: **we already have a linter** (`turnsdatalint`), and it already works. Adding a regex check for key format is a **small change**.

Your structured type requires:
1. Changing `type TurnDataKey string` to `type TurnDataKeyID struct`.
2. Updating **every key definition** from `const` to `var`.
3. Updating **every call site** from `t.Data[key]` to `t.Data[key.id]` (or whatever the accessor is).
4. Implementing `MarshalText` / `UnmarshalText` for YAML.

That's a **big refactor**. My approach is incremental:
1. Add a lint rule for key format.
2. Update keys to include namespace (e.g., `"geppetto.tool_config"`).
3. Optionally add versioning later (`@v1`).

Each step is **independently useful**, and we don't break existing code.

Also, about your "compile-time enforcement" argument: Go's zero values mean you **can't** prevent `TurnDataKeyID{slug: "oops"}` at compile time. You still need runtime validation (in `Set/Get` or in the linter).

---

### Priya responds to Casey

Casey, you want a "central registry" to prevent namespace collisions. That's a good idea, but let's think about how it works in practice.

**Option 1: Linter-based registry**
- Each package declares its namespace in a comment or config.
- The linter checks that all keys in that package use the declared namespace.
- Example: `moments/backend/pkg/turnkeys` declares `namespace: mento`, and the linter enforces that all keys start with `"mento."`.

**Option 2: Code-based registry**
- A central file (e.g., `geppetto/pkg/turns/namespaces.go`) lists all valid namespaces.
- Keys reference the namespace via a const or var.
- Example:
  ```go
  const NamespaceGeppetto = "geppetto"
  const NamespaceMento = "mento"
  var KeyToolConfig = TurnDataKeyID{vs: NamespaceGeppetto, slug: "tool_config", version: 1}
  ```

I prefer **Option 1** (linter-based) because it's less invasive. We don't need a central file that every package imports.

About your deprecation policy: I agree, we should mark old keys as deprecated. But Go doesn't have a `@deprecated` annotation (yet). We can use comments:
```go
// Deprecated: use KeyPersonIDV2 instead.
const PersonID turns.TurnDataKey = "mento.person_id"
```

And we can add a linter rule to warn on usage of deprecated keys.

---

### Casey responds to Jordan

Jordan, you said "versioning should be invisible to me unless I'm explicitly migrating." I agree, but here's the problem: **how do you know when to migrate?**

If we have:
```go
const PersonID turns.TurnDataKey = "mento.person_id"
const PersonIDV2 turns.TurnDataKey = "mento.person_id@v2"
```

And you're using `PersonID`, how do you find out that `PersonIDV2` exists? You have to:
1. Grep for `person_id` in the codebase.
2. Notice that there's a `V2` variant.
3. Read the docs (if they exist) to understand the difference.

That's not great. I want the **linter** to tell you:
```
warning: turnkeys.PersonID is deprecated; use turnkeys.PersonIDV2 instead
```

But that requires the linter to know about deprecations. We can do that with comments (`// Deprecated: ...`), but it's not automatic.

About your "namespace should be automatic" idea: I like it, but it's tricky. If we infer the namespace from the package path (`moments/backend/pkg/turnkeys` → `"mento"`), we need a **mapping** somewhere. That's either:
- A config file (`turnkeys_namespaces.yaml`).
- A convention (`pkg/turnkeys` → use the parent directory name).

I prefer the **convention** approach, but it's not foolproof.

---

### Jordan responds to Mina

Mina, I like your incremental approach. But here's my worry: if we add namespace prefixes to all keys (`"geppetto.tool_config"`), **every call site** that currently does:
```go
t.Data[turns.DataKeyToolConfig]
```

...keeps working, because the key is still a `const`. That's good.

But if we later want to add versioning (`@v1`), we have to **update the key strings again**:
```go
const DataKeyToolConfig TurnDataKey = "geppetto.tool_config@v1"
```

That's **another** refactor. And if someone forgets to add `@v1` to a key, we have inconsistency.

With Asha's structured type, versioning is **required** from day one. You can't create a key without specifying the version. That's stricter, but it's also **clearer**.

I'm starting to think we should do **both**:
1. Short-term: add namespace prefixes to string keys (Mina's approach).
2. Long-term: migrate to structured keys (Asha's approach).

That way, we get the benefits of both: incremental migration **and** long-term structure.

---

## Moderator Summary

### Key Tensions

1. **Structured vs Encoded Keys**
   - **Asha**: Structured type (`TurnDataKeyID{vs, slug, version}`) for compile-time enforcement.
   - **Mina**: Encoded string (`"namespace.slug@vN"`) for incremental migration and `const` support.
   - **Priya**: Both work; trade-off is compile-time enforcement (struct) vs `const` support (string).
   - **Tension**: Structured keys are stricter but require a big refactor (no `const`, need `MarshalText`). Encoded strings are incremental but rely on runtime validation (linter).

2. **Versioning: Explicit vs Implicit**
   - **Consensus**: Versioning should be **in the key identity** (not implicit renaming).
   - **Asha**: Version is a struct field (`version: 1`).
   - **Mina**: Version is a string suffix (`@v1`), optional (defaults to `v1` if omitted).
   - **Casey**: Version should be explicit and visible in key definitions.
   - **Jordan**: Versioning should be invisible unless migrating; linter should warn on deprecated keys.

3. **Application-Specific Keys: Where and How?**
   - **Consensus**: Application-specific keys live in their own packages (like `moments/backend/pkg/turnkeys`).
   - **Casey**: Need a central registry or linting rule to prevent namespace collisions.
   - **Mina**: Enforce namespace prefixes via linting (e.g., all Moments keys start with `"mento."`).
   - **Jordan**: Namespace should be automatic (inferred from package path or declared once per package).

4. **Migration Path**
   - **Mina**: Incremental (add namespace prefixes, then versioning, then maybe structured keys).
   - **Asha**: Big refactor (move to structured keys, enforce versioning from day one).
   - **Jordan**: Do both (short-term incremental, long-term structured).

### Interesting Ideas

- **Linter-based namespace registry** (Priya, Casey): Each package declares its namespace, and the linter enforces that all keys use it.
- **Deprecation via comments** (Casey, Priya): Mark old keys with `// Deprecated: use KeyXV2 instead`, and add a linter rule to warn on usage.
- **Automatic namespace inference** (Jordan): Infer namespace from package path (`moments/backend/pkg/turnkeys` → `"mento"`).
- **Hybrid approach** (Jordan): Start with encoded strings (incremental), migrate to structured keys (long-term).
- **Optional versioning** (Mina): Version defaults to `v1` if omitted, so existing keys don't need immediate updates.

### Open Questions

1. **Should we enforce versioning from day one, or make it optional?**
   - Asha: Enforce (version is required in struct).
   - Mina: Optional (defaults to `v1` if omitted in string).

2. **How do we handle YAML serialization for structured keys?**
   - Priya: Implement `MarshalText` / `UnmarshalText` to serialize as `"vs/slug@vN"`.
   - Question: What's the canonical encoding? `"vs/slug@vN"` or `"vs.slug@vN"` or `"vs:slug@vN"`?

3. **How do we prevent namespace collisions across packages?**
   - Casey: Central registry (config file or code).
   - Priya: Linter-based (each package declares namespace, linter enforces).
   - Jordan: Convention (infer from package path).

4. **What's the migration path for existing keys?**
   - If we add namespace prefixes, do we update all keys at once, or gradually?
   - If we move to structured keys, do we provide a compatibility shim (e.g., `TurnDataKey` becomes an alias for `TurnDataKeyID.String()`)?

5. **Should we support "legacy keys" for compatibility?**
   - Moments has `PersonIDLegacy turns.TurnDataKey = "person_id"` for old code.
   - Should we formalize this pattern, or discourage it?

### Next Steps

- **Debate Round 3**: Opaque wrappers vs helper functions (Q7-Q9).
- **Prototype**: Build a small proof-of-concept with both approaches:
  1. Encoded strings with namespace prefixes (`"geppetto.tool_config@v1"`).
  2. Structured keys (`TurnDataKeyID{vs, slug, version}`) with `MarshalText`.
- **RFC**: Draft a proposal based on these findings, including migration path.

---

## Related

- Candidates: `reference/01-debate-candidates-typed-turn-data-metadata.md`
- Questions: `reference/02-debate-questions-typed-turn-data-metadata.md`
- Debate Round 1: `reference/08-debate-round-1-q1-3-typed-accessors.md`
- Ticket analysis: `analysis/01-opaque-turn-data-typed-get-t-accessors.md`
