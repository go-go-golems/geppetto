// Package credscope holds the cross-path credential-scope regression matrix
// (GEPPETTO-CREDSCOPE-001). It has no production code: credscope_test.go
// drives every engine, adapter and the Pi bridge with marker secrets through
// a recording transport and asserts that no marker reaches an origin other
// than the one its configuration names, including after redirects.
//
// A new engine or adapter that attaches a credential must add a row.
package credscope
