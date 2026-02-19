# Changelog

## 2026-02-12

- Initial workspace created


## 2026-02-12

Implemented JS toolloop lifecycle hooks (before/after/error), hook-aware executor wiring, retry/abort policy controls, tests, and a smoke script.

### Related Files

- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/geppetto/ttmp/2026/02/12/GP-010-JS-TOOLLOOP-HOOKS--js-api-toolloop-lifecycle-hooks/scripts/test_toolloop_hooks_smoke.js — Added ticket smoke script
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/inference/toolloop/enginebuilder/builder.go — Added tool executor injection in builder/runner
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/inference/toolloop/enginebuilder/options.go — Added WithToolExecutor option
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/inference/tools/base_executor.go — Added current tool call context helpers
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/js/modules/geppetto/api.go — Added hook parsing
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/js/modules/geppetto/module_test.go — Added lifecycle hook scenario test

