---
Title: 'Webchat QA: stronger checks + error handling'
Ticket: GP-014-WEBCHAT-QUALITY-CHECKS
Status: active
Topics:
    - frontend
    - infrastructure
    - architecture
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: moments/Makefile
      Note: Reference lint-web workflow
    - Path: moments/lefthook.yml
      Note: Reference hook flow for web checks
    - Path: moments/web/biome.json
      Note: Reference Biome rule set
    - Path: moments/web/package.json
      Note: Reference lint/type-check scripts
    - Path: pinocchio/Makefile
      Note: Root lint targets (Go-only)
    - Path: pinocchio/cmd/web-chat/web/package.json
      Note: Webchat scripts; add lint/typecheck/check
    - Path: pinocchio/cmd/web-chat/web/src/chat/ChatWidget.tsx
      Note: Silent catches and UI error handling
    - Path: pinocchio/cmd/web-chat/web/src/ws/wsManager.ts
      Note: Hydration and WS error handling
    - Path: pinocchio/cmd/web-chat/web/tsconfig.json
      Note: Typecheck config and strictness
    - Path: pinocchio/lefthook.yml
      Note: Hook integration for lint/test
ExternalSources: []
Summary: Detailed plan to strengthen webchat linting/typechecking and improve runtime error visibility, using Moments as a reference.
LastUpdated: 2026-01-25T16:23:52-05:00
WhatFor: ""
WhenToUse: ""
---


# Webchat QA: Stronger Checks + Error Handling

## Goal

Define a concrete plan to improve developer tooling and runtime reliability for the webchat frontend. This includes:
- Stronger static checks (TypeScript + Biome lint rules).
- Consistent “check” workflows (Makefile, hooks, CI).
- Runtime error visibility (no silent failures, structured logs, error boundaries).

The plan is anchored in the existing Moments web pipeline to minimize invention and maximize consistency.

## Scope

Target codebase:
- `pinocchio/cmd/web-chat/web` (webchat frontend)
- Go backend tooling references where needed (`pinocchio/Makefile`, `pinocchio/lefthook.yml`)
- Moments web config as reference (`moments/web`)

## Evidence: Current Typecheck Failures

Running `npm run typecheck` in `pinocchio/cmd/web-chat/web` produces:

- `src/sem/registry.ts(154,52): error TS2345: Argument of type 'bigint' is not assignable to parameter of type 'number'.`
- `src/ws/wsManager.ts(28,47): error TS2345: Argument of type 'bigint' is not assignable to parameter of type 'number'.`
- `src/ws/wsManager.ts(161,13): error TS2552: Cannot find name 'isObject'. Did you mean 'Object'?`

These errors confirm that a standard typecheck would have caught the missing `isObject` symbol and the bigint/number mismatches.

## Current Webchat Tooling (Pinocchio)

### Scripts (webchat)

`pinocchio/cmd/web-chat/web/package.json` provides:
- `dev` → `vite`
- `typecheck` → `tsc -p tsconfig.json --noEmit`
- `build` → `vite build --outDir ../static/dist`

There is no Biome config or lint script for webchat, and `vite build` does not run `tsc`.

### Hooks / Linting (repo root)

`pinocchio/lefthook.yml` only runs Go lint/test on pre-commit and pre-push. Webchat is not included.

`pinocchio/Makefile` has Go lint/test targets only.

### Error Handling (webchat)

Error handling is currently *silent* in multiple paths:

- `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`
  - `ws.onmessage` JSON parse is wrapped in `try/catch { /* ignore */ }`.
  - `/timeline` hydration is wrapped in `try/catch { /* ignore */ }`.
  - `ws.close()` is wrapped in `try/catch { /* ignore */ }`.

- `pinocchio/cmd/web-chat/web/src/chat/ChatWidget.tsx`
  - URL parsing + history rewrite are wrapped in empty `catch` blocks.
  - `wsManager.connect` and `wsManager.ensureHydrated` are wrapped with empty `catch` blocks.

This pattern hides runtime failures (e.g., missing `isObject`), making bugs hard to detect.

## Moments Baseline (Reference Implementation)

### Biome Configuration

`moments/web/biome.json`:
- Linter enabled with `recommended` rules.
- Explicit overrides enforcing architectural boundaries via `noRestrictedImports`.
- Explicit rule tuning for correctness and suspicious patterns.

### Scripts + Check Flow

`moments/web/package.json`:
- `type-check`: `tsc -b --noEmit` (also runs version generation).
- `lint`: `biome ci .` (CI-friendly).
- `build`: runs typecheck + build.

`moments/Makefile`:
- `lint-web`: runs `pnpm run type-check` then `pnpm run lint`.

`moments/lefthook.yml`:
- Hooks call `cd web && pnpm run lint` (Biome) and backend lint.

Moments uses a unified lint/typecheck workflow across scripts, Makefile, and hooks; webchat does not.

## Gap Analysis (Webchat vs Moments)

### Static Checks

- **Webchat**: `tsc` exists but is not run by build or hooks; no Biome lint.
- **Moments**: Biome lint + TypeScript check are both first-class citizens in build and hooks.

### Error Visibility

- **Webchat**: multiple silent catches; no centralized logging or error boundary.
- **Moments**: not inspected for runtime logging, but static checks are stronger, reducing silent issues.

### Architectural Boundaries

- **Webchat**: no lint guardrails.
- **Moments**: explicit restrictions on cross-layer imports.

## Proposed Improvements

### 1) Add Biome linting to webchat

**Files to add/modify**:
- `pinocchio/cmd/web-chat/web/biome.json`
- `pinocchio/cmd/web-chat/web/package.json`

**Suggested biome.json (starter):**

```json
{
  "$schema": "https://biomejs.dev/schemas/2.3.8/schema.json",
  "files": {
    "ignoreUnknown": true,
    "includes": ["src/**", "vite.config.ts"]
  },
  "formatter": { "enabled": false },
  "linter": {
    "enabled": true,
    "rules": {
      "recommended": true,
      "suspicious": {
        "noEmptyBlockStatements": "error",
        "noExplicitAny": "off"
      },
      "correctness": {
        "useExhaustiveDependencies": "warn"
      }
    }
  }
}
```

**Rationale:**
- `noEmptyBlockStatements` forces us to explicitly document intentional ignores (or remove them).
- Mirrors Moments’ approach: lint is enabled, formatting disabled.

**Package.json scripts to add**:

```json
{
  "scripts": {
    "lint": "npx --yes @biomejs/biome@2.3.8 ci .",
    "lint:fix": "npx --yes @biomejs/biome@2.3.8 check --write .",
    "check": "npm run typecheck && npm run lint"
  }
}
```

### 2) Integrate typecheck + lint into build flow

Two options (both are valid; choose one):

**Option A: Make webchat build stricter**

Modify `pinocchio/cmd/web-chat/web/package.json`:

```json
"build": "npm run typecheck && vite build --outDir ../static/dist"
```

**Option B: Keep build fast, add `check` for CI/hook**

Add `check` script only and ensure CI/hook uses it.

### 3) Add webchat checks to repo-level workflows

**Makefile (pinocchio)**

Add targets that mirror Moments:

```make
web-typecheck:
	@cd cmd/web-chat/web && npm run typecheck

web-lint:
	@cd cmd/web-chat/web && npm run lint

web-check: web-typecheck web-lint
```

Then include `web-check` in root `lint` or in a new `lint-web` target.

**Lefthook (pinocchio/lefthook.yml)**

Add to pre-commit or pre-push:

```yaml
pre-commit:
  commands:
    web-check:
      glob: 'pinocchio/cmd/web-chat/web/src/**/*.{ts,tsx}'
      run: cd cmd/web-chat/web && npm run check
```

**CI**

If a GitHub Actions pipeline exists for pinocchio, add a step:

```yaml
- name: Webchat checks
  run: cd pinocchio/cmd/web-chat/web && npm run check
```

### 4) Replace silent catches with structured logging

**Goal**: No empty catch blocks. If errors are intentionally ignored, they must be logged or explicitly documented.

**New helper**: `pinocchio/cmd/web-chat/web/src/utils/logger.ts`

```ts
export type LogContext = {
  scope: string;
  convId?: string;
  runId?: string;
  seq?: number;
  extra?: Record<string, unknown>;
};

export function logInfo(msg: string, ctx?: LogContext) {
  console.info(`[webchat] ${msg}`, ctx ?? {});
}

export function logWarn(msg: string, ctx?: LogContext) {
  console.warn(`[webchat] ${msg}`, ctx ?? {});
}

export function logError(msg: string, err?: unknown, ctx?: LogContext) {
  console.error(`[webchat] ${msg}`, err, ctx ?? {});
}
```

**Usage in `wsManager.ts`**:

```ts
try {
  const payload = JSON.parse(String(m.data));
  ...
} catch (err) {
  logWarn('ws message parse failed', { scope: 'ws.onmessage', extra: { data: String(m.data).slice(0, 200) } });
}
```

**Hydration failure logging**:

```ts
try {
  const res = await fetch(...);
  if (!res.ok) {
    logWarn('hydrate failed', { scope: 'hydrate', extra: { status: res.status } });
    return;
  }
  const j = await res.json();
  if (!j || typeof j !== 'object') {
    logWarn('hydrate invalid payload', { scope: 'hydrate' });
    return;
  }
  ...
} catch (err) {
  logError('hydrate exception', err, { scope: 'hydrate' });
}
```

### 5) Add a React Error Boundary

**File**: `pinocchio/cmd/web-chat/web/src/components/ErrorBoundary.tsx`

```tsx
export class ErrorBoundary extends React.Component<{ children: React.ReactNode }, { hasError: boolean }> {
  state = { hasError: false };
  static getDerivedStateFromError() { return { hasError: true }; }
  componentDidCatch(error: unknown, info: unknown) {
    logError('render error', error, { scope: 'ErrorBoundary', extra: { info } });
  }
  render() {
    if (this.state.hasError) {
      return <div className="card">Something went wrong. Check logs.</div>;
    }
    return this.props.children;
  }
}
```

Wrap `App` or `ChatWidget` with this boundary in the entrypoint (e.g., `src/main.tsx`).

### 6) Improve error surfacing in UI state

Add a minimal error queue slice:
- `pinocchio/cmd/web-chat/web/src/store/errorsSlice.ts`

Pseudocode:

```ts
type AppError = { id: string; message: string; scope: string; time: number };

const errorsSlice = createSlice({
  name: 'errors',
  initialState: [] as AppError[],
  reducers: {
    reportError(state, action: PayloadAction<AppError>) { state.push(action.payload); },
    clearErrors(state) { return []; }
  }
});
```

Then in `logger.ts`, optionally dispatch an error action in dev mode to render in a debug panel.

### 7) Decide “throw vs log” semantics

**Guideline:**
- **Invariant broken** (e.g., missing required fields, impossible states): throw in dev, log + recover in prod.
- **External I/O failures** (fetch, WebSocket, clipboard): log with context; keep UI responsive.
- **Parsing failures**: log payload sample; do not throw unless in debug mode.

**Pseudocode helper**:

```ts
const DEV = import.meta.env.DEV;

export function assertInvariant(condition: boolean, message: string, ctx?: LogContext): asserts condition {
  if (!condition) {
    const err = new Error(message);
    logError(message, err, ctx);
    if (DEV) throw err;
  }
}
```

### 8) Normalize bigint fields at the boundary

We should decide on a consistent data model for timeline `version`, `createdAtMs`, `updatedAtMs`:

Option A (recommended for UI): convert bigint → number at the parsing boundary.

```ts
function asNumber(v: unknown): number | undefined {
  if (typeof v === 'bigint') return Number(v);
  if (typeof v === 'number') return v;
  return undefined;
}
```

This resolves the current typecheck errors and keeps Redux serializable.

## Suggested Implementation Order

1. Add `biome.json` + lint scripts.
2. Add `check` script and wire into Makefile/lefthook/CI.
3. Introduce logger helpers and remove empty catch blocks.
4. Add error boundary and (optional) error slice for UI.
5. Normalize bigint fields in timeline mapping.

## Files and Symbols Summary

- `pinocchio/cmd/web-chat/web/package.json`
  - Add `lint`, `lint:fix`, `check` scripts; optionally update `build`.

- `pinocchio/cmd/web-chat/web/biome.json`
  - Enable linting rules, especially `suspicious/noEmptyBlockStatements`.

- `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`
  - Replace empty catches with `logWarn`/`logError`.
  - Log hydration failures with HTTP status.

- `pinocchio/cmd/web-chat/web/src/chat/ChatWidget.tsx`
  - Avoid empty catches; log failures on URL updates and WS connection.

- `pinocchio/cmd/web-chat/web/src/utils/logger.ts`
  - Central logging helper with context.

- `pinocchio/cmd/web-chat/web/src/components/ErrorBoundary.tsx`
  - React error boundary around the app.

- `pinocchio/Makefile` + `pinocchio/lefthook.yml`
  - Add `web-check` target and hook integration.

- Reference files from Moments:
  - `moments/web/biome.json`
  - `moments/web/package.json`
  - `moments/Makefile`
  - `moments/lefthook.yml`

## Decisions (as requested)

- Package manager: stay on `npm` for webchat.
- Check scope: run webchat checks only when webchat files change (hooks/CI filter).
- Logging: client-side only (no backend log shipping).

## Summary

Webchat currently lacks a linting pipeline and tolerates silent errors, which makes bugs like hydration failure harder to detect. The Moments web pipeline provides a clear template: enforce typecheck + Biome lint in Makefile and hooks, and keep lint rules explicit. Combining that with explicit error logging and an error boundary will significantly improve both development feedback and runtime debuggability.
