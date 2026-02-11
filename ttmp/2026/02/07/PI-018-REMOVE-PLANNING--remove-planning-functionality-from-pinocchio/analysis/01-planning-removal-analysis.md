---
title: Planning Removal Analysis
doc_type: analysis
status: active
intent: long-term
topics:
  - pinocchio
  - refactoring
  - cleanup
created: 2026-02-07
updated: 2026-02-07
owners: []
---

# Planning Removal Analysis

## Executive Summary

The planning functionality in pinocchio is a comprehensive but apparently unused feature that enables LLM-based multi-step planning with iterations, reflections, and execution phases. This document details every file, type, and code section that must be modified or deleted to completely remove this functionality.

**Total estimated lines to remove/modify:** ~1,800+ lines across Go, Proto, and TypeScript

## Architecture Overview

The planning feature follows this data flow:

```
Events (typed_planning.go)
    ↓
SEM Translator (sem_translator.go) → SEM frames (JSON)
    ↓
Timeline Projector (timeline_projector.go) → PlanningSnapshotV1 (persisted)
    ↓
WebSocket → Frontend
    ↓
Registry (registry.ts) → Timeline slice (Redux)
    ↓
PlanningCard (cards.tsx) → UI
```

## Planning Event Types

Six event types are defined in `typed_planning.go`:

| Event Type | Description |
|------------|-------------|
| `EventPlanningStart` | Marks beginning of planning phase |
| `EventPlanningIteration` | Represents a single planning iteration with decision/strategy/progress |
| `EventPlanningReflection` | Planner reflecting on progress with score |
| `EventPlanningComplete` | Marks end of planning phase with final decision/directive |
| `EventExecutionStart` | Marks beginning of execution of the plan |
| `EventExecutionComplete` | Marks end of execution with status/tokens/error |

## Files to DELETE

### Go Files

**`pinocchio/pkg/inference/events/typed_planning.go`** (179 lines)

Contains all 6 event type structs and their constructors. Also contains `init()` function that registers event factories.

Key types:
- `EventPlanningStart`
- `EventPlanningIteration`
- `EventPlanningReflection`
- `EventPlanningComplete`
- `EventExecutionStart`
- `EventExecutionComplete`

### Proto Files

**`pinocchio/proto/sem/middleware/planning.proto`** (73 lines)

Defines the SEM event payloads:
- `PlanningRun` - shared identifier for planner run
- `PlanningStarted` - fired when planning begins
- `PlanningIteration` - fired per iteration
- `PlanningReflection` - intermediate reflection snapshots
- `PlanningCompleted` - fired when planning concludes
- `ExecutionStarted` - execution phase begins
- `ExecutionCompleted` - execution phase ends

**`pinocchio/proto/sem/timeline/planning.proto`** (52 lines)

Defines the timeline snapshot payloads:
- `PlanningIterationSnapshotV1`
- `ExecutionSnapshotV1`
- `PlanningSnapshotV1`

### Generated Files (Go)

These will be deleted and the containing packages regenerated:

- `pinocchio/pkg/sem/pb/proto/sem/middleware/planning.pb.go` (700 lines)
- `pinocchio/pkg/sem/pb/proto/sem/timeline/planning.pb.go` (452 lines)

### Generated Files (TypeScript)

- `pinocchio/cmd/web-chat/web/src/sem/pb/proto/sem/middleware/planning_pb.ts`
- `pinocchio/cmd/web-chat/web/src/sem/pb/proto/sem/timeline/planning_pb.ts`

## Files to MODIFY

### Proto Files

**`pinocchio/proto/sem/timeline/transport.proto`**

Remove:
```protobuf
import "proto/sem/timeline/planning.proto";
```

And in `TimelineEntityV1.snapshot` oneof, remove:
```protobuf
sem.timeline.PlanningSnapshotV1 planning = 18;
```

### Go Backend Files

**`pinocchio/pkg/webchat/timeline_projector.go`**

Remove:
1. The `planningAgg` struct (lines 28-31):
```go
type planningAgg struct {
	snap       *timelinepb.PlanningSnapshotV1
	iterations map[int32]*timelinepb.PlanningIterationSnapshotV1
}
```

2. The `planning` field from `TimelineProjector` struct:
```go
planning     map[string]*planningAgg // key: run_id
```

3. The initialization in `NewTimelineProjector`:
```go
planning:     map[string]*planningAgg{},
```

4. The case block in `ApplySemFrame` for planning events (lines ~298-301):
```go
case "planning.start", "planning.iteration", "planning.reflection", "planning.complete", "execution.start", "execution.complete":
	return p.applyPlanning(ctx, seq, env.Event.Type, env.Event.Data)
```

5. The entire `applyPlanning` method (~150 lines, lines 309-458)

---

**`pinocchio/pkg/webchat/sem_translator.go`**

Remove all 6 planning event handler registrations from `RegisterDefaultHandlers()`:

```go
// Lines ~506-580
semregistry.RegisterByType[*pinevents.EventPlanningStart](...)
semregistry.RegisterByType[*pinevents.EventPlanningIteration](...)
semregistry.RegisterByType[*pinevents.EventPlanningReflection](...)
semregistry.RegisterByType[*pinevents.EventPlanningComplete](...)
semregistry.RegisterByType[*pinevents.EventExecutionStart](...)
semregistry.RegisterByType[*pinevents.EventExecutionComplete](...)
```

Also remove the import:
```go
pinevents "github.com/go-go-golems/pinocchio/pkg/inference/events"
```

And the semMw alias may need cleanup if only planning types were used from it.

---

**`pinocchio/pkg/webchat/router.go`**

Remove the planning middleware check (lines 1079-1098):
```go
if waitErr != nil && conv != nil && conv.Sink != nil && middlewareEnabled(cfg.Middlewares, "planning") {
    // Ensure execution.complete exists...
    md := events.EventMetadata{...}
    ...
}
```

Also consider removing `middlewareEnabled` function if it's only used for planning.

---

**`pinocchio/cmd/web-chat/main.go`**

Remove the emit-planning-stubs parameter (line 63):
```go
parameters.NewParameterDefinition("emit-planning-stubs", parameters.ParameterTypeBool, parameters.WithDefault(false), parameters.WithHelp("Emit stub planning/thinking-mode semantic events (for UI demos); disabled by default")),
```

And any code that references this parameter value.

### TypeScript Frontend Files

**`pinocchio/cmd/web-chat/web/src/sem/registry.ts`**

Remove imports (lines 12-22):
```typescript
import {
  type ExecutionCompleted,
  ExecutionCompletedSchema,
  type ExecutionStarted,
  ExecutionStartedSchema,
  type PlanningCompleted,
  PlanningCompletedSchema,
  type PlanningIteration,
  PlanningIterationSchema,
  type PlanningReflection,
  PlanningReflectionSchema,
  type PlanningStarted,
  PlanningStartedSchema,
} from '../sem/pb/proto/sem/middleware/planning_pb';
```

Remove the `PlanningAgg` type (lines 83-103):
```typescript
type PlanningAgg = {...};
```

Remove `planningAggs` Map and helper functions (lines 105-143):
```typescript
const planningAggs = new Map<string, PlanningAgg>();
function ensurePlanningAgg(...) {...}
function planningEntityFromAgg(...) {...}
```

Remove the `planningAggs.clear()` call in `registerDefaultSemHandlers()` (line 147).

Remove all 6 planning/execution event handlers (lines 323-425):
```typescript
registerSem('planning.start', ...)
registerSem('planning.iteration', ...)
registerSem('planning.reflection', ...)
registerSem('planning.complete', ...)
registerSem('execution.start', ...)
registerSem('execution.complete', ...)
```

---

**`pinocchio/cmd/web-chat/web/src/sem/timelineMapper.ts`**

Remove the planning case (lines 81-114):
```typescript
if (kind === 'planning' && oneof.case === 'planning') {
  const iterations = ...
  ...
}
```

---

**`pinocchio/cmd/web-chat/web/src/webchat/cards.tsx`**

Remove the `PlanningCard` component (lines 154-210):
```typescript
export function PlanningCard({ e }: { e: RenderEntity }) {
  ...
}
```

---

**`pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx`**

Remove import (line 15):
```typescript
PlanningCard,
```

Remove from card kind array (line 104):
```typescript
{ slug: 'planning' }
```

Remove from renderers object (line 227):
```typescript
planning: PlanningCard,
```

---

**`pinocchio/cmd/web-chat/web/src/webchat/components/Timeline.tsx`**

Remove 'planning' from system lane categorization (line 23):
```typescript
e.kind === 'planning' ||
```

## Removal Order (Recommended)

1. **Delete proto files** - This ensures buf generate will fail until transport.proto is fixed
2. **Update transport.proto** - Remove the planning import and field
3. **Run buf generate** - Regenerate all pb.go and _pb.ts files
4. **Delete typed_planning.go** - Remove event types
5. **Update sem_translator.go** - Remove handlers (will fail to compile if events still referenced)
6. **Update timeline_projector.go** - Remove aggregation logic
7. **Update router.go** - Remove middleware check
8. **Update main.go** - Remove parameter
9. **Update frontend files** - registry.ts, timelineMapper.ts, cards.tsx, ChatWidget.tsx, Timeline.tsx
10. **Run tests** - Verify nothing breaks
11. **Clean up imports** - Remove any unused imports in modified files

## Verification Commands

After removal, verify no references remain:

```bash
# Check Go code
rg "Planning|Execution" pinocchio/pkg/ --type go | grep -v test

# Check proto
rg "planning" pinocchio/proto/

# Check frontend
rg "planning|Planning" pinocchio/cmd/web-chat/web/src/

# Compile
cd pinocchio && go build ./...

# Run tests
cd pinocchio && go test ./...
```

## Risk Assessment

**Low Risk:**
- The feature appears to be unused (no actual middleware implements it)
- The `emit-planning-stubs` flag defaults to false
- Code is well-isolated to specific files

**Medium Risk:**
- Generated protobuf files need careful regeneration
- Frontend may have subtle dependencies in Redux state shape

**Mitigation:**
- Run full test suite after each major step
- Keep a branch with the old code until verified in production
