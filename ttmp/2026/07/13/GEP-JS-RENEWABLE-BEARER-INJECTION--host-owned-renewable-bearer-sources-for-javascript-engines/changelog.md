# Changelog

## 2026-07-13

- Initial workspace created


## 2026-07-13

Created the intern-facing host-only bearer injection design and recorded the JavaScript engine construction gap.

### Related Files

- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/ttmp/2026/07/13/GEP-JS-RENEWABLE-BEARER-INJECTION--host-owned-renewable-bearer-sources-for-javascript-engines/design-doc/01-host-owned-renewable-bearer-source-injection-for-javascript-engines.md — Architecture, API, and implementation plan


## 2026-07-13

Added the Go-host-only bearer source option to JavaScript engine construction (commit 13621922).

### Related Files

- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/pkg/js/modules/geppetto/api_engine_builder.go — Factory wiring
- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/pkg/js/modules/geppetto/module.go — Registration option and private runtime field


## 2026-07-13

Added behavioral tests for host source injection, retained static-key validation, and no JavaScript source exposure (commit f962653d).

### Related Files

- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/pkg/js/modules/geppetto/api_engine_builder_test.go — Regression coverage


## 2026-07-13

Published host-registration documentation and completed focused normal/race validation for the JavaScript bearer-source integration (commit 351f5cbb).

### Related Files

- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/pkg/doc/playbooks/08-use-renewable-bearer-credentials.md — Host-only JavaScript registration API
- /home/manuel/workspaces/2026-07-10/refresh-oauth-token-geppetto/geppetto/pkg/js/modules/geppetto/api_engine_builder_test.go — Validated source and no-source behavior

