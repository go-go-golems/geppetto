# Changelog

## 2025-12-18

- Initial workspace created

## 2025-12-18

Implemented typed-key enforcement for `turnsdatalint` (accept typed vars/params/conversions; reject raw string literals + untyped string const identifiers). Updated analysistest fixtures accordingly and validated with build, tests, and `make linttool`.

### Related Files

- /home/manuel/workspaces/2025-11-18/fix-pinocchio-profiles/geppetto/pkg/analysis/turnsdatalint/analyzer.go:Core analyzer change
- /home/manuel/workspaces/2025-11-18/fix-pinocchio-profiles/geppetto/pkg/analysis/turnsdatalint/testdata/src/a/a.go:Updated analysistest cases
- /home/manuel/workspaces/2025-11-18/fix-pinocchio-profiles/geppetto/pkg/doc/topics/12-turnsdatalint.md:Updated documentation to match new rule and flags


## 2026-01-25

Ticket closed

