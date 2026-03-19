package main

import (
	"github.com/go-go-golems/geppetto/pkg/analysis/turnsdatalint"
	"golang.org/x/tools/go/analysis/multichecker"
)

// geppetto-lint bundles Geppetto's custom go/analysis analyzers into a single vettool binary.
//
// This is the preferred long-term packaging shape: add new analyzers under pkg/analysis/<name>,
// then register them here to make them available via:
//
//	go vet -vettool=/path/to/geppetto-lint ./...
func main() {
	multichecker.Main(
		turnsdatalint.Analyzer,
	)
}
