package main

import (
	"github.com/go-go-golems/geppetto/pkg/analysis/turnsdatalint"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(turnsdatalint.Analyzer)
}
