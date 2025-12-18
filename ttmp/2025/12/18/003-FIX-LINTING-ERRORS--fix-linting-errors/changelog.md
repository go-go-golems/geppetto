# Changelog

## 2025-12-18

- Initial workspace created
- Fix `gosec`/SSA panic by adding missing `turns.RunMetaKeyTraceID` constant; `make lintmax gosec` passing
- Update Watermill dependency from v1.5.0 to v1.5.1 to support `AddConsumerHandler` method (required for CI builds)
- Update `docker-lint` Makefile target to use golangci-lint v2.1.0 (matching GitHub Actions workflow version)


## 2025-12-18

Ticket closed

