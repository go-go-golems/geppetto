// Package runner provides an app-facing inference entry point above session,
// enginebuilder, toolloop, middlewarecfg, and tool registries.
//
// The package consumes fully resolved runtime input from the caller. It does
// not resolve profiles, merge patches, or own application policy. Callers are
// expected to decide:
//
//   - final StepSettings
//   - final system prompt
//   - middleware uses or concrete middlewares
//   - tool registrars and registry filtering policy
//   - runtime identity metadata
//
// The runner package then assembles Geppetto’s existing primitives into a
// prepared session and inference execution flow.
package runner
