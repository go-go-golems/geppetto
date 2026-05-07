# Changelog

## 2026-05-07

- Initial workspace created


## 2026-05-07

Created Anthropic observability ticket, source/test inventory, implementation tasks, design guide, and initial diary.

Implemented Claude observability and committed source changes in `cc714f4d1deeb2ed94e0afb2119d7bad126b3ec2` (`feat(claude): add observability hooks`). The implementation adds Claude engine observer/config options, provider routed-event records, compact publish-started records only, StandardEngineFactory option plumbing, and focused tests. Targeted tests passed; pre-commit `go test ./...` and `make lint` passed on retry after one unrelated flaky JS module test failure.

