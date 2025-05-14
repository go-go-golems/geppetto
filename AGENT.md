# AGENT.md - Geppetto Repository Guidelines

## Commands

### Build & Test
```
make build      # Build all packages
make test       # Run all tests
go test ./...   # Run all tests (alternate)
go test ./pkg/embeddings  # Run tests in specific package
go test ./pkg/embeddings -run=TestBasicCacheOperations  # Run a specific test
```

### Lint
```
make lint       # Run golangci-lint
make lintmax    # Run golangci-lint with more issues shown
```

## Code Style

- Go version: 1.24+
- Use contexts for cancellation where appropriate
- Write tests with descriptive names using table-driven approach
- Use testify for assertions (require for fatal, assert for non-fatal)
- Follow Go naming conventions (CamelCase for exported, camelCase for private)
- Document all exported types, functions, and methods
- Error handling: return errors, don't panic
- Organize imports: standard library first, then external packages
- Use functional options pattern for configurable components

<goGuidelines>
When implementing go interfaces, use the var _ Interface = &Foo{} to make sure the interface is always implemented correctly.
When building web applications, use htmx, bootstrap and the templ templating language.
Always use a context argument when appropriate.
Use cobra for command-line applications.
Use the "defaults" package name, instead of "default" package name, as it's reserved in go.
Use github.com/pkg/errors for wrapping errors.
When starting goroutines, use errgroup.
Don't make lint, just go straight to make build.
</goGuidelines>