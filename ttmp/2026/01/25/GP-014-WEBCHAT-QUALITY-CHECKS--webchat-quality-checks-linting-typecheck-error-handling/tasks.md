# Tasks

## TODO

- [x] Add tasks here

- [ ] Add Biome config for webchat (biome.json) based on Moments baseline
- [ ] Add npm scripts (lint, lint:fix, check) to webchat package.json; keep npm workflow
- [ ] Add webchat lint/typecheck/check targets to pinocchio Makefile
- [ ] Update pinocchio/lefthook.yml to run webchat check only when webchat files change
- [ ] Add/update CI job to run webchat check only when webchat files change (if CI pipeline exists)
- [ ] Add client-side logger helper (src/utils/logger.ts) with scoped log functions
- [ ] Replace empty catch blocks in wsManager.ts with logged errors (client-side only)
- [ ] Replace empty catch blocks in ChatWidget.tsx with logged errors (client-side only)
- [ ] Add React ErrorBoundary component and wrap the webchat app entry
- [ ] Normalize bigint fields in timeline mapping (registry.ts, wsManager.ts, timelineMapper.ts) to resolve typecheck errors
- [ ] Add optional client-side error queue (errorsSlice) and lightweight debug panel (no backend logging)
- [ ] Update webchat README/docs with new lint/typecheck/check instructions
