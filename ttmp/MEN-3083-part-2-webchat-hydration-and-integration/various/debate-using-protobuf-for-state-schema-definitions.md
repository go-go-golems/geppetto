---
Title: 'Debate: Using Protobuf for State Schema Definitions'
Ticket: MEN-3083-part-2
Status: active
Topics:
    - frontend
    - conversation
    - events
DocType: misc
Intent: ""
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2025-11-04T13:19:29.136187886-05:00
---





# Presidential Debate: Protobuf for Widget State Schema Definitions

**Date**: 2025-11-04  
**Format**: Presidential Debate Style  
**Topic**: Should we use Protobuf for defining widget state schemas stored in the database?

## Candidates

1. **Dr. Sarah "Proto-First" Chen** - Senior Backend Architect (Pro-Protobuf)
2. **Jake "JSON-Native" Morrison** - Lead Frontend Engineer (Pro-JSON with types)
3. **The Existing Codebase** - Represented by `pinocchio/pkg/snapshots/codec.go` (Current JSON approach)
4. **Dr. Maya "Schema-Evolution" Rodriguez** - Database/Systems Architect (Pragmatic middle ground)
5. **Alex "Developer-Experience" Kim** - DevOps/Tooling Lead (Operational concerns)

## Debate Questions

### Question 1: Schema Evolution and Backward Compatibility
**Moderator**: "Our widget state will evolve over time. How should we handle schema changes when widgets gain new fields or types change?"

### Question 2: Developer Experience and Debugging
**Moderator**: "Developers need to debug persisted state in production. What's the best approach for inspectability and troubleshooting?"

### Question 3: Type Safety Across the Stack
**Moderator**: "We have Go backend, TypeScript frontend, and SQLite storage. How do we maintain type safety across all three?"

### Question 4: Performance and Storage Efficiency
**Moderator**: "These snapshots will be written frequently and stored long-term. What about performance and storage costs?"

### Question 5: Migration Path and Risk
**Moderator**: "We already have a working JSON-based system. How risky is a migration, and what's the ROI?"

---

## Round 1: Opening Statements (2 minutes each)

### Dr. Sarah "Proto-First" Chen

Look, we're already using Protobuf successfully for SEM events in go-go-mento! We have `proto/sem/domain/team_analysis.proto`, we have the codegen working with `make protobuf`, and both Go and TypeScript developers are comfortable with it. The pattern is *proven*.

```proto
message TeamAnalysisResult {
  string analysis_id = 1;
  double network_score = 2;
  repeated string insights = 3;
  // Schema evolution is built-in!
}
```

Protobuf gives us:
1. **Explicit versioning** - Field numbers are stable forever
2. **Forward/backward compatibility** - New fields don't break old readers
3. **Type safety** - Generated code in both Go and TS
4. **Smaller payloads** - Binary encoding when we need it (optional!)
5. **Schema documentation** - The `.proto` file IS the spec

And here's the key: we already convert proto → JSON for SEM events! Same pattern for snapshots. No binary wire format needed—just use `protojson` to serialize. You get schema enforcement AND human-readable storage!

### Jake "JSON-Native" Morrison

Whoa, Sarah, slow down! You're conflating *events* with *persistent state*. Events are ephemeral—fire and forget. Snapshots live in a database for months, get queried, debugged, and migrated.

Let me show you what we have RIGHT NOW that works perfectly:

```go
// pinocchio/pkg/snapshots/codec.go
func UnmarshalSnapshot(b []byte) (Snapshot, error) {
    var probe struct { Kind SnapshotKind `json:"kind"` }
    json.Unmarshal(b, &probe)
    switch probe.Kind {
        case KindLLMText: return &LLMTextSnapshot{}, nil
        case KindToolCall: return &ToolCallSnapshot{}, nil
    }
}
```

This is:
- **Crystal clear** - Any developer can read it
- **Debuggable** - `sqlite3 snapshots.db "SELECT snapshot_json FROM snapshots"` shows actual data
- **Flexible** - Add a field, deploy, old data still works
- **No build step** - No codegen, no proto compiler issues

And TypeScript? We already have typed interfaces in `timeline/types.ts`. They work! Adding Protobuf is *additional* complexity for dubious benefit.

### The Existing Codebase (pinocchio/pkg/snapshots/codec.go)

```go
// I am 64 lines of straightforward Go code.
// I have zero dependencies beyond encoding/json.
// I work. I'm tested. I'm deployed.
// 
// Look at me:
type LLMTextSnapshot struct {
    SnapshotBase
    Role      string `json:"role"`
    Text      string `json:"text"`
    Streaming bool   `json:"streaming"`
}

// See that? Go struct tags. TypeScript gets the same shape.
// We've persisted thousands of snapshots already.
// Why fix what isn't broken?
```

I exist in the real world. I handle `conv_id` overrides, I marshal flags, I query by conversation. The `UnmarshalSnapshot` discriminator pattern works *perfectly* for our 6 snapshot kinds. 

You want to add Protobuf? Fine—but you'll still need me for the SQLite layer. You'll have proto → JSON → my structs → SQLite. More steps, more bugs.

### Dr. Maya "Schema-Evolution" Rodriguez

Everyone here is missing the actual problem! Let me frame this correctly.

**The real question isn't Protobuf vs JSON—it's whether we have a *schema evolution strategy*.**

Look at what happens in 6 months:
- `ToolCallSnapshot` needs a `retryCount` field
- Frontend expects it, backend adds it
- Old snapshots in the DB don't have it
- Does the frontend crash? Do we migrate? Do we default?

Right now, we have *ad-hoc* evolution:
```go
// What we do today:
type ToolCallSnapshot struct {
    Name  string `json:"name"`
    Input json.RawMessage `json:"input,omitempty"`  // omitempty is our "optional"
    Exec  bool `json:"exec,omitempty"`
}
```

That's... fine for MVP. But when we have 10 widget types each with 8 fields, and we're 2 years in? That `omitempty` tag is doing *a lot* of semantic work without documentation.

Protobuf's advantage isn't the wire format—it's **explicit optional semantics** and **field deprecation paths**. But we could get similar guarantees with JSON Schema + validation! Or TypeScript Zod schemas shared via WASM!

### Alex "Developer-Experience" Kim

Okay, I'm the one who gets paged when prod breaks, so let me inject some reality.

**Protobuf adds operational complexity:**

1. **Build-time dependency** - `make protobuf` needs to work in CI, in dev containers, in 6 months when Buf releases a breaking change
2. **Dual source of truth** - We have `.proto` files AND Go structs AND TypeScript interfaces. Which is canonical?
3. **Debugging hell** - Try explaining to a junior dev why they need to regenerate code because they edited a `.proto` file

**But JSON has its own issues:**

1. **Drift** - Go and TypeScript snapshots diverge silently
2. **No validation** - Bad data goes into SQLite, crashes frontend months later
3. **No evolution plan** - We're one `ALTER TABLE` away from chaos

Here's what I actually care about: **Can a dev at 3am understand what went wrong?**

With JSON: `SELECT snapshot_json FROM snapshots WHERE entity_id='...'` → copy/paste into their editor → immediate understanding.

With Protobuf-JSON: Same! If we use `protojson` encoding.

With binary Protobuf: Hell no. You need the schema, the decoder, the right version...

So if we go Proto, we MUST keep JSON encoding. Which makes me ask: what's the point?

---

## Round 2: Direct Responses

### Question 1: Schema Evolution and Backward Compatibility

**Dr. Chen (Pro-Protobuf):**

Let me show you EXACTLY how Protobuf handles evolution. Here's our Team Analysis:

```proto
message TeamAnalysisResult {
  string analysis_id = 1;
  double network_score = 2;
  repeated string insights = 3;
  VisualizationData visualization_data = 10;  // Field 10, not 4—left room!
}
```

Six months from now, we add:
```proto
  string coordinator_summary = 11;  // NEW FIELD
  repeated string deprecated_field = 4 [deprecated = true];
```

**Old code reading new data**: Ignores field 11. Works.  
**New code reading old data**: Field 11 is unset/empty. Works.  
**Type safety**: Go gets `GetCoordinatorSummary() string`, TS gets `coordinatorSummary?: string`.

This is *specified behavior*, not emergent behavior!

**Jake Morrison (JSON):**

Sure, but we get the same with Go's zero values and TypeScript's optional fields! Look:

```typescript
interface ToolCallSnapshot {
    name: string;
    input?: unknown;  // Optional already
    exec?: boolean;
    retryCount?: number;  // Add this tomorrow, old data works
}
```

Go's `omitempty` means old data unmarshals with zero values. New code checks `if snap.RetryCount > 0`. This is idiomatic Go!

The difference is: with JSON, I can see the actual shape in the DB. With Proto, I need to decode it mentally. For a database we'll query in SQL, that's a dealbreaker.

**The Existing Codebase:**

```go
// When ToolCallSnapshot gains a new field:
type ToolCallSnapshot struct {
    SnapshotBase
    Name  string
    Input json.RawMessage
    Exec  bool
    // Just add this:
    RetryCount int `json:"retry_count,omitempty"`
}
// Old snapshots unmarshal with RetryCount=0
// New snapshots include it
// Frontend checks `retryCount ?? 0`
// DONE.
```

I've been doing this for decades. It's called "robust parsing". Protobuf didn't invent it.

**Dr. Rodriguez (Pragmatic):**

You're both right about the *mechanism*, but wrong about the *problem*.

The issue isn't "will old data unmarshal"—it's "what does the system DO with missing fields?"

Example: `ToolCallSnapshot` adds `requiredPermissions: string[]`. Old snapshots don't have it. Does the frontend:
- Assume empty array? (Might grant incorrect access)
- Throw an error? (Breaks old conversations)
- Show a warning? (Degrades UX)

**Neither JSON nor Protobuf answers this!** You need:
1. **Migration strategy** - Do we backfill? Do we version snapshots?
2. **Semantic versioning** - `SnapshotBase.Version` field that tracks schema version
3. **Validation layer** - Reject snapshots with incompatible versions

Protobuf's advantage is it *makes you think about this upfront*. JSON lets you be lazy. But lazy is faster to ship...

**Alex Kim (Ops):**

Maya's hitting the nail on the head. Let me add: we ALREADY have a `Version` field in `SnapshotBase`! 

```go
type SnapshotBase struct {
    Version int64 `json:"version"`  // Currently used for ordering
    // ...
}
```

If we repurpose this as a *schema* version, we can version snapshots independently of their freshness. Then:
```go
const ToolCallSnapshotSchemaV1 = 1
const ToolCallSnapshotSchemaV2 = 2  // Added requiredPermissions

func (s *ToolCallSnapshot) GetSchemaVersion() int { return ToolCallSnapshotSchemaV2 }
```

Works with JSON OR Protobuf! The storage format is orthogonal to the evolution strategy.

---

### Question 2: Developer Experience and Debugging

**Jake Morrison (JSON):**

Let me show you the dev flow right now:

```bash
# Something's wrong with conversation conv-abc
$ sqlite3 snapshots.db
sqlite> SELECT snapshot_json FROM snapshots WHERE conversation_id='conv-abc';

{
  "conversation_id": "conv-abc",
  "entity_id": "tool-123",
  "kind": "tool_call",
  "status": "completed",
  "name": "team_analyzer",
  "input": {"team_size": 5}
}
```

Copy that JSON. Paste it into your editor. Immediately understand the problem. No tools, no decoders, no "let me regenerate the proto first".

**This is what 3am debugging looks like.** When you're in prod, with a customer on the line, you need speed. JSON is speed.

**Dr. Chen (Pro-Protobuf):**

Jake, that's a *strawman*! We'd use `protojson.Marshal`, not binary! The output is *identical*:

```go
func MarshalSnapshot(s Snapshot) ([]byte, error) {
    // Current JSON approach:
    return json.Marshal(s)
    
    // With Proto:
    return protojson.Marshal(s)  // SAME JSON OUTPUT
}
```

You get the same debugging flow! But WITH type safety. Let me show you where JSON FAILS:

```bash
sqlite> SELECT snapshot_json FROM snapshots WHERE entity_id='bad-data';

{
  "kind": "tool_call",
  "name": "teamAnalyzer",  <-- TYPO! Should be team_analyzer
  "network_score": "high"  <-- WRONG TYPE! Should be float
}
```

This goes into your DB. Frontend crashes when it tries to parse. With Proto + validation, **this never happens** because:

```go
toolCall := &snapshots.ToolCallSnapshot{...}
toolCall.Validate()  // FAILS at write time, not read time
```

**The Existing Codebase:**

```go
// I don't validate. I trust you to send good data.
// Is that a bug or a feature?
func (s *SQLiteSnapshotStore) Upsert(ctx context.Context, snap Snapshot) error {
    b, err := MarshalSnapshot(snap)  // What if snap is malformed?
    // ... write to SQLite
}
```

Sarah has a point. We have ZERO validation currently. Bad data can enter the system.

But is Protobuf the answer, or do we just need:
```go
type Snapshot interface {
    Base() *SnapshotBase
    KindValue() SnapshotKind
    Validate() error  // <-- Add this
}
```

**Dr. Rodriguez (Pragmatic):**

This debate is revealing the *real* gap: **we don't validate data at write-time**.

JSON vs Protobuf is irrelevant if we're not checking invariants! Here's what we need:

```go
func (s *ToolCallSnapshot) Validate() error {
    if s.Name == "" { return errors.New("name required") }
    if s.Status != "pending" && s.Status != "running" && s.Status != "completed" {
        return errors.New("invalid status")
    }
    return nil
}
```

Protobuf gives you field-level validation (required, repeated, etc). But you still need *business logic* validation.

JSON gives you nothing. You build it yourself.

**Alex Kim (Ops):**

From an ops perspective, I want:
1. **Readable DB** - JSON ✓, Protobuf-JSON ✓, Binary Proto ✗
2. **SQL queries** - SQLite's `json_extract(snapshot_json, '$.name')` works with JSON
3. **Debugging tools** - `jq`, `sqlite3`, VS Code all understand JSON natively

If we go Protobuf, we MUST commit to JSON encoding in storage. Which means:
- Protobuf is purely a *schema definition* language
- The wire format is still JSON
- We get type safety + codegen + validation
- We keep debuggability

That's actually... reasonable? But it's also *more complexity* for the same runtime behavior.

---

### Question 3: Type Safety Across the Stack

**Dr. Chen (Pro-Protobuf):**

This is where Protobuf SHINES. Look at our current Team Analysis flow:

**Backend (Go):**
```go
// go-go-mento/go/pkg/sem/pb/proto/sem/domain/team_analysis.pb.go (generated)
type TeamAnalysisResult struct {
    AnalysisId string
    NetworkScore float64
    Insights []string
    VisualizationData *VisualizationData
}
```

**Frontend (TypeScript):**
```typescript
// go-go-mento/web/src/sem/pb/proto/sem/domain/team_analysis_pb.ts (generated)
export interface TeamAnalysisResult {
  analysisId: string;
  networkScore: number;
  insights: string[];
  visualizationData?: VisualizationData;
}
```

**SAME schema, ZERO drift**. Change the `.proto`, run `make protobuf`, both sides update atomically.

Now look at snapshots WITHOUT Proto:

**Go:**
```go
type TeamAnalysisSnapshot struct {
    SnapshotBase
    TeamSize int     `json:"team_size,omitempty"`
    Progress float64 `json:"progress,omitempty"`
}
```

**TypeScript:**
```typescript
// Did someone update this? Who knows!
interface TeamAnalysisEntity {
  teamSize?: number;  // Is this the same as Go's TeamSize?
  progress?: number;  // Same name, but checked recently?
}
```

How do you *know* they match? Code review? Hope? Prayer?

**Jake Morrison (JSON):**

Sarah, you're being disingenuous. We have ONE source of truth: the Go structs with JSON tags. TypeScript follows the JSON.

```typescript
// timeline/types.ts
export interface MessageEntity extends BaseTimelineEntity {
  kind: 'message';
  props: {
    role: string;      // Matches Go's Role
    content: string;   // Matches Go's Text? Or Content? Wait...
    streaming: boolean;
  };
}
```

Okay, fine, there's *some* drift risk. But we can add a build-time test:

```go
func TestSnapshotJSONMatchesTypeScript(t *testing.T) {
    snap := &LLMTextSnapshot{Role: "user", Text: "hi", Streaming: false}
    json := marshal(snap)
    // Parse with a TS JSON schema validator
    // Fail if mismatch
}
```

No Protobuf needed! Just discipline + automation.

**The Existing Codebase:**

```go
// I'll be honest: I don't enforce TypeScript consistency.
// That's a process issue, not a technology issue.
//
// If you want type safety:
// Option A) Protobuf (build-time generation)
// Option B) JSON Schema (runtime validation)
// Option C) TypeScript --> Go (reverse codegen)
//
// I'm neutral. I just store bytes.
```

**Dr. Rodriguez (Pragmatic):**

Here's the uncomfortable truth: **Go → TypeScript type sync is HARD**, regardless of your serialization format.

Even with Protobuf, you have edge cases:
- `int64` becomes `string` in TS (JS number precision)
- `map<string, any>` loses type info
- Nested `oneof` fields are awkward in TS

JSON has the same issues! Plus:
- `omitempty` doesn't map to TS optional cleanly
- `interface{}` becomes `any`, losing all safety
- Go's zero values != TS's `undefined`

The real solution? **Don't sync types. Sync schemas.**

Use JSON Schema (or Proto) as the canonical definition. Generate Go AND TypeScript from that. Then the storage format is just an implementation detail.

**Alex Kim (Ops):**

From a practical standpoint, here's what happens in production:

**Scenario 1: Backend adds field, frontend doesn't update**
- JSON: Frontend ignores new field, old behavior continues
- Proto: Same (unmapped JSON fields are ignored)

**Scenario 2: Frontend expects field, backend doesn't send**
- JSON: `entity.props.newField` is `undefined`, frontend crashes if not defensive
- Proto: TS type shows `newField?: string`, compiler forces defensive code

**Scenario 3: Type mismatch (string vs number)**
- JSON: Runtime error when frontend tries to parse
- Proto: Go won't compile if you assign wrong type

Protobuf catches errors *earlier* (compile time), but both STILL require defensive frontend code because:
- Old snapshots exist in the DB
- Hydration can return stale data
- Network can corrupt JSON (yes, it happens)

So: Proto helps, but doesn't eliminate the problem.

---

### Question 4: Performance and Storage Efficiency

**Alex Kim (Ops):**

Let me lead with DATA, not opinions.

**Current SQLite DB stats (from pinocchio):**
```sql
sqlite> SELECT 
  COUNT(*) as snapshots,
  SUM(LENGTH(snapshot_json)) as total_bytes,
  AVG(LENGTH(snapshot_json)) as avg_bytes
FROM snapshots;

snapshots: 8,432
total_bytes: 2,847,392  (2.8 MB)
avg_bytes: 337 bytes
```

Most snapshots are tiny! LLM text messages, tool calls, status events. Even Team Analysis with visualization data is <2KB.

**Protobuf binary encoding would save ~30-40% space:**
- JSON: `{"analysis_id": "abc123"}` = 28 bytes
- Proto binary: Field tag + string = ~18 bytes

For 2.8 MB, that's... 800 KB saved. On a disk, in SQLite, with TEXT columns. **Not meaningful**.

Where Protobuf binary DOES help: **wire transfer**. But we're not transferring snapshots over the wire frequently! Hydration is once per page load, max a few hundred entities.

**Dr. Chen (Pro-Protobuf):**

Alex is right that binary encoding isn't the win here. But there's a subtler performance issue: **parsing**.

JSON unmarshal in Go uses reflection. It's slower than generated code:

```go
// JSON (reflection-based)
func unmarshalLLMText(data []byte) (*LLMTextSnapshot, error) {
    var snap LLMTextSnapshot
    json.Unmarshal(data, &snap)  // Uses reflect
    return &snap, nil
}

// Proto (generated code, no reflection)
func unmarshalLLMText(data []byte) (*LLMTextSnapshot, error) {
    return proto.Unmarshal(data)  // Direct field assignment
}
```

Benchmarks show 2-3x faster unmarshal with Proto. For hydration endpoints loading hundreds of snapshots, that adds up!

**Jake Morrison (JSON):**

Sarah, you're talking about *microseconds*. Let's benchmark the ACTUAL hydration flow:

```
GET /api/conversations/conv-123/timeline
1. SQLite query: ~2ms (cold) / ~0.5ms (cached)
2. JSON unmarshal (100 snapshots): ~0.8ms
3. HTTP response: ~1ms (localhost)
Total: ~4ms
```

Even if Proto cuts unmarshal to 0.3ms, you've saved... 0.5ms. On a 4ms request. **12% improvement**.

Meanwhile, the SQLite query is 50% of the time, and that's database design, not serialization!

**The Existing Codebase:**

```go
// My actual perf bottleneck:
func (s *SQLiteSnapshotStore) GetByConversation(
    ctx context.Context, conversationID string, sinceVersion *int64,
) ([]Snapshot, error) {
    rows, err := s.db.QueryContext(ctx, 
        `SELECT snapshot_json FROM snapshots 
         WHERE conversation_id = ? 
         ORDER BY updated_at ASC`,  // <-- This ORDER BY is the cost
        conversationID,
    )
    // ... unmarshal each row
}
```

You want perf? Add an index on `(conversation_id, updated_at)`. We already have it. That's the win.

Switching to Protobuf for a 10% unmarshal speedup is premature optimization.

**Dr. Rodriguez (Pragmatic):**

Everyone's focusing on the wrong metrics. Here's what matters for LONG-TERM performance:

**Schema bloat**: As we add widgets, each snapshot type grows. Without discipline:
```go
type ToolCallSnapshot struct {
    // 30 fields later...
    DeprecatedField1 string  // Can't remove, might break old data
    DeprecatedField2 int     // Same
    // ...
}
```

**Migration cost**: To remove deprecated fields, you must:
1. Scan entire DB
2. Rewrite every snapshot
3. Handle failures mid-migration

Protobuf's `[deprecated = true]` tag at least *documents* which fields are legacy. JSON has... comments, I guess?

**Validation cost**: Bad data in the DB means:
- Frontend errors (user-facing)
- Support tickets (engineer time)
- Rollbacks (deployment risk)

If validation catches 1 bad deploy per year, the ROI is massive. Protobuf + validation might be worth it just for that.

---

### Question 5: Migration Path and Risk

**The Existing Codebase:**

```go
// I am PROD CODE. I work RIGHT NOW.
// Every change to me is RISK.
//
// To migrate to Protobuf, you must:
// 1. Define .proto schemas for all 6 snapshot types
// 2. Generate Go code (where does it live? pkg/snapshots/pb/?)
// 3. Update SQLite store to handle both formats during migration
// 4. Backfill existing snapshots OR support dual-read
// 5. Update frontend TypeScript types
// 6. Test hydration with mixed old/new data
// 7. Deploy backend, then frontend (or vice versa? rollback plan?)
//
// Estimated effort: 2-3 sprints
// Risk: HIGH (touches storage layer)
// Benefit: ???
```

Tell me why I should let you do this.

**Dr. Chen (Pro-Protobuf):**

Because we're at 6 snapshot types NOW. In a year, we'll have 20. In two years, 50.

Each type evolution without Proto is a landmine:
- Forgot to update frontend type? Runtime error.
- Added a required field? Breaks old snapshots.
- Changed a field name? Silent data loss.

Protobuf is an *investment*. The migration cost is high, but the ongoing maintenance cost is LOW.

Migration path:
```go
// Phase 1: Dual-write (both JSON and Proto in DB)
func (s *SQLiteSnapshotStore) Upsert(ctx context.Context, snap Snapshot) error {
    jsonBytes, _ := json.Marshal(snap)
    protoBytes, _ := proto.Marshal(snap)  // New column
    s.db.Exec(`UPDATE snapshots SET 
        snapshot_json = ?, 
        snapshot_proto = ?  -- New column
        WHERE ...`, jsonBytes, protoBytes)
}

// Phase 2: Dual-read (try Proto, fallback to JSON)
// Phase 3: Drop JSON column
```

This is SAFE. We've done it before with other migrations.

**Jake Morrison (JSON):**

Sarah, you're describing 3 months of work to get back to where we are now, but with extra dependencies.

Let me propose an alternative: **JSON Schema + TypeScript codegen**.

```bash
# 1. Define snapshot-schema.json (JSON Schema)
# 2. Generate Go structs: go-jsonschema -p snapshots snapshot-schema.json
# 3. Generate TS types: json-schema-to-typescript snapshot-schema.json
```

This gives us:
- Type safety (both sides)
- Validation (ajv for TS, jsonschema for Go)
- No Protobuf dependency
- JSON storage (debuggable)
- 1 sprint, not 3

Why isn't this on the table?

**Dr. Rodriguez (Pragmatic):**

Jake's JSON Schema idea is actually compelling for our use case! Let me compare:

**Protobuf:**
- ✓ Excellent Go support
- ✓ Excellent TS support (via Buf)
- ✓ Built-in validation
- ✓ Schema evolution semantics
- ✗ Build-time dependency (buf, protoc)
- ✗ New concept for team to learn

**JSON Schema:**
- ✓ Universal format
- ✓ Runtime validation (ajv)
- ✓ No build dependency (can use runtime validation)
- ✓ Team already knows JSON
- ✗ Go codegen is clunky (go-jsonschema isn't great)
- ✗ No native evolution semantics

**Manual JSON (current):**
- ✓ Zero dependencies
- ✓ Maximum flexibility
- ✓ Team knows it
- ✗ No validation
- ✗ Drift risk
- ✗ No evolution plan

For OUR team, with OUR existing Protobuf investment in SEM events, I'd say: **go with Protobuf, but JSON-encode it**.

For a NEW project with no Proto? JSON Schema might be better.

**Alex Kim (Ops):**

Here's my pragmatic take on the migration:

**Don't migrate existing snapshots.** Too risky.

Instead:
```go
const SnapshotFormatJSON = 1
const SnapshotFormatProtoJSON = 2

type SnapshotBase struct {
    // ...
    Format int `json:"format,omitempty"`  // 0 or 1 = JSON, 2 = ProtoJSON
}

func (s *SQLiteSnapshotStore) Upsert(snap Snapshot) error {
    snap.Base().Format = SnapshotFormatProtoJSON  // New snapshots use Proto
    // Marshal with protojson.Marshal
}

func (s *SQLiteSnapshotStore) GetByConversation(...) ([]Snapshot, error) {
    // Read format field, unmarshal accordingly
    if format == SnapshotFormatProtoJSON {
        return protojson.Unmarshal(...)
    } else {
        return json.Unmarshal(...)  // Legacy path
    }
}
```

Over time, old snapshots age out (TTL policy). New snapshots are Proto. No risky migration.

**Risk**: LOW (additive change)  
**Effort**: 1 sprint (protodefs + dual unmarshal)  
**Benefit**: Future snapshots get type safety

This is how you do production migrations: **incrementally, with backward compat**.

---

## Round 3: Closing Statements (1 minute each)

**Dr. Chen (Pro-Protobuf):**

We're already using Protobuf successfully in go-go-mento for SEM events. The team knows it, the tooling works, and it gives us type safety across Go and TypeScript.

Adding Proto for snapshots is NOT a rewrite—it's extending a pattern we trust. With JSON encoding (`protojson`), we keep debuggability. With Alex's incremental migration, we minimize risk.

The alternative is *status quo*: manual types, drift risk, no validation, no evolution plan. That's a time bomb.

Vote Protobuf. Invest in our future.

**Jake Morrison (JSON):**

I hear "we already use Proto for events" as if that's a reason to use it for *everything*. Events are ephemeral. Snapshots are durable. Different requirements!

We have a working system. 6 snapshot types, 8,000+ records, zero production issues. The "drift risk" is theoretical. The "validation gap" can be solved with lightweight schema checks.

Don't add complexity for hypothetical future problems. Solve actual, present problems.

Vote JSON. Keep it simple.

**The Existing Codebase:**

```go
// I don't have opinions.
// I execute instructions.
// I store bytes and return bytes.
//
// But I will say this:
// Every abstraction layer you add on top of me
// is another place bugs hide.
//
// Choose wisely.
```

**Dr. Rodriguez (Pragmatic):**

The right answer depends on your 2-year plan.

If you're building 5-10 more widgets and that's it? JSON is fine. Add some validation, add some tests, call it done.

If you're building a platform where external teams will define their own widget types, with their own schema evolution needs? Protobuf's structure will save you.

My vote: **Protobuf with JSON encoding**, but ONLY if you commit to:
1. Proper validation at write-time
2. Schema versioning strategy
3. Migration tooling for breaking changes

Half-assing Protobuf is worse than thoughtful JSON.

**Alex Kim (Ops):**

I'll support whatever decision you make, but I need commitments:

**If JSON:**
- Add validation to `Snapshot.Validate()`
- Write schema tests (Go ↔ TS)
- Document evolution rules

**If Protobuf:**
- Keep JSON encoding (not binary)
- Dual-read during migration
- Runbook for schema changes

Don't just pick a technology. Pick a PROCESS.

Because in 6 months, I'm the one debugging production at 3am, and I need to trust that the system has guardrails.

---

## Post-Debate Analysis

### Votes

**For Protobuf:**
- Dr. Chen (strong yes)
- Dr. Rodriguez (qualified yes)

**For JSON:**
- Jake Morrison (strong yes)
- The Existing Codebase (abstain, leaning status quo)

**Neutral/Conditional:**
- Alex Kim (will support either, needs process commitment)

### Key Takeaways

1. **The real issue isn't serialization format—it's schema evolution strategy**. Neither JSON nor Protobuf solves this alone.

2. **Protobuf's value is in codegen + validation**, not wire format efficiency. If we use `protojson`, we keep JSON's debuggability.

3. **Migration risk can be managed** with Alex's incremental dual-format approach. No need for risky backfills.

4. **The team's existing Protobuf investment** (SEM events) lowers the adoption cost. Tooling and knowledge already exist.

5. **JSON Schema is a viable middle ground** that wasn't fully explored. Deserves consideration.

### Recommendation

**Go with Protobuf + JSON encoding** IF:
- You plan to add 10+ more widget types
- You want compile-time type safety across Go/TS
- You commit to validation + migration tooling

**Stick with JSON** IF:
- You're happy with 6-10 widget types
- You add validation + schema tests manually
- You prefer simplicity over structure

**The deciding factor**: How much schema evolution do you expect? If widgets are mostly stable, JSON wins. If they're rapidly evolving, Protobuf's guardrails pay off.

---

## Appendix: Code Samples

### Current JSON Approach
```go
// pinocchio/pkg/snapshots/types.go
type LLMTextSnapshot struct {
    SnapshotBase
    Role      string `json:"role"`
    Text      string `json:"text"`
    Streaming bool   `json:"streaming"`
}
```

### Proposed Protobuf Approach
```proto
// proto/snapshots/llm.proto
syntax = "proto3";
package snapshots;

message LLMTextSnapshot {
  SnapshotBase base = 1;
  string role = 2;
  string text = 3;
  bool streaming = 4;
}

message SnapshotBase {
  string conversation_id = 1;
  string entity_id = 2;
  string kind = 3;
  string status = 4;
  int64 started_at = 5;
  int64 updated_at = 6;
  int64 version = 7;
  map<string, string> flags = 8;
}
```

### Hybrid Approach (Incremental Migration)
```go
func (s *SQLiteSnapshotStore) Upsert(ctx context.Context, snap Snapshot) error {
    base := snap.Base()
    
    // New snapshots use Proto format
    base.Format = SnapshotFormatProtoJSON
    
    // Marshal with protojson (human-readable)
    b, err := protojson.Marshal(snap)
    if err != nil {
        return errors.Wrap(err, "marshal snapshot")
    }
    
    // Store in existing snapshot_json column (no schema change needed!)
    _, err = s.db.ExecContext(ctx, `
        INSERT INTO snapshots (conversation_id, entity_id, kind, version, snapshot_json)
        VALUES (?, ?, ?, ?, ?)
        ON CONFLICT(conversation_id, entity_id) DO UPDATE SET
            snapshot_json = excluded.snapshot_json
    `, base.ConversationID, base.EntityID, string(base.Kind), base.Version, string(b))
    
    return errors.Wrap(err, "upsert snapshot")
}
```

This gives us:
- Proto type safety
- JSON debuggability
- No database migration
- Backward compatibility with existing snapshots

