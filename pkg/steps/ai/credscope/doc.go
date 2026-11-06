// Package credscope holds the cross-path credential-scope regression matrix
// (GEPPETTO-CREDSCOPE-001). It has no production code: credscope_test.go
// and bound_paths_test.go drive 64 engine, adapter, counter, audio, cache and Pi paths with marker secrets through
// a recording transport and asserts that no marker reaches an origin other
// than the one its configuration names, including after redirects.
//
// A new authenticated path must add a row and use a bound connection. OAuth
// grant and issuer-specific helper tests supplement this matrix in their packages.
package credscope
