# Tasks

## TODO

- [x] Add tasks here

- [x] Add Biome config for webchat (biome.json) based on Moments baseline
- [x] Add npm scripts (lint, lint:fix, check) to webchat package.json; keep npm workflow
- [x] Add webchat lint/typecheck/check targets to pinocchio Makefile
- [x] Update pinocchio/lefthook.yml to run webchat check only when webchat files change
- [x] Add/update CI job to run webchat check only when webchat files change (if CI pipeline exists)
- [x] Add client-side logger helper (src/utils/logger.ts) with scoped log functions
- [x] Replace empty catch blocks in wsManager.ts with logged errors (client-side only)
- [x] Replace empty catch blocks in ChatWidget.tsx with logged errors (client-side only)
- [x] Add React ErrorBoundary component and wrap the webchat app entry
- [x] Normalize bigint fields in timeline mapping (registry.ts, wsManager.ts, timelineMapper.ts) to resolve typecheck errors
- [x] Add optional client-side error queue (errorsSlice) and lightweight debug panel (no backend logging)
- [x] Update webchat README/docs with new lint/typecheck/check instructions
