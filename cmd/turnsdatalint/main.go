package main

import (
	"github.com/go-go-golems/geppetto/internal/turnsdatalint"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(turnsdatalint.Analyzer)
}
