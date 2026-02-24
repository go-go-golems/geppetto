---
Title: TypeScript/React Baseline Typecheck Findings and Remediation Plan
Ticket: GP-03-ENGINE-TS-BASELINE-TYPECHECK
Status: active
Topics:
    - frontend
    - infrastructure
    - chat
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../go-go-os/packages/engine/package.json
      Note: Engine package dependency surface and peer/dev typing context.
    - Path: ../../../../../../../go-go-os/packages/engine/src/app/generateCardStories.tsx
      Note: Representative TS7016 and TS2742 export/type issues.
    - Path: ../../../../../../../go-go-os/packages/engine/src/components/widgets/Btn.tsx
      Note: Representative TS2322/TS2339 prop contract mismatch origin.
    - Path: ../../../../../../../go-go-os/packages/engine/src/hypercard/editor/editorLaunch.ts
      Note: Representative TS2307 missing redux typing error.
    - Path: ../../../../../../../go-go-os/packages/engine/tsconfig.json
      Note: Strict/declaration compiler settings driving failure modes.
ExternalSources: []
Summary: Baseline compile audit for @hypercard/engine documenting TypeScript/React error taxonomy, root causes, and phased remediation to restore strict declaration builds.
LastUpdated: 2026-02-24T22:53:00-05:00
WhatFor: Use this when planning or executing type-system remediation for go-go-os/packages/engine so strict package builds become green.
WhenToUse: When tsc -b fails for @hypercard/engine or when introducing new exported React/TS APIs that can affect declaration portability.
---


# TypeScript/React Baseline Typecheck Findings and Remediation Plan

## Implementation Status (2026-02-24)

Remediation is complete for the scoped GP-03 tasks.

What changed:

- Added package-local React and React type dev dependencies in `@hypercard/engine`.
- Added explicit story meta typing for `CodeEditorWindow` story module to avoid non-portable inferred declaration output.
- Removed direct `redux` type import from editor launch path by deriving dispatch action type from local `openWindow` action creator.

Verification:

- `pnpm --filter @hypercard/engine build` passes.
- `pnpm --filter @hypercard/engine test` passes.

Attached evidence:

- `sources/01-baseline-build.log`
- `sources/02-intermediate-build.log`
- `sources/03-green-build.log`
- `sources/04-green-test.log`

## Executive Summary

`@hypercard/engine` currently passes tests but fails strict TypeScript build (`tsc -b`) due to baseline typing drift that predates GP-01 profile registry work. The failure is broad (385 compile errors) but concentrated into a small set of repeated root causes. This ticket scopes a focused remediation effort so we can keep strict mode, preserve declaration generation quality, and unblock reliable package builds.

On 2026-02-23, build output was captured in `/tmp/engine-build-errors.log` with these counts:

- `TS7016`: 175 errors (missing declarations for `react`)
- `TS7006`: 83 errors (implicit `any` parameters)
- `TS2742`: 82 errors (non-portable inferred declaration types)
- `TS2322`: 28 errors (component prop contract mismatches)
- `TS18046`: 14 errors (`unknown` narrowing gaps)
- `TS2339`: 2 errors (property shape mismatch)
- `TS2307`: 1 error (missing `redux` module types)

This plan addresses errors in a dependency-first order, then API typing normalization, then strictness cleanup, then declaration portability stabilization.

## Problem Statement

The engine package is configured as a strict, declaration-emitting TypeScript library:

- `strict: true`
- `declaration: true`
- `composite: true`
- `moduleResolution: bundler`

This is correct for a reusable package, but the current local dependency graph and exported symbol typing patterns are inconsistent with those goals:

- package-local `react` / `@types/react` declarations are not reliably available at typecheck time (`TS7016` flood),
- exported functions/components rely on inferred structural React types that pull path-specific `.pnpm` references (`TS2742`),
- UI widget prop contracts and call sites have drifted (`TS2322`/`TS2339`),
- strict function and `unknown` handling is incomplete (`TS7006`/`TS18046`),
- one legacy `redux` import expects missing module types (`TS2307`).

The result is that package tests pass while package compilation and declaration generation fail, which blocks downstream package consumption confidence and release hygiene.

## Proposed Solution

Fix in four phases, each with a clear stop condition:

1. Dependency and compiler baseline stabilization
- Ensure `@hypercard/engine` has local dev-time type availability for React types used in strict library builds.
- Confirm root workspace install and package-local resolve paths match expected TS behavior under pnpm.
- Re-run `tsc -b` to verify `TS7016` drops to zero before other edits.

2. Exported API type explicitness
- Add explicit return/type annotations for exported functions and React components where inferred types leak path-specific internals.
- Use stable public React types (`ReactNode`, `FC` only where appropriate, explicit props/result types) to avoid `TS2742`.

3. Contract and strictness cleanup
- Fix widget prop interfaces and usage sites causing `TS2322`/`TS2339`.
- Type untyped callback parameters and reducer/event handlers causing `TS7006`.
- Add narrowers/guards for `unknown` values causing `TS18046`.

4. Residual module/type hygiene
- Resolve the single `redux` module declaration issue (`TS2307`) with either correct dependency typing or migration to toolkit imports already used in the repo.
- Remove remaining property-access mismatches.

Each phase should land as separate commits to keep review, rollback, and bisect boundaries clean.

## Design Decisions

1. Keep strict mode enabled
- Rationale: downgrading `strict` would hide real contract defects and postpone failure to consumers.

2. Keep declaration build enabled
- Rationale: `@hypercard/engine` is a shared package. Declaration emit is part of the product contract.

3. Prefer explicit exported types over inferred declarations
- Rationale: fixes `TS2742` at the source and makes public API contracts stable across package manager layouts.

4. Fix dependencies before source-level cleanup
- Rationale: source edits are noisy when the baseline declaration graph is broken; eliminating `TS7016` first reduces false signals.

5. Do not couple this ticket to GP-01 feature work
- Rationale: GP-01 profile functionality is testable and mostly merged; this is build-quality debt and should remain a dedicated quality lane.

## Alternatives Considered

1. Temporary `skipLibCheck`/`noEmit` or relaxed strictness
- Rejected: masks contract issues and leaves package unusable as a typed library.

2. Add broad `any` shims for React and unresolved modules
- Rejected: creates long-term type debt and undermines declaration value.

3. Large one-shot refactor across all frontend packages
- Rejected: too risky and hard to review; this ticket should remain focused on deterministic compile recovery in `@hypercard/engine`.

4. Deferring to consumer package type environments
- Rejected: library packages must typecheck in isolation within the monorepo.

## Implementation Plan

### Phase A: Baseline capture and tracking

- Keep `/tmp/engine-build-errors.log` as initial snapshot.
- Add task-level acceptance criteria in `tasks.md`.
- Define done-condition: `pnpm --filter @hypercard/engine build` succeeds with zero TS errors.

### Phase B: React declaration baseline

- Verify/package local dev dependency strategy for:
  - `react`
  - `react-dom`
  - `@types/react`
  - `@types/react-dom`
- Reinstall and re-run build.
- Confirm `TS7016` class is eliminated.

### Phase C: Export and declaration portability pass

- Triage high-churn exported modules first:
  - `src/app/generateCardStories.tsx`
  - `src/chat/components/*`
  - `src/components/widgets/*`
  - `src/desktop-react.ts`
- Add explicit signatures for exported helpers/components.
- Re-run build and verify major reduction in `TS2742`.

### Phase D: Contract strictness pass

- Resolve prop drift in button/chip/widget contracts.
- Fix implicit-any callback params and `unknown` comparator patterns.
- Close `TS2322`, `TS7006`, `TS18046`, `TS2339`.

### Phase E: Residual module cleanup

- Resolve `redux` module typing issue in `src/hypercard/editor/editorLaunch.ts`.
- Final compile, then run package tests:
  - `pnpm --filter @hypercard/engine build`
  - `pnpm --filter @hypercard/engine test`

### Phase F: Guardrails

- Optionally add a CI job gate (or enforce existing gate) that runs `pnpm --filter @hypercard/engine build` so regressions are caught early.

## Open Questions

1. Should React and React type packages be peer-only or dual peer+dev dependencies for library-local typecheck reliability in this workspace?
2. Should exported React component patterns be normalized to plain typed functions instead of mixed inferred forms to reduce future `TS2742` regressions?
3. Do we want to treat `TS7006` in internal-only files with explicit local aliases/interfaces or broader reusable utility type helpers?
4. Should `redux` references be removed entirely in favor of toolkit-typed imports if this package is now RTK-first?

## References

- Error snapshot: `/tmp/engine-build-errors.log`
- Package config: `go-go-os/packages/engine/package.json`
- Compiler config: `go-go-os/packages/engine/tsconfig.json`
- Representative failing files:
  - `go-go-os/packages/engine/src/app/generateCardStories.tsx`
  - `go-go-os/packages/engine/src/components/shell/windowing/PluginCardRenderer.tsx`
  - `go-go-os/packages/engine/src/components/widgets/Btn.tsx`
  - `go-go-os/packages/engine/src/diagnostics/useDiagnosticsSnapshot.ts`
  - `go-go-os/packages/engine/src/hypercard/editor/editorLaunch.ts`
