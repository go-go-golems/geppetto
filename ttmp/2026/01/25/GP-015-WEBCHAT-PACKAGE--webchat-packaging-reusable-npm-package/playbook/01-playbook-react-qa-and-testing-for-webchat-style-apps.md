---
Title: 'Playbook: React QA and testing for webchat-style apps'
Ticket: GP-015-WEBCHAT-PACKAGE
Status: active
Topics:
    - frontend
    - architecture
    - infrastructure
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/.github/workflows/webchat-check.yml
      Note: CI workflow for webchat check
    - Path: pinocchio/Makefile
      Note: Makefile targets for web checks
    - Path: pinocchio/cmd/web-chat/web/biome.json
      Note: Biome lint configuration example
    - Path: pinocchio/cmd/web-chat/web/package.json
      Note: npm scripts for typecheck/lint/check
    - Path: pinocchio/cmd/web-chat/web/src/chat/ChatWidget.tsx
      Note: Client error panel example
    - Path: pinocchio/cmd/web-chat/web/src/ws/wsManager.ts
      Note: Runtime error logging example
    - Path: pinocchio/lefthook.yml
      Note: Hook wiring for web checks
ExternalSources: []
Summary: Operational playbook for adding linting, typechecking, error surfacing, and CI checks to a React webchat app.
LastUpdated: 2026-01-25T17:11:39-05:00
WhatFor: Reusable QA and testing playbook for React webchat apps.
WhenToUse: Use when a React chat UI needs stronger QA and runtime error visibility.
---


# Playbook: React QA and testing for webchat-style apps

## Purpose

Provide a repeatable, end-to-end playbook for adding strong static checks, runtime error visibility, and CI enforcement to a React webchat app. The steps mirror what we implemented in webchat: TypeScript typecheck, Biome lint, Makefile and hook wiring, a web-only CI workflow, client-side logging, and a lightweight error panel.

## Environment Assumptions

- Node.js 18 or 20 is available.
- The React app uses TypeScript and Vite (or a similar bundler).
- The repo supports Makefile and lefthook (or can be adapted).
- The app can tolerate client-side logs and a small debug-only error panel.

> FUNDAMENTAL: Why static checks first?
> 
> Typechecks and linting are the cheapest bug detectors you will ever deploy. They run fast, they are deterministic, and they are easy to enforce on every change. Start there before you chase flakier integration tests.

## Commands

Use these commands as the canonical flow when you run or validate this playbook. Replace paths to match your repo layout.

```bash
# 1) Typecheck
cd cmd/web-chat/web
npm run typecheck

# 2) Lint
npm run lint

# 3) Combined check
npm run check
```

If you wire Makefile targets:

```bash
# From repo root
make web-typecheck
make web-lint
make web-check
```

## Exit Criteria

- `npm run check` succeeds locally.
- The pre-commit hook runs `npm run check` for webchat changes only.
- CI runs `npm run check` only when webchat files change.
- No empty catch blocks exist in the webchat runtime path.
- There is a client-side error boundary and a lightweight error queue panel.

## Steps

### Step 0: Baseline and scope

- Identify where the React app lives (e.g., `cmd/web-chat/web`).
- Identify the current scripts in `package.json`.
- Run typecheck and record current errors.

> FUNDAMENTAL: The first pass is about visibility, not perfection.
> 
> You should expect a first run to fail. The value is the error list. That list becomes your to-do set.

### Step 1: Add linting configuration (Biome)

1. Add `biome.json` to the app root.
2. Disable formatting and enable lint rules with `recommended` baseline.
3. Exclude generated code and storybook output from the lint scope.

Example:

```json
{
  "$schema": "https://biomejs.dev/schemas/2.3.8/schema.json",
  "files": {
    "ignoreUnknown": true,
    "includes": [
      "src/**",
      "vite.config.ts",
      "!!**/src/sem/pb",
      "!!**/storybook-static"
    ]
  },
  "formatter": { "enabled": false },
  "linter": {
    "enabled": true,
    "rules": {
      "recommended": true,
      "suspicious": { "noEmptyBlockStatements": "error", "noExplicitAny": "off" },
      "correctness": { "useExhaustiveDependencies": "warn" }
    }
  }
}
```

> FUNDAMENTAL: Lint is a policy. Keep policy explicit.
> 
> A lint config is the programming equivalent of a safety culture. Put the rules you care about in the config so future work preserves the same intent.

### Step 2: Add scripts in package.json

Add lint, lint:fix, and check scripts.

```json
{
  "scripts": {
    "typecheck": "tsc -p tsconfig.json --noEmit",
    "lint": "npx --yes @biomejs/biome@2.3.8 ci .",
    "lint:fix": "npx --yes @biomejs/biome@2.3.8 check --write .",
    "check": "npm run typecheck && npm run lint"
  }
}
```

> FUNDAMENTAL: One check command.
> 
> A single `check` script reduces cognitive load and makes hooks and CI stable.

### Step 3: Wire Makefile targets

Expose three targets so all developers can run the same checks from the repo root.

```make
web-typecheck:
	cd cmd/web-chat/web && npm run typecheck

web-lint:
	cd cmd/web-chat/web && npm run lint

web-check: web-typecheck web-lint
```

### Step 4: Add pre-commit hook (lefthook)

Run web checks only when webchat files change to keep hooks fast.

```yaml
pre-commit:
  commands:
    web-check:
      glob: 'cmd/web-chat/web/**'
      run: cd cmd/web-chat/web && npm run check
```

### Step 5: Add CI workflow scoped to webchat

Create a webchat-only workflow that triggers on file changes.

```yaml
on:
  pull_request:
    paths:
      - 'cmd/web-chat/web/**'
  push:
    branches: [ main ]
    paths:
      - 'cmd/web-chat/web/**'
```

### Step 6: Replace empty catch blocks with logging

- Create a `logger.ts` with `logInfo`, `logWarn`, and `logError`.
- Replace empty catch blocks in WebSocket, hydration, and clipboard paths.
- Include scope and context in logs so you can trace errors later.

> FUNDAMENTAL: Silent failures kill debuggability.
> 
> Every catch block is a decision. Either you handle it or you log it. Silence is rarely correct.

### Step 7: Add a client-side error boundary

Wrap your app in a React ErrorBoundary and log render errors.

```tsx
<ErrorBoundary>
  <App />
</ErrorBoundary>
```

### Step 8: Add an error queue and lightweight debug panel

- Add a Redux slice (or similar) to store recent errors.
- Render a small panel that shows the last N errors.
- Keep it light and optional. This is a developer-facing aid.

### Step 9: Fix typecheck failures (bigint and parsing)

- Normalize bigint fields at the edge before they enter Redux.
- Validate runtime payloads before usage.

```ts
export function toNumber(value: unknown): number | undefined {
  if (typeof value === 'number' && Number.isFinite(value)) return value;
  if (typeof value === 'bigint') return Number(value);
  if (typeof value === 'string') {
    const num = Number(value);
    if (Number.isFinite(num)) return num;
  }
  return undefined;
}
```

> FUNDAMENTAL: Parse and normalize at boundaries.
> 
> Treat server payloads as hostile until proven otherwise. Normalize once at the boundary so the rest of the app can remain clean.

### Step 10: Document the workflow

Add a short section to the app README:

```bash
npm run typecheck
npm run lint
npm run check
```

## Failure Modes and Fixes

- Biome warnings about ignore patterns: Use `files.includes` negation patterns like `!!**/src/sem/pb` instead of deprecated config keys.
- Lint failures in generated code: Exclude generated protobuf output from lint scope.
- Unstable hooks: Scope by `glob` so web checks only run when web files change.

## Notes

- Keep logging client-side only unless there is explicit authorization to ship logs to a server.
- Keep the debug panel small and non-blocking. The primary UI must still work without it.
- Run `npm run check` before every merge.
