# Changelog

## 2026-02-21

- Initial workspace created
- Added implementation plan, phased tasks, and detailed diary initialization.
- Implemented shared `geppetto/plugins` module for plugin contract helpers.
- Added geppetto module test for plugin helper exports and run-input validation.
- Migrated runner scripts to consume `require("geppetto/plugins")` and removed local duplicate helper.
- Implemented `geppetto/plugins` module and helper API (`3f5320f`).
- Migrated cozo runner scripts to shared helper import and removed local duplicate helper (`19ca200`).
