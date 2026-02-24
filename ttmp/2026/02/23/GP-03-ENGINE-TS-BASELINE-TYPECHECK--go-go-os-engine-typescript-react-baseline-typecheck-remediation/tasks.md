# Tasks

## TODO

- [x] Establish reproducible baseline (`pnpm --filter @hypercard/engine build`) and attach latest error snapshot to ticket.
- [x] Eliminate TS7016 React declaration failures in `@hypercard/engine` by fixing package-local type dependency visibility.
- [x] Resolve TS2742 declaration portability errors by adding explicit exported type annotations in high-churn modules.
- [x] Fix TS2322/TS2339 widget prop contract mismatches (Btn/Chip and dependent call sites).
- [x] Fix TS7006 implicit-any and TS18046 unknown narrowing issues in runtime/diagnostics flows.
- [x] Resolve TS2307 `redux` module typing issue in editor launch path.
- [x] Verify green build and tests (`pnpm --filter @hypercard/engine build && pnpm --filter @hypercard/engine test`).
